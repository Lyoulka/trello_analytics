# Tasks

## Критерии приёмки change

1. Login и защита маршрутов.
2. Dashboard + графики на данных API.
3. Cards table + card history + Trello links.
4. Reports + copy to clipboard.
5. Settings/workflow/sync status по ролям; data quality warnings.
6. Нет клиентского пересчёта метрик; Chart.js + vue-chartjs подключены.

---

## 1. Backend (trello_analytics_back)

- [ ] 1.1 Нет обязательных задач (контракты из auth-and-api).

## 2. Frontend (trello_analytics_front)

- [ ] 2.1 Добавить API client, обработку ошибок, типы ответов `/api/v1`.
- [ ] 2.2 Реализовать Login + session guard маршрутов.
- [ ] 2.3 Реализовать Dashboard со сводными KPI и общим фильтром периода.
- [ ] 2.4 Подключить Chart.js/vue-chartjs и графики минимального набора ТЗ.
- [ ] 2.5 Реализовать страницу Cards (фильтры, сортировка, поиск, пагинация, ссылка Trello).
- [ ] 2.6 Реализовать страницу истории карточки.
- [ ] 2.7 Реализовать Reports + copy text preview.
- [ ] 2.8 Реализовать Settings (workflow, thresholds, timezone) и отображение sync/data quality.
- [ ] 2.9 Реализовать Admin Users UI (create/list/deactivate) при роли Admin.
- [ ] 2.10 Добавить frontend-тесты на ключевые состояния (empty/error/warning/role).

## 3. Docs (trello_analytics_open_spec)

- [ ] 3.1 При необходимости кратко отметить выбор Chart.js в `docs/architecture-decisions.md` (OQ-005 закрыт этим change).

## 4. Validation

- [ ] 4.1 Сверить с критериями приёмки 1–6.
- [ ] 4.2 Выполнить `openspec validate frontend-analytics-ui --store trello-analytics` без ошибок.
