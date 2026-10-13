# REST API (`/api/v1`)

SSOT hand-written contracts for Trello Team Analytics (change `auth-and-api`).  
No OpenAPI/Swagger in v1. JSON field names are **snake_case**.

Base URL (local): `http://localhost:8080`. Frontend proxies `/api` via Vite.

## Auth model

- **Session:** HTTP-only cookie `tta_session` + server-side rows in `sessions`.
- Cookie value is random; DB stores **SHA-256 hash** only. Absolute TTL default **7d** (`SESSION_TTL`).
- Cookie flags: `HttpOnly`; `Secure` when `APP_ENV != development`; `SameSite=Lax`; `Path=/`.
- **Not JWT.** Logout revokes the session row; deactivate user revokes all sessions.
- Passwords: **bcrypt cost 12**. Never returned to clients.
- Roles: `admin` | `team_lead` | `member` (hierarchy Admin ⊃ Team Lead ⊃ Member).
- No public registration. First Admin via bootstrap env when `app_users` is empty.

### Public (no cookie)

| Method | Path | Notes |
|--------|------|--------|
| POST | `/api/v1/auth/login` | body `{email,password}` → sets cookie + user JSON |
| GET | `/health` | liveness |
| GET | `/ready` | readiness + DB |
| GET | `/api/v1/health` | same as `/ready` |

### Authenticated

| Method | Path | Min role |
|--------|------|----------|
| POST | `/api/v1/auth/logout` | Member+ |
| GET | `/api/v1/auth/me` | Member+ |

## Error envelope

```json
{"error":{"code":"...","message":"..."}}
```

- `401` — unauthenticated (`unauthenticated` / `invalid_credentials`)
- `403` — authenticated but insufficient role (`forbidden`)
- No stack traces, SQL, or secrets in responses.

## CORS

`FRONTEND_ORIGIN` (default `http://localhost:5173`) + `Access-Control-Allow-Credentials: true`.  
Wildcard `*` is not used (cookie auth).

## Board scope

Single board from `TRELLO_BOARD_ID` / `boards` table. Query `board_id` is **not** required.  
If the board is not imported yet → `503` `board_not_synced`.

## Users (Admin)

| Method | Path | Notes |
|--------|------|--------|
| GET | `/api/v1/users` | list |
| POST | `/api/v1/users` | `{email,password,display_name,role}` |
| PATCH | `/api/v1/users/{id}` | `role`, `display_name`, `is_active`, optional `password` |

No DELETE. Deactivate with `is_active=false` or password change → revoke all sessions. Passwords must be ≥12 characters.

## Trello members (board participants)

| Method | Path | Min role | Notes |
|--------|------|----------|--------|
| GET | `/api/v1/members` | Member+ | list imported Trello members: `id`, `trello_member_id`, `full_name`, `username`, `is_active` |
| PATCH | `/api/v1/members/{id}` | Team Lead+ | body `{ "is_active": bool }` — include/exclude from analytics |

`members.is_active` is an app analytics flag (default `true`). Trello sync does **not** overwrite it. When `is_active=false`, the member is omitted from workload; their assignments are ignored; cards whose current assignees are all inactive are excluded from analytics/reports metrics. Raw `GET /cards` browse is unchanged.

## Trello lists (columns)

| Method | Path | Min role | Notes |
|--------|------|----------|--------|
| GET | `/api/v1/lists` | Member+ | imported columns: `id`, `trello_list_id`, `name`, `is_archived`, `is_active`, `logical_stage` |
| PATCH | `/api/v1/lists/{id}` | Team Lead+ | body `{ "is_active": bool }` |

`trello_lists.is_active` defaults to `true`. Sync does **not** overwrite it. Cards whose **current** list has `is_active=false` are excluded from analytics/reports/charts. Raw `GET /cards` browse is unchanged.

## Analytics (Member+)

All require `from` and `to` (RFC3339 or `YYYY-MM-DD`). Optional `member_id`, `label_id` (internal UUIDs).

| Method | Path |
|--------|------|
| GET | `/api/v1/analytics/overview` |
| GET | `/api/v1/analytics/throughput` |
| GET | `/api/v1/analytics/lead-time` |
| GET | `/api/v1/analytics/cycle-time` |
| GET | `/api/v1/analytics/bugs` |
| GET | `/api/v1/analytics/reopened-bugs` |
| GET | `/api/v1/analytics/reopened-bugs/fix-time` |
| GET | `/api/v1/analytics/workload` |
| GET | `/api/v1/analytics/stale-tasks` |

Workload entries: `member_id`, optional `full_name` (from `members.full_name` / `username`), `active_cards`, `completed_cards`. Overview embeds the same `workload[]`.

Handlers call `analytics.Engine.Compute` and project fields; responses include `warnings` (incomplete history / workflow gaps — never invented timestamps).

Date-only `to` is treated as **exclusive next midnight UTC**.

## Cards (Member+)

| Method | Path | Notes |
|--------|------|--------|
| GET | `/api/v1/cards` | pagination `limit`/`offset`; sort `name\|updated\|created`; filters `member_id`, `label_id`, `list_id`, `archived`, `q`, `updated_from`, `updated_to` (date/RFC3339 on `last_trello_update_at`; date-only `to` = exclusive next midnight UTC). Items include optional `last_updated_by_member_id` / `last_updated_by_name` from latest action `idMemberCreator`. |
| GET | `/api/v1/cards/{id}` | detail |
| GET | `/api/v1/cards/{id}/history` | chronological normalized events; **no** `raw_payload`. Events include optional `member_name`, `source_list_name`, `destination_list_name` resolved from action payload / members / lists (FK columns may be null after import). |

## Reports (Member+)

| Method | Path | Notes |
|--------|------|--------|
| GET | `/api/v1/reports/daily` | same shape as period; requires `from`/`to` |
| GET | `/api/v1/reports/period` | ТЗ §6.3 JSON: summary, completed_tasks, bugs, current_work, problems, warnings; optional `deltas` vs previous window (`completed_delta`, `reopened_delta` only — no `wip_delta`, WIP is a current-board snapshot). Summary counts (`completed_count`, `bugs_fixed_count`, `reopened_count`, `wip_count`) and delta fields are JSON `null` when the analytics Engine marks the metric `Available=false` (e.g. incomplete workflow) — never a trustworthy `0`. Report cards include `member_ids` and `member_names` (display names for current assignees). |
| GET | `/api/v1/reports/preview` | same shape as period |

## Settings

| Method | Path | Min role | Notes |
|--------|------|----------|--------|
| GET | `/api/v1/settings` | Member+ | whitelisted non-secret keys only + `trello_api_key_configured` / `trello_token_configured` / `board_id` — **never** plaintext key/token |
| PATCH | `/api/v1/settings` | Team Lead+ | same whitelist as GET; Trello secrets in body → `400` `secrets_via_env` |
| GET | `/api/v1/settings/workflow` | Member+ | workflow settings |
| PUT | `/api/v1/settings/workflow` | Team Lead+ | full replace via `workflow.Service` |

## Sync (dual entrypoint)

HTTP and CLI (`cmd/trello-sync`) share `internal/syncer.Orchestrator`.

| Method | Path | Min role | Notes |
|--------|------|----------|--------|
| POST | `/api/v1/sync/start` | Admin | `202` `{run_id}` or `409` already running |
| GET | `/api/v1/sync/status` | Member+ | quality status, no secrets |
| GET | `/api/v1/sync/history` | Team Lead+ | recent `sync_runs` |
| POST | `/api/v1/sync/retry` | Admin | same as start |

Requires Trello env on the API process; otherwise `503` `sync_unavailable`.

## Bootstrap env

| Variable | Purpose |
|----------|---------|
| `BOOTSTRAP_ADMIN_EMAIL` | first Admin email when `app_users` empty |
| `BOOTSTRAP_ADMIN_PASSWORD` | first Admin password |
| `FRONTEND_ORIGIN` | CORS origin |
| `SESSION_TTL` | Go duration (default `168h`) |
| `APP_ENV` | `development` → cookie without Secure |

If `app_users` is empty and bootstrap env is missing → Warn log; server still starts.
