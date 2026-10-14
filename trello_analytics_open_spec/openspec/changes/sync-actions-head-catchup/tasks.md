# Tasks: sync-actions-head-catchup

## 1. Docs (trello_analytics_open_spec)

- [x] 1.1 Proposal / design / delta spec + этот `tasks.md`.
- [x] 1.2 Обновить `docs/trello-integration.md` §8.4.1: head catch-up на каждом sync после entities; курсор oldest не трогать.

## 2. Backend (trello_analytics_back)

- [x] 2.1 `InsertCardActionsPage` (insert-only, без upsert курсора); расширить `BoardActionsStore`.
- [x] 2.2 `ImportBoardActionsHeadCatchUp` + вызов из `ImportBoard` (gate count > 0 → head → primary).
- [x] 2.3 Убрать warning «no head catch-up» из `defaultWarnings`; поправить `status_test`.
- [x] 2.4 Тесты: head вставляет новые id / курсор не меняется / stop на duplicates / skip на пустой БД.
