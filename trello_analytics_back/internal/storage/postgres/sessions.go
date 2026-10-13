package postgres

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"

	"trello_analytics_back/internal/auth"
)

// CreateSession inserts a session row for the hashed cookie value.
func (s *Store) CreateSession(ctx context.Context, userID, tokenHash string, expiresAt time.Time) (sessionID string, err error) {
	if userID == "" || tokenHash == "" {
		return "", fmt.Errorf("create session: missing fields")
	}
	err = s.pool.QueryRow(ctx, `
		INSERT INTO sessions (user_id, token_hash, expires_at)
		VALUES ($1::uuid, $2, $3)
		RETURNING id::text
	`, userID, tokenHash, expiresAt.UTC()).Scan(&sessionID)
	if err != nil {
		return "", fmt.Errorf("create session: %w", err)
	}
	return sessionID, nil
}

// SessionUser is the joined session + user payload for auth middleware.
type SessionUser struct {
	SessionID string
	ExpiresAt time.Time
	User      auth.User
}

// GetActiveSessionByTokenHash returns a non-revoked, non-expired session with its user.
func (s *Store) GetActiveSessionByTokenHash(ctx context.Context, tokenHash string) (*SessionUser, error) {
	var su SessionUser
	err := s.pool.QueryRow(ctx, `
		SELECT s.id::text, s.expires_at,
			u.id::text, u.email, u.display_name, u.role, u.is_active
		FROM sessions s
		INNER JOIN app_users u ON u.id = s.user_id
		WHERE s.token_hash = $1
		  AND s.revoked_at IS NULL
		  AND s.expires_at > now()
		  AND u.is_active = true
	`, tokenHash).Scan(
		&su.SessionID, &su.ExpiresAt,
		&su.User.ID, &su.User.Email, &su.User.DisplayName, &su.User.Role, &su.User.IsActive,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("get session: %w", err)
	}
	return &su, nil
}

// RevokeSessionByTokenHash marks a session revoked (logout).
func (s *Store) RevokeSessionByTokenHash(ctx context.Context, tokenHash string) error {
	_, err := s.pool.Exec(ctx, `
		UPDATE sessions SET revoked_at = now()
		WHERE token_hash = $1 AND revoked_at IS NULL
	`, tokenHash)
	if err != nil {
		return fmt.Errorf("revoke session: %w", err)
	}
	return nil
}

// RevokeAllSessionsForUser revokes every active session for a user (deactivate).
func (s *Store) RevokeAllSessionsForUser(ctx context.Context, userID string) error {
	_, err := s.pool.Exec(ctx, `
		UPDATE sessions SET revoked_at = now()
		WHERE user_id = $1::uuid AND revoked_at IS NULL
	`, userID)
	if err != nil {
		return fmt.Errorf("revoke user sessions: %w", err)
	}
	return nil
}
