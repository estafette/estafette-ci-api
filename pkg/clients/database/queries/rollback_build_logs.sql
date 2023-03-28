WITH build_logs
       AS (DELETE FROM build_logs WHERE repo_source = @toSource AND repo_owner = @toOwner AND repo_name = @toName RETURNING id)
SELECT
  COUNT(build_logs)
FROM
  build_logs;
