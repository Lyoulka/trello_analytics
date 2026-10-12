# Trello Team Analytics — frontend (Vue 3)

См. канон: `trello_analytics_open_spec/AGENTS.md` и `docs/ai-agents.md`.

## Стек
- Vue 3 + TypeScript + Vite, Composition API, Vue Router (не Nuxt)
- Метрики только с backend `/api/v1`; null/missing ≠ 0
- OpenSpec store: `trello-analytics`

## Роли
`/worker-frontend`, `/orchestrator`, `/tester`, `/reviewer`, `/debugger`, `/architect`

Шаблоны: skill `agent-handoff`. Без commit/push/deploy без разрешения.
