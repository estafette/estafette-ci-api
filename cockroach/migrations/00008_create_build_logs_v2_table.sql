-- +goose Up
-- SQL in this section is executed when the migration is applied.
CREATE TABLE IF NOT EXISTS build_logs_v2 (
  id INT PRIMARY KEY DEFAULT unique_rowid(),
  repo_source VARCHAR(256),
  repo_owner VARCHAR(256),
  repo_name VARCHAR(256),
  repo_branch VARCHAR(256),
  repo_revision VARCHAR(256),
  steps JSONB,
  inserted_at TIMESTAMP WITH TIME ZONE DEFAULT now(),
  UNIQUE INDEX build_logs_v2_repo_source_repo_owner_repo_name_repo_revision_idx (repo_source, repo_owner, repo_name, repo_revision),
  INVERTED INDEX build_logs_v2_steps (steps)
);

-- +goose Down
-- SQL in this section is executed when the migration is rolled back.
DROP TABLE IF EXISTS builds_logs_v2;