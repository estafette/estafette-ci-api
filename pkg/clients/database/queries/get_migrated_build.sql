SELECT
  id,
  repo_source,
  repo_owner,
  repo_name,
  repo_branch,
  repo_revision,
  build_version,
  build_status,
  labels,
  release_targets,
  manifest,
  commits,
  triggers,
  inserted_at,
  started_at,
  updated_at,
  EXTRACT(EPOCH FROM AGE(COALESCE(started_at, inserted_at), inserted_at)),
  EXTRACT(EPOCH FROM AGE(updated_at, COALESCE(started_at, inserted_at))),
  triggered_by_event,
  groups,
  organizations
FROM
  builds
WHERE
  migrated_from = @id
LIMIT 1;
