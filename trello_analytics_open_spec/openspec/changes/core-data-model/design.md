# Design

## Context

См. proposal.md — Why. Сейчас в `trello_analytics_back/migrations` есть только `000001_bootstrap` (`pgcrypto`). Compose и health/ready уже работают. Целевая предварительная модель — ТЗ §9; ADR-003/005/010/013 задают неизменяемые события, UTC, одну доску и локальных пользователей.

## Goals / Non-Goals

**Goals:**

- Спроектировать и описать миграции продуктовых таблиц без реализации sync/analytics.
- Обеспечить UNIQUE по Trello id и FK между сущностями доски.
- Задокументировать схему в `docs/database.md`.

**Non-Goals:**

- Импортёр, scheduler, HTTP analytics endpoints, UI.
- Таблицы предрасчитанных агрегатов метрик (можно добавить позже отдельным change).
- Версионирование snapshots workflow (OQ-006) — достаточно колонок `updated_at` / `updated_by` в settings; полные snapshots — в `workflow-and-metrics`.

## Decisions

1. **Одна доска в схеме.** Таблица `boards` допускает несколько строк технически, но продукт v1 использует одну выбранную доску (ADR-010); уникальность `trello_board_id`.
2. **Внутренние UUID + внешние Trello id.** PK — `uuid`/`bigserial`; бизнес-ключ Trello — отдельная UNIQUE-колонка.
3. **`card_actions.raw_payload` как JSONB.** Нужен для отладки и неоднозначных событий; индексы по `action_date`, `card_id`, `action_type`, UNIQUE(`trello_action_id`).
4. **`logical_stage` на list и `is_bug_label` на label.** Черновые колонки для будущего workflow; заполнение и версионирование правил — в следующем change, здесь только место хранения.
5. **Секреты Trello.** В v1 — env / защищённый конфиг процесса, не plaintext в `app_settings`; в docs явно зафиксировать.
6. **Нумерация миграций.** Следующая после bootstrap: `000002_core_schema.up.sql` / `.down.sql` (допустимо разбить на 000002–000003, если файл слишком большой).

## Risks / Trade-offs

- [Ранняя фиксация колонок workflow] → Mitigation: nullable `logical_stage` / `is_bug_label`; семантика в `workflow-and-metrics`.
- [Отсутствие агрегатных таблиц] → Mitigation: пересчёт из событий; агрегаты добавить при perf-проблемах.
- [Down-миграция удалит данные] → Mitigation: документировать; в prod не применять down без бэкапа.

## Migration Plan

1. Добавить up/down миграции в `trello_analytics_back/migrations`.
2. Прогнать через существующий `scripts/dev.ps1 migrate-up` / Compose migrate.
3. Откат локально: `migrate-down` одной версии.
4. Обновить `docs/database.md`.

## Open Questions

- Точные типы для `assigned_at`/`removed_at` в `card_members`, если Trello не отдаёт даты назначения в sync — оставить nullable; заполнение в `trello-import-sync`.
