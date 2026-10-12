# Design

## Context

См. proposal.md. История в БД после `trello-import-sync`. ADR-004/006/007/008/009: метрики на backend; Cycle Time календарный; Throughput = уникальные карточки; workflow в БД; неполные данные явно.

## Goals / Non-Goals

**Goals:**

- Модель workflow settings + расчёт метрик из `card_actions` + текущего состояния.
- Документ формул `docs/analytics-metrics.md`.
- Unit-тесты граничных случаев.

**Non-Goals:**

- Active/working Cycle Time (OQ-010 / ADR-006).
- HTTP handlers и Vue (следующие changes).
- Закрытие OQ-003 жёстким правилом без настройки.

## Decisions

1. **Workflow config в БД.** Маппинг list → logical stage; множества start/done/blocked; bug label ids; stale threshold; timezone для границ периодов (хранение UTC, границы — в timezone настроек).
2. **Lead Time start (OQ-003).** Конфигурируемое правило: `first_enter_ready` | `card_created` | custom stage; если не задано — метрика Lead Time недоступна с warning.
3. **Completed / Throughput (ADR-007).** Уникальные карточки с переходом в done в периоде; перемещения между done-lists без возврата в work — не двойной учёт; reopen cycles — отдельно.
4. **Cycle Time.** Календарная разница first enter In Progress (или настроенный stage) → done; не называть «рабочим временем».
5. **Bug reopen.** Bug по label rule + возврат из done в work stage; процент по когорте завершённых багов (ТЗ §5.6).
6. **Attribution (OQ-007).** Team totals: карточка один раз; personal: карточка учитывается у каждого текущего/исторического assignee с явной пометкой в docs (не делить дробями в v1).
7. **Неполнота истории.** Если нет события start/done — карточка в «needs review» / исключается из метрики с warning, не с выдуманной датой.
8. **Версии workflow (OQ-006).** Минимум `updated_at` + `updated_by`; опционально JSON snapshot при сохранении settings для воспроизводимости отчёта.

## Risks / Trade-offs

- [Неверный маппинг списков] → Mitigation: UI/API warning если workflow неполный.
- [Label via updateCard idLabels] → Mitigation: интерпретация в importer/metrics с тестами; dedicated label actions если появятся.
- [Perf на больших actions] → Mitigation: индексы из data-model; агрегаты позже.

## Open Questions

- Default stale threshold и «data stale» sync age (OQ-008) — задать конфигурируемые дефолты в settings (например 7d / 24h) с возможностью смены Admin/Team Lead; финальные значения согласовать с тимлидом при apply.
