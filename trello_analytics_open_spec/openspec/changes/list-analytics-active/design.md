# Design: list-analytics-active

## Decisions

1. **Семантика:** `trello_lists.is_active` — продуктовый флаг «учитывать карточки этой колонки в аналитике». Default `true` на INSERT.

2. **Критерий исключения:** карточка исключается из `Engine.Compute` / reports / charts, если её **текущий** список (`cards.current_list_id`) ссылается на список с `is_active = false`. Исторические переходы через неактивный список у карточки, которая сейчас в активной колонке, **не** вычищаются отдельно.

3. **Без списка:** `current_list_id IS NULL` — карточка остаётся в агрегатах (как сейчас).

4. **Sync:** `UpsertLists` на CONFLICT обновляет name / is_archived / board_id / updated_at; **не** трогает `is_active`.

5. **API:** как у members — GET Member+, PATCH Team Lead+.

6. **UI:** страница «Колонки» с таблицей и toggle (Team Lead+ пишет).

7. **Сырые данные / GET /cards:** browse каталог не обязан скрывать карточки неактивных колонок; исключение — analytics path.

## Risks

- Деактивация Done-колонки уберёт из completed WIP-снапшот карточки, всё ещё лежащие в Done — ожидаемо по правилу «текущая колонка».
- Большое число уже импортированных списков получит `true` через DEFAULT миграции.
