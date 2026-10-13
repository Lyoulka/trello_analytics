# Tasks



## Критерии приёмки change



1. Login и защита маршрутов.

2. Dashboard + графики на данных API.

3. Cards table + card history + Trello links.

4. Reports + copy to clipboard.

5. Settings/workflow/sync status по ролям; data quality warnings.

6. Нет клиентского пересчёта метрик; Chart.js + vue-chartjs подключены.

7. Единый UI на shadcn-vue + Tailwind (референс компонентов); loading/empty/error; адаптивность; без выдуманных метрик в рабочих API.



---



## 1. Backend (trello_analytics_back)



- [x] 1.1 Нет обязательных задач (контракты из auth-and-api).



## 2. Frontend (trello_analytics_front)



- [x] 2.0 Подключить единый UI-подход: проверить `package.json`, установить Tailwind + shadcn-vue (Vite), зафиксировать токены (палитра/типографика/spacing) и app shell по референсу https://www.shadcn-vue.com/docs/components; не смешивать другие UI frameworks. Готово: init работает, базовые primitives (Button/Card/Alert/Skeleton/Empty) доступны, краткая карта экран→компоненты в коде или комментарии к layout.

- [x] 2.1 Добавить API client, обработку ошибок, типы ответов `/api/v1`.

- [x] 2.2 Реализовать Login + session guard маршрутов (стиль форм — shadcn Field/Input/Button).

- [x] 2.3 Реализовать Dashboard со сводными KPI и общим фильтром периода (Card/Badge; состояния loading/empty/error).

- [x] 2.4 Подключить Chart.js/vue-chartjs и графики минимального набора ТЗ (цвета серий из UI-токенов).

- [x] 2.5 Реализовать страницу Cards (фильтры, сортировка, поиск, пагинация, ссылка Trello; Table/Data Table).

- [x] 2.6 Реализовать страницу истории карточки.

- [x] 2.7 Реализовать Reports + copy text preview (единый shell и Card-секции).

- [x] 2.8 Реализовать Settings (workflow, thresholds, timezone) и отображение sync/data quality.

- [x] 2.9 Реализовать Admin Users UI (create/list/deactivate) при роли Admin.

- [x] 2.10 Добавить frontend-тесты на ключевые состояния (empty/error/warning/role); убедиться, что demo/mock данные не попадают в рабочие API-экраны без явной пометки.



## 3. Docs (trello_analytics_open_spec)



- [x] 3.1 Отметить в `docs/architecture-decisions.md`: OQ-005 → Chart.js + vue-chartjs; UI kit → shadcn-vue + Tailwind (референс visual direction).



## 4. Validation



- [x] 4.1 Сверить с критериями приёмки 1–7.

- [x] 4.2 Выполнить `openspec validate frontend-analytics-ui --store trello-analytics` без ошибок.


