package analytics

import (
	"math"
	"sort"
	"time"
)

// DurationStats holds mean/median over observed durations (seconds). Empty → Available=false.
type DurationStats struct {
	Available bool      `json:"available"`
	Count     int       `json:"count"`
	MeanSec   float64   `json:"mean_seconds,omitempty"`
	MedianSec float64   `json:"median_seconds,omitempty"`
	ValuesSec []float64 `json:"-"`
}

func durationStats(durs []time.Duration) DurationStats {
	if len(durs) == 0 {
		return DurationStats{Available: false}
	}
	vals := make([]float64, len(durs))
	var sum float64
	for i, d := range durs {
		sec := d.Seconds()
		vals[i] = sec
		sum += sec
	}
	sort.Float64s(vals)
	return DurationStats{
		Available: true,
		Count:     len(vals),
		MeanSec:   sum / float64(len(vals)),
		MedianSec: medianSorted(vals),
		ValuesSec: vals,
	}
}

func medianSorted(sorted []float64) float64 {
	n := len(sorted)
	if n == 0 {
		return 0
	}
	if n%2 == 1 {
		return sorted[n/2]
	}
	return (sorted[n/2-1] + sorted[n/2]) / 2
}

func round2(v float64) float64 {
	return math.Round(v*100) / 100
}

// setIn returns true if s is in set.
func setIn(set map[string]struct{}, s string) bool {
	if s == "" {
		return false
	}
	_, ok := set[s]
	return ok
}

func toSet(items []string) map[string]struct{} {
	out := make(map[string]struct{}, len(items))
	for _, s := range items {
		if s != "" {
			out[s] = struct{}{}
		}
	}
	return out
}

func ptr(s string) *string { return &s }
