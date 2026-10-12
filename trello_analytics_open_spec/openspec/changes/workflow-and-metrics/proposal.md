# Proposal

## Why

Сырые события Trello сами по себе не дают Throughput, Lead/Cycle Time и возвраты багов. Нужны настраиваемый workflow и прозрачные формулы на backend по сохранённой истории.

## What Changes

- Сопоставление списков Trello логическим этапам; правила start/done; метки багов; порог stale.
- Расчёт метрик: completed, Throughput, Lead Time, Cycle Time (календарный), reopen bugs, fix time, stale/WIP, workload.
- Документ `docs/analytics-metrics.md` с формулами и граничными случаями.
- Предупреждения при незаполненном workflow или неполной истории; запрет выдуманных дат.

## Capabilities

### New Capabilities

- `workflow-settings`: конфигурация этапов, start/done, bug labels и связанных правил в БД.
- `analytics-metrics`: расчёт командных метрик из сохранённых событий по правилам workflow.

### Modified Capabilities

- (нет)

## Impact

**Затронутые репозитории:**

- `trello_analytics_back` — domain analytics + settings; unit-тесты формул.
- `trello_analytics_open_spec` — artifacts и `docs/analytics-metrics.md`.
- `trello_analytics_front` — **вне scope** (отображение — в `frontend-analytics-ui`; HTTP-обёртки — в `auth-and-api`).

**Вне scope:** полноценный REST/OpenAPI (кроме внутренних сервисов), UI, sync importer.

**Открытые OQ:** OQ-003/007/008 остаются настраиваемыми; не закрывать «жёстким» дефолтом без предупреждения в UI/API.
