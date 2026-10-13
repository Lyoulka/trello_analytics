package analytics

import (
	"time"

	"trello_analytics_back/internal/workflow"
)

// CardState is current card snapshot for WIP/stale/workload.
type CardState struct {
	ID                 string
	TrelloCardID       string
	Name               string
	CurrentStage       string
	IsArchived         bool
	CreatedAtTrello    *time.Time
	LastActivityAt     *time.Time
	AssigneeMemberIDs  []string
	LabelIDs           []string
	IsBug              bool
}

// BoardInput is the in-memory board snapshot for Compute (unit-testable without DB).
type BoardInput struct {
	Settings       workflow.Settings
	// StagesByTrelloList maps trello list id → logical stage.
	StagesByTrelloList ListResolver
	// TrelloCardID → internal card id (optional; metrics key on trello id when missing).
	CardByTrelloID map[string]CardState
	// MemberDisplayNames maps internal member UUID → display label (full_name or username).
	MemberDisplayNames map[string]string
	Transitions        []StageTransition // chronological
	LastSyncSuccessAt  *time.Time
	AsOf               time.Time
}

// Query selects period and optional filters.
//
// Timezone contract: PeriodStart/PeriodEnd are compared directly to stored action
// timestamps (UTC). Callers must convert local calendar bounds using
// Settings.Timezone into UTC instants before constructing Query; Compute does not
// re-apply timezone.
type Query struct {
	PeriodStart time.Time
	PeriodEnd   time.Time // exclusive end recommended; inclusive checks use [start, end)
	MemberID    string    // optional: filter completed/throughput to cards with this assignee
	LabelID     string    // optional: filter to cards that currently have this label (internal UUID)
	LabelBugOnly bool     // unused in v1 core; reserved
}

// Result aggregates all metrics for a query.
type Result struct {
	Completed   CountMetric      `json:"completed"`
	Throughput  CountMetric      `json:"throughput"`
	LeadTime    DurationStats    `json:"lead_time"`
	CycleTime   DurationStats    `json:"cycle_time"`
	BugReopen   BugReopenMetric  `json:"bug_reopen"`
	FixTime     DurationStats    `json:"fix_time"`
	WIP         WIPMetric        `json:"wip"`
	Stale       StaleMetric      `json:"stale"`
	Workload    []WorkloadEntry  `json:"workload"`
	Warnings    []Warning        `json:"warnings"`
	NeedsReview []NeedsReviewItem `json:"needs_review"`
}

// CountMetric is unique-card count with availability flag.
type CountMetric struct {
	Available bool     `json:"available"`
	Count     int      `json:"count"`
	CardIDs   []string `json:"card_ids,omitempty"`
}

// BugReopenMetric: CompletedBugs / ReopenedBugs are **transition counts** (not unique cards).
// CompletedBugs = bug enters into done_stages in period; ReopenedBugs = leaves from bug_reopen_from_stages to non-done.
type BugReopenMetric struct {
	Available           bool     `json:"available"`
	CompletedBugs       int      `json:"completed_bugs"`
	ReopenedBugs        int      `json:"reopened_bugs"`
	ReopenPercent       float64  `json:"reopen_percent,omitempty"`
	CompletedBugCardIDs []string `json:"completed_bug_card_ids,omitempty"`
	ReopenedCardIDs     []string `json:"reopened_card_ids,omitempty"`
}

// WIPMetric is unfinished work snapshot.
type WIPMetric struct {
	Available  bool     `json:"available"`
	Count      int      `json:"count"`
	CardIDs    []string `json:"card_ids,omitempty"`
	BugCardIDs []string `json:"bug_card_ids,omitempty"` // WIP cards with IsBug (for reports)
	Blocked    int      `json:"blocked_count"`
}

// StaleMetric is cards without Trello activity beyond threshold.
type StaleMetric struct {
	Available bool     `json:"available"`
	Count     int      `json:"count"`
	CardIDs   []string `json:"card_ids,omitempty"`
}

// WorkloadEntry is personal attribution (full count per assignee).
type WorkloadEntry struct {
	MemberID       string `json:"member_id"`
	FullName       string `json:"full_name,omitempty"`
	ActiveCards    int    `json:"active_cards"`
	CompletedCards int    `json:"completed_cards"`
}

type computeCtx struct {
	in           BoardInput
	q            Query
	done         map[string]struct{}
	work         map[string]struct{}
	blocked      map[string]struct{}
	reopenFrom   map[string]struct{} // bug_reopen_from_stages
	byCard       map[string][]StageTransition
	warnings     []Warning
	needsReview  []NeedsReviewItem
	completedIDs map[string]struct{} // trello card ids completed in period
}

func compute(in BoardInput, q Query) Result {
	cc := &computeCtx{
		in:             in,
		q:              q,
		done:           toSet(in.Settings.DoneStages),
		work:           toSet(in.Settings.WorkStages),
		blocked:        toSet(in.Settings.BlockedStages),
		reopenFrom:     toSet(in.Settings.BugReopenFromStages),
		byCard:         GroupByCard(in.Transitions),
		completedIDs:   map[string]struct{}{},
	}
	if cc.in.AsOf.IsZero() {
		cc.in.AsOf = time.Now().UTC()
	}
	cc.checkWorkflowConfig()
	cc.checkDataStale()

	var res Result
	res.Completed, res.Throughput = cc.completedAndThroughput()
	res.LeadTime = cc.leadTime()
	res.CycleTime = cc.cycleTime()
	res.BugReopen, res.FixTime = cc.bugReopenAndFix()
	res.WIP, res.Stale = cc.wipAndStale()
	res.Workload = cc.workload()
	res.Warnings = cc.warnings
	res.NeedsReview = cc.needsReview
	return res
}

func (cc *computeCtx) checkWorkflowConfig() {
	s := cc.in.Settings
	if len(s.DoneStages) == 0 {
		cc.warnings = append(cc.warnings, Warning{
			Code: WarnMissingDoneStages, Message: "done_stages not configured", Metric: "completed",
		})
		cc.warnings = append(cc.warnings, Warning{
			Code: WarnWorkflowIncomplete, Message: "workflow incomplete: done_stages empty", Metric: "completed",
		})
	}
	if s.CycleStartStage == "" {
		cc.warnings = append(cc.warnings, Warning{
			Code: WarnMissingCycleStart, Message: "cycle_start_stage not configured", Metric: "cycle_time",
		})
	}
	if s.LeadTimeStartRule == "" || s.LeadTimeStartRule == workflow.LeadRuleUnset {
		cc.warnings = append(cc.warnings, Warning{
			Code: WarnLeadRuleUnset, Message: "lead_time_start_rule is unset", Metric: "lead_time",
		})
	}
	if len(s.BugReopenFromStages) == 0 {
		cc.warnings = append(cc.warnings, Warning{
			Code: WarnMissingBugReopenFromStages, Message: "bug_reopen_from_stages not configured", Metric: "bug_reopen",
		})
	}
}

func (cc *computeCtx) checkDataStale() {
	age := cc.in.Settings.DataStaleSyncAgeSeconds
	if age <= 0 {
		age = workflow.DefaultDataStaleSyncAgeSeconds
	}
	if cc.in.LastSyncSuccessAt == nil {
		cc.warnings = append(cc.warnings, Warning{
			Code: WarnDataStale, Message: "no successful sync recorded", Metric: "data_quality",
		})
		return
	}
	if cc.in.AsOf.Sub(*cc.in.LastSyncSuccessAt) > time.Duration(age)*time.Second {
		cc.warnings = append(cc.warnings, Warning{
			Code:    WarnDataStale,
			Message: "last successful sync older than data_stale_sync_age",
			Metric:  "data_quality",
		})
	}
}

func (cc *computeCtx) completedAndThroughput() (CountMetric, CountMetric) {
	if len(cc.done) == 0 {
		return CountMetric{Available: false}, CountMetric{Available: false}
	}
	ids := make([]string, 0)
	seen := map[string]struct{}{}
	for cardID, trs := range cc.byCard {
		if !cc.cardPassesMemberFilter(cardID) {
			continue
		}
		enter := firstDoneEnterInPeriod(trs, cc.done, cc.q.PeriodStart, cc.q.PeriodEnd)
		if enter == nil {
			continue
		}
		if _, ok := seen[cardID]; ok {
			continue
		}
		seen[cardID] = struct{}{}
		cc.completedIDs[cardID] = struct{}{}
		ids = append(ids, cardID)
	}
	m := CountMetric{Available: true, Count: len(ids), CardIDs: ids}
	// Throughput == completed unique cards (ADR-007).
	return m, m
}

// firstDoneEnterInPeriod finds first transition into done from non-done whose At is in [start,end).
func firstDoneEnterInPeriod(trs []StageTransition, done map[string]struct{}, start, end time.Time) *StageTransition {
	inDone := false
	var firstInPeriod *StageTransition
	for i := range trs {
		tr := &trs[i]
		if !tr.IsListMove && !tr.IsCreate {
			continue
		}
		toDone := setIn(done, tr.ToStage)
		fromDone := setIn(done, tr.FromStage)
		if tr.IsCreate {
			inDone = toDone
			continue
		}
		// Enter done from non-done.
		if toDone && !fromDone && !inDone {
			if !tr.At.Before(start) && tr.At.Before(end) {
				if firstInPeriod == nil {
					cp := *tr
					firstInPeriod = &cp
				}
			}
			inDone = true
			continue
		}
		// Leave done into non-done (work/other).
		if inDone && !toDone {
			inDone = false
		}
		// done→done: stay inDone, no new enter.
		if toDone && fromDone {
			inDone = true
		}
	}
	return firstInPeriod
}

func (cc *computeCtx) leadTime() DurationStats {
	s := cc.in.Settings
	rule := s.LeadTimeStartRule
	if rule == "" || rule == workflow.LeadRuleUnset {
		return DurationStats{Available: false}
	}
	if len(cc.done) == 0 {
		return DurationStats{Available: false}
	}
	if rule == workflow.LeadRuleFirstEnterStage && s.LeadTimeStartStage == "" {
		return DurationStats{Available: false}
	}

	var durs []time.Duration
	for cardID := range cc.completedIDs {
		trs := cc.byCard[cardID]
		// Same done event as Completed/Throughput: first done-enter in period.
		doneAt := periodDoneAt(trs, cc.done, cc.q.PeriodStart, cc.q.PeriodEnd)
		if doneAt == nil {
			cc.excludeCard(cardID, WarnMissingDone, "lead_time", "missing done timestamp")
			continue
		}
		startAt := leadStartAt(trs, rule, s.LeadTimeStartStage, cc.in.CardByTrelloID[cardID])
		if startAt == nil {
			cc.excludeCard(cardID, WarnMissingStart, "lead_time", "missing lead start event")
			continue
		}
		if !doneAt.After(*startAt) {
			cc.excludeCard(cardID, WarnIncompleteHistory, "lead_time", "done not after lead start")
			continue
		}
		durs = append(durs, doneAt.Sub(*startAt))
	}
	return durationStats(durs)
}

func (cc *computeCtx) cycleTime() DurationStats {
	s := cc.in.Settings
	if s.CycleStartStage == "" || len(cc.done) == 0 {
		return DurationStats{Available: false}
	}
	var durs []time.Duration
	for cardID := range cc.completedIDs {
		trs := cc.byCard[cardID]
		// Same done event as Completed/Throughput: first done-enter in period.
		doneAt := periodDoneAt(trs, cc.done, cc.q.PeriodStart, cc.q.PeriodEnd)
		if doneAt == nil {
			cc.excludeCard(cardID, WarnMissingDone, "cycle_time", "missing done timestamp")
			continue
		}
		startAt := firstEnterStageAt(trs, s.CycleStartStage)
		if startAt == nil {
			cc.excludeCard(cardID, WarnMissingStart, "cycle_time", "missing cycle start event")
			continue
		}
		if !doneAt.After(*startAt) {
			cc.excludeCard(cardID, WarnIncompleteHistory, "cycle_time", "done not after cycle start")
			continue
		}
		durs = append(durs, doneAt.Sub(*startAt))
	}
	return durationStats(durs)
}

func leadStartAt(trs []StageTransition, rule, stage string, card CardState) *time.Time {
	switch rule {
	case workflow.LeadRuleCardCreated:
		for _, tr := range trs {
			if tr.IsCreate {
				t := tr.At
				return &t
			}
		}
		if card.CreatedAtTrello != nil && !card.CreatedAtTrello.IsZero() {
			t := card.CreatedAtTrello.UTC()
			return &t
		}
		return nil
	case workflow.LeadRuleFirstEnterStage:
		return firstEnterStageAt(trs, stage)
	default:
		return nil
	}
}

func firstEnterStageAt(trs []StageTransition, stage string) *time.Time {
	if stage == "" {
		return nil
	}
	for _, tr := range trs {
		if tr.ToStage == stage && tr.FromStage != stage {
			t := tr.At
			return &t
		}
		if tr.IsCreate && tr.ToStage == stage {
			t := tr.At
			return &t
		}
	}
	return nil
}

// periodDoneAt returns the done timestamp for Lead/Cycle: first done-enter in [start,end),
// matching Completed/Throughput (not a later re-completion in the same period).
func periodDoneAt(trs []StageTransition, done map[string]struct{}, start, end time.Time) *time.Time {
	enter := firstDoneEnterInPeriod(trs, done, start, end)
	if enter == nil {
		return nil
	}
	t := enter.At
	return &t
}

func (cc *computeCtx) bugReopenAndFix() (BugReopenMetric, DurationStats) {
	// Denominator: bug transitions into done. Numerator: leave bug_reopen_from → non-done.
	// Both are transition counts (not unique cards).
	if len(cc.done) == 0 || len(cc.reopenFrom) == 0 {
		return BugReopenMetric{Available: false}, DurationStats{Available: false}
	}

	completedCards := map[string]struct{}{}
	reopenedCards := map[string]struct{}{}
	doneEnterCount := 0
	reopenCount := 0
	var fixDurs []time.Duration

	for cardID, trs := range cc.byCard {
		card, ok := cc.in.CardByTrelloID[cardID]
		if !ok || !card.IsBug {
			continue
		}
		for _, ev := range bugDoneEnterEventsInPeriod(trs, cc.done, cc.q.PeriodStart, cc.q.PeriodEnd) {
			doneEnterCount++
			completedCards[cardID] = struct{}{}
			_ = ev
		}
		for _, cyc := range findQAReopenCycles(trs, cc.reopenFrom, cc.done) {
			if !inPeriod(cyc.ReopenAt, cc.q.PeriodStart, cc.q.PeriodEnd) {
				continue
			}
			reopenCount++
			reopenedCards[cardID] = struct{}{}
			if cyc.FixDoneAt != nil && cyc.FixDoneAt.After(cyc.ReopenAt) {
				fixDurs = append(fixDurs, cyc.FixDoneAt.Sub(cyc.ReopenAt))
			}
		}
	}

	completedIDs := make([]string, 0, len(completedCards))
	for id := range completedCards {
		completedIDs = append(completedIDs, id)
	}
	reopenedIDs := make([]string, 0, len(reopenedCards))
	for id := range reopenedCards {
		reopenedIDs = append(reopenedIDs, id)
	}
	m := BugReopenMetric{
		Available:           true,
		CompletedBugs:       doneEnterCount,
		ReopenedBugs:        reopenCount,
		CompletedBugCardIDs: completedIDs,
		ReopenedCardIDs:     reopenedIDs,
	}
	if m.CompletedBugs > 0 {
		m.ReopenPercent = round2(100 * float64(m.ReopenedBugs) / float64(m.CompletedBugs))
	}
	return m, durationStats(fixDurs)
}

func inPeriod(t, start, end time.Time) bool {
	return !t.Before(start) && t.Before(end)
}

// bugDoneEnterEventsInPeriod lists each enter into done from non-done (and create into done) in period.
func bugDoneEnterEventsInPeriod(trs []StageTransition, done map[string]struct{}, start, end time.Time) []StageTransition {
	var out []StageTransition
	for _, tr := range trs {
		if !inPeriod(tr.At, start, end) {
			continue
		}
		if tr.IsCreate {
			if setIn(done, tr.ToStage) {
				out = append(out, tr)
			}
			continue
		}
		if !tr.IsListMove {
			continue
		}
		toDone := setIn(done, tr.ToStage)
		fromDone := setIn(done, tr.FromStage)
		if toDone && !fromDone {
			out = append(out, tr)
		}
	}
	return out
}

// findQAReopenCycles: leave reopenFrom → non-done; next done-enter closes fix time.
func findQAReopenCycles(trs []StageTransition, reopenFrom, done map[string]struct{}) []reopenCycle {
	var cycles []reopenCycle
	var open *reopenCycle
	for _, tr := range trs {
		if tr.IsCreate {
			continue
		}
		if !tr.IsListMove {
			continue
		}
		toDone := setIn(done, tr.ToStage)
		fromReview := setIn(reopenFrom, tr.FromStage)

		if open != nil && toDone {
			t := tr.At
			open.FixDoneAt = &t
			cycles = append(cycles, *open)
			open = nil
		}

		if fromReview && !toDone {
			if open != nil {
				cycles = append(cycles, *open)
			}
			open = &reopenCycle{ReopenAt: tr.At, ReopenActionID: tr.TrelloActionID}
		}
	}
	if open != nil {
		cycles = append(cycles, *open)
	}
	return cycles
}

type reopenCycle struct {
	ReopenAt       time.Time
	ReopenActionID string
	FixDoneAt      *time.Time
}

func (cc *computeCtx) wipAndStale() (WIPMetric, StaleMetric) {
	// Without done_stages every non-archived card looks like WIP — not trustworthy.
	if len(cc.done) == 0 {
		return WIPMetric{Available: false}, StaleMetric{Available: false}
	}
	threshold := cc.in.Settings.StaleThresholdSeconds
	if threshold <= 0 {
		threshold = workflow.DefaultStaleThresholdSeconds
	}
	wipIDs := make([]string, 0)
	wipBugIDs := make([]string, 0)
	staleIDs := make([]string, 0)
	blocked := 0

	for _, card := range cc.in.CardByTrelloID {
		if card.IsArchived {
			continue
		}
		stage := card.CurrentStage
		if setIn(cc.done, stage) {
			continue
		}
		// WIP: not done (includes unmapped / backlog / work / blocked).
		wipIDs = append(wipIDs, card.TrelloCardID)
		if card.IsBug {
			wipBugIDs = append(wipBugIDs, card.TrelloCardID)
		}
		if setIn(cc.blocked, stage) {
			blocked++
		}
		last := card.LastActivityAt
		if last == nil {
			// No activity timestamp → incomplete; do not invent; still may count stale only with known last.
			cc.warnings = append(cc.warnings, Warning{
				Code: WarnIncompleteHistory, Message: "no last activity for stale check", Metric: "stale", CardID: ptr(card.TrelloCardID),
			})
			continue
		}
		if cc.in.AsOf.Sub(*last) > time.Duration(threshold)*time.Second {
			staleIDs = append(staleIDs, card.TrelloCardID)
		}
	}

	return WIPMetric{Available: true, Count: len(wipIDs), CardIDs: wipIDs, BugCardIDs: wipBugIDs, Blocked: blocked},
		StaleMetric{Available: true, Count: len(staleIDs), CardIDs: staleIDs}
}

func (cc *computeCtx) workload() []WorkloadEntry {
	// Personal: full count per current assignee (OQ-007).
	type agg struct{ active, completed int }
	byMember := map[string]*agg{}

	ensure := func(id string) *agg {
		if a, ok := byMember[id]; ok {
			return a
		}
		a := &agg{}
		byMember[id] = a
		return a
	}

	for _, card := range cc.in.CardByTrelloID {
		for _, mid := range card.AssigneeMemberIDs {
			a := ensure(mid)
			if !card.IsArchived && !setIn(cc.done, card.CurrentStage) {
				a.active++
			}
			if _, ok := cc.completedIDs[card.TrelloCardID]; ok {
				a.completed++
			}
		}
	}

	out := make([]WorkloadEntry, 0, len(byMember))
	for mid, a := range byMember {
		name := ""
		if cc.in.MemberDisplayNames != nil {
			name = cc.in.MemberDisplayNames[mid]
		}
		out = append(out, WorkloadEntry{
			MemberID: mid, FullName: name, ActiveCards: a.active, CompletedCards: a.completed,
		})
	}
	return out
}

func (cc *computeCtx) cardPassesMemberFilter(trelloCardID string) bool {
	card, ok := cc.in.CardByTrelloID[trelloCardID]
	if !ok {
		return false
	}
	if cc.q.MemberID != "" {
		found := false
		for _, mid := range card.AssigneeMemberIDs {
			if mid == cc.q.MemberID {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}
	if cc.q.LabelID != "" {
		found := false
		for _, lid := range card.LabelIDs {
			if lid == cc.q.LabelID {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}
	return true
}

func (cc *computeCtx) excludeCard(cardID, code, metric, msg string) {
	cc.warnings = append(cc.warnings, Warning{
		Code: code, Message: msg, Metric: metric, CardID: ptr(cardID),
	})
	cc.needsReview = append(cc.needsReview, NeedsReviewItem{
		CardID: cardID, Reason: msg, Metric: metric,
	})
}
