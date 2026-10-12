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

- [ ] 1.1 Реализовать password hashing, sessions/cookies, POST login/logout, GET me.
- [ ] 1.2 Реализовать bootstrap Admin из env при пустых users.
- [ ] 1.3 Реализовать Admin users API (create/list/update/deactivate).
- [ ] 1.4 Добавить RBAC middleware и матрицу прав по design.
- [ ] 1.5 Подключить analytics endpoints к сервису метрик с фильтрами и warnings.
- [ ] 1.6 Реализовать cards list/detail/history с пагинацией и сортировкой.
- [ ] 1.7 Реализовать reports daily/period/preview.
- [ ] 1.8 Реализовать settings + workflow GET/PATCH|PUT с маскированием секретов.
- [ ] 1.9 Привязать sync start/status/history/retry к ролям; integration-тесты authz.
- [ ] 1.10 Опубликовать OpenAPI или `docs/api.md`.

## 2. Frontend (trello_analytics_front)

- [ ] 2.1 Нет обязательных UI-задач (типы можно добавить в frontend change).

## 3. Docs (trello_analytics_open_spec)

- [ ] 3.1 Добавить/синхронизировать `docs/api.md` (если не только generated OpenAPI в back).

## 4. Validation

- [ ] 4.1 Сверить с критериями приёмки 1–7.
- [ ] 4.2 Выполнить `openspec validate auth-and-api --store trello-analytics` без ошибок.
