# Proposal

## Why

Локальный Docker Compose не является production. Для первой рабочей версии нужно развёртывание на существующем сервере: Nginx, systemd, PostgreSQL, секреты, backup/restore и документированный update/rollback.

## What Changes

- Документы `docs/deployment.md` и `docs/operations.md` с процедурами сборки, деплоя, логов, backup/restore, update/rollback.
- Шаблоны unit-файла systemd и фрагмента Nginx (плейсхолдеры host/path — OQ-009, без выдуманных hostname).
- Health/readiness в эксплуатации; секреты вне Git и вне frontend build.
- Критерии готовности production из ТЗ §14.6.

## Capabilities

### New Capabilities

- `production-ops`: эксплуатация production на Nginx + systemd + PostgreSQL без обязательного Docker.

### Modified Capabilities

- (нет)

## Impact

**Затронутые репозитории:**

- `trello_analytics_open_spec` — docs и шаблоны конфигурации в docs или `deploy/` examples.
- `trello_analytics_back` — при необходимости флаги graceful shutdown / пути бинарника в docs; без обязательного изменения бизнес-логики.
- `trello_analytics_front` — production build инструкции; без изменения аналитических формул.

**Вне scope:** Docker Compose как production, выдуманные hostnames сервера, брокер сообщений.
