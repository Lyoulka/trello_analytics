package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
)

// CardListFilter is the query for GET /cards.
type CardListFilter struct {
	BoardID    string
	MemberID   string
	LabelID    string
	ListID     string
	Archived   *bool
	Search     string
	// UpdatedFrom / UpdatedTo filter last_trello_update_at in [from, to) (UTC).
	UpdatedFrom *time.Time
	UpdatedTo   *time.Time
	Sort        string // name | updated | created (default updated)
	Order       string // asc | desc (default desc)
	Limit       int
	Offset      int
}

// CardListItem is a list-row projection for the cards API.
type CardListItem struct {
	ID                 string     `json:"id"`
	TrelloCardID       string     `json:"trello_card_id"`
	Name               string     `json:"name"`
	CurrentListID      *string    `json:"current_list_id"`
	CurrentListName    *string    `json:"current_list_name"`
	IsArchived         bool       `json:"is_archived"`
	CreatedAtTrello    *time.Time `json:"created_at_trello"`
	LastTrelloUpdateAt *time.Time `json:"last_trello_update_at"`
	// LastUpdatedBy* = creator of the latest card_actions row by action_date
	// (idMemberCreator → members). Empty when no actions / unknown member.
	LastUpdatedByMemberID *string  `json:"last_updated_by_member_id"`
	LastUpdatedByName     *string  `json:"last_updated_by_name"`
	MemberIDs             []string `json:"member_ids"`
	LabelIDs              []string `json:"label_ids"`
}

// CardDetail is a single card with members and labels.
type CardDetail struct {
	CardListItem
	Description *string `json:"description"`
}

// CardHistoryItem is a normalized card_actions row (no raw_payload for Member).
// List/member display fields are resolved from FKs and/or raw_payload (import often leaves FKs null).
type CardHistoryItem struct {
	ID                  string    `json:"id"`
	TrelloActionID      string    `json:"trello_action_id"`
	ActionType          string    `json:"action_type"`
	ActionDate          time.Time `json:"action_date"`
	MemberID            *string   `json:"member_id"`
	MemberName          *string   `json:"member_name,omitempty"`
	SourceListID        *string   `json:"source_list_id"`
	SourceListName      *string   `json:"source_list_name,omitempty"`
	DestinationListID   *string   `json:"destination_list_id"`
	DestinationListName *string   `json:"destination_list_name,omitempty"`
}

// ListCards returns a page of cards for a board with optional filters.
func (s *Store) ListCards(ctx context.Context, f CardListFilter) ([]CardListItem, int, error) {
	if f.BoardID == "" {
		return nil, 0, fmt.Errorf("list cards: board id required")
	}
	if f.Limit <= 0 {
		f.Limit = 50
	}
	if f.Limit > 200 {
		f.Limit = 200
	}
	if f.Offset < 0 {
		f.Offset = 0
	}
	sortCol := "c.last_trello_update_at"
	switch strings.ToLower(f.Sort) {
	case "name":
		sortCol = "c.name"
	case "created":
		sortCol = "c.created_at_trello"
	case "updated", "":
		sortCol = "c.last_trello_update_at"
	default:
		return nil, 0, fmt.Errorf("list cards: invalid sort")
	}
	dir := "DESC"
	if strings.EqualFold(f.Order, "asc") {
		dir = "ASC"
	}

	where := []string{"c.board_id = $1::uuid"}
	args := []any{f.BoardID}
	argN := 2
	if f.MemberID != "" {
		where = append(where, fmt.Sprintf(`EXISTS (
			SELECT 1 FROM card_members cm
			WHERE cm.card_id = c.id AND cm.member_id = $%d::uuid AND cm.removed_at IS NULL
		)`, argN))
		args = append(args, f.MemberID)
		argN++
	}
	if f.LabelID != "" {
		where = append(where, fmt.Sprintf(`EXISTS (
			SELECT 1 FROM card_labels cl WHERE cl.card_id = c.id AND cl.label_id = $%d::uuid
		)`, argN))
		args = append(args, f.LabelID)
		argN++
	}
	if f.ListID != "" {
		where = append(where, fmt.Sprintf("c.current_list_id = $%d::uuid", argN))
		args = append(args, f.ListID)
		argN++
	}
	if f.Archived != nil {
		where = append(where, fmt.Sprintf("c.is_archived = $%d", argN))
		args = append(args, *f.Archived)
		argN++
	}
	if f.Search != "" {
		where = append(where, fmt.Sprintf("c.name ILIKE $%d", argN))
		args = append(args, "%"+f.Search+"%")
		argN++
	}
	if f.UpdatedFrom != nil {
		where = append(where, fmt.Sprintf("c.last_trello_update_at >= $%d", argN))
		args = append(args, f.UpdatedFrom.UTC())
		argN++
	}
	if f.UpdatedTo != nil {
		where = append(where, fmt.Sprintf("c.last_trello_update_at < $%d", argN))
		args = append(args, f.UpdatedTo.UTC())
		argN++
	}
	whereSQL := strings.Join(where, " AND ")

	var total int
	countSQL := `SELECT COUNT(*) FROM cards c WHERE ` + whereSQL
	if err := s.pool.QueryRow(ctx, countSQL, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count cards: %w", err)
	}

	limitArg := argN
	offsetArg := argN + 1
	args = append(args, f.Limit, f.Offset)
	q := fmt.Sprintf(`
		SELECT c.id::text, c.trello_card_id, c.name, c.current_list_id::text, tl.name,
			c.is_archived, c.created_at_trello, c.last_trello_update_at
		FROM cards c
		LEFT JOIN trello_lists tl ON tl.id = c.current_list_id
		WHERE %s
		ORDER BY %s %s NULLS LAST, c.id ASC
		LIMIT $%d OFFSET $%d
	`, whereSQL, sortCol, dir, limitArg, offsetArg)

	rows, err := s.pool.Query(ctx, q, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("list cards: %w", err)
	}
	defer rows.Close()

	var items []CardListItem
	var ids []string
	for rows.Next() {
		var it CardListItem
		if err := rows.Scan(
			&it.ID, &it.TrelloCardID, &it.Name, &it.CurrentListID, &it.CurrentListName,
			&it.IsArchived, &it.CreatedAtTrello, &it.LastTrelloUpdateAt,
		); err != nil {
			return nil, 0, err
		}
		it.MemberIDs = []string{}
		it.LabelIDs = []string{}
		items = append(items, it)
		ids = append(ids, it.ID)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, err
	}
	if len(ids) == 0 {
		return items, total, nil
	}

	members, err := s.cardMemberIDs(ctx, ids)
	if err != nil {
		return nil, 0, err
	}
	labels, err := s.cardLabelIDs(ctx, ids)
	if err != nil {
		return nil, 0, err
	}
	trelloIDs := make([]string, len(items))
	for i := range items {
		trelloIDs[i] = items[i].TrelloCardID
	}
	lastBy, err := s.cardLastUpdatedBy(ctx, trelloIDs)
	if err != nil {
		return nil, 0, err
	}
	for i := range items {
		items[i].MemberIDs = members[items[i].ID]
		if items[i].MemberIDs == nil {
			items[i].MemberIDs = []string{}
		}
		items[i].LabelIDs = labels[items[i].ID]
		if items[i].LabelIDs == nil {
			items[i].LabelIDs = []string{}
		}
		if u, ok := lastBy[items[i].TrelloCardID]; ok {
			if u.MemberID != "" {
				mid := u.MemberID
				items[i].LastUpdatedByMemberID = &mid
			}
			if u.Name != "" {
				name := u.Name
				items[i].LastUpdatedByName = &name
			}
		}
	}
	return items, total, nil
}

type lastUpdatedBy struct {
	MemberID string
	Name     string
}

// cardLastUpdatedBy resolves the creator of the newest action per trello card id.
// Must follow action_date (not list-move preference): dateLastActivity often comes from
// addMember/comment/updateCard without listAfter; preferring moves showed stale movers.
func (s *Store) cardLastUpdatedBy(ctx context.Context, trelloCardIDs []string) (map[string]lastUpdatedBy, error) {
	out := map[string]lastUpdatedBy{}
	if len(trelloCardIDs) == 0 {
		return out, nil
	}
	rows, err := s.pool.Query(ctx, `
		SELECT DISTINCT ON (ca.raw_payload #>> '{data,card,id}')
			ca.raw_payload #>> '{data,card,id}' AS trello_card_id,
			m.id::text,
			COALESCE(NULLIF(m.full_name, ''), NULLIF(m.username, ''), '')
		FROM card_actions ca
		LEFT JOIN members m ON m.trello_member_id = (ca.raw_payload->>'idMemberCreator')
		WHERE ca.raw_payload #>> '{data,card,id}' = ANY($1::text[])
		ORDER BY ca.raw_payload #>> '{data,card,id}', ca.action_date DESC
	`, trelloCardIDs)
	if err != nil {
		return nil, fmt.Errorf("card last updated by: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var tid string
		var mid, name *string
		if err := rows.Scan(&tid, &mid, &name); err != nil {
			return nil, err
		}
		u := lastUpdatedBy{}
		if mid != nil {
			u.MemberID = *mid
		}
		if name != nil {
			u.Name = *name
		}
		out[tid] = u
	}
	return out, rows.Err()
}

func (s *Store) cardMemberIDs(ctx context.Context, cardIDs []string) (map[string][]string, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT card_id::text, member_id::text
		FROM card_members
		WHERE removed_at IS NULL AND card_id = ANY($1::uuid[])
	`, cardIDs)
	if err != nil {
		return nil, fmt.Errorf("card members: %w", err)
	}
	defer rows.Close()
	out := map[string][]string{}
	for rows.Next() {
		var cid, mid string
		if err := rows.Scan(&cid, &mid); err != nil {
			return nil, err
		}
		out[cid] = append(out[cid], mid)
	}
	return out, rows.Err()
}

func (s *Store) cardLabelIDs(ctx context.Context, cardIDs []string) (map[string][]string, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT card_id::text, label_id::text
		FROM card_labels
		WHERE card_id = ANY($1::uuid[])
	`, cardIDs)
	if err != nil {
		return nil, fmt.Errorf("card labels: %w", err)
	}
	defer rows.Close()
	out := map[string][]string{}
	for rows.Next() {
		var cid, lid string
		if err := rows.Scan(&cid, &lid); err != nil {
			return nil, err
		}
		out[cid] = append(out[cid], lid)
	}
	return out, rows.Err()
}

// GetCardDetail loads one card by internal UUID, scoped to board.
func (s *Store) GetCardDetail(ctx context.Context, boardID, cardID string) (*CardDetail, error) {
	var d CardDetail
	err := s.pool.QueryRow(ctx, `
		SELECT c.id::text, c.trello_card_id, c.name, c.description, c.current_list_id::text, tl.name,
			c.is_archived, c.created_at_trello, c.last_trello_update_at
		FROM cards c
		LEFT JOIN trello_lists tl ON tl.id = c.current_list_id
		WHERE c.id = $1::uuid AND c.board_id = $2::uuid
	`, cardID, boardID).Scan(
		&d.ID, &d.TrelloCardID, &d.Name, &d.Description, &d.CurrentListID, &d.CurrentListName,
		&d.IsArchived, &d.CreatedAtTrello, &d.LastTrelloUpdateAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("get card: %w", err)
	}
	members, err := s.cardMemberIDs(ctx, []string{d.ID})
	if err != nil {
		return nil, err
	}
	labels, err := s.cardLabelIDs(ctx, []string{d.ID})
	if err != nil {
		return nil, err
	}
	d.MemberIDs = members[d.ID]
	if d.MemberIDs == nil {
		d.MemberIDs = []string{}
	}
	d.LabelIDs = labels[d.ID]
	if d.LabelIDs == nil {
		d.LabelIDs = []string{}
	}
	return &d, nil
}

// ListCardHistory returns chronological normalized actions for a card (no raw_payload).
// Member/list display names are resolved from raw_payload (+ members/trello_lists),
// because import often leaves card_actions FK columns null.
func (s *Store) ListCardHistory(ctx context.Context, boardID, cardID string) ([]CardHistoryItem, error) {
	var ok bool
	err := s.pool.QueryRow(ctx, `
		SELECT EXISTS(SELECT 1 FROM cards WHERE id = $1::uuid AND board_id = $2::uuid)
	`, cardID, boardID).Scan(&ok)
	if err != nil {
		return nil, fmt.Errorf("card history: %w", err)
	}
	if !ok {
		return nil, nil
	}

	rows, err := s.pool.Query(ctx, `
		SELECT ca.id::text, ca.trello_action_id, ca.action_type, ca.action_date,
			ca.member_id::text, ca.source_list_id::text, ca.destination_list_id::text,
			ca.raw_payload
		FROM card_actions ca
		WHERE ca.card_id = $1::uuid
		   OR (
			ca.card_id IS NULL
			AND ca.raw_payload #>> '{data,card,id}' = (
				SELECT trello_card_id FROM cards WHERE id = $1::uuid
			)
		   )
		ORDER BY ca.action_date ASC, ca.trello_action_id ASC
	`, cardID)
	if err != nil {
		return nil, fmt.Errorf("list card history: %w", err)
	}
	defer rows.Close()

	type pending struct {
		it   CardHistoryItem
		meta historyPayloadMeta
	}
	var pendingRows []pending
	memberIDs := map[string]struct{}{}
	listIDs := map[string]struct{}{}
	for rows.Next() {
		var it CardHistoryItem
		var payload json.RawMessage
		if err := rows.Scan(
			&it.ID, &it.TrelloActionID, &it.ActionType, &it.ActionDate,
			&it.MemberID, &it.SourceListID, &it.DestinationListID,
			&payload,
		); err != nil {
			return nil, err
		}
		meta := parseHistoryPayload(payload)
		if meta.SourceListName != "" {
			n := meta.SourceListName
			it.SourceListName = &n
		}
		if meta.DestListName != "" {
			n := meta.DestListName
			it.DestinationListName = &n
		}
		if meta.MemberTrelloID != "" {
			memberIDs[meta.MemberTrelloID] = struct{}{}
		}
		if meta.SourceListTrelloID != "" {
			listIDs[meta.SourceListTrelloID] = struct{}{}
		}
		if meta.DestListTrelloID != "" {
			listIDs[meta.DestListTrelloID] = struct{}{}
		}
		pendingRows = append(pendingRows, pending{it: it, meta: meta})
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	members, err := s.lookupMemberDisplayByTrelloIDs(ctx, keysOf(memberIDs))
	if err != nil {
		return nil, err
	}
	lists, err := s.lookupListDisplayByTrelloIDs(ctx, keysOf(listIDs))
	if err != nil {
		return nil, err
	}

	out := make([]CardHistoryItem, 0, len(pendingRows))
	for _, p := range pendingRows {
		it := p.it
		if m, ok := members[p.meta.MemberTrelloID]; ok {
			if it.MemberID == nil {
				id := m.ID
				it.MemberID = &id
			}
			if m.Name != "" {
				n := m.Name
				it.MemberName = &n
			}
		}
		if l, ok := lists[p.meta.SourceListTrelloID]; ok {
			if it.SourceListID == nil {
				id := l.ID
				it.SourceListID = &id
			}
			if it.SourceListName == nil && l.Name != "" {
				n := l.Name
				it.SourceListName = &n
			}
		}
		if l, ok := lists[p.meta.DestListTrelloID]; ok {
			if it.DestinationListID == nil {
				id := l.ID
				it.DestinationListID = &id
			}
			if it.DestinationListName == nil && l.Name != "" {
				n := l.Name
				it.DestinationListName = &n
			}
		}
		out = append(out, it)
	}
	return out, nil
}

type historyPayloadMeta struct {
	MemberTrelloID     string
	SourceListTrelloID string
	SourceListName     string
	DestListTrelloID   string
	DestListName       string
}

type idName struct {
	ID   string
	Name string
}

func parseHistoryPayload(payload json.RawMessage) historyPayloadMeta {
	var meta historyPayloadMeta
	if len(payload) == 0 {
		return meta
	}
	var root map[string]any
	if err := json.Unmarshal(payload, &root); err != nil {
		return meta
	}
	if v, ok := root["idMemberCreator"].(string); ok {
		meta.MemberTrelloID = v
	}
	data, _ := root["data"].(map[string]any)
	if data == nil {
		return meta
	}
	meta.SourceListTrelloID, meta.SourceListName = nestedIDName(data, "listBefore")
	meta.DestListTrelloID, meta.DestListName = nestedIDName(data, "listAfter")
	if meta.DestListTrelloID == "" && meta.DestListName == "" {
		meta.DestListTrelloID, meta.DestListName = nestedIDName(data, "list")
	}
	return meta
}

func nestedIDName(m map[string]any, key string) (id, name string) {
	v, _ := m[key].(map[string]any)
	if v == nil {
		return "", ""
	}
	id, _ = v["id"].(string)
	name, _ = v["name"].(string)
	return strings.TrimSpace(id), strings.TrimSpace(name)
}

func keysOf(set map[string]struct{}) []string {
	if len(set) == 0 {
		return nil
	}
	out := make([]string, 0, len(set))
	for k := range set {
		if k != "" {
			out = append(out, k)
		}
	}
	return out
}

func (s *Store) lookupMemberDisplayByTrelloIDs(ctx context.Context, trelloIDs []string) (map[string]idName, error) {
	out := map[string]idName{}
	if len(trelloIDs) == 0 {
		return out, nil
	}
	rows, err := s.pool.Query(ctx, `
		SELECT id::text, trello_member_id,
			COALESCE(NULLIF(full_name, ''), NULLIF(username, ''), '')
		FROM members
		WHERE trello_member_id = ANY($1::text[])
	`, trelloIDs)
	if err != nil {
		return nil, fmt.Errorf("lookup members for history: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var id, tid, name string
		if err := rows.Scan(&id, &tid, &name); err != nil {
			return nil, err
		}
		out[tid] = idName{ID: id, Name: name}
	}
	return out, rows.Err()
}

func (s *Store) lookupListDisplayByTrelloIDs(ctx context.Context, trelloIDs []string) (map[string]idName, error) {
	out := map[string]idName{}
	if len(trelloIDs) == 0 {
		return out, nil
	}
	rows, err := s.pool.Query(ctx, `
		SELECT id::text, trello_list_id, COALESCE(name, '')
		FROM trello_lists
		WHERE trello_list_id = ANY($1::text[])
	`, trelloIDs)
	if err != nil {
		return nil, fmt.Errorf("lookup lists for history: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var id, tid, name string
		if err := rows.Scan(&id, &tid, &name); err != nil {
			return nil, err
		}
		out[tid] = idName{ID: id, Name: name}
	}
	return out, rows.Err()
}

// LoadCardsByTrelloIDs loads card list items keyed by trello card id (for reports).
func (s *Store) LoadCardsByTrelloIDs(ctx context.Context, boardID string, trelloIDs []string) (map[string]CardListItem, error) {
	out := map[string]CardListItem{}
	if len(trelloIDs) == 0 {
		return out, nil
	}
	rows, err := s.pool.Query(ctx, `
		SELECT c.id::text, c.trello_card_id, c.name, c.current_list_id::text, tl.name,
			c.is_archived, c.created_at_trello, c.last_trello_update_at
		FROM cards c
		LEFT JOIN trello_lists tl ON tl.id = c.current_list_id
		WHERE c.board_id = $1::uuid AND c.trello_card_id = ANY($2::text[])
	`, boardID, trelloIDs)
	if err != nil {
		return nil, fmt.Errorf("load cards by trello ids: %w", err)
	}
	defer rows.Close()
	var ids []string
	for rows.Next() {
		var it CardListItem
		if err := rows.Scan(
			&it.ID, &it.TrelloCardID, &it.Name, &it.CurrentListID, &it.CurrentListName,
			&it.IsArchived, &it.CreatedAtTrello, &it.LastTrelloUpdateAt,
		); err != nil {
			return nil, err
		}
		out[it.TrelloCardID] = it
		ids = append(ids, it.ID)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	members, err := s.cardMemberIDs(ctx, ids)
	if err != nil {
		return nil, err
	}
	labels, err := s.cardLabelIDs(ctx, ids)
	if err != nil {
		return nil, err
	}
	for tid, it := range out {
		it.MemberIDs = members[it.ID]
		if it.MemberIDs == nil {
			it.MemberIDs = []string{}
		}
		it.LabelIDs = labels[it.ID]
		if it.LabelIDs == nil {
			it.LabelIDs = []string{}
		}
		out[tid] = it
	}
	return out, nil
}
