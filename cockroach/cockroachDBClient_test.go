package cockroach

// import (
// 	"testing"

// 	"github.com/prometheus/client_golang/prometheus"
// 	"github.com/stretchr/testify/assert"

// 	_ "github.com/mattn/go-sqlite3"
// )

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
