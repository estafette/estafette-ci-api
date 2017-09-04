-- +goose Up
-- SQL in this section is executed when the migration is applied.
ALTER TABLE build_logs ALTER id SET DEFAULT unique_rowid();

-- +goose Down
-- SQL in this section is executed when the migration is rolled back.
ALTER TABLE build_logs ALTER id DROP DEFAULT;