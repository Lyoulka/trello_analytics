package syncer

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"time"

	"trello_analytics_back/internal/importer"
	"trello_analytics_back/internal/storage/postgres"
	"trello_analytics_back/internal/trello"
)

// ErrAlreadyRunning is returned when single-flight rejects a new start.
var ErrAlreadyRunning = errors.New("sync already running")

// ErrRunFailed is returned by Run when the sync ended with status failed.
var ErrRunFailed = errors.New("sync run failed")

// ErrRunCancelled is returned by Run when the sync ended with status cancelled.
var ErrRunCancelled = errors.New("sync run cancelled")

// BoardImporter runs a full board import (entities + actions).
type BoardImporter interface {
	ImportBoard(ctx context.Context, opts importer.ImportBoardOptions) (importer.ImportBoardResult, error)
}

// RunStore persists sync_runs and status aggregates.
type RunStore interface {
	InsertSyncRun(ctx context.Context, startedAt time.Time) (id string, err error)
	FinishSyncRun(ctx context.Context, id, status string, counts postgres.SyncRunCounts, errorSummary *string) error
	ReclaimOrphanSyncRuns(ctx context.Context) (int64, error)
	GetSyncRun(ctx context.Context, id string) (*postgres.SyncRun, error)
	GetRunningSyncRun(ctx context.Context) (*postgres.SyncRun, error)
	GetLatestSyncRun(ctx context.Context) (*postgres.SyncRun, error)
	GetLastSuccessFinishedAt(ctx context.Context) (*time.Time, error)
	ActionDateRange(ctx context.Context) (earliest, latest *time.Time, err error)
	HasSyncCursor(ctx context.Context, resourceType string) (bool, error)
	CountCards(ctx context.Context) (int64, error)
	CountLists(ctx context.Context) (int64, error)
	CountMembers(ctx context.Context) (int64, error)
	CountLabels(ctx context.Context) (int64, error)
	CountCardActions(ctx context.Context) (int64, error)
}

// Config holds orchestrator wiring (non-secret board id + page size; secrets only for redaction).
type Config struct {
	BoardID         string
	ActionsPageSize int
	APIKey          string // for redaction only
	Token           string // for redaction only
	Logger          *slog.Logger
}

// Orchestrator owns single-flight sync and sync_runs lifecycle.
type Orchestrator struct {
	store    RunStore
	importer BoardImporter
	cfg      Config

	// lifeCtx is the process lifecycle context (from cmd/trello-sync). Run
	// goroutines derive from it so Ctrl+C cancels an in-flight import.
	lifeCtx context.Context

	mu        sync.Mutex
	running   bool
	runID     string
	runCancel context.CancelFunc
	runDone   chan struct{} // closed when in-flight run finishes
}

// NewOrchestrator builds a sync orchestrator. importer and store must be non-nil.
func NewOrchestrator(store RunStore, boardImp BoardImporter, cfg Config) *Orchestrator {
	if cfg.Logger == nil {
		cfg.Logger = slog.Default()
	}
	return &Orchestrator{
		store:    store,
		importer: boardImp,
		cfg:      cfg,
		lifeCtx:  context.Background(),
	}
}

// SetLifecycle sets the process context used for in-flight sync cancellation.
// Call once from cmd/trello-sync with the signal-derived context.
func (o *Orchestrator) SetLifecycle(ctx context.Context) {
	if ctx == nil {
		ctx = context.Background()
	}
	o.mu.Lock()
	o.lifeCtx = ctx
	o.mu.Unlock()
}

// ReclaimOrphans marks leftover running rows as failed (call once on process start).
func (o *Orchestrator) ReclaimOrphans(ctx context.Context) (int64, error) {
	n, err := o.store.ReclaimOrphanSyncRuns(ctx)
	if err != nil {
		return 0, err
	}
	if n > 0 {
		o.cfg.Logger.Warn("reclaimed orphan sync runs", "count", n)
	}
	return n, nil
}

// AlreadyRunningError carries the active run id for single-flight conflicts.
type AlreadyRunningError struct {
	RunID string
}

func (e *AlreadyRunningError) Error() string {
	return fmt.Sprintf("%v: run_id=%s", ErrAlreadyRunning, e.RunID)
}

func (e *AlreadyRunningError) Unwrap() error { return ErrAlreadyRunning }

// Start begins an async sync run. Returns run_id. If already running, returns *AlreadyRunningError.
// reqCtx is only used for short DB lookups/insert; the run itself uses SetLifecycle context.
func (o *Orchestrator) Start(reqCtx context.Context) (runID string, err error) {
	id, _, err := o.begin(reqCtx)
	return id, err
}

// Run starts a sync and blocks until it reaches a terminal status.
// Returns the run id. On terminal failure/cancel returns ErrRunFailed / ErrRunCancelled
// (possibly wrapped). Single-flight conflicts return *AlreadyRunningError.
func (o *Orchestrator) Run(ctx context.Context) (runID string, err error) {
	id, done, err := o.begin(ctx)
	if err != nil {
		return "", err
	}
	select {
	case <-done:
	case <-ctx.Done():
		o.CancelInFlight()
		<-done
	}

	lookupCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	row, err := o.store.GetSyncRun(lookupCtx, id)
	if err != nil {
		return id, err
	}
	if row == nil {
		return id, fmt.Errorf("sync: missing terminal status for run_id=%s", id)
	}
	switch row.Status {
	case postgres.SyncRunSuccess:
		return id, nil
	case postgres.SyncRunCancelled:
		return id, ErrRunCancelled
	case postgres.SyncRunFailed:
		return id, ErrRunFailed
	default:
		return id, fmt.Errorf("sync: unexpected status %q for run_id=%s", row.Status, id)
	}
}

// begin inserts a running row and starts execute in a goroutine.
// done is closed when the run finishes (always non-nil on success).
func (o *Orchestrator) begin(reqCtx context.Context) (runID string, done <-chan struct{}, err error) {
	if o.cfg.BoardID == "" {
		return "", nil, fmt.Errorf("sync: TRELLO_BOARD_ID is not configured")
	}
	if o.importer == nil || o.store == nil {
		return "", nil, fmt.Errorf("sync: importer and store are required")
	}

	o.mu.Lock()
	defer o.mu.Unlock()

	if o.running {
		return "", nil, &AlreadyRunningError{RunID: o.runID}
	}
	// Defense in depth: DB may still show running after crash before reclaim.
	if row, err := o.store.GetRunningSyncRun(reqCtx); err != nil {
		return "", nil, err
	} else if row != nil {
		return "", nil, &AlreadyRunningError{RunID: row.ID}
	}

	started := time.Now().UTC()
	id, err := o.store.InsertSyncRun(reqCtx, started)
	if err != nil {
		return "", nil, err
	}

	life := o.lifeCtx
	if life == nil {
		life = context.Background()
	}
	runCtx, cancel := context.WithCancel(life)
	ch := make(chan struct{})
	o.running = true
	o.runID = id
	o.runCancel = cancel
	o.runDone = ch

	go o.execute(runCtx, id, ch)
	return id, ch, nil
}

func (o *Orchestrator) execute(ctx context.Context, runID string, done chan struct{}) {
	defer close(done)
	defer o.clearRunning(runID)

	counts := postgres.SyncRunCounts{}
	result, err := o.importer.ImportBoard(ctx, importer.ImportBoardOptions{
		BoardID:          o.cfg.BoardID,
		ActionsPageLimit: o.cfg.ActionsPageSize,
	})

	counts.ProcessedCount = result.Actions.ActionsSeen +
		result.Entities.Lists + result.Entities.Members + result.Entities.Labels + result.Entities.Cards
	counts.CreatedCount = int(result.Actions.ActionsInserted)
	counts.UpdatedCount = result.Entities.Lists + result.Entities.Members + result.Entities.Labels + result.Entities.Cards
	if result.Entities.BoardID != "" {
		counts.UpdatedCount++ // board upsert
	}

	status := postgres.SyncRunSuccess
	var summary *string
	if err != nil {
		if errors.Is(err, context.Canceled) || errors.Is(ctx.Err(), context.Canceled) {
			status = postgres.SyncRunCancelled
			msg := "cancelled"
			summary = &msg
		} else {
			status = postgres.SyncRunFailed
			counts.ErrorCount = 1
			msg := redactErrorSummary(err, o.cfg.APIKey, o.cfg.Token)
			summary = &msg
		}
		o.cfg.Logger.Error("sync run finished with error", "run_id", runID, "status", status, "err", msgOrEmpty(summary))
	} else {
		o.cfg.Logger.Info("sync run succeeded", "run_id", runID, "processed", counts.ProcessedCount)
	}

	finishCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if ferr := o.store.FinishSyncRun(finishCtx, runID, status, counts, summary); ferr != nil {
		o.cfg.Logger.Error("failed to finish sync run", "run_id", runID, "err", ferr)
	}
}

func (o *Orchestrator) clearRunning(runID string) {
	o.mu.Lock()
	defer o.mu.Unlock()
	if o.runID == runID {
		o.running = false
		o.runID = ""
		if o.runCancel != nil {
			o.runCancel()
			o.runCancel = nil
		}
		o.runDone = nil
	}
}

// CancelInFlight cancels the active run context (graceful shutdown → cancelled).
func (o *Orchestrator) CancelInFlight() {
	o.mu.Lock()
	cancel := o.runCancel
	done := o.runDone
	o.mu.Unlock()
	if cancel != nil {
		cancel()
	}
	if done != nil {
		select {
		case <-done:
		case <-time.After(15 * time.Second):
		}
	}
}

// Running reports whether an in-process sync is active.
func (o *Orchestrator) Running() (bool, string) {
	o.mu.Lock()
	defer o.mu.Unlock()
	return o.running, o.runID
}

func redactErrorSummary(err error, secrets ...string) string {
	if err == nil {
		return ""
	}
	msg := err.Error()
	for _, s := range secrets {
		if s == "" {
			continue
		}
		msg = strings.ReplaceAll(msg, s, "***")
	}
	// Re-run URL redaction for any leaked query fragments.
	msg = trello.RedactURL(msg, secrets...)
	if len(msg) > 500 {
		msg = msg[:500] + "…"
	}
	return msg
}

func msgOrEmpty(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}
