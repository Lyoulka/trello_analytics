# Design

## Context

См. proposal.md. ADR-012/013: роли Admin/Team Lead/Member; локальные пароли; bootstrap Admin; без публичной регистрации. Метрики и sync — сервисный слой из предыдущих changes.

## Goals / Non-Goals

**Goals:**

- Auth endpoints + session/token security.
- REST `/api/v1` для analytics, cards, reports, settings, sync, users (Admin).
- OpenAPI / `docs/api.md`.

**Non-Goals:**

- Vue UI, OAuth/IdP, публичная регистрация.
- Изменение формул метрик.

## Decisions

1. **Механизм сессии v1 (зафиксировано):** HTTP-only cookie `tta_session` + server-side `sessions` в PostgreSQL (SHA-256 hash cookie value; absolute TTL default 7d). **Не JWT.** Logout = revoke row; deactivate = revoke all sessions.
2. **Пароли:** bcrypt cost 12 в `app_users.password_hash`.
3. **Bootstrap Admin:** env `BOOTSTRAP_ADMIN_EMAIL` + `BOOTSTRAP_ADMIN_PASSWORD` при пустой таблице users (одноразово); не публичный signup.
4. **RBAC:** middleware на каждом protected route; Member — read analytics/cards/reports; Team Lead — + workflow / non-secret settings; Admin — + users, sync control. Trello secrets only via env (PATCH secrets → `secrets_via_env`).
5. **Секреты Trello в API:** GET settings возвращает только `trello_*_configured` flags; plaintext key/token never.
6. **Ошибки:** единый JSON error envelope; без stack traces и секретов клиенту.
7. **API docs SSOT:** hand-written `docs/api.md` (no swagger in v1).
8. **API контракты (обзор):** полный перечень — `docs/api.md`. Кратко: auth public login; Member+ analytics/cards/reports/settings GET/sync status; Team Lead+ settings PATCH + workflow PUT + sync history; Admin users + sync start/retry.

Query filters: `from`, `to`, `member_id`, `label_id` (supported on analytics/cards).

## Risks / Trade-offs

- [Cookie CSRF] → Mitigation: SameSite=Lax/Strict + CSRF token если cross-site; same-origin через Nginx.
- [Широкий Member read] → Mitigation: ADR допускает; сузить позже без ломки ролей.

## Open Questions

- (closed) API docs: hand-written `docs/api.md`; no swagger in v1.
- (closed) Session: cookie + PostgreSQL sessions (not JWT).
