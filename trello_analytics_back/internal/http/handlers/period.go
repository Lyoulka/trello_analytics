package handlers

import (
	"fmt"
	"net/http"
	"time"

	"trello_analytics_back/internal/analytics"
)

// parsePeriodQuery reads from/to (RFC3339 or date YYYY-MM-DD) and optional member_id / label_id.
// from/to are required. label_id is supported (filters cards with that label).
func parsePeriodQuery(r *http.Request) (analytics.Query, error) {
	q := r.URL.Query()
	fromS := q.Get("from")
	toS := q.Get("to")
	if fromS == "" || toS == "" {
		return analytics.Query{}, fmt.Errorf("from and to are required")
	}
	from, err := parseTimeBound(fromS, false)
	if err != nil {
		return analytics.Query{}, fmt.Errorf("invalid from: %w", err)
	}
	to, err := parseTimeBound(toS, true)
	if err != nil {
		return analytics.Query{}, fmt.Errorf("invalid to: %w", err)
	}
	if !to.After(from) {
		return analytics.Query{}, fmt.Errorf("to must be after from")
	}
	return analytics.Query{
		PeriodStart: from,
		PeriodEnd:   to,
		MemberID:    q.Get("member_id"),
		LabelID:     q.Get("label_id"),
	}, nil
}

// parseTimeBound parses RFC3339 or YYYY-MM-DD.
// For date-only: startOfDay when end=false; exclusive next-day midnight when end=true.
func parseTimeBound(s string, end bool) (time.Time, error) {
	if t, err := time.Parse(time.RFC3339, s); err == nil {
		return t.UTC(), nil
	}
	d, err := time.Parse("2006-01-02", s)
	if err != nil {
		return time.Time{}, err
	}
	d = time.Date(d.Year(), d.Month(), d.Day(), 0, 0, 0, 0, time.UTC)
	if end {
		return d.Add(24 * time.Hour), nil
	}
	return d, nil
}
