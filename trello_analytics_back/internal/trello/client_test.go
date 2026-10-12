package trello

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestRedactURL(t *testing.T) {
	raw := "https://api.trello.com/1/boards/abc?key=SECRETKEY&token=SECRETTOKEN"
	got := RedactURL(raw, "SECRETKEY", "SECRETTOKEN")
	if strings.Contains(got, "SECRETKEY") || strings.Contains(got, "SECRETTOKEN") {
		t.Fatalf("secrets leaked in redacted URL: %s", got)
	}
	// QueryEscape turns "***" into "%2A%2A%2A".
	if !strings.Contains(got, "key=") || (!strings.Contains(got, "***") && !strings.Contains(got, "%2A%2A%2A")) {
		t.Fatalf("expected redacted key marker, got %s", got)
	}
}

func TestSummarizePage(t *testing.T) {
	actions := []Action{
		{ID: "a1", Date: "2024-01-02T00:00:00.000Z", Type: "createCard"},
		{ID: "a2", Date: "2024-01-01T00:00:00.000Z", Type: "updateCard"},
	}
	m := SummarizePage(1, 50, "", actions)
	if m.Count != 2 || m.FirstID != "a1" || m.LastID != "a2" {
		t.Fatalf("unexpected meta ids: %+v", m)
	}
	if m.MinDate != "2024-01-01T00:00:00.000Z" || m.MaxDate != "2024-01-02T00:00:00.000Z" {
		t.Fatalf("unexpected dates: %+v", m)
	}
}

func TestGetBoardActionsHTTPTest(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("key") == "" || r.URL.Query().Get("token") == "" {
			t.Errorf("missing auth query")
		}
		_ = json.NewEncoder(w).Encode([]Action{
			{ID: "x1", Type: "commentCard", Date: "2024-06-01T12:00:00.000Z"},
		})
	}))
	defer srv.Close()

	c := New("k", "t")
	c.BaseURL = srv.URL
	c.HTTP = srv.Client()

	actions, err := c.GetBoardActions(context.Background(), "board1", 10, "")
	if err != nil {
		t.Fatal(err)
	}
	if len(actions) != 1 || actions[0].ID != "x1" {
		t.Fatalf("unexpected actions: %+v", actions)
	}
}
