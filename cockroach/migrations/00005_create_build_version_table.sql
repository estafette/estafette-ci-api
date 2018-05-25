-- +goose Up
-- SQL in this section is executed when the migration is applied.
CREATE TABLE IF NOT EXISTS build_versions (
  id INT PRIMARY KEY DEFAULT unique_rowid(),
  repo_source VARCHAR(256),
  repo_full_name VARCHAR(256),
  auto_increment INT DEFAULT 1,
  updated_at TIMESTAMP WITH TIME ZONE DEFAULT now(),
  UNIQUE INDEX build_versions_repo_source_repo_full_name_idx (repo_source, repo_full_name)
);

-- +goose Down
-- SQL in this section is executed when the migration is rolled back.
DROP TABLE IF EXISTS build_versions;