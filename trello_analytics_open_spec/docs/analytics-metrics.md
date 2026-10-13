# Analytics metrics

**Status:** formulas for change `workflow-and-metrics` (backend service layer).  
**Related:** [ADR-004](architecture-decisions.md)–[ADR-009](architecture-decisions.md), [trello-integration.md](trello-integration.md) §5 (event → metric matrix), OpenSpec `openspec/changes/workflow-and-metrics/`.

Timestamps in storage are UTC (ADR-005). Period boundaries and display use `workflow_settings.timezone`.

**Never invent missing events or dates.** Incomplete history or unfilled workflow → warnings / `needs_review`, not silent zeros presented as truth.

---

## Workflow configuration

| Concept | Storage |
|---------|---------|
| List → logical stage | `trello_lists.logical_stage` (SSOT; no separate mapping table) |
| Board rules / thresholds | `workflow_settings` (1 row per board) |
| Bug classification | `labels.is_bug_label` (set on Save) |
| Audit | `updated_at` / `updated_by` + append-only `workflow_settings_revisions` |

### Lead Time start rule (OQ-003)

Configurable; **not** a hardcoded Ready/create default:

| Rule | Meaning |
|------|---------|
| `unset` (default) | Lead Time unavailable + warning `lead_rule_unset` |
| `card_created` | Start = `createCard` action date, else `cards.created_at_trello` if present |
| `first_enter_stage` | Start = first enter of `lead_time_start_stage` |

### Defaults (OQ-008 provisional)

| Setting | Default |
|---------|---------|
| `stale_threshold_seconds` | 604800 (7d) |
| `data_stale_sync_age_seconds` | 86400 (24h) |
| `timezone` | `UTC` |

Final UX numbers remain open with the team lead; changing settings does not require a code change.

### Multi-assignee (OQ-007)

- **Team totals:** a card counts once.
- **Personal / workload:** the card is counted in full for **each** current assignee (no fractional split in v1).

OQ-003 / OQ-007 / OQ-008 are closed by **configuration + warnings**, not by a hard-coded product rule without signalling.

---

## Stage transitions

Source: `card_actions` with `raw_payload` (importer may leave `card_id` / list FKs null).

| Action | Use |
|--------|-----|
| `updateCard` list move (`listBefore`/`listAfter` or `old.idList` → list) | Stage timeline |
| `createCard` | Lead rule `card_created`; optional initial list stage |
| Other `updateCard` | Ignored for stage timeline |
| `updateCard(closed)` archive | **Not** used as a metric timestamp (not confirmed in spike) |

Resolve: `data.card.id` → `cards.trello_card_id`; Trello list id → `trello_lists.logical_stage`.

---

## Formulas

### Completed / Throughput (ADR-007)

- Count **unique** cards whose **first** enter into a stage ∈ `done_stages` from a non-done stage falls in the period `[start, end)`.
- **done → done** (no return to work) does **not** double-count.
- Re-completion after return to work still counts as **one** card in Throughput for the period of the first done-enter in that period; reopen cycles are separate.

Throughput for a period equals the completed unique-card count (tasks completed, not number of Trello actions).

### Lead Time

- Duration from configured start rule to the **first** done-enter in the period (same event as Completed / Throughput), not a later re-completion in that period.
- Mean / median over cards with both boundaries.
- Missing start or done → exclude card + warning (`missing_start` / `missing_done` / `incomplete_history`); no invented dates.

### Cycle Time (calendar, ADR-006)

- Calendar difference: first enter `cycle_start_stage` → **first** done-enter in the period (same event as Completed).
- Must **not** be labelled as active working time.
- Missing cycle start config → warning `missing_cycle_start`; metric unavailable.
- Missing event → exclude + warning.

### Bug reopen + fix time (продуктовое правило команды)

- Bug = card has a label with `is_bug_label = true` (лейбл БАГ).
- Config: `bug_reopen_from_stages` — logical stages «на проверку» (пример: список «Тестирование (на проверку)» → `qa_review`).
- **Reopen (числитель)** = **число переходов** (list move) у багов в периоде: `from ∈ bug_reopen_from_stages` и `to ∉ done_stages` (в т.ч. обратно в «Тестирование»). Уход из «на проверку» **в Done** reopen’ом **не** считается.
- **Знаменатель** = **число переходов** у багов в периоде **в** `done_stages` (Готово / HORECA.RU - готово и др. с stage ∈ done); **done → done** не увеличивает счётчик.
- Reopen % = `(reopen transitions) / (bug done-enter transitions) × 100%`. Не уникальные карточки.
- Пустой знаменатель → % недоступен (не выдавать `0%` как истину).
- Fix time = от каждого учтённого reopen до следующего входа в done (если есть); несколько циклов — отдельные наблюдения.

### WIP / stale

- WIP: non-archived cards whose current stage is **not** in `done_stages`.
- Blocked count: subset with stage ∈ `blocked_stages`.
- Stale: WIP cards with no Trello activity longer than `stale_threshold_seconds` (last action / `last_trello_update_at`). Phrase as **absence of activity in Trello**, not proof the person was idle.
- No last-activity timestamp → warning; do not invent a date to force stale.
- Empty `done_stages` → WIP and Stale are **unavailable** (not trustworthy), not silent full-board WIP.

### Workload

- Per member: active (WIP) and completed-in-period cards among current assignees.
- Multi-assignee: full count on each assignee (OQ-007).

### Data stale

- If no successful sync, or last success older than `data_stale_sync_age_seconds` → warning `data_stale`.

---

## Warning codes

| Code | When |
|------|------|
| `workflow_incomplete` | Required workflow pieces missing |
| `lead_rule_unset` | Lead rule is `unset` |
| `missing_cycle_start` | `cycle_start_stage` empty |
| `missing_done_stages` | `done_stages` empty |
| `missing_work_stages` | `work_stages` empty |
| `incomplete_history` | History insufficient for a calculation |
| `missing_start` | No trustworthy start event |
| `missing_done` | No trustworthy done event |
| `ambiguous_reopen` | Leave-done event not clearly work reopen |
| `data_stale` | Sync age / missing success |

Incomplete workflow: related metrics are **not** returned as trustworthy stats (unavailable / empty with warnings), not silent zero-as-truth.

---

## Exclusions and edge cases

| Case | Behaviour |
|------|-----------|
| Archived card | Excluded from WIP/stale |
| Period before earliest imported action | Treat as incomplete; surface `incomplete_history` / period quality warnings when applicable |
| Unmapped list (`logical_stage` NULL) | Stage empty; may block start/done detection → exclude + warning |
| Active Cycle Time | Out of scope (OQ-010 / ADR-006) |

---

## Implementation

- Go: `trello_analytics_back/internal/workflow`, `trello_analytics_back/internal/analytics` (`Engine.Compute`).
- HTTP/OpenAPI/UI: later changes (`auth-and-api`, `frontend-analytics-ui`).
