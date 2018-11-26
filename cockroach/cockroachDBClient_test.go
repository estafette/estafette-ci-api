package cockroach

import (
	"regexp"
	"strconv"
	"testing"

	"github.com/estafette/estafette-ci-api/config"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/assert"
)

var (
	cdbClient = NewCockroachDBClient(config.DatabaseConfig{}, prometheus.NewCounterVec(
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

		query, _ = whereClauseGeneratorForAllFilters(query, "a", map[string][]string{})

		// act
		sql, _, err := query.ToSql()

		assert.Nil(t, err)
		assert.Equal(t, "SELECT a.id, a.repo_source, a.repo_owner, a.repo_name, a.repo_branch, a.repo_revision, a.build_version, a.build_status, a.labels, a.release_targets, a.manifest, a.commits, a.inserted_at, a.updated_at, a.duration::INT FROM builds a", sql)
	})

	t.Run("GeneratesQueryWithStatusFilter", func(t *testing.T) {

		query := cdbClient.selectBuildsQuery()

		query, _ = whereClauseGeneratorForAllFilters(query, "a", map[string][]string{
			"status": []string{
				"succeeded",
			},
		})

		// act
		sql, _, err := query.ToSql()

		assert.Nil(t, err)
		assert.Equal(t, "SELECT a.id, a.repo_source, a.repo_owner, a.repo_name, a.repo_branch, a.repo_revision, a.build_version, a.build_status, a.labels, a.release_targets, a.manifest, a.commits, a.inserted_at, a.updated_at, a.duration::INT FROM builds a WHERE a.build_status IN ($1)", sql)
	})

	t.Run("GeneratesQueryWithSinceFilter", func(t *testing.T) {

		query := cdbClient.selectBuildsQuery()

		query, _ = whereClauseGeneratorForAllFilters(query, "a", map[string][]string{
			"since": []string{
				"1d",
			},
		})

		// act
		sql, _, err := query.ToSql()

		assert.Nil(t, err)
		assert.Equal(t, "SELECT a.id, a.repo_source, a.repo_owner, a.repo_name, a.repo_branch, a.repo_revision, a.build_version, a.build_status, a.labels, a.release_targets, a.manifest, a.commits, a.inserted_at, a.updated_at, a.duration::INT FROM builds a WHERE a.inserted_at >= $1", sql)
	})

	t.Run("GeneratesQueryWithLabelsFilter", func(t *testing.T) {

		query := cdbClient.selectBuildsQuery()

		query, _ = whereClauseGeneratorForAllFilters(query, "a", map[string][]string{
			"labels": []string{
				"key=value",
			},
		})

		// act
		sql, _, err := query.ToSql()

		assert.Nil(t, err)
		assert.Equal(t, "SELECT a.id, a.repo_source, a.repo_owner, a.repo_name, a.repo_branch, a.repo_revision, a.build_version, a.build_status, a.labels, a.release_targets, a.manifest, a.commits, a.inserted_at, a.updated_at, a.duration::INT FROM builds a WHERE a.labels @> $1", sql)
	})

	t.Run("GeneratesQueryWithLabelsFilterAndOrderBy", func(t *testing.T) {

		pageSize := 15
		pageNumber := 2
		query := cdbClient.selectBuildsQuery().
			OrderBy("a.inserted_at DESC").
			Limit(uint64(pageSize)).
			Offset(uint64((pageNumber - 1) * pageSize))

		query, _ = whereClauseGeneratorForAllFilters(query, "a", map[string][]string{
			"labels": []string{
				"key=value",
			},
		})

		// act
		sql, _, err := query.ToSql()

		assert.Nil(t, err)
		assert.Equal(t, "SELECT a.id, a.repo_source, a.repo_owner, a.repo_name, a.repo_branch, a.repo_revision, a.build_version, a.build_status, a.labels, a.release_targets, a.manifest, a.commits, a.inserted_at, a.updated_at, a.duration::INT FROM builds a WHERE a.labels @> $1 ORDER BY a.inserted_at DESC LIMIT 15 OFFSET 15", sql)
	})

	t.Run("GeneratesGetPipelinesQuery", func(t *testing.T) {

		query := cdbClient.selectPipelinesQuery().
			LeftJoin("builds b ON a.repo_source=b.repo_source AND a.repo_owner=b.repo_owner AND a.repo_name=b.repo_name AND a.inserted_at < b.inserted_at").
			Where("b.id IS NULL").
			OrderBy("a.repo_source,a.repo_owner,a.repo_name").
			Limit(uint64(2)).
			Offset(uint64((2 - 1) * 20))

		// act
		sql, _, err := query.ToSql()

		assert.Nil(t, err)
		assert.Equal(t, "SELECT a.id, a.repo_source, a.repo_owner, a.repo_name, a.repo_branch, a.repo_revision, a.build_version, a.build_status, a.labels, a.release_targets, a.manifest, a.commits, a.inserted_at, a.updated_at, a.duration::INT FROM builds a LEFT JOIN builds b ON a.repo_source=b.repo_source AND a.repo_owner=b.repo_owner AND a.repo_name=b.repo_name AND a.inserted_at < b.inserted_at WHERE b.id IS NULL ORDER BY a.repo_source,a.repo_owner,a.repo_name LIMIT 2 OFFSET 20", sql)
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
