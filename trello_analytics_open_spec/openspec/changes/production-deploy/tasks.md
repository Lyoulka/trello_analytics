# Tasks

## Критерии приёмки change

1. Есть `docs/deployment.md` и `docs/operations.md`.
2. Есть примеры systemd unit и Nginx с плейсхолдерами (не выдуманные hostname).
3. Описаны secrets, health checks, backup/restore, update/rollback.
4. Чеклист готовности production без Docker Compose как runtime.
5. Код метрик/UI не переписывается этим change.

---

## 1. Backend (trello_analytics_back)

- [ ] 1.1 Проверить/документировать production build (`go build`) и graceful shutdown для systemd stop.
- [ ] 1.2 Убедиться, что health/ready пригодны для post-deploy проверки (без изменения бизнес-метрик).

## 2. Frontend (trello_analytics_front)

- [ ] 2.1 Документировать `npm run build` и ожидания base path `/` + API `/api` для Nginx.

## 3. Docs (trello_analytics_open_spec)

- [ ] 3.1 Создать `docs/deployment.md` (build, layout на сервере, systemd, Nginx, secrets, migrate, verify).
- [ ] 3.2 Создать `docs/operations.md` (logs, sync, backup, restore, update, rollback).
- [ ] 3.3 Добавить `docs/examples/trello-analytics.service.example` и `docs/examples/nginx.conf.example` с плейсхолдерами.
- [ ] 3.4 Добавить чеклист готовности production (ТЗ §14.6); отметить OQ-009 как открытый до обследования сервера.

## 4. Validation

- [ ] 4.1 Сверить с критериями приёмки 1–5.
- [ ] 4.2 Выполнить `openspec validate production-deploy --store trello-analytics` без ошибок.
