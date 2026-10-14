package importer

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"trello_analytics_back/internal/storage/postgres"
	"trello_analytics_back/internal/trello"
)

// BoardActionsClient fetches paginated board actions (satisfied by *trello.Client).
type BoardActionsClient interface {
	GetBoardActions(ctx context.Context, boardID string, limit int, before string) ([]trello.Action, error)
}

// BoardActionsStore persists actions pages and sync cursors.
type BoardActionsStore interface {
	GetSyncCursor(ctx context.Context, resourceType string) (cursorValue string, found bool, err error)
	UpsertSyncCursor(ctx context.Context, resourceType, cursorValue string) error
	CommitBoardActionsPage(ctx context.Context, resourceType string, actions []postgres.CardActionInsert, cursorValue string) (inserted int64, err error)
	// InsertCardActionsPage inserts without updating sync_cursors (head catch-up).
	InsertCardActionsPage(ctx context.Context, actions []postgres.CardActionInsert) (inserted int64, err error)
	CountCardActions(ctx context.Context) (int64, error)
}

// BoardActionsImporter loads board actions into card_actions with DB-backed resume.
type BoardActionsImporter struct {
	Client BoardActionsClient
	Store  BoardActionsStore
}

// ImportBoardActionsOptions configures one import run.
type ImportBoardActionsOptions struct {
	BoardID string
	// PageLimit is Trello actions page size (required > 0).
	PageLimit int
	// StopAfterPages stops after N successfully committed pages (interrupt/resume tests).
	// 0 means no artificial stop.
	StopAfterPages int
}

// ImportBoardActionsResult summarizes one run.
type ImportBoardActionsResult struct {
	PagesCommitted  int
	ActionsSeen     int
	ActionsInserted int64
	CursorValue     string
	StoppedEarly    bool // true when StopAfterPages halted the loop
	// Head* fill when ImportBoard runs head catch-up before primary resume.
	HeadPagesCommitted  int
	HeadActionsSeen     int
	HeadActionsInserted int64
	HeadSkipped         bool // true when catch-up skipped (no existing actions)
}

// ImportBoardActions resumes from sync_cursors (resource_type=board_actions) and
// commits each page atomically: INSERT … ON CONFLICT DO NOTHING + cursor upsert.
// Cursor advances only after a successful page commit. Mid-page errors leave the
// previous cursor unchanged.
func (imp *BoardActionsImporter) ImportBoardActions(ctx context.Context, opts ImportBoardActionsOptions) (ImportBoardActionsResult, error) {
	var out ImportBoardActionsResult
	if imp == nil || imp.Client == nil || imp.Store == nil {
		return out, fmt.Errorf("board actions importer: client and store are required")
	}
	if opts.BoardID == "" {
		return out, fmt.Errorf("board actions importer: board id is required")
	}
	if opts.PageLimit <= 0 {
		return out, fmt.Errorf("board actions importer: page limit must be > 0")
	}

	before, _, err := imp.Store.GetSyncCursor(ctx, postgres.ResourceBoardActions)
	if err != nil {
		return out, err
	}
	out.CursorValue = before

	for {
		if opts.StopAfterPages > 0 && out.PagesCommitted >= opts.StopAfterPages {
			out.StoppedEarly = true
			return out, nil
		}
		if err := ctx.Err(); err != nil {
			return out, err
		}

		actions, err := imp.Client.GetBoardActions(ctx, opts.BoardID, opts.PageLimit, before)
		if err != nil {
			// Fetch failed before commit — cursor must not advance.
			return out, fmt.Errorf("fetch board actions (before=%q): %w", before, err)
		}
		if len(actions) == 0 {
			// Empty board or exhausted page: ensure sync_cursors row exists so
			// primary_import_incomplete can clear after a successful run.
			// Preserve current before (NULL/empty on first empty-board success).
			if _, err := imp.Store.CommitBoardActionsPage(ctx, postgres.ResourceBoardActions, nil, before); err != nil {
				return out, fmt.Errorf("mark board actions backfill complete (before=%q): %w", before, err)
			}
			return out, nil
		}

		rows, nextBefore, err := mapActionsToInserts(actions)
		if err != nil {
			return out, err
		}

		inserted, err := imp.Store.CommitBoardActionsPage(ctx, postgres.ResourceBoardActions, rows, nextBefore)
		if err != nil {
			// Commit failed — cursor must not advance past the last successful page.
			return out, fmt.Errorf("commit board actions page (before=%q): %w", before, err)
		}

		out.PagesCommitted++
		out.ActionsSeen += len(actions)
		out.ActionsInserted += inserted
		out.CursorValue = nextBefore
		before = nextBefore

		if len(actions) < opts.PageLimit {
			return out, nil
		}
	}
}

// ImportBoardActionsHeadCatchUp fetches newest-first pages from the board head and
// inserts missing actions without moving the oldest backfill cursor.
// Call only when card_actions already has rows; otherwise returns HeadSkipped.
func (imp *BoardActionsImporter) ImportBoardActionsHeadCatchUp(ctx context.Context, opts ImportBoardActionsOptions) (ImportBoardActionsResult, error) {
	var out ImportBoardActionsResult
	if imp == nil || imp.Client == nil || imp.Store == nil {
		return out, fmt.Errorf("board actions importer: client and store are required")
	}
	if opts.BoardID == "" {
		return out, fmt.Errorf("board actions importer: board id is required")
	}
	if opts.PageLimit <= 0 {
		return out, fmt.Errorf("board actions importer: page limit must be > 0")
	}

	n, err := imp.Store.CountCardActions(ctx)
	if err != nil {
		return out, err
	}
	if n == 0 {
		out.HeadSkipped = true
		return out, nil
	}

	before := ""
	for {
		if opts.StopAfterPages > 0 && out.PagesCommitted >= opts.StopAfterPages {
			out.StoppedEarly = true
			return out, nil
		}
		if err := ctx.Err(); err != nil {
			return out, err
		}

		actions, err := imp.Client.GetBoardActions(ctx, opts.BoardID, opts.PageLimit, before)
		if err != nil {
			return out, fmt.Errorf("head catch-up fetch board actions (before=%q): %w", before, err)
		}
		if len(actions) == 0 {
			return out, nil
		}

		rows, nextBefore, err := mapActionsToInserts(actions)
		if err != nil {
			return out, err
		}

		inserted, err := imp.Store.InsertCardActionsPage(ctx, rows)
		if err != nil {
			return out, fmt.Errorf("head catch-up insert page (before=%q): %w", before, err)
		}

		out.PagesCommitted++
		out.ActionsSeen += len(actions)
		out.ActionsInserted += inserted
		out.HeadPagesCommitted = out.PagesCommitted
		out.HeadActionsSeen = out.ActionsSeen
		out.HeadActionsInserted = out.ActionsInserted

		if inserted == 0 {
			// Entire page already known — reached existing history.
			return out, nil
		}
		if len(actions) < opts.PageLimit {
			return out, nil
		}
		before = nextBefore
	}
}

func mapActionsToInserts(actions []trello.Action) ([]postgres.CardActionInsert, string, error) {
	rows := make([]postgres.CardActionInsert, 0, len(actions))
	for _, a := range actions {
		if a.ID == "" {
			return nil, "", fmt.Errorf("trello action missing id")
		}
		if a.Type == "" {
			return nil, "", fmt.Errorf("trello action %s missing type", a.ID)
		}
		ts, err := parseTrelloActionDate(a.Date)
		if err != nil {
			return nil, "", fmt.Errorf("trello action %s date: %w", a.ID, err)
		}
		payload, err := json.Marshal(a)
		if err != nil {
			return nil, "", fmt.Errorf("marshal action %s: %w", a.ID, err)
		}
		rows = append(rows, postgres.CardActionInsert{
			TrelloActionID: a.ID,
			ActionType:     a.Type,
			ActionDate:     ts,
			RawPayload:     payload,
		})
	}
	// Trello returns newest-first; last id is the next `before` cursor (spike / API contract).
	nextBefore := actions[len(actions)-1].ID
	return rows, nextBefore, nil
}

func parseTrelloActionDate(raw string) (time.Time, error) {
	if raw == "" {
		return time.Time{}, fmt.Errorf("empty date")
	}
	if t, err := time.Parse(time.RFC3339Nano, raw); err == nil {
		return t.UTC(), nil
	}
	if t, err := time.Parse(time.RFC3339, raw); err == nil {
		return t.UTC(), nil
	}
	return time.Time{}, fmt.Errorf("unsupported date format %q", raw)
}
