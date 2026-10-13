package handlers

import (
	"log/slog"
	"net/http"

	"trello_analytics_back/internal/analytics"
	"trello_analytics_back/internal/config"
	"trello_analytics_back/internal/storage/postgres"
)

// Analytics serves GET /api/v1/analytics/* via Engine.Compute.
type Analytics struct {
	Store  *postgres.Store
	Engine *analytics.Engine
	Cfg    config.Config
	Logger *slog.Logger
}

func (h *Analytics) compute(w http.ResponseWriter, r *http.Request) (analytics.Result, bool) {
	if h.Engine == nil {
		writeError(w, http.StatusServiceUnavailable, "analytics_unavailable", "analytics engine is not configured")
		return analytics.Result{}, false
	}
	q, err := parsePeriodQuery(r)
	if err != nil {
		writeBadRequest(w, "invalid_query", "invalid from/to or filter query")
		return analytics.Result{}, false
	}
	boardID, err := resolveBoardID(r.Context(), h.Store, h.Cfg.TrelloBoardID)
	if err != nil {
		writeBoardResolveError(w, err)
		return analytics.Result{}, false
	}
	res, err := h.Engine.Compute(r.Context(), boardID, q)
	if err != nil {
		if h.Logger != nil {
			h.Logger.Warn("analytics compute", "err", err)
		}
		WriteInternal(w)
		return analytics.Result{}, false
	}
	return res, true
}

// Overview returns a summary projection.
func (h *Analytics) Overview(w http.ResponseWriter, r *http.Request) {
	res, ok := h.compute(w, r)
	if !ok {
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"completed":    res.Completed,
		"throughput":   res.Throughput,
		"lead_time":    res.LeadTime,
		"cycle_time":   res.CycleTime,
		"bug_reopen":   res.BugReopen,
		"fix_time":     res.FixTime,
		"wip":          res.WIP,
		"stale":        res.Stale,
		"workload":     res.Workload,
		"warnings":     res.Warnings,
		"needs_review": res.NeedsReview,
	})
}

func (h *Analytics) Throughput(w http.ResponseWriter, r *http.Request) {
	res, ok := h.compute(w, r)
	if !ok {
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"throughput": res.Throughput, "warnings": filterWarnings(res.Warnings, "throughput", "completed", "data_quality", "workflow_incomplete", "missing_done_stages")})
}

func (h *Analytics) LeadTime(w http.ResponseWriter, r *http.Request) {
	res, ok := h.compute(w, r)
	if !ok {
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"lead_time": res.LeadTime, "warnings": filterWarnings(res.Warnings, "lead_time", "data_quality", "lead_rule_unset")})
}

func (h *Analytics) CycleTime(w http.ResponseWriter, r *http.Request) {
	res, ok := h.compute(w, r)
	if !ok {
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"cycle_time": res.CycleTime, "warnings": filterWarnings(res.Warnings, "cycle_time", "data_quality", "missing_cycle_start")})
}

func (h *Analytics) Bugs(w http.ResponseWriter, r *http.Request) {
	res, ok := h.compute(w, r)
	if !ok {
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"completed_bugs": res.BugReopen.CompletedBugs,
		"available":      res.BugReopen.Available,
		"warnings":       filterWarnings(res.Warnings, "bug_reopen", "data_quality", "missing_work_stages"),
	})
}

func (h *Analytics) ReopenedBugs(w http.ResponseWriter, r *http.Request) {
	res, ok := h.compute(w, r)
	if !ok {
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"bug_reopen": res.BugReopen, "warnings": filterWarnings(res.Warnings, "bug_reopen", "data_quality", "ambiguous_reopen")})
}

func (h *Analytics) FixTime(w http.ResponseWriter, r *http.Request) {
	res, ok := h.compute(w, r)
	if !ok {
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"fix_time": res.FixTime, "warnings": filterWarnings(res.Warnings, "bug_reopen", "data_quality")})
}

func (h *Analytics) Workload(w http.ResponseWriter, r *http.Request) {
	res, ok := h.compute(w, r)
	if !ok {
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"workload": res.Workload, "warnings": res.Warnings})
}

func (h *Analytics) StaleTasks(w http.ResponseWriter, r *http.Request) {
	res, ok := h.compute(w, r)
	if !ok {
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"stale": res.Stale, "wip": res.WIP, "warnings": filterWarnings(res.Warnings, "stale", "data_quality", "incomplete_history")})
}

func filterWarnings(all []analytics.Warning, metrics ...string) []analytics.Warning {
	allow := map[string]struct{}{}
	for _, m := range metrics {
		allow[m] = struct{}{}
	}
	var out []analytics.Warning
	for _, w := range all {
		if _, ok := allow[w.Metric]; ok || w.Metric == "" {
			out = append(out, w)
			continue
		}
		if _, ok := allow[w.Code]; ok {
			out = append(out, w)
		}
	}
	if out == nil {
		out = []analytics.Warning{}
	}
	return out
}
