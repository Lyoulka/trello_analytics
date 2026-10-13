---
name: browser-qa
description: >-
  Browser product QA workflow for Trello Team Analytics: OpenSpec acceptance
  checks in a real browser, defect packages, and sequential Worker → Reviewer →
  Tester → browser re-check loops. Use when running /browser-qa, /role-browser-qa,
  or end-to-end UI verification.
---

# Browser QA

## Before you start

1. Read OpenSpec change: `proposal` / `design` / `specs` / `tasks.md` (store `trello-analytics`).
2. Probe browser automation:
   - Prefer MCP `cursor-ide-browser` (`browser_navigate`, `browser_snapshot`,
     `browser_lock`, `browser_take_screenshot`, `browser_cdp`, …).
   - Else Playwright if installed and usable from this session.
   - If neither works → stop with **BLOCKED** (do not simulate).
3. Probe app:
   - Frontend: `http://localhost:5173`
   - Backend ready: `http://localhost:8080/ready`
   - Details: `trello_analytics_open_spec/docs/local-development.md`

## Scenario checklist

Cover only what is implemented; mark the rest **unverified**.

- [ ] Auth / session / role-gated UI (if present)
- [ ] Navigation / routing
- [ ] Tables: sort, pagination, empty
- [ ] Filters / period / timezone display
- [ ] Forms: validation, submit, error
- [ ] Charts / metric cards: loading, null≠0, warnings
- [ ] Reports
- [ ] Settings / workflow config
- [ ] Sync status UI (do **not** trigger live Trello sync without approval)
- [ ] Console: no unexpected errors
- [ ] Network: no unexpected 4xx/5xx on happy path
- [ ] Viewport: desktop + narrow (~375px)

For each step: expected vs actual from OpenSpec; capture evidence on failure.

## MCP browser sequence

```text
browser_tabs(list) → browser_navigate(url) → browser_lock(lock)
→ browser_snapshot → interactions (click/fill/type/…)
→ browser_take_screenshot on defects
→ browser_cdp / Runtime for console & failed requests when needed
→ browser_lock(unlock) when fully done
```

Do not use CDP `Input.*`. Prefer snapshot refs over raw coordinates.

## Fix loop (one defect at a time)

```text
Diagnose → Worker → Reviewer → Tester → (fix loop if needed) → Browser re-check → Status
```

| Step | Role | Must produce |
|------|------|----------------|
| Diagnose | browser-qa | Defect Package |
| Fix | one `worker-*` | Worker Report + regression test when feasible |
| Review | reviewer | Reviewer Report; no blocking Critical/High to continue |
| Test | tester | Tester Report PASS |
| Re-check | browser-qa | same scenario + related flows OK |

Max **3** Worker iterations per bug, then escalate (user / Architect).

### Task delegation prompt skeleton

Pass all context (subagents have no parent chat history):

```markdown
## Task package
- Change id:
- Task id(s): browser-qa defect <id>
- Repository: trello_analytics_front | trello_analytics_back | …
- Role requested: worker-frontend | …
- Goal: fix defect <id>
- Acceptance criteria: (from OpenSpec + expected behavior)
- Constraints: add regression test; no unrelated refactors; no commit/push/deploy
- Prior reports attached: Defect Package (+ later Worker/Reviewer/Tester)
```

## Safety

- No app code edits by browser-qa.
- No live Trello sync / destructive DB ops without user OK.
- No secrets in reports (redact tokens/passwords).
- No commit/push/deploy.

## Reports

Use templates in skill `agent-handoff`: Defect Package, Bug Card, Browser QA Summary.

## Additional resources

- Handoff examples: [examples.md](examples.md)
