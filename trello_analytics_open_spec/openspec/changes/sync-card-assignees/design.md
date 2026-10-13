# Design: Sync card assignees

## Decisions

1. **Источник истины текущего состава:** поля карточки Trello `idMembers` (и labels на карте: `idLabels` или embedded `labels`) при GET `/boards/{id}/cards`.
2. **Не** восстанавливать историю назначений из actions в v1 (addMemberToCard и т.п.) — только snapshot текущего состава при entity refresh (как WIP workload: current assignees).
3. Sync `card_members`: upsert активных пар; для пар, исчезнувших с карточки — `removed_at = now()` (или эквивалент soft-remove по схеме).
4. То же для `card_labels`.
5. `GetCards` fields MUST включать данные для связей; не тянуть лишний payload сверх необходимости.
6. Идемпотентность по UNIQUE (card_id, member_id) / (card_id, label_id).

## Risks

- Большие доски: один GET cards с доп. полями — ок для v1 (уже грузим all cards).
- Live sync нужен для проверки на стенде — только с согласия / через Admin UI.
