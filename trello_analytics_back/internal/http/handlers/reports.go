package handlers

import (
	"log/slog"
	"net/http"
	"time"

	"trello_analytics_back/internal/analytics"
	"trello_analytics_back/internal/config"
	"trello_analytics_back/internal/storage/postgres"
)

// Reports serves daily/period/preview (ТЗ §6.3 shape).
type Reports struct {
	Store  *postgres.Store
	Engine *analytics.Engine
	Cfg    config.Config
	Logger *slog.Logger
}

type reportResponse struct {
	Period    reportPeriod        `json:"period"`
	Summary   reportSummary       `json:"summary"`
	Completed []reportCard        `json:"completed_tasks"`
	Bugs      reportBugs          `json:"bugs"`
	Current   reportCurrent       `json:"current_work"`
	Problems  reportProblems      `json:"problems"`
	Warnings  []analytics.Warning `json:"warnings"`
	Deltas    *reportDeltas       `json:"deltas,omitempty"`
}

type reportPeriod struct {
	From time.Time `json:"from"`
	To   time.Time `json:"to"`
}

// reportSummary uses *int so unavailable Engine metrics encode as JSON null
// (not 0), matching analytics honesty: Available=false must not look like an empty period.
type reportSummary struct {
	CompletedCount *int `json:"completed_count"`
	BugsFixedCount *int `json:"bugs_fixed_count"`
	ReopenedCount  *int `json:"reopened_count"`
	WIPCount       *int `json:"wip_count"`
}

type reportCard struct {
	ID           string   `json:"id"`
	TrelloCardID string   `json:"trello_card_id"`
	Name         string   `json:"name"`
	MemberIDs    []string `json:"member_ids"`
	MemberNames  []string `json:"member_names"`
	LabelIDs     []string `json:"label_ids"`
	IsBug        bool     `json:"is_bug"`
	ListName     *string  `json:"list_name,omitempty"`
}

type reportBugs struct {
	Fixed     []reportCard `json:"fixed"`
	Reopened  []reportCard `json:"reopened"`
	StillOpen []reportCard `json:"still_in_work"`
}

type reportCurrent struct {
	WIP   []reportCard `json:"wip"`
	Stale []reportCard `json:"stale"`
}

type reportProblems struct {
	StaleCardIDs     []string                    `json:"stale_card_ids"`
	ReopenedCardIDs  []string                    `json:"reopened_card_ids"`
	NeedsReview      []analytics.NeedsReviewItem `json:"needs_review"`
	DataQualityHints []string                    `json:"data_quality_hints"`
}

// reportDeltas: null when either current or previous side is unavailable for that metric.
// No wip_delta: WIP is a current-board snapshot, so previous-window delta is always ~0 and misleading.
type reportDeltas struct {
	CompletedDelta *int `json:"completed_delta"`
	ReopenedDelta  *int `json:"reopened_delta"`
}

// Daily is period report with from/to (caller sets day bounds).
func (h *Reports) Daily(w http.ResponseWriter, r *http.Request) {
	h.writeReport(w, r, true)
}

// Period is the full period report.
func (h *Reports) Period(w http.ResponseWriter, r *http.Request) {
	h.writeReport(w, r, true)
}

// Preview uses the same shape as period (Architect Decision).
func (h *Reports) Preview(w http.ResponseWriter, r *http.Request) {
	h.writeReport(w, r, true)
}

func (h *Reports) writeReport(w http.ResponseWriter, r *http.Request, withDeltas bool) {
	if h.Engine == nil {
		writeError(w, http.StatusServiceUnavailable, "analytics_unavailable", "analytics engine is not configured")
		return
	}
	q, err := parsePeriodQuery(r)
	if err != nil {
		writeBadRequest(w, "invalid_query", err.Error())
		return
	}
	boardID, err := resolveBoardID(r.Context(), h.Store, h.Cfg.TrelloBoardID)
	if err != nil {
		writeBoardResolveError(w, err)
		return
	}
	res, err := h.Engine.Compute(r.Context(), boardID, q)
	if err != nil {
		h.warn(err)
		WriteInternal(w)
		return
	}

	ids := collectTrelloIDs(res)
	cards, err := h.Store.LoadCardsByTrelloIDs(r.Context(), boardID, ids)
	if err != nil {
		h.warn(err)
		WriteInternal(w)
		return
	}

	memberNameIDs := map[string]struct{}{}
	for _, c := range cards {
		for _, mid := range c.MemberIDs {
			if mid != "" {
				memberNameIDs[mid] = struct{}{}
			}
		}
	}
	memberNames, err := h.Store.LookupMemberNamesByIDs(r.Context(), keysOfStringSet(memberNameIDs))
	if err != nil {
		h.warn(err)
		WriteInternal(w)
		return
	}

	toCard := func(tid string, isBug bool) reportCard {
		c, ok := cards[tid]
		rc := reportCard{
			TrelloCardID: tid,
			Name:         tid,
			MemberIDs:    []string{},
			MemberNames:  []string{},
			LabelIDs:     []string{},
			IsBug:        isBug,
		}
		if ok {
			rc.ID = c.ID
			rc.Name = c.Name
			rc.MemberIDs = c.MemberIDs
			rc.LabelIDs = c.LabelIDs
			rc.ListName = c.CurrentListName
			names := make([]string, 0, len(c.MemberIDs))
			for _, mid := range c.MemberIDs {
				if n := memberNames[mid]; n != "" {
					names = append(names, n)
				}
			}
			rc.MemberNames = names
		}
		return rc
	}

	completedBugSet := map[string]struct{}{}
	for _, tid := range res.BugReopen.CompletedBugCardIDs {
		completedBugSet[tid] = struct{}{}
	}

	completed := make([]reportCard, 0, len(res.Completed.CardIDs))
	for _, tid := range res.Completed.CardIDs {
		_, isBug := completedBugSet[tid]
		completed = append(completed, toCard(tid, isBug))
	}

	fixed := make([]reportCard, 0, len(res.BugReopen.CompletedBugCardIDs))
	for _, tid := range res.BugReopen.CompletedBugCardIDs {
		fixed = append(fixed, toCard(tid, true))
	}

	reopened := make([]reportCard, 0, len(res.BugReopen.ReopenedCardIDs))
	for _, tid := range res.BugReopen.ReopenedCardIDs {
		reopened = append(reopened, toCard(tid, true))
	}

	wip := make([]reportCard, 0, len(res.WIP.CardIDs))
	wipBugSet := map[string]struct{}{}
	for _, tid := range res.WIP.BugCardIDs {
		wipBugSet[tid] = struct{}{}
	}
	for _, tid := range res.WIP.CardIDs {
		_, isBug := wipBugSet[tid]
		wip = append(wip, toCard(tid, isBug))
	}
	stale := make([]reportCard, 0, len(res.Stale.CardIDs))
	for _, tid := range res.Stale.CardIDs {
		stale = append(stale, toCard(tid, false))
	}

	stillOpen := make([]reportCard, 0, len(res.WIP.BugCardIDs))
	for _, tid := range res.WIP.BugCardIDs {
		stillOpen = append(stillOpen, toCard(tid, true))
	}

	hints := make([]string, 0)
	for _, w := range res.Warnings {
		if w.Metric == "data_quality" || w.Code == analytics.WarnDataStale || w.Code == analytics.WarnIncompleteHistory {
			hints = append(hints, w.Message)
		}
	}

	out := reportResponse{
		Period:    reportPeriod{From: q.PeriodStart, To: q.PeriodEnd},
		Summary:   projectReportSummary(res),
		Completed: completed,
		Bugs: reportBugs{
			Fixed:     fixed,
			Reopened:  reopened,
			StillOpen: stillOpen,
		},
		Current: reportCurrent{WIP: wip, Stale: stale},
		Problems: reportProblems{
			StaleCardIDs:     res.Stale.CardIDs,
			ReopenedCardIDs:  res.BugReopen.ReopenedCardIDs,
			NeedsReview:      res.NeedsReview,
			DataQualityHints: hints,
		},
		Warnings: res.Warnings,
	}

	if withDeltas {
		dur := q.PeriodEnd.Sub(q.PeriodStart)
		prevQ := analytics.Query{
			PeriodStart: q.PeriodStart.Add(-dur),
			PeriodEnd:   q.PeriodStart,
			MemberID:    q.MemberID,
			LabelID:     q.LabelID,
		}
		if prev, err := h.Engine.Compute(r.Context(), boardID, prevQ); err == nil {
			out.Deltas = projectReportDeltas(res, prev)
		}
	}

	writeJSON(w, http.StatusOK, out)
}

// projectReportSummary maps Engine availability onto JSON-nullable counts.
func projectReportSummary(res analytics.Result) reportSummary {
	return reportSummary{
		CompletedCount: availableCount(res.Completed.Available, res.Completed.Count),
		BugsFixedCount: availableCount(res.BugReopen.Available, res.BugReopen.CompletedBugs),
		ReopenedCount:  availableCount(res.BugReopen.Available, res.BugReopen.ReopenedBugs),
		WIPCount:       availableCount(res.WIP.Available, res.WIP.Count),
	}
}

func projectReportDeltas(curr, prev analytics.Result) *reportDeltas {
	return &reportDeltas{
		CompletedDelta: deltaWhenBothAvailable(
			curr.Completed.Available, prev.Completed.Available,
			curr.Completed.Count, prev.Completed.Count,
		),
		ReopenedDelta: deltaWhenBothAvailable(
			curr.BugReopen.Available, prev.BugReopen.Available,
			curr.BugReopen.ReopenedBugs, prev.BugReopen.ReopenedBugs,
		),
	}
}

func availableCount(available bool, count int) *int {
	if !available {
		return nil
	}
	v := count
	return &v
}

func deltaWhenBothAvailable(currOK, prevOK bool, curr, prev int) *int {
	if !currOK || !prevOK {
		return nil
	}
	v := curr - prev
	return &v
}

func collectTrelloIDs(res analytics.Result) []string {
	seen := map[string]struct{}{}
	var ids []string
	add := func(list []string) {
		for _, id := range list {
			if _, ok := seen[id]; ok {
				continue
			}
			seen[id] = struct{}{}
			ids = append(ids, id)
		}
	}
	add(res.Completed.CardIDs)
	add(res.BugReopen.CompletedBugCardIDs)
	add(res.BugReopen.ReopenedCardIDs)
	add(res.WIP.CardIDs)
	add(res.WIP.BugCardIDs)
	add(res.Stale.CardIDs)
	return ids
}

// bugsFixedConsistentWithCount reports whether bugs.fixed length matches CompletedBugs
// when card IDs are available from the engine (used by regression tests).
func bugsFixedConsistentWithCount(res analytics.Result) bool {
	if res.BugReopen.CompletedBugs == 0 {
		return len(res.BugReopen.CompletedBugCardIDs) == 0
	}
	return len(res.BugReopen.CompletedBugCardIDs) == res.BugReopen.CompletedBugs
}

func (h *Reports) warn(err error) {
	if h.Logger != nil {
		h.Logger.Warn("reports handler", "err", err)
	}
}

func keysOfStringSet(set map[string]struct{}) []string {
	if len(set) == 0 {
		return nil
	}
	out := make([]string, 0, len(set))
	for k := range set {
		out = append(out, k)
	}
	return out
}
