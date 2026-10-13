// Command trello-spike is a research/throwaway read-only CLI for exploring Trello API
// history (pagination, action types, archived cards, errors, idempotency, resume).
// It is NOT part of the product HTTP API and must not be wired into cmd/server.
package main

import (
	"bufio"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"trello_analytics_back/internal/trello"
)

const (
	defaultOutDir   = "tmp/trello-spike"
	defaultLimit    = 50
	resumeFileName  = "resume.json"
	keyScenariosMsg = "createCard,updateCard,addMemberToCard,removeMemberFromCard,addLabelToCard,removeLabelFromCard,commentCard"
)

func main() {
	if err := loadDotEnv(".env"); err != nil {
		fmt.Fprintf(os.Stderr, "warn: load .env: %v\n", err)
	}

	if len(os.Args) < 2 {
		printUsage()
		os.Exit(2)
	}

	mode := os.Args[1]
	args := os.Args[2:]

	var err error
	switch mode {
	case "board":
		err = runBoard(args)
	case "actions":
		err = runActions(args)
	case "archived-compare":
		err = runArchivedCompare(args)
	case "errors":
		err = runErrors(args)
	case "idempotency":
		err = runIdempotency(args)
	case "resume":
		err = runResume(args)
	case "-h", "--help", "help":
		printUsage()
		return
	default:
		fmt.Fprintf(os.Stderr, "unknown mode %q\n", mode)
		printUsage()
		os.Exit(2)
	}
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Fprintf(os.Stderr, `trello-spike — read-only Trello API research CLI (throwaway; not product API)

Env: TRELLO_API_KEY, TRELLO_TOKEN, TRELLO_BOARD_ID (optional .env)

Modes:
  board             board metadata + lists + cards sample
  actions           paginated board/card actions + type frequencies
  archived-compare  compare actions for archived vs open cards
  errors            invalid token / bad board id / 429 probe
  idempotency       re-fetch same page; expect 100%% action id overlap
  resume            stop after N pages, save cursor, continue; verify id union

Examples:
  go run ./cmd/trello-spike board --filter-archived
  go run ./cmd/trello-spike actions --limit 50 --max-pages 5
  go run ./cmd/trello-spike actions --card-id <id> --limit 20
  go run ./cmd/trello-spike archived-compare --limit 20
  go run ./cmd/trello-spike errors
  go run ./cmd/trello-spike idempotency --limit 50
  go run ./cmd/trello-spike resume --limit 50 --stop-after 2
`)
}

// --- shared flags / env ---

type commonFlags struct {
	outDir string
	limit  int
}

func addCommonFlags(fs *flag.FlagSet, c *commonFlags) {
	fs.StringVar(&c.outDir, "out-dir", defaultOutDir, "directory for JSONL mirrors and resume cursor (gitignored under tmp/)")
	fs.IntVar(&c.limit, "limit", defaultLimit, "Trello actions page size")
}

func requireCreds() (key, token, boardID string, err error) {
	key = os.Getenv("TRELLO_API_KEY")
	token = os.Getenv("TRELLO_TOKEN")
	boardID = os.Getenv("TRELLO_BOARD_ID")
	missing := []string{}
	if key == "" {
		missing = append(missing, "TRELLO_API_KEY")
	}
	if token == "" {
		missing = append(missing, "TRELLO_TOKEN")
	}
	if boardID == "" {
		missing = append(missing, "TRELLO_BOARD_ID")
	}
	if len(missing) > 0 {
		return "", "", "", fmt.Errorf("missing env: %s", strings.Join(missing, ", "))
	}
	return key, token, boardID, nil
}

func newClient() (*trello.Client, string, error) {
	key, token, boardID, err := requireCreds()
	if err != nil {
		return nil, "", err
	}
	return trello.New(key, token), boardID, nil
}

type jsonlWriter struct {
	stdout *bufio.Writer
	file   *os.File
	fw     *bufio.Writer
}

func openJSONL(outDir, name string) (*jsonlWriter, error) {
	w := &jsonlWriter{stdout: bufio.NewWriter(os.Stdout)}
	if outDir != "" {
		if err := os.MkdirAll(outDir, 0o755); err != nil {
			return nil, err
		}
		path := filepath.Join(outDir, name)
		f, err := os.Create(path)
		if err != nil {
			return nil, err
		}
		w.file = f
		w.fw = bufio.NewWriter(f)
		emitLog(map[string]any{"event": "mirror_open", "path": path})
	}
	return w, nil
}

func (w *jsonlWriter) Write(v any) error {
	b, err := json.Marshal(v)
	if err != nil {
		return err
	}
	b = append(b, '\n')
	if _, err := w.stdout.Write(b); err != nil {
		return err
	}
	if w.fw != nil {
		if _, err := w.fw.Write(b); err != nil {
			return err
		}
	}
	return nil
}

func (w *jsonlWriter) Close() error {
	_ = w.stdout.Flush()
	if w.fw != nil {
		_ = w.fw.Flush()
	}
	if w.file != nil {
		return w.file.Close()
	}
	return nil
}

// emitLog writes observation lines to stderr as JSON (never secrets).
func emitLog(v any) {
	b, err := json.Marshal(v)
	if err != nil {
		fmt.Fprintf(os.Stderr, "{\"event\":\"log_marshal_error\"}\n")
		return
	}
	fmt.Fprintf(os.Stderr, "%s\n", b)
}

// --- modes ---

func runBoard(args []string) error {
	fs := flag.NewFlagSet("board", flag.ExitOnError)
	var cf commonFlags
	addCommonFlags(fs, &cf)
	filterArchived := fs.Bool("filter-archived", false, "if set, also list closed (archived) cards separately")
	if err := fs.Parse(args); err != nil {
		return err
	}

	client, boardID, err := newClient()
	if err != nil {
		return err
	}
	ctx := context.Background()

	w, err := openJSONL(cf.outDir, "board.jsonl")
	if err != nil {
		return err
	}
	defer w.Close()

	board, err := client.GetBoard(ctx, boardID)
	if err != nil {
		return err
	}
	_ = w.Write(map[string]any{"kind": "board", "id": board.ID, "name": board.Name, "closed": board.Closed, "dateLastActivity": board.DateLastActivity})

	lists, err := client.GetLists(ctx, boardID, "all")
	if err != nil {
		return err
	}
	_ = w.Write(map[string]any{"kind": "lists_summary", "count": len(lists)})
	for _, l := range lists {
		_ = w.Write(map[string]any{"kind": "list", "id": l.ID, "name": l.Name, "closed": l.Closed, "pos": l.Pos})
	}

	cards, err := client.GetCards(ctx, boardID, "all")
	if err != nil {
		return err
	}
	openN, closedN := 0, 0
	for _, c := range cards {
		if c.Closed {
			closedN++
		} else {
			openN++
		}
	}
	_ = w.Write(map[string]any{
		"kind": "cards_summary", "filter": "all", "total": len(cards),
		"open": openN, "archived_closed": closedN,
	})

	sampleLimit := 10
	shownOpen, shownClosed := 0, 0
	for _, c := range cards {
		if c.Closed {
			if !*filterArchived && shownClosed >= 3 {
				continue
			}
			if *filterArchived && shownClosed >= sampleLimit {
				continue
			}
			shownClosed++
			_ = w.Write(map[string]any{"kind": "card", "id": c.ID, "name": truncate(c.Name, 80), "closed": true, "idList": c.IDList})
		} else {
			if shownOpen >= sampleLimit {
				continue
			}
			shownOpen++
			_ = w.Write(map[string]any{"kind": "card", "id": c.ID, "name": truncate(c.Name, 80), "closed": false, "idList": c.IDList})
		}
	}

	emitLog(map[string]any{"event": "board_done", "lists": len(lists), "cards": len(cards), "open": openN, "archived": closedN})
	return nil
}

func runActions(args []string) error {
	fs := flag.NewFlagSet("actions", flag.ExitOnError)
	var cf commonFlags
	addCommonFlags(fs, &cf)
	maxPages := fs.Int("max-pages", 5, "max action pages to fetch")
	cardID := fs.String("card-id", "", "if set, fetch card actions instead of board actions")
	if err := fs.Parse(args); err != nil {
		return err
	}

	client, boardID, err := newClient()
	if err != nil {
		return err
	}
	ctx := context.Background()

	mirror := "actions-board.jsonl"
	if *cardID != "" {
		mirror = "actions-card.jsonl"
	}
	w, err := openJSONL(cf.outDir, mirror)
	if err != nil {
		return err
	}
	defer w.Close()

	freq := map[string]int{}
	samples := map[string]map[string]any{} // type -> one minimized example
	keyTypes := strings.Split(keyScenariosMsg, ",")

	before := ""
	total := 0
	for page := 1; page <= *maxPages; page++ {
		var actions []trello.Action
		var fetchErr error
		if *cardID != "" {
			actions, fetchErr = client.GetCardActions(ctx, *cardID, cf.limit, before)
		} else {
			actions, fetchErr = client.GetBoardActions(ctx, boardID, cf.limit, before)
		}
		if fetchErr != nil {
			return fetchErr
		}
		meta := trello.SummarizePage(page, cf.limit, before, actions)
		emitLog(map[string]any{
			"event": "page", "page": meta.Page, "limit": meta.Limit, "before": meta.Before,
			"count": meta.Count, "min_date": meta.MinDate, "max_date": meta.MaxDate,
			"first_id": meta.FirstID, "last_id": meta.LastID,
		})
		_ = w.Write(map[string]any{"kind": "page_meta", "meta": meta})

		for _, a := range actions {
			freq[a.Type]++
			total++
			minA := trello.MinimizeAction(a)
			_ = w.Write(map[string]any{"kind": "action", "action": minA})
			if _, ok := samples[a.Type]; !ok {
				samples[a.Type] = minA
			}
		}

		if len(actions) == 0 {
			emitLog(map[string]any{"event": "pagination_end", "reason": "empty_page", "page": page})
			break
		}
		if len(actions) < cf.limit {
			emitLog(map[string]any{"event": "pagination_end", "reason": "short_page", "page": page, "count": len(actions)})
			break
		}
		before = actions[len(actions)-1].ID
	}

	_ = w.Write(map[string]any{"kind": "type_frequencies", "total_actions": total, "frequencies": freq})
	for _, kt := range keyTypes {
		ex, ok := samples[kt]
		if !ok {
			_ = w.Write(map[string]any{"kind": "key_scenario_sample", "type": kt, "status": "not_seen_in_sample"})
			continue
		}
		_ = w.Write(map[string]any{"kind": "key_scenario_sample", "type": kt, "status": "seen", "example": ex})
	}
	// Also dump other seen types as samples (capped).
	other := []string{}
	for t := range samples {
		found := false
		for _, kt := range keyTypes {
			if t == kt {
				found = true
				break
			}
		}
		if !found {
			other = append(other, t)
		}
	}
	sort.Strings(other)
	for _, t := range other {
		_ = w.Write(map[string]any{"kind": "other_type_sample", "type": t, "example": samples[t]})
	}

	emitLog(map[string]any{"event": "actions_done", "total": total, "distinct_types": len(freq)})
	return nil
}

func runArchivedCompare(args []string) error {
	fs := flag.NewFlagSet("archived-compare", flag.ExitOnError)
	var cf commonFlags
	addCommonFlags(fs, &cf)
	perCardPages := fs.Int("max-pages", 3, "max action pages per sample card")
	sampleN := fs.Int("sample", 3, "number of open and archived cards to sample")
	if err := fs.Parse(args); err != nil {
		return err
	}

	client, boardID, err := newClient()
	if err != nil {
		return err
	}
	ctx := context.Background()

	w, err := openJSONL(cf.outDir, "archived-compare.jsonl")
	if err != nil {
		return err
	}
	defer w.Close()

	cards, err := client.GetCards(ctx, boardID, "all")
	if err != nil {
		return err
	}
	var openCards, closedCards []trello.Card
	for _, c := range cards {
		if c.Closed {
			closedCards = append(closedCards, c)
		} else {
			openCards = append(openCards, c)
		}
	}
	// Prefer cards with dateLastActivity — first cards in filter=all may have empty /cards/{id}/actions.
	openCards = preferActiveCards(openCards)
	closedCards = preferActiveCards(closedCards)
	_ = w.Write(map[string]any{
		"kind": "card_pool", "total": len(cards), "open": len(openCards), "archived": len(closedCards),
		"sampling": "prefer non-empty dateLastActivity, then scan until sample cards with actions>0 when possible",
	})

	sampleCardActions := func(label string, pool []trello.Card) error {
		want := *sampleN
		if want == 0 {
			_ = w.Write(map[string]any{"kind": "card_actions_summary", "cohort": label, "status": "no_cards_in_cohort"})
			return nil
		}
		got := 0
		scanned := 0
		for _, c := range pool {
			if got >= want {
				break
			}
			scanned++
			freq := map[string]int{}
			total := 0
			before := ""
			for page := 1; page <= *perCardPages; page++ {
				actions, err := client.GetCardActions(ctx, c.ID, cf.limit, before)
				if err != nil {
					return err
				}
				meta := trello.SummarizePage(page, cf.limit, before, actions)
				emitLog(map[string]any{
					"event": "page", "cohort": label, "card_id": c.ID, "page": meta.Page,
					"limit": meta.Limit, "before": meta.Before, "count": meta.Count,
					"min_date": meta.MinDate, "max_date": meta.MaxDate,
					"first_id": meta.FirstID, "last_id": meta.LastID,
				})
				for _, a := range actions {
					freq[a.Type]++
					total++
				}
				if len(actions) == 0 || len(actions) < cf.limit {
					break
				}
				before = actions[len(actions)-1].ID
			}
			// Skip empty cards while we still have pool left to search; keep at least one empty observation.
			if total == 0 && got < want-1 && scanned < len(pool) {
				_ = w.Write(map[string]any{
					"kind": "card_actions_summary", "cohort": label, "status": "skipped_empty",
					"card_id": c.ID, "card_name": truncate(c.Name, 80), "closed": c.Closed,
				})
				continue
			}
			got++
			_ = w.Write(map[string]any{
				"kind": "card_actions_summary", "cohort": label,
				"card_id": c.ID, "card_name": truncate(c.Name, 80), "closed": c.Closed,
				"actions_fetched": total, "type_frequencies": freq,
			})
		}
		if got == 0 {
			_ = w.Write(map[string]any{"kind": "card_actions_summary", "cohort": label, "status": "no_cards_in_cohort"})
		}
		return nil
	}

	if err := sampleCardActions("open", openCards); err != nil {
		return err
	}
	if err := sampleCardActions("archived", closedCards); err != nil {
		return err
	}

	_ = w.Write(map[string]any{
		"kind": "compare_note",
		"note": "Compare actions_fetched and type_frequencies across cohorts; do not invent missing events.",
	})
	emitLog(map[string]any{"event": "archived_compare_done", "open_pool": len(openCards), "archived_pool": len(closedCards)})
	return nil
}

func runErrors(args []string) error {
	fs := flag.NewFlagSet("errors", flag.ExitOnError)
	var cf commonFlags
	addCommonFlags(fs, &cf)
	if err := fs.Parse(args); err != nil {
		return err
	}

	key, token, boardID, err := requireCreds()
	if err != nil {
		return err
	}
	client := trello.New(key, token)
	ctx := context.Background()

	w, err := openJSONL(cf.outDir, "errors.jsonl")
	if err != nil {
		return err
	}
	defer w.Close()

	// 1) Invalid token
	status, body, redURL, err := client.DoRaw(ctx, "/boards/"+boardID, nil, "invalid-token-for-spike")
	rec := map[string]any{
		"kind": "error_probe", "scenario": "invalid_token",
		"status": status, "url": redURL, "body_snippet": truncate(string(body), 160),
	}
	if err != nil {
		rec["transport_error"] = err.Error()
	}
	_ = w.Write(rec)
	emitLog(map[string]any{"event": "error_probe", "scenario": "invalid_token", "status": status})

	// 2) Bad board id
	status, body, redURL, err = client.DoRaw(ctx, "/boards/000000000000000000000000", nil, "")
	rec = map[string]any{
		"kind": "error_probe", "scenario": "bad_board_id",
		"status": status, "url": redURL, "body_snippet": truncate(string(body), 160),
	}
	if err != nil {
		rec["transport_error"] = err.Error()
	}
	_ = w.Write(rec)
	emitLog(map[string]any{"event": "error_probe", "scenario": "bad_board_id", "status": status})

	// 3) 429 — only document if actually observed; do not hammer the API.
	_ = w.Write(map[string]any{
		"kind": "error_probe", "scenario": "rate_limit_429",
		"status": "not_reproduced",
		"note":   "Spike does not deliberately trigger 429. If a live call returns 429, record it; otherwise leave unconfirmed.",
	})
	emitLog(map[string]any{"event": "error_probe", "scenario": "rate_limit_429", "status": "not_reproduced"})

	return nil
}

func runIdempotency(args []string) error {
	fs := flag.NewFlagSet("idempotency", flag.ExitOnError)
	var cf commonFlags
	addCommonFlags(fs, &cf)
	if err := fs.Parse(args); err != nil {
		return err
	}

	client, boardID, err := newClient()
	if err != nil {
		return err
	}
	ctx := context.Background()

	w, err := openJSONL(cf.outDir, "idempotency.jsonl")
	if err != nil {
		return err
	}
	defer w.Close()

	page1, err := client.GetBoardActions(ctx, boardID, cf.limit, "")
	if err != nil {
		return err
	}
	meta1 := trello.SummarizePage(1, cf.limit, "", page1)
	emitLog(map[string]any{
		"event": "page", "pass": 1, "page": meta1.Page, "limit": meta1.Limit, "before": meta1.Before,
		"count": meta1.Count, "min_date": meta1.MinDate, "max_date": meta1.MaxDate,
		"first_id": meta1.FirstID, "last_id": meta1.LastID,
	})

	set1 := map[string]struct{}{}
	for _, a := range page1 {
		set1[a.ID] = struct{}{}
	}

	page2, err := client.GetBoardActions(ctx, boardID, cf.limit, "")
	if err != nil {
		return err
	}
	meta2 := trello.SummarizePage(1, cf.limit, "", page2)
	emitLog(map[string]any{
		"event": "page", "pass": 2, "page": meta2.Page, "limit": meta2.Limit, "before": meta2.Before,
		"count": meta2.Count, "min_date": meta2.MinDate, "max_date": meta2.MaxDate,
		"first_id": meta2.FirstID, "last_id": meta2.LastID,
	})

	overlap := 0
	only2 := 0
	for _, a := range page2 {
		if _, ok := set1[a.ID]; ok {
			overlap++
		} else {
			only2++
		}
	}
	only1 := len(set1) - overlap
	pct := 0.0
	if len(page2) > 0 {
		pct = 100.0 * float64(overlap) / float64(len(page2))
	}

	result := map[string]any{
		"kind": "idempotency_result",
		"pass1_count": len(page1), "pass2_count": len(page2),
		"overlap_ids": overlap, "only_pass1": only1, "only_pass2": only2,
		"overlap_percent_of_pass2": pct,
		"expect_full_overlap":      true,
		"dedupe_algorithm":         "in-memory set of action.id; insert-ignore / upsert by trello_action_id",
		"full_overlap":             only1 == 0 && only2 == 0 && len(page1) == len(page2),
	}
	_ = w.Write(result)
	emitLog(map[string]any{"event": "idempotency_done", "overlap_percent": pct, "full_overlap": result["full_overlap"]})
	return nil
}

type resumeState struct {
	BoardID      string   `json:"board_id"`
	Limit        int      `json:"limit"`
	NextBefore   string   `json:"next_before"` // cursor: last action id of last completed page
	PagesDone    int      `json:"pages_done"`
	ActionIDs    []string `json:"action_ids"`
	UpdatedAtUTC string   `json:"updated_at_utc"`
}

func runResume(args []string) error {
	fs := flag.NewFlagSet("resume", flag.ExitOnError)
	var cf commonFlags
	addCommonFlags(fs, &cf)
	stopAfter := fs.Int("stop-after", 2, "pages to fetch before saving cursor (phase 1); phase 2 continues")
	continueOnly := fs.Bool("continue", false, "if set, only continue from saved resume.json (skip phase 1)")
	if err := fs.Parse(args); err != nil {
		return err
	}

	client, boardID, err := newClient()
	if err != nil {
		return err
	}
	ctx := context.Background()

	if err := os.MkdirAll(cf.outDir, 0o755); err != nil {
		return err
	}
	resumePath := filepath.Join(cf.outDir, resumeFileName)

	w, err := openJSONL(cf.outDir, "resume.jsonl")
	if err != nil {
		return err
	}
	defer w.Close()

	var phase1IDs []string
	var nextBefore string
	pagesDone := 0

	if !*continueOnly {
		// Phase 1: fetch stopAfter pages, save cursor.
		before := ""
		for page := 1; page <= *stopAfter; page++ {
			actions, err := client.GetBoardActions(ctx, boardID, cf.limit, before)
			if err != nil {
				return err
			}
			meta := trello.SummarizePage(page, cf.limit, before, actions)
			emitLog(map[string]any{
				"event": "page", "phase": 1, "page": meta.Page, "limit": meta.Limit, "before": meta.Before,
				"count": meta.Count, "min_date": meta.MinDate, "max_date": meta.MaxDate,
				"first_id": meta.FirstID, "last_id": meta.LastID,
			})
			_ = w.Write(map[string]any{"kind": "page_meta", "phase": 1, "meta": meta})
			for _, a := range actions {
				phase1IDs = append(phase1IDs, a.ID)
			}
			pagesDone = page
			if len(actions) == 0 {
				nextBefore = before
				break
			}
			nextBefore = actions[len(actions)-1].ID
			before = nextBefore
			if len(actions) < cf.limit {
				break
			}
		}
		st := resumeState{
			BoardID:      boardID,
			Limit:        cf.limit,
			NextBefore:   nextBefore,
			PagesDone:    pagesDone,
			ActionIDs:    phase1IDs,
			UpdatedAtUTC: time.Now().UTC().Format(time.RFC3339),
		}
		if err := writeResume(resumePath, st); err != nil {
			return err
		}
		_ = w.Write(map[string]any{
			"kind": "resume_checkpoint", "path": resumePath,
			"pages_done": pagesDone, "ids_count": len(phase1IDs), "next_before": nextBefore,
		})
		emitLog(map[string]any{"event": "resume_checkpoint_saved", "pages_done": pagesDone, "ids": len(phase1IDs)})
	} else {
		st, err := readResume(resumePath)
		if err != nil {
			return err
		}
		phase1IDs = st.ActionIDs
		nextBefore = st.NextBefore
		pagesDone = st.PagesDone
		if st.Limit != 0 {
			cf.limit = st.Limit
		}
		_ = w.Write(map[string]any{"kind": "resume_loaded", "pages_done": pagesDone, "ids_count": len(phase1IDs), "next_before": nextBefore})
	}

	// Phase 2: continue from cursor for the same number of pages (or until short/empty).
	phase2IDs := []string{}
	before := nextBefore
	phase2Pages := *stopAfter
	if *continueOnly {
		phase2Pages = *stopAfter
	}
	for page := 1; page <= phase2Pages; page++ {
		if before == "" && pagesDone > 0 {
			// Empty cursor after exhausting history — nothing to continue.
			emitLog(map[string]any{"event": "resume_continue_skip", "reason": "empty_cursor"})
			break
		}
		actions, err := client.GetBoardActions(ctx, boardID, cf.limit, before)
		if err != nil {
			return err
		}
		absPage := pagesDone + page
		meta := trello.SummarizePage(absPage, cf.limit, before, actions)
		emitLog(map[string]any{
			"event": "page", "phase": 2, "page": meta.Page, "limit": meta.Limit, "before": meta.Before,
			"count": meta.Count, "min_date": meta.MinDate, "max_date": meta.MaxDate,
			"first_id": meta.FirstID, "last_id": meta.LastID,
		})
		_ = w.Write(map[string]any{"kind": "page_meta", "phase": 2, "meta": meta})
		for _, a := range actions {
			phase2IDs = append(phase2IDs, a.ID)
		}
		if len(actions) == 0 || len(actions) < cf.limit {
			break
		}
		before = actions[len(actions)-1].ID
	}

	// Continuous control fetch: same page count from start — compare id union.
	controlIDs := []string{}
	before = ""
	controlPageCount := pagesDone + uniquePagesEstimate(phase2IDs, cf.limit)
	if controlPageCount < pagesDone {
		controlPageCount = pagesDone
	}
	for page := 1; page <= controlPageCount; page++ {
		actions, err := client.GetBoardActions(ctx, boardID, cf.limit, before)
		if err != nil {
			return err
		}
		for _, a := range actions {
			controlIDs = append(controlIDs, a.ID)
		}
		if len(actions) == 0 || len(actions) < cf.limit {
			break
		}
		before = actions[len(actions)-1].ID
	}

	unionSet := map[string]struct{}{}
	for _, id := range phase1IDs {
		unionSet[id] = struct{}{}
	}
	dupBetweenPhases := 0
	for _, id := range phase2IDs {
		if _, ok := unionSet[id]; ok {
			dupBetweenPhases++
		}
		unionSet[id] = struct{}{}
	}
	controlSet := map[string]struct{}{}
	for _, id := range controlIDs {
		controlSet[id] = struct{}{}
	}

	missingFromUnion := 0
	for id := range controlSet {
		if _, ok := unionSet[id]; !ok {
			missingFromUnion++
		}
	}
	extraInUnion := 0
	for id := range unionSet {
		if _, ok := controlSet[id]; !ok {
			extraInUnion++
		}
	}

	result := map[string]any{
		"kind":                 "resume_result",
		"phase1_ids":           len(phase1IDs),
		"phase2_ids":           len(phase2IDs),
		"union_unique_ids":     len(unionSet),
		"dup_between_phases":   dupBetweenPhases,
		"control_ids":          len(controlIDs),
		"control_unique_ids":   len(controlSet),
		"missing_vs_control":   missingFromUnion,
		"extra_vs_control":     extraInUnion,
		"resume_consistent":    missingFromUnion == 0 && extraInUnion == 0 && dupBetweenPhases == 0,
		"cursor_file":          resumePath,
		"dedupe_note":          "union of action.id from interrupted+continued pages vs continuous fetch of same page count",
	}
	_ = w.Write(result)
	emitLog(map[string]any{
		"event": "resume_done", "resume_consistent": result["resume_consistent"],
		"union": len(unionSet), "missing": missingFromUnion, "extra": extraInUnion,
	})
	return nil
}

func uniquePagesEstimate(ids []string, limit int) int {
	if limit <= 0 {
		return 0
	}
	return (len(ids) + limit - 1) / limit
}

// preferActiveCards puts cards with dateLastActivity first (stable for the rest).
func preferActiveCards(in []trello.Card) []trello.Card {
	active := make([]trello.Card, 0, len(in))
	rest := make([]trello.Card, 0, len(in))
	for _, c := range in {
		if strings.TrimSpace(c.DateLastActivity) != "" {
			active = append(active, c)
		} else {
			rest = append(rest, c)
		}
	}
	return append(active, rest...)
}

func writeResume(path string, st resumeState) error {
	b, err := json.MarshalIndent(st, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, b, 0o644)
}

func readResume(path string) (resumeState, error) {
	var st resumeState
	b, err := os.ReadFile(path)
	if err != nil {
		return st, err
	}
	if err := json.Unmarshal(b, &st); err != nil {
		return st, err
	}
	return st, nil
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "…"
}

// loadDotEnv loads KEY=VALUE lines from path if present (no export, no secrets logged).
func loadDotEnv(path string) error {
	f, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	defer f.Close()
	r := bufio.NewReader(f)
	for {
		line, err := r.ReadString('\n')
		line = strings.TrimSpace(line)
		if line != "" && !strings.HasPrefix(line, "#") {
			if i := strings.IndexByte(line, '='); i > 0 {
				k := strings.TrimSpace(line[:i])
				v := strings.TrimSpace(line[i+1:])
				v = strings.Trim(v, `"'`)
				if os.Getenv(k) == "" {
					_ = os.Setenv(k, v)
				}
			}
		}
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
	}
	return nil
}
