# Spec Delta

## Purpose

Описывает требования к production-развёртыванию и эксплуатации приложения на существующем сервере с Nginx, systemd и PostgreSQL без обязательного Docker Compose.

## ADDED Requirements

### Requirement: Документированное развёртывание
Проект MUST содержать `docs/deployment.md` с шагами сборки frontend/backend, размещения static, настройки systemd, Nginx proxy `/api`, секретов и проверки health.

#### Scenario: Плейсхолдеры вместо выдуманных host
- **WHEN** в документации указываются host/path/unit
- **THEN** MUST использоваться явные плейсхолдеры, если реальные значения сервера не подтверждены (OQ-009)

### Requirement: Systemd и надёжность процесса
Backend MUST запускаться как systemd-сервис с автозапуском после reboot и перезапуском после аварии; корректное завершение при stop MUST быть описано.

#### Scenario: Пример unit-файла
- **WHEN** change завершён
- **THEN** в docs/examples MUST быть пример unit-файла с EnvironmentFile и Restart политикой

### Requirement: Nginx static + API proxy
Production MUST раздавать собранный frontend и проксировать `/api` на Go backend; секреты backend MUST NOT попадать в frontend build.

#### Scenario: SPA fallback
- **WHEN** пользователь открывает клиентский маршрут напрямую
- **THEN** конфигурация Nginx MUST предусматривать fallback на `index.html`

### Requirement: Backup, restore, update, rollback
`docs/operations.md` MUST описывать логи, sync мониторинг, резервное копирование PostgreSQL, восстановление, обновление приложения и план отката.

#### Scenario: Update checklist
- **WHEN** выпускается новая версия
- **THEN** процедура MUST включать tests/build, backup при необходимости, migrate, restart, health check и rollback path

### Requirement: Критерии готовности production
Документация MUST отражать критерии ТЗ §14.6: доступ через Nginx, API, reboot survival, secrets hygiene, DB persistence, logs, backup/restore.

#### Scenario: Чеклист готовности
- **WHEN** оператор готовит приёмку production
- **THEN** в docs MUST быть проверяемый чеклист без требования Docker Compose на сервере
