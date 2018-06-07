package cockroach

import (
	"testing"

	"github.com/stretchr/testify/assert"

	sq "github.com/Masterminds/squirrel"
	_ "github.com/mattn/go-sqlite3"
)

func TestQueryBuilder(t *testing.T) {
	t.Run("GeneratesQuery", func(t *testing.T) {

		psql := sq.StatementBuilder.PlaceholderFormat(sq.Dollar)

		query :=
			psql.
				Select("id,repo_source,repo_owner,repo_name,repo_branch,repo_revision,build_version,build_status,'' AS labels,manifest,inserted_at,updated_at").
				From("(SELECT *, RANK() OVER (PARTITION BY repo_source,repo_owner,repo_name ORDER BY inserted_at DESC) AS build_version_rank FROM builds)").
				Where(sq.Eq{"build_version_rank": 1}).
				OrderBy("repo_source,repo_owner,repo_name").
				Limit(uint64(2)).
				Offset(uint64((2 - 1) * 20))

		// act
		sql, _, err := query.ToSql()

		assert.Nil(t, err)
		assert.Equal(t, "SELECT id,repo_source,repo_owner,repo_name,repo_branch,repo_revision,build_version,build_status,'' AS labels,manifest,inserted_at,updated_at FROM (SELECT *, RANK() OVER (PARTITION BY repo_source,repo_owner,repo_name ORDER BY inserted_at DESC) AS build_version_rank FROM builds) WHERE build_version_rank = $1 ORDER BY repo_source,repo_owner,repo_name LIMIT 2 OFFSET 20", sql)
	})
}

// func TestConnectWithDriverAndSource(t *testing.T) {

// 	t.Run("OpensConnectionToDatabase", func(t *testing.T) {

// 		cockroachDBClient := GetTestDBClient()

// 		// act
// 		err := cockroachDBClient.ConnectWithDriverAndSource("sqlite3", ":memory:")

// 		assert.Nil(t, err)
// 	})
// }

// func TestMigrateSchema(t *testing.T) {

// 	t.Run("MigratesDatabaseSchema", func(t *testing.T) {

// 		cockroachDBClient := GetTestDBClient()

// 		_ = cockroachDBClient.ConnectWithDriverAndSource("sqlite3", ":memory:")

// 		// act
// 		err := cockroachDBClient.MigrateSchema()

// 		assert.Nil(t, err)
// 	})
// }

// func TestInsertBuildJobLogs(t *testing.T) {

// 	t.Run("InsertsLogRow", func(t *testing.T) {

// 		cockroachDBClient := GetTestDBClient()
// 		_ = cockroachDBClient.ConnectWithDriverAndSource("sqlite3", ":memory:")
// 		_ = cockroachDBClient.MigrateSchema()

// 		buildJobLogs := BuildJobLogs{
// 			RepoFullName: "estafette/estafette-ci-api",
// 			RepoBranch:   "master",
// 			RepoRevision: "605fed1c8e57b6c11a24120e6be1344c0dad038a",
// 			RepoSource:   "github.com",
// 			LogText:      "First line\nSecond line",
// 		}

// 		// act
// 		err := cockroachDBClient.InsertBuildJobLogs(buildJobLogs)

// 		assert.Nil(t, err)
// 	})
// }

// func GetTestDBClient() DBClient {

// 	testPrometheusVec := prometheus.NewCounterVec(
// 		prometheus.CounterOpts{
// 			Name: "estafette_ci_api_outbound_api_call_totals",
// 			Help: "Total of outgoing api calls.",
// 		},
// 		[]string{"target"},
// 	)
// 	testRegistry := prometheus.NewRegistry()
// 	testRegistry.MustRegister(testPrometheusVec)

// 	return &cockroachDBClientImpl{
// 		databaseDriver:                  "sqlite3",
// 		migrationsDir:                   "./migrations",
// 		PrometheusOutboundAPICallTotals: testPrometheusVec,
// 	}
// }
