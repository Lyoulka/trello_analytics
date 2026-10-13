package postgres

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"

	"trello_analytics_back/internal/auth"
)

// AppUser is a row from app_users (includes password hash for login only).
type AppUser struct {
	ID           string
	Email        string
	PasswordHash string
	DisplayName  string
	Role         string
	IsActive     bool
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

// PublicUser strips the password hash for API responses.
func (u AppUser) Public() auth.User {
	return auth.User{
		ID:          u.ID,
		Email:       u.Email,
		DisplayName: u.DisplayName,
		Role:        u.Role,
		IsActive:    u.IsActive,
	}
}

// CountUsers returns the number of rows in app_users.
func (s *Store) CountUsers(ctx context.Context) (int64, error) {
	var n int64
	err := s.pool.QueryRow(ctx, `SELECT COUNT(*) FROM app_users`).Scan(&n)
	if err != nil {
		return 0, fmt.Errorf("count users: %w", err)
	}
	return n, nil
}

// CreateUser inserts a new app_users row.
func (s *Store) CreateUser(ctx context.Context, email, passwordHash, displayName, role string) (auth.User, error) {
	email = strings.TrimSpace(strings.ToLower(email))
	if email == "" || passwordHash == "" || displayName == "" || !auth.ValidRole(role) {
		return auth.User{}, fmt.Errorf("create user: invalid input")
	}
	var u AppUser
	err := s.pool.QueryRow(ctx, `
		INSERT INTO app_users (email, password_hash, display_name, role)
		VALUES ($1, $2, $3, $4)
		RETURNING id::text, email, password_hash, display_name, role, is_active, created_at, updated_at
	`, email, passwordHash, displayName, role).Scan(
		&u.ID, &u.Email, &u.PasswordHash, &u.DisplayName, &u.Role, &u.IsActive, &u.CreatedAt, &u.UpdatedAt,
	)
	if err != nil {
		return auth.User{}, fmt.Errorf("create user: %w", err)
	}
	return u.Public(), nil
}

// GetUserByEmail loads a user by email (case-insensitive).
func (s *Store) GetUserByEmail(ctx context.Context, email string) (*AppUser, error) {
	email = strings.TrimSpace(strings.ToLower(email))
	var u AppUser
	err := s.pool.QueryRow(ctx, `
		SELECT id::text, email, password_hash, display_name, role, is_active, created_at, updated_at
		FROM app_users WHERE lower(email) = $1
	`, email).Scan(
		&u.ID, &u.Email, &u.PasswordHash, &u.DisplayName, &u.Role, &u.IsActive, &u.CreatedAt, &u.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("get user by email: %w", err)
	}
	return &u, nil
}

// GetUserByID loads a user by UUID.
func (s *Store) GetUserByID(ctx context.Context, id string) (*AppUser, error) {
	var u AppUser
	err := s.pool.QueryRow(ctx, `
		SELECT id::text, email, password_hash, display_name, role, is_active, created_at, updated_at
		FROM app_users WHERE id = $1::uuid
	`, id).Scan(
		&u.ID, &u.Email, &u.PasswordHash, &u.DisplayName, &u.Role, &u.IsActive, &u.CreatedAt, &u.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("get user by id: %w", err)
	}
	return &u, nil
}

// ListUsers returns all app users ordered by email (no password hashes in Public).
func (s *Store) ListUsers(ctx context.Context) ([]auth.User, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT id::text, email, display_name, role, is_active
		FROM app_users
		ORDER BY email ASC
	`)
	if err != nil {
		return nil, fmt.Errorf("list users: %w", err)
	}
	defer rows.Close()
	var out []auth.User
	for rows.Next() {
		var u auth.User
		if err := rows.Scan(&u.ID, &u.Email, &u.DisplayName, &u.Role, &u.IsActive); err != nil {
			return nil, err
		}
		out = append(out, u)
	}
	return out, rows.Err()
}

// UserUpdate is a partial update for Admin PATCH.
type UserUpdate struct {
	DisplayName *string
	Role        *string
	IsActive    *bool
	PasswordHash *string
}

// UpdateUser applies a partial update and returns the public user.
func (s *Store) UpdateUser(ctx context.Context, id string, upd UserUpdate) (*auth.User, error) {
	cur, err := s.GetUserByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if cur == nil {
		return nil, nil
	}
	display := cur.DisplayName
	role := cur.Role
	active := cur.IsActive
	hash := cur.PasswordHash
	if upd.DisplayName != nil {
		display = strings.TrimSpace(*upd.DisplayName)
		if display == "" {
			return nil, fmt.Errorf("update user: empty display_name")
		}
	}
	if upd.Role != nil {
		if !auth.ValidRole(*upd.Role) {
			return nil, fmt.Errorf("update user: invalid role")
		}
		role = *upd.Role
	}
	if upd.IsActive != nil {
		active = *upd.IsActive
	}
	if upd.PasswordHash != nil {
		if *upd.PasswordHash == "" {
			return nil, fmt.Errorf("update user: empty password hash")
		}
		hash = *upd.PasswordHash
	}
	var u AppUser
	err = s.pool.QueryRow(ctx, `
		UPDATE app_users SET
			display_name = $2,
			role = $3,
			is_active = $4,
			password_hash = $5,
			updated_at = now()
		WHERE id = $1::uuid
		RETURNING id::text, email, password_hash, display_name, role, is_active, created_at, updated_at
	`, id, display, role, active, hash).Scan(
		&u.ID, &u.Email, &u.PasswordHash, &u.DisplayName, &u.Role, &u.IsActive, &u.CreatedAt, &u.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("update user: %w", err)
	}
	pub := u.Public()
	return &pub, nil
}
