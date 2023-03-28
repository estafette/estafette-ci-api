SELECT
  id,
  repo_source,
  repo_owner,
  repo_name,
  release,
  release_action,
  release_version,
  release_status,
  inserted_at,
  started_at,
  updated_at,
  EXTRACT(EPOCH FROM AGE(COALESCE(started_at, inserted_at), inserted_at)),
  EXTRACT(EPOCH FROM AGE(updated_at, COALESCE(started_at, inserted_at))),
  triggered_by_event,
  groups,
  organizations
FROM
  releases
WHERE
  migrated_from = @id
LIMIT 1;
