-- Bug reopen source stages (e.g. qa_review = «Тестирование (на проверку)»).
-- Reopen = leave these stages into a non-done stage; % counts transitions, not unique cards.

ALTER TABLE workflow_settings
    ADD COLUMN IF NOT EXISTS bug_reopen_from_stages TEXT[] NOT NULL DEFAULT '{}';

COMMENT ON COLUMN workflow_settings.bug_reopen_from_stages IS
    'Logical stages whose leave→non-done is a bug reopen (transition count).';
