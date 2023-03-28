SELECT
  id,
  status,
  last_step,
  builds,
  releases,
  total_duration,
  from_source,
  from_owner,
  from_name,
  to_source,
  to_owner,
  to_name,
  callback_url,
  error_details,
  queued_at,
  updated_at
FROM
  migration_task_queue
WHERE
  from_source = @fromSource AND
  from_owner = @fromOwner AND
  from_name = @fromName
