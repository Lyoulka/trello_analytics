package postgres

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
)

// Sync run statuses (CHECK constraint in migration 000002).
const (
	SyncRunRunning   = "running"
	SyncRunSuccess   = "success"
	SyncRunFailed    = "failed"
	SyncRunCancelled = "cancelled"
)

// SyncRun is a row from sync_runs.
type SyncRun struct {
	ID             string     `json:"id"`
	StartedAt      time.Time  `json:"started_at"`
	FinishedAt     *time.Time `json:"finished_at"`
	Status         string     `json:"status"`
	ProcessedCount int        `json:"processed_count"`
	CreatedCount   int        `json:"created_count"`
	UpdatedCount   int        `json:"updated_count"`
	ErrorCount     int        `json:"error_count"`
	ErrorSummary   *string    `json:"error_summary"`
}

// SyncRunCounts are counters updated during/after a run.
type SyncRunCounts struct {
	ProcessedCount int
	CreatedCount   int
	UpdatedCount   int
	ErrorCount     int
}

// InsertSyncRun creates a running sync_runs row and returns its UUID.
func (s *Store) InsertSyncRun(ctx context.Context, startedAt time.Time) (id string, err error) {
	err = s.pool.QueryRow(ctx, `
		INSERT INTO sync_runs (started_at, status)
		VALUES ($1, $2)
		RETURNING id::text
	`, startedAt.UTC(), SyncRunRunning).Scan(&id)
	if err != nil {
		return "", fmt.Errorf("insert sync run: %w", err)
	}
	return id, nil
}

// FinishSyncRun sets terminal status, finished_at, counters, and optional error_summary.
func (s *Store) FinishSyncRun(ctx context.Context, id, status string, counts SyncRunCounts, errorSummary *string) error {
	switch status {
	case SyncRunSuccess, SyncRunFailed, SyncRunCancelled:
	default:
		return fmt.Errorf("finish sync run: invalid terminal status %q", status)
	}
	tag, err := s.pool.Exec(ctx, `
		UPDATE sync_runs SET
			status = $2,
			finished_at = now(),
			processed_count = $3,
			created_count = $4,
			updated_count = $5,
			error_count = $6,
			error_summary = $7
		WHERE id = $1::uuid
	`, id, status, counts.ProcessedCount, counts.CreatedCount, counts.UpdatedCount, counts.ErrorCount, errorSummary)
	if err != nil {
		return fmt.Errorf("finish sync run %s: %w", id, err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("finish sync run %s: not found", id)
	}
	return nil
}

// UpdateSyncRunCounts updates counters on a still-running row (optional progress).
func (s *Store) UpdateSyncRunCounts(ctx context.Context, id string, counts SyncRunCounts) error {
	_, err := s.pool.Exec(ctx, `
		UPDATE sync_runs SET
			processed_count = $2,
			created_count = $3,
			updated_count = $4,
			error_count = $5
		WHERE id = $1::uuid AND status = $6
	`, id, counts.ProcessedCount, counts.CreatedCount, counts.UpdatedCount, counts.ErrorCount, SyncRunRunning)
	if err != nil {
		return fmt.Errorf("update sync run counts %s: %w", id, err)
	}
	return nil
}

// ReclaimOrphanSyncRuns marks all running rows as failed after process restart.
// error_summary is a fixed non-secret message.
func (s *Store) ReclaimOrphanSyncRuns(ctx context.Context) (int64, error) {
	const msg = "orphaned after process restart"
	tag, err := s.pool.Exec(ctx, `
		UPDATE sync_runs SET
			status = $1,
			finished_at = now(),
			error_summary = $2
		WHERE status = $3
	`, SyncRunFailed, msg, SyncRunRunning)
	if err != nil {
		return 0, fmt.Errorf("reclaim orphan sync runs: %w", err)
	}
	return tag.RowsAffected(), nil
}

// GetSyncRun returns a sync_runs row by id, or nil if not found.
func (s *Store) GetSyncRun(ctx context.Context, id string) (*SyncRun, error) {
	row, err := s.scanSyncRun(ctx, `
		SELECT id::text, started_at, finished_at, status,
			processed_count, created_count, updated_count, error_count, error_summary
		FROM sync_runs
		WHERE id = $1::uuid
	`, id)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get sync run %s: %w", id, err)
	}
	return row, nil
}

// GetRunningSyncRun returns the current running row, if any.
func (s *Store) GetRunningSyncRun(ctx context.Context) (*SyncRun, error) {
	row, err := s.scanSyncRun(ctx, `
		SELECT id::text, started_at, finished_at, status,
			processed_count, created_count, updated_count, error_count, error_summary
		FROM sync_runs
		WHERE status = $1
		ORDER BY started_at DESC
		LIMIT 1
	`, SyncRunRunning)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	return row, err
}

// GetLatestSyncRun returns the most recent run by started_at (any status), or nil.
func (s *Store) GetLatestSyncRun(ctx context.Context) (*SyncRun, error) {
	row, err := s.scanSyncRun(ctx, `
		SELECT id::text, started_at, finished_at, status,
			processed_count, created_count, updated_count, error_count, error_summary
		FROM sync_runs
		ORDER BY started_at DESC
		LIMIT 1
	`)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	return row, err
}

// GetLastSuccessFinishedAt returns finished_at of the latest success run, or nil.
func (s *Store) GetLastSuccessFinishedAt(ctx context.Context) (*time.Time, error) {
	var finished *time.Time
	err := s.pool.QueryRow(ctx, `
		SELECT finished_at
		FROM sync_runs
		WHERE status = $1 AND finished_at IS NOT NULL
		ORDER BY finished_at DESC
		LIMIT 1
	`, SyncRunSuccess).Scan(&finished)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("last success finished_at: %w", err)
	}
	return finished, nil
}

// ActionDateRange returns MIN/MAX(action_date) from card_actions (UTC), or nils if empty.
func (s *Store) ActionDateRange(ctx context.Context) (earliest, latest *time.Time, err error) {
	err = s.pool.QueryRow(ctx, `
		SELECT MIN(action_date), MAX(action_date)
		FROM card_actions
	`).Scan(&earliest, &latest)
	if err != nil {
		return nil, nil, fmt.Errorf("action date range: %w", err)
	}
	return earliest, latest, nil
}

// ListSyncRuns returns recent sync runs ordered by started_at DESC.
func (s *Store) ListSyncRuns(ctx context.Context, limit int) ([]SyncRun, error) {
	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}
	rows, err := s.pool.Query(ctx, `
		SELECT id::text, started_at, finished_at, status,
			processed_count, created_count, updated_count, error_count, error_summary
		FROM sync_runs
		ORDER BY started_at DESC
		LIMIT $1
	`, limit)
	if err != nil {
		return nil, fmt.Errorf("list sync runs: %w", err)
	}
	defer rows.Close()
	var out []SyncRun
	for rows.Next() {
		var r SyncRun
		if err := rows.Scan(
			&r.ID, &r.StartedAt, &r.FinishedAt, &r.Status,
			&r.ProcessedCount, &r.CreatedCount, &r.UpdatedCount, &r.ErrorCount, &r.ErrorSummary,
		); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

// HasSyncCursor reports whether a sync_cursors row exists for resource_type.
func (s *Store) HasSyncCursor(ctx context.Context, resourceType string) (bool, error) {
	var exists bool
	err := s.pool.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1 FROM sync_cursors WHERE resource_type = $1
		)
	`, resourceType).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("has sync cursor %q: %w", resourceType, err)
	}
	return exists, nil
}

func (s *Store) scanSyncRun(ctx context.Context, query string, args ...any) (*SyncRun, error) {
	var r SyncRun
	err := s.pool.QueryRow(ctx, query, args...).Scan(
		&r.ID,
		&r.StartedAt,
		&r.FinishedAt,
		&r.Status,
		&r.ProcessedCount,
		&r.CreatedCount,
		&r.UpdatedCount,
		&r.ErrorCount,
		&r.ErrorSummary,
	)
	if err != nil {
		return nil, err
	}
	return &r, nil
}
