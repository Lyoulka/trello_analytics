package trello

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"
)

func TestRetry_SucceedsAfterTransient503(t *testing.T) {
	var hits atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		n := hits.Add(1)
		if n < 3 {
			http.Error(w, "unavailable", http.StatusServiceUnavailable)
			return
		}
		_ = json.NewEncoder(w).Encode(Board{ID: "b1", Name: "ok"})
	}))
	defer srv.Close()

	c := New("k", "t")
	c.BaseURL = srv.URL
	c.HTTP = srv.Client()
	c.Sleep = func(context.Context, time.Duration) error { return nil }
	c.Jitter = func(max time.Duration) time.Duration { return 0 }

	b, err := c.GetBoard(context.Background(), "b1")
	if err != nil {
		t.Fatalf("GetBoard: %v", err)
	}
	if b.ID != "b1" {
		t.Fatalf("board=%+v", b)
	}
	if hits.Load() != 3 {
		t.Fatalf("hits=%d want 3", hits.Load())
	}
}

func TestRetry_NonRetryable401_NoExtraAttempts(t *testing.T) {
	var hits atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		hits.Add(1)
		http.Error(w, "nope", http.StatusUnauthorized)
	}))
	defer srv.Close()

	c := New("k", "t")
	c.BaseURL = srv.URL
	c.HTTP = srv.Client()
	c.Sleep = func(context.Context, time.Duration) error {
		t.Fatal("sleep should not be called for 401")
		return nil
	}

	_, err := c.GetBoard(context.Background(), "b1")
	if !IsUnauthorized(err) {
		t.Fatalf("want unauthorized, got %v", err)
	}
	if hits.Load() != 1 {
		t.Fatalf("hits=%d want 1", hits.Load())
	}
}

func TestRetry_HonorsRetryAfterOn429(t *testing.T) {
	var hits atomic.Int32
	var slept []time.Duration
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		n := hits.Add(1)
		if n == 1 {
			w.Header().Set("Retry-After", "7")
			http.Error(w, "slow down", http.StatusTooManyRequests)
			return
		}
		_ = json.NewEncoder(w).Encode(Board{ID: "b1", Name: "ok"})
	}))
	defer srv.Close()

	c := New("k", "t")
	c.BaseURL = srv.URL
	c.HTTP = srv.Client()
	c.Sleep = func(_ context.Context, d time.Duration) error {
		slept = append(slept, d)
		return nil
	}
	c.Jitter = func(max time.Duration) time.Duration { return max }

	if _, err := c.GetBoard(context.Background(), "b1"); err != nil {
		t.Fatal(err)
	}
	if len(slept) != 1 || slept[0] != 7*time.Second {
		t.Fatalf("slept=%v want [7s]", slept)
	}
	if !IsRateLimited(mapAPIError(&APIError{StatusCode: 429})) {
		t.Fatal("sanity: 429 should be rate limited")
	}
}

func TestRetry_ExhaustsMaxAttempts(t *testing.T) {
	var hits atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		hits.Add(1)
		http.Error(w, "boom", http.StatusBadGateway)
	}))
	defer srv.Close()

	c := New("k", "t")
	c.BaseURL = srv.URL
	c.HTTP = srv.Client()
	c.RetryMaxAttempts = 3
	c.Sleep = func(context.Context, time.Duration) error { return nil }
	c.Jitter = func(time.Duration) time.Duration { return 0 }

	_, err := c.GetBoard(context.Background(), "b1")
	if err == nil {
		t.Fatal("expected error")
	}
	var api *APIError
	if !IsRetryable(err) {
		t.Fatalf("502 should be retryable classification: %v", err)
	}
	_ = api
	if hits.Load() != 3 {
		t.Fatalf("hits=%d want 3", hits.Load())
	}
}

func TestIsRetryable_Classification(t *testing.T) {
	cases := []struct {
		err  error
		want bool
	}{
		{mapAPIError(&APIError{StatusCode: 401}), false},
		{mapAPIError(&APIError{StatusCode: 403}), false},
		{mapAPIError(&APIError{StatusCode: 404}), false},
		{mapAPIError(&APIError{StatusCode: 429}), true},
		{mapAPIError(&APIError{StatusCode: 500}), true},
		{mapAPIError(&APIError{StatusCode: 502}), true},
		{mapAPIError(&APIError{StatusCode: 503}), true},
		{mapAPIError(&APIError{StatusCode: 504}), true},
		{context.Canceled, false},
	}
	for _, tc := range cases {
		if got := IsRetryable(tc.err); got != tc.want {
			t.Fatalf("IsRetryable(%v)=%v want %v", tc.err, got, tc.want)
		}
	}
}

func TestComputeRetryDelay_ExponentialCap(t *testing.T) {
	c := New("k", "t")
	c.Jitter = func(max time.Duration) time.Duration { return max }

	if d := c.computeRetryDelay(1, nil); d != 1*time.Second {
		t.Fatalf("attempt1=%v", d)
	}
	if d := c.computeRetryDelay(2, nil); d != 2*time.Second {
		t.Fatalf("attempt2=%v", d)
	}
	if d := c.computeRetryDelay(7, nil); d != 60*time.Second {
		t.Fatalf("capped=%v", d)
	}
}
