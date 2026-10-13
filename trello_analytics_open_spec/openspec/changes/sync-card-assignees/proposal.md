# Change: Sync card assignees (and labels) into junction tables

## Why

Workload и фильтры по участникам пустые, хотя в Trello на карточках есть members: в БД `members` = 9, а `card_members` = 0. Аналогично `card_labels` = 0 при наличии `labels`.

Корневая причина (код):
- `GetCards` запрашивает fields без `idMembers` / labels;
- тип `trello.Card` не содержит assignees;
- importer upsert’ит cards/members/labels, но **не заполняет** `card_members` / `card_labels`.

Схема и core-data-model требуют хранить связи card–member / card–label; trello-import-sync явно не зафиксировал заполнение связей при entity refresh. Нужен явный delta + реализация.

## What Changes

- При entity import/sync: читать assignees (и labels) с карточек Trello и идемпотентно синхронизировать `card_members` / `card_labels` (текущее состояние; `removed_at` для снятых).
- Тесты importer + регрессия «после sync card_members > 0 при наличии idMembers».
- Docs: `trello-integration.md` / database note при необходимости.

## Impact

- `trello_analytics_back` — trello client, importer, postgres entities
- `trello_analytics_open_spec` — этот change
- После фикса: повторный sync → Workload на Dashboard
