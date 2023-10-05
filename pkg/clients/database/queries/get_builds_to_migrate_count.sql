SELECT
    COUNT(id)
FROM
    builds
WHERE
    repo_source = @fromSource AND
    repo_owner = @fromOwner AND
    repo_name = @fromName