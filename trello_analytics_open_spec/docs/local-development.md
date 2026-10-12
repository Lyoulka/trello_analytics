# Локальная разработка (Windows + Docker Desktop)

Инструкция для workspace:

```text
trello_analytics/
├── docker-compose.yml
├── .env.example
├── scripts/dev.ps1
├── trello_analytics_back/     # Go
├── trello_analytics_front/    # Vue 3 + Vite
└── trello_analytics_open_spec/
```

Production (Nginx / systemd) этой инструкцией **не** затрагивается.

## Требования

- Docker Desktop для Windows (Compose V2)
- PowerShell 5+ (или Git Bash + `make`)

## Первый запуск

```powershell
cd C:\Users\user\Developing\trello_analytics
.\scripts\dev.ps1 env-init
.\scripts\dev.ps1 up-build
.\scripts\dev.ps1 migrate-up
```

Откройте frontend: [http://localhost:5173](http://localhost:5173)

## Адреса сервисов

| Сервис | С хоста | Внутри Compose-сети |
|--------|---------|---------------------|
| Frontend (Vite) | http://localhost:5173 | http://frontend:5173 |
| Backend API | http://localhost:8080 | http://backend:8080 |
| Health | http://localhost:8080/health | — |
| Ready (с БД) | http://localhost:8080/ready | — |
| API health | http://localhost:8080/api/v1/health | проксируется Vite как `/api/...` |
| PostgreSQL | localhost:5432 | `postgres:5432` |

Backend подключается к БД по хосту **`postgres`**, не `localhost`.

Frontend в браузере ходит на Vite; запросы `/api/*`, `/health`, `/ready` проксируются на `backend:8080`.

## Команды (PowerShell)

| Действие | Команда |
|----------|---------|
| Старт | `.\scripts\dev.ps1 up` |
| Пересборка и старт | `.\scripts\dev.ps1 up-build` |
| Остановка без удаления данных | `.\scripts\dev.ps1 stop` |
| Убрать контейнеры (volume БД сохраняется) | `.\scripts\dev.ps1 down` |
| Логи | `.\scripts\dev.ps1 logs` |
| Логи backend / frontend / postgres | `.\scripts\dev.ps1 logs-backend` и т.д. |
| Миграции up | `.\scripts\dev.ps1 migrate-up` |
| Откат одной миграции | `.\scripts\dev.ps1 migrate-down` |
| Версия миграций | `.\scripts\dev.ps1 migrate-version` |
| Тесты | `.\scripts\dev.ps1 test` |
| **Сброс БД (разрушающе)** | `.\scripts\dev.ps1 db-reset` |

`db-reset` выполняет `docker compose down -v` — удаляет volume PostgreSQL.

Эквиваленты через Make (Git Bash / WSL): `make up`, `make stop`, `make migrate-up`, `make db-reset`, `make test`.

## Конфигурация и секреты

- Шаблон: корневой `.env.example`
- Рабочий файл: корневой `.env` (в `.gitignore`)
- Секреты не класть в исходники и образы
- Файл `trello_analytics_back/.env.example` относится к вспомогательному Trello/OpenSpec tooling и не заменяет корневой `.env` Compose

## Миграции

Инструмент: [golang-migrate](https://github.com/golang-migrate/migrate) (`migrate/migrate:v4.18.1`).

Файлы: `trello_analytics_back/migrations/`.

Стартовая миграция `000001_bootstrap` только включает extension `pgcrypto` — **бизнес-таблиц нет**.

## Hot reload

- Backend: Air пересобирает при изменении `.go`
- Frontend: Vite HMR; на Docker Desktop для Windows включён `watch.usePolling`

## Проверка работоспособности

```powershell
.\scripts\dev.ps1 config
docker compose ps
curl http://localhost:8080/ready
curl http://localhost:5173/
.\scripts\dev.ps1 test
```

Ожидание: `postgres` — healthy; `/ready` и `/api/v1/health` — `{"status":"ok","database":"up"}`.
