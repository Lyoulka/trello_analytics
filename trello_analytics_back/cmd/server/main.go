package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"trello_analytics_back/internal/analytics"
	"trello_analytics_back/internal/auth"
	"trello_analytics_back/internal/config"
	apphttp "trello_analytics_back/internal/http"
	"trello_analytics_back/internal/importer"
	"trello_analytics_back/internal/storage/postgres"
	"trello_analytics_back/internal/syncer"
	"trello_analytics_back/internal/trello"
	"trello_analytics_back/internal/workflow"
)

func main() {
	cfg := config.Load()
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	db, err := postgres.Connect(ctx, cfg.DatabaseURL)
	if err != nil {
		logger.Error("database connection failed", "err", err)
		os.Exit(1)
	}
	defer db.Close()

	store := postgres.NewStore(db)

	if err := auth.BootstrapAdmin(ctx, store, cfg.BootstrapAdminEmail, cfg.BootstrapAdminPassword, logger); err != nil {
		logger.Error("bootstrap admin failed", "err", err)
		os.Exit(1)
	}

	wf := &workflow.Service{Store: store}
	engine := &analytics.Engine{Store: store, Workflow: wf}

	var orch *syncer.Orchestrator
	if cfg.TrelloAPIKey != "" && cfg.TrelloToken != "" && cfg.TrelloBoardID != "" {
		client := trello.New(cfg.TrelloAPIKey, cfg.TrelloToken)
		boardImp := &importer.BoardImporter{
			Entities: &importer.EntitiesImporter{Client: client, Store: store},
			Actions:  &importer.BoardActionsImporter{Client: client, Store: store},
		}
		orch = syncer.NewOrchestrator(store, boardImp, syncer.Config{
			BoardID:         cfg.TrelloBoardID,
			ActionsPageSize: cfg.TrelloActionsPageSize,
			APIKey:          cfg.TrelloAPIKey,
			Token:           cfg.TrelloToken,
			Logger:          logger,
		})
		orch.SetLifecycle(ctx)
		if _, err := orch.ReclaimOrphans(ctx); err != nil {
			logger.Warn("reclaim orphan sync runs failed", "err", err)
		}
		logger.Info("HTTP sync entrypoint enabled (dual with CLI)")
	} else {
		logger.Warn("HTTP sync disabled: Trello credentials or board id missing")
	}

	srv := apphttp.NewServer(apphttp.Deps{
		Cfg:      cfg,
		Store:    store,
		Logger:   logger,
		Engine:   engine,
		Workflow: wf,
		Sync:     orch,
	})

	go func() {
		logger.Info("backend listening", "addr", cfg.HTTPAddr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Error("http server failed", "err", err)
			stop()
		}
	}()

	<-ctx.Done()
	logger.Info("shutting down")
	if orch != nil {
		orch.CancelInFlight()
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		logger.Error("graceful shutdown failed", "err", err)
	}
}
