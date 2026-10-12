# Tasks

## Критерии приёмки change

1. Продуктовые таблицы из ТЗ §9 создаются миграциями после bootstrap.
2. UNIQUE на внешних Trello id; FK между связанными сущностями.
3. `card_actions` с UNIQUE(`trello_action_id`) и поддержкой raw payload.
4. Таблицы sync runs/cursors, app_users, app_settings присутствуют.
5. `docs/database.md` описывает схему; секреты Trello не в plaintext settings.
6. Frontend и sync/analytics код не входят в этот change.

---

## 1. Backend (trello_analytics_back)

- [x] 1.1 Добавить up-миграцию продуктовой схемы: boards, trello_lists, cards, members, card_members, labels, card_labels с UNIQUE по trello id и FK.
- [x] 1.2 В той же или следующей миграции добавить card_actions (UNIQUE trello_action_id, индексы date/card/type, raw_payload), sync_runs, sync_cursors.
- [x] 1.3 Добавить app_users (email, password_hash, role, is_active) и app_settings; в комментарии/доке запретить plaintext Trello token в settings.
- [x] 1.4 Добавить соответствующие down-миграции.
- [x] 1.5 Прогнать migrate-up на чистой локальной БД и migrate-down одной версии; зафиксировать результат в checks.

## 2. Frontend (trello_analytics_front)

- [ ] 2.1 Нет обязательных задач (UI вне scope).

## 3. Docs (trello_analytics_open_spec)

- [x] 3.1 Создать `docs/database.md`: таблицы, ключи, индексы, UTC, сырые события vs агрегаты, правило секретов.
- [x] 3.2 При необходимости кратко сослаться на схему из `docs/architecture-decisions.md` / local-development (без дублирования полного DDL).

## 4. Validation

- [x] 4.1 Сверить с критериями приёмки 1–6.
- [x] 4.2 Выполнить `openspec validate core-data-model --store trello-analytics` без ошибок.
