package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
)

// BoardMember is an imported Trello participant (not an app_users row).
type BoardMember struct {
	ID             string
	TrelloMemberID string
	FullName       string
	Username       string
	IsActive       bool
}

// ListMembers returns all members ordered by display name / username.
func (s *Store) ListMembers(ctx context.Context) ([]BoardMember, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT id::text, trello_member_id,
			COALESCE(full_name, ''), COALESCE(username, ''), is_active
		FROM members
		ORDER BY COALESCE(NULLIF(full_name, ''), username, trello_member_id)
	`)
	if err != nil {
		return nil, fmt.Errorf("list members: %w", err)
	}
	defer rows.Close()

	var out []BoardMember
	for rows.Next() {
		var m BoardMember
		if err := rows.Scan(&m.ID, &m.TrelloMemberID, &m.FullName, &m.Username, &m.IsActive); err != nil {
			return nil, err
		}
		out = append(out, m)
	}
	return out, rows.Err()
}

// GetMember returns a member by internal UUID.
func (s *Store) GetMember(ctx context.Context, id string) (BoardMember, error) {
	var m BoardMember
	err := s.pool.QueryRow(ctx, `
		SELECT id::text, trello_member_id,
			COALESCE(full_name, ''), COALESCE(username, ''), is_active
		FROM members
		WHERE id = $1::uuid
	`, id).Scan(&m.ID, &m.TrelloMemberID, &m.FullName, &m.Username, &m.IsActive)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return BoardMember{}, fmt.Errorf("member not found: %w", err)
		}
		return BoardMember{}, fmt.Errorf("get member: %w", err)
	}
	return m, nil
}

// LookupMemberNamesByIDs maps internal member UUID → display name (full_name or username).
func (s *Store) LookupMemberNamesByIDs(ctx context.Context, ids []string) (map[string]string, error) {
	out := map[string]string{}
	if len(ids) == 0 {
		return out, nil
	}
	rows, err := s.pool.Query(ctx, `
		SELECT id::text, COALESCE(NULLIF(full_name, ''), NULLIF(username, ''), '')
		FROM members
		WHERE id = ANY($1::uuid[])
	`, ids)
	if err != nil {
		return nil, fmt.Errorf("lookup member names: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var id, name string
		if err := rows.Scan(&id, &name); err != nil {
			return nil, err
		}
		if name != "" {
			out[id] = name
		}
	}
	return out, rows.Err()
}

// SetMemberActive updates members.is_active (product analytics inclusion).
func (s *Store) SetMemberActive(ctx context.Context, id string, active bool) (BoardMember, error) {
	var m BoardMember
	err := s.pool.QueryRow(ctx, `
		UPDATE members
		SET is_active = $2, updated_at = now()
		WHERE id = $1::uuid
		RETURNING id::text, trello_member_id,
			COALESCE(full_name, ''), COALESCE(username, ''), is_active
	`, id, active).Scan(&m.ID, &m.TrelloMemberID, &m.FullName, &m.Username, &m.IsActive)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return BoardMember{}, fmt.Errorf("member not found: %w", err)
		}
		return BoardMember{}, fmt.Errorf("set member active: %w", err)
	}
	return m, nil
}
