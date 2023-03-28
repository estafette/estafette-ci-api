UPSERT
INTO
  release_logs
  (
    id,
    repo_source,
    repo_owner,
    repo_name,
    release_id,
    inserted_at,
    migrated_from
  )
SELECT
  (2000000000000000000 + release_logs.id) AS id,
  releases.repo_source,
  releases.repo_owner,
  releases.repo_name,
  releases.id                             AS release_id,
  release_logs.inserted_at,
  release_logs.id                         AS migrated_from
FROM
  release_logs
  JOIN releases ON release_logs.release_id = releases.migrated_from
WHERE
  release_logs.repo_source = @fromSource AND
  release_logs.repo_owner = @fromOwner AND
  release_logs.repo_name = @fromName;
