WITH computed_pipelines
       AS (DELETE FROM computed_pipelines WHERE
    repo_source = @toSource AND repo_owner = @toOwner AND repo_name = @toName RETURNING id)
SELECT
  COUNT(computed_pipelines)
FROM
  computed_pipelines;
