-- +goose Up
-- +goose StatementBegin
ALTER TABLE containers
    ADD COLUMN IF NOT EXISTS source TEXT NOT NULL DEFAULT 'docker',
    ADD COLUMN IF NOT EXISTS external_id TEXT NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS status TEXT NOT NULL DEFAULT 'UNKNOWN',
    ADD COLUMN IF NOT EXISTS labels JSONB NOT NULL DEFAULT '{}'::jsonb,
    ADD COLUMN IF NOT EXISTS last_seen_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    ADD COLUMN IF NOT EXISTS updated_at TIMESTAMPTZ NOT NULL DEFAULT now();

UPDATE containers SET external_id = id WHERE external_id = '';

ALTER TABLE alert_rules
    ADD COLUMN IF NOT EXISTS target_id TEXT,
    ADD COLUMN IF NOT EXISTS duration INTERVAL NOT NULL DEFAULT '0 seconds',
    ADD COLUMN IF NOT EXISTS updated_at TIMESTAMPTZ NOT NULL DEFAULT now();
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE alert_rules
    DROP COLUMN IF EXISTS target_id,
    DROP COLUMN IF EXISTS duration,
    DROP COLUMN IF EXISTS updated_at;

ALTER TABLE containers
    DROP COLUMN IF EXISTS source,
    DROP COLUMN IF EXISTS external_id,
    DROP COLUMN IF EXISTS status,
    DROP COLUMN IF EXISTS labels,
    DROP COLUMN IF EXISTS last_seen_at,
    DROP COLUMN IF EXISTS updated_at;
-- +goose StatementEnd
