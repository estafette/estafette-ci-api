SELECT
  migrated_from AS from_id,
  id            AS to_id
FROM
  build_logs
WHERE
  repo_source = @toSource AND
  repo_owner = @toOwner AND
  repo_name = @toName
