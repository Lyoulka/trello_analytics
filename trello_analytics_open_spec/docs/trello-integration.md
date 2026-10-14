# Интеграция с Trello

**Статус:** результаты spike `spike-trello-history` (выборка, не полный обход доски).  
Остаются caveats: полная глубина истории, dedicated label types, `updateCard(closed)`, live 429 и полнота card-level history — **не подтверждено**.  
Операционный sync (env, порядок импорта, resume, CLI) — §8 / changes `trello-import-sync` + `trello-sync-cli`; факты spike ниже не переписаны.

Связанные артефакты:

- OpenSpec change: `openspec/changes/spike-trello-history/`
- OpenSpec change: `openspec/changes/trello-import-sync/`
- OpenSpec change: `openspec/changes/trello-sync-cli/`
- Детальный отчёт эксперимента: `docs/trello-history-spike-results.md`
- OQ: `docs/architecture-decisions.md` (OQ-001 / OQ-002 частично закрыты)

**Дисклеймер:** все числа и типы ниже — по sample прогона CLI spike на одной доске; это не утверждение о полном объёме истории Trello API.

---

## 1. Цель интеграции

Подключить **одну** доску Trello, сохранить карточки, списки, участников, метки и доступную историю действий так, чтобы backend мог рассчитывать метрики доставки и качества без обращения к Trello на каждый запрос UI.

---

## 2. Принципы

1. Запросы к Trello выполняются только с backend.
2. Исходные события хранятся неизменяемо (идемпотентно по `trello_action_id`).
3. Отсутствующие данные не подменяются предположениями.
4. Ограничения истории явно документируются и показываются пользователю.
5. Импорт поддерживает пагинацию, resume и повторный запуск без дублей.

---

## 3. Что исследовано (чеклист spike)

| Тема | Статус | Результат (sample) |
|------|--------|-------------------|
| Доступные типы actions | ✅ частично | В board sample (250): `updateCard` 200, `addMemberToCard` 19, `createCard` 13, `addAttachmentToCard` 9, `commentCard` 8, `removeMemberFromCard` 1. Dedicated `addLabelToCard` / `removeLabelFromCard` — **не подтверждено** (не встретились). |
| Глубина истории | ⚠️ частично | Min≈`2026-10-07T07:44:37Z`, max≈`2026-10-12T11:41:23Z` на 5 страницах board actions. Полная глубина / усечение API — **не подтверждено**. |
| Пагинация и лимиты объёма | ✅ подтверждено (механизм) | Cursor: `before=<id последнего action на странице>`; page size `limit=50`; 5 страниц → 250 actions. |
| Actions для архивных карточек | ✅ частично | Endpoint card `/actions` для archived отвечает; в выборке есть ненулевые `actions_fetched`. Полнота vs board / все карточки — **не подтверждено**. Card `/actions` часто sparse относительно board. |
| Rate limits / ошибки API | ✅ частично | `401` (invalid token), `404` (bad board id) подтверждены. `429` — `not_reproduced` → **не подтверждено**. |
| Идемпотентность повторной загрузки | ✅ подтверждено (sample) | Повтор той же страницы: overlap id = 100% (`full_overlap=true`). |
| Resume после прерывания | ✅ подтверждено (sample) | `resume_consistent=true`; missing/extra vs control = 0. |
| Матрица «событие → метрика» | ✅ заполнена | См. §5; значения только по подтверждённым наблюдениям + Architect Decision. |
| Ограничения для UI | ✅ заполнены | См. §6. |

---

## 4. Типы / сценарии событий (по факту sample)

| Тип / сценарий | Зачем нужен | Проверено | Наблюдение |
|----------------|-------------|-----------|------------|
| `createCard` | Lead Time от create (если выбрано, OQ-003) | ✅ seen | Есть в board sample (13). |
| `updateCard` list move (`listBefore`/`listAfter`, `old.idList`) | Этапы, start/done, Cycle/Lead | ✅ seen | List move подтверждён (в sample десятки move). |
| `updateCard` с `idLabels` / `old.idLabels` | Возможный сигнал смены меток | ✅ seen (observation) | Встречается; **нужна доп. интерпретация** для метрик. Dedicated label types — **не подтверждено**. |
| `updateCard(closed)` архив | Архивация / исключения | ⏳ не подтверждено | В ключевых sample **не** зафиксирован event с `old.closed`. |
| `addMemberToCard` / `removeMemberFromCard` | Workload, история исполнителей | ✅ seen | Оба типа в board sample. |
| `addLabelToCard` / `removeLabelFromCard` | Классификация багов | ⏳ не подтверждено | `not_seen_in_sample`. |
| `commentCard` | Опционально активность | ✅ seen | В sample есть. |
| `addAttachmentToCard` | Прочий тип из API | ✅ seen | В sample есть; для core-метрик не используется. |

---

## 5. Матрица «событие Trello → возможность расчёта метрики»

Легенда: `да` / `частично` / `нет` / `нужна доп. интерпретация` / `не подтверждено`.

| Событие / данные | Completed / Throughput | Lead Time | Cycle Time | Bug reopen | Fix time reopen | Stale / WIP | Workload |
|------------------|------------------------|-----------|------------|------------|-----------------|-------------|----------|
| `createCard` | нет | нужна доп. интерпретация (OQ-003) | нет | нет | нет | нет | нет |
| `updateCard` (list move) | частично | частично / нужна доп. интерпретация | частично | нужна доп. интерпретация | нужна доп. интерпретация | частично | нет |
| `updateCard(closed)` | не подтверждено | не подтверждено | не подтверждено | не подтверждено | не подтверждено | не подтверждено | не подтверждено |
| `addMemberToCard` / `removeMemberFromCard` | нет | нет | нет | нет | нет | нет | частично |
| `addLabelToCard` / `removeLabelFromCard` (dedicated) | нет | нет | нет | нет | нет | нет | нет |
| `updateCard` (`idLabels`) | нет | нет | нет | нужна доп. интерпретация | нужна доп. интерпретация | нет | нет |
| `commentCard` | нет | нет | нет | нет | нет | нет | нет |
| `addAttachmentToCard` | нет | нет | нет | нет | нет | нет | нет |

---

## 6. Ограничения, которые нужно показывать пользователю

Обязательный список предупреждений / статусов (на базе spike; конкретика дат — по фактическому импорту):

1. **Окно доступной истории в sample ограничено.** Board actions на 5×50 страницах покрыли примерно `2026-10-07` … `2026-10-12` (UTC). Полная глубина доски **не подтверждена** — нельзя обещать метрики «за всё время доски» без полного импорта и проверки усечения API.
2. **Периоды до самой ранней загруженной даты события** считаются неполными; Lead/Cycle/Throughput за такие периоды — с предупреждением о пробелах.
3. **Метрики, завязанные на archive (`closed`) и dedicated label actions**, пока ограничены или недоступны: событие архивации и dedicated label types в sample **не подтверждены**.
4. **Card-level `/actions` часто sparse** относительно board feed; полнота истории по всем карточкам **не подтверждена**. UI не должен выдавать card timeline за полный lifecycle без оговорок.
5. **Ошибки импорта:** подтверждены сценарии `401` / `404`; поведение `429`/retry в live **не подтверждено** — при сбое sync показывать статус ошибки без утечки секретов.
6. **Время последней успешной синхронизации** и признак устаревания данных — обязательны в UI/API (пороги — OQ-008, вне этого spike).
7. **Идемпотентность и resume** на уровне эксперимента подтверждены (overlap 100%, `resume_consistent=true`); это не снимает требование показывать прогресс/пропуски при реальном sync.

---

## 7. Результаты эксперимента (кратко)

- **Дата эксперимента:** 2026-10-12 (прогон spike CLI).
- **Доска:** shortLink `<BOARD_SHORTLINK>` / id `<TRELLO_BOARD_ID>` (реальные значения только в локальном env / evidence; в docs без id доски и персональных данных участников).
- **Метод:** Go CLI `cmd/trello-spike` + `internal/trello`; credentials только из env; артефакты в `tmp/trello-spike/` (не в Git).
- **Выборка board:** lists=20; cards≈3119 (open≈944, archived≈2175); board actions sample 5×50=250.
- **Краткий вывод:** пагинация `before`, ключевые типы (`createCard`, list move `updateCard`, members, comments, attachments), идемпотентность и resume — подтверждены на sample. Полная глубина, archive-closed event, dedicated labels, 429 и полная card completeness — остаются **не подтверждено**.
- **Детали:** `docs/trello-history-spike-results.md`

---

## 8. Operational sync (`trello-import-sync` + `trello-sync-cli`)

Производственный импорт одной доски в PostgreSQL (backend). **Два entrypoint** (dual path, shared `internal/syncer.Orchestrator`):

1. **CLI** `cmd/trello-sync` — ops / cron / local operator.
2. **HTTP** `POST/GET /api/v1/sync/*` на `cmd/server` — Admin/Lead/Member по RBAC (см. `docs/api.md`).

In-server periodic scheduler **не** используется в v1. Секция описывает config и поведение sync; наблюдения spike (§3–§7) не изменяет.

### 8.1 Entrypoint (CLI)

Из каталога `trello_analytics_back`:

```bash
go run ./cmd/trello-sync run      # default; sync до завершения
go run ./cmd/trello-sync status   # JSON статуса качества данных в stdout
```

Без аргументов эквивалентно `run`. Spike CLI (`cmd/trello-spike`) — research tooling, не operational sync.

**Exit codes (`run`):** `0` — успех; `2` — usage/config; `3` — уже running (single-flight); `4` — failed; `5` — cancelled.

### 8.2 Env

| Переменная | Назначение |
|------------|------------|
| `DATABASE_URL` | PostgreSQL DSN (обязателен для CLI) |
| `TRELLO_API_KEY` | API key (только backend / env) |
| `TRELLO_TOKEN` | Token (только backend / env) |
| `TRELLO_BOARD_ID` | Id или shortLink доски |
| `TRELLO_ACTIONS_PAGE_SIZE` | Размер страницы board actions (по умолчанию `100`, clamp `1..1000`) |

`TRELLO_SYNC_INTERVAL` **не** используется: CLI и API его не читают; фоновый scheduler не поднимается. Без `DATABASE_URL` / key / token / board id CLI `run` не стартует. Для CLI `status` credentials Trello не обязательны (нужны DB + board id). HTTP sync на server требует тех же Trello env, иначе endpoints отвечают `503 sync_unavailable`.

Ops pattern: cron / systemd timer на CLI и/или Admin UI → HTTP start (unit — вне этого change).

### 8.3 Порядок импорта

В одном run: **board → lists → members/labels → cards → board actions** (пагинация). События не выдумываются; идемпотентность по trello ids (`ON CONFLICT`).

**Assignees / labels на карточках (snapshot):** при entity refresh текущий состав берётся из полей карточки Trello `idMembers` и `idLabels` (GET `/boards/{id}/cards`) и синхронизируется в `card_members` / `card_labels`. Это **не** восстановление истории назначений из actions (`addMemberToCard` и т.п.): `assigned_at` остаётся NULL, если API не отдаёт дату; снятый member помечается `removed_at`; для `card_labels` отсутствующие пары удаляются (в схеме нет soft-remove). Change: `openspec/changes/sync-card-assignees/`.

### 8.4 Resume (board actions)

- Курсор в `sync_cursors`: `resource_type=board_actions`, `cursor_value` = id последнего успешно закоммиченного action.
- Следующий запрос к Trello: `before=<cursor_value>` (как в spike).
- Курсор обновляется только после успешного commit страницы; сбой mid-page оставляет предыдущий курсор.
- Пустая доска / успешный прогон с 0 actions: строка `sync_cursors` для `board_actions` всё равно создаётся (`cursor_value` NULL), чтобы отметить завершение primary backfill.

### 8.4.1 После primary + head catch-up

После успешного первичного backfill курсор остаётся на **oldest** action id (или NULL при 0 actions). Повторные sync:

- **делают** refresh сущностей (board/lists/members/labels/cards);
- если в БД уже есть `card_actions` — **head catch-up**: страницы с головы ленты (newest-first, `before`), insert `ON CONFLICT DO NOTHING`, **без** обновления курсора oldest;
- стоп head-фазы: пустая страница, или страница без новых insert’ов, или короткая страница;
- затем primary resume дальше в прошлое через `before=<cursor>`, если backfill ещё не завершён.

Отдельная watermark-таблица не используется. Change: `openspec/changes/sync-actions-head-catchup/`.

### 8.5 Single-flight

- In-process mutex + статус `running` в `sync_runs`.
- Повторный `trello-sync run` при активном run → exit `3` (второй sync не стартует).
- При рестарте процесса orphan `running` reclaim → failed с пометкой orphaned, затем возможен новый start.

### 8.6 Retry

Временные ошибки: `429`, `5xx`, network/timeout. Политика: max **5** попыток на запрос; exponential backoff + jitter; при `429` учитывается `Retry-After`, если заголовок есть.  
`401` / `403` / `404` — без retry.  
Live `429` в spike — `not_reproduced` (§3); политика реализует HTTP retry semantics, не утверждает частоту rate limit Trello.

### 8.7 Status (CLI)

`go run ./cmd/trello-sync status` печатает JSON высокого уровня: `board_id`, `last_success_at`, `current_run` (id/status/counts/error_summary), `earliest_action_at` / `latest_action_at`, `counts` (cards/lists/members/labels/actions), `primary_import_incomplete`, `warnings`. Секреты в stdout/логах не отдаются.

HTTP sync снова в product path (`auth-and-api`): `POST /api/v1/sync/start|retry` (Admin), `GET /status` (Member+), `GET /history` (Team Lead+). Контракты — `docs/api.md`. CLI остаётся полноценным entrypoint.

### 8.8 Источник истории в v1

**Board actions feed — primary.** Card-level `/actions` в v1 **не используется** (card feed часто sparse относительно board; см. §6).
