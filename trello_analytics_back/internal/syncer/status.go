package syncer

import (
	"context"
	"time"

	"trello_analytics_back/internal/storage/postgres"
)

// StatusResponse is the CLI status JSON contract (Architect).
type StatusResponse struct {
	BoardID                 string         `json:"board_id"`
	LastSuccessAt           *time.Time     `json:"last_success_at"`
	CurrentRun              *RunStatusJSON `json:"current_run"`
	EarliestActionAt        *time.Time     `json:"earliest_action_at"`
	LatestActionAt          *time.Time     `json:"latest_action_at"`
	Counts                  StatusCounts   `json:"counts"`
	PrimaryImportIncomplete bool           `json:"primary_import_incomplete"`
	Warnings                []string       `json:"warnings"`
}

// RunStatusJSON is the current_run object.
type RunStatusJSON struct {
	RunID          string     `json:"run_id"`
	Status         string     `json:"status"`
	StartedAt      time.Time  `json:"started_at"`
	FinishedAt     *time.Time `json:"finished_at"`
	ProcessedCount int        `json:"processed_count"`
	CreatedCount   int        `json:"created_count"`
	UpdatedCount   int        `json:"updated_count"`
	ErrorCount     int        `json:"error_count"`
	ErrorSummary   *string    `json:"error_summary"`
}

// StatusCounts aggregates entity/action row counts.
type StatusCounts struct {
	Cards   int64 `json:"cards"`
	Lists   int64 `json:"lists"`
	Members int64 `json:"members"`
	Labels  int64 `json:"labels"`
	Actions int64 `json:"actions"`
}

// Status builds the sync quality status payload (no secrets).
func (o *Orchestrator) Status(ctx context.Context) (StatusResponse, error) {
	out := StatusResponse{
		BoardID:  o.cfg.BoardID,
		Warnings: defaultWarnings(),
	}

	lastSuccess, err := o.store.GetLastSuccessFinishedAt(ctx)
	if err != nil {
		return out, err
	}
	out.LastSuccessAt = lastSuccess

	latest, err := o.store.GetLatestSyncRun(ctx)
	if err != nil {
		return out, err
	}
	if latest != nil {
		out.CurrentRun = &RunStatusJSON{
			RunID:          latest.ID,
			Status:         latest.Status,
			StartedAt:      latest.StartedAt.UTC(),
			FinishedAt:     utcPtr(latest.FinishedAt),
			ProcessedCount: latest.ProcessedCount,
			CreatedCount:   latest.CreatedCount,
			UpdatedCount:   latest.UpdatedCount,
			ErrorCount:     latest.ErrorCount,
			ErrorSummary:   latest.ErrorSummary,
		}
	}

	earliest, latestAct, err := o.store.ActionDateRange(ctx)
	if err != nil {
		return out, err
	}
	out.EarliestActionAt = utcPtr(earliest)
	out.LatestActionAt = utcPtr(latestAct)

	cards, err := o.store.CountCards(ctx)
	if err != nil {
		return out, err
	}
	lists, err := o.store.CountLists(ctx)
	if err != nil {
		return out, err
	}
	members, err := o.store.CountMembers(ctx)
	if err != nil {
		return out, err
	}
	labels, err := o.store.CountLabels(ctx)
	if err != nil {
		return out, err
	}
	actions, err := o.store.CountCardActions(ctx)
	if err != nil {
		return out, err
	}
	out.Counts = StatusCounts{
		Cards:   cards,
		Lists:   lists,
		Members: members,
		Labels:  labels,
		Actions: actions,
	}

	hasCursor, err := o.store.HasSyncCursor(ctx, postgres.ResourceBoardActions)
	if err != nil {
		return out, err
	}

	// Incomplete until at least one full success and board_actions cursor exists.
	out.PrimaryImportIncomplete = lastSuccess == nil || !hasCursor
	if out.PrimaryImportIncomplete {
		out.Warnings = append(out.Warnings, "Primary import incomplete.")
	}
	if latest != nil && latest.Status == postgres.SyncRunFailed && latest.ErrorSummary != nil && *latest.ErrorSummary != "" {
		out.Warnings = append(out.Warnings, "Last sync run failed; see current_run.error_summary.")
	}

	return out, nil
}

func defaultWarnings() []string {
	return []string{
		"Board action history depth may be incomplete; full Trello history truncation is not confirmed.",
		"Card-level /actions feed is not used in v1; card history may be sparse relative to board feed.",
	}
}

func utcPtr(t *time.Time) *time.Time {
	if t == nil {
		return nil
	}
	u := t.UTC()
	return &u
}
