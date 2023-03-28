WITH releases
       AS (DELETE FROM releases WHERE repo_source = @toSource AND repo_owner = @toOwner AND repo_name = @toName RETURNING id)
SELECT
  COUNT(releases)
FROM
  releases;
