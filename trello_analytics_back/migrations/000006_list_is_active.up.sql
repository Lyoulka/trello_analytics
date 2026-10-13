-- Product flag: include cards currently in this list in analytics (list-analytics-active).
ALTER TABLE trello_lists
    ADD COLUMN IF NOT EXISTS is_active BOOLEAN NOT NULL DEFAULT true;

COMMENT ON COLUMN trello_lists.is_active IS
    'Product analytics inclusion for cards currently in this list; not overwritten by Trello sync';
