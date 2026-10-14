package postgres

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
)

// CardActionInsert is one raw Trello action row for idempotent insert.
// card_id / member_id / list FKs may be nil (allowed by schema; entity linking is later work).
type CardActionInsert struct {
	TrelloActionID string
	ActionType     string
	ActionDate     time.Time // UTC
	RawPayload     json.RawMessage
}

// CommitBoardActionsPage inserts actions (ON CONFLICT DO NOTHING) and upserts sync_cursors
// in a single transaction. Cursor advances only after successful COMMIT.
// cursorValue is the last committed action id (next Trello `before`); empty clears to NULL.
func (s *Store) CommitBoardActionsPage(
	ctx context.Context,
	resourceType string,
	actions []CardActionInsert,
	cursorValue string,
) (inserted int64, err error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return 0, fmt.Errorf("begin board actions page tx: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	n, err := insertCardActionsTx(ctx, tx, actions)
	if err != nil {
		return 0, err
	}
	if err := upsertSyncCursorTx(ctx, tx, resourceType, cursorValue); err != nil {
		return 0, err
	}
	if err := tx.Commit(ctx); err != nil {
		return 0, fmt.Errorf("commit board actions page: %w", err)
	}
	return n, nil
}

// InsertCardActionsPage inserts actions (ON CONFLICT DO NOTHING) without touching sync_cursors.
// Used by head catch-up so the oldest backfill cursor is preserved.
func (s *Store) InsertCardActionsPage(ctx context.Context, actions []CardActionInsert) (inserted int64, err error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return 0, fmt.Errorf("begin insert card actions page tx: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	n, err := insertCardActionsTx(ctx, tx, actions)
	if err != nil {
		return 0, err
	}
	if err := tx.Commit(ctx); err != nil {
		return 0, fmt.Errorf("commit insert card actions page: %w", err)
	}
	return n, nil
}

func insertCardActionsTx(ctx context.Context, tx pgx.Tx, actions []CardActionInsert) (int64, error) {
	var inserted int64
	for _, a := range actions {
		if a.TrelloActionID == "" {
			return 0, fmt.Errorf("card_actions insert: empty trello_action_id")
		}
		if a.ActionType == "" {
			return 0, fmt.Errorf("card_actions insert: empty action_type for %s", a.TrelloActionID)
		}
		if a.ActionDate.IsZero() {
			return 0, fmt.Errorf("card_actions insert: zero action_date for %s", a.TrelloActionID)
		}
		var payload any
		if len(a.RawPayload) > 0 {
			payload = a.RawPayload
		}
		tag, err := tx.Exec(ctx, `
			INSERT INTO card_actions (trello_action_id, action_type, action_date, raw_payload)
			VALUES ($1, $2, $3, $4)
			ON CONFLICT (trello_action_id) DO NOTHING
		`, a.TrelloActionID, a.ActionType, a.ActionDate.UTC(), payload)
		if err != nil {
			return 0, fmt.Errorf("insert card_action %s: %w", a.TrelloActionID, err)
		}
		inserted += tag.RowsAffected()
	}
	return inserted, nil
}

// CountCardActions returns total rows in card_actions (test / status helpers).
func (s *Store) CountCardActions(ctx context.Context) (int64, error) {
	var n int64
	err := s.pool.QueryRow(ctx, `SELECT COUNT(*) FROM card_actions`).Scan(&n)
	if err != nil {
		return 0, fmt.Errorf("count card_actions: %w", err)
	}
	return n, nil
}

// ListCardActionIDs returns all trello_action_id values ordered for stable assertions.
func (s *Store) ListCardActionIDs(ctx context.Context) ([]string, error) {
	rows, err := s.pool.Query(ctx, `SELECT trello_action_id FROM card_actions ORDER BY trello_action_id`)
	if err != nil {
		return nil, fmt.Errorf("list card_action ids: %w", err)
	}
	defer rows.Close()

	var ids []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}
