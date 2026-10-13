# Система AI-агентов (Cursor)

Документ описывает команду ролей для разработки Trello Team Analytics в Cursor.  
**Канон конфигурации:** этот репозиторий (`trello_analytics_open_spec`). Зеркала — в корне workspace, `trello_analytics_back`, `trello_analytics_front`.

## Поддерживаемые механизмы Cursor

| Механизм | Путь | Назначение |
|----------|------|------------|
| Subagents | `.cursor/agents/*.md` | Роли с изолированным контекстом; вызов `/name` или делегирование |
| Rules | `.cursor/rules/*.mdc` | SSOT общих и scoped правил |
| Skills | `.cursor/skills/*/SKILL.md` | `agent-handoff` — шаблоны отчётов; `browser-qa` — live UI цикл; OpenSpec skills без изменений |
| Commands | `.cursor/commands/role-*.md` | Ручной запуск ролей |
| AGENTS.md | корень репозитория | Краткий always-on контекст |

Полноценный автопайплайн между custom-агентами не гарантирован: Orchestrator + ручные `/role-*` + единые отчёты.

## Роли

| Slash | Роль |
|-------|------|
| `/orchestrator` или `/role-orchestrate` | Координация |
| `/architect` | Архитектура до кода |
| `/worker-go` | Backend Go |
| `/worker-frontend` | Vue 3 + TS |
| `/worker-db` | PostgreSQL / sync |
| `/worker-devops` | Docker / Nginx / systemd |
| `/tester` | Независимый QA |
| `/reviewer` | Ревью (readonly) |
| `/debugger` | Дефекты (не после каждого diff) |
| `/browser-qa` или `/role-browser-qa` | Live UI в реальном браузере + цикл исправления |

Специализации Worker — **отдельные агенты** (лучше матчинг `description` в Cursor).

## Процесс

1. Готовность требований OpenSpec  
2. Architect — при необходимости  
3. Один Worker  
4. Самопроверка Worker  
5. Tester  
6. Reviewer  
7. Исправления (Worker; Debugger при дефекте)  
8. Повторные проверки  
9. Приёмка по критериям  
10. Orchestrator Summary  
11. Опционально: `/browser-qa` — проверка работающего UI в реальном браузере; при багах цикл Defect Package → Worker → Reviewer → Tester → повтор в браузере  

**Short path:** docs/config/cosmetic без смены API/БД — без Architect, облегчённые Tester+Reviewer; критерии приёмки обязательны.

Передача: следующая роль только с отчётом предыдущей (`agent-handoff`). Не редактировать одни файлы двумя агентами параллельно. Commit/push/deploy — только по явному разрешению.

**Browser QA:** не правит код приложения; требует Browser MCP (`cursor-ide-browser`) или Playwright. Без инструмента — BLOCKED, без имитации. Делегирование Worker/Reviewer/Tester через Task, если доступно; иначе ручной handoff `/role-*`. Nested subagent может не уметь запускать дочерних — тогда только Defect Package + NEXT_ACTION.

## Примеры запуска ролей

```text
/role-orchestrate change spike-trello-history, задача 1.2 в backend
```

```text
/architect Нужно согласовать контракт GET /api/v1/metrics/throughput и модель периода
```

```text
/worker-go Task: change X, tasks.md 1.3 — эндпоинт health
```

```text
/worker-frontend Task: change X, 2.1 — страница dashboard filters
```

```text
/tester Проверь change X по критериям tasks.md и OpenSpec; вот Worker Report: …
```

```text
/reviewer Ревью diff change X; Worker Report: …
```

```text
/debugger Падает тест TestSyncIdempotent: <лог>. Найди первопричину.
```

```text
/browser-qa Проверь http://localhost:5173 по OpenSpec change X; sync с Trello не запускать
```

```text
/role-browser-qa Только Dashboard и фильтры; найденные баги — через Worker/Reviewer/Tester
```

## Пример полного цикла

1. Пользователь: `/role-orchestrate` change `add-throughput-api`, task `1.1`.  
2. Orchestrator проверяет `openspec/changes/add-throughput-api/` и `tasks.md`.  
3. При новом контракте → `/architect` → Architect Decision.  
4. `/worker-go` реализует task, тесты, Worker Report, чекбокс `- [x]`.  
5. `/tester` и `/reviewer` с Worker Report → PASS + принять (или FAIL/исправить).  
6. При FAIL → `/debugger` или снова Worker → повтор Tester.  
7. Orchestrator Summary: Done / Not done. Коммит — только если пользователь попросил.

Связь с OpenSpec apply: `/opsx-apply` остаётся способом взять следующую задачу из `tasks.md`; команда агентов добавляет роли качества вокруг реализации.

## Что автоматизировано / что вручную

| Авто / полуавто | Вручную |
|-----------------|---------|
| Делегирование subagent по `description` | Старт Orchestrator или `/role-*` |
| Подхват rules / AGENTS.md | Передача отчётов между шагами |
| OpenSpec skills (`opsx-*`) | Разрешение на commit/push/deploy |
| Browser QA Task-цикл Worker→Reviewer→Tester (если parent Agent + Task) | Handoff `/role-*`, если nested Task недоступен |
| | Short path vs полный цикл (решает Orchestrator/пользователь) |
| | Live Trello sync / разрушительные действия с данными |

## Конфликты стека (зафиксировано)

- Целевой стек в OpenSpec/ADR: Go + Vue/Vite — согласован.  
- `trello_analytics_back/.env.example` упоминает `node scripts/...` — вспомогательный tooling, не стек приложения.  
- Код back/front пока почти отсутствует — агенты готовы к появлению кода.  
- Канон агентов в git: `trello_analytics_open_spec`; корень workspace не git — держать зеркала синхронными при правках.

## Синхронизация зеркал

После изменения агентов/rules/skills/commands в open_spec скопируйте `.cursor/agents`, `.cursor/skills/agent-handoff`, `.cursor/skills/browser-qa`, `.cursor/commands/role-*` и при необходимости `AGENTS.md` / rules в:

- `trello_analytics/.cursor/` (workspace root)
- `trello_analytics_back/.cursor/`
- `trello_analytics_front/.cursor/`

## Ограничения Cursor

- Нет гарантированного автозапуска всей цепочки ролей end-to-end.  
- Subagents не видят историю родительского чата — передавайте Task package и отчёты явно.  
- `readonly: true` ограничивает запись у Architect/Tester/Reviewer/Browser QA.  
- Параллельные агенты без изоляции могут конфликтовать по файлам.  
- Nested subagent часто не может запускать дочерних — Browser QA тогда работает в режиме handoff.
