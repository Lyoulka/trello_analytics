package analytics

// ShouldExcludeCardForInactiveList reports whether a card's current list is inactive.
// Cards with no current list are kept (same as unassigned members edge case).
func ShouldExcludeCardForInactiveList(currentListID *string, inactiveListIDs map[string]struct{}) bool {
	if currentListID == nil || *currentListID == "" {
		return false
	}
	_, skip := inactiveListIDs[*currentListID]
	return skip
}
