package cockroach

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	sq "github.com/Masterminds/squirrel"
	"github.com/estafette/estafette-ci-contracts"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/rs/zerolog/log"

	// use postgres client library to connect to cockroachdb
	_ "github.com/lib/pq"
)

// DBClient is the interface for communicating with CockroachDB
type DBClient interface {
	Connect() error
	ConnectWithDriverAndSource(string, string) error
	InsertBuildJobLogs(BuildJobLogs) error
	GetBuildLogs(BuildJobLogs) ([]BuildJobLogRow, error)
	GetAutoIncrement(string, string) (int, error)
	InsertBuild(contracts.Build) error
	UpdateBuildStatus(string, string, string, string, string) error
	GetPipelines(int, int, map[string][]string) ([]*contracts.Pipeline, error)
	GetPipelinesCount(map[string][]string) (int, error)
	GetPipeline(string, string, string) (*contracts.Pipeline, error)
	GetPipelineBuilds(string, string, string, int, int) ([]*contracts.Build, error)
	GetPipelineBuildsCount(string, string, string) (int, error)
	GetPipelineBuild(string, string, string, string) (*contracts.Build, error)
	GetPipelineBuildLogs(string, string, string, string) (*contracts.BuildLog, error)
	InsertBuildLog(contracts.BuildLog) error
}

type cockroachDBClientImpl struct {
	databaseDriver                  string
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

// InsertBuildJobLogs inserts build logs into the database
func (dbc *cockroachDBClientImpl) InsertBuildJobLogs(buildJobLogs BuildJobLogs) (err error) {

	dbc.PrometheusOutboundAPICallTotals.With(prometheus.Labels{"target": "cockroachdb"}).Inc()

	// insert logs
	_, err = dbc.databaseConnection.Exec(
		`
		INSERT INTO
			build_logs
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
		)
		`,
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

	rows, err := dbc.databaseConnection.Query(
		`
		SELECT
			*
		FROM
			build_logs
		WHERE
			repo_full_name=$1 AND
			repo_branch=$2 AND
			repo_revision=$3 AND
			repo_source=$4
		`,
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
	_, err = dbc.databaseConnection.Exec(
		`
		INSERT INTO
			build_versions
		(
			repo_source,
			repo_full_name
		)
		VALUES
		(
			$1,
			$2
		)
		ON CONFLICT
		(
			repo_source,
			repo_full_name
		)
		DO UPDATE SET
			auto_increment = build_versions.auto_increment + 1,
			updated_at = now()
		`,
		gitSource,
		gitFullname,
	)
	if err != nil {
		return
	}

	dbc.PrometheusOutboundAPICallTotals.With(prometheus.Labels{"target": "cockroachdb"}).Inc()

	// fetching auto_increment value, because RETURNING is not supported with UPSERT / INSERT ON CONFLICT (see issue https://github.com/cockroachdb/cockroach/issues/6637)
	rows, err := dbc.databaseConnection.Query(
		`
		SELECT
			auto_increment
		FROM
			build_versions
		WHERE
			repo_source=$1 AND
			repo_full_name=$2
		`,
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

func (dbc *cockroachDBClientImpl) InsertBuild(build contracts.Build) (err error) {
	dbc.PrometheusOutboundAPICallTotals.With(prometheus.Labels{"target": "cockroachdb"}).Inc()

	labelsBytes, err := json.Marshal(build.Labels)
	if err != nil {
		return
	}
	commitsBytes, err := json.Marshal(build.Commits)
	if err != nil {
		return
	}

	// insert logs
	_, err = dbc.databaseConnection.Exec(
		`
		INSERT INTO
			builds
		(
			repo_source,
			repo_owner,
			repo_name,
			repo_branch,
			repo_revision,
			build_version,
			build_status,
			labels,
			manifest,
			commits
		)
		VALUES
		(
			$1,
			$2,
			$3,
			$4,
			$5,
			$6,
			$7,
			$8,
			$9,
			$10
		)
		`,
		build.RepoSource,
		build.RepoOwner,
		build.RepoName,
		build.RepoBranch,
		build.RepoRevision,
		build.BuildVersion,
		build.BuildStatus,
		labelsBytes,
		build.Manifest,
		commitsBytes,
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
		`
		UPDATE
			builds
		SET
			build_status=$1,
			updated_at=now()
		WHERE
			repo_source=$2 AND
			repo_owner=$3 AND
			repo_name=$4 AND
			repo_revision=$5
		`,
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

func (dbc *cockroachDBClientImpl) GetPipelines(pageNumber, pageSize int, filters map[string][]string) (pipelines []*contracts.Pipeline, err error) {

	dbc.PrometheusOutboundAPICallTotals.With(prometheus.Labels{"target": "cockroachdb"}).Inc()

	psql := sq.StatementBuilder.PlaceholderFormat(sq.Dollar)

	innerQuery :=
		psql.
			Select("*, RANK() OVER (PARTITION BY repo_source,repo_owner,repo_name ORDER BY inserted_at DESC) AS build_version_rank").
			From("builds")

	query :=
		psql.
			Select("id,repo_source,repo_owner,repo_name,repo_branch,repo_revision,build_version,build_status,labels,manifest,commits,inserted_at,updated_at").
			FromSelect(innerQuery, "ranked_builds").
			Where(sq.Eq{"build_version_rank": 1})

	query, err = whereClauseGeneratorForAllFilters(query, filters)
	if err != nil {
		return
	}

	query = query.
		OrderBy("repo_source,repo_owner,repo_name").
		Limit(uint64(pageSize)).
		Offset(uint64((pageNumber - 1) * pageSize))

	pipelines = make([]*contracts.Pipeline, 0)

	rows, err := query.RunWith(dbc.databaseConnection).Query()

	if err != nil {
		return
	}

	defer rows.Close()
	for rows.Next() {

		var labelsData, commitsData []uint8

		pipeline := contracts.Pipeline{}

		if err := rows.Scan(
			&pipeline.ID,
			&pipeline.RepoSource,
			&pipeline.RepoOwner,
			&pipeline.RepoName,
			&pipeline.RepoBranch,
			&pipeline.RepoRevision,
			&pipeline.BuildVersion,
			&pipeline.BuildStatus,
			&labelsData,
			&pipeline.Manifest,
			&commitsData,
			&pipeline.InsertedAt,
			&pipeline.UpdatedAt); err != nil {
			return nil, err
		}

		if len(labelsData) > 0 {
			if err = json.Unmarshal(labelsData, &pipeline.Labels); err != nil {
				return
			}
		}
		if len(commitsData) > 0 {
			if err = json.Unmarshal(commitsData, &pipeline.Commits); err != nil {
				return
			}
		}

		pipelines = append(pipelines, &pipeline)
	}

	return
}

func (dbc *cockroachDBClientImpl) GetPipelinesCount(filters map[string][]string) (totalCount int, err error) {

	dbc.PrometheusOutboundAPICallTotals.With(prometheus.Labels{"target": "cockroachdb"}).Inc()

	psql := sq.StatementBuilder.PlaceholderFormat(sq.Dollar)

	innerQuery :=
		psql.
			Select("*, RANK() OVER (PARTITION BY repo_source,repo_owner,repo_name ORDER BY inserted_at DESC) AS build_version_rank").
			From("builds")

	query :=
		psql.
			Select("COUNT(*)").
			FromSelect(innerQuery, "ranked_builds").
			Where(sq.Eq{"build_version_rank": 1})

	query, err = whereClauseGeneratorForAllFilters(query, filters)
	if err != nil {
		return
	}

	rows, err := query.RunWith(dbc.databaseConnection).Query()
	if err != nil {
		return
	}

	defer rows.Close()
	recordExists := rows.Next()

	if !recordExists {
		return
	}

	if err := rows.Scan(&totalCount); err != nil {
		return 0, err
	}

	return
}

func whereClauseGeneratorForAllFilters(query sq.SelectBuilder, filters map[string][]string) (sq.SelectBuilder, error) {

	query, err := whereClauseGeneratorForStatusFilter(query, filters)
	if err != nil {
		return query, err
	}
	query, err = whereClauseGeneratorForSinceFilter(query, filters)
	if err != nil {
		return query, err
	}
	query, err = whereClauseGeneratorForLabelsFilter(query, filters)
	if err != nil {
		return query, err
	}

	return query, nil
}

func whereClauseGeneratorForStatusFilter(query sq.SelectBuilder, filters map[string][]string) (sq.SelectBuilder, error) {

	if statuses, ok := filters["status"]; ok {
		query = query.Where(sq.Eq{"build_status": statuses})
	}

	return query, nil
}

func whereClauseGeneratorForSinceFilter(query sq.SelectBuilder, filters map[string][]string) (sq.SelectBuilder, error) {

	if since, ok := filters["since"]; ok {
		sinceValue := since[0]
		switch sinceValue {
		case "1d":
			query = query.Where(sq.GtOrEq{"inserted_at": time.Now().AddDate(0, 0, -1)})
		case "1w":
			query = query.Where(sq.GtOrEq{"inserted_at": time.Now().AddDate(0, 0, -7)})
		case "1m":
			query = query.Where(sq.GtOrEq{"inserted_at": time.Now().AddDate(0, -1, 0)})
		case "1y":
			query = query.Where(sq.GtOrEq{"inserted_at": time.Now().AddDate(-1, 0, 0)})
		}
	}

	return query, nil
}

func whereClauseGeneratorForLabelsFilter(query sq.SelectBuilder, filters map[string][]string) (sq.SelectBuilder, error) {

	if labels, ok := filters["labels"]; ok {

		labelsParam := []contracts.Label{}

		for _, label := range labels {
			keyValuePair := strings.Split(label, "=")

			if len(keyValuePair) == 2 {
				labelsParam = append(labelsParam, contracts.Label{
					Key:   keyValuePair[0],
					Value: keyValuePair[1],
				})
			}
		}

		if len(labelsParam) > 0 {
			bytes, err := json.Marshal(labelsParam)
			if err != nil {
				return query, err
			}

			query = query.Where("labels @> ?", string(bytes))
		}
	}

	return query, nil
}

func (dbc *cockroachDBClientImpl) GetPipeline(repoSource, repoOwner, repoName string) (pipeline *contracts.Pipeline, err error) {

	dbc.PrometheusOutboundAPICallTotals.With(prometheus.Labels{"target": "cockroachdb"}).Inc()

	rows, err := dbc.databaseConnection.Query(
		`
		SELECT
			id,
			repo_source,
			repo_owner,
			repo_name,
			repo_branch,
			repo_revision,
			build_version,
			build_status,
			labels,
			manifest,
			commits,
			inserted_at,
			updated_at
		FROM
			builds
		WHERE
			repo_source=$1 AND
			repo_owner=$2 AND
			repo_name=$3
		ORDER BY
			inserted_at DESC
		LIMIT 1
		OFFSET 0
		`,
		repoSource,
		repoOwner,
		repoName,
	)
	if err != nil {
		return
	}

	recordExists := false

	defer rows.Close()
	recordExists = rows.Next()

	if !recordExists {
		return
	}

	var labelsData, commitsData []uint8

	pipeline = &contracts.Pipeline{}

	if err := rows.Scan(
		&pipeline.ID,
		&pipeline.RepoSource,
		&pipeline.RepoOwner,
		&pipeline.RepoName,
		&pipeline.RepoBranch,
		&pipeline.RepoRevision,
		&pipeline.BuildVersion,
		&pipeline.BuildStatus,
		&labelsData,
		&pipeline.Manifest,
		&commitsData,
		&pipeline.InsertedAt,
		&pipeline.UpdatedAt); err != nil {
		return nil, err
	}

	if len(labelsData) > 0 {
		if err = json.Unmarshal(labelsData, &pipeline.Labels); err != nil {
			return
		}
	}
	if len(commitsData) > 0 {
		if err = json.Unmarshal(commitsData, &pipeline.Commits); err != nil {
			return
		}
	}

	return
}

func (dbc *cockroachDBClientImpl) GetPipelineBuilds(repoSource, repoOwner, repoName string, pageNumber, pageSize int) (builds []*contracts.Build, err error) {

	dbc.PrometheusOutboundAPICallTotals.With(prometheus.Labels{"target": "cockroachdb"}).Inc()

	builds = make([]*contracts.Build, 0)

	rows, err := dbc.databaseConnection.Query(
		`
		SELECT
			id,
			repo_source,
			repo_owner,
			repo_name,
			repo_branch,
			repo_revision,
			build_version,
			build_status,
			labels,
			manifest,
			commits,
			inserted_at,
			updated_at
		FROM
			builds
		WHERE
			repo_source=$1 AND
			repo_owner=$2 AND
			repo_name=$3
		ORDER BY
			inserted_at DESC
		LIMIT $4
		OFFSET $5
		`,
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

		var labelsData, commitsData []uint8

		build := contracts.Build{}

		if err := rows.Scan(
			&build.ID,
			&build.RepoSource,
			&build.RepoOwner,
			&build.RepoName,
			&build.RepoBranch,
			&build.RepoRevision,
			&build.BuildVersion,
			&build.BuildStatus,
			&labelsData,
			&build.Manifest,
			&commitsData,
			&build.InsertedAt,
			&build.UpdatedAt); err != nil {
			return nil, err
		}

		if len(labelsData) > 0 {
			if err = json.Unmarshal(labelsData, &build.Labels); err != nil {
				return
			}
		}
		if len(commitsData) > 0 {
			if err = json.Unmarshal(commitsData, &build.Commits); err != nil {
				return
			}
		}

		builds = append(builds, &build)
	}

	return
}

func (dbc *cockroachDBClientImpl) GetPipelineBuildsCount(repoSource, repoOwner, repoName string) (totalCount int, err error) {

	dbc.PrometheusOutboundAPICallTotals.With(prometheus.Labels{"target": "cockroachdb"}).Inc()

	rows, err := dbc.databaseConnection.Query(
		`
		SELECT
			COUNT(*)
		FROM
			builds
		WHERE
			repo_source=$1 AND
			repo_owner=$2 AND
			repo_name=$3
		`,
		repoSource,
		repoOwner,
		repoName,
	)
	if err != nil {
		return
	}

	defer rows.Close()
	recordExists := rows.Next()

	if !recordExists {
		return
	}

	if err := rows.Scan(&totalCount); err != nil {
		return 0, err
	}

	return
}

func (dbc *cockroachDBClientImpl) GetPipelineBuild(repoSource, repoOwner, repoName, repoRevision string) (build *contracts.Build, err error) {

	dbc.PrometheusOutboundAPICallTotals.With(prometheus.Labels{"target": "cockroachdb"}).Inc()

	rows, err := dbc.databaseConnection.Query(
		`
		SELECT
			id,
			repo_source,
			repo_owner,
			repo_name,
			repo_branch,
			repo_revision,
			build_version,
			build_status,
			labels,
			manifest,
			commits,
			inserted_at,
			updated_at
		FROM
			builds
		WHERE
			repo_source=$1 AND
			repo_owner=$2 AND
			repo_name=$3 AND
			repo_revision=$4
		ORDER BY
			inserted_at DESC
		LIMIT 1
		OFFSET 0
		`,
		repoSource,
		repoOwner,
		repoName,
		repoRevision,
	)
	if err != nil {
		return
	}

	recordExists := false

	defer rows.Close()
	recordExists = rows.Next()

	if !recordExists {
		return
	}

	var labelsData, commitsData []uint8

	build = &contracts.Build{}

	if err := rows.Scan(
		&build.ID,
		&build.RepoSource,
		&build.RepoOwner,
		&build.RepoName,
		&build.RepoBranch,
		&build.RepoRevision,
		&build.BuildVersion,
		&build.BuildStatus,
		&labelsData,
		&build.Manifest,
		&commitsData,
		&build.InsertedAt,
		&build.UpdatedAt); err != nil {
		return nil, err
	}

	if len(labelsData) > 0 {
		if err = json.Unmarshal(labelsData, &build.Labels); err != nil {
			return nil, err
		}
	}
	if len(commitsData) > 0 {
		if err = json.Unmarshal(commitsData, &build.Commits); err != nil {
			return nil, err
		}
	}

	return
}

func (dbc *cockroachDBClientImpl) GetPipelineBuildLogs(repoSource, repoOwner, repoName, repoRevision string) (buildLog *contracts.BuildLog, err error) {

	dbc.PrometheusOutboundAPICallTotals.With(prometheus.Labels{"target": "cockroachdb"}).Inc()

	rows, err := dbc.databaseConnection.Query(
		`
		SELECT
			id,
			repo_source,
			repo_owner,
			repo_name,
			repo_branch,
			repo_revision,
			steps,
			inserted_at
		FROM
			build_logs_v2
		WHERE
			repo_source=$1 AND
			repo_owner=$2 AND
			repo_name=$3 AND
			repo_revision=$4
		LIMIT 1
		`,
		repoSource,
		repoOwner,
		repoName,
		repoRevision,
	)
	if err != nil {
		return
	}

	recordExists := false

	defer rows.Close()
	recordExists = rows.Next()

	if !recordExists {
		return
	}

	buildLog = &contracts.BuildLog{}

	var stepsData []uint8

	if err = rows.Scan(&buildLog.ID, &buildLog.RepoSource, &buildLog.RepoOwner, &buildLog.RepoName, &buildLog.RepoBranch, &buildLog.RepoRevision, &stepsData, &buildLog.InsertedAt); err != nil {
		return
	}

	if err = json.Unmarshal(stepsData, &buildLog.Steps); err != nil {
		return
	}

	return
}

func (dbc *cockroachDBClientImpl) InsertBuildLog(buildLog contracts.BuildLog) (err error) {

	dbc.PrometheusOutboundAPICallTotals.With(prometheus.Labels{"target": "cockroachdb"}).Inc()

	bytes, err := json.Marshal(buildLog.Steps)
	if err != nil {
		return
	}

	// insert logs
	_, err = dbc.databaseConnection.Exec(
		`
		INSERT INTO
			build_logs_v2
		(
			repo_source,
			repo_owner,
			repo_name,
			repo_branch,
			repo_revision,
			steps
		)
		VALUES
		(
			$1,
			$2,
			$3,
			$4,
			$5,
			$6
		)
		`,
		buildLog.RepoSource,
		buildLog.RepoOwner,
		buildLog.RepoName,
		buildLog.RepoBranch,
		buildLog.RepoRevision,
		bytes,
	)

	if err != nil {
		return
	}

	return
}
