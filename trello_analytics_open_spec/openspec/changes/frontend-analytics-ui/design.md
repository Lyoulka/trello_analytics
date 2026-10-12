# Design

## Context

См. proposal.md. API контракты — `auth-and-api`. Frontend scaffold уже есть (Vue 3 + Vite + Router). ADR-004: формулы только с backend.

## Goals / Non-Goals

**Goals:**

- Полноценный UI v1: auth, dashboard, charts, cards, history, reports, settings, sync status.
- Единый API client и UX состояний.

**Non-Goals:**

- Дублирование формул метрик на клиенте.
- LLM-генерация текстов.
- Nuxt.

## Decisions

1. **Структура страниц:** `src/pages/` — Login, Dashboard, Cards, CardHistory, Reports, Settings, Users (Admin).
2. **State:** Composition API + лёгкий store при необходимости (Pinia, если уже не выбран — добавить Pinia для session/filters).
3. **Charts (OQ-005):** Chart.js + `vue-chartjs`.
4. **Фильтры периода:** общий composable (`from`/`to`/preset) согласованно для dashboard/charts/tables.
5. **Копирование отчёта:** клиент собирает текстовый preview из API `reports/preview` в clipboard.
6. **Роли в UI:** скрывать недоступные действия; решение всегда на API.
7. **Data quality banners:** показывать warnings из API (incomplete history, workflow missing, sync stale).

## Risks / Trade-offs

- [Расхождение UI прав и API] → Mitigation: опираться на `me.role` + обработку 403.
- [Большие таблицы] → Mitigation: серверная пагинация, не грузить всю историю на клиент.

## Open Questions

- Точный UI kit (нативный CSS vs лёгкая библиотека) — по умолчанию существующие стили проекта / минимальный CSS без тяжёлого UI framework.
