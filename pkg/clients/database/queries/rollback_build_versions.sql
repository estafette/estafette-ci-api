WITH build_versions
       AS (DELETE FROM build_versions WHERE repo_source = @toSource AND repo_full_name = @toFullName RETURNING id)
SELECT
  COUNT(build_versions)
FROM
  build_versions;
