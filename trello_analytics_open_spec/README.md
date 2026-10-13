# trello_analytics_open_spec

[OpenSpec](https://openspec.dev) store проекта **Trello Team Analytics**.

Планирование (specs и changes) живёт здесь, а не в коде. Код-репозитории только указывают на этот store.

| | |
|---|---|
| **Store id** | `trello-analytics` |
| **Код-репозитории** | `trello_analytics_back` — бэкенд (**Go**); `trello_analytics_front` — фронтенд (**Vue 3 + TypeScript + Vite**) |
| **БД** | PostgreSQL |
| **Локально** | Docker Compose |
| **Production** | Nginx + systemd (без обязательного Docker) |
| **Структура** | `openspec/specs/` — принятые спеки; `openspec/changes/` — активные изменения |
| **ТЗ** | [`docs/technical-specification.md`](docs/technical-specification.md) |
| **ADR** | [`docs/architecture-decisions.md`](docs/architecture-decisions.md) |
| **Локальный Docker** | [`docs/local-development.md`](docs/local-development.md) — Compose, порты, миграции, тесты |
| **AI-агенты (Cursor)** | [`docs/ai-agents.md`](docs/ai-agents.md) — роли Orchestrator/Worker/Tester/Reviewer/… |

Рекомендуемая раскладка каталогов (рядом друг с другом):

```
trello_analytics/
├── trello_analytics_open_spec/   # этот store
├── trello_analytics_back/        # Go
└── trello_analytics_front/       # Vue 3 + TypeScript + Vite
```

---

## 0. Один раз на машину

```bash
npm install -g @fission-ai/openspec
openspec store register <путь>/trello_analytics_open_spec --id trello-analytics
openspec store list
openspec store doctor trello-analytics
```

Путь регистрации должен указывать на каталог `trello_analytics` (не на опечатку в имени каталога).

## 1. Подключение код-репозитория

В корне `trello_analytics_back` и `trello_analytics_front` уже лежит `openspec/config.yaml`:

```yaml
store: trello-analytics
```

и файлы Cursor от `openspec init --tools cursor`. **Не** создавайте в код-репозиториях локальные `openspec/specs/` и `openspec/changes/` — иначе `store:` игнорируется.

## 2. Планирование change

```bash
openspec new change <change-id> --store trello-analytics
# proposal → design → specs → tasks
openspec validate <change-id> --store trello-analytics
```

Один change может затрагивать и бэк, и фронт, и документацию store. В `tasks.md` секции разделяются по репозиториям:

```markdown
## 1. Backend (trello_analytics_back)
- [ ] 1.1 Эндпоинт GET /api/v1/... — отдаёт ... (готово: ...)

## 2. Frontend (trello_analytics_front)
- [ ] 2.1 Страница src/pages/... выводит ... (готово: ...)

## 3. Docs (trello_analytics_open_spec)
- [ ] 3.1 Обновлён docs/trello-integration.md (готово: ...)
```

## 3. Реализация (apply)

Откройте код-репозиторий (или workset с обоими) и в чате Cursor:

```text
/opsx-apply <change-id>
```

Apply делается по одному репозиторию: сначала `trello_analytics_back`, потом `trello_analytics_front` (если требуется).

Workset, чтобы видеть спеки и код вместе.

PowerShell (одна строка; `^` из CMD здесь не работает, флаг — `--member`, не `--add`):

```powershell
openspec workset create trello-analytics --member <путь>/trello_analytics_open_spec --member <путь>/trello_analytics_back --member <путь>/trello_analytics_front
openspec workset open trello-analytics
```

Или с переносом строк в PowerShell (обратный апостроф `` ` ``):

```powershell
openspec workset create trello-analytics `
  --member <путь>/trello_analytics_open_spec `
  --member <путь>/trello_analytics_back `
  --member <путь>/trello_analytics_front
```

## 4. Архив

Когда все пункты `tasks.md` закрыты:

```bash
openspec archive <change-id> --store trello-analytics
```

---

## Полезные команды

| Команда | Назначение |
|--------|------------|
| `openspec store doctor trello-analytics` | Проверка регистрации store |
| `openspec list --store trello-analytics` | Активные changes |
| `openspec show <change>` | Показать change |
| `openspec validate <change> --store trello-analytics` | Проверка change |
| `openspec instructions apply --change <change> --store trello-analytics` | Инструкции на реализацию |
