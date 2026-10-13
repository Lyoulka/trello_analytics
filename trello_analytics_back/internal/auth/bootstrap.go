package auth

import (
	"context"
	"fmt"
	"log/slog"
)

// UserStore is the persistence surface needed for bootstrap and login.
type UserStore interface {
	CountUsers(ctx context.Context) (int64, error)
	CreateUser(ctx context.Context, email, passwordHash, displayName, role string) (User, error)
}

// BootstrapAdmin creates the first Admin when app_users is empty and env credentials are set.
// If users already exist → no-op. If empty and env missing → Warn log, server continues.
func BootstrapAdmin(ctx context.Context, store UserStore, email, password string, logger *slog.Logger) error {
	if logger == nil {
		logger = slog.Default()
	}
	if store == nil {
		return fmt.Errorf("auth: bootstrap store is required")
	}
	n, err := store.CountUsers(ctx)
	if err != nil {
		return fmt.Errorf("auth: count users: %w", err)
	}
	if n > 0 {
		return nil
	}
	if email == "" || password == "" {
		logger.Warn("bootstrap admin skipped: app_users empty and BOOTSTRAP_ADMIN_EMAIL/PASSWORD not set")
		return nil
	}
	if err := ValidatePassword(password); err != nil {
		return fmt.Errorf("auth: bootstrap admin password must be at least %d characters", MinPasswordLength)
	}
	hash, err := HashPassword(password)
	if err != nil {
		return err
	}
	u, err := store.CreateUser(ctx, email, hash, "Admin", RoleAdmin)
	if err != nil {
		return fmt.Errorf("auth: bootstrap admin: %w", err)
	}
	logger.Info("bootstrap admin created", "user_id", u.ID, "email", u.Email)
	return nil
}
