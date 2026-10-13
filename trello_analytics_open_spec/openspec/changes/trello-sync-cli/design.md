# Design

## Context

Change `trello-import-sync` реализовал importer, syncer, `sync_runs`/`sync_cursors`, retry и HTTP `POST/GET /api/v1/sync/*` плюс goroutine-scheduler в `cmd/server`. Продуктовое требование скорректировано: **импорт только через консольную команду**. HTTP start/status и in-server scheduler больше не entrypoint.

## Goals / Non-Goals

**Goals:**

- Единый production CLI для старта sync и чтения статуса качества данных.
- Переиспользование `internal/importer` + `internal/syncer` (single-flight, sync_runs, resume).
- Убрать из `cmd/server` автозапуск sync и HTTP sync endpoints.
- Документировать CLI в `docs/trello-integration.md` без переписывания spike-фактов.

**Non-Goals:**

- Новый sync engine / head catch-up.
- Auth-gated REST sync API (`auth-and-api`).
- Frontend UI статуса.
- Замена `cmd/trello-spike`.

## Decisions

1. **Binary:** `cmd/trello-sync` в `trello_analytics_back`.
2. **Subcommands (минимально):**
   - `run` (или default без args) — синхронный прогон импорта до завершения; exit `0` при success, ненулевой при failed/cancelled/ошибке старта.
   - `status` — печать агрегированного статуса (те же поля смысла, что у прежнего status JSON) в stdout **без** key/token; предпочтительно JSON для машинной читаемости.
3. **Config/env:** `DATABASE_URL`, `TRELLO_API_KEY`, `TRELLO_TOKEN`, `TRELLO_BOARD_ID`, `TRELLO_ACTIONS_PAGE_SIZE` (как сейчас). `TRELLO_SYNC_INTERVAL` **не** используется CLI и **не** запускает sync в server.
4. **Reuse:** CLI собирает store + Trello client + `BoardImporter` + `Orchestrator` так же, как раньше `cmd/server`; вызывает синхронный start/wait (не async HTTP 202). Single-flight и orphan reclaim сохраняются.
5. **Deprecate HTTP sync:** удалить `POST /api/v1/sync/start` и `GET /api/v1/sync/status` из server mux; убрать передачу orchestrator в HTTP server для sync. Health/ready остаются.
6. **Deprecate in-server scheduler:** не вызывать `RunScheduler` из `cmd/server`; не поднимать sync orchestrator в server process (если нужен только health — sync wiring только в CLI).
7. **Spike:** `cmd/trello-spike` без изменений контракта; не пишет operational sync path.

## Risks / Trade-offs

- [Нет фонового sync в API-процессе] → Mitigation: ops запускает CLI по cron/systemd timer (вне scope этого change зафиксировать только entrypoint).
- [Долгий sync блокирует CLI процесс] → Ожидаемо; не HTTP timeout.
- [Регрессия для тех, кто уже дергал HTTP start] → **BREAKING**; документировать в §Operational sync.

## Migration Plan

1. Добавить CLI + тесты.
2. Удалить sync routes и scheduler из server.
3. Обновить docs / `.env.example` (убрать или пометить `TRELLO_SYNC_INTERVAL` как unused для server).
4. Локально: `go run ./cmd/trello-sync run` вместо `curl POST …/sync/start`.

## Open Questions

- Нужен ли systemd timer unit в `production-deploy` — вне этого change (только упомянуть в docs как возможный ops pattern).
