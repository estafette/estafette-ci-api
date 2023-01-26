INSERT INTO
  build_logs
  (
    repo_source,
    repo_owner,
    repo_name,
    repo_branch,
    repo_revision,
    steps,
    inserted_at,
    build_id,
    migrated_from
  )
SELECT
  b.repo_source,
  b.repo_owner,
  b.repo_name,
  bl.repo_branch,
  bl.repo_revision,
  bl.steps,
  bl.inserted_at,
  b.id  AS build_id,
  bl.id AS migrated_from
FROM
  build_logs AS bl
  JOIN builds AS b ON bl.build_id = b.migrated_from
WHERE
  repo_source = @toSource AND
  repo_owner = @toOwner AND
  repo_name = @toName
ON CONFLICT
  (
  repo_source,
  repo_owner,
  repo_name,
  inserted_at
  )
  DO UPDATE SET migrated_from = excluded.migrated_from
RETURNING
  migrated_from AS from_id,
  id AS to_id;
