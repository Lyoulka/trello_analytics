# Tasks

## Критерии приёмки

1. В UI есть таблица участников Trello с переключателем `is_active`.
2. После `is_active=false` участник исчезает из workload/графика «Нагрузка по участникам» и не атрибутируется в аналитике.
3. Карточки, у которых среди текущих исполнителей нет активных участников, не входят в метрики analytics/reports.
4. Повторный Trello sync не сбрасывает ручной `is_active`.
5. Тесты покрывают фильтр аналитики и PATCH members; docs обновлены.

## 1. Docs (trello_analytics_open_spec)

- [x] 1.1 Delta specs + `docs/api.md` (+ `docs/database.md` про семантику `members.is_active`).

## 2. Backend (trello_analytics_back)

- [x] 2.1 UpsertMembers: не перезаписывать `is_active` на CONFLICT; importer не обязан передавать Trello deactivated в флаг.
- [x] 2.2 Store: ListMembers, SetMemberActive; handlers GET/PATCH `/api/v1/members`; RBAC Member+ / Team Lead+.
- [x] 2.3 Analytics load/Engine: учитывать только active assignees; исключать карточки без активных исполнителей при наличии назначений.
- [x] 2.4 Unit/handler tests (upsert preserve, inactive exclusion, PATCH authz).

## 3. Frontend (trello_analytics_front)

- [x] 3.1 API client + типы для members.
- [x] 3.2 Страница таблицы участников с toggle `is_active` (Team Lead+), route + nav.
- [x] 3.3 Минимальные тесты страницы/API клиента.
