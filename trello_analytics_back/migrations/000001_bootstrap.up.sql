-- Infrastructure bootstrap only: no business tables.
-- Enables gen_random_uuid() for future schemas; timestamps stay UTC at app layer.
CREATE EXTENSION IF NOT EXISTS pgcrypto;
