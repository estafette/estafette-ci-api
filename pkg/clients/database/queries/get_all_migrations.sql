SELECT
  id,
  status,
  last_step,
  from_source,
  from_owner,
  from_name,
  to_source,
  to_owner,
  to_name,
  error_details,
  queued_at,
  updated_at
FROM
  migration_task_queue
ORDER BY
  updated_at DESC;
