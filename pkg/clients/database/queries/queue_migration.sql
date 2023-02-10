INSERT INTO
  migration_task_queue
  (
    id,
    from_source,
    from_owner,
    from_name,
    to_source,
    to_owner,
    to_name,
    callback_url
  )
VALUES
  (
    @id,
    @fromSource,
    @fromOwner,
    @fromName,
    @toSource,
    @toOwner,
    @toName,
    @callbackURL
  )
ON CONFLICT
  (
  from_source, from_owner, from_name, to_source, to_owner, to_name
  )
  DO UPDATE SET updated_at    = NOW(),
                error_details = NULL,
                status        = CASE WHEN @status <> '' THEN @status ELSE migration_task_queue.status END,
                last_step     = CASE WHEN @lastStep <> '' THEN @lastStep ELSE migration_task_queue.last_step END
RETURNING
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
  updated_at;
