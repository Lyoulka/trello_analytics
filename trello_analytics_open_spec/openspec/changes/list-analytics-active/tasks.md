# Tasks

## Критерии приёмки

1. У колонки можно выставить `is_active` в UI.
2. Карточки в колонке с `is_active=false` не входят в analytics/reports/charts (WIP, stale, workload, completed и т.д.).
3. Sync не сбрасывает ручной `is_active`.
4. Тесты + docs обновлены.

## 1. Docs (trello_analytics_open_spec)

- [x] 1.1 Delta specs + `docs/api.md` + `docs/database.md`.

## 2. Backend (trello_analytics_back)

- [x] 2.1 Миграция `trello_lists.is_active`; UpsertLists не перезаписывает флаг.
- [x] 2.2 Store ListLists / SetListActive; handlers GET/PATCH `/api/v1/lists`; RBAC.
- [x] 2.3 Engine/load: исключать карточки текущей неактивной колонки.
- [x] 2.4 Unit/handler tests.

## 3. Frontend (trello_analytics_front)

- [x] 3.1 API client + типы.
- [x] 3.2 Страница колонок с toggle, route + nav.
- [x] 3.3 Минимальные тесты.
