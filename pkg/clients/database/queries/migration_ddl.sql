-- modify builds for migrations
CREATE UNIQUE INDEX IF NOT EXISTS unique_build ON builds (repo_source, repo_owner, repo_name, started_at);
ALTER TABLE builds ADD IF NOT EXISTS migrated_from BIGINT;

-- modify build_logs for migrations
CREATE UNIQUE INDEX IF NOT EXISTS unique_build_log ON build_logs (repo_source, repo_owner, repo_name, inserted_at);
ALTER TABLE build_logs ADD IF NOT EXISTS migrated_from BIGINT;

-- modify build_versions for migrations
ALTER TABLE build_versions ADD IF NOT EXISTS migrated_from BIGINT;

-- modify releases for migrations
CREATE UNIQUE INDEX IF NOT EXISTS unique_release ON releases (repo_source, repo_owner, repo_name, started_at);
ALTER TABLE releases ADD IF NOT EXISTS migrated_from BIGINT;

-- modify release_logs for migrations
CREATE UNIQUE INDEX IF NOT EXISTS unique_release_log ON release_logs (repo_source, repo_owner, repo_name, inserted_at);
ALTER TABLE release_logs ADD IF NOT EXISTS migrated_from BIGINT;

CREATE TABLE IF NOT EXISTS migration_queue
(
  id            VARCHAR(256)                               NOT NULL CONSTRAINT migration_queue_pk PRIMARY KEY,
  status        VARCHAR(256)             DEFAULT 'queued'  NOT NULL CONSTRAINT migration_queue_status_check CHECK (status <> ''),
  last_step     VARCHAR(256)             DEFAULT 'waiting' NOT NULL CONSTRAINT migration_queue_last_step_check CHECK (last_step <> ''),
  from_source   VARCHAR(256)                               NOT NULL,
  from_owner    VARCHAR(256)                               NOT NULL,
  from_name     VARCHAR(256)                               NOT NULL,
  to_source     VARCHAR(256)                               NOT NULL,
  to_owner      VARCHAR(256)                               NOT NULL,
  to_name       VARCHAR(256)                               NOT NULL,
  callback_url  VARCHAR(256),
  error_details VARCHAR(256),
  queued_at     TIMESTAMP WITH TIME ZONE DEFAULT NOW()     NOT NULL,
  updated_at    TIMESTAMP WITH TIME ZONE DEFAULT NOW()     NOT NULL
);

CREATE UNIQUE INDEX IF NOT EXISTS unique_migration_queue ON migration_queue (from_source, from_owner, from_name, to_source, to_owner, to_name);

ALTER TABLE migration_queue OWNER TO root;
