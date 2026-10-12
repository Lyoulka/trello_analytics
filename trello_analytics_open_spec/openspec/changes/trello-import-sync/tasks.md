# Tasks

## Критерии приёмки change

1. Первичный импорт lists/cards/members/labels/actions в БД.
2. Идемпотентность по trello ids; resume по курсору.
3. sync_runs + защита от параллельного sync.
4. Статус качества данных без утечки секретов.
5. Фоновый интервал из config + ручной start.
6. Нет формул метрик и UI Dashboard.

---

## 1. Backend (trello_analytics_back)

- [ ] 1.1 Вынести/доработать production Trello client (timeouts, ошибки 401/404/429) на базе spike; credentials только из env.
- [ ] 1.2 Реализовать importer: board metadata, lists, members, labels, cards (open+archived).
- [ ] 1.3 Реализовать постраничную загрузку board actions с записью в card_actions и обновлением sync_cursors.
- [ ] 1.4 Обеспечить ON CONFLICT / идемпотентность для сущностей и actions; тест на повторный импорт без дублей.
- [ ] 1.5 Реализовать resume: прерывание и продолжение с курсора; тест или интеграционный сценарий.
- [ ] 1.6 Реализовать sync orchestration: sync_runs, single-flight, ручной start, scheduler по интервалу из config.
- [ ] 1.7 Экспонировать GET sync status (и POST start) с полями качества данных; без секретов в ответах/логах.
- [ ] 1.8 Обработать retry/backoff для временных ошибок; зафиксировать политику в коде/комментарии.

## 2. Frontend (trello_analytics_front)

- [ ] 2.1 Нет обязательных задач (UI статуса — в frontend-analytics-ui).

## 3. Docs (trello_analytics_open_spec)

- [ ] 3.1 При необходимости обновить `docs/trello-integration.md` секцией operational sync (не переписывая spike-факты).

## 4. Validation

- [ ] 4.1 Сверить с критериями приёмки 1–6.
- [ ] 4.2 Выполнить `openspec validate trello-import-sync --store trello-analytics` без ошибок.
