package postgres

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
)

// WorkflowSettingsRow is the board-level workflow rules row.
type WorkflowSettingsRow struct {
	ID                      string
	BoardID                 string
	LeadTimeStartRule       string
	LeadTimeStartStage      *string
	CycleStartStage         *string
	DoneStages              []string
	BlockedStages           []string
	WorkStages              []string
	BugReopenFromStages     []string
	StaleThresholdSeconds   int
	DataStaleSyncAgeSeconds int
	Timezone                string
	UpdatedAt               time.Time
	UpdatedBy               *string
}

// ListStageRow is a board list with optional logical stage.
type ListStageRow struct {
	ID            string
	TrelloListID  string
	Name          string
	LogicalStage  *string
	IsArchived    bool
	IsActive      bool // product analytics inclusion for cards currently in this list
}

// LabelBugRow is a board label with bug flag.
type LabelBugRow struct {
	ID           string
	TrelloLabelID string
	Name         *string
	IsBugLabel   bool
}

// GetWorkflowSettings loads workflow_settings for a board. found=false when no row.
func (s *Store) GetWorkflowSettings(ctx context.Context, boardID string) (WorkflowSettingsRow, bool, error) {
	var row WorkflowSettingsRow
	var leadStage, cycleStage, updatedBy *string
	err := s.pool.QueryRow(ctx, `
		SELECT id::text, board_id::text, lead_time_start_rule,
			lead_time_start_stage, cycle_start_stage,
			COALESCE(done_stages, '{}'), COALESCE(blocked_stages, '{}'), COALESCE(work_stages, '{}'),
			COALESCE(bug_reopen_from_stages, '{}'),
			stale_threshold_seconds, data_stale_sync_age_seconds, timezone,
			updated_at, updated_by::text
		FROM workflow_settings
		WHERE board_id = $1::uuid
	`, boardID).Scan(
		&row.ID, &row.BoardID, &row.LeadTimeStartRule,
		&leadStage, &cycleStage,
		&row.DoneStages, &row.BlockedStages, &row.WorkStages,
		&row.BugReopenFromStages,
		&row.StaleThresholdSeconds, &row.DataStaleSyncAgeSeconds, &row.Timezone,
		&row.UpdatedAt, &updatedBy,
	)
	if err == pgx.ErrNoRows {
		return WorkflowSettingsRow{}, false, nil
	}
	if err != nil {
		return WorkflowSettingsRow{}, false, fmt.Errorf("get workflow_settings: %w", err)
	}
	row.LeadTimeStartStage = leadStage
	row.CycleStartStage = cycleStage
	row.UpdatedBy = updatedBy
	if row.DoneStages == nil {
		row.DoneStages = []string{}
	}
	if row.BlockedStages == nil {
		row.BlockedStages = []string{}
	}
	if row.WorkStages == nil {
		row.WorkStages = []string{}
	}
	if row.BugReopenFromStages == nil {
		row.BugReopenFromStages = []string{}
	}
	return row, true, nil
}

// ListStagesByBoard returns lists for a board with logical_stage.
func (s *Store) ListStagesByBoard(ctx context.Context, boardID string) ([]ListStageRow, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT id::text, trello_list_id, name, logical_stage, is_archived, is_active
		FROM trello_lists
		WHERE board_id = $1::uuid
		ORDER BY name
	`, boardID)
	if err != nil {
		return nil, fmt.Errorf("list stages by board: %w", err)
	}
	defer rows.Close()

	var out []ListStageRow
	for rows.Next() {
		var r ListStageRow
		if err := rows.Scan(&r.ID, &r.TrelloListID, &r.Name, &r.LogicalStage, &r.IsArchived, &r.IsActive); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

// ListLabelsByBoard returns labels for a board with is_bug_label.
func (s *Store) ListLabelsByBoard(ctx context.Context, boardID string) ([]LabelBugRow, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT id::text, trello_label_id, name, is_bug_label
		FROM labels
		WHERE board_id = $1::uuid
		ORDER BY name NULLS LAST
	`, boardID)
	if err != nil {
		return nil, fmt.Errorf("list labels by board: %w", err)
	}
	defer rows.Close()

	var out []LabelBugRow
	for rows.Next() {
		var r LabelBugRow
		if err := rows.Scan(&r.ID, &r.TrelloLabelID, &r.Name, &r.IsBugLabel); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

// SaveWorkflowSettingsTx upserts settings, updates list stages and bug labels, inserts revision.
// listStages maps internal list UUID → logical stage (empty string clears to NULL).
// bugLabelIDs are internal label UUIDs that should have is_bug_label=true; others on board → false.
func (s *Store) SaveWorkflowSettingsTx(
	ctx context.Context,
	boardID string,
	row WorkflowSettingsRow,
	listStages map[string]string,
	bugLabelIDs []string,
	settingsJSON json.RawMessage,
	updatedBy *string,
) (WorkflowSettingsRow, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return WorkflowSettingsRow{}, fmt.Errorf("begin workflow save tx: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var updatedByArg any
	if updatedBy != nil && *updatedBy != "" {
		updatedByArg = *updatedBy
	}

	err = tx.QueryRow(ctx, `
		INSERT INTO workflow_settings (
			board_id, lead_time_start_rule, lead_time_start_stage, cycle_start_stage,
			done_stages, blocked_stages, work_stages, bug_reopen_from_stages,
			stale_threshold_seconds, data_stale_sync_age_seconds, timezone,
			updated_at, updated_by
		) VALUES (
			$1::uuid, $2, $3, $4,
			$5, $6, $7, $8,
			$9, $10, $11,
			now(), $12::uuid
		)
		ON CONFLICT (board_id) DO UPDATE SET
			lead_time_start_rule = EXCLUDED.lead_time_start_rule,
			lead_time_start_stage = EXCLUDED.lead_time_start_stage,
			cycle_start_stage = EXCLUDED.cycle_start_stage,
			done_stages = EXCLUDED.done_stages,
			blocked_stages = EXCLUDED.blocked_stages,
			work_stages = EXCLUDED.work_stages,
			bug_reopen_from_stages = EXCLUDED.bug_reopen_from_stages,
			stale_threshold_seconds = EXCLUDED.stale_threshold_seconds,
			data_stale_sync_age_seconds = EXCLUDED.data_stale_sync_age_seconds,
			timezone = EXCLUDED.timezone,
			updated_at = now(),
			updated_by = EXCLUDED.updated_by
		RETURNING id::text, board_id::text, lead_time_start_rule,
			lead_time_start_stage, cycle_start_stage,
			COALESCE(done_stages, '{}'), COALESCE(blocked_stages, '{}'), COALESCE(work_stages, '{}'),
			COALESCE(bug_reopen_from_stages, '{}'),
			stale_threshold_seconds, data_stale_sync_age_seconds, timezone,
			updated_at, updated_by::text
	`,
		boardID, row.LeadTimeStartRule, nullStringPtr(row.LeadTimeStartStage), nullStringPtr(row.CycleStartStage),
		nonNilStrings(row.DoneStages), nonNilStrings(row.BlockedStages), nonNilStrings(row.WorkStages),
		nonNilStrings(row.BugReopenFromStages),
		row.StaleThresholdSeconds, row.DataStaleSyncAgeSeconds, row.Timezone,
		updatedByArg,
	).Scan(
		&row.ID, &row.BoardID, &row.LeadTimeStartRule,
		&row.LeadTimeStartStage, &row.CycleStartStage,
		&row.DoneStages, &row.BlockedStages, &row.WorkStages,
		&row.BugReopenFromStages,
		&row.StaleThresholdSeconds, &row.DataStaleSyncAgeSeconds, &row.Timezone,
		&row.UpdatedAt, &row.UpdatedBy,
	)
	if err != nil {
		return WorkflowSettingsRow{}, fmt.Errorf("upsert workflow_settings: %w", err)
	}

	// Apply list→stage map for listed IDs; leave unlisted lists unchanged.
	for listID, stage := range listStages {
		var stageArg any
		if stage != "" {
			stageArg = stage
		}
		tag, err := tx.Exec(ctx, `
			UPDATE trello_lists
			SET logical_stage = $2, updated_at = now()
			WHERE id = $1::uuid AND board_id = $3::uuid
		`, listID, stageArg, boardID)
		if err != nil {
			return WorkflowSettingsRow{}, fmt.Errorf("update list stage %s: %w", listID, err)
		}
		if tag.RowsAffected() == 0 {
			return WorkflowSettingsRow{}, fmt.Errorf("update list stage: list %s not found on board", listID)
		}
	}

	// Reset all board bug flags, then set selected.
	if _, err := tx.Exec(ctx, `
		UPDATE labels SET is_bug_label = false WHERE board_id = $1::uuid
	`, boardID); err != nil {
		return WorkflowSettingsRow{}, fmt.Errorf("clear bug labels: %w", err)
	}
	for _, labelID := range bugLabelIDs {
		tag, err := tx.Exec(ctx, `
			UPDATE labels SET is_bug_label = true
			WHERE id = $1::uuid AND board_id = $2::uuid
		`, labelID, boardID)
		if err != nil {
			return WorkflowSettingsRow{}, fmt.Errorf("set bug label %s: %w", labelID, err)
		}
		if tag.RowsAffected() == 0 {
			return WorkflowSettingsRow{}, fmt.Errorf("set bug label: label %s not found on board", labelID)
		}
	}

	if _, err := tx.Exec(ctx, `
		INSERT INTO workflow_settings_revisions (board_id, settings_json, created_by)
		VALUES ($1::uuid, $2, $3::uuid)
	`, boardID, settingsJSON, updatedByArg); err != nil {
		return WorkflowSettingsRow{}, fmt.Errorf("insert workflow revision: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return WorkflowSettingsRow{}, fmt.Errorf("commit workflow save: %w", err)
	}
	return row, nil
}

// CountWorkflowRevisions returns revision count for a board (tests).
func (s *Store) CountWorkflowRevisions(ctx context.Context, boardID string) (int64, error) {
	var n int64
	err := s.pool.QueryRow(ctx, `
		SELECT COUNT(*) FROM workflow_settings_revisions WHERE board_id = $1::uuid
	`, boardID).Scan(&n)
	if err != nil {
		return 0, fmt.Errorf("count workflow revisions: %w", err)
	}
	return n, nil
}

func nullStringPtr(p *string) any {
	if p == nil || *p == "" {
		return nil
	}
	return *p
}

func nonNilStrings(s []string) []string {
	if s == nil {
		return []string{}
	}
	return s
}
