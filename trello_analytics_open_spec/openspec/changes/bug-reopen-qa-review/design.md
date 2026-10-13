# Design: Bug reopen QA review

## Decision

1. **Bug** = label with `is_bug_label` (unchanged).
2. **`bug_reopen_from_stages`** — logical stages for «на проверку» (config; list map e.g. «Тестирование (на проверку)» → `qa_review`).
3. **Denominator** (`completed_bugs` in API): count of **list-move (or create) transitions into `done_stages`** in period for bug cards. done→done does not add.
4. **Numerator** (`reopened_bugs`): count of **list-move transitions** with `from ∈ bug_reopen_from_stages` and `to ∉ done_stages` in period for bug cards. Transition into done from qa_review is **not** reopen.
5. **%** = numerator/denominator×100 when denominator > 0; else omit percent (UI «—»).
6. Counts are **transitions**, not unique cards. Card ID lists in API remain unique cards that contributed ≥1 counted transition (for Reports).
7. Fix time: for each counted reopen transition, duration to next done-enter after it (if any).
