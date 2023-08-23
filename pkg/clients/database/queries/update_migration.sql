UPDATE migration_task_queue
SET
  updated_at     = NOW(),
  last_step      = CASE WHEN @lastStep <> '' THEN @lastStep ELSE migration_task_queue.last_step END,
  builds         = CASE WHEN @builds > 0 THEN @builds ELSE migration_task_queue.builds END,
  releases       = CASE WHEN @releases > 0 THEN @releases ELSE migration_task_queue.releases END,
  total_duration = CASE
                     WHEN @totalDuration > 0 THEN migration_task_queue.total_duration + @totalDuration
                     ELSE migration_task_queue.total_duration END,
  error_details  = CASE
                     WHEN @errorDetails::VARCHAR IS NOT NULL AND @errorDetails::VARCHAR <> ''
                       THEN CONCAT(migration_task_queue.error_details, @errorDetails::VARCHAR)
                     ELSE migration_task_queue.error_details END,
  status         = CASE WHEN @status <> '' THEN @status ELSE migration_task_queue.status END
WHERE
  id = @id;
