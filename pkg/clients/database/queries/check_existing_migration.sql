SELECT
  TRUE AS exists,
  COALESCE(mq.id, '')
FROM
  computed_pipelines AS cp
  LEFT JOIN migration_task_queue mq ON cp.migration_task_id = mq.id
WHERE
  cp.repo_source = @fromSource AND
  cp.repo_owner = @fromOwner AND
  cp.repo_name = @fromName
LIMIT 1;
