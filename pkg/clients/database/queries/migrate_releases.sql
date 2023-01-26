INSERT INTO
  releases
  (
    repo_source,
    repo_owner,
    repo_name,
    release,
    release_version,
    release_status,
    inserted_at,
    updated_at,
    release_action,
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
  @toOwner  AS repo_owner,
  @toName   AS repo_name,
  release,
  release_version,
  release_status,
  inserted_at,
  updated_at,
  release_action,
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
  id        AS migrated_from
FROM
  releases
WHERE
  repo_source = @fromSource AND
  repo_owner = @fromOwner AND
  repo_name = @fromName
ON CONFLICT
  (
  repo_source,
  repo_owner,
  repo_name,
  started_at
  )
  DO UPDATE SET migrated_from = excluded.migrated_from
RETURNING
  migrated_from AS from_id,
  id AS to_id;
