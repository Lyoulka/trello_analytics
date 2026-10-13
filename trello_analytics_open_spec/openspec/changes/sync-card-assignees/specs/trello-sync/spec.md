# Spec Delta — card assignees / labels sync

## ADDED Requirements

### Requirement: Синхронизация текущих исполнителей карточки
При entity import/refresh система MUST сохранять текущих исполнителей карточки из Trello (`idMembers`) в `card_members` с привязкой к `members`, без выдумывания дат назначения, если API их не отдаёт (`assigned_at` NULL допустим).

#### Scenario: Карточка с участниками
- **WHEN** Trello card содержит один или более `idMembers` и sync сущностей успешен
- **THEN** для каждого такого member MUST существовать активная строка в `card_members` (removed_at IS NULL)

#### Scenario: Участник снят с карточки
- **WHEN** повторный sync видит, что member больше не в `idMembers` карточки
- **THEN** соответствующая связь MUST перестать считаться активной (removed_at или удаление по правилам схемы)

#### Scenario: Workload после sync
- **WHEN** есть активные `card_members` и карточки в WIP/completed
- **THEN** analytics workload MUST возвращать ненулевой список участников (при наличии данных)

### Requirement: Синхронизация текущих labels карточки
При том же entity refresh система MUST сохранять текущие labels карточки в `card_labels` аналогично assignees.

#### Scenario: Карточка с labels
- **WHEN** Trello card содержит labels/idLabels и sync успешен
- **THEN** активные строки `card_labels` MUST соответствовать набору на карточке
