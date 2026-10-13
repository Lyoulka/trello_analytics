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

func TestAPIErrorClassification(t *testing.T) {
	cases := []struct {
		status int
		check  func(error) bool
		name   string
	}{
		{401, IsUnauthorized, "401"},
		{404, IsNotFound, "404"},
		{429, IsRateLimited, "429"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				http.Error(w, "nope", tc.status)
			}))
			defer srv.Close()

			c := New("k", "t")
			c.BaseURL = srv.URL
			c.HTTP = srv.Client()

			_, err := c.GetBoard(context.Background(), "b1")
			if err == nil {
				t.Fatal("expected error")
			}
			if !tc.check(err) {
				t.Fatalf("classification failed for %s: %v", tc.name, err)
			}
			if strings.Contains(err.Error(), "k") && strings.Contains(err.Error(), "token=") {
				t.Fatalf("possible secret leak in error: %v", err)
			}
		})
	}
}

func TestGetCardsRequestsIdMembersAndIdLabels(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasSuffix(r.URL.Path, "/cards") {
			http.NotFound(w, r)
			return
		}
		fields := r.URL.Query().Get("fields")
		if !strings.Contains(fields, "idMembers") || !strings.Contains(fields, "idLabels") {
			t.Errorf("GetCards fields missing link ids: %q", fields)
		}
		_ = json.NewEncoder(w).Encode([]Card{{
			ID: "c1", Name: "Card", IDList: "l1", IDBoard: "b1",
			IDMembers: []string{"m1"}, IDLabels: []string{"lab1"},
		}})
	}))
	defer srv.Close()

	c := New("k", "t")
	c.BaseURL = srv.URL
	c.HTTP = srv.Client()

	cards, err := c.GetCards(context.Background(), "b1", "all")
	if err != nil {
		t.Fatal(err)
	}
	if len(cards) != 1 || len(cards[0].IDMembers) != 1 || cards[0].IDMembers[0] != "m1" {
		t.Fatalf("idMembers: %+v", cards)
	}
	if len(cards[0].IDLabels) != 1 || cards[0].IDLabels[0] != "lab1" {
		t.Fatalf("idLabels: %+v", cards)
	}
}

func TestGetMembersAndLabels(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.HasSuffix(r.URL.Path, "/members"):
			_ = json.NewEncoder(w).Encode([]Member{{ID: "m1", FullName: "Ada", Username: "ada"}})
		case strings.HasSuffix(r.URL.Path, "/labels"):
			_ = json.NewEncoder(w).Encode([]Label{{ID: "l1", Name: "bug", Color: "red", IDBoard: "b1"}})
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	c := New("k", "t")
	c.BaseURL = srv.URL
	c.HTTP = srv.Client()

	members, err := c.GetMembers(context.Background(), "b1")
	if err != nil || len(members) != 1 || members[0].ID != "m1" {
		t.Fatalf("members: %+v err=%v", members, err)
	}
	labels, err := c.GetLabels(context.Background(), "b1")
	if err != nil || len(labels) != 1 || labels[0].ID != "l1" {
		t.Fatalf("labels: %+v err=%v", labels, err)
	}
}

func TestNewFromEnvRequiresCredentials(t *testing.T) {
	t.Setenv("TRELLO_API_KEY", "")
	t.Setenv("TRELLO_TOKEN", "")
	if _, err := NewFromEnv(); err == nil {
		t.Fatal("expected error when credentials missing")
	}
	t.Setenv("TRELLO_API_KEY", "k")
	t.Setenv("TRELLO_TOKEN", "tok")
	c, err := NewFromEnv()
	if err != nil || c.Key != "k" || c.Token != "tok" {
		t.Fatalf("NewFromEnv: %+v err=%v", c, err)
	}
	if c.HTTP == nil || c.HTTP.Timeout != defaultTimeout {
		t.Fatalf("expected 30s timeout, got %+v", c.HTTP)
	}
}
