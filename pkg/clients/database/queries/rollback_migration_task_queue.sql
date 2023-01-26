WITH migration_task_queue
       AS (DELETE FROM migration_task_queue WHERE from_source = @fromSource AND
                                                  from_owner = @fromOwner AND
                                                  from_name = @fromName RETURNING id)
SELECT
  COUNT(migration_task_queue)
FROM
  migration_task_queue;
