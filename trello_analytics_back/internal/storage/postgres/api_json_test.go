package postgres

import (
	"encoding/json"
	"strings"
	"testing"
	"time"
)

func TestCardListItem_JSONSnakeCase(t *testing.T) {
	now := time.Date(2024, 1, 2, 3, 4, 5, 0, time.UTC)
	listID := "list-1"
	listName := "Doing"
	item := CardListItem{
		ID:                 "card-uuid",
		TrelloCardID:       "trello-1",
		Name:               "Task",
		CurrentListID:      &listID,
		CurrentListName:    &listName,
		IsArchived:         false,
		CreatedAtTrello:    &now,
		LastTrelloUpdateAt: &now,
		MemberIDs:          []string{"m1"},
		LabelIDs:           []string{"l1"},
	}
	b, err := json.Marshal(item)
	if err != nil {
		t.Fatal(err)
	}
	s := string(b)
	for _, want := range []string{
		`"id"`, `"trello_card_id"`, `"current_list_id"`, `"current_list_name"`,
		`"is_archived"`, `"created_at_trello"`, `"last_trello_update_at"`,
		`"last_updated_by_member_id"`, `"last_updated_by_name"`,
		`"member_ids"`, `"label_ids"`,
	} {
		if !strings.Contains(s, want) {
			t.Fatalf("missing %s in %s", want, s)
		}
	}
	for _, bad := range []string{`"TrelloCardID"`, `"CurrentListID"`, `"MemberIDs"`} {
		if strings.Contains(s, bad) {
			t.Fatalf("PascalCase key leaked: %s in %s", bad, s)
		}
	}
}

func TestCardHistoryItem_JSONSnakeCase(t *testing.T) {
	item := CardHistoryItem{
		ID:             "a1",
		TrelloActionID: "tact",
		ActionType:     "updateCard",
		ActionDate:     time.Date(2024, 1, 2, 3, 4, 5, 0, time.UTC),
	}
	b, err := json.Marshal(item)
	if err != nil {
		t.Fatal(err)
	}
	s := string(b)
	for _, want := range []string{`"trello_action_id"`, `"action_type"`, `"action_date"`, `"source_list_id"`, `"destination_list_id"`} {
		if !strings.Contains(s, want) {
			t.Fatalf("missing %s in %s", want, s)
		}
	}
	if strings.Contains(s, `"TrelloActionID"`) {
		t.Fatalf("PascalCase leaked: %s", s)
	}
}

func TestSyncRun_JSONSnakeCase(t *testing.T) {
	run := SyncRun{
		ID:             "run-1",
		StartedAt:      time.Date(2024, 1, 2, 3, 4, 5, 0, time.UTC),
		Status:         SyncRunSuccess,
		ProcessedCount: 1,
		CreatedCount:   2,
		UpdatedCount:   3,
		ErrorCount:     0,
	}
	b, err := json.Marshal(run)
	if err != nil {
		t.Fatal(err)
	}
	s := string(b)
	for _, want := range []string{
		`"started_at"`, `"finished_at"`, `"processed_count"`, `"created_count"`,
		`"updated_count"`, `"error_count"`, `"error_summary"`,
	} {
		if !strings.Contains(s, want) {
			t.Fatalf("missing %s in %s", want, s)
		}
	}
	if strings.Contains(s, `"ProcessedCount"`) {
		t.Fatalf("PascalCase leaked: %s", s)
	}
}
