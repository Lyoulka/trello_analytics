package handlers

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strings"

	"github.com/jackc/pgx/v5"

	"trello_analytics_back/internal/storage/postgres"
)

// Members handles Trello board participant list / is_active updates.
type Members struct {
	Store  *postgres.Store
	Logger *slog.Logger
}

type memberJSON struct {
	ID             string `json:"id"`
	TrelloMemberID string `json:"trello_member_id"`
	FullName       string `json:"full_name"`
	Username       string `json:"username"`
	IsActive       bool   `json:"is_active"`
}

type patchMemberRequest struct {
	IsActive *bool `json:"is_active"`
}

func toMemberJSON(m postgres.BoardMember) memberJSON {
	return memberJSON{
		ID:             m.ID,
		TrelloMemberID: m.TrelloMemberID,
		FullName:       m.FullName,
		Username:       m.Username,
		IsActive:       m.IsActive,
	}
}

// List returns all imported Trello members.
func (h *Members) List(w http.ResponseWriter, r *http.Request) {
	if h == nil || h.Store == nil {
		writeError(w, http.StatusServiceUnavailable, "store_unavailable", "store is not configured")
		return
	}
	list, err := h.Store.ListMembers(r.Context())
	if err != nil {
		h.warn(err)
		WriteInternal(w)
		return
	}
	out := make([]memberJSON, 0, len(list))
	for _, m := range list {
		out = append(out, toMemberJSON(m))
	}
	writeJSON(w, http.StatusOK, map[string]any{"members": out})
}

// Patch updates is_active for a Trello member (Team Lead+ via middleware).
func (h *Members) Patch(w http.ResponseWriter, r *http.Request) {
	if h == nil || h.Store == nil {
		writeError(w, http.StatusServiceUnavailable, "store_unavailable", "store is not configured")
		return
	}
	id := strings.TrimSpace(r.PathValue("id"))
	if id == "" {
		writeBadRequest(w, "invalid_input", "member id is required")
		return
	}
	var req patchMemberRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeBadRequest(w, "invalid_body", "invalid JSON body")
		return
	}
	if req.IsActive == nil {
		writeBadRequest(w, "invalid_input", "is_active is required")
		return
	}
	m, err := h.Store.SetMemberActive(r.Context(), id, *req.IsActive)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) || strings.Contains(err.Error(), "member not found") {
			writeNotFound(w, "member not found")
			return
		}
		h.warn(err)
		WriteInternal(w)
		return
	}
	writeJSON(w, http.StatusOK, toMemberJSON(m))
}

func (h *Members) warn(err error) {
	if h.Logger != nil {
		h.Logger.Warn("members handler", "err", err)
	}
}
