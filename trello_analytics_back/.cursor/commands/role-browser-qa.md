# Role: Browser QA

Запусти `/browser-qa` (readonly) для проверки **работающего** приложения в реальном браузере и цикла исправления дефектов.

1. Прочитай skill `browser-qa` и `agent-handoff`.
2. Подтверди Browser MCP / Playwright; без инструмента — **BLOCKED**, не имитируй проверки.
3. Сверь OpenSpec критерии приёмки; убедись, что UI доступен (`http://localhost:5173`, API `http://localhost:8080`).
4. Пройди реализованные сценарии (поведение, консоль, сеть, viewport).
5. На каждый подтверждённый баг: Defect Package → Worker → Reviewer → Tester → повтор в браузере.
6. Делегируй через Task subagents последовательно, если доступно; иначе выдай handoff с `NEXT_ACTION: /role-*` и дождись отчётов.
7. Заверши **Browser QA Summary**. Не правь код приложения сам. Не коммить/push/deploy.

Пример:

```text
/role-browser-qa Проверь локальный стенд по OpenSpec change X; исправь найденные UI-баги через цикл Worker/Reviewer/Tester
```
