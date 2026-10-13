# Модель данных PostgreSQL

**Статус:** продуктовая схема после bootstrap (`000001` + `000002_core_schema` + `000003_workflow_settings`).  
**Миграции:** `trello_analytics_back/migrations/`.  
**Связано:** [ADR-003](architecture-decisions.md) (сырые события), [ADR-005](architecture-decisions.md) (UTC), [ADR-008](architecture-decisions.md) (workflow), OpenSpec changes `core-data-model`, `workflow-and-metrics`.

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
| `is_active` | BOOLEAN NOT NULL DEFAULT true — продуктовый флаг «учитывать карточки колонки в аналитике» (change `list-analytics-active`); sync не перезаписывает |
| `logical_stage` | TEXT NULL — SSOT маппинга list → logical stage (`workflow-and-metrics`) |
| `created_at`, `updated_at` | TIMESTAMPTZ |

**Индекс:** `(board_id)`.

### cards

| Колонка | Тип / ограничения |
|---------|-------------------|
| `id` | UUID PK |
| `trello_card_id` | TEXT UNIQUE NOT NULL |
| `board_id` | UUID FK → `boards` ON DELETE **RESTRICT** |
| `name` | TEXT NOT NULL |
| `description` | TEXT NULL |
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

`is_active` — **продуктовый** флаг «учитывать участника в аналитике» (не зеркало Trello `deactivated`). Default `true` при INSERT; entity sync обновляет имя/username и **не** перезаписывает `is_active`. Управление: `PATCH /api/v1/members/{id}` (см. `docs/api.md` и change `member-analytics-active`).

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

### sessions

Server-side sessions for cookie `tta_session` (auth-and-api). Cookie plaintext is **never** stored — only SHA-256 hex of the cookie value in `token_hash`. Absolute TTL (default 7d) via `expires_at`. Logout sets `revoked_at`; user deactivate revokes all rows for `user_id`.

| Колонка | Тип / ограничения |
|---------|-------------------|
| `id` | UUID PK |
| `user_id` | UUID NOT NULL FK → `app_users` ON DELETE **CASCADE** |
| `token_hash` | TEXT UNIQUE NOT NULL |
| `expires_at` | TIMESTAMPTZ NOT NULL |
| `created_at` | TIMESTAMPTZ NOT NULL |
| `revoked_at` | TIMESTAMPTZ NULL |

**Индексы:** `user_id`, `expires_at`, partial active on `token_hash` where `revoked_at IS NULL`.

### workflow_settings

Board-level правила workflow и пороги (1 ряд на board). Маппинг списков — в `trello_lists.logical_stage`, не здесь.

| Колонка | Тип / ограничения |
|---------|-------------------|
| `id` | UUID PK |
| `board_id` | UUID UNIQUE NOT NULL FK → `boards` ON DELETE **RESTRICT** |
| `lead_time_start_rule` | TEXT NOT NULL DEFAULT `unset`, CHECK ∈ `unset` \| `card_created` \| `first_enter_stage` |
| `lead_time_start_stage` | TEXT NULL |
| `cycle_start_stage` | TEXT NULL |
| `done_stages`, `blocked_stages`, `work_stages` | TEXT[] NOT NULL DEFAULT `{}` |
| `stale_threshold_seconds` | INT NOT NULL DEFAULT 604800 (7d) |
| `data_stale_sync_age_seconds` | INT NOT NULL DEFAULT 86400 (24h) |
| `timezone` | TEXT NOT NULL DEFAULT `UTC` |
| `updated_at` | TIMESTAMPTZ NOT NULL |
| `updated_by` | UUID NULL FK → `app_users` ON DELETE **SET NULL** |

### workflow_settings_revisions

Append-only JSON snapshot при каждом Save (ADR-008 / OQ-006).

| Колонка | Тип / ограничения |
|---------|-------------------|
| `id` | UUID PK |
| `board_id` | UUID NOT NULL FK → `boards` ON DELETE **RESTRICT** |
| `settings_json` | JSONB NOT NULL (list→stage, bug label ids, rules/пороги) |
| `created_at` | TIMESTAMPTZ NOT NULL |
| `created_by` | UUID NULL FK → `app_users` ON DELETE **SET NULL** |

**Индекс:** `(board_id, created_at DESC)`.

---

## ON DELETE (сводка)

| Связь | Поведение |
|-------|-----------|
| `boards` → lists / cards / labels; `cards.current_list_id` | RESTRICT |
| `card_actions.card_id` | RESTRICT |
| `card_actions.member_id` / list FKs | SET NULL |
| `card_members`, `card_labels` | CASCADE с обеих сторон |
| `app_settings.updated_by` | SET NULL |
| `sessions.user_id` | CASCADE |
| `workflow_settings.board_id` / revisions | RESTRICT |
| `workflow_settings.updated_by` / revisions `created_by` | SET NULL |

---

## Миграции

1. `000001_bootstrap` — `pgcrypto` (`gen_random_uuid`).
2. `000002_core_schema` — boards…app_settings (см. выше).
3. `000003_workflow_settings` — `workflow_settings` + `workflow_settings_revisions`.
4. `000004_sessions` — `sessions` (+ indexes).

Откат `000004` удаляет `sessions`. Откат `000003` удаляет только workflow-таблицы. Откат `000002` удаляет продуктовые таблицы; bootstrap / `pgcrypto` не трогает.

Локально: см. `docs/local-development.md` (`.\scripts\dev.ps1 migrate-up` / `migrate-down`).
