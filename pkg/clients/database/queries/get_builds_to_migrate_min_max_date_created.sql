SELECT
  COALESCE(MIN(inserted_at), '1970-01-01'::timestamptz) AS oldest_created_date,
  COALESCE(Max(inserted_at), '1970-01-01'::timestamptz)  AS max_created_date
FROM
  builds
WHERE
  repo_source= @fromSource
  AND repo_owner = @fromOwner
  AND repo_name = @fromName