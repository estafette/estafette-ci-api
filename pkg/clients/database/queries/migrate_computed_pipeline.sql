UPSERT
INTO
  computed_pipelines
  (
    id,
    pipeline_id,
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
    release_targets,
    first_inserted_at,
    triggers,
    triggered_by_event,
    last_updated_at,
    archived,
    recent_committers,
    recent_releasers,
    started_at,
    organizations,
    groups,
    extra_info,
    notifications,
    bots
  )
SELECT
  (2000000000000000000 + id) AS id,
  pipeline_id,
  @toSource                  AS repo_source,
  @toOwner                   AS repo_owner,
  @toName                    AS repo_name,
  repo_branch,
  repo_revision,
  build_version,
  build_status,
  labels,
  manifest,
  NOW(),
  updated_at,
  commits,
  release_targets,
  first_inserted_at,
  triggers,
  triggered_by_event,
  NOW(),
  true                       AS archived,
  recent_committers,
  recent_releasers,
  started_at,
  organizations,
  groups,
  extra_info,
  notifications,
  bots
FROM
  computed_pipelines
WHERE
  repo_source = @fromSource AND
  repo_owner = @fromOwner AND
  repo_name = @fromName;
