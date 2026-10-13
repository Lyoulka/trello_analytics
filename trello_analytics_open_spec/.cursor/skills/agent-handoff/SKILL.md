---
name: agent-handoff
description: >-
  Unified task handoff and report templates for Worker, Debugger, Tester,
  Reviewer, Architect, Orchestrator, and Browser QA. Use when starting or
  finishing a role, transferring work between agents, or writing acceptance
  summaries.
---

# Agent handoff

Read this skill before writing a role report. Copy the matching template; fill every section. Mark unknown items as `N/A` with a reason.

## Task package (Orchestrator → role)

```markdown
## Task package
- Change id:
- Task id(s) from tasks.md:
- Repository: trello_analytics_back | trello_analytics_front | trello_analytics_open_spec
- Role requested:
- Goal:
- Acceptance criteria:
- Constraints / out of scope:
- Prior reports attached: (none | Architect | Worker | …)
- Short path allowed: yes | no
```

## Worker Report

```markdown
## Worker Report
- Role: worker-go | worker-frontend | worker-db | worker-devops
- Change / task ids:
- Scope completed:
- Files changed:
- Checks run: (command → result; or "not run: reason")
- OpenSpec tasks.md updated: yes | no
- Risks / ambiguities:
- Ready for Tester/Reviewer: yes | no
```

## Debugger Report

```markdown
## Debugger Report
- Symptom:
- Reproduction: (steps | not reproducible: why)
- Root cause:
- Evidence:
- Fix (minimal):
- Regression test: added | N/A (why)
- Checks re-run:
- Residual risks:
```

## Tester Report

```markdown
## Tester Report
- Result: PASS | FAIL | BLOCKED
- Change / task ids:
- Scenarios covered: (positive / negative / boundary)
- Checks run:
- Defects (each FAIL):
  - Id:
  - Steps:
  - Expected:
  - Actual:
  - Severity:
- Coverage gaps:
- BLOCKED reason: (if applicable)
```

## Reviewer Report

```markdown
## Reviewer Report
- Overall: approve | request changes | request decision
- Change / task ids:

### Findings
#### Critical
- File / location:
- Problem:
- Impact:
- Recommended fix:

#### High
…

#### Medium
…

#### Low
…

### Security and architecture risks
-

### Missing tests
-

### Recommendation
принять | исправить | запросить решение
```

## Architect Decision

```markdown
## Architect Decision
- Question / change:
- Ambiguities resolved | still open:
- API / data contracts:
- Cross-repo dependencies:
- Risks (perf / security / ops):
- ADR / OpenSpec updates required:
- Ready for Worker: yes | no
```

## Orchestrator Summary

```markdown
## Orchestrator Summary
- Change / task ids:
- Acceptance criteria met: yes | no | partial
- Role statuses: Architect / Worker / Tester / Reviewer / Debugger / Browser QA
- Open Critical/High:
- Next action:
- Final: Done | Not done
- Notes: (no commit/push/deploy unless user approved)
```

## Defect Package (Browser QA → Worker)

```markdown
## Defect Package
- Id: BUG-<n>
- Severity: Critical | High | Medium | Low
- Change / OpenSpec refs:
- Repository (likely): trello_analytics_front | trello_analytics_back | …
- Worker role suggested: worker-frontend | worker-go | worker-db | worker-devops
- Steps to reproduce:
- Expected:
- Actual:
- Evidence: (screenshot paths / console / network / API status)
- Acceptance criteria to restore:
- Constraints: regression test required when feasible; no unrelated refactor; no live Trello sync without approval; no commit/push/deploy
- NEXT_ACTION: (Task auto | /worker-… manual)
```

## Bug Card (Browser QA)

```markdown
## Bug Card
- Id: BUG-<n>
- Description:
- Severity:
- Steps to reproduce:
- Files changed: (after Worker | none yet)
- Reviewer result: approve | request changes | N/A
- Tester result: PASS | FAIL | BLOCKED | N/A
- Browser re-check: pass | fail | not run
- Status: FIXED | OPEN | BLOCKED
- Residual risks:
```

## Browser QA Summary

```markdown
## Browser QA Summary
- App URL / version under test:
- Browser tool: cursor-ide-browser | Playwright | unavailable
- Delegation mode: Auto (Task) | Handoff (manual /role-*)
- Scenarios passed:
- Scenarios failed:
- Unverified areas:
- Bugs FIXED:
- Bugs OPEN:
- Bugs BLOCKED:
- Overall: PASS | FAIL | BLOCKED
- Next action:
- Notes: (no commit/push/deploy unless user approved)
```
