package handlers

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"trello_analytics_back/internal/storage/postgres"
)

// Health serves liveness/readiness endpoints.
type Health struct {
	db     *pgxpool.Pool
	logger *slog.Logger
}

func NewHealth(db *pgxpool.Pool, logger *slog.Logger) *Health {
	return &Health{db: db, logger: logger}
}

type healthResponse struct {
	Status   string `json:"status"`
	Database string `json:"database,omitempty"`
}

// Liveness confirms the process is up (no DB check).
func (h *Health) Liveness(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, healthResponse{Status: "ok"})
}

// Readiness confirms PostgreSQL is reachable.
func (h *Health) Readiness(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
	defer cancel()

	if err := postgres.Ping(ctx, h.db); err != nil {
		h.logger.Warn("readiness check failed", "err", err)
		writeJSON(w, http.StatusServiceUnavailable, healthResponse{
			Status:   "unavailable",
			Database: "down",
		})
		return
	}

	writeJSON(w, http.StatusOK, healthResponse{
		Status:   "ok",
		Database: "up",
	})
}

// APIHealth is the versioned health probe for frontend / proxy checks.
func (h *Health) APIHealth(w http.ResponseWriter, r *http.Request) {
	h.Readiness(w, r)
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}
