UPDATE migration_queue
SET status = 'in_progress'
WHERE
    id IN (SELECT id FROM migration_queue WHERE status = 'queued' ORDER BY queued_at ASC LIMIT 1)
RETURNING
  id,
  status,
  last_step,
  from_source,
  from_owner,
  from_name,
  to_source,
  to_owner,
  to_name,
  callback_url,
  error_details,
  queued_at,
  updated_at;
