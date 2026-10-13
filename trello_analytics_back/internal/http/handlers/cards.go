package handlers

import (
	"log/slog"
	"net/http"
	"strconv"
	"strings"

	"trello_analytics_back/internal/config"
	"trello_analytics_back/internal/storage/postgres"
)

// Cards serves list/detail/history.
type Cards struct {
	Store  *postgres.Store
	Cfg    config.Config
	Logger *slog.Logger
}

// List handles GET /api/v1/cards.
func (h *Cards) List(w http.ResponseWriter, r *http.Request) {
	boardID, err := resolveBoardID(r.Context(), h.Store, h.Cfg.TrelloBoardID)
	if err != nil {
		writeBoardResolveError(w, err)
		return
	}
	q := r.URL.Query()
	limit, _ := strconv.Atoi(q.Get("limit"))
	offset, _ := strconv.Atoi(q.Get("offset"))
	f := postgres.CardListFilter{
		BoardID:  boardID,
		MemberID: q.Get("member_id"),
		LabelID:  q.Get("label_id"),
		ListID:   q.Get("list_id"),
		Search:   q.Get("q"),
		Sort:     q.Get("sort"),
		Order:    q.Get("order"),
		Limit:    limit,
		Offset:   offset,
	}
	if a := q.Get("archived"); a != "" {
		v := a == "true" || a == "1"
		f.Archived = &v
	}
	if fromS := q.Get("updated_from"); fromS != "" {
		t, err := parseTimeBound(fromS, false)
		if err != nil {
			writeBadRequest(w, "invalid_query", "invalid updated_from")
			return
		}
		f.UpdatedFrom = &t
	}
	if toS := q.Get("updated_to"); toS != "" {
		t, err := parseTimeBound(toS, true)
		if err != nil {
			writeBadRequest(w, "invalid_query", "invalid updated_to")
			return
		}
		f.UpdatedTo = &t
	}
	if f.UpdatedFrom != nil && f.UpdatedTo != nil && !f.UpdatedTo.After(*f.UpdatedFrom) {
		writeBadRequest(w, "invalid_query", "updated_to must be after updated_from")
		return
	}
	items, total, err := h.Store.ListCards(r.Context(), f)
	if err != nil {
		if strings.Contains(err.Error(), "invalid sort") {
			writeBadRequest(w, "invalid_query", "sort must be name, updated, or created")
			return
		}
		h.warn(err)
		WriteInternal(w)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"items":  items,
		"total":  total,
		"limit":  f.Limit,
		"offset": f.Offset,
	})
}

// Detail handles GET /api/v1/cards/{id}.
func (h *Cards) Detail(w http.ResponseWriter, r *http.Request) {
	boardID, err := resolveBoardID(r.Context(), h.Store, h.Cfg.TrelloBoardID)
	if err != nil {
		writeBoardResolveError(w, err)
		return
	}
	id := r.PathValue("id")
	d, err := h.Store.GetCardDetail(r.Context(), boardID, id)
	if err != nil {
		h.warn(err)
		WriteInternal(w)
		return
	}
	if d == nil {
		writeNotFound(w, "card not found")
		return
	}
	writeJSON(w, http.StatusOK, d)
}

// History handles GET /api/v1/cards/{id}/history (normalized, no raw_payload).
func (h *Cards) History(w http.ResponseWriter, r *http.Request) {
	boardID, err := resolveBoardID(r.Context(), h.Store, h.Cfg.TrelloBoardID)
	if err != nil {
		writeBoardResolveError(w, err)
		return
	}
	id := r.PathValue("id")
	items, err := h.Store.ListCardHistory(r.Context(), boardID, id)
	if err != nil {
		h.warn(err)
		WriteInternal(w)
		return
	}
	if items == nil {
		writeNotFound(w, "card not found")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"events": items})
}

func (h *Cards) warn(err error) {
	if h.Logger != nil {
		h.Logger.Warn("cards handler", "err", err)
	}
}
