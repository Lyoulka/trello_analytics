package postgres

import (
	"context"
	"fmt"
	"time"
)

// BoardUpsert is board metadata for idempotent persistence.
type BoardUpsert struct {
	TrelloBoardID string
	Name          string
}

// ListUpsert is a Trello list column for idempotent persistence.
type ListUpsert struct {
	TrelloListID string
	Name         string
	IsArchived   bool
}

// MemberUpsert is a Trello member for idempotent persistence.
// IsActive is applied only on INSERT (default true for analytics inclusion).
// ON CONFLICT must not overwrite members.is_active — that flag is product-managed.
type MemberUpsert struct {
	TrelloMemberID string
	FullName       string
	Username       string
	IsActive       bool // INSERT only; ignored on update
}

// LabelUpsert is a Trello label for idempotent persistence.
type LabelUpsert struct {
	TrelloLabelID string
	Name          string
	Color         string
}

// CardUpsert is a Trello card for idempotent persistence.
type CardUpsert struct {
	TrelloCardID       string
	Name               string
	TrelloListID       string // resolved to current_list_id UUID when present in DB
	IsArchived         bool
	LastTrelloUpdateAt *time.Time
	// TrelloMemberIDs / TrelloLabelIDs are the current snapshot from the card
	// (idMembers / idLabels). Empty slice clears active links for that card.
	TrelloMemberIDs []string
	TrelloLabelIDs  []string
}

// UpsertBoard inserts or updates boards by trello_board_id (name, updated_at).
// Returns the internal UUID as string.
func (s *Store) UpsertBoard(ctx context.Context, b BoardUpsert) (boardID string, err error) {
	if b.TrelloBoardID == "" {
		return "", fmt.Errorf("upsert board: empty trello_board_id")
	}
	if b.Name == "" {
		return "", fmt.Errorf("upsert board: empty name")
	}
	err = s.pool.QueryRow(ctx, `
		INSERT INTO boards (trello_board_id, name, updated_at)
		VALUES ($1, $2, now())
		ON CONFLICT (trello_board_id) DO UPDATE SET
			name = EXCLUDED.name,
			updated_at = now()
		RETURNING id::text
	`, b.TrelloBoardID, b.Name).Scan(&boardID)
	if err != nil {
		return "", fmt.Errorf("upsert board %s: %w", b.TrelloBoardID, err)
	}
	return boardID, nil
}

// UpsertLists upserts lists for a board by trello_list_id (name, is_archived, updated_at).
// Returns map trello_list_id → internal UUID string.
func (s *Store) UpsertLists(ctx context.Context, boardID string, lists []ListUpsert) (map[string]string, error) {
	out := make(map[string]string, len(lists))
	for _, l := range lists {
		if l.TrelloListID == "" {
			return nil, fmt.Errorf("upsert list: empty trello_list_id")
		}
		if l.Name == "" {
			return nil, fmt.Errorf("upsert list %s: empty name", l.TrelloListID)
		}
		var id string
		// is_active is product-managed; INSERT uses DEFAULT true; CONFLICT must not overwrite it.
		err := s.pool.QueryRow(ctx, `
			INSERT INTO trello_lists (trello_list_id, board_id, name, is_archived, updated_at)
			VALUES ($1, $2::uuid, $3, $4, now())
			ON CONFLICT (trello_list_id) DO UPDATE SET
				board_id = EXCLUDED.board_id,
				name = EXCLUDED.name,
				is_archived = EXCLUDED.is_archived,
				updated_at = now()
			RETURNING id::text
		`, l.TrelloListID, boardID, l.Name, l.IsArchived).Scan(&id)
		if err != nil {
			return nil, fmt.Errorf("upsert list %s: %w", l.TrelloListID, err)
		}
		out[l.TrelloListID] = id
	}
	return out, nil
}

// UpsertMembers upserts members by trello_member_id (full_name, username, updated_at).
// is_active is set only on INSERT; subsequent syncs preserve the product flag.
func (s *Store) UpsertMembers(ctx context.Context, members []MemberUpsert) (map[string]string, error) {
	out := make(map[string]string, len(members))
	for _, m := range members {
		if m.TrelloMemberID == "" {
			return nil, fmt.Errorf("upsert member: empty trello_member_id")
		}
		var id string
		err := s.pool.QueryRow(ctx, `
			INSERT INTO members (trello_member_id, full_name, username, is_active, updated_at)
			VALUES ($1, $2, $3, $4, now())
			ON CONFLICT (trello_member_id) DO UPDATE SET
				full_name = EXCLUDED.full_name,
				username = EXCLUDED.username,
				updated_at = now()
			RETURNING id::text
		`, m.TrelloMemberID, nullIfEmpty(m.FullName), nullIfEmpty(m.Username), m.IsActive).Scan(&id)
		if err != nil {
			return nil, fmt.Errorf("upsert member %s: %w", m.TrelloMemberID, err)
		}
		out[m.TrelloMemberID] = id
	}
	return out, nil
}

// UpsertLabels upserts labels by trello_label_id (name, color; labels have no updated_at column).
func (s *Store) UpsertLabels(ctx context.Context, boardID string, labels []LabelUpsert) (map[string]string, error) {
	out := make(map[string]string, len(labels))
	for _, l := range labels {
		if l.TrelloLabelID == "" {
			return nil, fmt.Errorf("upsert label: empty trello_label_id")
		}
		var id string
		err := s.pool.QueryRow(ctx, `
			INSERT INTO labels (trello_label_id, board_id, name, color)
			VALUES ($1, $2::uuid, $3, $4)
			ON CONFLICT (trello_label_id) DO UPDATE SET
				board_id = EXCLUDED.board_id,
				name = EXCLUDED.name,
				color = EXCLUDED.color
			RETURNING id::text
		`, l.TrelloLabelID, boardID, nullIfEmpty(l.Name), nullIfEmpty(l.Color)).Scan(&id)
		if err != nil {
			return nil, fmt.Errorf("upsert label %s: %w", l.TrelloLabelID, err)
		}
		out[l.TrelloLabelID] = id
	}
	return out, nil
}

// UpsertCards upserts cards by trello_card_id (name, flags, list FK, updated_at)
// and syncs card_members / card_labels from the card's current idMembers/idLabels snapshot.
// first_synced_at is set on insert only and preserved on conflict.
// memberIDs / labelIDs map trello id → internal UUID (from UpsertMembers / UpsertLabels).
func (s *Store) UpsertCards(ctx context.Context, boardID string, listIDs, memberIDs, labelIDs map[string]string, cards []CardUpsert) (map[string]string, error) {
	out := make(map[string]string, len(cards))
	for _, c := range cards {
		if c.TrelloCardID == "" {
			return nil, fmt.Errorf("upsert card: empty trello_card_id")
		}
		if c.Name == "" {
			return nil, fmt.Errorf("upsert card %s: empty name", c.TrelloCardID)
		}
		var listID any
		if c.TrelloListID != "" {
			if id, ok := listIDs[c.TrelloListID]; ok {
				listID = id
			}
			// Unknown list id: leave current_list_id NULL rather than inventing a list.
		}
		var lastUpdate any
		if c.LastTrelloUpdateAt != nil && !c.LastTrelloUpdateAt.IsZero() {
			lastUpdate = c.LastTrelloUpdateAt.UTC()
		}
		var id string
		err := s.pool.QueryRow(ctx, `
			INSERT INTO cards (
				trello_card_id, board_id, name, current_list_id, is_archived,
				last_trello_update_at, first_synced_at, updated_at
			)
			VALUES ($1, $2::uuid, $3, $4::uuid, $5, $6, now(), now())
			ON CONFLICT (trello_card_id) DO UPDATE SET
				board_id = EXCLUDED.board_id,
				name = EXCLUDED.name,
				current_list_id = EXCLUDED.current_list_id,
				is_archived = EXCLUDED.is_archived,
				last_trello_update_at = EXCLUDED.last_trello_update_at,
				updated_at = now()
			RETURNING id::text
		`, c.TrelloCardID, boardID, c.Name, listID, c.IsArchived, lastUpdate).Scan(&id)
		if err != nil {
			return nil, fmt.Errorf("upsert card %s: %w", c.TrelloCardID, err)
		}
		out[c.TrelloCardID] = id

		activeMembers := resolveKnownIDs(c.TrelloMemberIDs, memberIDs)
		if err := s.syncCardMembers(ctx, id, activeMembers); err != nil {
			return nil, fmt.Errorf("sync card_members for %s: %w", c.TrelloCardID, err)
		}
		activeLabels := resolveKnownIDs(c.TrelloLabelIDs, labelIDs)
		if err := s.syncCardLabels(ctx, id, activeLabels); err != nil {
			return nil, fmt.Errorf("sync card_labels for %s: %w", c.TrelloCardID, err)
		}
	}
	return out, nil
}

// resolveKnownIDs maps Trello ids to internal UUIDs; unknown ids are skipped (no invented FKs).
func resolveKnownIDs(trelloIDs []string, known map[string]string) []string {
	if len(trelloIDs) == 0 {
		return nil
	}
	out := make([]string, 0, len(trelloIDs))
	seen := make(map[string]struct{}, len(trelloIDs))
	for _, tid := range trelloIDs {
		if tid == "" {
			continue
		}
		id, ok := known[tid]
		if !ok || id == "" {
			continue
		}
		if _, dup := seen[id]; dup {
			continue
		}
		seen[id] = struct{}{}
		out = append(out, id)
	}
	return out
}

// syncCardMembers upserts active pairs and soft-removes missing ones (removed_at = now()).
// assigned_at stays NULL: card snapshot from Trello does not provide assignment timestamps.
func (s *Store) syncCardMembers(ctx context.Context, cardID string, memberIDs []string) error {
	for _, memberID := range memberIDs {
		_, err := s.pool.Exec(ctx, `
			INSERT INTO card_members (card_id, member_id, assigned_at, removed_at)
			VALUES ($1::uuid, $2::uuid, NULL, NULL)
			ON CONFLICT (card_id, member_id) DO UPDATE SET
				removed_at = NULL
		`, cardID, memberID)
		if err != nil {
			return fmt.Errorf("upsert card_member %s/%s: %w", cardID, memberID, err)
		}
	}
	if len(memberIDs) == 0 {
		_, err := s.pool.Exec(ctx, `
			UPDATE card_members
			SET removed_at = now()
			WHERE card_id = $1::uuid AND removed_at IS NULL
		`, cardID)
		if err != nil {
			return fmt.Errorf("soft-remove all card_members for %s: %w", cardID, err)
		}
		return nil
	}
	_, err := s.pool.Exec(ctx, `
		UPDATE card_members
		SET removed_at = now()
		WHERE card_id = $1::uuid
		  AND removed_at IS NULL
		  AND NOT (member_id = ANY ($2::uuid[]))
	`, cardID, memberIDs)
	if err != nil {
		return fmt.Errorf("soft-remove stale card_members for %s: %w", cardID, err)
	}
	return nil
}

// syncCardLabels replaces the card's label set: upsert current pairs, DELETE missing
// (schema has no removed_at on card_labels).
func (s *Store) syncCardLabels(ctx context.Context, cardID string, labelIDs []string) error {
	for _, labelID := range labelIDs {
		_, err := s.pool.Exec(ctx, `
			INSERT INTO card_labels (card_id, label_id)
			VALUES ($1::uuid, $2::uuid)
			ON CONFLICT (card_id, label_id) DO NOTHING
		`, cardID, labelID)
		if err != nil {
			return fmt.Errorf("upsert card_label %s/%s: %w", cardID, labelID, err)
		}
	}
	if len(labelIDs) == 0 {
		_, err := s.pool.Exec(ctx, `
			DELETE FROM card_labels WHERE card_id = $1::uuid
		`, cardID)
		if err != nil {
			return fmt.Errorf("delete all card_labels for %s: %w", cardID, err)
		}
		return nil
	}
	_, err := s.pool.Exec(ctx, `
		DELETE FROM card_labels
		WHERE card_id = $1::uuid
		  AND NOT (label_id = ANY ($2::uuid[]))
	`, cardID, labelIDs)
	if err != nil {
		return fmt.Errorf("delete stale card_labels for %s: %w", cardID, err)
	}
	return nil
}

// CountBoards returns row count in boards (tests / status helpers).
func (s *Store) CountBoards(ctx context.Context) (int64, error) {
	return s.countTable(ctx, "boards")
}

// CountLists returns row count in trello_lists.
func (s *Store) CountLists(ctx context.Context) (int64, error) {
	return s.countTable(ctx, "trello_lists")
}

// CountMembers returns row count in members.
func (s *Store) CountMembers(ctx context.Context) (int64, error) {
	return s.countTable(ctx, "members")
}

// CountLabels returns row count in labels.
func (s *Store) CountLabels(ctx context.Context) (int64, error) {
	return s.countTable(ctx, "labels")
}

// CountCards returns row count in cards.
func (s *Store) CountCards(ctx context.Context) (int64, error) {
	return s.countTable(ctx, "cards")
}

// CountActiveCardMembers returns card_members rows with removed_at IS NULL.
func (s *Store) CountActiveCardMembers(ctx context.Context) (int64, error) {
	var n int64
	err := s.pool.QueryRow(ctx, `SELECT COUNT(*) FROM card_members WHERE removed_at IS NULL`).Scan(&n)
	if err != nil {
		return 0, fmt.Errorf("count active card_members: %w", err)
	}
	return n, nil
}

// CountCardLabels returns row count in card_labels.
func (s *Store) CountCardLabels(ctx context.Context) (int64, error) {
	return s.countTable(ctx, "card_labels")
}

// CountCardMembersTotal returns all card_members rows (including soft-removed).
func (s *Store) CountCardMembersTotal(ctx context.Context) (int64, error) {
	return s.countTable(ctx, "card_members")
}

func (s *Store) countTable(ctx context.Context, table string) (int64, error) {
	// table names are internal constants only — never user input.
	var n int64
	err := s.pool.QueryRow(ctx, fmt.Sprintf(`SELECT COUNT(*) FROM %s`, table)).Scan(&n)
	if err != nil {
		return 0, fmt.Errorf("count %s: %w", table, err)
	}
	return n, nil
}

// GetBoardNameByTrelloID returns boards.name for assertions (tests).
func (s *Store) GetBoardNameByTrelloID(ctx context.Context, trelloBoardID string) (string, error) {
	var name string
	err := s.pool.QueryRow(ctx, `SELECT name FROM boards WHERE trello_board_id = $1`, trelloBoardID).Scan(&name)
	if err != nil {
		return "", err
	}
	return name, nil
}

// GetCardNameByTrelloID returns cards.name for assertions (tests).
func (s *Store) GetCardNameByTrelloID(ctx context.Context, trelloCardID string) (string, error) {
	var name string
	err := s.pool.QueryRow(ctx, `SELECT name FROM cards WHERE trello_card_id = $1`, trelloCardID).Scan(&name)
	if err != nil {
		return "", err
	}
	return name, nil
}
