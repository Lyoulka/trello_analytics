package workflow_test

import (
	"context"
	"os"
	"testing"
	"time"

	"trello_analytics_back/internal/storage/postgres"
	"trello_analytics_back/internal/workflow"

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
			WHERE table_schema = 'public' AND table_name = 'workflow_settings'
		)
	`).Scan(&exists)
	if err != nil || !exists {
		t.Skipf("workflow_settings missing; run migrate-up for 000003 first")
	}
	return pool
}

// cleanupTestBoard deletes board-scoped rows in FK-safe order.
// Each statement is a separate Exec: pgx prepared statements reject multi-command SQL
// (SQLSTATE 42601), and ignoring that error leaves prior Save data for the next -count run.
func cleanupTestBoard(t *testing.T, ctx context.Context, pool *pgxpool.Pool, trelloBoardID string) {
	t.Helper()
	stmts := []string{
		`DELETE FROM workflow_settings_revisions WHERE board_id IN (SELECT id FROM boards WHERE trello_board_id = $1)`,
		`DELETE FROM workflow_settings WHERE board_id IN (SELECT id FROM boards WHERE trello_board_id = $1)`,
		`DELETE FROM card_labels WHERE card_id IN (SELECT id FROM cards WHERE board_id IN (SELECT id FROM boards WHERE trello_board_id = $1))`,
		`DELETE FROM cards WHERE board_id IN (SELECT id FROM boards WHERE trello_board_id = $1)`,
		`DELETE FROM labels WHERE board_id IN (SELECT id FROM boards WHERE trello_board_id = $1)`,
		`DELETE FROM trello_lists WHERE board_id IN (SELECT id FROM boards WHERE trello_board_id = $1)`,
		`DELETE FROM boards WHERE trello_board_id = $1`,
	}
	for _, q := range stmts {
		if _, err := pool.Exec(ctx, q, trelloBoardID); err != nil {
			t.Fatalf("cleanup %q: %v", q, err)
		}
	}
}

func TestLoadSave_AuditAndBugLabels(t *testing.T) {
	pool := openTestDB(t)
	ctx := context.Background()
	store := postgres.NewStore(pool)

	// Isolate: use a dedicated trello board id.
	const trelloBoard = "wf-test-board-1"
	cleanupTestBoard(t, ctx, pool, trelloBoard)
	t.Cleanup(func() { cleanupTestBoard(t, context.Background(), pool, trelloBoard) })

	boardID, err := store.UpsertBoard(ctx, postgres.BoardUpsert{TrelloBoardID: trelloBoard, Name: "WF Test"})
	if err != nil {
		t.Fatalf("board: %v", err)
	}
	lists, err := store.UpsertLists(ctx, boardID, []postgres.ListUpsert{
		{TrelloListID: "L-ready", Name: "Ready"},
		{TrelloListID: "L-done", Name: "Done"},
	})
	if err != nil {
		t.Fatalf("lists: %v", err)
	}
	labels, err := store.UpsertLabels(ctx, boardID, []postgres.LabelUpsert{
		{TrelloLabelID: "LB-bug", Name: "Bug"},
		{TrelloLabelID: "LB-feat", Name: "Feature"},
	})
	if err != nil {
		t.Fatalf("labels: %v", err)
	}

	svc := &workflow.Service{Store: store}
	loaded, err := svc.Load(ctx, boardID)
	if err != nil {
		t.Fatalf("load empty: %v", err)
	}
	if loaded.LeadTimeStartRule != workflow.LeadRuleUnset {
		t.Fatalf("default rule=%q", loaded.LeadTimeStartRule)
	}

	saved, err := svc.Save(ctx, boardID, workflow.Settings{
		LeadTimeStartRule:       workflow.LeadRuleFirstEnterStage,
		LeadTimeStartStage:      "ready",
		CycleStartStage:         "in_progress",
		DoneStages:              []string{"done"},
		WorkStages:              []string{"in_progress"},
		BlockedStages:           []string{"blocked"},
		StaleThresholdSeconds:   3600,
		DataStaleSyncAgeSeconds: 7200,
		Timezone:                "Europe/Moscow",
		ListStages: map[string]string{
			lists["L-ready"]: "ready",
			lists["L-done"]:  "done",
		},
		BugLabelIDs: []string{labels["LB-bug"]},
	}, nil)
	if err != nil {
		t.Fatalf("save: %v", err)
	}
	if saved.UpdatedAt.IsZero() {
		t.Fatal("updated_at must be set")
	}
	if saved.Timezone != "Europe/Moscow" {
		t.Fatalf("tz=%q", saved.Timezone)
	}

	n, err := store.CountWorkflowRevisions(ctx, boardID)
	if err != nil || n != 1 {
		t.Fatalf("revisions=%d err=%v", n, err)
	}

	again, err := svc.Load(ctx, boardID)
	if err != nil {
		t.Fatalf("reload: %v", err)
	}
	if again.ListStages[lists["L-ready"]] != "ready" {
		t.Fatalf("list stages=%v", again.ListStages)
	}
	if len(again.BugLabelIDs) != 1 || again.BugLabelIDs[0] != labels["LB-bug"] {
		t.Fatalf("bug labels=%v", again.BugLabelIDs)
	}

	labelRows, err := store.ListLabelsByBoard(ctx, boardID)
	if err != nil {
		t.Fatalf("labels: %v", err)
	}
	var bugTrue, featFalse bool
	for _, l := range labelRows {
		if l.ID == labels["LB-bug"] && l.IsBugLabel {
			bugTrue = true
		}
		if l.ID == labels["LB-feat"] && !l.IsBugLabel {
			featFalse = true
		}
	}
	if !bugTrue || !featFalse {
		t.Fatalf("bug flags not applied: %+v", labelRows)
	}

	// Second save appends another revision.
	_, err = svc.Save(ctx, boardID, again, nil)
	if err != nil {
		t.Fatalf("save2: %v", err)
	}
	n, err = store.CountWorkflowRevisions(ctx, boardID)
	if err != nil || n != 2 {
		t.Fatalf("revisions after 2nd save=%d err=%v", n, err)
	}
}

func TestSave_RejectsInvalidLeadRule(t *testing.T) {
	pool := openTestDB(t)
	store := postgres.NewStore(pool)
	svc := &workflow.Service{Store: store}
	_, err := svc.Save(context.Background(), "00000000-0000-0000-0000-000000000001", workflow.Settings{
		LeadTimeStartRule: "nope",
	}, nil)
	if err == nil {
		t.Fatal("expected invalid lead_time_start_rule error")
	}
}
