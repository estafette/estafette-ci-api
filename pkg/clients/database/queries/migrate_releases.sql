UPSERT
INTO
  releases
  (
    id,
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
  (2000000000000000000 + id) AS id,
  @toSource                  AS repo_source,
  @toOwner                   AS repo_owner,
  @toName                    AS repo_name,
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
  id                         AS migrated_from
FROM
  releases
WHERE
  repo_source = @fromSource AND
  repo_owner = @fromOwner AND
  repo_name = @fromName;
