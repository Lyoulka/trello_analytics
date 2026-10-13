package auth

import "context"

type ctxKey int

const userKey ctxKey = 1

// User is the authenticated principal (no password hash).
type User struct {
	ID          string
	Email       string
	DisplayName string
	Role        string
	IsActive    bool
}

// WithUser stores the authenticated user on the request context.
func WithUser(ctx context.Context, u User) context.Context {
	return context.WithValue(ctx, userKey, u)
}

// UserFromContext returns the authenticated user if present.
func UserFromContext(ctx context.Context) (User, bool) {
	u, ok := ctx.Value(userKey).(User)
	return u, ok
}
