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
	GetBuildLogs(BuildJobLogs) ([]BuildJobLogRow, error)
	GetAutoIncrement(string, string) (int, error)
	InsertBuild(Build) error
	UpdateBuildStatus(string, string, string, string, string) error
	GetPipelines(int, int) ([]*Build, error)
	GetPipelineBuilds(string, string, string, int, int) ([]*Build, error)
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
	_, err = dbc.databaseConnection.Exec(
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

	return
}

// GetBuildJobLogs reads a specific log
func (dbc *cockroachDBClientImpl) GetBuildLogs(buildJobLogs BuildJobLogs) (logs []BuildJobLogRow, err error) {

	dbc.PrometheusOutboundAPICallTotals.With(prometheus.Labels{"target": "cockroachdb"}).Inc()

	logs = make([]BuildJobLogRow, 0)

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

		logRow := BuildJobLogRow{}

		if err := rows.Scan(&logRow.ID, &logRow.RepoFullName, &logRow.RepoBranch, &logRow.RepoRevision, &logRow.RepoSource, &logRow.LogText, &logRow.InsertedAt); err != nil {
			return nil, err
		}

		logs = append(logs, logRow)
	}

	return
}

// GetBuildJobLogs reads a specific log
func (dbc *cockroachDBClientImpl) GetAutoIncrement(gitSource, gitFullname string) (autoincrement int, err error) {

	dbc.PrometheusOutboundAPICallTotals.With(prometheus.Labels{"target": "cockroachdb"}).Inc()

	// insert or increment if record for repo_source and repo_full_name combination already exists
	_, err = dbc.databaseConnection.Exec("INSERT INTO build_versions (repo_source, repo_full_name) VALUES ($1, $2) ON CONFLICT (repo_source, repo_full_name) DO UPDATE SET auto_increment = build_versions.auto_increment + 1, updated_at = now()",
		gitSource,
		gitFullname,
	)
	if err != nil {
		return
	}

	dbc.PrometheusOutboundAPICallTotals.With(prometheus.Labels{"target": "cockroachdb"}).Inc()

	// fetching auto_increment value, because RETURNING is not supported with UPSERT / INSERT ON CONFLICT (see issue https://github.com/cockroachdb/cockroach/issues/6637)
	rows, err := dbc.databaseConnection.Query("SELECT auto_increment FROM build_versions WHERE repo_source=$1 AND repo_full_name=$2",
		gitSource,
		gitFullname,
	)
	if err != nil {
		return
	}

	defer rows.Close()
	for rows.Next() {
		if err = rows.Scan(&autoincrement); err != nil {
			return
		}
	}

	return
}

func (dbc *cockroachDBClientImpl) InsertBuild(build Build) (err error) {
	dbc.PrometheusOutboundAPICallTotals.With(prometheus.Labels{"target": "cockroachdb"}).Inc()

	// insert logs
	_, err = dbc.databaseConnection.Exec(
		"INSERT INTO builds (repo_source,repo_owner,repo_name,repo_branch,repo_revision,build_version,build_status,manifest) VALUES ($1,$2,$3,$4,$5,$6,$7,$8)",
		build.RepoSource,
		build.RepoOwner,
		build.RepoName,
		build.RepoBranch,
		build.RepoRevision,
		build.BuildVersion,
		build.BuildStatus,
		build.Manifest,
	)

	if err != nil {
		return
	}

	return
}

func (dbc *cockroachDBClientImpl) UpdateBuildStatus(repoSource, repoOwner, repoName, repoRevision, buildStatus string) (err error) {

	dbc.PrometheusOutboundAPICallTotals.With(prometheus.Labels{"target": "cockroachdb"}).Inc()

	// insert logs
	_, err = dbc.databaseConnection.Exec(
		"UPDATE builds SET build_status=$1,updated_at=now() WHERE repo_source=$2 AND repo_owner=$3 AND repo_name=$4 AND repo_revision=$5",
		buildStatus,
		repoSource,
		repoOwner,
		repoName,
		repoRevision,
	)

	if err != nil {
		return
	}

	return
}

func (dbc *cockroachDBClientImpl) GetPipelines(pageNumber, pageSize int) (builds []*Build, err error) {

	dbc.PrometheusOutboundAPICallTotals.With(prometheus.Labels{"target": "cockroachdb"}).Inc()

	builds = make([]*Build, 0)
	rows, err := dbc.databaseConnection.Query(`SELECT CONCAT(repo_source,'/',repo_owner,'/',repo_name) as id,repo_source,repo_owner,repo_name,repo_branch,repo_revision,build_version,build_status,'' as labels,manifest,inserted_at,updated_at FROM (
     SELECT *, 
       RANK() OVER (PARTITION BY repo_source,repo_owner,repo_name ORDER BY inserted_at DESC) build_version_rank
       FROM builds
	 ) where build_version_rank = 1 ORDER BY repo_source,repo_owner,repo_name LIMIT $1 OFFSET $2`,
		pageSize,
		(pageNumber-1)*pageSize,
	)
	if err != nil {
		return
	}

	defer rows.Close()
	for rows.Next() {

		build := Build{}

		if err := rows.Scan(
			&build.ID,
			&build.RepoSource,
			&build.RepoOwner,
			&build.RepoName,
			&build.RepoBranch,
			&build.RepoRevision,
			&build.BuildVersion,
			&build.BuildStatus,
			&build.Labels,
			&build.Manifest,
			&build.InsertedAt,
			&build.UpdatedAt); err != nil {
			return nil, err
		}

		builds = append(builds, &build)
	}

	return
}

func (dbc *cockroachDBClientImpl) GetPipelineBuilds(repoSource, repoOwner, repoName string, pageNumber, pageSize int) (builds []*Build, err error) {

	dbc.PrometheusOutboundAPICallTotals.With(prometheus.Labels{"target": "cockroachdb"}).Inc()

	builds = make([]*Build, 0)

	rows, err := dbc.databaseConnection.Query("SELECT CONCAT(repo_source,'/',repo_owner,'/',repo_name) as id,repo_source,repo_owner,repo_name,repo_branch,repo_revision,build_version,build_status,'' as labels,manifest,inserted_at,updated_at FROM builds WHERE repo_source=$1 AND repo_owner=$2 AND repo_name=$3 ORDER BY inserted_at DESC LIMIT $4 OFFSET $5",
		repoSource,
		repoOwner,
		repoName,
		pageSize,
		(pageNumber-1)*pageSize,
	)
	if err != nil {
		return
	}

	defer rows.Close()
	for rows.Next() {

		build := Build{}

		if err := rows.Scan(
			&build.ID,
			&build.RepoSource,
			&build.RepoOwner,
			&build.RepoName,
			&build.RepoBranch,
			&build.RepoRevision,
			&build.BuildVersion,
			&build.BuildStatus,
			&build.Labels,
			&build.Manifest,
			&build.InsertedAt,
			&build.UpdatedAt); err != nil {
			return nil, err
		}

		builds = append(builds, &build)
	}

	return
}
