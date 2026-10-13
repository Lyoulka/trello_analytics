package postgres

import (
	"context"
	"os"
	"testing"
	"time"
)

func TestUpsertLists_PreservesIsActiveOnConflict(t *testing.T) {
	url := os.Getenv("TEST_DATABASE_URL")
	if url == "" {
		url = os.Getenv("DATABASE_URL")
	}
	if url == "" {
		url = "postgres://trello:trello_dev_password@localhost:5432/trello_analytics?sslmode=disable"
	}
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	pool, err := Connect(ctx, url)
	if err != nil {
		t.Skipf("postgres unavailable: %v", err)
	}
	t.Cleanup(pool.Close)

	// Need a board for FK.
	var boardID string
	tidBoard := "test-list-active-board-" + time.Now().UTC().Format("20060102150405.000000000")
	err = pool.QueryRow(ctx, `
		INSERT INTO boards (trello_board_id, name) VALUES ($1, 'list-active-test')
		RETURNING id::text
	`, tidBoard).Scan(&boardID)
	if err != nil {
		t.Skipf("boards insert failed (migrations?): %v", err)
	}
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), `DELETE FROM trello_lists WHERE board_id = $1::uuid`, boardID)
		_, _ = pool.Exec(context.Background(), `DELETE FROM boards WHERE id = $1::uuid`, boardID)
	})

	store := NewStore(pool)
	tid := "test-list-active-" + time.Now().UTC().Format("20060102150405.000000000")
	ids, err := store.UpsertLists(ctx, boardID, []ListUpsert{{
		TrelloListID: tid, Name: "Parking", IsArchived: false,
	}})
	if err != nil {
		t.Fatal(err)
	}
	id := ids[tid]

	if _, err := store.SetListActive(ctx, id, false); err != nil {
		t.Fatal(err)
	}

	if _, err := store.UpsertLists(ctx, boardID, []ListUpsert{{
		TrelloListID: tid, Name: "Parking Renamed", IsArchived: false,
	}}); err != nil {
		t.Fatal(err)
	}

	lists, err := store.ListBoardLists(ctx)
	if err != nil {
		t.Fatal(err)
	}
	var found *BoardList
	for i := range lists {
		if lists[i].ID == id {
			found = &lists[i]
			break
		}
	}
	if found == nil {
		t.Fatal("list not found")
	}
	if found.IsActive {
		t.Fatal("is_active must stay false after sync upsert")
	}
	if found.Name != "Parking Renamed" {
		t.Fatalf("name=%q", found.Name)
	}
}
