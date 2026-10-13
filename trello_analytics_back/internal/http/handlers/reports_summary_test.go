package handlers



import (

	"encoding/json"

	"testing"



	"trello_analytics_back/internal/analytics"

)



func TestProjectReportSummary_UnavailableIsJSONNull(t *testing.T) {

	// Mirrors Engine incomplete workflow: Count=0 + Available=false must not become summary 0.

	res := analytics.Result{

		Completed: analytics.CountMetric{Available: false, Count: 0},

		BugReopen: analytics.BugReopenMetric{Available: false, CompletedBugs: 0, ReopenedBugs: 0},

		WIP:       analytics.WIPMetric{Available: false, Count: 0},

	}

	sum := projectReportSummary(res)

	raw, err := json.Marshal(sum)

	if err != nil {

		t.Fatal(err)

	}

	var got map[string]any

	if err := json.Unmarshal(raw, &got); err != nil {

		t.Fatal(err)

	}

	for _, key := range []string{"completed_count", "bugs_fixed_count", "reopened_count", "wip_count"} {

		if got[key] != nil {

			t.Fatalf("%s: want JSON null, got %#v (raw %s)", key, got[key], raw)

		}

	}

}



func TestProjectReportSummary_AvailableKeepsZero(t *testing.T) {

	res := analytics.Result{

		Completed: analytics.CountMetric{Available: true, Count: 0},

		BugReopen: analytics.BugReopenMetric{Available: true, CompletedBugs: 0, ReopenedBugs: 0},

		WIP:       analytics.WIPMetric{Available: true, Count: 0},

	}

	sum := projectReportSummary(res)

	raw, err := json.Marshal(sum)

	if err != nil {

		t.Fatal(err)

	}

	var got map[string]any

	if err := json.Unmarshal(raw, &got); err != nil {

		t.Fatal(err)

	}

	for _, key := range []string{"completed_count", "bugs_fixed_count", "reopened_count", "wip_count"} {

		n, ok := got[key].(float64)

		if !ok || n != 0 {

			t.Fatalf("%s: want numeric 0 when Available, got %#v (raw %s)", key, got[key], raw)

		}

	}

}



func TestProjectReportDeltas_NullWhenEitherSideUnavailable(t *testing.T) {

	curr := analytics.Result{

		Completed: analytics.CountMetric{Available: false, Count: 0},

		BugReopen: analytics.BugReopenMetric{Available: true, ReopenedBugs: 2},

		WIP:       analytics.WIPMetric{Available: true, Count: 5},

	}

	prev := analytics.Result{

		Completed: analytics.CountMetric{Available: true, Count: 3},

		BugReopen: analytics.BugReopenMetric{Available: false, ReopenedBugs: 0},

		WIP:       analytics.WIPMetric{Available: true, Count: 4},

	}

	d := projectReportDeltas(curr, prev)

	raw, err := json.Marshal(d)

	if err != nil {

		t.Fatal(err)

	}

	var got map[string]any

	if err := json.Unmarshal(raw, &got); err != nil {

		t.Fatal(err)

	}

	if got["completed_delta"] != nil {

		t.Fatalf("completed_delta: want null when curr unavailable, got %#v", got["completed_delta"])

	}

	if got["reopened_delta"] != nil {

		t.Fatalf("reopened_delta: want null when prev unavailable, got %#v", got["reopened_delta"])

	}

	if _, hasWIP := got["wip_delta"]; hasWIP {

		t.Fatalf("wip_delta must be omitted from deltas, got %#v (raw %s)", got["wip_delta"], raw)

	}

}


