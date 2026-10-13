package handlers

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"strings"

	"trello_analytics_back/internal/auth"
	"trello_analytics_back/internal/storage/postgres"
)

// Users handles Admin user CRUD (no DELETE — deactivate).
type Users struct {
	Store  *postgres.Store
	Logger *slog.Logger
}

type createUserRequest struct {
	Email       string `json:"email"`
	Password    string `json:"password"`
	DisplayName string `json:"display_name"`
	Role        string `json:"role"`
}

type patchUserRequest struct {
	DisplayName *string `json:"display_name"`
	Role        *string `json:"role"`
	IsActive    *bool   `json:"is_active"`
	Password    *string `json:"password"`
}

type userJSON struct {
	ID          string `json:"id"`
	Email       string `json:"email"`
	DisplayName string `json:"display_name"`
	Role        string `json:"role"`
	IsActive    bool   `json:"is_active"`
}

func toUserJSON(u auth.User) userJSON {
	return userJSON{
		ID: u.ID, Email: u.Email, DisplayName: u.DisplayName, Role: u.Role, IsActive: u.IsActive,
	}
}

// List returns all users.
func (h *Users) List(w http.ResponseWriter, r *http.Request) {
	users, err := h.Store.ListUsers(r.Context())
	if err != nil {
		h.warn(err)
		WriteInternal(w)
		return
	}
	out := make([]userJSON, 0, len(users))
	for _, u := range users {
		out = append(out, toUserJSON(u))
	}
	writeJSON(w, http.StatusOK, map[string]any{"users": out})
}

// Create creates a user (Admin only — enforced by middleware).
func (h *Users) Create(w http.ResponseWriter, r *http.Request) {
	var req createUserRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeBadRequest(w, "invalid_body", "invalid JSON body")
		return
	}
	req.Email = strings.TrimSpace(strings.ToLower(req.Email))
	req.DisplayName = strings.TrimSpace(req.DisplayName)
	if req.Email == "" || req.Password == "" || req.DisplayName == "" || !auth.ValidRole(req.Role) {
		writeBadRequest(w, "invalid_input", "email, password, display_name, and valid role are required")
		return
	}
	if err := auth.ValidatePassword(req.Password); err != nil {
		writeBadRequest(w, "invalid_password", "password must be at least 12 characters")
		return
	}
	hash, err := auth.HashPassword(req.Password)
	if err != nil {
		h.warn(err)
		WriteInternal(w)
		return
	}
	u, err := h.Store.CreateUser(r.Context(), req.Email, hash, req.DisplayName, req.Role)
	if err != nil {
		if strings.Contains(err.Error(), "duplicate") || strings.Contains(err.Error(), "unique") {
			writeBadRequest(w, "email_taken", "email already exists")
			return
		}
		h.warn(err)
		WriteInternal(w)
		return
	}
	writeJSON(w, http.StatusCreated, toUserJSON(u))
}

// Patch updates role / display_name / is_active / optional password.
func (h *Users) Patch(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		writeBadRequest(w, "invalid_id", "user id required")
		return
	}
	var req patchUserRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeBadRequest(w, "invalid_body", "invalid JSON body")
		return
	}
	upd := postgres.UserUpdate{
		DisplayName: req.DisplayName,
		Role:        req.Role,
		IsActive:    req.IsActive,
	}
	passwordChanged := false
	if req.Password != nil {
		if err := auth.ValidatePassword(*req.Password); err != nil {
			writeBadRequest(w, "invalid_password", "password must be at least 12 characters")
			return
		}
		hash, err := auth.HashPassword(*req.Password)
		if err != nil {
			h.warn(err)
			WriteInternal(w)
			return
		}
		upd.PasswordHash = &hash
		passwordChanged = true
	}
	u, err := h.Store.UpdateUser(r.Context(), id, upd)
	if err != nil {
		if strings.Contains(err.Error(), "invalid role") || strings.Contains(err.Error(), "empty display_name") {
			writeBadRequest(w, "invalid_input", "invalid role or display_name")
			return
		}
		h.warn(err)
		WriteInternal(w)
		return
	}
	if u == nil {
		writeNotFound(w, "user not found")
		return
	}
	// Deactivate or password change → revoke all sessions.
	if passwordChanged || (req.IsActive != nil && !*req.IsActive) {
		_ = h.Store.RevokeAllSessionsForUser(r.Context(), id)
	}
	writeJSON(w, http.StatusOK, toUserJSON(*u))
}

func (h *Users) warn(err error) {
	if h.Logger != nil {
		h.Logger.Warn("users handler", "err", err)
	}
}
