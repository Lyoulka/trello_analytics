# Design

## Context

См. proposal.md. История в БД после `trello-import-sync`. ADR-004/006/007/008/009: метрики на backend; Cycle Time календарный; Throughput = уникальные карточки; workflow в БД; неполные данные явно.

## Goals / Non-Goals

**Goals:**

- Модель workflow settings + расчёт метрик из `card_actions` + текущего состояния.
- Документ формул `docs/analytics-metrics.md`.
- Unit-тесты граничных случаев.

**Non-Goals:**

- Active/working Cycle Time (OQ-010 / ADR-006).
- HTTP handlers и Vue (следующие changes).
- Закрытие OQ-003 жёстким правилом без настройки.
- Rewrite importer/syncer; enrichment `card_actions.card_id` / list FKs.

## Decisions

1. **List → stage SSOT:** колонка `trello_lists.logical_stage` (уже в `000002`). Отдельную mapping-таблицу не вводить. Значение — свободный stage key; `NULL` = незамаплено.
2. **Board workflow rules в БД (`000003`).** Таблица `workflow_settings` (1 ряд на board): роли этапов (`done_stages` / `blocked_stages` / `work_stages`), `cycle_start_stage`, Lead start rule, пороги stale / data_stale, timezone. Роли не дублировать в `logical_stage`.
3. **Bug labels:** SSOT — `labels.is_bug_label`. `Save` выставляет `true` для выбранных label UUID доски и `false` для остальных label этой доски.
4. **Audit + revisions (OQ-006):** `updated_at` / `updated_by` на `workflow_settings` + append-only `workflow_settings_revisions` (полный JSON snapshot при каждом Save). Engine v1 считает по **текущим** settings.
5. **Lead Time start (OQ-003):** `unset` | `card_created` | `first_enter_stage` (+ `lead_time_start_stage`). Default = `unset` → Lead недоступен + warning (не hardcoded Ready/create).
6. **Completed / Throughput (ADR-007):** уникальные карточки с первым в периоде входом в stage ∈ `done_stages` из non-done; done→done без work — не двойной учёт; повторный done после work — всё ещё 1 в Throughput; циклы — в reopen.
7. **Cycle Time:** календарная разница first enter `cycle_start_stage` → done; не называть «рабочим временем».
8. **Bug reopen:** bug по `is_bug_label` + card_labels; reopen = done → stage ∈ `work_stages`; done→done ≠ reopen; % по когорте (§5.6 ТЗ).
9. **Attribution (OQ-007):** team totals — карточка 1 раз; personal/workload — full count у каждого текущего assignee (без дробления).
10. **Неполнота истории:** нет start/done → exclude + warning / needs_review; не выдумывать даты.
11. **Transitions из raw_payload:** importer пишет `action_type` / `action_date` / `raw_payload`; metrics парсят list moves (`listBefore`/`listAfter` / `old.idList`) и `createCard` для Lead rule `card_created`. Importer не переписывать.
12. **Пакеты Go:** `internal/workflow` (Load/Save), `internal/analytics` (`Engine.Compute`), `internal/storage/postgres` helpers. HTTP/OpenAPI — вне scope.

### DDL `000003_workflow_settings`

```sql
-- workflow_settings
id UUID PK
board_id UUID NOT NULL UNIQUE → boards
lead_time_start_rule TEXT NOT NULL DEFAULT 'unset'
  CHECK IN ('unset','card_created','first_enter_stage')
lead_time_start_stage TEXT NULL
cycle_start_stage TEXT NULL
done_stages / blocked_stages / work_stages TEXT[] NOT NULL DEFAULT '{}'
stale_threshold_seconds INT NOT NULL DEFAULT 604800   -- 7d
data_stale_sync_age_seconds INT NOT NULL DEFAULT 86400 -- 24h
timezone TEXT NOT NULL DEFAULT 'UTC'
updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
updated_by UUID NULL → app_users ON DELETE SET NULL

-- workflow_settings_revisions
id UUID PK
board_id UUID NOT NULL → boards
settings_json JSONB NOT NULL  -- list→stage, bug labels, rules/пороги
created_at TIMESTAMPTZ NOT NULL DEFAULT now()
created_by UUID NULL → app_users ON DELETE SET NULL
INDEX (board_id, created_at DESC)
```

### Service API (внутренний)

- `workflow.Load(ctx, boardID) (Settings, error)`
- `workflow.Save(ctx, boardID, Settings, updatedBy *string)` — транзакция: upsert settings + UPDATE lists/labels + INSERT revision
- `analytics.Engine.Compute(ctx, Query) (Result, error)`
- Warnings: `workflow_incomplete`, `lead_rule_unset`, `missing_cycle_start`, `missing_done_stages`, `missing_work_stages`, `incomplete_history`, `missing_start`, `missing_done`, `ambiguous_reopen`, `data_stale`

## Risks / Trade-offs

- [Неверный маппинг списков] → Mitigation: warning если workflow неполный.
- [JSONB parse на каждый Compute] → Mitigation: ok для v1/одной доски; агрегаты позже.
- [Label via updateCard idLabels] → Mitigation: bug class через `is_bug_label` + `card_labels`; неоднозначность → warning.
- [Archive `updateCard(closed)` не подтверждён] → не использовать как timestamp.

## Open Questions

- Финальные product-числа OQ-008 с тимлидом (проvisional defaults 7d/24h уже в DDL).
- OQ-003/007 остаются конфигурируемыми; не закрывать hardcoded без warning.
