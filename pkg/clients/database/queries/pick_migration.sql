UPDATE migration_task_queue
SET
  status = 'in_progress'
WHERE
    id IN (SELECT
             mq.id AS id
           FROM
             migration_task_queue AS mq
           WHERE
             mq.status = 'queued' AND
             CONCAT(mq.from_source, mq.from_owner, mq.from_name) NOT IN
             (SELECT DISTINCT
                CONCAT(b.repo_source, b.repo_owner, b.repo_name)
              FROM
                builds b
              WHERE b.build_status = 'running') AND
             CONCAT(mq.from_source, mq.from_owner, mq.from_name) NOT IN
             (SELECT DISTINCT
                CONCAT(r.repo_source, r.repo_owner, r.repo_name)
              FROM
                releases r
              WHERE r.release_status = 'running')
           ORDER BY mq.queued_at ASC
           LIMIT @maxTasks)
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
