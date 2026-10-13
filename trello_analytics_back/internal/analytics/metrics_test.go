package analytics_test

import (
	"testing"
	"time"

	"trello_analytics_back/internal/analytics"
	"trello_analytics_back/internal/workflow"
)

func baseSettings() workflow.Settings {
	return workflow.Settings{
		LeadTimeStartRule:       workflow.LeadRuleFirstEnterStage,
		LeadTimeStartStage:      "ready",
		CycleStartStage:         "in_progress",
		DoneStages:              []string{"done"},
		WorkStages:              []string{"in_progress", "review", "testing"},
		BugReopenFromStages:     []string{"qa_review"},
		BlockedStages:           []string{"blocked"},
		StaleThresholdSeconds:   86400, // 1d for tests
		DataStaleSyncAgeSeconds: 86400,
		Timezone:                "UTC",
	}
}

func period() analytics.Query {
	return analytics.Query{
		PeriodStart: mustTime("2024-01-01T00:00:00Z"),
		PeriodEnd:   mustTime("2024-02-01T00:00:00Z"),
	}
}

func TestCompleted_DoneToDoneNoDouble(t *testing.T) {
	// ready → done → done2 (also done stage) → should count once
	settings := baseSettings()
	settings.DoneStages = []string{"done", "done_archive"}
	trs := []analytics.StageTransition{
		move("c1", "ready", "done", "2024-01-10T10:00:00Z"),
		move("c1", "done", "done_archive", "2024-01-11T10:00:00Z"),
	}
	in := analytics.BoardInput{
		Settings: settings,
		CardByTrelloID: map[string]analytics.CardState{
			"c1": {TrelloCardID: "c1", CurrentStage: "done_archive"},
		},
		Transitions: trs,
		AsOf:        mustTime("2024-01-20T00:00:00Z"),
		LastSyncSuccessAt: timePtr(mustTime("2024-01-19T00:00:00Z")),
	}
	res := analytics.ComputeOn(in, period())
	if !res.Completed.Available || res.Completed.Count != 1 {
		t.Fatalf("completed=%+v", res.Completed)
	}
	if res.Throughput.Count != 1 {
		t.Fatalf("throughput=%+v", res.Throughput)
	}
}

func TestCompleted_ReopenStillOneThroughput(t *testing.T) {
	trs := []analytics.StageTransition{
		move("c1", "ready", "done", "2024-01-05T10:00:00Z"),
		move("c1", "done", "in_progress", "2024-01-06T10:00:00Z"),
		move("c1", "in_progress", "done", "2024-01-07T10:00:00Z"),
	}
	in := boardWithCards(trs, map[string]analytics.CardState{
		"c1": {TrelloCardID: "c1", CurrentStage: "done", IsBug: true},
	})
	res := analytics.ComputeOn(in, period())
	if res.Completed.Count != 1 {
		t.Fatalf("want throughput 1, got %d", res.Completed.Count)
	}
}

func TestLeadAndCycle_MissingStartExcluded(t *testing.T) {
	// Card enters done without ever entering ready / in_progress.
	trs := []analytics.StageTransition{
		move("c1", "backlog", "done", "2024-01-10T10:00:00Z"),
	}
	in := boardWithCards(trs, map[string]analytics.CardState{
		"c1": {TrelloCardID: "c1", CurrentStage: "done"},
	})
	res := analytics.ComputeOn(in, period())
	if res.LeadTime.Available && res.LeadTime.Count > 0 {
		t.Fatalf("lead should exclude missing start: %+v", res.LeadTime)
	}
	if res.CycleTime.Available && res.CycleTime.Count > 0 {
		t.Fatalf("cycle should exclude missing start: %+v", res.CycleTime)
	}
	if !hasWarn(res, analytics.WarnMissingStart) {
		t.Fatalf("expected missing_start warning, got %+v", res.Warnings)
	}
}

func TestLeadAndCycle_HappyPath(t *testing.T) {
	trs := []analytics.StageTransition{
		create("c1", "ready", "2024-01-01T08:00:00Z"),
		move("c1", "ready", "in_progress", "2024-01-02T08:00:00Z"),
		move("c1", "in_progress", "done", "2024-01-04T08:00:00Z"),
	}
	in := boardWithCards(trs, map[string]analytics.CardState{
		"c1": {TrelloCardID: "c1", CurrentStage: "done"},
	})
	res := analytics.ComputeOn(in, period())
	if !res.LeadTime.Available || res.LeadTime.Count != 1 {
		t.Fatalf("lead=%+v", res.LeadTime)
	}
	// Lead: ready enter (create) → done = 3 days
	wantLead := (3 * 24 * time.Hour).Seconds()
	if res.LeadTime.MeanSec != wantLead {
		t.Fatalf("lead mean want %v got %v", wantLead, res.LeadTime.MeanSec)
	}
	// Cycle: in_progress → done = 2 days
	wantCycle := (2 * 24 * time.Hour).Seconds()
	if res.CycleTime.MeanSec != wantCycle {
		t.Fatalf("cycle mean want %v got %v", wantCycle, res.CycleTime.MeanSec)
	}
}

func TestLeadAndCycle_UsesFirstDoneEnterInPeriod(t *testing.T) {
	// Two done-enters in one period: Lead/Cycle must use the first (same as Completed), not the last.
	trs := []analytics.StageTransition{
		create("c1", "ready", "2024-01-01T08:00:00Z"),
		move("c1", "ready", "in_progress", "2024-01-02T08:00:00Z"),
		move("c1", "in_progress", "done", "2024-01-04T08:00:00Z"), // first done-enter
		move("c1", "done", "in_progress", "2024-01-10T08:00:00Z"),
		move("c1", "in_progress", "done", "2024-01-20T08:00:00Z"), // second — must not stretch durations
	}
	in := boardWithCards(trs, map[string]analytics.CardState{
		"c1": {TrelloCardID: "c1", CurrentStage: "done"},
	})
	res := analytics.ComputeOn(in, period())
	if res.Completed.Count != 1 {
		t.Fatalf("completed want 1, got %+v", res.Completed)
	}
	wantLead := (3 * 24 * time.Hour).Seconds() // Jan1 → Jan4
	if !res.LeadTime.Available || res.LeadTime.MeanSec != wantLead {
		t.Fatalf("lead mean want %v got %+v", wantLead, res.LeadTime)
	}
	wantCycle := (2 * 24 * time.Hour).Seconds() // Jan2 → Jan4
	if !res.CycleTime.Available || res.CycleTime.MeanSec != wantCycle {
		t.Fatalf("cycle mean want %v got %+v", wantCycle, res.CycleTime)
	}
}

func TestLeadRuleUnset_Unavailable(t *testing.T) {
	s := baseSettings()
	s.LeadTimeStartRule = workflow.LeadRuleUnset
	trs := []analytics.StageTransition{
		move("c1", "ready", "done", "2024-01-10T10:00:00Z"),
	}
	in := boardWithCards(trs, map[string]analytics.CardState{
		"c1": {TrelloCardID: "c1", CurrentStage: "done"},
	})
	in.Settings = s
	res := analytics.ComputeOn(in, period())
	if res.LeadTime.Available {
		t.Fatal("lead must be unavailable when rule unset")
	}
	if !hasWarn(res, analytics.WarnLeadRuleUnset) {
		t.Fatalf("expected lead_rule_unset, got %+v", res.Warnings)
	}
}

func TestBugReopen_DoneToDoneNotInDenominator(t *testing.T) {
	s := baseSettings()
	s.DoneStages = []string{"done", "done2"}
	trs := []analytics.StageTransition{
		move("bug1", "in_progress", "done", "2024-01-05T10:00:00Z"),
		move("bug1", "done", "done2", "2024-01-06T10:00:00Z"),
	}
	in := boardWithCards(trs, map[string]analytics.CardState{
		"bug1": {TrelloCardID: "bug1", CurrentStage: "done2", IsBug: true},
	})
	in.Settings = s
	res := analytics.ComputeOn(in, period())
	if res.BugReopen.ReopenedBugs != 0 {
		t.Fatalf("want 0 reopen transitions: %+v", res.BugReopen)
	}
	if res.BugReopen.CompletedBugs != 1 {
		t.Fatalf("done→done must not add denominator: completed=%d", res.BugReopen.CompletedBugs)
	}
}

func TestBugReopen_QAReviewTransitions(t *testing.T) {
	// Denominator: 2 done-enters. Numerator: qa_review→testing and qa_review→in_progress (not →done).
	trs := []analytics.StageTransition{
		move("bug1", "testing", "qa_review", "2024-01-04T10:00:00Z"),
		move("bug1", "qa_review", "testing", "2024-01-05T10:00:00Z"), // reopen 1
		move("bug1", "testing", "qa_review", "2024-01-06T10:00:00Z"),
		move("bug1", "qa_review", "done", "2024-01-07T10:00:00Z"), // done-enter 1 (not reopen)
		move("bug1", "done", "qa_review", "2024-01-08T10:00:00Z"),
		move("bug1", "qa_review", "in_progress", "2024-01-09T10:00:00Z"), // reopen 2
		move("bug1", "in_progress", "done", "2024-01-11T10:00:00Z"),      // done-enter 2; fix 2d from reopen2
	}
	in := boardWithCards(trs, map[string]analytics.CardState{
		"bug1": {TrelloCardID: "bug1", CurrentStage: "done", IsBug: true},
	})
	res := analytics.ComputeOn(in, period())
	if res.BugReopen.CompletedBugs != 2 || res.BugReopen.ReopenedBugs != 2 {
		t.Fatalf("want 2/2 transitions, got %+v", res.BugReopen)
	}
	if res.BugReopen.ReopenPercent != 100 {
		t.Fatalf("reopen%% want 100, got %v", res.BugReopen.ReopenPercent)
	}
	if !res.FixTime.Available || res.FixTime.Count != 2 {
		t.Fatalf("fix time: both reopens have a subsequent done: %+v", res.FixTime)
	}
}

func TestBugReopen_CountsTransitionsNotUniqueCards(t *testing.T) {
	trs := []analytics.StageTransition{
		move("bug1", "qa_review", "testing", "2024-01-05T10:00:00Z"),
		move("bug1", "testing", "qa_review", "2024-01-06T10:00:00Z"),
		move("bug1", "qa_review", "testing", "2024-01-07T10:00:00Z"),
		move("bug1", "testing", "done", "2024-01-08T10:00:00Z"),
		move("bug1", "done", "qa_review", "2024-01-09T10:00:00Z"),
		move("bug1", "qa_review", "done", "2024-01-10T10:00:00Z"),
	}
	in := boardWithCards(trs, map[string]analytics.CardState{
		"bug1": {TrelloCardID: "bug1", CurrentStage: "done", IsBug: true},
	})
	res := analytics.ComputeOn(in, period())
	// 2 reopen transitions, 2 done-enters, 1 unique card
	if res.BugReopen.ReopenedBugs != 2 || res.BugReopen.CompletedBugs != 2 {
		t.Fatalf("transition counts: %+v", res.BugReopen)
	}
	if len(res.BugReopen.CompletedBugCardIDs) != 1 || len(res.BugReopen.ReopenedCardIDs) != 1 {
		t.Fatalf("unique card ids: completed=%v reopened=%v", res.BugReopen.CompletedBugCardIDs, res.BugReopen.ReopenedCardIDs)
	}
}

func TestBugReopen_OutsidePeriodIgnored(t *testing.T) {
	trs := []analytics.StageTransition{
		move("bug-out", "qa_review", "testing", "2023-12-12T10:00:00Z"),
		move("bug-out", "testing", "done", "2023-12-17T10:00:00Z"),
		move("bug-in", "qa_review", "testing", "2024-01-06T10:00:00Z"),
		move("bug-in", "testing", "done", "2024-01-08T10:00:00Z"),
	}
	in := boardWithCards(trs, map[string]analytics.CardState{
		"bug-in":  {TrelloCardID: "bug-in", CurrentStage: "done", IsBug: true},
		"bug-out": {TrelloCardID: "bug-out", CurrentStage: "done", IsBug: true},
	})
	res := analytics.ComputeOn(in, period())
	if res.BugReopen.CompletedBugs != 1 || res.BugReopen.ReopenedBugs != 1 {
		t.Fatalf("period-scoped transitions: %+v", res.BugReopen)
	}
	if res.BugReopen.ReopenPercent != 100 {
		t.Fatalf("reopen%% want 100, got %v", res.BugReopen.ReopenPercent)
	}
}

func TestBugReopen_QAReviewToDoneIsNotReopen(t *testing.T) {
	trs := []analytics.StageTransition{
		move("bug1", "qa_review", "done", "2024-01-05T10:00:00Z"),
	}
	in := boardWithCards(trs, map[string]analytics.CardState{
		"bug1": {TrelloCardID: "bug1", CurrentStage: "done", IsBug: true},
	})
	res := analytics.ComputeOn(in, period())
	if res.BugReopen.ReopenedBugs != 0 || res.BugReopen.CompletedBugs != 1 {
		t.Fatalf("qa_review→done: %+v", res.BugReopen)
	}
}

func TestWIP_Stale_Workload(t *testing.T) {
	asOf := mustTime("2024-01-20T00:00:00Z")
	in := analytics.BoardInput{
		Settings: baseSettings(),
		CardByTrelloID: map[string]analytics.CardState{
			"w1": {
				TrelloCardID: "w1", CurrentStage: "in_progress",
				LastActivityAt: timePtr(mustTime("2024-01-10T00:00:00Z")), // 10d > 1d threshold
				AssigneeMemberIDs: []string{"m1", "m2"},
			},
			"w2": {
				TrelloCardID: "w2", CurrentStage: "blocked",
				LastActivityAt: timePtr(mustTime("2024-01-19T12:00:00Z")),
				AssigneeMemberIDs: []string{"m1"},
			},
			"done1": {
				TrelloCardID: "done1", CurrentStage: "done",
				LastActivityAt: timePtr(mustTime("2024-01-01T00:00:00Z")),
				AssigneeMemberIDs: []string{"m2"},
			},
		},
		MemberDisplayNames: map[string]string{"m1": "Ada", "m2": "Grace"},
		Transitions: []analytics.StageTransition{
			move("done1", "in_progress", "done", "2024-01-05T00:00:00Z"),
		},
		AsOf:              asOf,
		LastSyncSuccessAt: timePtr(mustTime("2024-01-19T00:00:00Z")),
	}
	res := analytics.ComputeOn(in, period())
	if res.WIP.Count != 2 || res.WIP.Blocked != 1 {
		t.Fatalf("wip=%+v", res.WIP)
	}
	if res.Stale.Count != 1 || res.Stale.CardIDs[0] != "w1" {
		t.Fatalf("stale=%+v", res.Stale)
	}
	// Multi-assignee: w1 counted fully for m1 and m2.
	byM := map[string]analytics.WorkloadEntry{}
	for _, w := range res.Workload {
		byM[w.MemberID] = w
	}
	if byM["m1"].ActiveCards != 2 {
		t.Fatalf("m1 active=%d", byM["m1"].ActiveCards)
	}
	if byM["m2"].ActiveCards != 1 {
		t.Fatalf("m2 active=%d", byM["m2"].ActiveCards)
	}
	if byM["m2"].CompletedCards != 1 {
		t.Fatalf("m2 completed=%d (done1 assigned)", byM["m2"].CompletedCards)
	}
	if byM["m1"].FullName != "Ada" || byM["m2"].FullName != "Grace" {
		t.Fatalf("workload names: m1=%q m2=%q", byM["m1"].FullName, byM["m2"].FullName)
	}
}

func TestIncompleteWorkflow_MetricsUnavailable(t *testing.T) {
	in := analytics.BoardInput{
		Settings: workflow.Settings{
			LeadTimeStartRule: workflow.LeadRuleUnset,
			// no done/work/cycle
			StaleThresholdSeconds:   604800,
			DataStaleSyncAgeSeconds: 86400,
		},
		CardByTrelloID: map[string]analytics.CardState{
			"w1": {
				TrelloCardID: "w1", CurrentStage: "in_progress",
				LastActivityAt: timePtr(mustTime("2024-01-01T00:00:00Z")),
			},
		},
		AsOf:              mustTime("2024-01-20T00:00:00Z"),
		LastSyncSuccessAt: timePtr(mustTime("2024-01-19T00:00:00Z")),
	}
	res := analytics.ComputeOn(in, period())
	if res.Completed.Available {
		t.Fatal("completed should be unavailable without done_stages")
	}
	if res.WIP.Available {
		t.Fatal("WIP must be unavailable without done_stages")
	}
	if res.Stale.Available {
		t.Fatal("Stale must be unavailable without done_stages")
	}
	if !hasWarn(res, analytics.WarnMissingDoneStages) || !hasWarn(res, analytics.WarnWorkflowIncomplete) {
		t.Fatalf("warnings=%+v", res.Warnings)
	}
}

func TestArchiveExcludedFromWIP(t *testing.T) {
	in := analytics.BoardInput{
		Settings: baseSettings(),
		CardByTrelloID: map[string]analytics.CardState{
			"a1": {
				TrelloCardID: "a1", CurrentStage: "in_progress", IsArchived: true,
				LastActivityAt: timePtr(mustTime("2024-01-01T00:00:00Z")),
			},
		},
		AsOf:              mustTime("2024-01-20T00:00:00Z"),
		LastSyncSuccessAt: timePtr(mustTime("2024-01-19T00:00:00Z")),
	}
	res := analytics.ComputeOn(in, period())
	if res.WIP.Count != 0 {
		t.Fatalf("archived must not be WIP: %+v", res.WIP)
	}
}

func boardWithCards(trs []analytics.StageTransition, cards map[string]analytics.CardState) analytics.BoardInput {
	return analytics.BoardInput{
		Settings:          baseSettings(),
		CardByTrelloID:    cards,
		Transitions:       trs,
		AsOf:              mustTime("2024-01-20T00:00:00Z"),
		LastSyncSuccessAt: timePtr(mustTime("2024-01-19T00:00:00Z")),
	}
}

func move(card, from, to, at string) analytics.StageTransition {
	return analytics.StageTransition{
		TrelloActionID: card + at + from + to,
		TrelloCardID:   card,
		ActionType:     "updateCard",
		At:             mustTime(at),
		FromStage:      from,
		ToStage:        to,
		IsListMove:     true,
	}
}

func create(card, to, at string) analytics.StageTransition {
	return analytics.StageTransition{
		TrelloActionID: "create-" + card,
		TrelloCardID:   card,
		ActionType:     "createCard",
		At:             mustTime(at),
		ToStage:        to,
		IsCreate:       true,
	}
}

func timePtr(t time.Time) *time.Time { return &t }

func hasWarn(res analytics.Result, code string) bool {
	for _, w := range res.Warnings {
		if w.Code == code {
			return true
		}
	}
	return false
}
