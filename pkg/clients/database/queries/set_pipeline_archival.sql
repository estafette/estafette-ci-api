UPDATE
  computed_pipelines
SET
  archived = @archived
WHERE
  repo_source = @source AND
  repo_owner = @owner AND
  repo_name = @name;
