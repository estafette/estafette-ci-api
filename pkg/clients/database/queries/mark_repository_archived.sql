UPDATE computed_pipelines
SET archived = TRUE
WHERE
    CONCAT(repo_source, '/', repo_owner, '/', repo_name) IN (@pickedRepos);
