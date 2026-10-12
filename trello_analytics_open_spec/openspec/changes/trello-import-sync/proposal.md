# Proposal

## Why

После схемы БД нужен механизм загрузки и обновления данных одной доски Trello. Без идемпотентного импорта истории с resume нельзя считать метрики и показывать качество данных.

## What Changes

- Реализовать импорт lists, cards, members, labels и доступных actions в PostgreSQL.
- Обеспечить пагинацию, resume по курсорам, идемпотентность по `trello_action_id`.
- Ввести журнал sync runs, фоновый и ручной запуск, защиту от параллельных конфликтующих sync.
- Фиксировать ошибки и ограничения истории без подмены данных; экспонировать статус sync для API/UI.
- Опереться на результаты spike (`docs/trello-integration.md`).

## Capabilities

### New Capabilities

- `trello-sync`: импорт и регулярная синхронизация одной доски Trello в PostgreSQL с идемпотентностью, resume и наблюдаемым статусом.

### Modified Capabilities

- (нет в main specs; зависит от будущего archive `data-model`)

## Impact

**Затронутые репозитории:**

- `trello_analytics_back` — пакеты trello client/importer/sync, scheduler, storage; опционально минимальные sync status endpoints для проверки (полные auth-gated API — в `auth-and-api`).
- `trello_analytics_open_spec` — artifacts; обновление `docs/trello-integration.md` при уточнении operational sync.
- `trello_analytics_front` — **вне scope** (экраны статуса — в `frontend-analytics-ui`).

**Вне scope:** формулы метрик, workflow mapping UI, Dashboard, production deploy.

**Зависимость:** `core-data-model` должен быть применён (таблицы существуют).
