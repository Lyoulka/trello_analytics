# Модель данных PostgreSQL

**Статус:** продуктовая схема после bootstrap (`000001` + `000002_core_schema`).  
**Миграции:** `trello_analytics_back/migrations/`.  
**Связано:** [ADR-003](architecture-decisions.md) (сырые события), [ADR-005](architecture-decisions.md) (UTC), OpenSpec change `core-data-model`.

Документ описывает актуальную продуктовую схему. Полный DDL — в миграциях; здесь — таблицы, ключи, индексы и правила хранения.

---

## Соглашения

| Тема | Правило |
|------|---------|
| PK сущностей | `UUID`, `DEFAULT gen_random_uuid()` (расширение `pgcrypto` из bootstrap) |
| Junction-таблицы | Составной PK без отдельного `id` |
| Внешние id Trello | `TEXT`, `UNIQUE NOT NULL` |
| Время | `TIMESTAMPTZ`; в хранилище и при записи — **UTC** (ADR-005). Timezone — для отображения и границ периодов |
| Сырые события vs агрегаты | Исходные actions — в `card_actions` (в т.ч. `raw_payload`). Таблицы агрегатов/метрик в этом change **нет**; пересчёт метрик не удаляет и не перезаписывает сырые события |
| Секреты | Trello token/key **не** хранятся plaintext в `app_settings.value`. Только env / защищённый конфиг |

---

## Таблицы

### boards

| Колонка | Тип / ограничения |
|---------|-------------------|
| `id` | UUID PK |
| `trello_board_id` | TEXT UNIQUE NOT NULL |
| `name` | TEXT NOT NULL |
| `created_at`, `updated_at` | TIMESTAMPTZ |

### trello_lists

| Колонка | Тип / ограничения |
|---------|-------------------|
| `id` | UUID PK |
| `trello_list_id` | TEXT UNIQUE NOT NULL |
| `board_id` | UUID FK → `boards` ON DELETE **RESTRICT** |
| `name` | TEXT NOT NULL |
| `is_archived` | BOOLEAN NOT NULL DEFAULT false |
| `logical_stage` | TEXT NULL (маппинг workflow; детали — later changes) |
| `created_at`, `updated_at` | TIMESTAMPTZ |

**Индекс:** `(board_id)`.

### cards

| Колонка | Тип / ограничения |
|---------|-------------------|
| `id` | UUID PK |
| `trello_card_id` | TEXT UNIQUE NOT NULL |
| `board_id` | UUID FK → `boards` ON DELETE **RESTRICT** |
| `name` | TEXT NOT NULL |
| `description`, `url` | TEXT NULL |
| `current_list_id` | UUID FK → `trello_lists` NULL, ON DELETE **RESTRICT** |
| `created_at_trello` | TIMESTAMPTZ NULL |
| `is_archived` | BOOLEAN NOT NULL DEFAULT false |
| `last_trello_update_at`, `first_synced_at` | TIMESTAMPTZ NULL |
| `updated_at` | TIMESTAMPTZ |

**Индекс:** `(board_id)`.

### members

| Колонка | Тип / ограничения |
|---------|-------------------|
| `id` | UUID PK |
| `trello_member_id` | TEXT UNIQUE NOT NULL |
| `full_name`, `username` | TEXT NULL |
| `is_active` | BOOLEAN NOT NULL DEFAULT true |
| `created_at`, `updated_at` | TIMESTAMPTZ |

Участники Trello **не** создают автоматически записи в `app_users` (ADR-013).

### card_members

Связь card ↔ member. PK `(card_id, member_id)`.  
`assigned_at`, `removed_at` — TIMESTAMPTZ NULL.  
FK: оба конца ON DELETE **CASCADE**.

### labels

| Колонка | Тип / ограничения |
|---------|-------------------|
| `id` | UUID PK |
| `trello_label_id` | TEXT UNIQUE NOT NULL |
| `board_id` | UUID FK → `boards` ON DELETE **RESTRICT** |
| `name`, `color` | TEXT NULL |
| `is_bug_label` | BOOLEAN NOT NULL DEFAULT false |

**Индекс:** `(board_id)`.

### card_labels

Связь card ↔ label. PK `(card_id, label_id)`.  
FK: оба конца ON DELETE **CASCADE**.

### card_actions

Неизменяемое хранилище исходных событий Trello (ADR-003). Идемпотентность импорта — `UNIQUE (trello_action_id)`.

| Колонка | Тип / ограничения |
|---------|-------------------|
| `id` | UUID PK |
| `trello_action_id` | TEXT UNIQUE NOT NULL |
| `card_id` | UUID FK → `cards` NULL, ON DELETE **RESTRICT** |
| `action_type` | TEXT NOT NULL |
| `action_date` | TIMESTAMPTZ NOT NULL |
| `member_id` | UUID FK → `members` NULL, ON DELETE **SET NULL** |
| `source_list_id`, `destination_list_id` | UUID FK → `trello_lists` NULL, ON DELETE **SET NULL** |
| `raw_payload` | JSONB NULL |
| `created_at` | TIMESTAMPTZ NOT NULL DEFAULT now() |

**Индексы:** `(action_date)`, `(card_id)`, `(action_type)`.

### sync_runs

Журнал запусков синхронизации.

| Колонка | Тип / ограничения |
|---------|-------------------|
| `id` | UUID PK |
| `started_at` | TIMESTAMPTZ NOT NULL |
| `finished_at` | TIMESTAMPTZ NULL |
| `status` | TEXT NOT NULL, CHECK ∈ `running` \| `success` \| `failed` \| `cancelled` |
| `processed_count`, `created_count`, `updated_count`, `error_count` | INT NOT NULL DEFAULT 0 |
| `error_summary` | TEXT NULL |

### sync_cursors

Курсоры прогресса импорта (resume).

| Колонка | Тип / ограничения |
|---------|-------------------|
| `id` | UUID PK |
| `resource_type` | TEXT NOT NULL UNIQUE |
| `cursor_value` | TEXT NULL |
| `last_success_at` | TIMESTAMPTZ NULL |
| `updated_at` | TIMESTAMPTZ NOT NULL |

### app_users

Локальные аккаунты приложения (ADR-012, ADR-013). Seed первого Admin — вне этой миграции.

| Колонка | Тип / ограничения |
|---------|-------------------|
| `id` | UUID PK |
| `email` | TEXT UNIQUE NOT NULL |
| `password_hash` | TEXT NOT NULL |
| `display_name` | TEXT NOT NULL |
| `role` | TEXT NOT NULL, CHECK ∈ `admin` \| `team_lead` \| `member` |
| `is_active` | BOOLEAN NOT NULL DEFAULT true |
| `created_at`, `updated_at` | TIMESTAMPTZ |

### app_settings

Несекретные настройки приложения. PK = `key` (TEXT).

| Колонка | Тип / ограничения |
|---------|-------------------|
| `key` | TEXT PK |
| `value` | TEXT NOT NULL |
| `updated_at` | TIMESTAMPTZ NOT NULL |
| `updated_by` | UUID NULL FK → `app_users` ON DELETE **SET NULL** |

`COMMENT ON TABLE` в миграции явно запрещает plaintext Trello token/key в `value`.

---

## ON DELETE (сводка)

| Связь | Поведение |
|-------|-----------|
| `boards` → lists / cards / labels; `cards.current_list_id` | RESTRICT |
| `card_actions.card_id` | RESTRICT |
| `card_actions.member_id` / list FKs | SET NULL |
| `card_members`, `card_labels` | CASCADE с обеих сторон |
| `app_settings.updated_by` | SET NULL |

---

## Миграции

1. `000001_bootstrap` — `pgcrypto` (`gen_random_uuid`).
2. `000002_core_schema` — все таблицы выше.

Откат `000002` удаляет только продуктовые таблицы; bootstrap / `pgcrypto` не трогает.

Локально: см. `docs/local-development.md` (`.\scripts\dev.ps1 migrate-up` / `migrate-down`).
