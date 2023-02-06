UPDATE migration_queue
SET updated_at    = NOW(),
    last_step     = @lastStep,
    error_details = CONCAT(error_details, '\n', @errorDetails),
    status        = CASE WHEN @status <> '' THEN @status ELSE migration_queue.status END
WHERE
  id = @id;
