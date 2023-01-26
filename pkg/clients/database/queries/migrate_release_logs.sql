INSERT INTO
  release_logs
  (
    repo_source,
    repo_owner,
    repo_name,
    release_id,
    steps,
    inserted_at,
    migrated_from
  )
SELECT
  r.repo_source,
  r.repo_owner,
  r.repo_name,
  r.id  AS release_id,
  rl.steps,
  rl.inserted_at,
  rl.id AS migrated_from
FROM
  release_logs AS rl
  JOIN releases AS r ON rl.release_id = r.migrated_from
WHERE
    r.repo_source = 'X' AND
    r.repo_owner = 'Y' AND
    r.repo_name = 'Z'
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
