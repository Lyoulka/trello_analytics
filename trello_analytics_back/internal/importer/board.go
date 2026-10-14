package importer

import (
	"context"
	"fmt"
	"os"
	"strconv"

	"trello_analytics_back/internal/storage/postgres"
)

const (
	defaultActionsPageSize = 100
	minActionsPageSize     = 1
	maxActionsPageSize     = 1000
)

// BoardImporter orchestrates entity import then board-actions import (resume via 1.5).
// sync_runs / scheduler live in internal/syncer and call ImportBoard.
type BoardImporter struct {
	Entities *EntitiesImporter
	Actions  *BoardActionsImporter
}

// ImportBoardOptions configures a full board import (entities + actions).
type ImportBoardOptions struct {
	BoardID string
	// ActionsPageLimit is Trello actions page size. 0 → env TRELLO_ACTIONS_PAGE_SIZE or 100 (clamped 1..1000).
	ActionsPageLimit int
	// StopAfterActionsPages forwards to BoardActionsImporter (tests / interrupt).
	StopAfterActionsPages int
}

// ImportBoardResult combines entity and actions summaries.
type ImportBoardResult struct {
	Entities ImportEntitiesResult
	Actions  ImportBoardActionsResult
}

// ImportBoard runs entities, optional head catch-up, then primary board-actions resume.
// Head catch-up does not move the oldest backfill cursor (InsertCardActionsPage only).
func (imp *BoardImporter) ImportBoard(ctx context.Context, opts ImportBoardOptions) (ImportBoardResult, error) {
	var out ImportBoardResult
	if imp == nil || imp.Entities == nil || imp.Actions == nil {
		return out, fmt.Errorf("board importer: entities and actions importers are required")
	}
	if opts.BoardID == "" {
		return out, fmt.Errorf("board importer: board id is required")
	}

	entities, err := imp.Entities.ImportEntities(ctx, ImportEntitiesOptions{BoardID: opts.BoardID})
	if err != nil {
		return out, fmt.Errorf("import entities: %w", err)
	}
	out.Entities = entities

	pageLimit := resolveActionsPageLimit(opts.ActionsPageLimit)
	actionOpts := ImportBoardActionsOptions{
		BoardID:   opts.BoardID,
		PageLimit: pageLimit,
		// StopAfter applies only to primary resume (tests); head uses full catch-up.
	}

	// If actions were wiped but oldest cursor remains, primary would page with
	// before=<oldest> and insert nothing. Clear cursor so backfill starts at head.
	nActions, err := imp.Actions.Store.CountCardActions(ctx)
	if err != nil {
		return out, fmt.Errorf("count card actions: %w", err)
	}
	if nActions == 0 {
		if err := imp.Actions.Store.UpsertSyncCursor(ctx, postgres.ResourceBoardActions, ""); err != nil {
			return out, fmt.Errorf("reset board_actions cursor after empty actions: %w", err)
		}
	}

	head, err := imp.Actions.ImportBoardActionsHeadCatchUp(ctx, actionOpts)
	if err != nil {
		out.Actions = head
		return out, fmt.Errorf("import board actions head catch-up: %w", err)
	}

	actionOpts.StopAfterPages = opts.StopAfterActionsPages
	actions, err := imp.Actions.ImportBoardActions(ctx, actionOpts)
	if err != nil {
		// Preserve head counters on failure of primary.
		actions.HeadPagesCommitted = head.HeadPagesCommitted
		actions.HeadActionsSeen = head.HeadActionsSeen
		actions.HeadActionsInserted = head.HeadActionsInserted
		actions.HeadSkipped = head.HeadSkipped
		out.Actions = actions
		return out, fmt.Errorf("import board actions: %w", err)
	}
	actions.HeadPagesCommitted = head.HeadPagesCommitted
	actions.HeadActionsSeen = head.HeadActionsSeen
	actions.HeadActionsInserted = head.HeadActionsInserted
	actions.HeadSkipped = head.HeadSkipped
	if head.HeadSkipped {
		actions.HeadSkipped = true
	}
	out.Actions = actions
	return out, nil
}

func resolveActionsPageLimit(explicit int) int {
	if explicit > 0 {
		return clampActionsPageSize(explicit)
	}
	if v := os.Getenv("TRELLO_ACTIONS_PAGE_SIZE"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			return clampActionsPageSize(n)
		}
	}
	return defaultActionsPageSize
}

func clampActionsPageSize(n int) int {
	if n < minActionsPageSize {
		return minActionsPageSize
	}
	if n > maxActionsPageSize {
		return maxActionsPageSize
	}
	return n
}
