package analytics_test

import (
	"testing"

	"trello_analytics_back/internal/analytics"
)

func TestFilterActiveAssignees(t *testing.T) {
	res := analytics.FilterActiveAssignees([]analytics.AssigneeActivity{
		{CardID: "c-mix", MemberID: "m-active", IsActive: true},
		{CardID: "c-mix", MemberID: "m-off", IsActive: false},
		{CardID: "c-only-off", MemberID: "m-off2", IsActive: false},
		{CardID: "c-only-off", MemberID: "m-off3", IsActive: false},
	})
	if len(res.Assignees["c-mix"]) != 1 || res.Assignees["c-mix"][0] != "m-active" {
		t.Fatalf("mix assignees=%v", res.Assignees["c-mix"])
	}
	if _, ok := res.ExcludeCardIDs["c-only-off"]; !ok {
		t.Fatal("expected c-only-off excluded")
	}
	if _, ok := res.ExcludeCardIDs["c-mix"]; ok {
		t.Fatal("c-mix must not be excluded")
	}
	if _, ok := res.Assignees["c-only-off"]; ok {
		t.Fatal("inactive-only card must have no active assignees")
	}
}

func TestFilterActiveAssignees_UnassignedNotExcluded(t *testing.T) {
	res := analytics.FilterActiveAssignees(nil)
	if len(res.ExcludeCardIDs) != 0 {
		t.Fatalf("exclude=%v", res.ExcludeCardIDs)
	}
}

// Mirrors engine.loadFromStore skip: inactive-only card IDs must not enter CardByTrelloID.
func TestFilterActiveAssignees_ExcludeIDsMatchEngineSkipSet(t *testing.T) {
	res := analytics.FilterActiveAssignees([]analytics.AssigneeActivity{
		{CardID: "keep-unassigned-style", MemberID: "m1", IsActive: true},
		{CardID: "drop-me", MemberID: "m2", IsActive: false},
	})
	cards := map[string]string{"keep-unassigned-style": "t-keep", "drop-me": "t-drop", "no-assignees": "t-free"}
	kept := map[string]bool{}
	for cardID, trelloID := range cards {
		if _, skip := res.ExcludeCardIDs[cardID]; skip {
			continue
		}
		kept[trelloID] = true
	}
	if !kept["t-keep"] || !kept["t-free"] {
		t.Fatalf("kept=%v", kept)
	}
	if kept["t-drop"] {
		t.Fatal("inactive-only card must be skipped like Engine")
	}
}
