package handlers_test

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	"trello_analytics_back/internal/auth"
	"trello_analytics_back/internal/http/handlers"
	"trello_analytics_back/internal/http/middleware"
)

func TestListsPatch_MuxAuthz_MemberVsTeamLead(t *testing.T) {
	listsH := &handlers.Lists{}

	chain := func(minRole string, h http.HandlerFunc) http.Handler {
		return middleware.RequireRole(minRole)(h)
	}

	mux := http.NewServeMux()
	mux.Handle("GET /api/v1/lists", chain(auth.RoleMember, listsH.List))
	mux.Handle("PATCH /api/v1/lists/{id}", chain(auth.RoleTeamLead, listsH.Patch))

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
		body   string
		status int
	}{
		{"member_patch_forbidden", http.MethodPatch, "/api/v1/lists/00000000-0000-0000-0000-000000000001", auth.RoleMember, `{"is_active":false}`, http.StatusForbidden},
		{"lead_patch_passes_authz", http.MethodPatch, "/api/v1/lists/00000000-0000-0000-0000-000000000001", auth.RoleTeamLead, `{"is_active":false}`, http.StatusServiceUnavailable},
		{"member_list_passes_authz", http.MethodGet, "/api/v1/lists", auth.RoleMember, "", http.StatusServiceUnavailable},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var req *http.Request
			if tc.body != "" {
				req = httptest.NewRequest(tc.method, tc.path, bytes.NewReader([]byte(tc.body)))
			} else {
				req = httptest.NewRequest(tc.method, tc.path, nil)
			}
			req = withRole(req, tc.role)
			rec := httptest.NewRecorder()
			mux.ServeHTTP(rec, req)
			if rec.Code != tc.status {
				t.Fatalf("status=%d want %d body=%s", rec.Code, tc.status, rec.Body.String())
			}
		})
	}
}
