package handlers

import (
	"testing"

	"trello_analytics_back/internal/analytics"
)

func TestBugsFixedConsistentWithCompletedCount(t *testing.T) {
	res := analytics.Result{
		BugReopen: analytics.BugReopenMetric{
			Available:           true,
			CompletedBugs:       2,
			CompletedBugCardIDs: []string{"bug-a", "bug-b"},
		},
	}
	if !bugsFixedConsistentWithCount(res) {
		t.Fatal("CompletedBugCardIDs must match CompletedBugs when count > 0")
	}

	empty := analytics.Result{
		BugReopen: analytics.BugReopenMetric{CompletedBugs: 0},
	}
	if !bugsFixedConsistentWithCount(empty) {
		t.Fatal("zero completed bugs should have empty id list")
	}

	inconsistent := analytics.Result{
		BugReopen: analytics.BugReopenMetric{
			CompletedBugs:       1,
			CompletedBugCardIDs: nil,
		},
	}
	if bugsFixedConsistentWithCount(inconsistent) {
		t.Fatal("count>0 with empty ids must be inconsistent")
	}
}

func TestCollectTrelloIDs_IncludesCompletedBugs(t *testing.T) {
	res := analytics.Result{
		BugReopen: analytics.BugReopenMetric{
			CompletedBugs:       1,
			CompletedBugCardIDs: []string{"bug-fixed"},
			ReopenedCardIDs:     []string{"bug-reopened"},
		},
		WIP: analytics.WIPMetric{
			CardIDs:    []string{"wip-1"},
			BugCardIDs: []string{"wip-bug"},
		},
	}
	ids := collectTrelloIDs(res)
	want := map[string]bool{"bug-fixed": true, "bug-reopened": true, "wip-1": true, "wip-bug": true}
	for _, id := range ids {
		delete(want, id)
	}
	if len(want) != 0 {
		t.Fatalf("missing ids: %v", want)
	}
}
