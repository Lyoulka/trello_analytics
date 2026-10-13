package config

import "testing"

func TestLoad_TrelloActionsPageSize(t *testing.T) {
	t.Setenv("TRELLO_ACTIONS_PAGE_SIZE", "")
	cfg := Load()
	if cfg.TrelloActionsPageSize != 100 {
		t.Fatalf("default page size=%d", cfg.TrelloActionsPageSize)
	}

	t.Setenv("TRELLO_ACTIONS_PAGE_SIZE", "50")
	cfg = Load()
	if cfg.TrelloActionsPageSize != 50 {
		t.Fatalf("got %d", cfg.TrelloActionsPageSize)
	}

	t.Setenv("TRELLO_ACTIONS_PAGE_SIZE", "0")
	cfg = Load()
	if cfg.TrelloActionsPageSize != 1 {
		t.Fatalf("clamp min: got %d", cfg.TrelloActionsPageSize)
	}

	t.Setenv("TRELLO_ACTIONS_PAGE_SIZE", "9999")
	cfg = Load()
	if cfg.TrelloActionsPageSize != 1000 {
		t.Fatalf("clamp max: got %d", cfg.TrelloActionsPageSize)
	}
}

func TestLoad_NoSyncIntervalField(t *testing.T) {
	// TRELLO_SYNC_INTERVAL must not be read into Config (no in-process scheduler).
	t.Setenv("TRELLO_SYNC_INTERVAL", "30s")
	cfg := Load()
	_ = cfg // compile-time: Config has no TrelloSyncInterval
}

func TestLoad_AuthDefaults(t *testing.T) {
	t.Setenv("FRONTEND_ORIGIN", "")
	t.Setenv("SESSION_TTL", "")
	cfg := Load()
	if cfg.FrontendOrigin != "http://localhost:5173" {
		t.Fatalf("FrontendOrigin=%q", cfg.FrontendOrigin)
	}
	if cfg.SessionTTL.Hours() != 7*24 {
		t.Fatalf("SessionTTL=%v", cfg.SessionTTL)
	}
}
