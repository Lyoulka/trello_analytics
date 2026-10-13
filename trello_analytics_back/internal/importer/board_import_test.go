package importer_test

import (
	"context"
	"testing"

	"trello_analytics_back/internal/importer"
	"trello_analytics_back/internal/storage/postgres"
)

func TestImportEntities_AndIdempotentReimport(t *testing.T) {
	pool := openTestDB(t)
	const boardID = "boardEnt1"
	isolateTestBoard(t, pool, boardID)
	store := postgres.NewStore(pool)

	snap := sampleBoardSnapshot(boardID)
	// Mutate names on second pass to verify DO UPDATE.
	srv := newFakeBoardServer(t, snap)
	defer srv.Close()
	client := trelloClientForServer(srv)

	ent := &importer.EntitiesImporter{Client: client, Store: store}
	ctx := context.Background()

	r1, err := ent.ImportEntities(ctx, importer.ImportEntitiesOptions{BoardID: boardID})
	if err != nil {
		t.Fatalf("first entity import: %v", err)
	}
	if r1.Lists != 2 || r1.Members != 2 || r1.Labels != 1 || r1.Cards != 2 || r1.ArchivedCards != 1 {
		t.Fatalf("unexpected entity counts: %+v", r1)
	}

	assertCounts := func(t *testing.T) {
		t.Helper()
		checks := []struct {
			name string
			got  int64
			want int64
		}{
			{"boards", countBoardsByTrelloID(t, ctx, pool, boardID), 1},
			{"lists", countListsForBoard(t, ctx, pool, boardID), 2},
			{"members", countMembersForBoardPrefix(t, ctx, pool, boardID), 2},
			{"labels", countLabelsForBoard(t, ctx, pool, boardID), 1},
			{"cards", countCardsForBoard(t, ctx, pool, boardID), 2},
			{"active_card_members", countActiveCardMembersForBoard(t, ctx, pool, boardID), 1},
			{"card_labels", countCardLabelsForBoard(t, ctx, pool, boardID), 1},
		}
		for _, c := range checks {
			if c.got != c.want {
				t.Fatalf("%s: want %d got %d", c.name, c.want, c.got)
			}
		}
	}
	assertCounts(t)

	name, err := store.GetBoardNameByTrelloID(ctx, boardID)
	if err != nil || name != "Sample Board" {
		t.Fatalf("board name: %q err=%v", name, err)
	}

	// Re-import with updated names — row counts must stay stable; names must update.
	snap.Board.Name = "Sample Board v2"
	snap.Cards[0].Name = "Open card renamed"
	srv2 := newFakeBoardServer(t, snap)
	defer srv2.Close()
	ent.Client = trelloClientForServer(srv2)

	r2, err := ent.ImportEntities(ctx, importer.ImportEntitiesOptions{BoardID: boardID})
	if err != nil {
		t.Fatalf("second entity import: %v", err)
	}
	if r2.Cards != 2 {
		t.Fatalf("second import cards: %+v", r2)
	}
	assertCounts(t)

	name, err = store.GetBoardNameByTrelloID(ctx, boardID)
	if err != nil || name != "Sample Board v2" {
		t.Fatalf("board name after update: %q err=%v", name, err)
	}
	cardName, err := store.GetCardNameByTrelloID(ctx, boardID+"-card-1")
	if err != nil || cardName != "Open card renamed" {
		t.Fatalf("card name after update: %q err=%v", cardName, err)
	}
}

func TestImportEntities_CardMembersAndLabelsSnapshot(t *testing.T) {
	pool := openTestDB(t)
	const boardID = "boardLinks1"
	isolateTestBoard(t, pool, boardID)
	store := postgres.NewStore(pool)

	snap := sampleBoardSnapshot(boardID)
	// Two assignees on open card; archived card stays without links.
	snap.Cards[0].IDMembers = []string{boardID + "-mem-1", boardID + "-mem-2"}
	snap.Cards[0].IDLabels = []string{boardID + "-lab-1"}

	srv := newFakeBoardServer(t, snap)
	defer srv.Close()
	ent := &importer.EntitiesImporter{Client: trelloClientForServer(srv), Store: store}
	ctx := context.Background()

	if _, err := ent.ImportEntities(ctx, importer.ImportEntitiesOptions{BoardID: boardID}); err != nil {
		t.Fatalf("import with links: %v", err)
	}
	if active := countActiveCardMembersForBoard(t, ctx, pool, boardID); active != 2 {
		t.Fatalf("active card_members: want 2 got %d", active)
	}
	if labels := countCardLabelsForBoard(t, ctx, pool, boardID); labels != 1 {
		t.Fatalf("card_labels: want 1 got %d", labels)
	}

	// Soft-remove one member; drop label; re-import must be idempotent on remaining links.
	snap.Cards[0].IDMembers = []string{boardID + "-mem-1"}
	snap.Cards[0].IDLabels = nil
	srv2 := newFakeBoardServer(t, snap)
	defer srv2.Close()
	ent.Client = trelloClientForServer(srv2)

	if _, err := ent.ImportEntities(ctx, importer.ImportEntitiesOptions{BoardID: boardID}); err != nil {
		t.Fatalf("re-import after remove: %v", err)
	}
	if active := countActiveCardMembersForBoard(t, ctx, pool, boardID); active != 1 {
		t.Fatalf("after remove active card_members: want 1 got %d", active)
	}
	if total := countCardMembersTotalForBoard(t, ctx, pool, boardID); total != 2 {
		t.Fatalf("soft-removed rows should remain: total want 2 got %d", total)
	}
	if labels := countCardLabelsForBoard(t, ctx, pool, boardID); labels != 0 {
		t.Fatalf("card_labels cleared: want 0 got %d", labels)
	}

	// Re-assign previously removed member clears removed_at (active again).
	snap.Cards[0].IDMembers = []string{boardID + "-mem-1", boardID + "-mem-2"}
	srv3 := newFakeBoardServer(t, snap)
	defer srv3.Close()
	ent.Client = trelloClientForServer(srv3)
	if _, err := ent.ImportEntities(ctx, importer.ImportEntitiesOptions{BoardID: boardID}); err != nil {
		t.Fatalf("re-assign import: %v", err)
	}
	if active := countActiveCardMembersForBoard(t, ctx, pool, boardID); active != 2 {
		t.Fatalf("re-assign active card_members: want 2 got %d", active)
	}
	if total := countCardMembersTotalForBoard(t, ctx, pool, boardID); total != 2 {
		t.Fatalf("re-assign should not duplicate rows: total want 2 got %d", total)
	}
}

func TestImportBoard_WiresEntitiesThenActionsWithCursor(t *testing.T) {
	pool := openTestDB(t)
	const boardID = "boardFull1"
	isolateTestBoard(t, pool, boardID)
	store := postgres.NewStore(pool)

	snap := sampleBoardSnapshot(boardID)
	srv := newFakeBoardServer(t, snap)
	defer srv.Close()
	client := trelloClientForServer(srv)

	boardImp := &importer.BoardImporter{
		Entities: &importer.EntitiesImporter{Client: client, Store: store},
		Actions:  &importer.BoardActionsImporter{Client: client, Store: store},
	}
	ctx := context.Background()

	// Interrupt actions after 1 page (page size 2 → 2 of 4 actions).
	r1, err := boardImp.ImportBoard(ctx, importer.ImportBoardOptions{
		BoardID:               boardID,
		ActionsPageLimit:      2,
		StopAfterActionsPages: 1,
	})
	if err != nil {
		t.Fatalf("import board phase1: %v", err)
	}
	if r1.Entities.Cards != 2 || r1.Entities.Lists != 2 {
		t.Fatalf("entities incomplete: %+v", r1.Entities)
	}
	if !r1.Actions.StoppedEarly || r1.Actions.PagesCommitted != 1 {
		t.Fatalf("actions phase1: %+v", r1.Actions)
	}
	if r1.Actions.ActionsInserted != 2 {
		t.Fatalf("want 2 actions inserted, got %d", r1.Actions.ActionsInserted)
	}
	wantCursor := snap.Actions[1].ID
	if r1.Actions.CursorValue != wantCursor {
		t.Fatalf("cursor: want %s got %s", wantCursor, r1.Actions.CursorValue)
	}
	cur, found, err := store.GetSyncCursor(ctx, postgres.ResourceBoardActions)
	if err != nil || !found || cur != wantCursor {
		t.Fatalf("DB cursor: found=%v value=%q err=%v", found, cur, err)
	}

	// Resume via same entrypoint (entities re-upsert; actions continue from cursor).
	r2, err := boardImp.ImportBoard(ctx, importer.ImportBoardOptions{
		BoardID:          boardID,
		ActionsPageLimit: 2,
	})
	if err != nil {
		t.Fatalf("import board phase2: %v", err)
	}
	if r2.Actions.StoppedEarly {
		t.Fatalf("phase2 should not stop early: %+v", r2.Actions)
	}
	if r2.Actions.ActionsInserted != 2 {
		t.Fatalf("phase2 want 2 new actions, got %+v", r2.Actions)
	}

	if nActions := countActionsForBoard(t, ctx, pool, boardID); nActions != 4 {
		t.Fatalf("want 4 unique actions, got %d", nActions)
	}
	if nCards := countCardsForBoard(t, ctx, pool, boardID); nCards != 2 {
		t.Fatalf("want 2 cards after re-run entities, got %d", nCards)
	}

	// Full re-import: no duplicate entities or actions.
	r3, err := boardImp.ImportBoard(ctx, importer.ImportBoardOptions{
		BoardID:          boardID,
		ActionsPageLimit: 2,
	})
	if err != nil {
		t.Fatalf("re-import: %v", err)
	}
	if r3.Actions.ActionsInserted != 0 {
		t.Fatalf("re-import actions should insert 0, got %d", r3.Actions.ActionsInserted)
	}
	if nActions := countActionsForBoard(t, ctx, pool, boardID); nActions != 4 {
		t.Fatalf("after re-import want 4 actions, got %d", nActions)
	}
	if nBoards := countBoardsByTrelloID(t, ctx, pool, boardID); nBoards != 1 {
		t.Fatalf("after re-import want 1 board, got %d", nBoards)
	}
}
