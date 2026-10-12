# Trello Team Analytics

Внутреннее веб-приложение для аналитики командной разработки по данным Trello.

Система подключается к одной доске, сохраняет историю изменений карточек и считает метрики доставки и качества: скорость закрытия задач, lead time, задержки, возвраты багов, нагрузка по участникам и обзоры за выбранный период.

## Стек

| Слой | Технологии |
|------|------------|
| Backend | Go, REST API |
| Frontend | Vue 3, TypeScript, Vite |
| БД | PostgreSQL |
| Локально | Docker Compose |
| Production | Nginx + systemd (без обязательного Docker) |

Планирование требований ведётся через [OpenSpec](https://openspec.dev) в каталоге `trello_analytics_open_spec`.

## Структура репозитория

```text
trello_analytics/
├── trello_analytics_open_spec/   # OpenSpec, ТЗ, ADR, документация
├── trello_analytics_back/        # Backend (Go)
├── trello_analytics_front/       # Frontend (Vue 3 + Vite)
├── docker-compose.yml
├── scripts/dev.ps1               # Команды для PowerShell
└── Makefile                      # Эквиваленты для Git Bash / WSL
```

## Быстрый старт (Windows + Docker Desktop)

```powershell
.\scripts\dev.ps1 env-init
.\scripts\dev.ps1 up-build
.\scripts\dev.ps1 migrate-up
```

После запуска:

| Сервис | URL |
|--------|-----|
| Frontend | http://localhost:5173 |
| Backend API | http://localhost:8080 |
| Health / Ready | http://localhost:8080/health , http://localhost:8080/ready |
| PostgreSQL | localhost:5432 |

Конфигурация: скопируйте `.env.example` → `.env` (скрипт `env-init` делает это сам). Секреты в Git не коммитятся.

Полезные команды:

```powershell
.\scripts\dev.ps1 logs
.\scripts\dev.ps1 test
.\scripts\dev.ps1 stop
.\scripts\dev.ps1 down
```

Через Make: `make up`, `make migrate-up`, `make test`, `make down`.

Подробнее: [docs/local-development.md](trello_analytics_open_spec/docs/local-development.md).

## Документация

| Документ | Описание |
|----------|----------|
| [Техническое задание](trello_analytics_open_spec/docs/technical-specification.md) | Цели, роли, функциональность |
| [ADR](trello_analytics_open_spec/docs/architecture-decisions.md) | Архитектурные решения |
| [Локальная разработка](trello_analytics_open_spec/docs/local-development.md) | Compose, порты, миграции, тесты |
| [Интеграция с Trello](trello_analytics_open_spec/docs/trello-integration.md) | Работа с Trello API |
| [OpenSpec store](trello_analytics_open_spec/README.md) | Спеки, changes, apply |
| [AI-агенты](trello_analytics_open_spec/docs/ai-agents.md) | Роли Cursor (Orchestrator, Worker, …) |

## Принципы данных

- Хранится сырая история Trello; метрики считаются из событий и правил workflow.
- Пропуски в истории не заполняются молча — фиксируются и предупреждаются.
- В хранилище timestamps в UTC; таймзона используется для отображения и границ периодов.

## Статус

Проект в активной разработке. Активные OpenSpec-изменения: `trello_analytics_open_spec/openspec/changes/`.
