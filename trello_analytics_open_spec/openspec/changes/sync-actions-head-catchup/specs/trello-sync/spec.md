# Spec Delta — head catch-up board actions

## MODIFIED Requirements

### Requirement: Инкрементальный sync с head catch-up
Повторный sync после того, как в БД уже есть board actions, MUST подтягивать новые head board actions (newest-first, идемпотентный insert) **до или вместе с** resume primary backfill; MUST NOT сбрасывать или перезаписывать курсор oldest (`sync_cursors` / `board_actions`) фазой head catch-up.

#### Scenario: Повторный sync догоняет голову ленты
- **WHEN** в `card_actions` уже есть события и в Trello появились более новые board actions
- **THEN** sync MUST вставить недостающие head actions (`ON CONFLICT DO NOTHING`) и MUST NOT изменить `cursor_value` oldest backfill из-за head-фазы

#### Scenario: Пустая история — только primary
- **WHEN** в БД ещё нет `card_actions`
- **THEN** sync MUST NOT запускать head catch-up как полный обход; primary backfill/resume остаётся единственным путём загрузки actions

#### Scenario: Стоп на известных событиях
- **WHEN** страница head catch-up целиком уже есть в БД (`inserted = 0`)
- **THEN** head-фаза MUST остановиться без дальнейшей пагинации в прошлое ради catch-up

## REMOVED Requirements

### Requirement: Инкрементальный sync без head catch-up (v1)
*(заменено требованием с head catch-up выше)*
