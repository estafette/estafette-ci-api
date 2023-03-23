SELECT DISTINCT ON (a.release)
  a.id,
  a.repo_source,
  a.repo_owner,
  a.repo_name,
  a.release,
  a.release_action,
  a.release_version,
  a.release_status,
  a.inserted_at,
  a.started_at,
  a.updated_at,
  EXTRACT(EPOCH FROM AGE(COALESCE(a.started_at, a.inserted_at), a.inserted_at)),
  EXTRACT(EPOCH FROM AGE(a.updated_at, COALESCE(a.started_at, a.inserted_at))),
  a.triggered_by_event,
  a.groups,
  a.organizations
FROM
  releases a
WHERE
  a.repo_source = @toSource AND
  a.repo_owner = @toOwner AND
  a.repo_name = @toName
ORDER BY release, updated_at DESC
LIMIT @maxReleases;

