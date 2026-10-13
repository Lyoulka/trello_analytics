package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
)

// ResourceBoardActions is the sync_cursors.resource_type for board actions pagination.
const ResourceBoardActions = "board_actions"

// GetSyncCursor returns cursor_value for resource_type.
// found=false when no row exists; empty/NULL cursor_value means "start without before".
func (s *Store) GetSyncCursor(ctx context.Context, resourceType string) (cursorValue string, found bool, err error) {
	var value *string
	err = s.pool.QueryRow(ctx, `
		SELECT cursor_value
		FROM sync_cursors
		WHERE resource_type = $1
	`, resourceType).Scan(&value)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", false, nil
	}
	if err != nil {
		return "", false, fmt.Errorf("get sync cursor %q: %w", resourceType, err)
	}
	if value == nil {
		return "", true, nil
	}
	return *value, true, nil
}

// UpsertSyncCursor writes cursor_value and bumps last_success_at / updated_at (UTC via now()).
// Prefer CommitBoardActionsPage for page atomicity; this helper is for direct updates.
func (s *Store) UpsertSyncCursor(ctx context.Context, resourceType, cursorValue string) error {
	_, err := s.pool.Exec(ctx, upsertSyncCursorSQL, resourceType, nullIfEmpty(cursorValue))
	if err != nil {
		return fmt.Errorf("upsert sync cursor %q: %w", resourceType, err)
	}
	return nil
}

const upsertSyncCursorSQL = `
INSERT INTO sync_cursors (resource_type, cursor_value, last_success_at, updated_at)
VALUES ($1, $2, now(), now())
ON CONFLICT (resource_type) DO UPDATE SET
	cursor_value = EXCLUDED.cursor_value,
	last_success_at = EXCLUDED.last_success_at,
	updated_at = EXCLUDED.updated_at
`

func upsertSyncCursorTx(ctx context.Context, tx pgx.Tx, resourceType, cursorValue string) error {
	_, err := tx.Exec(ctx, upsertSyncCursorSQL, resourceType, nullIfEmpty(cursorValue))
	if err != nil {
		return fmt.Errorf("upsert sync cursor %q: %w", resourceType, err)
	}
	return nil
}

func nullIfEmpty(s string) any {
	if s == "" {
		return nil
	}
	return s
}
