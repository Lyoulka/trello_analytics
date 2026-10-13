package analytics

// AssigneeActivity describes current card↔member links with product is_active.
type AssigneeActivity struct {
	CardID   string
	MemberID string
	IsActive bool
}

// ActiveAssigneesResult is the filtered assignee map and cards to drop from Compute.
type ActiveAssigneesResult struct {
	// Assignees maps internal card id → active member ids only.
	Assignees map[string][]string
	// ExcludeCardIDs are cards that have assignments but no active assignee.
	ExcludeCardIDs map[string]struct{}
}

// FilterActiveAssignees applies members.is_active rules for analytics loading.
// Cards with no assignees at all are kept; cards with only inactive assignees are excluded.
func FilterActiveAssignees(rows []AssigneeActivity) ActiveAssigneesResult {
	assignees := map[string][]string{}
	hasAny := map[string]bool{}
	hasActive := map[string]bool{}
	for _, r := range rows {
		hasAny[r.CardID] = true
		if !r.IsActive {
			continue
		}
		hasActive[r.CardID] = true
		assignees[r.CardID] = append(assignees[r.CardID], r.MemberID)
	}
	exclude := map[string]struct{}{}
	for cardID := range hasAny {
		if !hasActive[cardID] {
			exclude[cardID] = struct{}{}
		}
	}
	return ActiveAssigneesResult{Assignees: assignees, ExcludeCardIDs: exclude}
}
