SELECT
  MIN(inserted_at) AS oldest_created_date,
  MAX(inserted_at) AS max_created_date
FROM
  builds
WHERE
  repo_source= @fromSource
  AND repo_owner = @fromOwner
  AND repo_name = @fromName