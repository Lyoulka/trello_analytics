package postgres

import (
	"context"
	"os"
	"testing"
	"time"
)

func TestUpsertMembers_PreservesIsActiveOnConflict(t *testing.T) {
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

	store := NewStore(pool)
	tid := "test-member-active-" + time.Now().UTC().Format("20060102150405.000000000")

	ids, err := store.UpsertMembers(ctx, []MemberUpsert{{
		TrelloMemberID: tid, FullName: "One", Username: "one", IsActive: true,
	}})
	if err != nil {
		t.Fatal(err)
	}
	id := ids[tid]
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), `DELETE FROM members WHERE id = $1::uuid`, id)
	})

	if _, err := store.SetMemberActive(ctx, id, false); err != nil {
		t.Fatal(err)
	}

	if _, err := store.UpsertMembers(ctx, []MemberUpsert{{
		TrelloMemberID: tid, FullName: "One Updated", Username: "one", IsActive: true,
	}}); err != nil {
		t.Fatal(err)
	}

	m, err := store.GetMember(ctx, id)
	if err != nil {
		t.Fatal(err)
	}
	if m.IsActive {
		t.Fatal("is_active must stay false after sync upsert")
	}
	if m.FullName != "One Updated" {
		t.Fatalf("full_name=%q", m.FullName)
	}
}
