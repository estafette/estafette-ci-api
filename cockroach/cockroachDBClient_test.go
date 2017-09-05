package cockroach

import (
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/assert"

	_ "github.com/mattn/go-sqlite3"
)

func TestConnectWithDriverAndSource(t *testing.T) {

	t.Run("OpensConnectionToDatabase", func(t *testing.T) {

		cockroachDBClient := &cockroachDBClientImpl{
			databaseDialect: "sqlite3",
			migrationsDir:   "./cockroach/migrations",
			PrometheusOutboundAPICallTotals: prometheus.NewCounterVec(prometheus.CounterOpts{
				Name: "estafette_ci_api_inbound_event_totals",
			}, []string{"event", "source"})}

		// act
		err := cockroachDBClient.ConnectWithDriverAndSource("sqlite3", "file::memory:?mode=memory&cache=shared")

		assert.Nil(t, err)
	})
}

func TestMigrateSchema(t *testing.T) {

	t.Run("MigratesDatabaseSchema", func(t *testing.T) {

		cockroachDBClient := &cockroachDBClientImpl{
			databaseDialect: "sqlite3",
			migrationsDir:   "./cockroach/migrations",
			PrometheusOutboundAPICallTotals: prometheus.NewCounterVec(prometheus.CounterOpts{
				Name: "estafette_ci_api_inbound_event_totals",
			}, []string{"event", "source"})}

		_ = cockroachDBClient.ConnectWithDriverAndSource("sqlite3", "file::memory:?mode=memory&cache=shared")

		// act
		err := cockroachDBClient.MigrateSchema()

		assert.Nil(t, err)
	})
}

func TestInsertBuildJobLogs(t *testing.T) {

	t.Run("InsertsLogRow", func(t *testing.T) {

		cockroachDBClient := &cockroachDBClientImpl{
			databaseDialect: "sqlite3",
			migrationsDir:   "./cockroach/migrations",
			PrometheusOutboundAPICallTotals: prometheus.NewCounterVec(prometheus.CounterOpts{
				Name: "estafette_ci_api_inbound_event_totals",
			}, []string{"event", "source"})}

		_ = cockroachDBClient.ConnectWithDriverAndSource("sqlite3", "file::memory:?mode=memory&cache=shared")
		_ = cockroachDBClient.MigrateSchema()

		buildJobLogs := BuildJobLogs{
			RepoFullName: "estafette/estafette-ci-api",
			RepoBranch:   "master",
			RepoRevision: "605fed1c8e57b6c11a24120e6be1344c0dad038a",
			RepoSource:   "github",
			LogText:      "First line\nSecond line",
		}

		// act
		err := cockroachDBClient.InsertBuildJobLogs(buildJobLogs)

		assert.Nil(t, err)
	})
}
