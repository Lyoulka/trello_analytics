# Design

## Context

См. proposal.md. Spike подтвердил пагинацию `before=<action id>`, ключевые типы actions, идемпотентность id и resume на sample; полная глубина, dedicated labels, `updateCard(closed)`, live 429 — не подтверждены (`docs/trello-integration.md`). Схема БД — change `core-data-model`.

## Goals / Non-Goals

**Goals:**

- Первичный импорт и инкрементальный sync одной доски в PostgreSQL.
- Идемпотентность, resume, журнал runs, single-flight sync.
- Статус качества данных (earliest/latest action, counts, errors, last success).

**Non-Goals:**

- Расчёт метрик и workflow UI.
- Полноценный публичный REST с auth (минимальный внутренний/admin sync status допустим; финальные endpoints — `auth-and-api`).

## Decisions

1. **Reuse `internal/trello` из spike.** Вынести production-ready client (timeouts, error mapping); CLI spike сохранить как tooling, не как sync entrypoint.
2. **Порядок импорта:** board → lists → members/labels → cards → board actions (пагинация) → при необходимости card actions для gaps; не выдумывать события.
3. **Идемпотентность:** `INSERT … ON CONFLICT (trello_action_id) DO NOTHING` (и аналоги для сущностей по trello id).
4. **Курсоры:** `sync_cursors.resource_type` + `cursor_value` (например last action id / page token); обновлять после успешного commit страницы.
5. **Single-flight:** advisory lock PostgreSQL или in-process mutex + статус `running` в `sync_runs`; повторный start не запускает второй активный sync.
6. **Scheduler:** интервал из config/env; в v1 без брокера (ADR-011).
7. **429:** retry с backoff по политике; если live 429 не воспроизводился — реализовать обработку кодов без утверждения частоты лимита.
8. **API контракт (минимальный, до auth change):**  
   - `POST /api/v1/sync/start` → `{ "run_id": "…" }` или 409 если уже running  
   - `GET /api/v1/sync/status` → last success, current run, earliest/latest action, counts, error summary  
   Полноценная авторизация навешивается в `auth-and-api`.

## Risks / Trade-offs

- [Неполная глубина истории] → Mitigation: статус + warnings; не заполнять gaps.
- [Card actions sparse vs board] → Mitigation: опираться на board feed как primary; card history с оговоркой.
- [Долгий первичный импорт] → Mitigation: resume + прогресс в sync_runs.

## Migration Plan

Зависит от применённых миграций `core-data-model`. Откат: остановка sync; данные событий сохраняются (не удалять при ошибке run).

## Open Questions

- Нужен ли отдельный worker-процесс vs goroutine в `cmd/server` — по умолчанию goroutine + scheduler в server process для v1.
