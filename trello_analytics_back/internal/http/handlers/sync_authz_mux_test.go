package handlers_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"trello_analytics_back/internal/auth"
	"trello_analytics_back/internal/http/handlers"
	"trello_analytics_back/internal/http/middleware"
)

// Table-driven mux coverage: sync start Member→403 vs Admin (auth injected via context).
func TestSyncStart_MuxAuthz_MemberVsAdmin(t *testing.T) {
	syncH := &handlers.Sync{} // Orch nil → Admin path yields 503 after authz passes

	chain := func(minRole string, h http.HandlerFunc) http.Handler {
		return middleware.RequireRole(minRole)(h)
	}

	mux := http.NewServeMux()
	mux.Handle("POST /api/v1/sync/start", chain(auth.RoleAdmin, syncH.Start))
	mux.Handle("GET /api/v1/sync/status", chain(auth.RoleMember, syncH.Status))

	// Inject authenticated user without DB session (practical fake auth context).
	withRole := func(r *http.Request, role string) *http.Request {
		return r.WithContext(auth.WithUser(r.Context(), auth.User{
			ID: "u-test", Email: "t@example.com", Role: role, IsActive: true,
		}))
	}

	cases := []struct {
		name   string
		method string
		path   string
		role   string
		status int
	}{
		{"member_start_forbidden", http.MethodPost, "/api/v1/sync/start", auth.RoleMember, http.StatusForbidden},
		{"lead_start_forbidden", http.MethodPost, "/api/v1/sync/start", auth.RoleTeamLead, http.StatusForbidden},
		{"admin_start_passes_authz", http.MethodPost, "/api/v1/sync/start", auth.RoleAdmin, http.StatusServiceUnavailable},
		{"member_status_passes_authz", http.MethodGet, "/api/v1/sync/status", auth.RoleMember, http.StatusServiceUnavailable},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(tc.method, tc.path, nil)
			req = withRole(req, tc.role)
			rec := httptest.NewRecorder()
			mux.ServeHTTP(rec, req)
			if rec.Code != tc.status {
				t.Fatalf("status=%d want %d body=%s", rec.Code, tc.status, rec.Body.String())
			}
		})
	}
}
