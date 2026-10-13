package handlers

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"strings"

	"trello_analytics_back/internal/auth"
	"trello_analytics_back/internal/config"
	"trello_analytics_back/internal/storage/postgres"
	"trello_analytics_back/internal/workflow"
)

// Settings serves GET/PATCH /settings and GET/PUT /settings/workflow.
type Settings struct {
	Store    *postgres.Store
	Workflow *workflow.Service
	Cfg      config.Config
	Logger   *slog.Logger
}

// Whitelisted non-secret app_settings keys (Team Lead+).
var settingsWhitelist = map[string]struct{}{
	"timezone_display":       {},
	"default_period_days":    {},
	"report_title":           {},
	"ui_locale":              {},
	"stale_threshold_override_note": {},
}

// Get returns non-secret settings + trello_*_configured flags (never plaintext key/token).
func (h *Settings) Get(w http.ResponseWriter, r *http.Request) {
	rows, err := h.Store.ListAppSettings(r.Context())
	if err != nil {
		h.warn(err)
		WriteInternal(w)
		return
	}
	// Same whitelist as PATCH — do not leak unknown/legacy app_settings keys.
	settings := map[string]string{}
	for _, row := range rows {
		if _, allowed := settingsWhitelist[row.Key]; !allowed {
			continue
		}
		settings[row.Key] = row.Value
	}
	boardID := ""
	if b, err := h.Store.GetBoardByTrelloID(r.Context(), h.Cfg.TrelloBoardID); err == nil && b != nil {
		boardID = b.ID
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"settings":                 settings,
		"board_id":                 boardID,
		"trello_board_id":          h.Cfg.TrelloBoardID,
		"trello_api_key_configured": h.Cfg.TrelloAPIKey != "",
		"trello_token_configured":  h.Cfg.TrelloToken != "",
	})
}

// Patch updates whitelisted non-secret keys. Trello secrets in body → 400 secrets_via_env.
func (h *Settings) Patch(w http.ResponseWriter, r *http.Request) {
	u, ok := auth.UserFromContext(r.Context())
	if !ok {
		WriteUnauthorized(w)
		return
	}
	var body map[string]json.RawMessage
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeBadRequest(w, "invalid_body", "invalid JSON body")
		return
	}
	secretKeys := []string{
		"trello_api_key", "trello_token", "trello_key", "api_key", "token",
		"TRELLO_API_KEY", "TRELLO_TOKEN",
	}
	for _, sk := range secretKeys {
		if _, exists := body[sk]; exists {
			writeBadRequest(w, "secrets_via_env", "Trello secrets must be set via environment variables, not API")
			return
		}
	}
	// Also reject nested "settings" map containing secrets.
	if raw, ok := body["settings"]; ok {
		var nested map[string]string
		if err := json.Unmarshal(raw, &nested); err == nil {
			for k := range nested {
				lk := strings.ToLower(k)
				if strings.Contains(lk, "token") || strings.Contains(lk, "api_key") || strings.Contains(lk, "secret") {
					writeBadRequest(w, "secrets_via_env", "Trello secrets must be set via environment variables, not API")
					return
				}
			}
			for k, v := range nested {
				if _, allowed := settingsWhitelist[k]; !allowed {
					writeBadRequest(w, "unsupported_setting", "setting key not allowed: "+k)
					return
				}
				uid := u.ID
				if err := h.Store.UpsertAppSetting(r.Context(), k, v, &uid); err != nil {
					h.warn(err)
					WriteInternal(w)
					return
				}
			}
			h.Get(w, r)
			return
		}
	}
	// Flat key/value patch of whitelisted keys.
	for k, raw := range body {
		if k == "settings" {
			continue
		}
		if _, allowed := settingsWhitelist[k]; !allowed {
			writeBadRequest(w, "unsupported_setting", "setting key not allowed: "+k)
			return
		}
		var v string
		if err := json.Unmarshal(raw, &v); err != nil {
			writeBadRequest(w, "invalid_body", "setting values must be strings")
			return
		}
		uid := u.ID
		if err := h.Store.UpsertAppSetting(r.Context(), k, v, &uid); err != nil {
			h.warn(err)
			WriteInternal(w)
			return
		}
	}
	h.Get(w, r)
}

// GetWorkflow returns workflow settings for the configured board.
func (h *Settings) GetWorkflow(w http.ResponseWriter, r *http.Request) {
	boardID, err := resolveBoardID(r.Context(), h.Store, h.Cfg.TrelloBoardID)
	if err != nil {
		writeBoardResolveError(w, err)
		return
	}
	s, err := h.Workflow.Load(r.Context(), boardID)
	if err != nil {
		h.warn(err)
		WriteInternal(w)
		return
	}
	writeJSON(w, http.StatusOK, s)
}

// PutWorkflow replaces workflow settings (Team Lead+).
func (h *Settings) PutWorkflow(w http.ResponseWriter, r *http.Request) {
	u, ok := auth.UserFromContext(r.Context())
	if !ok {
		WriteUnauthorized(w)
		return
	}
	boardID, err := resolveBoardID(r.Context(), h.Store, h.Cfg.TrelloBoardID)
	if err != nil {
		writeBoardResolveError(w, err)
		return
	}
	var in workflow.Settings
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		writeBadRequest(w, "invalid_body", "invalid JSON body")
		return
	}
	uid := u.ID
	saved, err := h.Workflow.Save(r.Context(), boardID, in, &uid)
	if err != nil {
		if strings.Contains(err.Error(), "workflow:") {
			writeBadRequest(w, "invalid_workflow", "invalid workflow settings")
			return
		}
		h.warn(err)
		WriteInternal(w)
		return
	}
	writeJSON(w, http.StatusOK, saved)
}

func (h *Settings) warn(err error) {
	if h.Logger != nil {
		h.Logger.Warn("settings handler", "err", err)
	}
}
