package analytics

import (
	"context"
	"fmt"
	"time"

	"trello_analytics_back/internal/storage/postgres"
	"trello_analytics_back/internal/workflow"
)

// BoardLoader loads a BoardInput for Compute. Satisfied by Engine's postgres path and tests.
type BoardLoader interface {
	LoadBoard(ctx context.Context, boardID string, asOf time.Time) (BoardInput, error)
}

// Engine computes analytics metrics from stored history + workflow settings.
type Engine struct {
	Store    *postgres.Store
	Workflow *workflow.Service
	// Loader overrides DB loading (tests). When nil, Engine loads via Store+Workflow.
	Loader BoardLoader
}

// Compute runs all metrics for the board and period. Missing start/done → warnings, never invented dates.
func (e *Engine) Compute(ctx context.Context, boardID string, q Query) (Result, error) {
	if e == nil {
		return Result{}, fmt.Errorf("analytics: engine is nil")
	}
	asOf := time.Now().UTC()
	var in BoardInput
	var err error
	if e.Loader != nil {
		in, err = e.Loader.LoadBoard(ctx, boardID, asOf)
	} else {
		in, err = e.loadFromStore(ctx, boardID, asOf)
	}
	if err != nil {
		return Result{}, err
	}
	if q.PeriodEnd.IsZero() || !q.PeriodEnd.After(q.PeriodStart) {
		return Result{}, fmt.Errorf("analytics: invalid period")
	}
	return compute(in, q), nil
}

// ComputeOn computes metrics on an explicit BoardInput (unit tests / offline).
func ComputeOn(in BoardInput, q Query) Result {
	return compute(in, q)
}

func (e *Engine) loadFromStore(ctx context.Context, boardID string, asOf time.Time) (BoardInput, error) {
	if e.Store == nil || e.Workflow == nil {
		return BoardInput{}, fmt.Errorf("analytics: store and workflow are required")
	}
	settings, err := e.Workflow.Load(ctx, boardID)
	if err != nil {
		return BoardInput{}, err
	}

	lists, err := e.Store.ListStagesByBoard(ctx, boardID)
	if err != nil {
		return BoardInput{}, err
	}
	stagesByTrello := make(ListResolver, len(lists))
	listUUIDToStage := make(map[string]string, len(lists))
	inactiveLists := map[string]struct{}{}
	for _, l := range lists {
		if !l.IsActive {
			inactiveLists[l.ID] = struct{}{}
		}
		if l.LogicalStage != nil && *l.LogicalStage != "" {
			stagesByTrello[l.TrelloListID] = *l.LogicalStage
			listUUIDToStage[l.ID] = *l.LogicalStage
		}
	}

	labels, err := e.Store.ListLabelsByBoard(ctx, boardID)
	if err != nil {
		return BoardInput{}, err
	}
	bugLabels := map[string]struct{}{}
	for _, l := range labels {
		if l.IsBugLabel {
			bugLabels[l.ID] = struct{}{}
		}
	}

	cards, err := e.Store.LoadAnalyticsCards(ctx, boardID)
	if err != nil {
		return BoardInput{}, err
	}
	members, err := e.Store.LoadAnalyticsCardMembers(ctx, boardID)
	if err != nil {
		return BoardInput{}, err
	}
	memberRows, err := e.Store.LoadAnalyticsMembers(ctx, boardID)
	if err != nil {
		return BoardInput{}, err
	}
	cardLabels, err := e.Store.LoadAnalyticsCardLabels(ctx, boardID)
	if err != nil {
		return BoardInput{}, err
	}
	activity := make([]AssigneeActivity, len(members))
	for i, m := range members {
		activity[i] = AssigneeActivity{CardID: m.CardID, MemberID: m.MemberID, IsActive: m.IsActive}
	}
	activeFilter := FilterActiveAssignees(activity)
	assignees := activeFilter.Assignees
	memberNames := make(map[string]string, len(memberRows))
	for _, m := range memberRows {
		name := m.FullName
		if name == "" {
			name = m.Username
		}
		if name != "" {
			memberNames[m.ID] = name
		}
	}
	labelsOnCard := map[string][]string{}
	bugsOnCard := map[string]bool{}
	for _, cl := range cardLabels {
		labelsOnCard[cl.CardID] = append(labelsOnCard[cl.CardID], cl.LabelID)
		if _, ok := bugLabels[cl.LabelID]; ok {
			bugsOnCard[cl.CardID] = true
		}
	}

	cardByTrello := make(map[string]CardState, len(cards))
	for _, c := range cards {
		// Cards with assignments but no active assignee are excluded from all metrics.
		if _, skip := activeFilter.ExcludeCardIDs[c.ID]; skip {
			continue
		}
		// Cards currently in an inactive column are excluded from all metrics.
		if ShouldExcludeCardForInactiveList(c.CurrentListID, inactiveLists) {
			continue
		}
		stage := ""
		if c.CurrentListID != nil {
			stage = listUUIDToStage[*c.CurrentListID]
		}
		last := c.LastTrelloUpdateAt
		cardByTrello[c.TrelloCardID] = CardState{
			ID:                c.ID,
			TrelloCardID:      c.TrelloCardID,
			Name:              c.Name,
			CurrentStage:      stage,
			IsArchived:        c.IsArchived,
			CreatedAtTrello:   c.CreatedAtTrello,
			LastActivityAt:    last,
			AssigneeMemberIDs: assignees[c.ID],
			LabelIDs:          labelsOnCard[c.ID],
			IsBug:             bugsOnCard[c.ID],
		}
	}

	actions, err := e.Store.LoadAnalyticsActions(ctx, boardID)
	if err != nil {
		return BoardInput{}, err
	}
	raw := make([]RawAction, len(actions))
	for i, a := range actions {
		raw[i] = RawAction{
			TrelloActionID: a.TrelloActionID,
			ActionType:     a.ActionType,
			ActionDate:     a.ActionDate,
			RawPayload:     a.RawPayload,
		}
	}
	trs := ExtractTransitions(raw, stagesByTrello)
	// Defense in depth: keep only transitions for cards loaded for this board.
	keptTransitions := make([]StageTransition, 0, len(trs))
	for _, tr := range trs {
		if _, ok := cardByTrello[tr.TrelloCardID]; ok {
			keptTransitions = append(keptTransitions, tr)
		}
	}
	trs = keptTransitions

	// Enrich last activity from latest transition when card.last_trello_update_at is null.
	lastByCard := map[string]time.Time{}
	for _, tr := range trs {
		if t, ok := lastByCard[tr.TrelloCardID]; !ok || tr.At.After(t) {
			lastByCard[tr.TrelloCardID] = tr.At
		}
	}
	for id, st := range cardByTrello {
		if st.LastActivityAt == nil {
			if t, ok := lastByCard[id]; ok {
				tt := t
				st.LastActivityAt = &tt
				cardByTrello[id] = st
			}
		}
	}

	lastSync, err := e.Store.LastSyncSuccessAt(ctx)
	if err != nil {
		return BoardInput{}, err
	}

	return BoardInput{
		Settings:           settings,
		StagesByTrelloList: stagesByTrello,
		CardByTrelloID:     cardByTrello,
		MemberDisplayNames: memberNames,
		Transitions:        trs,
		LastSyncSuccessAt:  lastSync,
		AsOf:               asOf,
	}, nil
}
