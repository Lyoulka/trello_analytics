package importer_test

import (
	"context"
	"fmt"
	"testing"

	"trello_analytics_back/internal/importer"
	"trello_analytics_back/internal/storage/postgres"
)

// Integration: fake Trello HTTP + real Postgres (migrations assumed applied).
// Tests isolate a synthetic board id and must not TRUNCATE shared app tables.

func TestImportBoardActions_ResumeAfterInterrupt(t *testing.T) {
	pool := openTestDB(t)
	const boardID = "boardResume1"
	isolateTestBoard(t, pool, boardID)
	store := postgres.NewStore(pool)

	const pageLimit = 2
	// 3 full pages → 6 actions (newest-first ids boardResume1-act-01 …).
	all := fakeActions(boardID, 6)
	srv := newFakeBoardActionsServer(t, boardID, all)
	defer srv.Close()

	client := trelloClientForServer(srv)

	imp := &importer.BoardActionsImporter{Client: client, Store: store}
	ctx := context.Background()

	// Phase 1: interrupt after 2 committed pages.
	r1, err := imp.ImportBoardActions(ctx, importer.ImportBoardActionsOptions{
		BoardID:        boardID,
		PageLimit:      pageLimit,
		StopAfterPages: 2,
	})
	if err != nil {
		t.Fatalf("phase1 import: %v", err)
	}
	if !r1.StoppedEarly || r1.PagesCommitted != 2 {
		t.Fatalf("phase1: want StoppedEarly with 2 pages, got %+v", r1)
	}
	if r1.ActionsSeen != 4 || r1.ActionsInserted != 4 {
		t.Fatalf("phase1: want 4 seen/inserted, got seen=%d inserted=%d", r1.ActionsSeen, r1.ActionsInserted)
	}
	wantCursor := all[3].ID // last of page 2
	if r1.CursorValue != wantCursor {
		t.Fatalf("phase1 cursor: want %s got %s", wantCursor, r1.CursorValue)
	}

	cur, found, err := store.GetSyncCursor(ctx, postgres.ResourceBoardActions)
	if err != nil || !found || cur != wantCursor {
		t.Fatalf("DB cursor after phase1: found=%v value=%q err=%v", found, cur, err)
	}
	idsAfterPhase1 := listActionIDsForBoard(t, ctx, pool, boardID)
	if len(idsAfterPhase1) != 4 {
		t.Fatalf("phase1 rows: want 4 got %d (%v)", len(idsAfterPhase1), idsAfterPhase1)
	}

	// Phase 2: resume from DB cursor; no StopAfter.
	r2, err := imp.ImportBoardActions(ctx, importer.ImportBoardActionsOptions{
		BoardID:   boardID,
		PageLimit: pageLimit,
	})
	if err != nil {
		t.Fatalf("phase2 import: %v", err)
	}
	if r2.StoppedEarly || r2.PagesCommitted != 1 {
		t.Fatalf("phase2: want 1 more page without early stop, got %+v", r2)
	}
	if r2.ActionsSeen != 2 || r2.ActionsInserted != 2 {
		t.Fatalf("phase2: want 2 new actions, got seen=%d inserted=%d", r2.ActionsSeen, r2.ActionsInserted)
	}
	if r2.CursorValue != all[5].ID {
		t.Fatalf("phase2 cursor: want %s got %s", all[5].ID, r2.CursorValue)
	}

	idsFinal := listActionIDsForBoard(t, ctx, pool, boardID)
	if len(idsFinal) != 6 {
		t.Fatalf("final unique rows: want 6 got %d (%v)", len(idsFinal), idsFinal)
	}
	// Prior rows preserved (phase1 subset still present).
	for _, id := range idsAfterPhase1 {
		if !contains(idsFinal, id) {
			t.Fatalf("phase1 id %s missing after resume", id)
		}
	}

	// Reset cursor to empty and re-fetch from start: ON CONFLICT must not duplicate rows.
	if err := store.UpsertSyncCursor(ctx, postgres.ResourceBoardActions, ""); err != nil {
		t.Fatalf("reset cursor: %v", err)
	}
	r3, err := imp.ImportBoardActions(ctx, importer.ImportBoardActionsOptions{
		BoardID:   boardID,
		PageLimit: pageLimit,
	})
	if err != nil {
		t.Fatalf("phase3 re-import: %v", err)
	}
	if n := countActionsForBoard(t, ctx, pool, boardID); n != 6 {
		t.Fatalf("after re-import want 6 rows, got %d (phase3 inserted=%d seen=%d)", n, r3.ActionsInserted, r3.ActionsSeen)
	}
	if r3.ActionsInserted != 0 {
		t.Fatalf("re-import should insert 0 new rows via ON CONFLICT, got %d", r3.ActionsInserted)
	}
}

func TestImportBoardActions_EmptyBoardCreatesCursor(t *testing.T) {
	pool := openTestDB(t)
	const boardID = "boardEmpty1"
	isolateTestBoard(t, pool, boardID)
	store := postgres.NewStore(pool)

	srv := newFakeBoardActionsServer(t, boardID, nil)
	defer srv.Close()

	imp := &importer.BoardActionsImporter{Client: trelloClientForServer(srv), Store: store}
	ctx := context.Background()

	r, err := imp.ImportBoardActions(ctx, importer.ImportBoardActionsOptions{
		BoardID:   boardID,
		PageLimit: 50,
	})
	if err != nil {
		t.Fatalf("empty board import: %v", err)
	}
	if r.PagesCommitted != 0 || r.ActionsSeen != 0 || r.ActionsInserted != 0 || r.StoppedEarly {
		t.Fatalf("empty board result: %+v", r)
	}

	has, err := store.HasSyncCursor(ctx, postgres.ResourceBoardActions)
	if err != nil {
		t.Fatal(err)
	}
	if !has {
		t.Fatal("expected sync_cursors row after empty-board success (primary_import_incomplete gate)")
	}
	cur, found, err := store.GetSyncCursor(ctx, postgres.ResourceBoardActions)
	if err != nil || !found {
		t.Fatalf("GetSyncCursor: found=%v err=%v", found, err)
	}
	if cur != "" {
		t.Fatalf("empty board cursor_value want empty/NULL, got %q", cur)
	}
	if n := countActionsForBoard(t, ctx, pool, boardID); n != 0 {
		t.Fatalf("card_actions: want 0 got %d", n)
	}
}

func TestImportBoardActions_MidPageCommitErrorDoesNotAdvanceCursor(t *testing.T) {
	pool := openTestDB(t)
	const boardID = "boardFail1"
	isolateTestBoard(t, pool, boardID)
	base := postgres.NewStore(pool)

	const pageLimit = 2
	all := fakeActions(boardID, 4)
	srv := newFakeBoardActionsServer(t, boardID, all)
	defer srv.Close()

	client := trelloClientForServer(srv)

	ctx := context.Background()
	okImp := &importer.BoardActionsImporter{Client: client, Store: base}
	r1, err := okImp.ImportBoardActions(ctx, importer.ImportBoardActionsOptions{
		BoardID:        boardID,
		PageLimit:      pageLimit,
		StopAfterPages: 1,
	})
	if err != nil {
		t.Fatalf("seed page: %v", err)
	}
	cursorBefore := r1.CursorValue
	if cursorBefore != all[1].ID {
		t.Fatalf("seed cursor want %s got %s", all[1].ID, cursorBefore)
	}

	failStore := &failNextCommitStore{inner: base, failRemaining: 1}
	failImp := &importer.BoardActionsImporter{Client: client, Store: failStore}
	_, err = failImp.ImportBoardActions(ctx, importer.ImportBoardActionsOptions{
		BoardID:   boardID,
		PageLimit: pageLimit,
	})
	if err == nil {
		t.Fatal("expected commit error on next page")
	}

	cur, found, err := base.GetSyncCursor(ctx, postgres.ResourceBoardActions)
	if err != nil || !found {
		t.Fatalf("cursor after failed page: found=%v err=%v", found, err)
	}
	if cur != cursorBefore {
		t.Fatalf("cursor advanced on mid-page error: before=%s after=%s", cursorBefore, cur)
	}
	if n := countActionsForBoard(t, ctx, pool, boardID); n != 2 {
		t.Fatalf("rows after failed page: want 2 (page1 only) got %d", n)
	}
}

type failNextCommitStore struct {
	inner         *postgres.Store
	failRemaining int
}

func (f *failNextCommitStore) GetSyncCursor(ctx context.Context, resourceType string) (string, bool, error) {
	return f.inner.GetSyncCursor(ctx, resourceType)
}

func (f *failNextCommitStore) UpsertSyncCursor(ctx context.Context, resourceType, cursorValue string) error {
	return f.inner.UpsertSyncCursor(ctx, resourceType, cursorValue)
}

func (f *failNextCommitStore) CommitBoardActionsPage(
	ctx context.Context,
	resourceType string,
	actions []postgres.CardActionInsert,
	cursorValue string,
) (int64, error) {
	if f.failRemaining > 0 {
		f.failRemaining--
		return 0, fmt.Errorf("injected commit failure")
	}
	return f.inner.CommitBoardActionsPage(ctx, resourceType, actions, cursorValue)
}

func (f *failNextCommitStore) InsertCardActionsPage(ctx context.Context, actions []postgres.CardActionInsert) (int64, error) {
	return f.inner.InsertCardActionsPage(ctx, actions)
}

func (f *failNextCommitStore) CountCardActions(ctx context.Context) (int64, error) {
	return f.inner.CountCardActions(ctx)
}
