-- +goose Up
-- SQL in this section is executed when the migration is applied.
CREATE INDEX builds_logs_repo_full_name_repo_branch_repo_revision_repo_source_idx ON build_logs (repo_full_name, repo_branch, repo_revision, repo_source);

-- +goose Down
-- SQL in this section is executed when the migration is rolled back.
DROP INDEX build_logs@builds_logs_repo_full_name_repo_branch_repo_revision_repo_source_idx;