INSERT INTO
  builds
(
  repo_source,
  repo_owner,
  repo_name,
  repo_branch,
  repo_revision,
  build_version,
  build_status,
  labels,
  manifest,
  inserted_at,
  updated_at,
  commits,
  releases,
  triggers,
  release_targets,
  triggered_by_event,
  cpu_request,
  cpu_limit,
  cpu_max_usage,
  memory_request,
  memory_limit,
  memory_max_usage,
  started_at,
  organizations,
  groups,
  migrated_from
)
SELECT
  @toSource AS repo_source,
  @toOwner AS repo_owner,
  @toName AS repo_name,
  repo_branch,
  repo_revision,
  build_version,
  build_status,
  labels,
  manifest,
  inserted_at,
  updated_at,
  commits,
  releases,
  triggers,
  release_targets,
  triggered_by_event,
  cpu_request,
  cpu_limit,
  cpu_max_usage,
  memory_request,
  memory_limit,
  memory_max_usage,
  started_at,
  organizations,
  groups,
  id AS migrated_from
FROM
  builds
WHERE
  repo_source = @fromSource
  AND
  repo_owner = @fromOwner
  AND
  repo_name = @fromName
ON
  CONFLICT
  (
    repo_source,
    repo_owner,
    repo_name,
    started_at
  )
DO
  UPDATE SET migrated_from = excluded.migrated_from
RETURNING
  migrated_from as from_id,
  id as to_id;
