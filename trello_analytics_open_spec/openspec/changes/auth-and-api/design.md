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

1. **Механизм сессии v1:** HTTP-only secure cookie с server-side session id в БД (таблица sessions) **или** signed JWT в cookie — выбрать cookie+server session для простого revoke на logout; зафиксировать в реализации один вариант.
2. **Пароли:** bcrypt/argon2id hash в `app_users.password_hash`.
3. **Bootstrap Admin:** env `BOOTSTRAP_ADMIN_EMAIL` + `BOOTSTRAP_ADMIN_PASSWORD` при пустой таблице users (одноразово); не публичный signup.
4. **RBAC:** middleware на каждом protected route; Member — read analytics/cards/reports; Team Lead — + workflow business settings; Admin — + users, integration secrets config, sync control.
5. **Секреты Trello в API:** GET settings маскирует token (`***` / отсутствует); PATCH secrets только Admin.
6. **Ошибки:** единый JSON error envelope; без stack traces и секретов клиенту.
7. **API контракты (обзор):**

| Method | Path | Роль (мин.) |
|--------|------|-------------|
| POST | `/api/v1/auth/login` | public |
| POST | `/api/v1/auth/logout` | authenticated |
| GET | `/api/v1/auth/me` | authenticated |
| GET | `/api/v1/analytics/overview` | Member+ |
| GET | `/api/v1/analytics/throughput` | Member+ |
| GET | `/api/v1/analytics/lead-time` | Member+ |
| GET | `/api/v1/analytics/cycle-time` | Member+ |
| GET | `/api/v1/analytics/bugs` | Member+ |
| GET | `/api/v1/analytics/reopened-bugs` | Member+ |
| GET | `/api/v1/analytics/reopened-bugs/fix-time` | Member+ |
| GET | `/api/v1/analytics/workload` | Member+ |
| GET | `/api/v1/analytics/stale-tasks` | Member+ |
| GET | `/api/v1/cards` | Member+ |
| GET | `/api/v1/cards/{id}` | Member+ |
| GET | `/api/v1/cards/{id}/history` | Member+ |
| GET | `/api/v1/reports/daily` | Member+ |
| GET | `/api/v1/reports/period` | Member+ |
| GET | `/api/v1/reports/preview` | Member+ |
| GET/PATCH | `/api/v1/settings` | GET Member+ (masked); PATCH Admin/Lead по полям |
| GET/PUT | `/api/v1/settings/workflow` | GET Member+; PUT Team Lead+ |
| POST | `/api/v1/sync/start` | Admin |
| GET | `/api/v1/sync/status` | Member+ |
| GET | `/api/v1/sync/history` | Admin/Lead |
| POST | `/api/v1/sync/retry` | Admin |
| CRUD users | `/api/v1/users` | Admin |

Query filters: `from`, `to`, `member_id`, `label_id`, timezone boundaries per settings.

## Risks / Trade-offs

- [Cookie CSRF] → Mitigation: SameSite=Lax/Strict + CSRF token если cross-site; same-origin через Nginx.
- [Широкий Member read] → Mitigation: ADR допускает; сузить позже без ломки ролей.

## Open Questions

- Точный package для OpenAPI generation — зафиксировать при apply (swagger/oapi-codegen или hand-written `docs/api.md`).
