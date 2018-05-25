-- +goose Up
-- SQL in this section is executed when the migration is applied.
DROP TABLE IF EXISTS build_version_details;

-- +goose Down
-- SQL in this section is executed when the migration is rolled back.
CREATE TABLE build_version_details (
  id INT PRIMARY KEY DEFAULT unique_rowid(),
  build_version VARCHAR(256),
  repo_source VARCHAR(256),
  repo_full_name VARCHAR(256),
  repo_branch VARCHAR(256),
  repo_revision VARCHAR(256),
  manifest TEXT,
  inserted_at TIMESTAMP WITH TIME ZONE DEFAULT now(),
  UNIQUE INDEX build_version_details_build_version_repo_source_repo_full_name_idx (build_version, repo_source, repo_full_name)
);