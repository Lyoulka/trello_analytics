package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
)

// AnalyticsCard is card state needed by the metrics engine.
type AnalyticsCard struct {
	ID                 string
	TrelloCardID       string
	Name               string
	CurrentListID      *string
	CreatedAtTrello    *time.Time
	IsArchived         bool
	LastTrelloUpdateAt *time.Time
}

// AnalyticsAction is a raw action row for transition parsing.
type AnalyticsAction struct {
	ID             string
	TrelloActionID string
	ActionType     string
	ActionDate     time.Time
	RawPayload     json.RawMessage
}

// AnalyticsCardMember is a current card↔member assignment (removed_at IS NULL).
type AnalyticsCardMember struct {
	CardID   string
	MemberID string
	IsActive bool // members.is_active — product analytics inclusion flag
}

// AnalyticsMember is a board-visible member display row for workload labels.
type AnalyticsMember struct {
	ID       string
	FullName string
	Username string
}

// AnalyticsCardLabel is a card↔label link.
type AnalyticsCardLabel struct {
	CardID  string
	LabelID string
}

// LoadAnalyticsCards loads cards for a board.
func (s *Store) LoadAnalyticsCards(ctx context.Context, boardID string) ([]AnalyticsCard, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT id::text, trello_card_id, name, current_list_id::text,
			created_at_trello, is_archived, last_trello_update_at
		FROM cards
		WHERE board_id = $1::uuid
	`, boardID)
	if err != nil {
		return nil, fmt.Errorf("load analytics cards: %w", err)
	}
	defer rows.Close()

	var out []AnalyticsCard
	for rows.Next() {
		var c AnalyticsCard
		if err := rows.Scan(
			&c.ID, &c.TrelloCardID, &c.Name, &c.CurrentListID,
			&c.CreatedAtTrello, &c.IsArchived, &c.LastTrelloUpdateAt,
		); err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

// LoadAnalyticsActions loads card_actions for cards belonging to the board, ordered by date ascending.
// v1: importer often leaves card_id NULL; match via card_id FK or raw_payload data.card.id → cards.trello_card_id.
func (s *Store) LoadAnalyticsActions(ctx context.Context, boardID string) ([]AnalyticsAction, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT ca.id::text, ca.trello_action_id, ca.action_type, ca.action_date, ca.raw_payload
		FROM card_actions ca
		WHERE EXISTS (
			SELECT 1 FROM cards c
			WHERE c.board_id = $1::uuid
			  AND (
				c.id = ca.card_id
				OR (
					ca.card_id IS NULL
					AND c.trello_card_id = ca.raw_payload #>> '{data,card,id}'
				)
			  )
		)
		ORDER BY ca.action_date ASC, ca.trello_action_id ASC
	`, boardID)
	if err != nil {
		return nil, fmt.Errorf("load analytics actions: %w", err)
	}
	defer rows.Close()

	var out []AnalyticsAction
	for rows.Next() {
		var a AnalyticsAction
		if err := rows.Scan(&a.ID, &a.TrelloActionID, &a.ActionType, &a.ActionDate, &a.RawPayload); err != nil {
			return nil, err
		}
		out = append(out, a)
	}
	return out, rows.Err()
}

// LoadAnalyticsCardMembers loads current assignees (removed_at IS NULL) with is_active.
func (s *Store) LoadAnalyticsCardMembers(ctx context.Context, boardID string) ([]AnalyticsCardMember, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT cm.card_id::text, cm.member_id::text, m.is_active
		FROM card_members cm
		INNER JOIN cards c ON c.id = cm.card_id
		INNER JOIN members m ON m.id = cm.member_id
		WHERE c.board_id = $1::uuid AND cm.removed_at IS NULL
	`, boardID)
	if err != nil {
		return nil, fmt.Errorf("load analytics card members: %w", err)
	}
	defer rows.Close()

	var out []AnalyticsCardMember
	for rows.Next() {
		var m AnalyticsCardMember
		if err := rows.Scan(&m.CardID, &m.MemberID, &m.IsActive); err != nil {
			return nil, err
		}
		out = append(out, m)
	}
	return out, rows.Err()
}

// LoadAnalyticsMembers loads active members that currently appear on cards of the board.
func (s *Store) LoadAnalyticsMembers(ctx context.Context, boardID string) ([]AnalyticsMember, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT DISTINCT m.id::text, COALESCE(m.full_name, ''), COALESCE(m.username, '')
		FROM members m
		INNER JOIN card_members cm ON cm.member_id = m.id AND cm.removed_at IS NULL
		INNER JOIN cards c ON c.id = cm.card_id
		WHERE c.board_id = $1::uuid AND m.is_active = true
	`, boardID)
	if err != nil {
		return nil, fmt.Errorf("load analytics members: %w", err)
	}
	defer rows.Close()

	var out []AnalyticsMember
	for rows.Next() {
		var m AnalyticsMember
		if err := rows.Scan(&m.ID, &m.FullName, &m.Username); err != nil {
			return nil, err
		}
		out = append(out, m)
	}
	return out, rows.Err()
}

// LoadAnalyticsCardLabels loads card_labels for cards on a board.
func (s *Store) LoadAnalyticsCardLabels(ctx context.Context, boardID string) ([]AnalyticsCardLabel, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT cl.card_id::text, cl.label_id::text
		FROM card_labels cl
		INNER JOIN cards c ON c.id = cl.card_id
		WHERE c.board_id = $1::uuid
	`, boardID)
	if err != nil {
		return nil, fmt.Errorf("load analytics card labels: %w", err)
	}
	defer rows.Close()

	var out []AnalyticsCardLabel
	for rows.Next() {
		var l AnalyticsCardLabel
		if err := rows.Scan(&l.CardID, &l.LabelID); err != nil {
			return nil, err
		}
		out = append(out, l)
	}
	return out, rows.Err()
}

// LastSyncSuccessAt returns finished_at of the latest success sync_run, if any.
func (s *Store) LastSyncSuccessAt(ctx context.Context) (*time.Time, error) {
	var t time.Time
	err := s.pool.QueryRow(ctx, `
		SELECT finished_at FROM sync_runs
		WHERE status = 'success' AND finished_at IS NOT NULL
		ORDER BY finished_at DESC
		LIMIT 1
	`).Scan(&t)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("last sync success: %w", err)
	}
	utc := t.UTC()
	return &utc, nil
}
