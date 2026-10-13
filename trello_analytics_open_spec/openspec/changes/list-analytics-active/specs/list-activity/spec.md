# Spec Delta: list-activity

## Purpose

Даёт управлять включением колонок (списков) Trello в аналитику через `trello_lists.is_active` и исключать карточки из неактивных колонок из отчётов и графиков.

## ADDED Requirements

### Requirement: Продуктовый флаг активности колонки
Система MUST хранить у каждого импортированного списка Trello булев флаг `is_active` со значением по умолчанию `true`. Флаг MUST означать включение карточек этой колонки в аналитику.

#### Scenario: Первый импорт списка
- **WHEN** список впервые сохраняется из Trello sync
- **THEN** запись MUST создаваться с `is_active = true`

#### Scenario: Sync не сбрасывает ручную деактивацию
- **WHEN** оператор установил `is_active = false`, а затем выполнен повторный entity sync того же `trello_list_id`
- **THEN** `is_active` MUST остаться `false`; обновляются отображаемые поля (имя, archived)

### Requirement: API управления колонками
Система MUST предоставлять `GET /api/v1/lists` (Member+) и `PATCH /api/v1/lists/{id}` (Team Lead+) для изменения `is_active`.

#### Scenario: Переключение активности
- **WHEN** Team Lead отправляет `PATCH /api/v1/lists/{id}` с `{ "is_active": false }`
- **THEN** список MUST сохраниться с `is_active = false` и последующие analytics/reports MUST учитывать это без re-sync Trello

#### Scenario: Member не может менять активность
- **WHEN** Member вызывает PATCH lists
- **THEN** API MUST ответить 403

### Requirement: Исключение карточек неактивных колонок из аналитики
Расчёт analytics и reports MUST исключать карточки, чей текущий `current_list_id` ссылается на список с `is_active = false`. Карточки без текущего списка MUST сохранять текущее поведение агрегатов.

#### Scenario: Карточка в неактивной колонке
- **WHEN** карточка находится в списке с `is_active = false`
- **THEN** карточка MUST NOT входить в completed/throughput/WIP/stale/bugs/workload и связанные поля reports

#### Scenario: Карточка в активной колонке
- **WHEN** карточка находится в списке с `is_active = true` (или default)
- **THEN** карточка MUST учитываться в аналитике по существующим правилам workflow
