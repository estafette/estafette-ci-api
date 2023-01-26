WITH computed_releases
       AS (DELETE FROM computed_releases WHERE
    repo_source = @toSource AND repo_owner = @toOwner AND repo_name = @toName RETURNING id)
SELECT
  COUNT(computed_releases)
FROM
  computed_releases;
