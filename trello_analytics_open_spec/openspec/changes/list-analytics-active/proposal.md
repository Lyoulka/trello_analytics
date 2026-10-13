# Change: Управление активностью колонок (списков) Trello в аналитике

## Why

Часть колонок доски (парковки, Icebox, служебные списки) не должны влиять на метрики. Нужен продуктовый флаг `is_active` на `trello_lists`: при `false` карточки, **сейчас** находящиеся в такой колонке, исключаются из analytics/reports/charts.

## What Changes

- Колонка `trello_lists.is_active BOOLEAN NOT NULL DEFAULT true` (миграция).
- Sync **не** перезаписывает `is_active` при upsert списка.
- API: `GET /api/v1/lists`, `PATCH /api/v1/lists/{id}` `{ "is_active": bool }` (Team Lead+ на PATCH).
- Engine: карточки с `current_list_id` → список с `is_active=false` не входят в Compute (и их transitions).
- Карточки без `current_list_id` сохраняют текущее поведение.
- Frontend: страница/таблица колонок с переключателем (по аналогии с участниками).
- Docs: `docs/api.md`, `docs/database.md`.

## Impact

- `trello_analytics_open_spec` — этот change + docs
- `trello_analytics_back` — migration, store, engine, handlers
- `trello_analytics_front` — UI колонок
