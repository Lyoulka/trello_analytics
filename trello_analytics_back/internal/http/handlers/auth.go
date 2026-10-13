package handlers

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"time"

	"trello_analytics_back/internal/auth"
	"trello_analytics_back/internal/config"
	"trello_analytics_back/internal/storage/postgres"
)

// Auth handles login/logout/me.
type Auth struct {
	Store  *postgres.Store
	Cfg    config.Config
	Logger *slog.Logger
}

type loginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type meResponse struct {
	ID          string `json:"id"`
	Email       string `json:"email"`
	DisplayName string `json:"display_name"`
	Role        string `json:"role"`
	IsActive    bool   `json:"is_active"`
}

// Login authenticates and sets the session cookie.
func (h *Auth) Login(w http.ResponseWriter, r *http.Request) {
	var req loginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeBadRequest(w, "invalid_body", "invalid JSON body")
		return
	}
	// Generic failure — do not reveal whether email exists.
	fail := func() {
		writeError(w, http.StatusUnauthorized, "invalid_credentials", "invalid email or password")
	}
	u, err := h.Store.GetUserByEmail(r.Context(), req.Email)
	if err != nil {
		h.logErr("login lookup", err)
		WriteInternal(w)
		return
	}
	// Always run bcrypt (dummy hash when user missing/inactive) so timing does not enumerate emails.
	hash := auth.DummyPasswordHash()
	if u != nil && u.IsActive {
		hash = u.PasswordHash
	}
	passwordOK := auth.CheckPassword(hash, req.Password)
	if u == nil || !u.IsActive || !passwordOK {
		fail()
		return
	}
	raw, err := auth.NewSessionToken()
	if err != nil {
		h.logErr("session token", err)
		WriteInternal(w)
		return
	}
	ttl := h.Cfg.SessionTTL
	if ttl <= 0 {
		ttl = 7 * 24 * time.Hour
	}
	expires := time.Now().UTC().Add(ttl)
	if _, err := h.Store.CreateSession(r.Context(), u.ID, auth.HashToken(raw), expires); err != nil {
		h.logErr("create session", err)
		WriteInternal(w)
		return
	}
	http.SetCookie(w, &http.Cookie{
		Name:     auth.CookieName,
		Value:    raw,
		Path:     "/",
		Expires:  expires,
		MaxAge:   int(ttl.Seconds()),
		HttpOnly: true,
		Secure:   h.Cfg.AppEnv != "development",
		SameSite: http.SameSiteLaxMode,
	})
	writeJSON(w, http.StatusOK, meResponse{
		ID: u.ID, Email: u.Email, DisplayName: u.DisplayName, Role: u.Role, IsActive: u.IsActive,
	})
}

// Logout revokes the current session cookie.
func (h *Auth) Logout(w http.ResponseWriter, r *http.Request) {
	if c, err := r.Cookie(auth.CookieName); err == nil && c.Value != "" {
		_ = h.Store.RevokeSessionByTokenHash(r.Context(), auth.HashToken(c.Value))
	}
	http.SetCookie(w, &http.Cookie{
		Name:     auth.CookieName,
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   h.Cfg.AppEnv != "development",
		SameSite: http.SameSiteLaxMode,
	})
	w.WriteHeader(http.StatusNoContent)
}

// Me returns the authenticated user.
func (h *Auth) Me(w http.ResponseWriter, r *http.Request) {
	u, ok := auth.UserFromContext(r.Context())
	if !ok {
		WriteUnauthorized(w)
		return
	}
	writeJSON(w, http.StatusOK, meResponse{
		ID: u.ID, Email: u.Email, DisplayName: u.DisplayName, Role: u.Role, IsActive: u.IsActive,
	})
}

func (h *Auth) logErr(msg string, err error) {
	if h.Logger != nil {
		h.Logger.Warn(msg, "err", err)
	}
}
