# Proposal

## Why

Локальная инфраструктура (Compose, health/ready, bootstrap-миграция) уже есть, но продуктовой схемы PostgreSQL из ТЗ нет. Без таблиц для доски, карточек, событий, sync-состояния и пользователей нельзя реализовать импорт, метрики и API.

## What Changes

- Ввести продуктовую схему PostgreSQL по ТЗ §9 (boards, lists, cards, members, labels, actions, sync, users, settings).
- Зафиксировать ограничения целостности, UNIQUE по внешним id Trello, индексы для выборок по времени и карточкам.
- Хранить временные отметки в UTC; исходные события — отдельно от будущих агрегатов.
- Описать схему в `docs/database.md`.
- Не реализовывать sync engine, аналитику и UI в этом change.

## Capabilities

### New Capabilities

- `data-model`: требования к схеме PostgreSQL, миграциям, целостности и документированию модели данных для одной доски Trello.

### Modified Capabilities

- (нет — базовых спек ещё нет)

## Impact

**Затронутые репозитории:**

- `trello_analytics_back` — миграции в `migrations/`, при необходимости типы/репозитории storage без бизнес-логики sync/analytics.
- `trello_analytics_open_spec` — OpenSpec artifacts и `docs/database.md`.
- `trello_analytics_front` — **вне scope**.

**Вне scope:** импортёр Trello, scheduler, формулы метрик, REST analytics/auth UI, production Nginx/systemd.

**Зависимости:** последующие changes `trello-import-sync`, `workflow-and-metrics`, `auth-and-api` опираются на эту схему.
