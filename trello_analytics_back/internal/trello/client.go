package trello

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"
)

const (
	defaultBaseURL = "https://api.trello.com/1"
	defaultTimeout = 30 * time.Second
)

// Client is a thin read-only Trello REST client (stdlib only).
// Credentials must come from env / protected config — never hardcode or persist in DB.
type Client struct {
	HTTP    *http.Client
	BaseURL string
	Key     string
	Token   string

	// RetryMaxAttempts overrides default (5). Values <= 0 mean default.
	RetryMaxAttempts int
	// Sleep and Jitter are optional hooks for tests (nil → production defaults).
	Sleep  sleepFunc
	Jitter func(max time.Duration) time.Duration
}

// New creates a Client with a 30s timeout. Key/token must come from env/caller — never hardcode.
func New(key, token string) *Client {
	return &Client{
		HTTP: &http.Client{
			Timeout: defaultTimeout,
		},
		BaseURL: defaultBaseURL,
		Key:     key,
		Token:   token,
	}
}

// NewFromEnv builds a Client from TRELLO_API_KEY and TRELLO_TOKEN.
// Returns an error if either credential is missing (no silent empty auth).
func NewFromEnv() (*Client, error) {
	key := os.Getenv("TRELLO_API_KEY")
	token := os.Getenv("TRELLO_TOKEN")
	if key == "" || token == "" {
		return nil, fmt.Errorf("trello: TRELLO_API_KEY and TRELLO_TOKEN are required")
	}
	return New(key, token), nil
}

// GetBoard GET /boards/{id}
func (c *Client) GetBoard(ctx context.Context, boardID string) (*Board, error) {
	var b Board
	if err := c.getJSON(ctx, "/boards/"+url.PathEscape(boardID), nil, &b); err != nil {
		return nil, err
	}
	return &b, nil
}

// GetLists GET /boards/{id}/lists?filter=...
// filter: "all" | "open" | "closed" (empty → Trello default, typically open).
func (c *Client) GetLists(ctx context.Context, boardID string, filter string) ([]List, error) {
	q := url.Values{}
	if filter != "" {
		q.Set("filter", filter)
	}
	var lists []List
	if err := c.getJSON(ctx, "/boards/"+url.PathEscape(boardID)+"/lists", q, &lists); err != nil {
		return nil, err
	}
	return lists, nil
}

// GetMembers GET /boards/{id}/members
func (c *Client) GetMembers(ctx context.Context, boardID string) ([]Member, error) {
	q := url.Values{}
	q.Set("fields", "id,fullName,username,deactivated")
	var members []Member
	if err := c.getJSON(ctx, "/boards/"+url.PathEscape(boardID)+"/members", q, &members); err != nil {
		return nil, err
	}
	return members, nil
}

// GetLabels GET /boards/{id}/labels
func (c *Client) GetLabels(ctx context.Context, boardID string) ([]Label, error) {
	q := url.Values{}
	q.Set("fields", "id,idBoard,name,color")
	var labels []Label
	if err := c.getJSON(ctx, "/boards/"+url.PathEscape(boardID)+"/labels", q, &labels); err != nil {
		return nil, err
	}
	return labels, nil
}

// GetCards GET /boards/{id}/cards?filter=...
// filter: "all" | "open" | "closed" (Trello card filters).
// fields include idMembers/idLabels for junction sync; avoid nested members/labels payloads.
func (c *Client) GetCards(ctx context.Context, boardID, filter string) ([]Card, error) {
	q := url.Values{}
	if filter != "" {
		q.Set("filter", filter)
	}
	q.Set("fields", "id,name,closed,idList,idBoard,dateLastActivity,idMembers,idLabels")
	var cards []Card
	if err := c.getJSON(ctx, "/boards/"+url.PathEscape(boardID)+"/cards", q, &cards); err != nil {
		return nil, err
	}
	return cards, nil
}

// GetBoardActions GET /boards/{id}/actions?limit=&before=
func (c *Client) GetBoardActions(ctx context.Context, boardID string, limit int, before string) ([]Action, error) {
	return c.getActions(ctx, "/boards/"+url.PathEscape(boardID)+"/actions", limit, before)
}

// GetCardActions GET /cards/{id}/actions?limit=&before=
func (c *Client) GetCardActions(ctx context.Context, cardID string, limit int, before string) ([]Action, error) {
	return c.getActions(ctx, "/cards/"+url.PathEscape(cardID)+"/actions", limit, before)
}

func (c *Client) getActions(ctx context.Context, path string, limit int, before string) ([]Action, error) {
	q := url.Values{}
	if limit > 0 {
		q.Set("limit", strconv.Itoa(limit))
	}
	if before != "" {
		q.Set("before", before)
	}
	var actions []Action
	if err := c.getJSON(ctx, path, q, &actions); err != nil {
		return nil, err
	}
	return actions, nil
}

// DoRaw performs GET path with optional query (auth injected). Used by spike/errors mode.
// Does not retry — callers that need retry use JSON helpers (getJSON).
// Returns status, body, redacted URL string.
func (c *Client) DoRaw(ctx context.Context, path string, extra url.Values, overrideToken string) (status int, body []byte, redactedURL string, err error) {
	status, body, redactedURL, _, err = c.doOnce(ctx, path, extra, overrideToken)
	return status, body, redactedURL, err
}

func (c *Client) doOnce(ctx context.Context, path string, extra url.Values, overrideToken string) (status int, body []byte, redactedURL string, retryAfter time.Duration, err error) {
	u, err := c.buildURL(path, extra, overrideToken)
	if err != nil {
		return 0, nil, "", 0, err
	}
	redactedURL = RedactURL(u.String(), c.Key, c.Token, overrideToken)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return 0, nil, redactedURL, 0, err
	}
	resp, err := c.HTTP.Do(req)
	if err != nil {
		return 0, nil, redactedURL, 0, err
	}
	defer resp.Body.Close()
	retryAfter = parseRetryAfter(resp.Header.Get("Retry-After"))
	// Boards can return large card/action payloads; 1 MiB truncates JSON mid-stream.
	const maxBody = 32 << 20
	body, err = io.ReadAll(io.LimitReader(resp.Body, maxBody+1))
	if err != nil {
		return resp.StatusCode, nil, redactedURL, retryAfter, err
	}
	if len(body) > maxBody {
		return resp.StatusCode, nil, redactedURL, retryAfter, fmt.Errorf("trello response exceeds %d bytes (url=%s)", maxBody, redactedURL)
	}
	return resp.StatusCode, body, redactedURL, retryAfter, nil
}

func parseRetryAfter(h string) time.Duration {
	h = strings.TrimSpace(h)
	if h == "" {
		return 0
	}
	if secs, err := strconv.Atoi(h); err == nil && secs >= 0 {
		return time.Duration(secs) * time.Second
	}
	if t, err := http.ParseTime(h); err == nil {
		d := time.Until(t)
		if d < 0 {
			return 0
		}
		return d
	}
	return 0
}

func (c *Client) getJSON(ctx context.Context, path string, extra url.Values, dest any) error {
	return c.withRetry(ctx, func() error {
		return c.getJSONOnce(ctx, path, extra, dest)
	})
}

func (c *Client) getJSONOnce(ctx context.Context, path string, extra url.Values, dest any) error {
	status, body, redacted, retryAfter, err := c.doOnce(ctx, path, extra, "")
	if err != nil {
		return err
	}
	if status < 200 || status >= 300 {
		return mapAPIError(&APIError{
			StatusCode: status,
			Status:     fmt.Sprintf("%d", status),
			Body:       truncate(string(body), 200),
			URL:        redacted,
			RetryAfter: retryAfter,
		})
	}
	if err := json.Unmarshal(body, dest); err != nil {
		return fmt.Errorf("decode json: %w (url=%s)", err, redacted)
	}
	return nil
}

func (c *Client) buildURL(path string, extra url.Values, overrideToken string) (*url.URL, error) {
	base := strings.TrimRight(c.BaseURL, "/")
	u, err := url.Parse(base + path)
	if err != nil {
		return nil, err
	}
	q := u.Query()
	for k, vs := range extra {
		for _, v := range vs {
			q.Add(k, v)
		}
	}
	q.Set("key", c.Key)
	token := c.Token
	if overrideToken != "" {
		token = overrideToken
	}
	q.Set("token", token)
	u.RawQuery = q.Encode()
	return u, nil
}

// SummarizePage builds pagination observation for a fetched actions page.
func SummarizePage(page, limit int, before string, actions []Action) PageMeta {
	m := PageMeta{
		Page:   page,
		Limit:  limit,
		Before: before,
		Count:  len(actions),
	}
	if len(actions) == 0 {
		return m
	}
	m.FirstID = actions[0].ID
	m.LastID = actions[len(actions)-1].ID
	minD, maxD := actions[0].Date, actions[0].Date
	for _, a := range actions[1:] {
		if a.Date < minD {
			minD = a.Date
		}
		if a.Date > maxD {
			maxD = a.Date
		}
	}
	m.MinDate = minD
	m.MaxDate = maxD
	return m
}

// RedactURL replaces key/token query values with *** so logs stay safe.
func RedactURL(raw string, secrets ...string) string {
	out := raw
	for _, s := range secrets {
		if s == "" {
			continue
		}
		out = strings.ReplaceAll(out, url.QueryEscape(s), "***")
		out = strings.ReplaceAll(out, s, "***")
	}
	// Also strip common query keys if still present unescaped somehow.
	if u, err := url.Parse(out); err == nil {
		q := u.Query()
		if q.Has("key") {
			q.Set("key", "***")
		}
		if q.Has("token") {
			q.Set("token", "***")
		}
		u.RawQuery = q.Encode()
		return u.String()
	}
	return out
}

// MinimizeAction returns a redacted/minimal action view for JSONL (less PII).
func MinimizeAction(a Action) map[string]any {
	out := map[string]any{
		"id":   a.ID,
		"type": a.Type,
		"date": a.Date,
	}
	if a.IDMemberCreator != "" {
		out["idMemberCreator"] = a.IDMemberCreator
	}
	if a.Data != nil {
		out["data"] = minimizeData(a.Data)
	}
	return out
}

func minimizeData(data map[string]any) map[string]any {
	// Keep structural ids/names useful for metrics research; drop long text/comments bodies.
	keep := map[string]any{}
	for k, v := range data {
		switch k {
		case "text", "textData":
			keep[k] = "[redacted]"
		case "card", "list", "listBefore", "listAfter", "board", "label", "member", "old", "checklist", "checkItem":
			keep[k] = scrubNested(v)
		default:
			keep[k] = scrubNested(v)
		}
	}
	return keep
}

func scrubNested(v any) any {
	switch t := v.(type) {
	case map[string]any:
		out := make(map[string]any, len(t))
		for k, vv := range t {
			if k == "desc" || k == "text" {
				out[k] = "[redacted]"
				continue
			}
			// Prefer id/name/closed/idList fields; keep others but scrub deep strings lightly.
			out[k] = scrubNested(vv)
		}
		return out
	case []any:
		out := make([]any, len(t))
		for i, vv := range t {
			out[i] = scrubNested(vv)
		}
		return out
	case string:
		if len(t) > 120 {
			return t[:120] + "…"
		}
		return t
	default:
		return v
	}
}
