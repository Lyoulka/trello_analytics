# Spec Delta

## Purpose

Обеспечивает запуск импорта и синхронизации одной доски Trello в PostgreSQL исключительно через CLI, с журналом runs, защитой от параллельного sync и статусом качества данных без утечки секретов.

## ADDED Requirements

### Requirement: CLI как единственный entrypoint импорта
Система MUST запускать импорт/sync одной доски только через консольную команду backend. HTTP endpoints start/status sync и фоновый scheduler внутри API-сервера MUST NOT быть product entrypoint для импорта.

#### Scenario: Успешный запуск через CLI
- **WHEN** оператор запускает CLI sync с корректными env (credentials, board id, database)
- **THEN** система MUST выполнить импорт в PostgreSQL и завершить процесс с кодом успеха при успешном run

#### Scenario: HTTP sync start недоступен
- **WHEN** клиент вызывает прежний `POST /api/v1/sync/start` на API-сервере
- **THEN** система MUST NOT запускать sync через этот путь (маршрут отсутствует или не выполняет импорт)

#### Scenario: Нет in-server scheduler
- **WHEN** API-сервер работает с любым значением прежнего sync interval env
- **THEN** процесс сервера MUST NOT периодически запускать Trello sync самостоятельно

### Requirement: Статус через CLI без секретов
Система MUST предоставлять статус качества данных (last success, current/last run, counts, earliest/latest action, incomplete flag, warnings) через CLI; вывод MUST NOT содержать Trello key/token.

#### Scenario: Чтение статуса
- **WHEN** оператор запускает CLI status
- **THEN** вывод MUST включать агрегированные показатели качества без токенов Trello

### Requirement: Single-flight при CLI-запуске
Повторный запуск CLI sync, пока активен конфликтующий run той же доски, MUST NOT стартовать второй импорт; команда MUST завершиться с явным ненулевым статусом/сообщением об уже running.

#### Scenario: Повторный CLI run во время running
- **WHEN** sync уже выполняется и запускается второй CLI run
- **THEN** система MUST отказать во втором конфликтующем процессе с явным признаком `sync_already_running` (или эквивалент)

### Requirement: Переиспользование импорта без подмены данных
CLI MUST использовать тот же pipeline импорта (entities + board actions, идемпотентность, resume по курсору), что и operational sync; отсутствующая история MUST NOT подменяться выдуманными событиями.

#### Scenario: Resume после прерывания CLI
- **WHEN** CLI sync прерван после части страниц actions и запущен снова
- **THEN** система MUST продолжить с сохранённого курсора и не потерять уже сохранённые события
