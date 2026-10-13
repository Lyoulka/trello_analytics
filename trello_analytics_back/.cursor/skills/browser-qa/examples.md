# Browser QA — examples

## Explicit start

```text
/browser-qa Проверь локальный стенд http://localhost:5173 по active OpenSpec change; баги гоняй через Worker/Reviewer/Tester
```

```text
/role-browser-qa Только сценарии Dashboard + фильтры периода; sync с Trello не запускать
```

## Auto vs handoff

**Auto** (parent Agent with Task):

1. browser-qa finds BUG-1 (filter clears table incorrectly).
2. Task → `worker-frontend` with Defect Package.
3. After Worker Report → Task → `reviewer`.
4. After approve → Task → `tester`.
5. Tester PASS → browser-qa re-runs filter scenario → FIXED.

**Handoff** (nested / no child Task):

```markdown
NEXT_ACTION: /worker-frontend
Attach: Defect Package BUG-1 below.
After Worker Report, reply here or run /reviewer then /tester.
```

## Defect severity guide

| Severity | Example |
|----------|---------|
| Critical | Login broken; blank app; data corruption UI |
| High | Wrong metric vs OpenSpec; filter ignores period |
| Medium | Broken empty state; console error on happy path |
| Low | Minor layout glitch; non-blocking a11y nit |

## BLOCKED examples

- Browser MCP tools missing / fail to connect.
- `localhost:5173` or `/ready` unreachable.
- Scenario needs admin user but credentials not provided.
- Fix requires live Trello full sync — awaiting user permission.
