package importer_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"

	"trello_analytics_back/internal/storage/postgres"
	"trello_analytics_back/internal/trello"

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
		t.Skipf("postgres unavailable (%v); apply migrations and set DATABASE_URL/TEST_DATABASE_URL", err)
	}
	t.Cleanup(pool.Close)

	var exists bool
	err = pool.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1 FROM information_schema.tables
			WHERE table_schema = 'public' AND table_name = 'card_actions'
		)
	`).Scan(&exists)
	if err != nil || !exists {
		t.Skipf("card_actions missing; run migrate-up first (err=%v exists=%v)", err, exists)
	}
	return pool
}

// isolateTestBoard scopes integration tests to one synthetic trello_board_id.
// Never TRUNCATE shared tables — that would wipe the local app DB (incl. workflow_settings via CASCADE).
// Also snapshots/restores the global board_actions sync cursor so live sync progress is preserved.
func isolateTestBoard(t *testing.T, pool *pgxpool.Pool, trelloBoardID string) {
	t.Helper()
	if trelloBoardID == "" {
		t.Fatal("isolateTestBoard: empty trelloBoardID")
	}
	ctx := context.Background()
	guard := snapshotBoardActionsCursor(t, ctx, pool)
	cleanupBoardScopedData(t, ctx, pool, trelloBoardID)
	// Actions importer resumes from the global board_actions cursor; clear it for the test
	// so we never page with a live/foreign before= id. Restored in Cleanup.
	if _, err := pool.Exec(ctx, `DELETE FROM sync_cursors WHERE resource_type = $1`, postgres.ResourceBoardActions); err != nil {
		t.Fatalf("clear board_actions cursor for test: %v", err)
	}
	t.Cleanup(func() {
		cleanupBoardScopedData(t, context.Background(), pool, trelloBoardID)
		guard.restore(t, context.Background(), pool)
	})
}

type boardActionsCursorSnapshot struct {
	hadRow         bool
	cursorValue    *string
	lastSuccessAt  *time.Time
	updatedAt      *time.Time
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
		t.Fatalf("snapshot board_actions cursor: %v", err)
	}
	snap.hadRow = true
	return snap
}

func (s boardActionsCursorSnapshot) restore(t *testing.T, ctx context.Context, pool *pgxpool.Pool) {
	t.Helper()
	if !s.hadRow {
		if _, err := pool.Exec(ctx, `
			DELETE FROM sync_cursors WHERE resource_type = $1
		`, postgres.ResourceBoardActions); err != nil {
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

// cleanupBoardScopedData removes only rows belonging to the synthetic board (+ prefixed members/actions).
func cleanupBoardScopedData(t *testing.T, ctx context.Context, pool *pgxpool.Pool, trelloBoardID string) {
	t.Helper()
	prefix := trelloBoardID + "-%"
	type stmt struct {
		q    string
		args []any
	}
	stmts := []stmt{
		{
			`DELETE FROM workflow_settings_revisions
			 WHERE board_id IN (SELECT id FROM boards WHERE trello_board_id = $1)`,
			[]any{trelloBoardID},
		},
		{
			`DELETE FROM workflow_settings
			 WHERE board_id IN (SELECT id FROM boards WHERE trello_board_id = $1)`,
			[]any{trelloBoardID},
		},
		// Actions may have NULL card_id; always match synthetic trello_action_id prefix.
		{
			`DELETE FROM card_actions
			 WHERE trello_action_id LIKE $1
			    OR card_id IN (
					SELECT c.id FROM cards c
					JOIN boards b ON b.id = c.board_id
					WHERE b.trello_board_id = $2
				)`,
			[]any{prefix, trelloBoardID},
		},
		{
			`DELETE FROM card_labels
			 WHERE card_id IN (
				SELECT c.id FROM cards c
				JOIN boards b ON b.id = c.board_id
				WHERE b.trello_board_id = $1
			 )`,
			[]any{trelloBoardID},
		},
		{
			`DELETE FROM card_members
			 WHERE card_id IN (
				SELECT c.id FROM cards c
				JOIN boards b ON b.id = c.board_id
				WHERE b.trello_board_id = $1
			 )`,
			[]any{trelloBoardID},
		},
		{
			`DELETE FROM cards
			 WHERE board_id IN (SELECT id FROM boards WHERE trello_board_id = $1)`,
			[]any{trelloBoardID},
		},
		{
			`DELETE FROM labels
			 WHERE board_id IN (SELECT id FROM boards WHERE trello_board_id = $1)`,
			[]any{trelloBoardID},
		},
		{
			`DELETE FROM trello_lists
			 WHERE board_id IN (SELECT id FROM boards WHERE trello_board_id = $1)`,
			[]any{trelloBoardID},
		},
		{
			`DELETE FROM boards WHERE trello_board_id = $1`,
			[]any{trelloBoardID},
		},
		{
			`DELETE FROM members WHERE trello_member_id LIKE $1`,
			[]any{prefix},
		},
	}
	for _, s := range stmts {
		if _, err := pool.Exec(ctx, s.q, s.args...); err != nil {
			t.Fatalf("cleanup board %q: %v\nSQL: %s", trelloBoardID, err, s.q)
		}
	}
}

func countBoardsByTrelloID(t *testing.T, ctx context.Context, pool *pgxpool.Pool, trelloBoardID string) int64 {
	t.Helper()
	var n int64
	if err := pool.QueryRow(ctx, `SELECT COUNT(*) FROM boards WHERE trello_board_id = $1`, trelloBoardID).Scan(&n); err != nil {
		t.Fatalf("count boards: %v", err)
	}
	return n
}

func countListsForBoard(t *testing.T, ctx context.Context, pool *pgxpool.Pool, trelloBoardID string) int64 {
	t.Helper()
	var n int64
	err := pool.QueryRow(ctx, `
		SELECT COUNT(*) FROM trello_lists l
		JOIN boards b ON b.id = l.board_id
		WHERE b.trello_board_id = $1
	`, trelloBoardID).Scan(&n)
	if err != nil {
		t.Fatalf("count lists: %v", err)
	}
	return n
}

func countCardsForBoard(t *testing.T, ctx context.Context, pool *pgxpool.Pool, trelloBoardID string) int64 {
	t.Helper()
	var n int64
	err := pool.QueryRow(ctx, `
		SELECT COUNT(*) FROM cards c
		JOIN boards b ON b.id = c.board_id
		WHERE b.trello_board_id = $1
	`, trelloBoardID).Scan(&n)
	if err != nil {
		t.Fatalf("count cards: %v", err)
	}
	return n
}

func countLabelsForBoard(t *testing.T, ctx context.Context, pool *pgxpool.Pool, trelloBoardID string) int64 {
	t.Helper()
	var n int64
	err := pool.QueryRow(ctx, `
		SELECT COUNT(*) FROM labels l
		JOIN boards b ON b.id = l.board_id
		WHERE b.trello_board_id = $1
	`, trelloBoardID).Scan(&n)
	if err != nil {
		t.Fatalf("count labels: %v", err)
	}
	return n
}

func countMembersForBoardPrefix(t *testing.T, ctx context.Context, pool *pgxpool.Pool, trelloBoardID string) int64 {
	t.Helper()
	var n int64
	if err := pool.QueryRow(ctx, `SELECT COUNT(*) FROM members WHERE trello_member_id LIKE $1`, trelloBoardID+"-%").Scan(&n); err != nil {
		t.Fatalf("count members: %v", err)
	}
	return n
}

func countActiveCardMembersForBoard(t *testing.T, ctx context.Context, pool *pgxpool.Pool, trelloBoardID string) int64 {
	t.Helper()
	var n int64
	err := pool.QueryRow(ctx, `
		SELECT COUNT(*) FROM card_members cm
		JOIN cards c ON c.id = cm.card_id
		JOIN boards b ON b.id = c.board_id
		WHERE b.trello_board_id = $1 AND cm.removed_at IS NULL
	`, trelloBoardID).Scan(&n)
	if err != nil {
		t.Fatalf("count active card_members: %v", err)
	}
	return n
}

func countCardMembersTotalForBoard(t *testing.T, ctx context.Context, pool *pgxpool.Pool, trelloBoardID string) int64 {
	t.Helper()
	var n int64
	err := pool.QueryRow(ctx, `
		SELECT COUNT(*) FROM card_members cm
		JOIN cards c ON c.id = cm.card_id
		JOIN boards b ON b.id = c.board_id
		WHERE b.trello_board_id = $1
	`, trelloBoardID).Scan(&n)
	if err != nil {
		t.Fatalf("count card_members: %v", err)
	}
	return n
}

func countCardLabelsForBoard(t *testing.T, ctx context.Context, pool *pgxpool.Pool, trelloBoardID string) int64 {
	t.Helper()
	var n int64
	err := pool.QueryRow(ctx, `
		SELECT COUNT(*) FROM card_labels cl
		JOIN cards c ON c.id = cl.card_id
		JOIN boards b ON b.id = c.board_id
		WHERE b.trello_board_id = $1
	`, trelloBoardID).Scan(&n)
	if err != nil {
		t.Fatalf("count card_labels: %v", err)
	}
	return n
}

func countActionsForBoard(t *testing.T, ctx context.Context, pool *pgxpool.Pool, trelloBoardID string) int64 {
	t.Helper()
	var n int64
	if err := pool.QueryRow(ctx, `SELECT COUNT(*) FROM card_actions WHERE trello_action_id LIKE $1`, trelloBoardID+"-%").Scan(&n); err != nil {
		t.Fatalf("count actions: %v", err)
	}
	return n
}

func listActionIDsForBoard(t *testing.T, ctx context.Context, pool *pgxpool.Pool, trelloBoardID string) []string {
	t.Helper()
	rows, err := pool.Query(ctx, `
		SELECT trello_action_id FROM card_actions
		WHERE trello_action_id LIKE $1
		ORDER BY trello_action_id
	`, trelloBoardID+"-%")
	if err != nil {
		t.Fatalf("list actions: %v", err)
	}
	defer rows.Close()
	var out []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			t.Fatalf("scan action id: %v", err)
		}
		out = append(out, id)
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("list actions rows: %v", err)
	}
	return out
}

func fakeActions(boardID string, n int) []trello.Action {
	out := make([]trello.Action, n)
	base := time.Date(2024, 6, 1, 12, 0, 0, 0, time.UTC)
	for i := 0; i < n; i++ {
		out[i] = trello.Action{
			ID:   fmt.Sprintf("%s-act-%02d", boardID, i+1),
			Type: "updateCard",
			Date: base.Add(-time.Duration(i) * time.Hour).Format(time.RFC3339Nano),
			Data: map[string]any{"card": map[string]any{"id": boardID + "-card-1"}},
		}
	}
	return out
}

func newFakeBoardActionsServer(t *testing.T, boardID string, all []trello.Action) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/boards/"+boardID+"/actions" {
			http.NotFound(w, r)
			return
		}
		limit := 50
		if v := r.URL.Query().Get("limit"); v != "" {
			n, err := strconv.Atoi(v)
			if err != nil {
				http.Error(w, "bad limit", http.StatusBadRequest)
				return
			}
			limit = n
		}
		before := r.URL.Query().Get("before")
		start := 0
		if before != "" {
			start = -1
			for i, a := range all {
				if a.ID == before {
					start = i + 1
					break
				}
			}
			if start < 0 {
				http.Error(w, "unknown before", http.StatusBadRequest)
				return
			}
		}
		end := start + limit
		if end > len(all) {
			end = len(all)
		}
		page := []trello.Action{}
		if start < len(all) {
			page = all[start:end]
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(page)
	}))
}

// fakeBoardSnapshot is a minimal coherent board payload for entity + actions tests.
type fakeBoardSnapshot struct {
	Board   trello.Board
	Lists   []trello.List
	Members []trello.Member
	Labels  []trello.Label
	Cards   []trello.Card
	Actions []trello.Action
}

func newFakeBoardServer(t *testing.T, snap fakeBoardSnapshot) *httptest.Server {
	t.Helper()
	boardID := snap.Board.ID
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		path := r.URL.Path
		switch {
		case path == "/boards/"+boardID:
			_ = json.NewEncoder(w).Encode(snap.Board)
		case path == "/boards/"+boardID+"/lists":
			_ = json.NewEncoder(w).Encode(snap.Lists)
		case path == "/boards/"+boardID+"/members":
			_ = json.NewEncoder(w).Encode(snap.Members)
		case path == "/boards/"+boardID+"/labels":
			_ = json.NewEncoder(w).Encode(snap.Labels)
		case path == "/boards/"+boardID+"/cards":
			_ = json.NewEncoder(w).Encode(snap.Cards)
		case path == "/boards/"+boardID+"/actions":
			serveActionsPage(w, r, snap.Actions)
		default:
			http.NotFound(w, r)
		}
	}))
}

func serveActionsPage(w http.ResponseWriter, r *http.Request, all []trello.Action) {
	limit := 50
	if v := r.URL.Query().Get("limit"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			limit = n
		}
	}
	before := r.URL.Query().Get("before")
	start := 0
	if before != "" {
		start = -1
		for i, a := range all {
			if a.ID == before {
				start = i + 1
				break
			}
		}
		if start < 0 {
			http.Error(w, "unknown before", http.StatusBadRequest)
			return
		}
	}
	end := start + limit
	if end > len(all) {
		end = len(all)
	}
	page := []trello.Action{}
	if start < len(all) {
		page = all[start:end]
	}
	_ = json.NewEncoder(w).Encode(page)
}

func sampleBoardSnapshot(boardID string) fakeBoardSnapshot {
	listOpen := boardID + "-list-open"
	listArch := boardID + "-list-arch"
	mem1 := boardID + "-mem-1"
	mem2 := boardID + "-mem-2"
	lab1 := boardID + "-lab-1"
	card1 := boardID + "-card-1"
	card2 := boardID + "-card-2"
	return fakeBoardSnapshot{
		Board: trello.Board{ID: boardID, Name: "Sample Board"},
		Lists: []trello.List{
			{ID: listOpen, Name: "Todo", IDBoard: boardID, Closed: false},
			{ID: listArch, Name: "Old", IDBoard: boardID, Closed: true},
		},
		Members: []trello.Member{
			{ID: mem1, FullName: "Ada Lovelace", Username: "ada"},
			{ID: mem2, FullName: "Inactive", Username: "gone", Deactivated: true},
		},
		Labels: []trello.Label{
			{ID: lab1, IDBoard: boardID, Name: "bug", Color: "red"},
		},
		Cards: []trello.Card{
			{
				ID: card1, Name: "Open card", IDList: listOpen, IDBoard: boardID, Closed: false,
				DateLastActivity: "2024-06-02T10:00:00.000Z",
				IDMembers:        []string{mem1},
				IDLabels:         []string{lab1},
			},
			{
				ID: card2, Name: "Archived card", IDList: listArch, IDBoard: boardID, Closed: true,
				DateLastActivity: "2024-05-01T10:00:00.000Z",
			},
		},
		Actions: fakeActions(boardID, 4),
	}
}

func contains(ids []string, want string) bool {
	for _, id := range ids {
		if id == want {
			return true
		}
	}
	return false
}

func trelloClientForServer(srv *httptest.Server) *trello.Client {
	c := trello.New("test-key", "test-token")
	c.BaseURL = strings.TrimRight(srv.URL, "/")
	c.HTTP = srv.Client()
	return c
}
