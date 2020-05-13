package cockroachdb

import (
	"context"
	"fmt"
	"regexp"
	"strconv"
	"testing"
	"time"

	sq "github.com/Masterminds/squirrel"
	"github.com/estafette/estafette-ci-api/config"
	contracts "github.com/estafette/estafette-ci-contracts"
	manifest "github.com/estafette/estafette-ci-manifest"
	"github.com/stretchr/testify/assert"
)

var (
	cdbClient = client{}
)

func TestIntegrationGetAutoIncrement(t *testing.T) {
	t.Run("ReturnsAnIncrementingCountForUniqueRepo", func(t *testing.T) {

		if testing.Short() {
			t.Skip("skipping test in short mode.")
		}

		ctx := context.Background()
		cockroachdbClient := getCockroachdbClient(ctx, t)

		// act
		autoincrement, err := cockroachdbClient.GetAutoIncrement(ctx, "github", "estafette", "estafette-ci-api")

		assert.Nil(t, err)
		assert.True(t, autoincrement > 0)
	})

	t.Run("ReturnsLargerCountForSubsequentRequests", func(t *testing.T) {

		if testing.Short() {
			t.Skip("skipping test in short mode.")
		}

		ctx := context.Background()
		cockroachdbClient := getCockroachdbClient(ctx, t)

		// act
		autoincrement1, err := cockroachdbClient.GetAutoIncrement(ctx, "github", "estafette", "estafette-ci-api")
		autoincrement2, err := cockroachdbClient.GetAutoIncrement(ctx, "github", "estafette", "estafette-ci-api")

		assert.Nil(t, err)
		assert.True(t, autoincrement1 > 0)
		assert.True(t, autoincrement2 > 0)
		assert.True(t, autoincrement2 > autoincrement1)
	})
}

func TestIntegrationInsertBuild(t *testing.T) {
	t.Run("ReturnsInsertedBuildWithID", func(t *testing.T) {

		if testing.Short() {
			t.Skip("skipping test in short mode.")
		}

		ctx := context.Background()
		cockroachdbClient := getCockroachdbClient(ctx, t)
		build := getBuild()
		jobResources := getJobResources()

		// act
		insertedBuild, err := cockroachdbClient.InsertBuild(ctx, build, jobResources)

		assert.Nil(t, err)
		assert.NotNil(t, insertedBuild)
		assert.True(t, insertedBuild.ID != "")
	})
}

func TestIntegrationUpdateBuildStatus(t *testing.T) {
	t.Run("UpdatesStatusForInsertedBuild", func(t *testing.T) {

		if testing.Short() {
			t.Skip("skipping test in short mode.")
		}

		ctx := context.Background()
		cockroachdbClient := getCockroachdbClient(ctx, t)
		build := getBuild()
		jobResources := getJobResources()
		insertedBuild, err := cockroachdbClient.InsertBuild(ctx, build, jobResources)
		assert.Nil(t, err)
		buildID, err := strconv.Atoi(insertedBuild.ID)
		assert.Nil(t, err)

		// act
		err = cockroachdbClient.UpdateBuildStatus(ctx, insertedBuild.RepoSource, insertedBuild.RepoOwner, insertedBuild.RepoName, buildID, "succeeded")

		assert.Nil(t, err)
	})

	t.Run("UpdatesStatusForNonExistingBuild", func(t *testing.T) {

		if testing.Short() {
			t.Skip("skipping test in short mode.")
		}

		ctx := context.Background()
		cockroachdbClient := getCockroachdbClient(ctx, t)
		build := getBuild()
		buildID := 15

		// act
		err := cockroachdbClient.UpdateBuildStatus(ctx, build.RepoSource, build.RepoOwner, build.RepoName, buildID, "succeeded")

		assert.Nil(t, err)
	})
}

func TestIntegrationUpdateBuildResourceUtilization(t *testing.T) {
	t.Run("UpdatesJobResourceForInsertedBuild", func(t *testing.T) {

		if testing.Short() {
			t.Skip("skipping test in short mode.")
		}

		ctx := context.Background()
		cockroachdbClient := getCockroachdbClient(ctx, t)
		build := getBuild()
		jobResources := getJobResources()
		insertedBuild, err := cockroachdbClient.InsertBuild(ctx, build, jobResources)
		assert.Nil(t, err)
		buildID, err := strconv.Atoi(insertedBuild.ID)
		assert.Nil(t, err)

		newJobResources := JobResources{
			CPURequest:    float64(0.3),
			CPULimit:      float64(4.0),
			MemoryRequest: float64(67108864),
			MemoryLimit:   float64(21474836480),
		}

		// act
		err = cockroachdbClient.UpdateBuildResourceUtilization(ctx, insertedBuild.RepoSource, insertedBuild.RepoOwner, insertedBuild.RepoName, buildID, newJobResources)

		assert.Nil(t, err)
	})

	t.Run("UpdatesJobResourcesForNonExistingBuild", func(t *testing.T) {

		if testing.Short() {
			t.Skip("skipping test in short mode.")
		}

		ctx := context.Background()
		cockroachdbClient := getCockroachdbClient(ctx, t)
		build := getBuild()
		buildID := 15

		newJobResources := JobResources{
			CPURequest:    float64(0.3),
			CPULimit:      float64(4.0),
			MemoryRequest: float64(67108864),
			MemoryLimit:   float64(21474836480),
		}

		// act
		err := cockroachdbClient.UpdateBuildResourceUtilization(ctx, build.RepoSource, build.RepoOwner, build.RepoName, buildID, newJobResources)

		assert.Nil(t, err)
	})
}

func TestIntegrationInsertRelease(t *testing.T) {
	t.Run("ReturnsInsertedReleaseWithID", func(t *testing.T) {

		if testing.Short() {
			t.Skip("skipping test in short mode.")
		}

		ctx := context.Background()
		cockroachdbClient := getCockroachdbClient(ctx, t)
		release := getRelease()
		jobResources := getJobResources()

		// act
		insertedRelease, err := cockroachdbClient.InsertRelease(ctx, release, jobResources)

		assert.Nil(t, err)
		assert.NotNil(t, insertedRelease)
		assert.True(t, insertedRelease.ID != "")
	})
}

func TestIntegrationUpdateReleaseStatus(t *testing.T) {
	t.Run("UpdatesStatusForInsertedRelease", func(t *testing.T) {

		if testing.Short() {
			t.Skip("skipping test in short mode.")
		}

		ctx := context.Background()
		cockroachdbClient := getCockroachdbClient(ctx, t)
		release := getRelease()
		jobResources := getJobResources()
		insertedRelease, err := cockroachdbClient.InsertRelease(ctx, release, jobResources)
		assert.Nil(t, err)
		releaseID, err := strconv.Atoi(insertedRelease.ID)
		assert.Nil(t, err)

		// act
		err = cockroachdbClient.UpdateReleaseStatus(ctx, insertedRelease.RepoSource, insertedRelease.RepoOwner, insertedRelease.RepoName, releaseID, "succeeded")

		assert.Nil(t, err)
	})

	t.Run("UpdatesStatusForNonExistingRelease", func(t *testing.T) {

		if testing.Short() {
			t.Skip("skipping test in short mode.")
		}

		ctx := context.Background()
		cockroachdbClient := getCockroachdbClient(ctx, t)
		release := getRelease()
		releaseID := 15

		// act
		err := cockroachdbClient.UpdateReleaseStatus(ctx, release.RepoSource, release.RepoOwner, release.RepoName, releaseID, "succeeded")

		assert.Nil(t, err)
	})
}

func TestIntegrationUpdateReleaseResourceUtilization(t *testing.T) {
	t.Run("UpdatesJobResourceForInsertedBuild", func(t *testing.T) {

		if testing.Short() {
			t.Skip("skipping test in short mode.")
		}

		ctx := context.Background()
		cockroachdbClient := getCockroachdbClient(ctx, t)
		release := getRelease()
		jobResources := getJobResources()
		insertedRelease, err := cockroachdbClient.InsertRelease(ctx, release, jobResources)
		assert.Nil(t, err)
		releaseID, err := strconv.Atoi(insertedRelease.ID)
		assert.Nil(t, err)

		newJobResources := JobResources{
			CPURequest:    float64(0.3),
			CPULimit:      float64(4.0),
			MemoryRequest: float64(67108864),
			MemoryLimit:   float64(21474836480),
		}

		// act
		err = cockroachdbClient.UpdateReleaseResourceUtilization(ctx, release.RepoSource, release.RepoOwner, release.RepoName, releaseID, newJobResources)

		assert.Nil(t, err)
	})

	t.Run("UpdatesJobResourcesForNonExistingBuild", func(t *testing.T) {

		if testing.Short() {
			t.Skip("skipping test in short mode.")
		}

		ctx := context.Background()
		cockroachdbClient := getCockroachdbClient(ctx, t)
		release := getRelease()
		releaseID := 15

		newJobResources := JobResources{
			CPURequest:    float64(0.3),
			CPULimit:      float64(4.0),
			MemoryRequest: float64(67108864),
			MemoryLimit:   float64(21474836480),
		}

		// act
		err := cockroachdbClient.UpdateReleaseResourceUtilization(ctx, release.RepoSource, release.RepoOwner, release.RepoName, releaseID, newJobResources)

		assert.Nil(t, err)
	})
}

func TestIntegrationInsertBuildLog(t *testing.T) {
	t.Run("ReturnsInsertedBuildLogWithIDWhenWriteLogToDatabaseIsTrue", func(t *testing.T) {

		if testing.Short() {
			t.Skip("skipping test in short mode.")
		}

		ctx := context.Background()
		cockroachdbClient := getCockroachdbClient(ctx, t)
		buildLog := getBuildLog()

		// act
		insertedBuildLog, err := cockroachdbClient.InsertBuildLog(ctx, buildLog, true)

		assert.Nil(t, err)
		assert.NotNil(t, insertedBuildLog)
		assert.True(t, insertedBuildLog.ID != "")
	})

	t.Run("ReturnsInsertedBuildLogWithIDWhenWriteLogToDatabaseIsFalse", func(t *testing.T) {

		if testing.Short() {
			t.Skip("skipping test in short mode.")
		}

		ctx := context.Background()
		cockroachdbClient := getCockroachdbClient(ctx, t)
		buildLog := getBuildLog()

		// act
		insertedBuildLog, err := cockroachdbClient.InsertBuildLog(ctx, buildLog, false)

		assert.Nil(t, err)
		assert.NotNil(t, insertedBuildLog)
		assert.True(t, insertedBuildLog.ID != "")
	})
}

func TestIntegrationInsertReleaseLog(t *testing.T) {
	t.Run("ReturnsInsertedReleaseLogWithIDWhenWriteLogToDatabaseIsTrue", func(t *testing.T) {

		if testing.Short() {
			t.Skip("skipping test in short mode.")
		}

		ctx := context.Background()
		cockroachdbClient := getCockroachdbClient(ctx, t)
		releaseLog := getReleaseLog()

		// act
		insertedReleaseLog, err := cockroachdbClient.InsertReleaseLog(ctx, releaseLog, true)

		assert.Nil(t, err)
		assert.NotNil(t, insertedReleaseLog)
		assert.True(t, insertedReleaseLog.ID != "")
	})

	t.Run("ReturnsInsertedReleaseLogWithIDWhenWriteLogToDatabaseIsFalse", func(t *testing.T) {

		if testing.Short() {
			t.Skip("skipping test in short mode.")
		}

		ctx := context.Background()
		cockroachdbClient := getCockroachdbClient(ctx, t)
		releaseLog := getReleaseLog()

		// act
		insertedReleaseLog, err := cockroachdbClient.InsertReleaseLog(ctx, releaseLog, false)

		assert.Nil(t, err)
		assert.NotNil(t, insertedReleaseLog)
		assert.True(t, insertedReleaseLog.ID != "")
	})
}

func TestQueryBuilder(t *testing.T) {
	t.Run("GeneratesQueryWithoutFilters", func(t *testing.T) {

		query := cdbClient.selectBuildsQuery()

		query, _ = whereClauseGeneratorForAllFilters(query, "a", "inserted_at", map[string][]string{})

		// act
		sql, _, err := query.ToSql()

		assert.Nil(t, err)
		assert.Equal(t, "SELECT a.id, a.repo_source, a.repo_owner, a.repo_name, a.repo_branch, a.repo_revision, a.build_version, a.build_status, a.labels, a.release_targets, a.manifest, a.commits, a.triggers, a.inserted_at, a.updated_at, a.duration::INT, a.triggered_by_event FROM builds a", sql)
	})

	t.Run("GeneratesQueryWithStatusFilter", func(t *testing.T) {

		query := cdbClient.selectBuildsQuery()

		query, _ = whereClauseGeneratorForAllFilters(query, "a", "inserted_at", map[string][]string{
			"status": []string{
				"succeeded",
			},
		})

		// act
		sql, _, err := query.ToSql()

		assert.Nil(t, err)
		assert.Equal(t, "SELECT a.id, a.repo_source, a.repo_owner, a.repo_name, a.repo_branch, a.repo_revision, a.build_version, a.build_status, a.labels, a.release_targets, a.manifest, a.commits, a.triggers, a.inserted_at, a.updated_at, a.duration::INT, a.triggered_by_event FROM builds a WHERE a.build_status IN ($1)", sql)
	})

	t.Run("GeneratesQueryWithSinceFilter", func(t *testing.T) {

		query := cdbClient.selectBuildsQuery()

		query, _ = whereClauseGeneratorForAllFilters(query, "a", "inserted_at", map[string][]string{
			"since": []string{
				"1d",
			},
		})

		// act
		sql, _, err := query.ToSql()

		assert.Nil(t, err)
		assert.Equal(t, "SELECT a.id, a.repo_source, a.repo_owner, a.repo_name, a.repo_branch, a.repo_revision, a.build_version, a.build_status, a.labels, a.release_targets, a.manifest, a.commits, a.triggers, a.inserted_at, a.updated_at, a.duration::INT, a.triggered_by_event FROM builds a WHERE a.inserted_at >= $1", sql)
	})

	t.Run("GeneratesQueryWithLabelsFilter", func(t *testing.T) {

		query := cdbClient.selectBuildsQuery()

		query, _ = whereClauseGeneratorForAllFilters(query, "a", "inserted_at", map[string][]string{
			"labels": []string{
				"key=value",
			},
		})

		// act
		sql, _, err := query.ToSql()

		assert.Nil(t, err)
		assert.Equal(t, "SELECT a.id, a.repo_source, a.repo_owner, a.repo_name, a.repo_branch, a.repo_revision, a.build_version, a.build_status, a.labels, a.release_targets, a.manifest, a.commits, a.triggers, a.inserted_at, a.updated_at, a.duration::INT, a.triggered_by_event FROM builds a WHERE a.labels @> $1", sql)
	})

	t.Run("GeneratesQueryWithLabelsFilterAndOrderBy", func(t *testing.T) {

		pageSize := 15
		pageNumber := 2
		query := cdbClient.selectBuildsQuery().
			OrderBy("a.inserted_at DESC").
			Limit(uint64(pageSize)).
			Offset(uint64((pageNumber - 1) * pageSize))

		query, _ = whereClauseGeneratorForAllFilters(query, "a", "inserted_at", map[string][]string{
			"labels": []string{
				"key=value",
			},
		})

		// act
		sql, _, err := query.ToSql()

		assert.Nil(t, err)
		assert.Equal(t, "SELECT a.id, a.repo_source, a.repo_owner, a.repo_name, a.repo_branch, a.repo_revision, a.build_version, a.build_status, a.labels, a.release_targets, a.manifest, a.commits, a.triggers, a.inserted_at, a.updated_at, a.duration::INT, a.triggered_by_event FROM builds a WHERE a.labels @> $1 ORDER BY a.inserted_at DESC LIMIT 15 OFFSET 15", sql)
	})

	t.Run("GeneratesGetPipelinesQuery", func(t *testing.T) {

		query := cdbClient.selectPipelinesQuery().
			OrderBy("a.repo_source,a.repo_owner,a.repo_name").
			Limit(uint64(2)).
			Offset(uint64((2 - 1) * 20))

		// act
		sql, _, err := query.ToSql()

		assert.Nil(t, err)
		assert.Equal(t, "SELECT a.pipeline_id, a.repo_source, a.repo_owner, a.repo_name, a.repo_branch, a.repo_revision, a.build_version, a.build_status, a.labels, a.release_targets, a.manifest, a.commits, a.triggers, a.inserted_at, a.updated_at, a.duration::INT, a.last_updated_at, a.triggered_by_event FROM computed_pipelines a ORDER BY a.repo_source,a.repo_owner,a.repo_name LIMIT 2 OFFSET 20", sql)
	})

	t.Run("GeneratesFrequentLabelsQuery", func(t *testing.T) {

		arrayElementsQuery :=
			sq.StatementBuilder.
				Select("a.id, jsonb_array_elements(a.labels) AS l").
				From("computed_pipelines a").
				Where("jsonb_typeof(labels) = 'array'")

		arrayElementsQuery = arrayElementsQuery.Where(sq.GtOrEq{fmt.Sprintf("%v.inserted_at", "a"): time.Now().Add(time.Duration(-1) * time.Hour)})
		arrayElementsQuery = arrayElementsQuery.Where(sq.Eq{fmt.Sprintf("%v.build_status", "a"): []string{"succeeded"}})
		arrayElementsQuery = arrayElementsQuery.Where(fmt.Sprintf("%v.labels @> ?", "a"), "{\"group\":\"group-a\"}")

		selectCountQuery :=
			sq.StatementBuilder.
				Select("l->>'key' AS key, l->>'value' AS value, id").
				FromSelect(arrayElementsQuery, "b")

		groupByQuery :=
			sq.StatementBuilder.
				Select("key, value, count(DISTINCT id) AS pipelinesCount").
				FromSelect(selectCountQuery, "c").
				GroupBy("key, value")

		query :=
			sq.StatementBuilder.PlaceholderFormat(sq.Dollar).
				Select("key, value, pipelinesCount").
				FromSelect(groupByQuery, "d").
				Where(sq.Gt{"pipelinesCount": 1}).
				OrderBy("pipelinesCount DESC, key, value").
				Limit(uint64(7))

			// act
		sql, _, err := query.ToSql()

		assert.Nil(t, err)
		assert.Equal(t, "SELECT key, value, pipelinesCount FROM (SELECT key, value, count(DISTINCT id) AS pipelinesCount FROM (SELECT l->>'key' AS key, l->>'value' AS value, id FROM (SELECT a.id, jsonb_array_elements(a.labels) AS l FROM computed_pipelines a WHERE jsonb_typeof(labels) = 'array' AND a.inserted_at >= $1 AND a.build_status IN ($2) AND a.labels @> $3) AS b) AS c GROUP BY key, value) AS d WHERE pipelinesCount > $4 ORDER BY pipelinesCount DESC, key, value LIMIT 7", sql)
	})

	t.Run("GeneratesUpdateBuildStatusQuery", func(t *testing.T) {

		buildStatus := "canceling"
		buildID := 5
		repoSource := "github.com"
		repoOwner := "estafette"
		repoName := "estafette-ci-api"
		allowedBuildStatusesToTransitionFrom := []string{"running"}

		psql := sq.StatementBuilder.PlaceholderFormat(sq.Dollar)

		query := psql.
			Update("builds").
			Set("build_status", buildStatus).
			Set("updated_at", sq.Expr("now()")).
			Set("duration", sq.Expr("age(now(), inserted_at)")).
			Where(sq.Eq{"id": buildID}).
			Where(sq.Eq{"repo_source": repoSource}).
			Where(sq.Eq{"repo_owner": repoOwner}).
			Where(sq.Eq{"repo_name": repoName}).
			Where(sq.Eq{"build_status": allowedBuildStatusesToTransitionFrom}).
			Suffix("RETURNING id, repo_source, repo_owner, repo_name, repo_branch, repo_revision, build_version, build_status, labels, release_targets, manifest, commits, triggers, inserted_at, updated_at, duration::INT, triggered_by_event")

		// act
		sql, _, err := query.ToSql()

		assert.Nil(t, err)
		assert.Equal(t, "UPDATE builds SET build_status = $1, updated_at = now(), duration = age(now(), inserted_at) WHERE id = $2 AND repo_source = $3 AND repo_owner = $4 AND repo_name = $5 AND build_status IN ($6) RETURNING id, repo_source, repo_owner, repo_name, repo_branch, repo_revision, build_version, build_status, labels, release_targets, manifest, commits, triggers, inserted_at, updated_at, duration::INT, triggered_by_event", sql)
	})

	t.Run("GeneratesUpdateReleaseStatusQuery", func(t *testing.T) {

		releaseStatus := "canceling"
		releaseID := 5
		repoSource := "github.com"
		repoOwner := "estafette"
		repoName := "estafette-ci-api"
		allowedReleaseStatusesToTransitionFrom := []string{"running"}

		psql := sq.StatementBuilder.PlaceholderFormat(sq.Dollar)

		query := psql.
			Update("releases").
			Set("release_status", releaseStatus).
			Set("updated_at", sq.Expr("now()")).
			Set("duration", sq.Expr("age(now(), inserted_at)")).
			Where(sq.Eq{"id": releaseID}).
			Where(sq.Eq{"repo_source": repoSource}).
			Where(sq.Eq{"repo_owner": repoOwner}).
			Where(sq.Eq{"repo_name": repoName}).
			Where(sq.Eq{"release_status": allowedReleaseStatusesToTransitionFrom}).
			Suffix("RETURNING id, repo_source, repo_owner, repo_name, release, release_action, release_version, release_status, inserted_at, updated_at, duration::INT, triggered_by_event")

		// act
		sql, _, err := query.ToSql()

		assert.Nil(t, err)
		assert.Equal(t, "UPDATE releases SET release_status = $1, updated_at = now(), duration = age(now(), inserted_at) WHERE id = $2 AND repo_source = $3 AND repo_owner = $4 AND repo_name = $5 AND release_status IN ($6) RETURNING id, repo_source, repo_owner, repo_name, release, release_action, release_version, release_status, inserted_at, updated_at, duration::INT, triggered_by_event", sql)
	})

	t.Run("GeneratesRecentBuildQuery", func(t *testing.T) {

		psql := sq.StatementBuilder.PlaceholderFormat(sq.Dollar)

		innerquery := psql.
			Select("a.*, ROW_NUMBER() OVER (PARTITION BY a.repo_branch ORDER BY a.inserted_at DESC) AS rn").
			From("builds a").
			Where(sq.Eq{"a.repo_source": "github.com"}).
			Where(sq.Eq{"a.repo_owner": "estafette"}).
			Where(sq.Eq{"a.repo_name": "estafette-ci-web"}).
			OrderBy("a.inserted_at DESC").
			Limit(uint64(50))

		query := psql.
			Select("a.id, a.repo_source, a.repo_owner, a.repo_name, a.repo_branch, a.repo_revision, a.build_version, a.build_status, a.labels, a.release_targets, a.manifest, a.commits, a.triggers, a.inserted_at, a.updated_at, a.duration::INT, a.triggered_by_event").
			Prefix("WITH ranked_builds AS (?)", innerquery).
			From("ranked_builds a").
			Where("a.rn = 1").
			OrderBy("a.inserted_at DESC").
			Limit(uint64(5))

		// act
		sql, _, err := query.ToSql()

		assert.Nil(t, err)
		assert.Equal(t, "WITH ranked_builds AS (SELECT a.*, ROW_NUMBER() OVER (PARTITION BY a.repo_branch ORDER BY a.inserted_at DESC) AS rn FROM builds a WHERE a.repo_source = $1 AND a.repo_owner = $2 AND a.repo_name = $3 ORDER BY a.inserted_at DESC LIMIT 50) SELECT a.id, a.repo_source, a.repo_owner, a.repo_name, a.repo_branch, a.repo_revision, a.build_version, a.build_status, a.labels, a.release_targets, a.manifest, a.commits, a.triggers, a.inserted_at, a.updated_at, a.duration::INT, a.triggered_by_event FROM ranked_builds a WHERE a.rn = 1 ORDER BY a.inserted_at DESC LIMIT 5", sql)
	})
}

func TestAutoincrement(t *testing.T) {

	t.Run("TestAutoincrementRegex", func(t *testing.T) {

		buildVersion := "0.0.126-MeD-1234123"
		re := regexp.MustCompile(`^[0-9]+\.[0-9]+\.([0-9]+)(-[0-9a-zA-Z-/]+)?$`)
		match := re.FindStringSubmatch(buildVersion)
		autoincrement := 0
		if len(match) > 1 {
			autoincrement, _ = strconv.Atoi(match[1])
		}

		assert.Equal(t, 126, autoincrement)
	})
}

func getCockroachdbClient(ctx context.Context, t *testing.T) Client {

	dbConfig := config.DatabaseConfig{
		DatabaseName:   "defaultdb",
		Host:           "cockroachdb",
		Insecure:       true,
		CertificateDir: "",
		Port:           26257,
		User:           "root",
		Password:       "",
	}

	cockroachdbClient := NewClient(dbConfig)
	err := cockroachdbClient.Connect(ctx)

	assert.Nil(t, err)

	return cockroachdbClient
}

func getBuild() contracts.Build {
	return contracts.Build{
		RepoSource:     "github.com",
		RepoOwner:      "estafette",
		RepoName:       "estafette-ci-api",
		RepoBranch:     "master",
		RepoRevision:   "08e9480b75154b5584995053344beb4d4aef65f4",
		BuildVersion:   "0.0.99",
		BuildStatus:    "pending",
		Labels:         []contracts.Label{{Key: "app-group", Value: "estafette-ci"}, {Key: "language", Value: "golang"}},
		ReleaseTargets: []contracts.ReleaseTarget{},
		Manifest:       "",
		Commits:        []contracts.GitCommit{},
		Triggers:       []manifest.EstafetteTrigger{},
		Events:         []manifest.EstafetteEvent{},
	}
}

func getRelease() contracts.Release {
	return contracts.Release{
		Name:           "production",
		Action:         "",
		RepoSource:     "github.com",
		RepoOwner:      "estafette",
		RepoName:       "estafette-ci-api",
		ReleaseVersion: "0.0.99",
		ReleaseStatus:  "pending",
		Events:         []manifest.EstafetteEvent{},
	}
}

func getJobResources() JobResources {
	return JobResources{
		CPURequest:    float64(0.1),
		CPULimit:      float64(7.0),
		MemoryRequest: float64(67108864),
		MemoryLimit:   float64(21474836480),
	}
}

func getBuildLog() contracts.BuildLog {
	return contracts.BuildLog{
		RepoSource:   "github.com",
		RepoOwner:    "estafette",
		RepoName:     "estafette-ci-api",
		RepoBranch:   "master",
		RepoRevision: "08e9480b75154b5584995053344beb4d4aef65f4",
		Steps: []*contracts.BuildLogStep{
			&contracts.BuildLogStep{
				Step: "stage-1",
				Image: &contracts.BuildLogStepDockerImage{
					Name: "golang",
					Tag:  "1.14.2-alpine3.11",
				},
				Duration: time.Duration(1234567),
				Status:   "SUCCEEDED",
				LogLines: []contracts.BuildLogLine{
					contracts.BuildLogLine{
						LineNumber: 1,
						Timestamp:  time.Now().UTC(),
						StreamType: "stdout",
						Text:       "ok",
					},
				},
			},
		},
	}
}

func getReleaseLog() contracts.ReleaseLog {
	return contracts.ReleaseLog{
		RepoSource: "github.com",
		RepoOwner:  "estafette",
		RepoName:   "estafette-ci-api",
		Steps: []*contracts.BuildLogStep{
			&contracts.BuildLogStep{
				Step: "stage-1",
				Image: &contracts.BuildLogStepDockerImage{
					Name: "golang",
					Tag:  "1.14.2-alpine3.11",
				},
				Duration: time.Duration(1234567),
				Status:   "SUCCEEDED",
				LogLines: []contracts.BuildLogLine{
					contracts.BuildLogLine{
						LineNumber: 1,
						Timestamp:  time.Now().UTC(),
						StreamType: "stdout",
						Text:       "ok",
					},
				},
			},
		},
	}
}
