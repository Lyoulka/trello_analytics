# Change: Bug reopen по QA «на проверку» (переходы)

## Why

Текущая формула (баг с лейблом + reopen = done→work, %, уникальные карточки, знаменатель = first done-enter) не соответствует процессу команды: reopen — уход из «Тестирование (на проверку)» не в Done; знаменатель — переходы карточек с лейблом БАГ в Done (Готово / HORECA готово).

## What Changes

- Знаменатель Bug reopen %: **число переходов** карточек с bug-label в `done_stages` за период.
- Числитель: **число переходов** из `bug_reopen_from_stages` (напр. `qa_review`) в любой stage ∉ `done_stages`.
- Конфиг: `bug_reopen_from_stages` + маппинг списка «Тестирование (на проверку)» → отдельный logical stage.
- Документ `docs/analytics-metrics.md` и delta spec.

## Impact

- `trello_analytics_back` — workflow schema/settings, analytics engine, tests
- `trello_analytics_front` — поле Settings (опционально)
- `trello_analytics_open_spec` — metrics docs + this change
