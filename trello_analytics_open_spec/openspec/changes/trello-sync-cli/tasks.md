# Tasks

## Критерии приёмки change

1. Импорт одной доски запускается только через CLI (`cmd/trello-sync`).
2. CLI переиспользует importer/syncer: идемпотентность, resume, sync_runs, single-flight.
3. `POST /api/v1/sync/start` и `GET /api/v1/sync/status` не являются entrypoint (удалены или не выполняют sync).
4. In-server scheduler sync в `cmd/server` отключён/удалён.
5. CLI `status` показывает качество данных без секретов.
6. Docs §Operational sync обновлены под CLI-only; spike-факты не переписаны.

---

## 1. Backend (trello_analytics_back)

- [x] 1.1 Добавить `cmd/trello-sync`: subcommands `run` (синхронный sync до завершения) и `status` (stdout JSON без секретов); credentials/DB/board из env.
- [x] 1.2 Подключить существующий Orchestrator/BoardImporter; orphan reclaim; single-flight → ненулевой exit при already running.
- [x] 1.3 Удалить из `cmd/server` wiring sync orchestrator, `RunScheduler` и HTTP handlers `POST/GET /api/v1/sync/*`.
- [x] 1.4 Обновить `.env.example` / config: `TRELLO_SYNC_INTERVAL` не управляет server scheduler (убрать или пометить unused); тесты на CLI wiring и отсутствие sync routes.
- [x] 1.5 Регрессия: `go test ./...` зелёный; resume/idempotency importer не сломаны.

## 2. Frontend (trello_analytics_front)

- [x] 2.1 Нет обязательных задач.

## 3. Docs (trello_analytics_open_spec)

- [x] 3.1 Обновить `docs/trello-integration.md` §8: CLI-only entrypoint, примеры `run`/`status`; deprecate HTTP start/scheduler; не переписывать spike §3–§7.

## 4. Validation

- [x] 4.1 Сверить с критериями приёмки 1–6.
- [x] 4.2 Выполнить `openspec validate trello-sync-cli --store trello-analytics` без ошибок.
