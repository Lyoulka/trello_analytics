# Tasks

## Критерии приёмки change

1. Workflow settings сохраняются в БД с audit fields.
2. Completed/Throughput/Lead/Cycle/reopen/fix-time/stale/workload считаются на backend.
3. Incomplete history и незаполненный workflow дают warnings, не выдуманные даты.
4. Unit-тесты покрывают граничные случаи формул.
5. Есть `docs/analytics-metrics.md`.
6. Нет обязательного UI и полного REST OpenAPI (сервисный слой достаточно).

---

## 1. Backend (trello_analytics_back)

- [ ] 1.1 Реализовать хранение/чтение workflow settings (stages, start/done/blocked, bug labels, stale threshold, timezone, Lead Time start rule).
- [ ] 1.2 Реализовать интерпретацию list-move events → stage transitions.
- [ ] 1.3 Реализовать completed + Throughput (уникальные карточки, ADR-007) с фильтрами периода.
- [ ] 1.4 Реализовать Lead Time и календарный Cycle Time; тесты на missing start/done.
- [ ] 1.5 Реализовать bug reopen + fix time + multi-cycle; тест done-to-done ≠ reopen.
- [ ] 1.6 Реализовать WIP/stale/workload с правилом multi-assignee из design.
- [ ] 1.7 Добавить unit-тесты граничных случаев (повторные циклы, archive, неполная история).

## 2. Frontend (trello_analytics_front)

- [ ] 2.1 Нет обязательных задач.

## 3. Docs (trello_analytics_open_spec)

- [ ] 3.1 Создать `docs/analytics-metrics.md` с формулами и exclusions.
- [ ] 3.2 Отметить в docs, что OQ-003/007/008 закрываются конфигурацией + предупреждениями, не жёстким hardcoded правилом.

## 4. Validation

- [ ] 4.1 Сверить с критериями приёмки 1–6.
- [ ] 4.2 Выполнить `openspec validate workflow-and-metrics --store trello-analytics` без ошибок.
