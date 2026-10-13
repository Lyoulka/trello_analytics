package analytics_test

import (
	"testing"

	"trello_analytics_back/internal/analytics"
)

func TestShouldExcludeCardForInactiveList(t *testing.T) {
	inactive := map[string]struct{}{"list-off": {}}
	idOff := "list-off"
	idOn := "list-on"

	if analytics.ShouldExcludeCardForInactiveList(nil, inactive) {
		t.Fatal("nil current list must not be excluded")
	}
	if analytics.ShouldExcludeCardForInactiveList(&idOn, inactive) {
		t.Fatal("active list must not be excluded")
	}
	if !analytics.ShouldExcludeCardForInactiveList(&idOff, inactive) {
		t.Fatal("inactive list must be excluded")
	}
}
