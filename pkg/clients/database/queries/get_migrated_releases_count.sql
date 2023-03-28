SELECT
  COUNT(id)
FROM
  releases
WHERE
  repo_source = @toSource AND
  repo_owner = @toOwner AND
  repo_name = @toName
