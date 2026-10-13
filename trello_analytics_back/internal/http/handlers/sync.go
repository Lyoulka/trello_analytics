package handlers

import (
	"errors"
	"log/slog"
	"net/http"
	"strconv"

	"trello_analytics_back/internal/storage/postgres"
	"trello_analytics_back/internal/syncer"
)

// Sync binds HTTP sync endpoints to the Orchestrator (dual entrypoint with CLI).
type Sync struct {
	Orch   *syncer.Orchestrator
	Store  *postgres.Store
	Logger *slog.Logger
}

// Start handles POST /api/v1/sync/start → 202 {run_id} or 409 already running.
func (h *Sync) Start(w http.ResponseWriter, r *http.Request) {
	if h.Orch == nil {
		writeError(w, http.StatusServiceUnavailable, "sync_unavailable", "sync orchestrator is not configured")
		return
	}
	runID, err := h.Orch.Start(r.Context())
	if err != nil {
		var already *syncer.AlreadyRunningError
		if errors.As(err, &already) {
			writeJSON(w, http.StatusConflict, map[string]any{
				"error": map[string]string{
					"code":    "already_running",
					"message": "sync already running",
				},
				"run_id": already.RunID,
			})
			return
		}
		if h.Logger != nil {
			h.Logger.Warn("sync start", "err", err)
		}
		WriteInternal(w)
		return
	}
	writeJSON(w, http.StatusAccepted, map[string]string{"run_id": runID})
}

// Status handles GET /api/v1/sync/status (Member+).
func (h *Sync) Status(w http.ResponseWriter, r *http.Request) {
	if h.Orch == nil {
		writeError(w, http.StatusServiceUnavailable, "sync_unavailable", "sync orchestrator is not configured")
		return
	}
	st, err := h.Orch.Status(r.Context())
	if err != nil {
		if h.Logger != nil {
			h.Logger.Warn("sync status", "err", err)
		}
		WriteInternal(w)
		return
	}
	writeJSON(w, http.StatusOK, st)
}

// History handles GET /api/v1/sync/history (Team Lead+).
func (h *Sync) History(w http.ResponseWriter, r *http.Request) {
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	runs, err := h.Store.ListSyncRuns(r.Context(), limit)
	if err != nil {
		if h.Logger != nil {
			h.Logger.Warn("sync history", "err", err)
		}
		WriteInternal(w)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"runs": runs})
}

// Retry handles POST /api/v1/sync/retry (Admin) — same as Start.
func (h *Sync) Retry(w http.ResponseWriter, r *http.Request) {
	h.Start(w, r)
}
