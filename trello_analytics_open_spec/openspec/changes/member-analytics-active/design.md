# Design: member-analytics-active

## Decisions

1. **Семантика `members.is_active`:** продуктовый флаг «включать участника в аналитику», а не зеркало Trello `deactivated`. Default `true` на INSERT.

2. **Sync / upsert:** при `ON CONFLICT` обновлять только `full_name`, `username`, `updated_at`. Поле `is_active` на UPDATE **не** трогать. Trello `deactivated` в v1 не маппится в этот флаг.

3. **Правила исключения из аналитики (Engine / reports):**
   - В `AssigneeMemberIDs` попадают только участники с `is_active = true`.
   - Неактивные не появляются в `workload[]` и не должны отдаваться как опции фильтра по участнику, если опции строятся из workload/members API с учётом активности.
   - Карточка **исключается** из Compute (throughput, lead/cycle, bugs, WIP/stale, reports), если у неё есть хотя бы одно текущее назначение (`card_members.removed_at IS NULL`) и **ни одного** активного исполнителя.
   - Карточки без назначений остаются в агрегатах доски (как сейчас).
   - Multi-assignee: карточка с ≥1 активным исполнителем остаётся в агрегатах; атрибуция только на активных (полный count на каждого — OQ-007 без изменений).

4. **Сырые данные:** `card_members` / история не удаляются. Каталог `GET /cards` может по-прежнему показывать все карточки доски (browse); исключение — analytics/reports/charts.

5. **API:**
   - `GET /api/v1/members` — Member+; список всех импортированных участников с `id`, `full_name`, `username`, `is_active`, `trello_member_id`.
   - `PATCH /api/v1/members/{id}` — Team Lead+; body `{ "is_active": bool }`.

6. **UI:** страница «Участники» (Team Lead+ для переключения; Member может только смотреть или страница сразу Team Lead+). Переключатель в таблице по аналогии с UsersPage. Nav link в AppShell.

7. **Миграции:** колонка уже есть в `000002` — новая миграция не обязательна, если семантика меняется только в коде/доках. При необходимости — комментарий в `docs/database.md`.

## Risks

- Уже сохранённые `is_active=false` из Trello deactivated останутся false до ручного включения — приемлемо; при желании оператор включит в UI.
- Исключение «только неактивные assignees» меняет цифры отчётов — ожидаемо и документируется в acceptance.
