// Command trello-sync is the sole product entrypoint for board import/sync.
// It reuses internal/importer + internal/syncer (single-flight, resume, sync_runs).
package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/jackc/pgx/v5/pgxpool"

	"trello_analytics_back/internal/config"
	"trello_analytics_back/internal/importer"
	"trello_analytics_back/internal/storage/postgres"
	"trello_analytics_back/internal/syncer"
	"trello_analytics_back/internal/trello"
)

const (
	exitOK             = 0
	exitUsage          = 2
	exitAlreadyRunning = 3
	exitFailed         = 4
	exitCancelled      = 5
)

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}

func run(args []string, stdout, stderr io.Writer) int {
	cmd, ok := parseCommand(args)
	if !ok {
		printUsage(stderr)
		return exitUsage
	}

	cfg := config.Load()
	logger := slog.New(slog.NewJSONHandler(stderr, &slog.HandlerOptions{Level: slog.LevelInfo}))

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	switch cmd {
	case "run":
		return cmdRun(ctx, cfg, logger, stderr)
	case "status":
		return cmdStatus(ctx, cfg, logger, stdout, stderr)
	default:
		printUsage(stderr)
		return exitUsage
	}
}

// parseCommand returns the subcommand. No args → "run". Unknown → ok=false.
func parseCommand(args []string) (cmd string, ok bool) {
	if len(args) == 0 {
		return "run", true
	}
	switch args[0] {
	case "run", "status":
		return args[0], true
	case "-h", "--help", "help":
		return "", false
	default:
		return "", false
	}
}

func printUsage(w io.Writer) {
	fmt.Fprintf(w, `trello-sync — board import/sync CLI (sole sync entrypoint)

Usage:
  trello-sync [run]    run sync to completion (default)
  trello-sync status   print sync quality status as JSON

Env: DATABASE_URL, TRELLO_API_KEY, TRELLO_TOKEN, TRELLO_BOARD_ID,
     TRELLO_ACTIONS_PAGE_SIZE (optional, default 100)

Exit codes (run): 0 success, 2 usage/config, 3 already running, 4 failed, 5 cancelled
Exit codes (status): 0 ok, 2 usage/config
`)
}

func cmdRun(ctx context.Context, cfg config.Config, logger *slog.Logger, stderr io.Writer) int {
	if code, msg := requireSyncEnv(cfg, true); code != exitOK {
		fmt.Fprintln(stderr, msg)
		return code
	}

	db, err := postgres.Connect(ctx, cfg.DatabaseURL)
	if err != nil {
		fmt.Fprintf(stderr, "database connection failed: %v\n", err)
		return exitUsage
	}
	defer db.Close()

	orch := newOrchestrator(cfg, db, logger)
	orch.SetLifecycle(ctx)

	if _, err := orch.ReclaimOrphans(ctx); err != nil {
		fmt.Fprintf(stderr, "orphan reclaim failed: %v\n", err)
		return exitFailed
	}

	runID, err := orch.Run(ctx)
	if err != nil {
		return mapRunError(err, runID, stderr)
	}
	logger.Info("sync completed", "run_id", runID, "status", "success")
	return exitOK
}

func cmdStatus(ctx context.Context, cfg config.Config, logger *slog.Logger, stdout, stderr io.Writer) int {
	if code, msg := requireSyncEnv(cfg, false); code != exitOK {
		fmt.Fprintln(stderr, msg)
		return code
	}

	db, err := postgres.Connect(ctx, cfg.DatabaseURL)
	if err != nil {
		fmt.Fprintf(stderr, "database connection failed: %v\n", err)
		return exitUsage
	}
	defer db.Close()

	orch := newOrchestrator(cfg, db, logger)
	st, err := orch.Status(ctx)
	if err != nil {
		fmt.Fprintf(stderr, "status failed: %v\n", err)
		return exitUsage
	}
	// Defense: never print obvious secret fragments if redaction missed something.
	if st.CurrentRun != nil && st.CurrentRun.ErrorSummary != nil {
		s := *st.CurrentRun.ErrorSummary
		if strings.Contains(strings.ToLower(s), "token=") || strings.Contains(s, "key=") {
			redacted := "[redacted]"
			st.CurrentRun.ErrorSummary = &redacted
		}
	}
	enc := json.NewEncoder(stdout)
	enc.SetIndent("", "  ")
	if err := enc.Encode(st); err != nil {
		fmt.Fprintf(stderr, "encode status: %v\n", err)
		return exitUsage
	}
	return exitOK
}

func newOrchestrator(cfg config.Config, db *pgxpool.Pool, logger *slog.Logger) *syncer.Orchestrator {
	store := postgres.NewStore(db)
	client := trello.New(cfg.TrelloAPIKey, cfg.TrelloToken)
	boardImp := &importer.BoardImporter{
		Entities: &importer.EntitiesImporter{Client: client, Store: store},
		Actions:  &importer.BoardActionsImporter{Client: client, Store: store},
	}
	return syncer.NewOrchestrator(store, boardImp, syncer.Config{
		BoardID:         cfg.TrelloBoardID,
		ActionsPageSize: cfg.TrelloActionsPageSize,
		APIKey:          cfg.TrelloAPIKey,
		Token:           cfg.TrelloToken,
		Logger:          logger,
	})
}

// requireSyncEnv validates env for run (credentials required) or status (board+db).
func requireSyncEnv(cfg config.Config, needCredentials bool) (code int, msg string) {
	if cfg.DatabaseURL == "" {
		return exitUsage, "config: DATABASE_URL is required"
	}
	if cfg.TrelloBoardID == "" {
		return exitUsage, "config: TRELLO_BOARD_ID is required"
	}
	if needCredentials {
		if cfg.TrelloAPIKey == "" || cfg.TrelloToken == "" {
			return exitUsage, "config: TRELLO_API_KEY and TRELLO_TOKEN are required"
		}
	}
	return exitOK, ""
}

func mapRunError(err error, runID string, stderr io.Writer) int {
	var already *syncer.AlreadyRunningError
	if errors.As(err, &already) {
		fmt.Fprintf(stderr, "sync_already_running run_id=%s\n", already.RunID)
		return exitAlreadyRunning
	}
	if errors.Is(err, syncer.ErrRunCancelled) {
		fmt.Fprintf(stderr, "sync cancelled run_id=%s\n", runID)
		return exitCancelled
	}
	if errors.Is(err, syncer.ErrRunFailed) {
		fmt.Fprintf(stderr, "sync failed run_id=%s\n", runID)
		return exitFailed
	}
	fmt.Fprintf(stderr, "sync error: %v\n", err)
	return exitFailed
}
