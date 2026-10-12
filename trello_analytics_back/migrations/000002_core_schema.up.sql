-- Product schema (ТЗ §9 / core-data-model).
-- Timestamps: TIMESTAMPTZ; store and write in UTC (ADR-005).
-- Raw Trello events live in card_actions; aggregate/metric tables are out of this migration.

-- boards
CREATE TABLE boards (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    trello_board_id TEXT NOT NULL UNIQUE,
    name            TEXT NOT NULL,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- trello_lists
CREATE TABLE trello_lists (
    id             UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    trello_list_id TEXT NOT NULL UNIQUE,
    board_id       UUID NOT NULL REFERENCES boards (id) ON DELETE RESTRICT,
    name           TEXT NOT NULL,
    is_archived    BOOLEAN NOT NULL DEFAULT false,
    logical_stage  TEXT NULL,
    created_at     TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at     TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_trello_lists_board_id ON trello_lists (board_id);

-- cards
CREATE TABLE cards (
    id                    UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    trello_card_id        TEXT NOT NULL UNIQUE,
    board_id              UUID NOT NULL REFERENCES boards (id) ON DELETE RESTRICT,
    name                  TEXT NOT NULL,
    description           TEXT NULL,
    url                   TEXT NULL,
    current_list_id       UUID NULL REFERENCES trello_lists (id) ON DELETE RESTRICT,
    created_at_trello     TIMESTAMPTZ NULL,
    is_archived           BOOLEAN NOT NULL DEFAULT false,
    last_trello_update_at TIMESTAMPTZ NULL,
    first_synced_at       TIMESTAMPTZ NULL,
    updated_at            TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_cards_board_id ON cards (board_id);

-- members
CREATE TABLE members (
    id               UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    trello_member_id TEXT NOT NULL UNIQUE,
    full_name        TEXT NULL,
    username         TEXT NULL,
    is_active        BOOLEAN NOT NULL DEFAULT true,
    created_at       TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at       TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- card_members (junction; composite PK)
CREATE TABLE card_members (
    card_id     UUID NOT NULL REFERENCES cards (id) ON DELETE CASCADE,
    member_id   UUID NOT NULL REFERENCES members (id) ON DELETE CASCADE,
    assigned_at TIMESTAMPTZ NULL,
    removed_at  TIMESTAMPTZ NULL,
    PRIMARY KEY (card_id, member_id)
);

-- labels
CREATE TABLE labels (
    id             UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    trello_label_id TEXT NOT NULL UNIQUE,
    board_id       UUID NOT NULL REFERENCES boards (id) ON DELETE RESTRICT,
    name           TEXT NULL,
    color          TEXT NULL,
    is_bug_label   BOOLEAN NOT NULL DEFAULT false
);

CREATE INDEX idx_labels_board_id ON labels (board_id);

-- card_labels (junction; composite PK)
CREATE TABLE card_labels (
    card_id  UUID NOT NULL REFERENCES cards (id) ON DELETE CASCADE,
    label_id UUID NOT NULL REFERENCES labels (id) ON DELETE CASCADE,
    PRIMARY KEY (card_id, label_id)
);

-- card_actions: immutable raw event store; UNIQUE(trello_action_id) for idempotent import
CREATE TABLE card_actions (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    trello_action_id    TEXT NOT NULL UNIQUE,
    card_id             UUID NULL REFERENCES cards (id) ON DELETE RESTRICT,
    action_type         TEXT NOT NULL,
    action_date         TIMESTAMPTZ NOT NULL,
    member_id           UUID NULL REFERENCES members (id) ON DELETE SET NULL,
    source_list_id      UUID NULL REFERENCES trello_lists (id) ON DELETE SET NULL,
    destination_list_id UUID NULL REFERENCES trello_lists (id) ON DELETE SET NULL,
    raw_payload         JSONB NULL,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_card_actions_action_date ON card_actions (action_date);
CREATE INDEX idx_card_actions_card_id ON card_actions (card_id);
CREATE INDEX idx_card_actions_action_type ON card_actions (action_type);

-- sync_runs
CREATE TABLE sync_runs (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    started_at      TIMESTAMPTZ NOT NULL,
    finished_at     TIMESTAMPTZ NULL,
    status          TEXT NOT NULL,
    processed_count INT NOT NULL DEFAULT 0,
    created_count   INT NOT NULL DEFAULT 0,
    updated_count   INT NOT NULL DEFAULT 0,
    error_count     INT NOT NULL DEFAULT 0,
    error_summary   TEXT NULL,
    CONSTRAINT sync_runs_status_check CHECK (status IN ('running', 'success', 'failed', 'cancelled'))
);

-- sync_cursors
CREATE TABLE sync_cursors (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    resource_type   TEXT NOT NULL UNIQUE,
    cursor_value    TEXT NULL,
    last_success_at TIMESTAMPTZ NULL,
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- app_users
CREATE TABLE app_users (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    email         TEXT NOT NULL UNIQUE,
    password_hash TEXT NOT NULL,
    display_name  TEXT NOT NULL,
    role          TEXT NOT NULL,
    is_active     BOOLEAN NOT NULL DEFAULT true,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT app_users_role_check CHECK (role IN ('admin', 'team_lead', 'member'))
);

-- app_settings: non-secret configuration only
-- FORBIDDEN: store Trello API token/key (or other secrets) as plaintext in value.
-- Trello credentials MUST come from environment / protected config only.
CREATE TABLE app_settings (
    key        TEXT PRIMARY KEY,
    value      TEXT NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_by UUID NULL REFERENCES app_users (id) ON DELETE SET NULL
);

COMMENT ON TABLE app_settings IS
    'Application settings. Do NOT store Trello API token/key or other secrets as plaintext in value; secrets via env / protected config only.';
