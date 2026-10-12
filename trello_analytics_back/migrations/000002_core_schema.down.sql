-- Drop product schema in reverse FK dependency order.
-- Do NOT drop pgcrypto or touch bootstrap (000001).

DROP TABLE IF EXISTS app_settings;
DROP TABLE IF EXISTS app_users;
DROP TABLE IF EXISTS sync_cursors;
DROP TABLE IF EXISTS sync_runs;
DROP TABLE IF EXISTS card_actions;
DROP TABLE IF EXISTS card_labels;
DROP TABLE IF EXISTS labels;
DROP TABLE IF EXISTS card_members;
DROP TABLE IF EXISTS members;
DROP TABLE IF EXISTS cards;
DROP TABLE IF EXISTS trello_lists;
DROP TABLE IF EXISTS boards;
