UPSERT
INTO
  build_versions
  (
    id,
    repo_source,
    repo_full_name,
    auto_increment,
    updated_at,
    migrated_from
  )
SELECT
  (2000000000000000000 + id) AS id,
  @toSourceName              AS repo_source,
  @toFullName                AS repo_full_name,
  auto_increment,
  updated_at,
  id                         AS migrated_from
FROM
  build_versions
WHERE
  repo_source = @fromSourceName AND
  repo_full_name = @fromFullName;
