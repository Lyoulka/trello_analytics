package workflow

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"trello_analytics_back/internal/storage/postgres"
)

// Lead Time start rules (OQ-003). Default unset → Lead Time unavailable with warning.
const (
	LeadRuleUnset           = "unset"
	LeadRuleCardCreated     = "card_created"
	LeadRuleFirstEnterStage = "first_enter_stage"
)

// Default thresholds (OQ-008 provisional).
const (
	DefaultStaleThresholdSeconds   = 604800 // 7d
	DefaultDataStaleSyncAgeSeconds = 86400  // 24h
	DefaultTimezone                = "UTC"
)

// Settings is the board workflow configuration used by analytics.
type Settings struct {
	BoardID                 string            `json:"board_id"`
	LeadTimeStartRule       string            `json:"lead_time_start_rule"`
	LeadTimeStartStage      string            `json:"lead_time_start_stage,omitempty"` // required when rule = first_enter_stage
	CycleStartStage         string            `json:"cycle_start_stage,omitempty"`
	DoneStages              []string          `json:"done_stages"`
	BlockedStages           []string          `json:"blocked_stages"`
	WorkStages              []string          `json:"work_stages"`
	// BugReopenFromStages: leave → non-done counts as bug reopen (transition count).
	BugReopenFromStages     []string          `json:"bug_reopen_from_stages"`
	StaleThresholdSeconds   int               `json:"stale_threshold_seconds"`
	DataStaleSyncAgeSeconds int               `json:"data_stale_sync_age_seconds"`
	Timezone                string            `json:"timezone"`
	// ListStages maps internal list UUID → logical stage key (empty clears).
	ListStages map[string]string `json:"list_stages"`
	// BugLabelIDs are internal label UUIDs classified as bugs.
	BugLabelIDs []string  `json:"bug_label_ids"`
	UpdatedAt   time.Time `json:"updated_at,omitempty"`
	UpdatedBy   *string   `json:"updated_by,omitempty"`
}

// Snapshot is the JSON revision payload (full settings + maps).
type Snapshot struct {
	LeadTimeStartRule       string            `json:"lead_time_start_rule"`
	LeadTimeStartStage      string            `json:"lead_time_start_stage,omitempty"`
	CycleStartStage         string            `json:"cycle_start_stage,omitempty"`
	DoneStages              []string          `json:"done_stages"`
	BlockedStages           []string          `json:"blocked_stages"`
	WorkStages              []string          `json:"work_stages"`
	BugReopenFromStages     []string          `json:"bug_reopen_from_stages"`
	StaleThresholdSeconds   int               `json:"stale_threshold_seconds"`
	DataStaleSyncAgeSeconds int               `json:"data_stale_sync_age_seconds"`
	Timezone                string            `json:"timezone"`
	ListStages              map[string]string `json:"list_stages"`
	BugLabelIDs             []string          `json:"bug_label_ids"`
}

// Service loads and saves workflow settings.
type Service struct {
	Store *postgres.Store
}

// Load returns current settings for a board (defaults if no row yet).
func (svc *Service) Load(ctx context.Context, boardID string) (Settings, error) {
	if svc == nil || svc.Store == nil {
		return Settings{}, fmt.Errorf("workflow: store is required")
	}
	if boardID == "" {
		return Settings{}, fmt.Errorf("workflow: board id is required")
	}

	lists, err := svc.Store.ListStagesByBoard(ctx, boardID)
	if err != nil {
		return Settings{}, err
	}
	labels, err := svc.Store.ListLabelsByBoard(ctx, boardID)
	if err != nil {
		return Settings{}, err
	}

	listStages := make(map[string]string, len(lists))
	for _, l := range lists {
		if l.LogicalStage != nil && *l.LogicalStage != "" {
			listStages[l.ID] = *l.LogicalStage
		}
	}
	var bugIDs []string
	for _, l := range labels {
		if l.IsBugLabel {
			bugIDs = append(bugIDs, l.ID)
		}
	}

	row, found, err := svc.Store.GetWorkflowSettings(ctx, boardID)
	if err != nil {
		return Settings{}, err
	}
	if !found {
		return Settings{
			BoardID:                 boardID,
			LeadTimeStartRule:       LeadRuleUnset,
			DoneStages:              []string{},
			BlockedStages:           []string{},
			WorkStages:              []string{},
			BugReopenFromStages:     []string{},
			StaleThresholdSeconds:   DefaultStaleThresholdSeconds,
			DataStaleSyncAgeSeconds: DefaultDataStaleSyncAgeSeconds,
			Timezone:                DefaultTimezone,
			ListStages:              listStages,
			BugLabelIDs:             bugIDs,
		}, nil
	}

	out := Settings{
		BoardID:                 boardID,
		LeadTimeStartRule:       row.LeadTimeStartRule,
		DoneStages:              append([]string{}, row.DoneStages...),
		BlockedStages:           append([]string{}, row.BlockedStages...),
		WorkStages:              append([]string{}, row.WorkStages...),
		BugReopenFromStages:     append([]string{}, row.BugReopenFromStages...),
		StaleThresholdSeconds:   row.StaleThresholdSeconds,
		DataStaleSyncAgeSeconds: row.DataStaleSyncAgeSeconds,
		Timezone:                row.Timezone,
		ListStages:              listStages,
		BugLabelIDs:             bugIDs,
		UpdatedAt:               row.UpdatedAt,
		UpdatedBy:               row.UpdatedBy,
	}
	if row.LeadTimeStartStage != nil {
		out.LeadTimeStartStage = *row.LeadTimeStartStage
	}
	if row.CycleStartStage != nil {
		out.CycleStartStage = *row.CycleStartStage
	}
	return out, nil
}

// Save upserts board rules, applies list stages and bug labels, appends a revision.
func (svc *Service) Save(ctx context.Context, boardID string, in Settings, updatedBy *string) (Settings, error) {
	if svc == nil || svc.Store == nil {
		return Settings{}, fmt.Errorf("workflow: store is required")
	}
	if boardID == "" {
		return Settings{}, fmt.Errorf("workflow: board id is required")
	}
	if in.LeadTimeStartRule == "" {
		in.LeadTimeStartRule = LeadRuleUnset
	}
	if err := validateSettings(in); err != nil {
		return Settings{}, err
	}

	row := postgres.WorkflowSettingsRow{
		LeadTimeStartRule:       in.LeadTimeStartRule,
		DoneStages:              in.DoneStages,
		BlockedStages:           in.BlockedStages,
		WorkStages:              in.WorkStages,
		BugReopenFromStages:     in.BugReopenFromStages,
		StaleThresholdSeconds:   in.StaleThresholdSeconds,
		DataStaleSyncAgeSeconds: in.DataStaleSyncAgeSeconds,
		Timezone:                in.Timezone,
	}
	if in.LeadTimeStartStage != "" {
		s := in.LeadTimeStartStage
		row.LeadTimeStartStage = &s
	}
	if in.CycleStartStage != "" {
		s := in.CycleStartStage
		row.CycleStartStage = &s
	}
	if row.StaleThresholdSeconds <= 0 {
		row.StaleThresholdSeconds = DefaultStaleThresholdSeconds
	}
	if row.DataStaleSyncAgeSeconds <= 0 {
		row.DataStaleSyncAgeSeconds = DefaultDataStaleSyncAgeSeconds
	}
	if row.Timezone == "" {
		row.Timezone = DefaultTimezone
	}

	listStages := in.ListStages
	if listStages == nil {
		listStages = map[string]string{}
	}
	bugIDs := in.BugLabelIDs
	if bugIDs == nil {
		bugIDs = []string{}
	}

	snap := Snapshot{
		LeadTimeStartRule:       row.LeadTimeStartRule,
		LeadTimeStartStage:      in.LeadTimeStartStage,
		CycleStartStage:         in.CycleStartStage,
		DoneStages:              nonNil(in.DoneStages),
		BlockedStages:           nonNil(in.BlockedStages),
		WorkStages:              nonNil(in.WorkStages),
		BugReopenFromStages:     nonNil(in.BugReopenFromStages),
		StaleThresholdSeconds:   row.StaleThresholdSeconds,
		DataStaleSyncAgeSeconds: row.DataStaleSyncAgeSeconds,
		Timezone:                row.Timezone,
		ListStages:              listStages,
		BugLabelIDs:             bugIDs,
	}
	payload, err := json.Marshal(snap)
	if err != nil {
		return Settings{}, fmt.Errorf("workflow: marshal snapshot: %w", err)
	}

	saved, err := svc.Store.SaveWorkflowSettingsTx(ctx, boardID, row, listStages, bugIDs, payload, updatedBy)
	if err != nil {
		return Settings{}, err
	}

	out := Settings{
		BoardID:                 boardID,
		LeadTimeStartRule:       saved.LeadTimeStartRule,
		DoneStages:              append([]string{}, saved.DoneStages...),
		BlockedStages:           append([]string{}, saved.BlockedStages...),
		WorkStages:              append([]string{}, saved.WorkStages...),
		BugReopenFromStages:     append([]string{}, saved.BugReopenFromStages...),
		StaleThresholdSeconds:   saved.StaleThresholdSeconds,
		DataStaleSyncAgeSeconds: saved.DataStaleSyncAgeSeconds,
		Timezone:                saved.Timezone,
		ListStages:              listStages,
		BugLabelIDs:             bugIDs,
		UpdatedAt:               saved.UpdatedAt,
		UpdatedBy:               saved.UpdatedBy,
	}
	if saved.LeadTimeStartStage != nil {
		out.LeadTimeStartStage = *saved.LeadTimeStartStage
	}
	if saved.CycleStartStage != nil {
		out.CycleStartStage = *saved.CycleStartStage
	}
	return out, nil
}

func validateSettings(in Settings) error {
	switch in.LeadTimeStartRule {
	case LeadRuleUnset, LeadRuleCardCreated, LeadRuleFirstEnterStage:
	case "":
		// treat empty as unset at save time
	default:
		return fmt.Errorf("workflow: invalid lead_time_start_rule %q", in.LeadTimeStartRule)
	}
	rule := in.LeadTimeStartRule
	if rule == "" {
		rule = LeadRuleUnset
	}
	if rule == LeadRuleFirstEnterStage && in.LeadTimeStartStage == "" {
		return fmt.Errorf("workflow: lead_time_start_stage required when rule is first_enter_stage")
	}
	return nil
}

func nonNil(s []string) []string {
	if s == nil {
		return []string{}
	}
	return s
}
