package postgres

import (
	"context"
	"encoding/json"
	"os"
	"testing"
	"time"
)

func TestCardLastUpdatedBy_UsesNewestActionNotOldMove(t *testing.T) {
	url := os.Getenv("TEST_DATABASE_URL")
	if url == "" {
		url = os.Getenv("DATABASE_URL")
	}
	if url == "" {
		url = "postgres://trello:trello_dev_password@localhost:5432/trello_analytics?sslmode=disable"
	}
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	pool, err := Connect(ctx, url)
	if err != nil {
		t.Skipf("postgres unavailable: %v", err)
	}
	t.Cleanup(pool.Close)
	store := NewStore(pool)

	suffix := time.Now().UTC().Format("20060102150405.000000000")
	cardTID := "test-lub-card-" + suffix
	moverTID := "test-lub-mover-" + suffix
	joinerTID := "test-lub-joiner-" + suffix
	moveAID := "test-lub-act-move-" + suffix
	joinAID := "test-lub-act-join-" + suffix

	memberIDs, err := store.UpsertMembers(ctx, []MemberUpsert{
		{TrelloMemberID: moverTID, FullName: "Old Mover", Username: "mover", IsActive: true},
		{TrelloMemberID: joinerTID, FullName: "New Joiner", Username: "joiner", IsActive: true},
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), `DELETE FROM members WHERE trello_member_id = ANY($1::text[])`,
			[]string{moverTID, joinerTID})
	})

	oldMove := mustRawJSON(t, map[string]any{
		"id": moveAID, "type": "updateCard", "idMemberCreator": moverTID,
		"data": map[string]any{
			"card":      map[string]any{"id": cardTID},
			"listAfter": map[string]any{"id": "list-x", "name": "Doing"},
		},
	})
	newJoin := mustRawJSON(t, map[string]any{
		"id": joinAID, "type": "addMemberToCard", "idMemberCreator": joinerTID,
		"data": map[string]any{
			"card": map[string]any{"id": cardTID},
		},
	})
	base := time.Date(2026, 10, 14, 5, 0, 0, 0, time.UTC)
	if _, err := store.InsertCardActionsPage(ctx, []CardActionInsert{
		{TrelloActionID: moveAID, ActionType: "updateCard", ActionDate: base, RawPayload: oldMove},
		{TrelloActionID: joinAID, ActionType: "addMemberToCard", ActionDate: base.Add(30 * time.Second), RawPayload: newJoin},
	}); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), `DELETE FROM card_actions WHERE trello_action_id = ANY($1::text[])`,
			[]string{moveAID, joinAID})
	})

	got, err := store.cardLastUpdatedBy(ctx, []string{cardTID})
	if err != nil {
		t.Fatal(err)
	}
	u, ok := got[cardTID]
	if !ok {
		t.Fatal("missing card in lastUpdatedBy")
	}
	if u.Name != "New Joiner" {
		t.Fatalf("want newest action creator New Joiner, got %q (member=%s)", u.Name, u.MemberID)
	}
	if u.MemberID != memberIDs[joinerTID] {
		t.Fatalf("member id: want %s got %s", memberIDs[joinerTID], u.MemberID)
	}
}

func mustRawJSON(t *testing.T, v any) json.RawMessage {
	t.Helper()
	b, err := json.Marshal(v)
	if err != nil {
		t.Fatal(err)
	}
	return b
}
