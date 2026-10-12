# Design

## Context

См. proposal.md. Локальный Compose уже есть; production — существующий сервер без обязательного Docker (ТЗ §14, ADR-001). OQ-009 (конкретные host/path/unit) не закрывать выдуманными значениями — только плейсхолдеры.

## Goals / Non-Goals

**Goals:**

- Воспроизводимые инструкции deployment/operations.
- Шаблоны systemd + Nginx.
- Backup/restore, update/rollback, secrets hygiene.

**Non-Goals:**

- Фактический деплой на сервер пользователя в рамках планирования.
- Docker Compose как production runtime.
- Закрытие OQ-009 без обследования инфраструктуры.

## Decisions

1. **Frontend:** `npm run build` → static root; Nginx `try_files` для Vue Router; API через `/api/` proxy на backend.
2. **Backend:** один бинарник `trello-analytics-api` (имя уточнить при сборке); `Restart=on-failure` в systemd; `EnvironmentFile=` для секретов вне Git.
3. **PostgreSQL:** отдельная DB/user; миграции controlled на update.
4. **Docs layout:** `docs/deployment.md`, `docs/operations.md`; примеры конфигов в `docs/examples/` (`nginx.conf.example`, `trello-analytics.service.example`) с `YOUR_DOMAIN`, `YOUR_APP_ROOT`.
5. **Health:** `/health` и `/ready` используются в процедуре проверки после деплоя; ограничение внешнего доступа — на усмотрение Nginx (документировать рекомендацию).

## Risks / Trade-offs

- [Неизвестная текущая Nginx схема] → Mitigation: только фрагменты + checklist влияния на другие vhosts.
- [Миграции на prod] → Mitigation: backup before migrate; rollback plan.

## Open Questions

- OQ-009 остаётся открытым до обследования сервера; apply change заполняет docs шаблонами, не «фактами» host.
