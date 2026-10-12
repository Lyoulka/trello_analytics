package http

import (
	"log/slog"
	"net/http"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"trello_analytics_back/internal/config"
	"trello_analytics_back/internal/http/handlers"
)

// NewServer builds the HTTP server with health/readiness routes.
func NewServer(cfg config.Config, db *pgxpool.Pool, logger *slog.Logger) *http.Server {
	mux := http.NewServeMux()

	h := handlers.NewHealth(db, logger)
	mux.HandleFunc("GET /health", h.Liveness)
	mux.HandleFunc("GET /ready", h.Readiness)
	mux.HandleFunc("GET /api/v1/health", h.APIHealth)

	return &http.Server{
		Addr:              cfg.HTTPAddr,
		Handler:           withCORS(mux),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      15 * time.Second,
		IdleTimeout:       60 * time.Second,
	}
}

// withCORS allows the Vite dev origin during local development.
func withCORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}
