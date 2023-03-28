UPSERT
INTO
  build_logs
  (
    id,
    repo_source,
    repo_owner,
    repo_name,
    repo_branch,
    repo_revision,
    inserted_at,
    build_id,
    migrated_from
  )
SELECT
  (2000000000000000000 + build_logs.id) AS id,
  builds.repo_source,
  builds.repo_owner,
  builds.repo_name,
  build_logs.repo_branch,
  build_logs.repo_revision,
  build_logs.inserted_at,
  builds.id                             AS build_id,
  build_logs.id                         AS migrated_from
FROM
  build_logs
  JOIN builds ON build_logs.build_id = builds.migrated_from
WHERE
  build_logs.repo_source = @fromSource AND
  build_logs.repo_owner = @fromOwner AND
  build_logs.repo_name = @fromName;
