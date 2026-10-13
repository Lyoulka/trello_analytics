package http

import (
	"log/slog"
	"net/http"
	"time"

	"trello_analytics_back/internal/analytics"
	"trello_analytics_back/internal/auth"
	"trello_analytics_back/internal/config"
	"trello_analytics_back/internal/http/handlers"
	"trello_analytics_back/internal/http/middleware"
	"trello_analytics_back/internal/storage/postgres"
	"trello_analytics_back/internal/syncer"
	"trello_analytics_back/internal/workflow"
)

// Deps wires HTTP server dependencies.
type Deps struct {
	Cfg      config.Config
	Store    *postgres.Store
	Logger   *slog.Logger
	Engine   *analytics.Engine
	Workflow *workflow.Service
	Sync     *syncer.Orchestrator // optional; nil → sync endpoints return 503
}

// NewServer builds the HTTP server with auth, analytics, cards, reports, settings, and sync routes.
func NewServer(d Deps) *http.Server {
	if d.Logger == nil {
		d.Logger = slog.Default()
	}
	mux := http.NewServeMux()

	health := handlers.NewHealth(d.Store.Pool(), d.Logger)
	mux.HandleFunc("GET /health", health.Liveness)
	mux.HandleFunc("GET /ready", health.Readiness)
	mux.HandleFunc("GET /api/v1/health", health.APIHealth)

	authH := &handlers.Auth{Store: d.Store, Cfg: d.Cfg, Logger: d.Logger}
	usersH := &handlers.Users{Store: d.Store, Logger: d.Logger}
	membersH := &handlers.Members{Store: d.Store, Logger: d.Logger}
	listsH := &handlers.Lists{Store: d.Store, Logger: d.Logger}
	analyticsH := &handlers.Analytics{Store: d.Store, Engine: d.Engine, Cfg: d.Cfg, Logger: d.Logger}
	cardsH := &handlers.Cards{Store: d.Store, Cfg: d.Cfg, Logger: d.Logger}
	reportsH := &handlers.Reports{Store: d.Store, Engine: d.Engine, Cfg: d.Cfg, Logger: d.Logger}
	settingsH := &handlers.Settings{Store: d.Store, Workflow: d.Workflow, Cfg: d.Cfg, Logger: d.Logger}
	syncH := &handlers.Sync{Orch: d.Sync, Store: d.Store, Logger: d.Logger}

	authenticate := middleware.Authenticate(d.Store, d.Logger)

	mux.HandleFunc("POST /api/v1/auth/login", authH.Login)

	protected := func(minRole string, h http.HandlerFunc) http.Handler {
		return chain(h, authenticate, middleware.RequireRole(minRole))
	}

	mux.Handle("POST /api/v1/auth/logout", protected(auth.RoleMember, authH.Logout))
	mux.Handle("GET /api/v1/auth/me", protected(auth.RoleMember, authH.Me))

	mux.Handle("GET /api/v1/users", protected(auth.RoleAdmin, usersH.List))
	mux.Handle("POST /api/v1/users", protected(auth.RoleAdmin, usersH.Create))
	mux.Handle("PATCH /api/v1/users/{id}", protected(auth.RoleAdmin, usersH.Patch))

	mux.Handle("GET /api/v1/members", protected(auth.RoleMember, membersH.List))
	mux.Handle("PATCH /api/v1/members/{id}", protected(auth.RoleTeamLead, membersH.Patch))

	mux.Handle("GET /api/v1/lists", protected(auth.RoleMember, listsH.List))
	mux.Handle("PATCH /api/v1/lists/{id}", protected(auth.RoleTeamLead, listsH.Patch))

	mux.Handle("GET /api/v1/analytics/overview", protected(auth.RoleMember, analyticsH.Overview))
	mux.Handle("GET /api/v1/analytics/throughput", protected(auth.RoleMember, analyticsH.Throughput))
	mux.Handle("GET /api/v1/analytics/lead-time", protected(auth.RoleMember, analyticsH.LeadTime))
	mux.Handle("GET /api/v1/analytics/cycle-time", protected(auth.RoleMember, analyticsH.CycleTime))
	mux.Handle("GET /api/v1/analytics/bugs", protected(auth.RoleMember, analyticsH.Bugs))
	mux.Handle("GET /api/v1/analytics/reopened-bugs", protected(auth.RoleMember, analyticsH.ReopenedBugs))
	mux.Handle("GET /api/v1/analytics/reopened-bugs/fix-time", protected(auth.RoleMember, analyticsH.FixTime))
	mux.Handle("GET /api/v1/analytics/workload", protected(auth.RoleMember, analyticsH.Workload))
	mux.Handle("GET /api/v1/analytics/stale-tasks", protected(auth.RoleMember, analyticsH.StaleTasks))

	mux.Handle("GET /api/v1/cards", protected(auth.RoleMember, cardsH.List))
	mux.Handle("GET /api/v1/cards/{id}", protected(auth.RoleMember, cardsH.Detail))
	mux.Handle("GET /api/v1/cards/{id}/history", protected(auth.RoleMember, cardsH.History))

	mux.Handle("GET /api/v1/reports/daily", protected(auth.RoleMember, reportsH.Daily))
	mux.Handle("GET /api/v1/reports/period", protected(auth.RoleMember, reportsH.Period))
	mux.Handle("GET /api/v1/reports/preview", protected(auth.RoleMember, reportsH.Preview))

	mux.Handle("GET /api/v1/settings", protected(auth.RoleMember, settingsH.Get))
	mux.Handle("PATCH /api/v1/settings", protected(auth.RoleTeamLead, settingsH.Patch))
	mux.Handle("GET /api/v1/settings/workflow", protected(auth.RoleMember, settingsH.GetWorkflow))
	mux.Handle("PUT /api/v1/settings/workflow", protected(auth.RoleTeamLead, settingsH.PutWorkflow))

	mux.Handle("POST /api/v1/sync/start", protected(auth.RoleAdmin, syncH.Start))
	mux.Handle("GET /api/v1/sync/status", protected(auth.RoleMember, syncH.Status))
	mux.Handle("GET /api/v1/sync/history", protected(auth.RoleTeamLead, syncH.History))
	mux.Handle("POST /api/v1/sync/retry", protected(auth.RoleAdmin, syncH.Retry))

	handler := middleware.CORS(d.Cfg.FrontendOrigin)(mux)

	return &http.Server{
		Addr:              d.Cfg.HTTPAddr,
		Handler:           handler,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       30 * time.Second,
		WriteTimeout:      60 * time.Second,
		IdleTimeout:       60 * time.Second,
	}
}

func chain(h http.Handler, mws ...func(http.Handler) http.Handler) http.Handler {
	for i := len(mws) - 1; i >= 0; i-- {
		h = mws[i](h)
	}
	return h
}
