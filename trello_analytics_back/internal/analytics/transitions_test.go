package analytics_test

import (
	"encoding/json"
	"testing"
	"time"

	"trello_analytics_back/internal/analytics"
)

func TestExtractTransitions_ListMoveAndCreate(t *testing.T) {
	stages := analytics.ListResolver{
		"listReady": "ready",
		"listProg":  "in_progress",
		"listDone":  "done",
	}
	actions := []analytics.RawAction{
		rawCreate("a0", "card1", "listReady", "2024-01-01T10:00:00Z"),
		rawMove("a1", "card1", "listReady", "listProg", "2024-01-02T10:00:00Z"),
		rawMove("a2", "card1", "listProg", "listDone", "2024-01-03T10:00:00Z"),
		// Non-move updateCard (name change) — ignored.
		{
			TrelloActionID: "a3",
			ActionType:     "updateCard",
			ActionDate:     mustTime("2024-01-03T11:00:00Z"),
			RawPayload: mustJSON(map[string]any{
				"id": "a3", "type": "updateCard",
				"data": map[string]any{
					"card": map[string]any{"id": "card1", "name": "x"},
					"old":  map[string]any{"name": "y"},
				},
			}),
		},
	}
	trs := analytics.ExtractTransitions(actions, stages)
	if len(trs) != 3 {
		t.Fatalf("want 3 transitions, got %d", len(trs))
	}
	if !trs[0].IsCreate || trs[0].ToStage != "ready" {
		t.Fatalf("create: %+v", trs[0])
	}
	if !trs[1].IsListMove || trs[1].FromStage != "ready" || trs[1].ToStage != "in_progress" {
		t.Fatalf("move1: %+v", trs[1])
	}
	if trs[2].ToStage != "done" {
		t.Fatalf("move2: %+v", trs[2])
	}
}

func TestExtractTransitions_OldIdListFallback(t *testing.T) {
	stages := analytics.ListResolver{"L1": "a", "L2": "b"}
	payload := mustJSON(map[string]any{
		"id": "x", "type": "updateCard",
		"data": map[string]any{
			"card": map[string]any{"id": "c1", "idList": "L2"},
			"list": map[string]any{"id": "L2"},
			"old":  map[string]any{"idList": "L1"},
		},
	})
	trs := analytics.ExtractTransitions([]analytics.RawAction{{
		TrelloActionID: "x", ActionType: "updateCard",
		ActionDate: mustTime("2024-01-01T00:00:00Z"), RawPayload: payload,
	}}, stages)
	if len(trs) != 1 || trs[0].FromTrelloListID != "L1" || trs[0].ToTrelloListID != "L2" {
		t.Fatalf("got %+v", trs)
	}
}

func rawCreate(id, card, list, at string) analytics.RawAction {
	return analytics.RawAction{
		TrelloActionID: id,
		ActionType:     "createCard",
		ActionDate:     mustTime(at),
		RawPayload: mustJSON(map[string]any{
			"id": id, "type": "createCard",
			"data": map[string]any{
				"card": map[string]any{"id": card},
				"list": map[string]any{"id": list},
			},
		}),
	}
}

func rawMove(id, card, before, after, at string) analytics.RawAction {
	return analytics.RawAction{
		TrelloActionID: id,
		ActionType:     "updateCard",
		ActionDate:     mustTime(at),
		RawPayload: mustJSON(map[string]any{
			"id": id, "type": "updateCard",
			"data": map[string]any{
				"card":       map[string]any{"id": card},
				"listBefore": map[string]any{"id": before},
				"listAfter":  map[string]any{"id": after},
			},
		}),
	}
}

func mustTime(s string) time.Time {
	t, err := time.Parse(time.RFC3339, s)
	if err != nil {
		panic(err)
	}
	return t.UTC()
}

func mustJSON(v any) json.RawMessage {
	b, err := json.Marshal(v)
	if err != nil {
		panic(err)
	}
	return b
}
