WITH release_logs
       AS (DELETE FROM release_logs WHERE repo_source = @toSource AND repo_owner = @toOwner AND repo_name = @toName RETURNING id)
SELECT
  COUNT(release_logs)
FROM
  release_logs;
