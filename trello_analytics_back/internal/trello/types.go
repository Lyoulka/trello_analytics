package trello

// Board is a minimal board metadata payload from Trello.
type Board struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Closed      bool   `json:"closed"`
	DateLastActivity string `json:"dateLastActivity,omitempty"`
	URL         string `json:"url,omitempty"`
}

// List is a board list (column).
type List struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Closed   bool   `json:"closed"`
	IDBoard  string `json:"idBoard"`
	Pos      float64 `json:"pos"`
}

// Card is a board card (may be archived when Closed=true).
type Card struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Closed   bool   `json:"closed"`
	IDList   string `json:"idList"`
	IDBoard  string `json:"idBoard"`
	DateLastActivity string `json:"dateLastActivity,omitempty"`
}

// Action is a Trello action/event. Payload kept as raw map for spike inspection.
type Action struct {
	ID              string         `json:"id"`
	IDMemberCreator string         `json:"idMemberCreator,omitempty"`
	Type            string         `json:"type"`
	Date            string         `json:"date"`
	Data            map[string]any `json:"data,omitempty"`
}

// PageMeta is pagination observation for one actions page (logged / JSONL).
type PageMeta struct {
	Page    int    `json:"page"`
	Limit   int    `json:"limit"`
	Before  string `json:"before,omitempty"`
	Count   int    `json:"count"`
	MinDate string `json:"min_date,omitempty"`
	MaxDate string `json:"max_date,omitempty"`
	FirstID string `json:"first_id,omitempty"`
	LastID  string `json:"last_id,omitempty"`
}

// APIError is a non-2xx Trello response with redacted URL.
type APIError struct {
	StatusCode int
	Status     string
	Body       string
	URL        string // secrets redacted
}

func (e *APIError) Error() string {
	return "trello api: " + e.Status + " url=" + e.URL + " body=" + truncate(e.Body, 200)
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "…"
}
