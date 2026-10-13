package handlers_test

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"trello_analytics_back/internal/auth"
	"trello_analytics_back/internal/config"
	apphttp "trello_analytics_back/internal/http"
	"trello_analytics_back/internal/http/middleware"
	"trello_analytics_back/internal/storage/postgres"
)

func TestRequireRole_Matrix(t *testing.T) {
	cases := []struct {
		have   string
		need   string
		status int
	}{
		{auth.RoleMember, auth.RoleAdmin, http.StatusForbidden},
		{auth.RoleAdmin, auth.RoleAdmin, http.StatusOK},
		{auth.RoleTeamLead, auth.RoleTeamLead, http.StatusOK},
		{auth.RoleMember, auth.RoleMember, http.StatusOK},
		{auth.RoleTeamLead, auth.RoleAdmin, http.StatusForbidden},
	}
	for _, tc := range cases {
		h := middleware.RequireRole(tc.need)(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusOK)
		}))
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		ctx := auth.WithUser(req.Context(), auth.User{ID: "1", Role: tc.have, IsActive: true})
		req = req.WithContext(ctx)
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)
		if rec.Code != tc.status {
			t.Fatalf("have=%s need=%s: status=%d want %d", tc.have, tc.need, rec.Code, tc.status)
		}
	}
}

func TestLogin_InvalidBody(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	store := postgres.NewStore(nil)
	srv := apphttp.NewServer(apphttp.Deps{
		Cfg:    config.Config{HTTPAddr: ":0", FrontendOrigin: "http://localhost:5173"},
		Store:  store,
		Logger: logger,
	})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewReader([]byte(`{`)))
	rec := httptest.NewRecorder()
	srv.Handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status=%d", rec.Code)
	}
	var env map[string]any
	if err := json.NewDecoder(rec.Body).Decode(&env); err != nil {
		t.Fatal(err)
	}
	errObj, _ := env["error"].(map[string]any)
	if errObj["code"] != "invalid_body" {
		t.Fatalf("error=%v", env)
	}
}

func TestErrorEnvelope_NoStack(t *testing.T) {
	rec := httptest.NewRecorder()
	// Use exported helper via unauthorized path on a fake handler.
	h := middleware.Authenticate(postgres.NewStore(nil), nil)(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	h.ServeHTTP(rec, req)
	body := rec.Body.String()
	if bytes.Contains([]byte(body), []byte("goroutine")) || bytes.Contains([]byte(body), []byte("SELECT")) {
		t.Fatalf("leaked internals: %s", body)
	}
}
