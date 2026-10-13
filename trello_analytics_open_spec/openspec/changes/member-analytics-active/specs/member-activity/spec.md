# Spec Delta: member-activity

## Purpose

Даёт управлять включением участников Trello в аналитику через `members.is_active` и исключать неактивных из отчётов и графиков без удаления сырых данных.

## ADDED Requirements

### Requirement: Продуктовый флаг активности участника
Система MUST хранить у каждого импортированного участника Trello булев флаг `is_active` со значением по умолчанию `true` при первом появлении в БД. Флаг MUST означать включение участника в аналитику приложения и MUST NOT быть обязательным зеркалом статуса deactivated в Trello.

#### Scenario: Первый импорт участника
- **WHEN** участник впервые сохраняется из Trello sync
- **THEN** запись MUST создаваться с `is_active = true` (если иное не задано явно продуктовым API)

#### Scenario: Sync не сбрасывает ручную деактивацию
- **WHEN** оператор установил `is_active = false`, а затем выполнен повторный entity sync того же `trello_member_id`
- **THEN** `is_active` MUST остаться `false`; обновляются только отображаемые поля (имя/username) при наличии изменений

### Requirement: API управления участниками
Система MUST предоставлять `GET /api/v1/members` (Member+) со списком участников и `PATCH /api/v1/members/{id}` (Team Lead+) для изменения `is_active`.

#### Scenario: Список участников
- **WHEN** авторизованный Member запрашивает `GET /api/v1/members`
- **THEN** ответ MUST содержать участников с полями `id`, `trello_member_id`, `full_name`, `username`, `is_active`

#### Scenario: Переключение активности
- **WHEN** Team Lead или Admin отправляет `PATCH /api/v1/members/{id}` с `{ "is_active": false }`
- **THEN** участник MUST сохраниться с `is_active = false` и последующие analytics/reports MUST учитывать это сразу (без обязательного re-sync Trello)

#### Scenario: Member не может менять активность
- **WHEN** пользователь с ролью Member вызывает PATCH members
- **THEN** API MUST ответить отказом по RBAC (403)

### Requirement: Исключение неактивных из аналитики и отчётов
Расчёт analytics и reports MUST игнорировать назначения неактивных участников. Карточка, у которой есть текущие назначения, но среди них нет ни одного активного участника, MUST быть исключена из метрик отчётов и графиков (включая workload). Карточки без назначений MUST сохранять текущее поведение агрегатов доски.

#### Scenario: Workload без неактивного
- **WHEN** у участника `is_active = false` и он назначен на карточки
- **THEN** `workload` / overview MUST NOT содержать запись с его `member_id`

#### Scenario: Карточка только с неактивными исполнителями
- **WHEN** у карточки все текущие исполнители неактивны
- **THEN** карточка MUST NOT входить в completed/throughput/WIP/stale/bugs и связанные поля period/daily/preview reports

#### Scenario: Multi-assignee с активным и неактивным
- **WHEN** на карточке есть хотя бы один активный исполнитель и один неактивный
- **THEN** карточка MUST учитываться в агрегатах доски; в персональной атрибуции MUST учитываться только активный исполнитель
