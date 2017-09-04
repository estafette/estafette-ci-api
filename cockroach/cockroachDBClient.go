package cockroach

import (
	"database/sql"
	"fmt"

	"github.com/pressly/goose"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/rs/zerolog/log"
)

// DBClient is the interface for communicating with CockroachDB
type DBClient interface {
	Connect() error
	MigrateSchema() error
	InsertBuildJobLogs(BuildJobLogs) error
}

type cockroachDBClientImpl struct {
	cockroachDatabase               string
	cockroachHost                   string
	cockroachInsecure               bool
	cockroachCertificateDir         string
	cockroachPort                   int
	cockroachUser                   string
	cockroachPassword               string
	PrometheusOutboundAPICallTotals *prometheus.CounterVec
	databaseConnection              *sql.DB
}

// NewCockroachDBClient returns a new cockroach.DBClient
func NewCockroachDBClient(cockroachDatabase, cockroachHost string, cockroachInsecure bool, cockroachCertificateDir string, cockroachPort int, cockroachUser, cockroachPassword string, prometheusOutboundAPICallTotals *prometheus.CounterVec) (cockroachDBClient DBClient) {

	cockroachDBClient = &cockroachDBClientImpl{
		cockroachDatabase:               cockroachDatabase,
		cockroachHost:                   cockroachHost,
		cockroachInsecure:               cockroachInsecure,
		cockroachCertificateDir:         cockroachCertificateDir,
		cockroachPort:                   cockroachPort,
		cockroachUser:                   cockroachUser,
		cockroachPassword:               cockroachPassword,
		PrometheusOutboundAPICallTotals: prometheusOutboundAPICallTotals,
	}

	return
}

// Connect sets up a connection with CockroachDB
func (dbc *cockroachDBClientImpl) Connect() (err error) {

	log.Debug().Msgf("Connecting to database %v on host %v...", dbc.cockroachDatabase, dbc.cockroachHost)

	sslMode := ""
	if dbc.cockroachInsecure {
		sslMode = "?sslmode=disable"
	}

	dataSourceName := fmt.Sprintf("postgresql://%v:%v@%v:%v/%v%v", dbc.cockroachUser, dbc.cockroachPassword, dbc.cockroachHost, dbc.cockroachPort, dbc.cockroachDatabase, sslMode)

	dbc.databaseConnection, err = sql.Open("postgres", dataSourceName)
	if err != nil {
		return
	}

	return
}

// MigrateSchema migrates the schema in CockroachDB
func (dbc *cockroachDBClientImpl) MigrateSchema() (err error) {

	err = goose.Up(dbc.databaseConnection, "/migrations")
	if err != nil {
		return err
	}

	return
}

func (dbc *cockroachDBClientImpl) InsertBuildJobLogs(buildJobLogs BuildJobLogs) (err error) {

	// Insert two rows into the "accounts" table.
	_, err = dbc.databaseConnection.Exec(
		`INSERT INTO build_logs
		(
			repo_full_name,
			repo_branch,
			repo_revision,
			repo_source,
			log_text
		)
		VALUES
		(
			$1,
			$2,
			$3,
			$4,
			$5
		)`,
		buildJobLogs.RepoFullName,
		buildJobLogs.RepoBranch,
		buildJobLogs.RepoRevision,
		buildJobLogs.RepoSource,
		buildJobLogs.LogText)

	if err != nil {
		return
	}

	return
}
