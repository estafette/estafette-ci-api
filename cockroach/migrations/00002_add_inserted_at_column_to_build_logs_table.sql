-- +goose Up
-- SQL in this section is executed when the migration is applied.
ALTER TABLE build_logs ADD inserted_at TIMESTAMP DEFAULT now();

-- +goose Down
-- SQL in this section is executed when the migration is rolled back.
ALTER TABLE build_logs DROP inserted_at;