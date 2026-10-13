package http_test

import (
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"trello_analytics_back/internal/config"
	apphttp "trello_analytics_back/internal/http"
	"trello_analytics_back/internal/storage/postgres"
)

func TestNewServer_HealthAndAuthRequired(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	// Pool may be nil for route registration tests that never hit DB-backed handlers.
	store := postgres.NewStore(nil)
	srv := apphttp.NewServer(apphttp.Deps{
		Cfg:    config.Config{HTTPAddr: ":0", FrontendOrigin: "http://localhost:5173"},
		Store:  store,
		Logger: logger,
	})

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rec := httptest.NewRecorder()
	srv.Handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("GET /health status=%d", rec.Code)
	}

	// Protected routes require auth → 401 without cookie.
	req = httptest.NewRequest(http.MethodGet, "/api/v1/auth/me", nil)
	rec = httptest.NewRecorder()
	srv.Handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("GET /api/v1/auth/me without cookie: status=%d want 401", rec.Code)
	}

	// Sync routes exist (auth required).
	req = httptest.NewRequest(http.MethodPost, "/api/v1/sync/start", nil)
	rec = httptest.NewRecorder()
	srv.Handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("POST /api/v1/sync/start without cookie: status=%d want 401", rec.Code)
	}

	// CORS should not be *
	req = httptest.NewRequest(http.MethodOptions, "/api/v1/auth/login", nil)
	rec = httptest.NewRecorder()
	srv.Handler.ServeHTTP(rec, req)
	if got := rec.Header().Get("Access-Control-Allow-Origin"); got != "http://localhost:5173" {
		t.Fatalf("CORS origin=%q", got)
	}
	if got := rec.Header().Get("Access-Control-Allow-Credentials"); got != "true" {
		t.Fatalf("CORS credentials=%q", got)
	}
}
