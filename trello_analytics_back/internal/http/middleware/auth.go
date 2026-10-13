package middleware

import (
	"log/slog"
	"net/http"

	"trello_analytics_back/internal/auth"
	"trello_analytics_back/internal/http/handlers"
	"trello_analytics_back/internal/storage/postgres"
)

// Authenticate loads the session from cookie and attaches the user to context.
// Missing/invalid session → 401.
func Authenticate(store *postgres.Store, logger *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			c, err := r.Cookie(auth.CookieName)
			if err != nil || c.Value == "" {
				handlers.WriteUnauthorized(w)
				return
			}
			hash := auth.HashToken(c.Value)
			su, err := store.GetActiveSessionByTokenHash(r.Context(), hash)
			if err != nil {
				if logger != nil {
					logger.Warn("session lookup failed", "err", err)
				}
				handlers.WriteInternal(w)
				return
			}
			if su == nil {
				handlers.WriteUnauthorized(w)
				return
			}
			ctx := auth.WithUser(r.Context(), su.User)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// RequireRole rejects requests whose user role is below minRole (403).
func RequireRole(minRole string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			u, ok := auth.UserFromContext(r.Context())
			if !ok {
				handlers.WriteUnauthorized(w)
				return
			}
			if !auth.RoleAtLeast(u.Role, minRole) {
				handlers.WriteForbidden(w)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
