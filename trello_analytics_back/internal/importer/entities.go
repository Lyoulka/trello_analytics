package importer

import (
	"context"
	"fmt"
	"time"

	"trello_analytics_back/internal/storage/postgres"
	"trello_analytics_back/internal/trello"
)

// EntitiesClient fetches board entity payloads (satisfied by *trello.Client).
type EntitiesClient interface {
	GetBoard(ctx context.Context, boardID string) (*trello.Board, error)
	GetLists(ctx context.Context, boardID string, filter string) ([]trello.List, error)
	GetMembers(ctx context.Context, boardID string) ([]trello.Member, error)
	GetLabels(ctx context.Context, boardID string) ([]trello.Label, error)
	GetCards(ctx context.Context, boardID, filter string) ([]trello.Card, error)
}

// EntitiesStore persists board entities with ON CONFLICT upserts.
type EntitiesStore interface {
	UpsertBoard(ctx context.Context, b postgres.BoardUpsert) (boardID string, err error)
	UpsertLists(ctx context.Context, boardID string, lists []postgres.ListUpsert) (map[string]string, error)
	UpsertMembers(ctx context.Context, members []postgres.MemberUpsert) (map[string]string, error)
	UpsertLabels(ctx context.Context, boardID string, labels []postgres.LabelUpsert) (map[string]string, error)
	UpsertCards(ctx context.Context, boardID string, listIDs, memberIDs, labelIDs map[string]string, cards []postgres.CardUpsert) (map[string]string, error)
}

// EntitiesImporter imports board metadata, lists, members, labels, and cards.
type EntitiesImporter struct {
	Client EntitiesClient
	Store  EntitiesStore
}

// ImportEntitiesOptions configures one entity import run.
type ImportEntitiesOptions struct {
	BoardID string
}

// ImportEntitiesResult summarizes entity persistence counts.
type ImportEntitiesResult struct {
	BoardID       string // internal UUID
	Lists         int
	Members       int
	Labels        int
	Cards         int
	ArchivedCards int
}

// ImportEntities loads board → lists → members/labels → cards (filter=all, including archived)
// and upserts into core schema tables by Trello ids.
func (imp *EntitiesImporter) ImportEntities(ctx context.Context, opts ImportEntitiesOptions) (ImportEntitiesResult, error) {
	var out ImportEntitiesResult
	if imp == nil || imp.Client == nil || imp.Store == nil {
		return out, fmt.Errorf("entities importer: client and store are required")
	}
	if opts.BoardID == "" {
		return out, fmt.Errorf("entities importer: board id is required")
	}

	board, err := imp.Client.GetBoard(ctx, opts.BoardID)
	if err != nil {
		return out, fmt.Errorf("fetch board: %w", err)
	}
	if board.ID == "" {
		return out, fmt.Errorf("fetch board: empty id in response")
	}
	name := board.Name
	if name == "" {
		name = board.ID // schema requires NOT NULL name; never invent business data beyond id fallback
	}
	boardUUID, err := imp.Store.UpsertBoard(ctx, postgres.BoardUpsert{
		TrelloBoardID: board.ID,
		Name:          name,
	})
	if err != nil {
		return out, err
	}
	out.BoardID = boardUUID

	lists, err := imp.Client.GetLists(ctx, opts.BoardID, "all")
	if err != nil {
		return out, fmt.Errorf("fetch lists: %w", err)
	}
	listUpserts := make([]postgres.ListUpsert, 0, len(lists))
	for _, l := range lists {
		listName := l.Name
		if listName == "" {
			listName = l.ID
		}
		listUpserts = append(listUpserts, postgres.ListUpsert{
			TrelloListID: l.ID,
			Name:         listName,
			IsArchived:   l.Closed,
		})
	}
	listIDs, err := imp.Store.UpsertLists(ctx, boardUUID, listUpserts)
	if err != nil {
		return out, err
	}
	out.Lists = len(listIDs)

	members, err := imp.Client.GetMembers(ctx, opts.BoardID)
	if err != nil {
		return out, fmt.Errorf("fetch members: %w", err)
	}
	memberUpserts := make([]postgres.MemberUpsert, 0, len(members))
	for _, m := range members {
		// is_active is product-managed; always insert as true for new members.
		// Upsert must not overwrite an existing is_active on conflict.
		memberUpserts = append(memberUpserts, postgres.MemberUpsert{
			TrelloMemberID: m.ID,
			FullName:       m.FullName,
			Username:       m.Username,
			IsActive:       true,
		})
	}
	memberIDs, err := imp.Store.UpsertMembers(ctx, memberUpserts)
	if err != nil {
		return out, err
	}
	out.Members = len(memberUpserts)

	labels, err := imp.Client.GetLabels(ctx, opts.BoardID)
	if err != nil {
		return out, fmt.Errorf("fetch labels: %w", err)
	}
	labelUpserts := make([]postgres.LabelUpsert, 0, len(labels))
	for _, l := range labels {
		labelUpserts = append(labelUpserts, postgres.LabelUpsert{
			TrelloLabelID: l.ID,
			Name:          l.Name,
			Color:         l.Color,
		})
	}
	labelIDs, err := imp.Store.UpsertLabels(ctx, boardUUID, labelUpserts)
	if err != nil {
		return out, err
	}
	out.Labels = len(labelUpserts)

	cards, err := imp.Client.GetCards(ctx, opts.BoardID, "all")
	if err != nil {
		return out, fmt.Errorf("fetch cards: %w", err)
	}
	cardUpserts := make([]postgres.CardUpsert, 0, len(cards))
	for _, c := range cards {
		cardName := c.Name
		if cardName == "" {
			cardName = c.ID
		}
		var lastAct *time.Time
		if c.DateLastActivity != "" {
			if ts, err := parseTrelloActionDate(c.DateLastActivity); err == nil {
				lastAct = &ts
			}
			// Unparseable activity date: leave NULL; do not invent timestamps.
		}
		cardUpserts = append(cardUpserts, postgres.CardUpsert{
			TrelloCardID:       c.ID,
			Name:               cardName,
			TrelloListID:       c.IDList,
			IsArchived:         c.Closed,
			LastTrelloUpdateAt: lastAct,
			TrelloMemberIDs:    append([]string(nil), c.IDMembers...),
			TrelloLabelIDs:     append([]string(nil), c.IDLabels...),
		})
		if c.Closed {
			out.ArchivedCards++
		}
	}
	if _, err := imp.Store.UpsertCards(ctx, boardUUID, listIDs, memberIDs, labelIDs, cardUpserts); err != nil {
		return out, err
	}
	out.Cards = len(cardUpserts)
	return out, nil
}
