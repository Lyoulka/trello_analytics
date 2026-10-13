package importer

import (
	"context"
	"fmt"
	"os"
	"strconv"
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

// ImportBoard runs entities first, then BoardActionsImporter.ImportBoardActions.
// Does not rewrite resume logic — reuses CommitBoardActionsPage / ResourceBoardActions.
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
	actions, err := imp.Actions.ImportBoardActions(ctx, ImportBoardActionsOptions{
		BoardID:        opts.BoardID,
		PageLimit:      pageLimit,
		StopAfterPages: opts.StopAfterActionsPages,
	})
	if err != nil {
		out.Actions = actions
		return out, fmt.Errorf("import board actions: %w", err)
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
