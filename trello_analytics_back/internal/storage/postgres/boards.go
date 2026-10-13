package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
)

// BoardRow is a boards table row.
type BoardRow struct {
	ID            string
	TrelloBoardID string
	Name          string
}

// GetBoardByTrelloID returns the board row for a Trello board id, if present.
func (s *Store) GetBoardByTrelloID(ctx context.Context, trelloBoardID string) (*BoardRow, error) {
	if trelloBoardID == "" {
		return nil, nil
	}
	var b BoardRow
	err := s.pool.QueryRow(ctx, `
		SELECT id::text, trello_board_id, name
		FROM boards WHERE trello_board_id = $1
	`, trelloBoardID).Scan(&b.ID, &b.TrelloBoardID, &b.Name)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("get board by trello id: %w", err)
	}
	return &b, nil
}

// GetBoardByID returns a board by internal UUID.
func (s *Store) GetBoardByID(ctx context.Context, id string) (*BoardRow, error) {
	var b BoardRow
	err := s.pool.QueryRow(ctx, `
		SELECT id::text, trello_board_id, name
		FROM boards WHERE id = $1::uuid
	`, id).Scan(&b.ID, &b.TrelloBoardID, &b.Name)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("get board by id: %w", err)
	}
	return &b, nil
}
