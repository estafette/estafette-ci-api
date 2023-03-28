UPDATE
  computed_pipelines
SET
  migration_task_id = @id
WHERE
  repo_source = @fromSource AND
  repo_owner = @fromOwner AND
  repo_name = @fromName
