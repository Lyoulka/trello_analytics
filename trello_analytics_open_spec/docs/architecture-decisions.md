# Архитектурные решения (ADR)

**Статус:** принятые решения зафиксированы; открытые вопросы вынесены отдельно.  
**Основание:** `docs/technical-specification.md` и утверждённый целевой стек.  
**Дата фиксации:** 2026-10-12.

Документ описывает решения для первой версии. Он не заменяет ТЗ и не предполагает факты о production-сервере или глубине истории Trello, пока они не подтверждены отдельно.

---

## 1. Принятые решения

### ADR-001. Целевой стек

| Слой | Решение |
|------|---------|
| Backend | Go, REST API (`/api/v1`) |
| Frontend | Vue 3 + TypeScript + Vite, Composition API, Vue Router |
| Database | PostgreSQL |
| Local development | Docker Compose (Postgres + backend + frontend dev) |
| Production | Существующий сервер: Nginx (static + proxy), Go как systemd-сервис, PostgreSQL без обязательного Docker |
| Requirements | OpenSpec store `trello-analytics` |
| IDE | Cursor |

**Почему:** соответствует утверждённому ТЗ; OpenSpec-контекст, ранее упоминавший Node.js/Nuxt 3, считается устаревшим и приведён к этому стеку.

### ADR-002. Структура репозиториев

- `trello_analytics_open_spec` — требования, changes, specs, проектная документация.
- `trello_analytics_back` — код backend (Go).
- `trello_analytics_front` — код frontend (Vue 3).

Спеки и changes не дублируются в код-репозиториях; там только `openspec/config.yaml` с `store: trello-analytics`.

### ADR-003. Неизменяемое хранение исходных событий Trello

События Trello сохраняются как исходные записи (включая идентификаторы Trello и при необходимости raw payload).  
Расчёт метрик не должен изменять или удалять исходные события.  
При смене формул или правил workflow метрики пересчитываются из сохранённой истории.

### ADR-004. Расчёт метрик только на backend

Frontend не дублирует бизнес-формулы аналитики.  
Все показатели (Throughput, Lead Time, Cycle Time, возвраты багов и т.д.) приходят из REST API backend.

### ADR-005. Время: UTC в хранилище, timezone в отображении

- В PostgreSQL временные отметки хранятся в согласованном UTC-формате.
- Отображение и границы отчётных периодов учитывают настроенный часовой пояс приложения.
- Часовой пояс задаётся в настройках, а не зашивается в код метрик.

### ADR-006. Cycle Time в v1 — календарный

В первой версии Cycle Time считается как календарная разница между отметками начала и завершения.  
Календарное время нельзя выдавать за «активное рабочее время».  
Модель исключаемых периодов / active time — вне обязательного scope v1.

### ADR-007. Throughput: уникальные карточки; циклы возврата отдельно

- Throughput и количество завершённых задач за период считают **уникальные карточки**, завершённые в периоде.
- Перемещение между списками завершения без возврата в работу не даёт двойного учёта.
- Повторные циклы завершения / возвраты учитываются в отдельной аналитике (reopened bugs / cycles), а не в базовом Throughput.

### ADR-008. Версионирование правил workflow

Соответствие списков Trello логическим этапам, правила start/done, метки багов и связанные исключения хранятся в БД.  
Изменения конфигурации версионируются (как минимум время изменения и автор; предпочтительно снимок правил, по которым считался отчёт), чтобы было понятно, по каким правилам получен результат.

### ADR-009. Неполные исторические данные — явно

Если API не отдаёт часть истории или событие неоднозначно:

- данные **не** подменяются предположениями;
- ограничение фиксируется в документации и статусе импорта;
- UI/API показывают предупреждение о качестве данных;
- неоднозначные события остаются в сырой истории, но не используются в расчётах, для которых они недостаточно достоверны.

### ADR-010. Одна доска Trello в v1

Подключается одна выбранная доска. Мульти-борд, мульти-workspace и произвольное подключение новых досок через UI — вне v1.

### ADR-011. Синхронизация

- Первичный импорт с пагинацией, resume и идемпотентностью (`UNIQUE` по `trello_action_id`).
- Регулярный фоновый sync + ручной запуск.
- Защита от параллельных конфликтующих sync-задач.
- Интервал sync задаётся конфигурацией.
- В v1 отдельный брокер сообщений не требуется.

### ADR-012. Авторизация и роли

Роли: Admin, Team Lead, Member.  
Проверка прав — на backend.  
Секреты Trello не возвращаются клиенту в открытом виде.  
Запросы к Trello выполняются только с backend.  
Модель входа v1 и онбординг пользователей — см. ADR-013.

### ADR-013. Онбординг пользователей v1 (локальные аккаунты, без публичной регистрации)

**Статус:** принято (подтверждено 2026-10-12).  
**Закрывает:** OQ-004.

В первой версии используется **собственная авторизация** с локальными паролями (`password_hash` в `app_users`), без внешнего IdP и без публичной саморегистрации.

Попадание пользователей в систему:

1. **Bootstrap первого Admin** при деплое/миграции (seed из env или одноразовый bootstrap-механизм), не через публичную форму.
2. **Admin создаёт остальных пользователей** в UI настроек: email, display name, роль, временный пароль (или аналог выдачи первичного доступа).
3. Пользователь входит через `POST /api/v1/auth/login` (далее `logout` / `me`).

Ограничения v1:

- Нет endpoint/экрана публичной регистрации.
- Участники Trello (`members`) не создают автоматически аккаунты приложения (`app_users`).
- Invite-ссылки / magic link / OAuth — вне обязательного scope v1 (можно добавить позже без ломки ролевой модели).

**Реализация сессии (auth-and-api):** HTTP-only cookie `tta_session` + таблица `sessions` в PostgreSQL (hash cookie value, absolute TTL default 7d). **JWT не используется.** Пароли — bcrypt cost 12. Контракты — `docs/api.md`.

### ADR-014. Dual sync entrypoint (CLI + HTTP)

**Статус:** принято (auth-and-api).

Operational board import shared via `internal/syncer.Orchestrator`:

- CLI `cmd/trello-sync` — ops / cron.
- HTTP `/api/v1/sync/*` on `cmd/server` — RBAC (Admin start/retry; Member+ status; Team Lead+ history).

No in-process periodic scheduler in v1. Trello credentials remain env-only.

### ADR-015. Frontend UI kit и графики (v1)

**Статус:** принято (frontend-analytics-ui, 2026-10-13).  
**Закрывает:** OQ-005.

| Тема | Решение |
|------|---------|
| UI kit | **shadcn-vue + Tailwind CSS** (Neutral tokens, компоненты копируются в `trello_analytics_front/src/components/ui/`). Без параллельных UI frameworks. |
| Графики | **Chart.js + vue-chartjs**; цвета серий из CSS-токенов `--chart-*`. |
| Метрики | Только из `/api/v1` (ADR-004). `null` / `available=false` ≠ `0`. |
| Time-series | Нет dedicated series endpoint в `docs/api.md`. UI строит дневной ряд через ≤31 суточных вызовов metric API **или** показывает Empty. Workload — bar chart из `workload[]` overview/workload. |
| Маршруты | `/login`; `/` → `/dashboard`; `/dashboard`; `/cards`; `/cards/:id/history`; `/reports`; `/settings`; `/users` (Admin). Guard: `GET /auth/me` + `meta.minRole`. |

---


## 2. Решения, требующие дополнительного подтверждения

Эти пункты **не** считаются полностью закрытыми (кроме отмеченных ниже как частично закрытые). Их нельзя «додумать» в коде без явного решения.

### Частично закрытые по итогам `spike-trello-history`

| ID | Статус | Вопрос | Что подтверждено (sample) | Что остаётся открытым |
|----|--------|--------|---------------------------|------------------------|
| OQ-001 | **частично закрыт** | Глубина и полнота истории actions Trello на целевой доске | Пагинация `before=<last action id>` подтверждена; на выборке 5×50 board actions диапазон дат ≈`2026-10-07T07:44:37Z` … ≈`2026-10-12T11:41:23Z` | Полная глубина истории доски и возможное усечение API — **не подтверждено** |
| OQ-002 | **частично закрыт** | Набор реально доступных типов actions (в т.ч. для архивных карточек) | Подтверждены типы: `updateCard`, `addMemberToCard`, `createCard`, `addAttachmentToCard`, `commentCard`, `removeMemberFromCard`; list move через `updateCard` (`listBefore`/`listAfter`); actions endpoint для archived cards отвечает (частично) | Dedicated `addLabelToCard`/`removeLabelFromCard`; `updateCard(closed)` как archive event; полнота card-level history для всех карточек — **не подтверждено** |

Детали: `docs/trello-integration.md`, `docs/trello-history-spike-results.md`.

### Открытые

| ID | Вопрос | Зачем нужно | Как закрыть |
|----|--------|-------------|-------------|
| OQ-003 | Точное определение начала Lead Time (Ready vs create vs иное правило команды) | Формула Lead Time | **Конфиг** `lead_time_start_rule` (default `unset` + warning); финальный product default — с тимлидом; см. `docs/analytics-metrics.md` |
| OQ-005 | Библиотека графиков на frontend | Зависимость UI до реализации Dashboard | **Закрыто (frontend-analytics-ui):** Chart.js + `vue-chartjs`. UI kit: **shadcn-vue + Tailwind CSS** (Neutral tokens; компоненты в `trello_analytics_front/src/components/ui/`). Time-series на клиенте — только через ≤31 суточных вызовов metric API или Empty; отдельного series/per-card chart endpoint в `docs/api.md` нет (gap осознанный, контракты backend не менялись). |
| OQ-006 | Формат версий workflow (только `updated_at` vs полные snapshots) | Воспроизводимость отчётов | **частично закрыт:** audit + `workflow_settings_revisions` snapshots; привязка отчёта к revision_id — later |
| OQ-007 | Правило attribution при нескольких исполнителях карточки в персональной аналитике | Workload по участникам | **Конфиг/docs:** team = 1×; personal = full count каждому assignee; см. `docs/analytics-metrics.md` |
| OQ-008 | Пороги «данных устарели» и «задача зависла» по умолчанию | UX предупреждений | **Проvisional defaults** 7d / 24h в settings; финальные UX-числа — с тимлидом |
| OQ-009 | Конкретные host/path/systemd unit/Nginx site на production-сервере | Deploy | Только после обследования существующей инфраструктуры (не выдумывать) |
| OQ-010 | Нужен ли в v1 active/working Cycle Time | Scope метрик | По умолчанию нет (ADR-006); пересмотр после реальных данных |

**Закрытые ранее:** OQ-004 → ADR-013; OQ-005 → Chart.js + vue-chartjs + UI kit shadcn-vue/Tailwind (`frontend-analytics-ui`, 2026-10-13).  
**Не закрывать этим change:** OQ-003, OQ-006…OQ-010.

---

## 3. Явно вне scope первой версии

- Несколько досок Trello
- Jira / YouTrack / GitHub / GitLab
- LLM-генерация текстов отчётов
- Уведомления в мессенджеры
- Рейтинги сотрудников
- Брокер сообщений
- Production через Docker Compose
- Публичная саморегистрация пользователей
- Внешний IdP / OAuth как обязательный вход v1

---

## 4. Связанные документы

- `docs/technical-specification.md` — полное ТЗ
- `docs/database.md` — продуктовая схема PostgreSQL (таблицы, ключи, индексы, UTC, секреты); DDL в `trello_analytics_back/migrations/`
- `docs/api.md` — REST `/api/v1` contracts (auth, analytics, cards, reports, settings, sync)
- `docs/analytics-metrics.md` — формулы метрик, exclusions, warnings, OQ-003/007/008
- `docs/trello-integration.md` — интеграция и ограничения истории (заполняется по итогам spike)
- OpenSpec changes (порядок реализации):
  - `openspec/changes/spike-trello-history/` — этап 1, research истории Trello
  - `openspec/changes/core-data-model/` — продуктовая схема PostgreSQL
  - `openspec/changes/trello-import-sync/` — импорт и синхронизация
  - `openspec/changes/workflow-and-metrics/` — workflow и расчёт метрик
  - `openspec/changes/auth-and-api/` — авторизация и REST `/api/v1`
  - `openspec/changes/frontend-analytics-ui/` — Vue UI аналитики
  - `openspec/changes/production-deploy/` — Nginx/systemd/ops docs
