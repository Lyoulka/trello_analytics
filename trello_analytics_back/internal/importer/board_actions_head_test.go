package importer_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"trello_analytics_back/internal/importer"
	"trello_analytics_back/internal/storage/postgres"
	"trello_analytics_back/internal/trello"
)

func TestImportBoardActionsHeadCatchUp_InsertsNewWithoutMovingCursor(t *testing.T) {
	pool := openTestDB(t)
	const boardID = "boardHead1"
	isolateTestBoard(t, pool, boardID)
	store := postgres.NewStore(pool)
	ctx := context.Background()

	seed := fakeActions(boardID, 6)
	srv := newFakeBoardActionsServer(t, boardID, seed)
	imp := &importer.BoardActionsImporter{Client: trelloClientForServer(srv), Store: store}

	r1, err := imp.ImportBoardActions(ctx, importer.ImportBoardActionsOptions{
		BoardID:   boardID,
		PageLimit: 2,
	})
	if err != nil {
		t.Fatalf("primary seed: %v", err)
	}
	srv.Close()
	oldestCursor := r1.CursorValue
	if oldestCursor != seed[5].ID {
		t.Fatalf("seed cursor want %s got %s", seed[5].ID, oldestCursor)
	}

	newer := newerHeadActions(boardID, 2)
	live := append(newer, seed...)
	srv2 := newFakeBoardActionsServer(t, boardID, live)
	defer srv2.Close()
	imp.Client = trelloClientForServer(srv2)

	head, err := imp.ImportBoardActionsHeadCatchUp(ctx, importer.ImportBoardActionsOptions{
		BoardID:   boardID,
		PageLimit: 2,
	})
	if err != nil {
		t.Fatalf("head catch-up: %v", err)
	}
	if head.HeadSkipped {
		t.Fatal("expected head catch-up to run")
	}
	if head.ActionsInserted != 2 {
		t.Fatalf("head inserted: want 2 got %d (seen=%d pages=%d)", head.ActionsInserted, head.ActionsSeen, head.PagesCommitted)
	}

	cur, found, err := store.GetSyncCursor(ctx, postgres.ResourceBoardActions)
	if err != nil || !found {
		t.Fatalf("cursor after head: found=%v err=%v", found, err)
	}
	if cur != oldestCursor {
		t.Fatalf("head must not move oldest cursor: before=%s after=%s", oldestCursor, cur)
	}

	if n := countActionsForBoard(t, ctx, pool, boardID); n != 8 {
		t.Fatalf("want 8 actions for board, got %d", n)
	}
	ids := listActionIDsForBoard(t, ctx, pool, boardID)
	for _, a := range newer {
		if !contains(ids, a.ID) {
			t.Fatalf("missing newer action %s in %v", a.ID, ids)
		}
	}
}

func TestImportBoardActionsHeadCatchUp_StopsOnAllDuplicatePage(t *testing.T) {
	pool := openTestDB(t)
	const boardID = "boardHeadDup"
	isolateTestBoard(t, pool, boardID)
	store := postgres.NewStore(pool)
	ctx := context.Background()

	all := fakeActions(boardID, 4)
	srv := newFakeBoardActionsServer(t, boardID, all)
	defer srv.Close()
	imp := &importer.BoardActionsImporter{Client: trelloClientForServer(srv), Store: store}

	if _, err := imp.ImportBoardActions(ctx, importer.ImportBoardActionsOptions{
		BoardID:   boardID,
		PageLimit: 2,
	}); err != nil {
		t.Fatalf("seed: %v", err)
	}

	head, err := imp.ImportBoardActionsHeadCatchUp(ctx, importer.ImportBoardActionsOptions{
		BoardID:   boardID,
		PageLimit: 2,
	})
	if err != nil {
		t.Fatalf("head: %v", err)
	}
	if head.ActionsInserted != 0 {
		t.Fatalf("all-duplicate head want inserted=0, got %d", head.ActionsInserted)
	}
	if head.PagesCommitted != 1 {
		t.Fatalf("want stop after first all-duplicate page, pages=%d", head.PagesCommitted)
	}
}

func TestImportBoardActionsHeadCatchUp_SkippedWhenEmpty(t *testing.T) {
	// Do not wipe shared card_actions — use a stub store with Count=0.
	store := &countZeroActionsStore{}
	srv := newFakeBoardActionsServer(t, "boardHeadEmpty", fakeActions("boardHeadEmpty", 2))
	defer srv.Close()
	imp := &importer.BoardActionsImporter{Client: trelloClientForServer(srv), Store: store}

	head, err := imp.ImportBoardActionsHeadCatchUp(context.Background(), importer.ImportBoardActionsOptions{
		BoardID:   "boardHeadEmpty",
		PageLimit: 2,
	})
	if err != nil {
		t.Fatalf("head: %v", err)
	}
	if !head.HeadSkipped {
		t.Fatalf("expected HeadSkipped on empty DB, got %+v", head)
	}
	if head.ActionsInserted != 0 || head.PagesCommitted != 0 {
		t.Fatalf("skipped head must not fetch/insert: %+v", head)
	}
}

// countZeroActionsStore only implements CountCardActions=0 for the head skip gate.
type countZeroActionsStore struct{}

func (countZeroActionsStore) GetSyncCursor(context.Context, string) (string, bool, error) {
	return "", false, nil
}
func (countZeroActionsStore) UpsertSyncCursor(context.Context, string, string) error { return nil }
func (countZeroActionsStore) CommitBoardActionsPage(context.Context, string, []postgres.CardActionInsert, string) (int64, error) {
	return 0, nil
}
func (countZeroActionsStore) InsertCardActionsPage(context.Context, []postgres.CardActionInsert) (int64, error) {
	return 0, nil
}
func (countZeroActionsStore) CountCardActions(context.Context) (int64, error) { return 0, nil }

// newerHeadActions builds n actions newer than fakeActions base, with ids matching boardID-% filter.
func newerHeadActions(boardID string, n int) []trello.Action {
	out := make([]trello.Action, n)
	base := time.Date(2024, 6, 1, 18, 0, 0, 0, time.UTC) // newer than fakeActions base (12:00)
	for i := 0; i < n; i++ {
		out[i] = trello.Action{
			ID:   fmt.Sprintf("%s-act-n%02d", boardID, i+1),
			Type: "updateCard",
			Date: base.Add(-time.Duration(i) * time.Minute).Format(time.RFC3339Nano),
			Data: map[string]any{"card": map[string]any{"id": boardID + "-card-1"}},
		}
	}
	return out
}
