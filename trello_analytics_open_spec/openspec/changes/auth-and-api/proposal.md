# Proposal

## Why

Аналитика и sync должны быть доступны команде через REST `/api/v1` с проверкой ролей на сервере. Без auth и контрактов API frontend не может безопасно потреблять метрики и настройки.

## What Changes

- Локальная авторизация (ADR-013): login/logout/me, bootstrap Admin, Admin создаёт пользователей; без публичной регистрации.
- Роли Admin / Team Lead / Member и серверная проверка прав.
- REST endpoints аналитики, cards/history, reports, settings/workflow, sync (ТЗ §10).
- OpenAPI или `docs/api.md`; секреты Trello не возвращаются клиенту.

## Capabilities

### New Capabilities

- `auth`: вход, сессии/токены, роли, bootstrap Admin, управление пользователями Admin.
- `rest-api`: контракты `/api/v1` для analytics, cards, reports, settings, sync.

### Modified Capabilities

- (нет)

## Impact

**Затронутые репозитории:**

- `trello_analytics_back` — auth middleware, handlers, OpenAPI.
- `trello_analytics_open_spec` — artifacts и `docs/api.md`.
- `trello_analytics_front` — **вне scope** экранов (типы API понадобятся во frontend change).

**Вне scope:** Vue страницы, production Nginx, LLM-отчёты.
