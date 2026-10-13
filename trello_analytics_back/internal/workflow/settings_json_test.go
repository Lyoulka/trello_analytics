package workflow

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestSettings_JSONSnakeCase(t *testing.T) {
	s := Settings{
		BoardID:                 "b1",
		LeadTimeStartRule:       LeadRuleCardCreated,
		DoneStages:              []string{"done"},
		BlockedStages:           []string{},
		WorkStages:              []string{"in_progress"},
		StaleThresholdSeconds:   100,
		DataStaleSyncAgeSeconds: 200,
		Timezone:                "UTC",
		ListStages:              map[string]string{"l1": "done"},
		BugLabelIDs:             []string{"lab1"},
	}
	b, err := json.Marshal(s)
	if err != nil {
		t.Fatal(err)
	}
	out := string(b)
	for _, want := range []string{
		`"board_id"`, `"lead_time_start_rule"`, `"done_stages"`, `"blocked_stages"`,
		`"work_stages"`, `"stale_threshold_seconds"`, `"data_stale_sync_age_seconds"`,
		`"list_stages"`, `"bug_label_ids"`,
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("missing %s in %s", want, out)
		}
	}
	if strings.Contains(out, `"LeadTimeStartRule"`) || strings.Contains(out, `"BugLabelIDs"`) {
		t.Fatalf("PascalCase leaked: %s", out)
	}
}
