package syncer_test

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"trello_analytics_back/internal/importer"
	"trello_analytics_back/internal/storage/postgres"
	"trello_analytics_back/internal/syncer"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

func openTestDB(t *testing.T) *pgxpool.Pool {
	t.Helper()
	url := os.Getenv("TEST_DATABASE_URL")
	if url == "" {
		url = os.Getenv("DATABASE_URL")
	}
	if url == "" {
		url = "postgres://trello:trello_dev_password@localhost:5432/trello_analytics?sslmode=disable"
	}
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	pool, err := postgres.Connect(ctx, url)
	if err != nil {
		t.Skipf("postgres unavailable (%v)", err)
	}
	t.Cleanup(pool.Close)
	var exists bool
	err = pool.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1 FROM information_schema.tables
			WHERE table_schema = 'public' AND table_name = 'sync_runs'
		)
	`).Scan(&exists)
	if err != nil || !exists {
		t.Skipf("sync_runs missing; run migrate-up first")
	}
	return pool
}

// trackSyncRunCleanup deletes only sync_runs created by this test — never TRUNCATE shared history.
func trackSyncRunCleanup(t *testing.T, pool *pgxpool.Pool) func(runID string) {
	t.Helper()
	var mu sync.Mutex
	var ids []string
	t.Cleanup(func() {
		mu.Lock()
		defer mu.Unlock()
		ctx := context.Background()
		for _, id := range ids {
			if id == "" {
				continue
			}
			if _, err := pool.Exec(ctx, `DELETE FROM sync_runs WHERE id = $1::uuid`, id); err != nil {
				t.Errorf("cleanup sync_run %s: %v", id, err)
			}
		}
	})
	return func(runID string) {
		if runID == "" {
			return
		}
		mu.Lock()
		ids = append(ids, runID)
		mu.Unlock()
	}
}

type boardActionsCursorSnapshot struct {
	hadRow        bool
	cursorValue   *string
	lastSuccessAt *time.Time
	updatedAt     *time.Time
}

func snapshotBoardActionsCursor(t *testing.T, ctx context.Context, pool *pgxpool.Pool) boardActionsCursorSnapshot {
	t.Helper()
	var snap boardActionsCursorSnapshot
	err := pool.QueryRow(ctx, `
		SELECT cursor_value, last_success_at, updated_at
		FROM sync_cursors
		WHERE resource_type = $1
	`, postgres.ResourceBoardActions).Scan(&snap.cursorValue, &snap.lastSuccessAt, &snap.updatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return snap
	}
	if err != nil {
		t.Fatalf("snapshot cursor: %v", err)
	}
	snap.hadRow = true
	return snap
}

func (s boardActionsCursorSnapshot) restore(t *testing.T, ctx context.Context, pool *pgxpool.Pool) {
	t.Helper()
	if !s.hadRow {
		if _, err := pool.Exec(ctx, `DELETE FROM sync_cursors WHERE resource_type = $1`, postgres.ResourceBoardActions); err != nil {
			t.Fatalf("restore cursor delete: %v", err)
		}
		return
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO sync_cursors (resource_type, cursor_value, last_success_at, updated_at)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (resource_type) DO UPDATE SET
			cursor_value = EXCLUDED.cursor_value,
			last_success_at = EXCLUDED.last_success_at,
			updated_at = EXCLUDED.updated_at
	`, postgres.ResourceBoardActions, s.cursorValue, s.lastSuccessAt, s.updatedAt); err != nil {
		t.Fatalf("restore cursor upsert: %v", err)
	}
}

// withIncompletePrimaryImport temporarily removes board_actions cursor so Status reports incomplete.
func withIncompletePrimaryImport(t *testing.T, pool *pgxpool.Pool) {
	t.Helper()
	ctx := context.Background()
	guard := snapshotBoardActionsCursor(t, ctx, pool)
	if _, err := pool.Exec(ctx, `DELETE FROM sync_cursors WHERE resource_type = $1`, postgres.ResourceBoardActions); err != nil {
		t.Fatalf("clear cursor for status test: %v", err)
	}
	t.Cleanup(func() { guard.restore(t, context.Background(), pool) })
}

type blockingImporter struct {
	started chan struct{}
	release chan struct{}
	mu      sync.Mutex
	calls   int
}

func (b *blockingImporter) ImportBoard(ctx context.Context, _ importer.ImportBoardOptions) (importer.ImportBoardResult, error) {
	b.mu.Lock()
	b.calls++
	b.mu.Unlock()
	select {
	case b.started <- struct{}{}:
	default:
	}
	select {
	case <-b.release:
		return importer.ImportBoardResult{}, nil
	case <-ctx.Done():
		return importer.ImportBoardResult{}, ctx.Err()
	}
}

func TestOrchestrator_SingleFlightAlreadyRunning(t *testing.T) {
	pool := openTestDB(t)
	remember := trackSyncRunCleanup(t, pool)
	store := postgres.NewStore(pool)

	imp := &blockingImporter{
		started: make(chan struct{}, 1),
		release: make(chan struct{}),
	}
	life, cancelLife := context.WithCancel(context.Background())
	defer cancelLife()

	orch := syncer.NewOrchestrator(store, imp, syncer.Config{
		BoardID: "board-sf",
		Logger:  slog.Default(),
	})
	orch.SetLifecycle(life)

	id1, err := orch.Start(context.Background())
	remember(id1)
	if err != nil {
		t.Fatalf("first start: %v", err)
	}
	select {
	case <-imp.started:
	case <-time.After(3 * time.Second):
		t.Fatal("importer did not start")
	}

	_, err = orch.Start(context.Background())
	var already *syncer.AlreadyRunningError
	if !errors.As(err, &already) {
		t.Fatalf("second start want AlreadyRunning, got %v", err)
	}
	if already.RunID != id1 {
		t.Fatalf("run_id=%s want %s", already.RunID, id1)
	}

	// Run also rejects while in-flight.
	_, err = orch.Run(context.Background())
	if !errors.As(err, &already) {
		t.Fatalf("Run while busy want AlreadyRunning, got %v", err)
	}

	close(imp.release)
	deadline := time.Now().Add(5 * time.Second)
	for {
		running, _ := orch.Running()
		if !running {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("run did not finish")
		}
		time.Sleep(20 * time.Millisecond)
	}
}

func TestOrchestrator_RunSuccessAndFailed(t *testing.T) {
	pool := openTestDB(t)
	remember := trackSyncRunCleanup(t, pool)
	store := postgres.NewStore(pool)

	okImp := &okImporter{}
	orch := syncer.NewOrchestrator(store, okImp, syncer.Config{
		BoardID: "board-run-ok",
		Logger:  slog.Default(),
	})
	orch.SetLifecycle(context.Background())

	runID, err := orch.Run(context.Background())
	remember(runID)
	if err != nil {
		t.Fatalf("Run success: %v", err)
	}
	if runID == "" {
		t.Fatal("empty run_id")
	}
	row, err := store.GetSyncRun(context.Background(), runID)
	if err != nil || row == nil || row.Status != postgres.SyncRunSuccess {
		t.Fatalf("row=%+v err=%v", row, err)
	}

	failOrch := syncer.NewOrchestrator(store, &failImporter{errMsg: "boom"}, syncer.Config{
		BoardID: "board-run-fail",
		Logger:  slog.Default(),
	})
	failOrch.SetLifecycle(context.Background())
	failID, err := failOrch.Run(context.Background())
	remember(failID)
	if !errors.Is(err, syncer.ErrRunFailed) {
		t.Fatalf("want ErrRunFailed, got %v", err)
	}
}

func TestOrchestrator_StatusShapeNoSecrets(t *testing.T) {
	pool := openTestDB(t)
	remember := trackSyncRunCleanup(t, pool)
	withIncompletePrimaryImport(t, pool)
	store := postgres.NewStore(pool)

	secretKey := "SUPER_SECRET_KEY"
	secretTok := "SUPER_SECRET_TOKEN"
	imp := &failImporter{errMsg: "trello api: 401 url=https://api.trello.com/1/boards/x?key=" + secretKey + "&token=" + secretTok + " body=unauthorized"}

	orch := syncer.NewOrchestrator(store, imp, syncer.Config{
		BoardID: "board-status",
		APIKey:  secretKey,
		Token:   secretTok,
		Logger:  slog.Default(),
	})
	life, cancel := context.WithCancel(context.Background())
	defer cancel()
	orch.SetLifecycle(life)

	runID, err := orch.Run(context.Background())
	remember(runID)
	if !errors.Is(err, syncer.ErrRunFailed) {
		t.Fatalf("want failed run, got runID=%s err=%v", runID, err)
	}

	st, err := orch.Status(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	raw, err := json.Marshal(st)
	if err != nil {
		t.Fatal(err)
	}
	s := string(raw)
	if strings.Contains(s, secretKey) || strings.Contains(s, secretTok) {
		t.Fatalf("secrets leaked in status JSON: %s", s)
	}
	var m map[string]any
	if err := json.Unmarshal(raw, &m); err != nil {
		t.Fatal(err)
	}
	for _, k := range []string{
		"board_id", "last_success_at", "current_run", "earliest_action_at", "latest_action_at",
		"counts", "primary_import_incomplete", "warnings",
	} {
		if _, ok := m[k]; !ok {
			t.Fatalf("missing key %q in %s", k, s)
		}
	}
	counts, ok := m["counts"].(map[string]any)
	if !ok {
		t.Fatalf("counts: %T", m["counts"])
	}
	for _, k := range []string{"cards", "lists", "members", "labels", "actions"} {
		if _, ok := counts[k]; !ok {
			t.Fatalf("counts missing %q", k)
		}
	}
	if m["board_id"] != "board-status" {
		t.Fatalf("board_id=%v", m["board_id"])
	}
	if m["primary_import_incomplete"] != true {
		t.Fatalf("expected incomplete primary import")
	}
	cr, ok := m["current_run"].(map[string]any)
	if !ok || cr["run_id"] != runID {
		t.Fatalf("current_run=%v want run_id=%s", m["current_run"], runID)
	}
	if cr["status"] != "failed" {
		t.Fatalf("status=%v", cr["status"])
	}
}

type okImporter struct{}

func (okImporter) ImportBoard(context.Context, importer.ImportBoardOptions) (importer.ImportBoardResult, error) {
	return importer.ImportBoardResult{}, nil
}

type failImporter struct {
	err    error
	errMsg string
}

func (f *failImporter) ImportBoard(context.Context, importer.ImportBoardOptions) (importer.ImportBoardResult, error) {
	if f.err != nil {
		return importer.ImportBoardResult{}, f.err
	}
	return importer.ImportBoardResult{}, &secretError{msg: f.errMsg}
}

type secretError struct{ msg string }

func (e *secretError) Error() string { return e.msg }

func TestOrchestrator_ReclaimOrphans(t *testing.T) {
	pool := openTestDB(t)
	remember := trackSyncRunCleanup(t, pool)
	store := postgres.NewStore(pool)
	ctx := context.Background()

	id, err := store.InsertSyncRun(ctx, time.Now().UTC())
	remember(id)
	if err != nil {
		t.Fatal(err)
	}
	orch := syncer.NewOrchestrator(store, &failImporter{errMsg: "unused"}, syncer.Config{
		BoardID: "b",
		Logger:  slog.Default(),
	})
	n, err := orch.ReclaimOrphans(ctx)
	if err != nil {
		t.Fatalf("reclaim err=%v", err)
	}
	// Other orphaned running rows may exist in a shared DB; only require ours is reclaimed.
	if n < 1 {
		t.Fatalf("reclaim n=%d want >= 1", n)
	}
	row, err := store.GetSyncRun(ctx, id)
	if err != nil || row == nil {
		t.Fatal(err)
	}
	if row.ID != id || row.Status != postgres.SyncRunFailed {
		t.Fatalf("row=%+v", row)
	}
	if row.ErrorSummary == nil || *row.ErrorSummary != "orphaned after process restart" {
		t.Fatalf("summary=%v", row.ErrorSummary)
	}
}

func TestOrchestrator_RunCancelled(t *testing.T) {
	pool := openTestDB(t)
	remember := trackSyncRunCleanup(t, pool)
	store := postgres.NewStore(pool)

	imp := &blockingImporter{
		started: make(chan struct{}, 1),
		release: make(chan struct{}), // never released; cancel instead
	}
	life, cancelLife := context.WithCancel(context.Background())
	orch := syncer.NewOrchestrator(store, imp, syncer.Config{
		BoardID: "board-cancel",
		Logger:  slog.Default(),
	})
	orch.SetLifecycle(life)

	type result struct {
		id  string
		err error
	}
	resCh := make(chan result, 1)
	go func() {
		id, err := orch.Run(life)
		resCh <- result{id: id, err: err}
	}()

	select {
	case <-imp.started:
	case <-time.After(3 * time.Second):
		t.Fatal("not started")
	}
	cancelLife()

	select {
	case res := <-resCh:
		remember(res.id)
		if !errors.Is(res.err, syncer.ErrRunCancelled) {
			t.Fatalf("want ErrRunCancelled, got %v", res.err)
		}
		row, err := store.GetSyncRun(context.Background(), res.id)
		if err != nil || row == nil {
			t.Fatal(err)
		}
		if row.Status != postgres.SyncRunCancelled {
			t.Fatalf("status=%s want cancelled", row.Status)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("Run did not return")
	}
}
