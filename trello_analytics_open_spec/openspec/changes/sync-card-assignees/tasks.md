# Tasks

## Критерии приёмки

1. После sync при карточках с members в Trello `card_members` содержит активные связи.
2. Workload API возвращает записи по участникам (при наличии assignees).
3. Повторный sync идемпотентен; снятый в Trello member → не active в `card_members`.
4. Тесты покрывают импорт связей.

## 1. Docs (trello_analytics_open_spec)

- [x] 1.1 Delta spec + отметить в `docs/trello-integration.md` snapshot assignees/labels при entity refresh.

## 2. Backend (trello_analytics_back)

- [x] 2.1 Расширить `trello.Card` + `GetCards` fields (`idMembers`, labels).
- [x] 2.2 Postgres: sync `card_members` / `card_labels` from card upserts.
- [x] 2.3 Importer: передавать связи при ImportEntities.
- [x] 2.4 Unit/integration tests (fake Trello + DB или store mock).
