# Trello Team Analytics — инструкции для AI-агентов

## Стек (утверждён)

- Backend: Go, REST `/api/v1` — репозиторий `trello_analytics_back`
- Frontend: Vue 3 + TypeScript + Vite — `trello_analytics_front`
- Database: PostgreSQL
- Local: Docker Compose; Production: Nginx + systemd + PostgreSQL
- Requirements: OpenSpec store `trello-analytics` в `trello_analytics_open_spec`

Не предлагать Node.js API или Nuxt 3 как целевой стек приложения.

## Принцип данных

Метрики считаются только по сохранённой истории событий Trello. Недостающие события и даты нельзя придумывать или выдавать за достоверные.

## OpenSpec

- Спеки и changes только в этом store.
- В code-репозиториях — лишь `openspec/config.yaml` с `store: trello-analytics`.
- Прогресс — чекбоксы в `tasks.md`. Apply по одному репозиторию, первая открытая задача секции.
- Документы: `docs/technical-specification.md`, `docs/architecture-decisions.md`, `docs/trello-integration.md`.
- Команда агентов: `docs/ai-agents.md`.

## Роли

Запуск: `/orchestrator`, `/architect`, `/worker-go`, `/worker-frontend`, `/worker-db`, `/worker-devops`, `/debugger`, `/tester`, `/reviewer` или команды `/role-*`.

Общие правила — SSOT в `.cursor/rules` и этом файле. Роли их не переопределяют. Шаблоны отчётов — skill `agent-handoff`.

## Запреты для всех ролей

- Не коммитить, не push, не deploy без явного разрешения пользователя.
- Не отключать тесты/проверки «ради зелёного».
- Не утверждать, что проверки прошли, если они не запускались.
- Не менять несвязанные архивные OpenSpec changes.
- Не использовать абсолютные пути машин — только имена репозиториев.
