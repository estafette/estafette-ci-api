package cockroach

import (
	"database/sql"
	"fmt"

	"github.com/pressly/goose"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/rs/zerolog/log"

	// use postgres client library to connect to cockroachdb
	_ "github.com/lib/pq"
)

// DBClient is the interface for communicating with CockroachDB
type DBClient interface {
	Connect() error
	ConnectWithDriverAndSource(string, string) error
	MigrateSchema() error
	InsertBuildJobLogs(BuildJobLogs) error
	GetBuildLogs(BuildJobLogs) error
}

type cockroachDBClientImpl struct {
	databaseDriver                  string
	migrationsDir                   string
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
		databaseDriver:                  "postgres",
		migrationsDir:                   "/migrations",
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

	return dbc.ConnectWithDriverAndSource(dbc.databaseDriver, dataSourceName)
}

// ConnectWithDriverAndSource set up a connection with any database
func (dbc *cockroachDBClientImpl) ConnectWithDriverAndSource(driverName string, dataSourceName string) (err error) {

	dbc.databaseConnection, err = sql.Open(driverName, dataSourceName)
	if err != nil {
		return
	}

	return
}

// MigrateSchema migrates the schema in CockroachDB
func (dbc *cockroachDBClientImpl) MigrateSchema() (err error) {

	err = goose.SetDialect(dbc.databaseDriver)
	if err != nil {
		return err
	}

	err = goose.Status(dbc.databaseConnection, dbc.migrationsDir)
	if err != nil {
		return err
	}

	err = goose.Up(dbc.databaseConnection, dbc.migrationsDir)
	if err != nil {
		return err
	}

	return
}

// InsertBuildJobLogs inserts build logs into the database
func (dbc *cockroachDBClientImpl) InsertBuildJobLogs(buildJobLogs BuildJobLogs) (err error) {

	dbc.PrometheusOutboundAPICallTotals.With(prometheus.Labels{"target": "cockroachdb"}).Inc()

	// insert logs
	r, err := dbc.databaseConnection.Exec(
		"INSERT INTO build_logs (repo_full_name,repo_branch,repo_revision,repo_source,log_text) VALUES ($1,$2,$3,$4,$5)",
		buildJobLogs.RepoFullName,
		buildJobLogs.RepoBranch,
		buildJobLogs.RepoRevision,
		buildJobLogs.RepoSource,
		buildJobLogs.LogText,
	)

	if err != nil {
		return
	}

	lastInsertID, err := r.LastInsertId()
	if err != nil {
		log.Warn().Err(err).
			Interface("buildJobLogs", buildJobLogs).
			Msgf("Getting LastInsertId for %v failed", buildJobLogs.RepoFullName)
	}

	rowsAffected, err := r.RowsAffected()
	if err != nil {
		log.Warn().Err(err).
			Interface("buildJobLogs", buildJobLogs).
			Msgf("Getting RowsAffected for %v failed", buildJobLogs.RepoFullName)
	}

	log.Debug().
		Int64("LastInsertId", lastInsertID).
		Int64("RowsAffected", rowsAffected).
		Msg("Inserted log record")

	return
}

// GetBuildJobLogs reads a specific log
func (dbc *cockroachDBClientImpl) GetBuildLogs(buildJobLogs BuildJobLogs) (err error) {

	rows, err := dbc.databaseConnection.Query("SELECT * FROM build_logs WHERE repo_full_name=$1 AND repo_branch=$2 AND repo_revision=$3 AND repo_source=$4",
		buildJobLogs.RepoFullName,
		buildJobLogs.RepoBranch,
		buildJobLogs.RepoRevision,
		buildJobLogs.RepoSource,
	)
	if err != nil {
		return
	}

	defer rows.Close()
	for rows.Next() {
		var id int
		var repo_full_name, repo_branch, repo_revision, repo_source, log_text string

		if err := rows.Scan(&id, &repo_full_name, &repo_branch, &repo_revision, &repo_source, &log_text); err != nil {
			return err
		}

		log.Debug().
			Int("id", id).
			Str("repo_full_name", repo_full_name).
			Str("repo_branch", repo_branch).
			Str("repo_revision", repo_revision).
			Str("repo_source", repo_source).
			Str("log_text", log_text).
			Msg("Read logs")
	}

	return
}
