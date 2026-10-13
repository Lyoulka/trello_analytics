package handlers_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"trello_analytics_back/internal/auth"
	"trello_analytics_back/internal/http/middleware"
)

// Authz matrix for sync endpoints (task 1.9): role gate only (DB session mocked via context).
func TestSyncAuthz_RoleMatrix(t *testing.T) {
	cases := []struct {
		name   string
		role   string
		need   string
		status int
	}{
		{"member_cannot_start", auth.RoleMember, auth.RoleAdmin, http.StatusForbidden},
		{"lead_cannot_start", auth.RoleTeamLead, auth.RoleAdmin, http.StatusForbidden},
		{"admin_can_start", auth.RoleAdmin, auth.RoleAdmin, http.StatusOK},
		{"member_can_status", auth.RoleMember, auth.RoleMember, http.StatusOK},
		{"member_cannot_history", auth.RoleMember, auth.RoleTeamLead, http.StatusForbidden},
		{"lead_can_history", auth.RoleTeamLead, auth.RoleTeamLead, http.StatusOK},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			h := middleware.RequireRole(tc.need)(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(http.StatusOK)
			}))
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			req = req.WithContext(auth.WithUser(req.Context(), auth.User{ID: "u1", Role: tc.role, IsActive: true}))
			rec := httptest.NewRecorder()
			h.ServeHTTP(rec, req)
			if rec.Code != tc.status {
				t.Fatalf("status=%d want %d", rec.Code, tc.status)
			}
		})
	}
}
