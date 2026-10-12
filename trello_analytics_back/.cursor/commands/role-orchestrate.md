# Role: Orchestrator

Запусти процесс координации задачи OpenSpec.

1. Прочитай `AGENTS.md`, `docs/ai-agents.md` и skill `agent-handoff`.
2. Работай как subagent `/orchestrator` (или следуй его инструкции).
3. Проверь готовность требований change; собери Task package.
4. Architect — только если нужны контракты/архитектура.
5. Назначь одного Worker; затем Tester и Reviewer.
6. Debugger — только при дефекте / падении тестов / интеграции.
7. Заверши **Orchestrator Summary**. Не коммить/push/deploy без разрешения пользователя.

Если задача простая (docs/config без смены контрактов) — допустим short path из `agent-team` rules.
