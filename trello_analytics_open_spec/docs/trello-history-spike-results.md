# Результаты эксперимента: история Trello API

**Change:** `spike-trello-history`  
**Статус:** заполнен по evidence sample + Architect Decision  
**Секреты / PII:** API key/token, имена участников, тексты комментариев/описаний и сырой JSONL **не** включаются. Только агрегаты и обезличенные id доски.

**Дисклеймер:** числа ниже — по конкретной выборке прогона; это не полная история доски и не гарантия поведения API во всех условиях.

---

## 1. Метод

- **Дата прогона:** 2026-10-12
- **Инструмент:** `trello_analytics_back` — CLI `cmd/trello-spike`, клиент `internal/trello` (read-only)
- **Credentials:** env only (не в Git, не в этом документе)
- **Доска:** shortLink `<BOARD_SHORTLINK>`, id `<TRELLO_BOARD_ID>` (реальные значения только в локальном env / evidence, не в Git)
- **Параметры board actions:** `limit=50`, 5 страниц; cursor `before=<last action id>`
- **Evidence (локально, не commit):** `trello_analytics_back/tmp/trello-spike/`

---

## 2. Выборка

| Показатель | Значение (sample) |
|------------|-------------------|
| Lists | 20 |
| Cards total | ≈3119 |
| Cards open | ≈944 |
| Cards archived (`closed`) | ≈2175 |
| Board actions загружено | 250 (5×50) |
| Min action date (board sample) | ≈`2026-10-07T07:44:37Z` |
| Max action date (board sample) | ≈`2026-10-12T11:41:23Z` |

Полный обход всех страниц до исчерпания API **не** выполнялся в рамках зафиксированного sample.

---

## 3. Наблюдения по типам actions (board sample)

| type | count | list before/after | Заметки |
|------|-------|-------------------|---------|
| `updateCard` | 200 | да (для list move) | List move: `listBefore`/`listAfter` + `old.idList`. Также pos/name/desc/`idLabels`. |
| `addMemberToCard` | 19 | — | Seen |
| `createCard` | 13 | — | Seen |
| `addAttachmentToCard` | 9 | — | Seen |
| `commentCard` | 8 | — | Текст в evidence redact; в docs не копируется |
| `removeMemberFromCard` | 1 | — | Seen |
| `addLabelToCard` | 0 | — | **не подтверждено** (`not_seen_in_sample`) |
| `removeLabelFromCard` | 0 | — | **не подтверждено** (`not_seen_in_sample`) |

**Доп. observation:** `updateCard` с полями `idLabels` / `old.idLabels` встречался в sample → для матрицы метрик: **нужна доп. интерпретация** (dedicated label types по-прежнему не подтверждены).

**`updateCard(closed)`:** в ключевых sample **не** зафиксирован → **не подтверждено**.

---

## 4. Пагинация и глубина

- **Механизм:** следующая страница board actions запрашивается с `before=<id последнего action текущей страницы>`.
- **Наблюдаемый диапазон дат sample:** ≈`2026-10-07T07:44:37Z` … ≈`2026-10-12T11:41:23Z`.
- **Полная глубина / усечение API:** **не подтверждено** (sample ≠ full board depth).
- **Лимит страницы в прогоне:** 50.

---

## 5. Архивные карточки и card `/actions`

- Endpoint actions для archived cards **отвечает** (HTTP успех на выборке).
- В compare-выборке у archived cohort встречались ненулевые `actions_fetched` (например 1–3 `updateCard` на карточку); у части open cards — `skipped_empty` / 0–1 action.
- **Card `/actions` часто sparse** относительно board feed (пример одной open card: 8 actions на `limit=20`, типы в основном `updateCard` + `commentCard`).
- Полнота card-vs-board для **всех** карточек — **не подтверждено**.

---

## 6. Ошибки и rate limits

| Сценарий | Результат |
|----------|-----------|
| Invalid token | HTTP **401** (`invalid app token`); credentials в URL маскируются |
| Несуществующий board id | HTTP **404** |
| Rate limit 429 | **`not_reproduced`** → поведение retry **не подтверждено** |

Рекомендация для будущего sync (гипотеза, не live-факт): при появлении 429 — backoff/retry с уважением к заголовкам лимита; до подтверждения не утверждать конкретные лимиты Trello.

---

## 7. Идемпотентность и resume

| Проверка | Результат |
|----------|-----------|
| Повтор той же страницы | `full_overlap=true`, overlap 100% id (`pass1=pass2=50`) |
| Алгоритм дедупа (эксперимент) | in-memory set `action.id`; продуктово — insert-ignore / upsert by `trello_action_id` |
| Resume | прерывание после 2 страниц → cursor `before` → продолжение; `resume_consistent=true`; missing/extra vs continuous control = 0 |

---

## 8. Выводы для продукта

**Можно считать достоверным на уровне sample:**

- Пагинация board actions через `before`
- Наличие типов: `createCard`, `updateCard` (в т.ч. list move), members add/remove, `commentCard`, `addAttachmentToCard`
- Observation: label changes могут приходить как `updateCard`+`idLabels`
- Archived card actions endpoint отвечает
- Идемпотентность по action id и resume без пропусков/лишних id в контрольном прогоне

**Показывать как ограничение / оставлять «не подтверждено»:**

- Полная глубина истории и возможное усечение API
- Dedicated `addLabelToCard` / `removeLabelFromCard`
- `updateCard(closed)` как archive event
- Live 429 / retry
- Полная completeness card-level vs board для всех cards

**Блокеры / следующие решения (не закрываются этим change):**

- OQ-003 Lead Time start-rule
- OQ-005…OQ-010 без изменений этим spike
- Production sync / analytics engine — вне scope
