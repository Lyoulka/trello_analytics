-- Workflow settings per board (ADR-008) + append-only revisions for audit snapshots.
-- List→stage SSOT remains trello_lists.logical_stage; bug flags remain labels.is_bug_label.

CREATE TABLE workflow_settings (
    id                          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    board_id                    UUID NOT NULL UNIQUE REFERENCES boards (id) ON DELETE RESTRICT,
    lead_time_start_rule        TEXT NOT NULL DEFAULT 'unset',
    lead_time_start_stage       TEXT NULL,
    cycle_start_stage           TEXT NULL,
    done_stages                 TEXT[] NOT NULL DEFAULT '{}',
    blocked_stages              TEXT[] NOT NULL DEFAULT '{}',
    work_stages                 TEXT[] NOT NULL DEFAULT '{}',
    stale_threshold_seconds     INT NOT NULL DEFAULT 604800,
    data_stale_sync_age_seconds INT NOT NULL DEFAULT 86400,
    timezone                    TEXT NOT NULL DEFAULT 'UTC',
    updated_at                  TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_by                  UUID NULL REFERENCES app_users (id) ON DELETE SET NULL,
    CONSTRAINT workflow_settings_lead_time_start_rule_check
        CHECK (lead_time_start_rule IN ('unset', 'card_created', 'first_enter_stage'))
);

CREATE TABLE workflow_settings_revisions (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    board_id      UUID NOT NULL REFERENCES boards (id) ON DELETE RESTRICT,
    settings_json JSONB NOT NULL,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    created_by    UUID NULL REFERENCES app_users (id) ON DELETE SET NULL
);

CREATE INDEX idx_workflow_settings_revisions_board_created
    ON workflow_settings_revisions (board_id, created_at DESC);

COMMENT ON TABLE workflow_settings IS
    'Board-level workflow rules and thresholds. List→stage mapping lives on trello_lists.logical_stage.';

COMMENT ON TABLE workflow_settings_revisions IS
    'Append-only JSON snapshots of workflow settings on each Save (ADR-008 / OQ-006).';
