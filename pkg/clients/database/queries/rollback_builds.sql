WITH builds
       AS (DELETE FROM builds WHERE repo_source = @toSource AND repo_owner = @toOwner AND repo_name = @toName RETURNING id)
SELECT
  COUNT(builds)
FROM
  builds;
