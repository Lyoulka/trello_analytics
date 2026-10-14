# Change: Head catch-up for board actions

## Why

После завершённого primary backfill повторный sync обновляет сущности (`dateLastActivity` → `last_trello_update_at`), но **не** подтягивает новые board actions с головы ленты. Колонка «Обновлено» на `/cards` склеивает свежую дату карточки со **старым** автором из `card_actions` — расхождение с live Trello. Аналитика по событиям тоже отстаёт.

OpenSpec `trello-import-sync` явно отложил head catch-up как follow-up.

## What Changes

- При каждом `ImportBoard`: после entity refresh, если в БД уже есть actions — newest-first catch-up с головы ленты (`before` пагинация), insert `ON CONFLICT DO NOTHING`, **без** сдвига курсора oldest backfill.
- Затем существующий primary resume/backfill как раньше.
- Убрать status warning «no head catch-up»; обновить `docs/trello-integration.md` §8.4.1.
- Тесты importer + status.

## Impact

- `trello_analytics_back` — importer, postgres insert-only page, syncer warnings
- `trello_analytics_open_spec` — этот change + docs
- `trello_analytics_front` — **вне scope** (UI подтянет автора после sync)
