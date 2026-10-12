# Spec Delta

## Purpose

Определяет REST API `/api/v1` для аналитики, карточек, отчётов, настроек и синхронизации с едиными фильтрами периода и документированными контрактами.

## ADDED Requirements

### Requirement: Префикс и документация API
Все внутренние product endpoints MUST использовать префикс `/api/v1`; контракты MUST быть описаны в OpenAPI и/или `docs/api.md`.

#### Scenario: Документ контрактов
- **WHEN** change завершён
- **THEN** существует машиночитаемый OpenAPI или `docs/api.md` с перечисленными endpoints ТЗ §10

### Requirement: Analytics endpoints
Система MUST предоставлять endpoints overview, throughput, lead-time, cycle-time, bugs, reopened-bugs, fix-time, workload, stale-tasks с фильтрами периода и применимыми member/label фильтрами.

#### Scenario: Overview за период
- **WHEN** клиент запрашивает overview с from/to
- **THEN** ответ MUST содержать сводные показатели или явные warnings о невозможности расчёта

#### Scenario: Неполные данные
- **WHEN** истории недостаточно для метрики
- **THEN** ответ MUST включать предупреждение о качестве данных, а не выдуманные значения

### Requirement: Cards и history
Система MUST предоставлять список карточек с пагинацией/фильтрами/сортировкой, карточку по id и хронологию событий; неоднозначные события остаются в сырой истории.

#### Scenario: История карточки
- **WHEN** запрашивается `/cards/{id}/history`
- **THEN** события MUST возвращаться в хронологическом порядке с типами и датами

### Requirement: Reports endpoints
Система MUST предоставлять отчёты daily/period/preview на данных PostgreSQL с указанием полноты данных.

#### Scenario: Period report
- **WHEN** запрашивается отчёт за период
- **THEN** ответ MUST включать сводку, выполненные задачи, баги/reopen, текущие проблемы и data quality hints

### Requirement: Settings и sync endpoints
Система MUST предоставлять чтение/изменение settings и workflow, а также start/status/history/retry sync с ограничениями по ролям из capability `auth`.

#### Scenario: Sync status без секретов
- **WHEN** запрашивается sync status
- **THEN** ответ MUST содержать last success и качество данных без Trello credentials
