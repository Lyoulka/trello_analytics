# Design: Head catch-up for board actions

## Context

Primary backfill пишет `sync_cursors.board_actions` на **oldest** action id. Повторный `ImportBoardActions` с `before=<oldest>` уходит дальше в прошлое (или получает пустую страницу) и **не** видит новые события у головы ленты.

## Decisions

1. **Когда:** head catch-up только если `CountCardActions > 0` (пустая БД → только primary, без двойного полного обхода).
2. **Порядок в `ImportBoard`:** entities → head catch-up (опционально) → primary `ImportBoardActions`.
3. **Пагинация:** как spike/primary — newest-first, `before=<last id on page>`. Старт с `before=""`.
4. **Persist:** insert-only page (`ON CONFLICT DO NOTHING`), **не** обновлять `sync_cursors` для `board_actions`.
5. **Стоп:** пустая страница; или `inserted == 0` на непустой (дошли до известных); или `len < pageLimit`.
6. **Курсор oldest:** не сбрасывать и не двигать head-фазой; primary resume без изменений.
7. **Warnings:** убрать «no head catch-up» из default status warnings.

## Risks

- [Gap между newest known и head] → Mitigation: catch-up идёт до первой all-duplicate page; idempotent insert.
- [Пустая БД + head] → Mitigation: gate `count > 0`.
- [Большой backlog новых actions] → Mitigation: тот же page size / retry, что primary; single-flight sync.

## Migration

Схема не меняется. Отдельная watermark-таблица не вводится.
