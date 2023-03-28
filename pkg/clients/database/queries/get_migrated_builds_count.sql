SELECT
  COUNT(id)
FROM
  builds
WHERE
  repo_source = @toSource AND
  repo_owner = @toOwner AND
  repo_name = @toName
