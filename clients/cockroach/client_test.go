package cockroach

import (
	"fmt"
	"regexp"
	"strconv"
	"testing"
	"time"

	sq "github.com/Masterminds/squirrel"
	"github.com/estafette/estafette-ci-api/config"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/assert"
)

var (
	cdbClient = NewClient(config.DatabaseConfig{}, prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "estafette_ci_api_outbound_api_call_totals",
			Help: "Total of outgoing api calls.",
		},
		[]string{"target"},
	))
)

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
