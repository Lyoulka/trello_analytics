package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
)

// BoardList is an imported Trello column (list).
type BoardList struct {
	ID           string
	TrelloListID string
	Name         string
	IsArchived   bool
	IsActive     bool
	LogicalStage *string
}

// ListBoardLists returns all lists ordered by name.
func (s *Store) ListBoardLists(ctx context.Context) ([]BoardList, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT id::text, trello_list_id, name, is_archived, is_active, logical_stage
		FROM trello_lists
		ORDER BY name
	`)
	if err != nil {
		return nil, fmt.Errorf("list board lists: %w", err)
	}
	defer rows.Close()

	var out []BoardList
	for rows.Next() {
		var l BoardList
		if err := rows.Scan(&l.ID, &l.TrelloListID, &l.Name, &l.IsArchived, &l.IsActive, &l.LogicalStage); err != nil {
			return nil, err
		}
		out = append(out, l)
	}
	return out, rows.Err()
}

// SetListActive updates trello_lists.is_active (product analytics inclusion).
func (s *Store) SetListActive(ctx context.Context, id string, active bool) (BoardList, error) {
	var l BoardList
	err := s.pool.QueryRow(ctx, `
		UPDATE trello_lists
		SET is_active = $2, updated_at = now()
		WHERE id = $1::uuid
		RETURNING id::text, trello_list_id, name, is_archived, is_active, logical_stage
	`, id, active).Scan(&l.ID, &l.TrelloListID, &l.Name, &l.IsArchived, &l.IsActive, &l.LogicalStage)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return BoardList{}, fmt.Errorf("list not found: %w", err)
		}
		return BoardList{}, fmt.Errorf("set list active: %w", err)
	}
	return l, nil
}
