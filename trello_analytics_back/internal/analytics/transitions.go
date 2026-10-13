package analytics

import (
	"encoding/json"
	"time"
)

// StageTransition is a resolved list-move (or create) event on a card timeline.
type StageTransition struct {
	TrelloActionID   string
	TrelloCardID     string
	ActionType       string
	At               time.Time
	FromTrelloListID string // empty for create without known list
	ToTrelloListID   string
	FromStage        string // may be empty if list unmapped
	ToStage          string
	IsCreate         bool
	IsListMove       bool
}

// RawAction is the minimal action shape for transition extraction (tests + engine).
type RawAction struct {
	TrelloActionID string
	ActionType     string
	ActionDate     time.Time
	RawPayload     json.RawMessage
}

// ListResolver maps trello list id → logical stage (empty if unmapped).
type ListResolver map[string]string

// ExtractTransitions builds chronologically ordered stage transitions from raw actions.
// Only updateCard list-moves and createCard are used for stage timelines.
func ExtractTransitions(actions []RawAction, stagesByTrelloList ListResolver) []StageTransition {
	var out []StageTransition
	for _, a := range actions {
		switch a.ActionType {
		case "createCard":
			tr, ok := parseCreateCard(a, stagesByTrelloList)
			if ok {
				out = append(out, tr)
			}
		case "updateCard":
			tr, ok := parseListMove(a, stagesByTrelloList)
			if ok {
				out = append(out, tr)
			}
		}
	}
	return out
}

func parseCreateCard(a RawAction, stages ListResolver) (StageTransition, bool) {
	cardID, listID := extractCreateLists(a.RawPayload)
	if cardID == "" {
		return StageTransition{}, false
	}
	tr := StageTransition{
		TrelloActionID: a.TrelloActionID,
		TrelloCardID:   cardID,
		ActionType:     a.ActionType,
		At:             a.ActionDate.UTC(),
		ToTrelloListID: listID,
		IsCreate:       true,
	}
	if listID != "" {
		tr.ToStage = stages[listID]
	}
	return tr, true
}

func parseListMove(a RawAction, stages ListResolver) (StageTransition, bool) {
	cardID, before, after, ok := extractListMoveIDs(a.RawPayload)
	if !ok || cardID == "" || after == "" {
		return StageTransition{}, false
	}
	// Same-list update is not a stage move.
	if before != "" && before == after {
		return StageTransition{}, false
	}
	tr := StageTransition{
		TrelloActionID:   a.TrelloActionID,
		TrelloCardID:     cardID,
		ActionType:       a.ActionType,
		At:               a.ActionDate.UTC(),
		FromTrelloListID: before,
		ToTrelloListID:   after,
		FromStage:        stages[before],
		ToStage:          stages[after],
		IsListMove:       true,
	}
	return tr, true
}

func extractCreateLists(payload json.RawMessage) (cardID, listID string) {
	if len(payload) == 0 {
		return "", ""
	}
	var root map[string]any
	if err := json.Unmarshal(payload, &root); err != nil {
		return "", ""
	}
	data, _ := root["data"].(map[string]any)
	if data == nil {
		return "", ""
	}
	cardID = nestedID(data, "card")
	listID = nestedID(data, "list")
	return cardID, listID
}

func extractListMoveIDs(payload json.RawMessage) (cardID, before, after string, ok bool) {
	if len(payload) == 0 {
		return "", "", "", false
	}
	var root map[string]any
	if err := json.Unmarshal(payload, &root); err != nil {
		return "", "", "", false
	}
	data, _ := root["data"].(map[string]any)
	if data == nil {
		return "", "", "", false
	}
	cardID = nestedID(data, "card")
	before = nestedID(data, "listBefore")
	after = nestedID(data, "listAfter")
	if after == "" {
		// Fallback: old.idList → list.id / card.idList
		if old, _ := data["old"].(map[string]any); old != nil {
			if v, ok := old["idList"].(string); ok {
				before = v
			}
		}
		if after == "" {
			after = nestedID(data, "list")
		}
		if after == "" {
			if card, _ := data["card"].(map[string]any); card != nil {
				if v, ok := card["idList"].(string); ok {
					after = v
				}
			}
		}
	}
	if after == "" {
		return cardID, before, after, false
	}
	// Require evidence of a list change: listBefore/listAfter or old.idList.
	if before == "" && nestedID(data, "listBefore") == "" {
		if old, _ := data["old"].(map[string]any); old == nil || old["idList"] == nil {
			return cardID, before, after, false
		}
	}
	return cardID, before, after, true
}

func nestedID(m map[string]any, key string) string {
	v, _ := m[key].(map[string]any)
	if v == nil {
		return ""
	}
	id, _ := v["id"].(string)
	return id
}

// GroupByCard returns transitions keyed by trello card id, preserving order.
func GroupByCard(trs []StageTransition) map[string][]StageTransition {
	out := make(map[string][]StageTransition)
	for _, tr := range trs {
		out[tr.TrelloCardID] = append(out[tr.TrelloCardID], tr)
	}
	return out
}
