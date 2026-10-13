package analytics

// Warning codes (Architect Decision).
const (
	WarnWorkflowIncomplete = "workflow_incomplete"
	WarnLeadRuleUnset      = "lead_rule_unset"
	WarnMissingCycleStart  = "missing_cycle_start"
	WarnMissingDoneStages  = "missing_done_stages"
	WarnMissingWorkStages         = "missing_work_stages"
	WarnMissingBugReopenFromStages = "missing_bug_reopen_from_stages"
	WarnIncompleteHistory         = "incomplete_history"
	WarnMissingStart              = "missing_start"
	WarnMissingDone               = "missing_done"
	WarnAmbiguousReopen           = "ambiguous_reopen" // legacy; new reopen rule has no ambiguous leave
	WarnDataStale                 = "data_stale"
)

// Warning is a non-fatal data/config quality signal. Never invent timestamps to avoid these.
type Warning struct {
	Code    string  `json:"code"`
	Message string  `json:"message"`
	Metric  string  `json:"metric,omitempty"`
	CardID  *string `json:"card_id,omitempty"`
}

// NeedsReviewItem flags a card that must not be silently included in a metric.
type NeedsReviewItem struct {
	CardID  string `json:"card_id"`
	Reason  string `json:"reason"`
	Metric  string `json:"metric,omitempty"`
	ActionID string `json:"action_id,omitempty"`
}
