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

// Lists handles Trello column list / is_active updates.
type Lists struct {
	Store  *postgres.Store
	Logger *slog.Logger
}

type listJSON struct {
	ID           string  `json:"id"`
	TrelloListID string  `json:"trello_list_id"`
	Name         string  `json:"name"`
	IsArchived   bool    `json:"is_archived"`
	IsActive     bool    `json:"is_active"`
	LogicalStage *string `json:"logical_stage"`
}

type patchListRequest struct {
	IsActive *bool `json:"is_active"`
}

func toListJSON(l postgres.BoardList) listJSON {
	return listJSON{
		ID:           l.ID,
		TrelloListID: l.TrelloListID,
		Name:         l.Name,
		IsArchived:   l.IsArchived,
		IsActive:     l.IsActive,
		LogicalStage: l.LogicalStage,
	}
}

// List returns all imported Trello lists.
func (h *Lists) List(w http.ResponseWriter, r *http.Request) {
	if h == nil || h.Store == nil {
		writeError(w, http.StatusServiceUnavailable, "store_unavailable", "store is not configured")
		return
	}
	items, err := h.Store.ListBoardLists(r.Context())
	if err != nil {
		h.warn(err)
		WriteInternal(w)
		return
	}
	out := make([]listJSON, 0, len(items))
	for _, l := range items {
		out = append(out, toListJSON(l))
	}
	writeJSON(w, http.StatusOK, map[string]any{"lists": out})
}

// Patch updates is_active for a Trello list (Team Lead+ via middleware).
func (h *Lists) Patch(w http.ResponseWriter, r *http.Request) {
	if h == nil || h.Store == nil {
		writeError(w, http.StatusServiceUnavailable, "store_unavailable", "store is not configured")
		return
	}
	id := strings.TrimSpace(r.PathValue("id"))
	if id == "" {
		writeBadRequest(w, "invalid_input", "list id is required")
		return
	}
	var req patchListRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeBadRequest(w, "invalid_body", "invalid JSON body")
		return
	}
	if req.IsActive == nil {
		writeBadRequest(w, "invalid_input", "is_active is required")
		return
	}
	l, err := h.Store.SetListActive(r.Context(), id, *req.IsActive)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) || strings.Contains(err.Error(), "list not found") {
			writeNotFound(w, "list not found")
			return
		}
		h.warn(err)
		WriteInternal(w)
		return
	}
	writeJSON(w, http.StatusOK, toListJSON(l))
}

func (h *Lists) warn(err error) {
	if h.Logger != nil {
		h.Logger.Warn("lists handler", "err", err)
	}
}
