-- !! Note !! These queries are run manually before migration is started

-- modify builds for migrations
CREATE UNIQUE INDEX unique_build ON builds (repo_source, repo_owner, repo_name, started_at);
ALTER TABLE builds ADD migrated_from BIGINT;

-- modify build_logs for migrations
CREATE UNIQUE INDEX unique_build_log ON build_logs (repo_source, repo_owner, repo_name, inserted_at);
ALTER TABLE build_logs ADD migrated_from BIGINT;

-- modify build_versions for migrations
ALTER TABLE build_versions ADD migrated_from BIGINT;

-- modify releases for migrations
CREATE UNIQUE INDEX unique_release ON releases (repo_source, repo_owner, repo_name, started_at);
ALTER TABLE releases ADD migrated_from BIGINT;

-- modify release_logs for migrations
CREATE UNIQUE INDEX unique_release_log ON release_logs (repo_source, repo_owner, repo_name, inserted_at);
ALTER TABLE release_logs ADD migrated_from BIGINT;
