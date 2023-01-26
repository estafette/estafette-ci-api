INSERT INTO
  build_versions
  (
    repo_source,
    repo_full_name,
    auto_increment,
    updated_at,
    migrated_from
  )
SELECT
  @toSourceName AS repo_source,
  @toFullName AS repo_full_name,
  bv.auto_increment,
  bv.updated_at,
  bv.id AS migrated_from
FROM
  build_versions AS bv
WHERE
  repo_source = @fromSourceName AND
  repo_full_name = @fromFullName
ON CONFLICT
  (
  repo_source,
  repo_full_name
  )
  DO UPDATE SET migrated_from = excluded.migrated_from
RETURNING
  migrated_from AS from_id,
  id AS to_id;
