# Tasks

## Критерии приёмки change

1. Login/logout/me работают; пароли только hash.
2. Bootstrap Admin + Admin CRUD users; нет публичной регистрации.
3. RBAC на всех protected endpoints.
4. Analytics/cards/reports/settings/sync endpoints из ТЗ §10.
5. Секреты Trello маскируются; ошибки без утечек.
6. OpenAPI или `docs/api.md` актуален.
7. Frontend экраны вне scope.

---

## 1. Backend (trello_analytics_back)

- [x] 1.1 Реализовать password hashing, sessions/cookies, POST login/logout, GET me.
- [x] 1.2 Реализовать bootstrap Admin из env при пустых users.
- [x] 1.3 Реализовать Admin users API (create/list/update/deactivate).
- [x] 1.4 Добавить RBAC middleware и матрицу прав по design.
- [x] 1.5 Подключить analytics endpoints к сервису метрик с фильтрами и warnings.
- [x] 1.6 Реализовать cards list/detail/history с пагинацией и сортировкой.
- [x] 1.7 Реализовать reports daily/period/preview.
- [x] 1.8 Реализовать settings + workflow GET/PATCH|PUT с маскированием секретов.
- [x] 1.9 Привязать sync start/status/history/retry к ролям; integration-тесты authz.
- [x] 1.10 Опубликовать OpenAPI или `docs/api.md`.

## 2. Frontend (trello_analytics_front)

- [x] 2.1 Нет обязательных UI-задач (типы можно добавить в frontend change).

## 3. Docs (trello_analytics_open_spec)

- [x] 3.1 Добавить/синхронизировать `docs/api.md` (если не только generated OpenAPI в back).

## 4. Validation

- [x] 4.1 Сверить с критериями приёмки 1–7.
- [x] 4.2 Выполнить `openspec validate auth-and-api --store trello-analytics` без ошибок.
