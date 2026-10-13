# Proposal

## Why

Импорт доски Trello в PostgreSQL уже реализован в change `trello-import-sync`, но запуск идёт через HTTP (`POST /api/v1/sync/start`) и in-server scheduler. Для ops и локальной разработки нужен **единственный** entrypoint — консольная команда Go; HTTP start и фоновый sync в процессе API-сервера не должны оставаться способом импорта.

## What Changes

- Добавить production CLI (`cmd/trello-sync` или эквивалент) для запуска импорта/sync одной доски с credentials и DB из env.
- CLI MUST переиспользовать существующий importer/syncer (идемпотентность, resume, sync_runs, single-flight) — без дублирования spike CLI.
- **BREAKING:** deprecate и удалить из product path `POST /api/v1/sync/start` и in-server scheduler (`TRELLO_SYNC_INTERVAL` в `cmd/server`).
- Статус качества данных — через CLI (stdout / subcommand `status`); HTTP `GET /api/v1/sync/status` deprecate/удалить из product path (полный auth-gated status API — в `auth-and-api`, если понадобится позже).
- Обновить `docs/trello-integration.md` §Operational sync под CLI-only.
- Spike CLI (`cmd/trello-spike`) остаётся research tooling, не sync entrypoint.

## Capabilities

### New Capabilities

- `trello-sync`: запуск импорта/синхронизации одной доски Trello в PostgreSQL **только через CLI**, с журналом runs, single-flight и наблюдаемым статусом без утечки секретов; HTTP start/status и in-server scheduler не являются entrypoint.

### Modified Capabilities

- (нет в main specs; supersede operational entrypoint из change `trello-import-sync`)

## Impact

**Затронутые репозитории:**

- `trello_analytics_back` — новый `cmd/…` CLI; отключение/удаление sync HTTP handlers и scheduler из `cmd/server`; wiring существующего `internal/syncer` / `internal/importer`.
- `trello_analytics_open_spec` — artifacts этого change; правка `docs/trello-integration.md` §8.
- `trello_analytics_front` — **вне scope**.

**Вне scope:** формулы метрик, Dashboard UI, auth, head catch-up actions, production deploy, rewrite spike CLI.

**Зависимость:** `trello-import-sync` (importer/syncer/storage уже есть).
