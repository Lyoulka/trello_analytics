package main

import (
	"bytes"
	"errors"
	"strings"
	"testing"

	"trello_analytics_back/internal/config"
	"trello_analytics_back/internal/syncer"
)

func TestParseCommand(t *testing.T) {
	t.Parallel()
	cases := []struct {
		args    []string
		wantCmd string
		wantOK  bool
	}{
		{nil, "run", true},
		{[]string{}, "run", true},
		{[]string{"run"}, "run", true},
		{[]string{"status"}, "status", true},
		{[]string{"help"}, "", false},
		{[]string{"--help"}, "", false},
		{[]string{"unknown"}, "", false},
	}
	for _, tc := range cases {
		cmd, ok := parseCommand(tc.args)
		if ok != tc.wantOK || cmd != tc.wantCmd {
			t.Fatalf("args=%v → cmd=%q ok=%v want %q/%v", tc.args, cmd, ok, tc.wantCmd, tc.wantOK)
		}
	}
}

func TestRun_UnknownCommandUsage(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := run([]string{"nope"}, &stdout, &stderr)
	if code != exitUsage {
		t.Fatalf("exit=%d want %d", code, exitUsage)
	}
	if !strings.Contains(stderr.String(), "trello-sync") {
		t.Fatalf("usage missing in stderr: %s", stderr.String())
	}
}

func TestRequireSyncEnv(t *testing.T) {
	cfg := config.Config{DatabaseURL: "postgres://x", TrelloBoardID: "b"}
	if code, _ := requireSyncEnv(cfg, true); code != exitUsage {
		t.Fatalf("run without credentials: code=%d", code)
	}
	cfg.TrelloAPIKey = "k"
	cfg.TrelloToken = "t"
	if code, _ := requireSyncEnv(cfg, true); code != exitOK {
		t.Fatalf("run with credentials: code=%d", code)
	}
	cfg2 := config.Config{DatabaseURL: "postgres://x", TrelloBoardID: "b"}
	if code, _ := requireSyncEnv(cfg2, false); code != exitOK {
		t.Fatalf("status without credentials: code=%d", code)
	}
	cfg3 := config.Config{DatabaseURL: "postgres://x"}
	if code, _ := requireSyncEnv(cfg3, false); code != exitUsage {
		t.Fatalf("status without board: code=%d", code)
	}
}

func TestMapRunError(t *testing.T) {
	var stderr bytes.Buffer
	if code := mapRunError(&syncer.AlreadyRunningError{RunID: "r1"}, "", &stderr); code != exitAlreadyRunning {
		t.Fatalf("already: %d", code)
	}
	if !strings.Contains(stderr.String(), "sync_already_running") {
		t.Fatalf("stderr=%s", stderr.String())
	}
	stderr.Reset()
	if code := mapRunError(syncer.ErrRunFailed, "r2", &stderr); code != exitFailed {
		t.Fatalf("failed: %d", code)
	}
	if code := mapRunError(syncer.ErrRunCancelled, "r3", &stderr); code != exitCancelled {
		t.Fatalf("cancelled: %d", code)
	}
	if code := mapRunError(errors.New("boom"), "r4", &stderr); code != exitFailed {
		t.Fatalf("other: %d", code)
	}
}
