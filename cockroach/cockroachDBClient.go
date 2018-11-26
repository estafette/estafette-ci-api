package cockroach

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	sq "github.com/Masterminds/squirrel"
	"github.com/estafette/estafette-ci-api/config"
	"github.com/estafette/estafette-ci-contracts"
	"github.com/estafette/estafette-ci-manifest"
	_ "github.com/lib/pq" // use postgres client library to connect to cockroachdb
	"github.com/prometheus/client_golang/prometheus"
	"github.com/rs/zerolog/log"
	yaml "gopkg.in/yaml.v2"
)

// DBClient is the interface for communicating with CockroachDB
type DBClient interface {
	Connect() error
	ConnectWithDriverAndSource(string, string) error
	GetAutoIncrement(string, string) (int, error)

	InsertBuild(contracts.Build) (contracts.Build, error)
	UpdateBuildStatus(string, string, string, int, string) error
	InsertRelease(contracts.Release) (contracts.Release, error)
	UpdateReleaseStatus(string, string, string, int, string) error
	InsertBuildLog(contracts.BuildLog) error
	InsertReleaseLog(contracts.ReleaseLog) error

	UpsertComputedPipelineByRepo(string, string, string) (*contracts.Pipeline, error)
	UpsertComputedPipeline(*contracts.Pipeline) (*contracts.Pipeline, error)

	GetPipelines(int, int, map[string][]string) ([]*contracts.Pipeline, error)
	GetPipelinesByRepoName(string) ([]*contracts.Pipeline, error)
	GetPipelinesCount(map[string][]string) (int, error)
	GetPipeline(string, string, string) (*contracts.Pipeline, error)
	GetPipelineBuilds(string, string, string, int, int, map[string][]string) ([]*contracts.Build, error)
	GetPipelineBuildsCount(string, string, string, map[string][]string) (int, error)
	GetPipelineBuild(string, string, string, string) (*contracts.Build, error)
	GetPipelineBuildByID(string, string, string, int) (*contracts.Build, error)
	GetLastPipelineBuild(string, string, string) (*contracts.Build, error)
	GetPipelineBuildsByVersion(string, string, string, string) ([]*contracts.Build, error)
	GetPipelineBuildLogs(string, string, string, string, string, string) (*contracts.BuildLog, error)
	GetPipelineReleases(string, string, string, int, int, map[string][]string) ([]*contracts.Release, error)
	GetPipelineReleasesCount(string, string, string, map[string][]string) (int, error)
	GetPipelineRelease(string, string, string, int) (*contracts.Release, error)
	GetPipelineLastReleasesByName(string, string, string, string, []string) ([]contracts.Release, error)
	GetPipelineReleaseLogs(string, string, string, int) (*contracts.ReleaseLog, error)
	GetBuildsCount(map[string][]string) (int, error)
	GetReleasesCount(map[string][]string) (int, error)
	GetBuildsDuration(map[string][]string) (time.Duration, error)
	GetFirstBuildTimes() ([]time.Time, error)
	GetFirstReleaseTimes() ([]time.Time, error)

	selectBuildsQuery() sq.SelectBuilder
	selectPipelinesQuery() sq.SelectBuilder
	selectReleasesQuery() sq.SelectBuilder
}

type cockroachDBClientImpl struct {
	databaseDriver                  string
	config                          config.DatabaseConfig
	PrometheusOutboundAPICallTotals *prometheus.CounterVec
	databaseConnection              *sql.DB
}

// NewCockroachDBClient returns a new cockroach.DBClient
func NewCockroachDBClient(config config.DatabaseConfig, prometheusOutboundAPICallTotals *prometheus.CounterVec) (cockroachDBClient DBClient) {

	cockroachDBClient = &cockroachDBClientImpl{
		databaseDriver:                  "postgres",
		config:                          config,
		PrometheusOutboundAPICallTotals: prometheusOutboundAPICallTotals,
	}

	return
}

// Connect sets up a connection with CockroachDB
func (dbc *cockroachDBClientImpl) Connect() (err error) {

	log.Debug().Msgf("Connecting to database %v on host %v...", dbc.config.DatabaseName, dbc.config.Host)

	sslMode := ""
	if dbc.config.Insecure {
		sslMode = "?sslmode=disable"
	}

	dataSourceName := fmt.Sprintf("postgresql://%v:%v@%v:%v/%v%v", dbc.config.User, dbc.config.Password, dbc.config.Host, dbc.config.Port, dbc.config.DatabaseName, sslMode)

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

// GetAutoIncrement returns the autoincrement number for a pipeline
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
			build_versions a
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

func (dbc *cockroachDBClientImpl) InsertBuild(build contracts.Build) (insertedBuild contracts.Build, err error) {
	dbc.PrometheusOutboundAPICallTotals.With(prometheus.Labels{"target": "cockroachdb"}).Inc()

	sort.Slice(build.Labels, func(i, j int) bool {
		return build.Labels[i].Key < build.Labels[j].Key
	})

	labelsBytes, err := json.Marshal(build.Labels)
	if err != nil {
		return
	}
	releaseTargetsBytes, err := json.Marshal(build.ReleaseTargets)
	if err != nil {
		return
	}
	commitsBytes, err := json.Marshal(build.Commits)
	if err != nil {
		return
	}

	// insert logs
	row := dbc.databaseConnection.QueryRow(
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
			release_targets,
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
			$10,
			$11
		)
		RETURNING
			id
		`,
		build.RepoSource,
		build.RepoOwner,
		build.RepoName,
		build.RepoBranch,
		build.RepoRevision,
		build.BuildVersion,
		build.BuildStatus,
		labelsBytes,
		releaseTargetsBytes,
		build.Manifest,
		commitsBytes,
	)

	insertedBuild = build

	if err = row.Scan(&insertedBuild.ID); err != nil {
		return
	}

	// update computed_pipeline table
	go dbc.UpsertComputedPipeline(dbc.mapBuildToPipeline(&insertedBuild))

	return
}

func (dbc *cockroachDBClientImpl) UpdateBuildStatus(repoSource, repoOwner, repoName string, buildID int, buildStatus string) (err error) {

	dbc.PrometheusOutboundAPICallTotals.With(prometheus.Labels{"target": "cockroachdb"}).Inc()

	// update build status
	_, err = dbc.databaseConnection.Exec(
		`
	UPDATE
		builds
	SET
		build_status=$1,
		updated_at=now(),
		duration=age(now(), inserted_at)
	WHERE
		id=$5 AND
		repo_source=$2 AND
		repo_owner=$3 AND
		repo_name=$4
	`,
		buildStatus,
		repoSource,
		repoOwner,
		repoName,
		buildID,
	)
	if err != nil {
		return
	}

	go dbc.UpsertComputedPipelineByRepo(repoSource, repoOwner, repoName)

	return
}

func (dbc *cockroachDBClientImpl) InsertRelease(release contracts.Release) (insertedRelease contracts.Release, err error) {
	dbc.PrometheusOutboundAPICallTotals.With(prometheus.Labels{"target": "cockroachdb"}).Inc()

	// insert logs
	rows, err := dbc.databaseConnection.Query(
		`
		INSERT INTO
			releases
		(
			repo_source,
			repo_owner,
			repo_name,
			release,
			release_action,
			release_version,
			release_status,
			triggered_by
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
			$8
		)
		RETURNING 
			id
		`,
		release.RepoSource,
		release.RepoOwner,
		release.RepoName,
		release.Name,
		release.Action,
		release.ReleaseVersion,
		release.ReleaseStatus,
		release.TriggeredBy,
	)

	if err != nil {
		return insertedRelease, err
	}

	defer rows.Close()
	recordExists := rows.Next()

	if !recordExists {
		return
	}

	insertedRelease = release

	if err = rows.Scan(&insertedRelease.ID); err != nil {
		return
	}

	go dbc.UpsertComputedPipelineByRepo(insertedRelease.RepoSource, insertedRelease.RepoOwner, insertedRelease.RepoName)

	return
}

func (dbc *cockroachDBClientImpl) UpdateReleaseStatus(repoSource, repoOwner, repoName string, id int, releaseStatus string) (err error) {

	dbc.PrometheusOutboundAPICallTotals.With(prometheus.Labels{"target": "cockroachdb"}).Inc()

	// insert logs
	_, err = dbc.databaseConnection.Exec(
		`
		UPDATE
			releases
		SET
			release_status=$1,
			updated_at=now(),
			duration=age(now(), inserted_at)
		WHERE
			id=$2 AND
			repo_source=$3 AND
			repo_owner=$4 AND
			repo_name=$5
		`,
		releaseStatus,
		id,
		repoSource,
		repoOwner,
		repoName,
	)

	if err != nil {
		return
	}

	go dbc.UpsertComputedPipelineByRepo(repoSource, repoOwner, repoName)

	return
}

func (dbc *cockroachDBClientImpl) InsertBuildLog(buildLog contracts.BuildLog) (err error) {

	dbc.PrometheusOutboundAPICallTotals.With(prometheus.Labels{"target": "cockroachdb"}).Inc()

	bytes, err := json.Marshal(buildLog.Steps)
	if err != nil {
		return
	}

	buildID, err := strconv.Atoi(buildLog.BuildID)
	if err != nil {
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
			build_id,
			steps
		)
		VALUES
		(
			$1,
			$2,
			$3,
			$4,
			$5,
			$6,
			$7
		)
		`,
		buildLog.RepoSource,
		buildLog.RepoOwner,
		buildLog.RepoName,
		buildLog.RepoBranch,
		buildLog.RepoRevision,
		buildID,
		bytes,
	)

	if err != nil {
		return
	}

	return
}

func (dbc *cockroachDBClientImpl) InsertReleaseLog(releaseLog contracts.ReleaseLog) (err error) {

	dbc.PrometheusOutboundAPICallTotals.With(prometheus.Labels{"target": "cockroachdb"}).Inc()

	bytes, err := json.Marshal(releaseLog.Steps)
	if err != nil {
		return
	}

	releaseID, err := strconv.Atoi(releaseLog.ReleaseID)
	if err != nil {
		return err
	}

	// insert logs
	_, err = dbc.databaseConnection.Exec(
		`
		INSERT INTO
			release_logs
		(
			repo_source,
			repo_owner,
			repo_name,
			release_id,
			steps
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
		releaseLog.RepoSource,
		releaseLog.RepoOwner,
		releaseLog.RepoName,
		releaseID,
		bytes,
	)

	if err != nil {
		return
	}

	return
}

func (dbc *cockroachDBClientImpl) UpsertComputedPipelineByRepo(repoSource, repoOwner, repoName string) (upsertedPipeline *contracts.Pipeline, err error) {

	// get computed pipeline
	lastBuild, err := dbc.GetLastPipelineBuild(repoSource, repoOwner, repoName)
	if err != nil {
		log.Error().Err(err).Msgf("Failed getting last build for upserting computed pipeline %v/%v/%v", repoSource, repoOwner, repoName)
		return
	}
	if lastBuild == nil {
		log.Error().Msgf("Failed getting last build for upserting computed pipeline %v/%v/%v", repoSource, repoOwner, repoName)
		return
	}

	// get build for release
	return dbc.UpsertComputedPipeline(dbc.mapBuildToPipeline(lastBuild))
}

func (dbc *cockroachDBClientImpl) UpsertComputedPipeline(pipeline *contracts.Pipeline) (upsertedPipeline *contracts.Pipeline, err error) {
	dbc.PrometheusOutboundAPICallTotals.With(prometheus.Labels{"target": "cockroachdb"}).Inc()

	upsertedPipeline = pipeline

	dbc.enrichPipeline(upsertedPipeline)

	sort.Slice(upsertedPipeline.Labels, func(i, j int) bool {
		return upsertedPipeline.Labels[i].Key < upsertedPipeline.Labels[j].Key
	})

	labelsBytes, err := json.Marshal(upsertedPipeline.Labels)
	if err != nil {
		log.Error().Err(err).Msgf("Failed upserting computed pipeline %v/%v/%v", pipeline.RepoSource, pipeline.RepoOwner, pipeline.RepoName)
		return
	}
	releaseTargetsBytes, err := json.Marshal(upsertedPipeline.ReleaseTargets)
	if err != nil {
		log.Error().Err(err).Msgf("Failed upserting computed pipeline %v/%v/%v", pipeline.RepoSource, pipeline.RepoOwner, pipeline.RepoName)
		return
	}
	commitsBytes, err := json.Marshal(upsertedPipeline.Commits)
	if err != nil {
		log.Error().Err(err).Msgf("Failed upserting computed pipeline %v/%v/%v", pipeline.RepoSource, pipeline.RepoOwner, pipeline.RepoName)
		return
	}

	// upsert computed pipeline
	_, err = dbc.databaseConnection.Exec(
		`
		INSERT INTO
			computed_pipelines
		(
			pipeline_id,
			repo_source,
			repo_owner,
			repo_name,
			repo_branch,
			repo_revision,
			build_version,
			build_status,
			labels,
			release_targets,
			manifest,
			commits,
			inserted_at,
			updated_at,
			duration
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
			$10,
			$11,
			$12,
			$13,
			$14,
			$15
		)
		ON CONFLICT
		(
			repo_source,
			repo_owner,
			repo_name
		)
		DO UPDATE SET
			pipeline_id = excluded.pipeline_id,
			repo_branch = excluded.repo_branch,
			repo_revision = excluded.repo_revision,
			build_version = excluded.build_version,
			build_status = excluded.build_status,
			labels = excluded.labels,
			release_targets = excluded.release_targets,
			manifest = excluded.manifest,
			commits = excluded.commits,
			inserted_at = excluded.inserted_at,
			updated_at = excluded.updated_at,
			duration = excluded.duration
		`,
		upsertedPipeline.ID,
		upsertedPipeline.RepoSource,
		upsertedPipeline.RepoOwner,
		upsertedPipeline.RepoName,
		upsertedPipeline.RepoBranch,
		upsertedPipeline.RepoRevision,
		upsertedPipeline.BuildVersion,
		upsertedPipeline.BuildStatus,
		labelsBytes,
		releaseTargetsBytes,
		upsertedPipeline.Manifest,
		commitsBytes,
		upsertedPipeline.InsertedAt,
		upsertedPipeline.UpdatedAt,
		upsertedPipeline.Duration,
	)
	if err != nil {
		log.Error().Err(err).Msgf("Failed upserting computed pipeline %v/%v/%v", pipeline.RepoSource, pipeline.RepoOwner, pipeline.RepoName)
		return
	}

	return
}

func (dbc *cockroachDBClientImpl) GetPipelines(pageNumber, pageSize int, filters map[string][]string) (pipelines []*contracts.Pipeline, err error) {

	dbc.PrometheusOutboundAPICallTotals.With(prometheus.Labels{"target": "cockroachdb"}).Inc()

	// generate query
	query := dbc.selectPipelinesQuery().
		LeftJoin("builds b ON a.repo_source=b.repo_source AND a.repo_owner=b.repo_owner AND a.repo_name=b.repo_name AND a.inserted_at < b.inserted_at").
		Where("b.id IS NULL").
		OrderBy("a.repo_source,a.repo_owner,a.repo_name").
		Limit(uint64(pageSize)).
		Offset(uint64((pageNumber - 1) * pageSize))

	// dynamically set where clauses for filtering
	query, err = whereClauseGeneratorForAllFilters(query, "a", filters)
	if err != nil {
		return
	}

	// execute query
	rows, err := query.RunWith(dbc.databaseConnection).Query()
	if err != nil {
		return
	}

	// read rows
	if pipelines, err = dbc.scanPipelines(rows); err != nil {
		return
	}

	return
}

func (dbc *cockroachDBClientImpl) GetPipelinesByRepoName(repoName string) (pipelines []*contracts.Pipeline, err error) {

	dbc.PrometheusOutboundAPICallTotals.With(prometheus.Labels{"target": "cockroachdb"}).Inc()

	// generate query
	query := dbc.selectPipelinesQuery().
		LeftJoin("builds b ON a.repo_source=b.repo_source AND a.repo_owner=b.repo_owner AND a.repo_name=b.repo_name AND a.inserted_at < b.inserted_at").
		Where("b.id IS NULL").
		Where(sq.Eq{"a.repo_name": repoName})

	// execute query
	rows, err := query.RunWith(dbc.databaseConnection).Query()
	if err != nil {
		return
	}

	// read rows
	if pipelines, err = dbc.scanPipelines(rows); err != nil {
		return
	}

	return
}

func (dbc *cockroachDBClientImpl) GetPipelinesCount(filters map[string][]string) (totalCount int, err error) {

	dbc.PrometheusOutboundAPICallTotals.With(prometheus.Labels{"target": "cockroachdb"}).Inc()

	// generate query
	query :=
		sq.StatementBuilder.PlaceholderFormat(sq.Dollar).
			Select("COUNT(a.id)").
			From("builds a").
			LeftJoin("builds b ON a.repo_source=b.repo_source AND a.repo_owner=b.repo_owner AND a.repo_name=b.repo_name AND a.inserted_at < b.inserted_at").
			Where("b.id IS NULL")

	// dynamically set where clauses for filtering
	query, err = whereClauseGeneratorForAllFilters(query, "a", filters)
	if err != nil {
		return
	}

	// execute query
	row := query.RunWith(dbc.databaseConnection).QueryRow()
	if err = row.Scan(&totalCount); err != nil {
		return
	}

	return
}

func (dbc *cockroachDBClientImpl) GetPipeline(repoSource, repoOwner, repoName string) (pipeline *contracts.Pipeline, err error) {

	dbc.PrometheusOutboundAPICallTotals.With(prometheus.Labels{"target": "cockroachdb"}).Inc()

	// generate query
	query := dbc.selectPipelinesQuery().
		Where(sq.Eq{"a.repo_source": repoSource}).
		Where(sq.Eq{"a.repo_owner": repoOwner}).
		Where(sq.Eq{"a.repo_name": repoName}).
		OrderBy("a.inserted_at DESC").
		Limit(uint64(1))

	// execute query
	row := query.RunWith(dbc.databaseConnection).QueryRow()
	if pipeline, err = dbc.scanPipeline(row); err != nil {
		return
	}

	return
}

func (dbc *cockroachDBClientImpl) GetPipelineBuilds(repoSource, repoOwner, repoName string, pageNumber, pageSize int, filters map[string][]string) (builds []*contracts.Build, err error) {

	dbc.PrometheusOutboundAPICallTotals.With(prometheus.Labels{"target": "cockroachdb"}).Inc()

	// generate query
	query := dbc.selectBuildsQuery().
		Where(sq.Eq{"a.repo_source": repoSource}).
		Where(sq.Eq{"a.repo_owner": repoOwner}).
		Where(sq.Eq{"a.repo_name": repoName}).
		OrderBy("a.inserted_at DESC").
		Limit(uint64(pageSize)).
		Offset(uint64((pageNumber - 1) * pageSize))

	// dynamically set where clauses for filtering
	query, err = whereClauseGeneratorForAllFilters(query, "a", filters)
	if err != nil {
		return
	}

	// execute query
	rows, err := query.RunWith(dbc.databaseConnection).Query()
	if err != nil {
		return
	}

	// read rows
	if builds, err = dbc.scanBuilds(rows); err != nil {
		return
	}

	return
}

func (dbc *cockroachDBClientImpl) GetPipelineBuildsCount(repoSource, repoOwner, repoName string, filters map[string][]string) (totalCount int, err error) {

	dbc.PrometheusOutboundAPICallTotals.With(prometheus.Labels{"target": "cockroachdb"}).Inc()

	// generate query
	query :=
		sq.StatementBuilder.PlaceholderFormat(sq.Dollar).
			Select("COUNT(*)").
			From("builds a").
			Where(sq.Eq{"a.repo_source": repoSource}).
			Where(sq.Eq{"a.repo_owner": repoOwner}).
			Where(sq.Eq{"a.repo_name": repoName})

	// dynamically set where clauses for filtering
	query, err = whereClauseGeneratorForAllFilters(query, "a", filters)
	if err != nil {
		return
	}

	// execute query
	row := query.RunWith(dbc.databaseConnection).QueryRow()
	if err = row.Scan(&totalCount); err != nil {
		return
	}

	return
}

func (dbc *cockroachDBClientImpl) GetPipelineBuild(repoSource, repoOwner, repoName, repoRevision string) (build *contracts.Build, err error) {

	dbc.PrometheusOutboundAPICallTotals.With(prometheus.Labels{"target": "cockroachdb"}).Inc()

	// generate query
	query := dbc.selectBuildsQuery().
		Where(sq.Eq{"a.repo_source": repoSource}).
		Where(sq.Eq{"a.repo_owner": repoOwner}).
		Where(sq.Eq{"a.repo_name": repoName}).
		Where(sq.Eq{"a.repo_revision": repoRevision}).
		OrderBy("a.inserted_at DESC").
		Limit(uint64(1))

	// execute query
	row := query.RunWith(dbc.databaseConnection).QueryRow()
	if build, err = dbc.scanBuild(row); err != nil {
		return
	}

	return
}

func (dbc *cockroachDBClientImpl) GetPipelineBuildByID(repoSource, repoOwner, repoName string, id int) (build *contracts.Build, err error) {

	dbc.PrometheusOutboundAPICallTotals.With(prometheus.Labels{"target": "cockroachdb"}).Inc()

	// generate query
	query := dbc.selectBuildsQuery().
		Where(sq.Eq{"a.id": id}).
		Where(sq.Eq{"a.repo_source": repoSource}).
		Where(sq.Eq{"a.repo_owner": repoOwner}).
		Where(sq.Eq{"a.repo_name": repoName}).
		OrderBy("a.inserted_at DESC").
		Limit(uint64(1))

	// execute query
	row := query.RunWith(dbc.databaseConnection).QueryRow()
	if build, err = dbc.scanBuild(row); err != nil {
		return
	}

	return
}

func (dbc *cockroachDBClientImpl) GetLastPipelineBuild(repoSource, repoOwner, repoName string) (build *contracts.Build, err error) {

	dbc.PrometheusOutboundAPICallTotals.With(prometheus.Labels{"target": "cockroachdb"}).Inc()

	// generate query
	query := dbc.selectBuildsQuery().
		Where(sq.Eq{"a.repo_source": repoSource}).
		Where(sq.Eq{"a.repo_owner": repoOwner}).
		Where(sq.Eq{"a.repo_name": repoName}).
		OrderBy("a.inserted_at DESC").
		Limit(uint64(1))

	// execute query
	row := query.RunWith(dbc.databaseConnection).QueryRow()
	if build, err = dbc.scanBuild(row); err != nil {
		return
	}

	return
}

func (dbc *cockroachDBClientImpl) GetPipelineBuildsByVersion(repoSource, repoOwner, repoName, buildVersion string) (builds []*contracts.Build, err error) {
	dbc.PrometheusOutboundAPICallTotals.With(prometheus.Labels{"target": "cockroachdb"}).Inc()

	// generate query
	query := dbc.selectBuildsQuery().
		Where(sq.Eq{"a.repo_source": repoSource}).
		Where(sq.Eq{"a.repo_owner": repoOwner}).
		Where(sq.Eq{"a.repo_name": repoName}).
		Where(sq.Eq{"a.build_version": buildVersion}).
		OrderBy("a.inserted_at DESC")

	// execute query
	rows, err := query.RunWith(dbc.databaseConnection).Query()
	if err != nil {
		return
	}

	// read rows
	if builds, err = dbc.scanBuilds(rows); err != nil {
		return
	}

	return
}

func (dbc *cockroachDBClientImpl) GetPipelineBuildLogs(repoSource, repoOwner, repoName, repoBranch, repoRevision, buildID string) (buildLog *contracts.BuildLog, err error) {

	dbc.PrometheusOutboundAPICallTotals.With(prometheus.Labels{"target": "cockroachdb"}).Inc()

	// generate query
	query := dbc.selectBuildLogsQuery().
		Where(sq.Eq{"a.repo_source": repoSource}).
		Where(sq.Eq{"a.repo_owner": repoOwner}).
		Where(sq.Eq{"a.repo_name": repoName}).
		Where(sq.Eq{"a.repo_branch": repoBranch}).
		Where(sq.Eq{"a.repo_revision": repoRevision}).
		OrderBy("a.inserted_at DESC").
		Limit(uint64(1))

	buildLog = &contracts.BuildLog{}
	var stepsData []uint8
	var rowBuildID sql.NullInt64

	// execute query
	row := query.RunWith(dbc.databaseConnection).QueryRow()
	if err = row.Scan(&buildLog.ID,
		&buildLog.RepoSource,
		&buildLog.RepoOwner,
		&buildLog.RepoName,
		&buildLog.RepoBranch,
		&buildLog.RepoRevision,
		&rowBuildID,
		&stepsData,
		&buildLog.InsertedAt); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return
	}

	if err = json.Unmarshal(stepsData, &buildLog.Steps); err != nil {
		return
	}

	if rowBuildID.Valid {
		buildLog.BuildID = strconv.FormatInt(rowBuildID.Int64, 10)

		// if theses logs have been stored with build_id it could be a rebuild version with multiple logs, so match the supplied build id
		if buildLog.BuildID == buildID {
			return
		}

		// otherwise reset to make sure we don't return the wrong logs if this is still a running build?
		// buildLog = &contracts.BuildLog{}
	}

	return
}

func (dbc *cockroachDBClientImpl) GetPipelineReleases(repoSource, repoOwner, repoName string, pageNumber, pageSize int, filters map[string][]string) (releases []*contracts.Release, err error) {

	dbc.PrometheusOutboundAPICallTotals.With(prometheus.Labels{"target": "cockroachdb"}).Inc()

	// generate query
	query := dbc.selectReleasesQuery().
		Where(sq.Eq{"a.repo_source": repoSource}).
		Where(sq.Eq{"a.repo_owner": repoOwner}).
		Where(sq.Eq{"a.repo_name": repoName}).
		OrderBy("a.inserted_at DESC").
		Limit(uint64(pageSize)).
		Offset(uint64((pageNumber - 1) * pageSize))

	// dynamically set where clauses for filtering
	query, err = whereClauseGeneratorForAllReleaseFilters(query, "a", filters)
	if err != nil {
		return
	}

	// execute query
	rows, err := query.RunWith(dbc.databaseConnection).Query()
	if err != nil {
		return
	}

	// read rows
	if releases, err = dbc.scanReleases(rows); err != nil {
		return
	}

	return
}

func (dbc *cockroachDBClientImpl) GetPipelineReleasesCount(repoSource, repoOwner, repoName string, filters map[string][]string) (totalCount int, err error) {

	dbc.PrometheusOutboundAPICallTotals.With(prometheus.Labels{"target": "cockroachdb"}).Inc()

	// generate query
	query :=
		sq.StatementBuilder.PlaceholderFormat(sq.Dollar).
			Select("COUNT(*)").
			From("releases a").
			Where(sq.Eq{"a.repo_source": repoSource}).
			Where(sq.Eq{"a.repo_owner": repoOwner}).
			Where(sq.Eq{"a.repo_name": repoName})

	// dynamically set where clauses for filtering
	query, err = whereClauseGeneratorForAllReleaseFilters(query, "a", filters)
	if err != nil {
		return
	}

	// execute query
	row := query.RunWith(dbc.databaseConnection).QueryRow()
	if err = row.Scan(&totalCount); err != nil {
		return
	}

	return
}

func (dbc *cockroachDBClientImpl) GetPipelineRelease(repoSource, repoOwner, repoName string, id int) (release *contracts.Release, err error) {

	dbc.PrometheusOutboundAPICallTotals.With(prometheus.Labels{"target": "cockroachdb"}).Inc()

	// generate query
	query := dbc.selectReleasesQuery().
		Where(sq.Eq{"a.id": id}).
		Where(sq.Eq{"a.repo_source": repoSource}).
		Where(sq.Eq{"a.repo_owner": repoOwner}).
		Where(sq.Eq{"a.repo_name": repoName}).
		OrderBy("a.inserted_at DESC").
		Limit(uint64(1))

	// execute query
	row := query.RunWith(dbc.databaseConnection).QueryRow()
	if release, err = dbc.scanRelease(row); err != nil {
		return
	}

	return
}

func (dbc *cockroachDBClientImpl) GetPipelineLastReleasesByName(repoSource, repoOwner, repoName, releaseName string, actions []string) (releases []contracts.Release, err error) {

	dbc.PrometheusOutboundAPICallTotals.With(prometheus.Labels{"target": "cockroachdb"}).Inc()

	// generate query
	query := dbc.selectReleasesQuery().
		LeftJoin("releases b on a.release=b.release AND a.repo_source=b.repo_source AND a.repo_owner=b.repo_owner AND a.repo_name=b.repo_name AND a.release_action=b.release_action AND a.inserted_at < b.inserted_at").
		Where(sq.Eq{"a.release": releaseName}).
		Where(sq.Eq{"a.repo_source": repoSource}).
		Where(sq.Eq{"a.repo_owner": repoOwner}).
		Where(sq.Eq{"a.repo_name": repoName}).
		Where("b.id IS NULL").
		OrderBy("a.inserted_at DESC")

	if len(actions) > 0 {
		query = query.Where(sq.Eq{"a.release_action": actions})
	} else {
		query = query.Where(sq.Eq{"a.release_action": ""})
	}

	// execute query
	rows, err := query.RunWith(dbc.databaseConnection).Query()
	if err != nil {
		return
	}

	// read rows
	releasesPointers := []*contracts.Release{}
	if releasesPointers, err = dbc.scanReleases(rows); err != nil {
		return releases, err
	}

	// copy pointer values
	releases = make([]contracts.Release, 0)
	for _, r := range releasesPointers {
		releases = append(releases, *r)
	}

	return
}

func (dbc *cockroachDBClientImpl) GetPipelineReleaseLogs(repoSource, repoOwner, repoName string, id int) (releaseLog *contracts.ReleaseLog, err error) {

	dbc.PrometheusOutboundAPICallTotals.With(prometheus.Labels{"target": "cockroachdb"}).Inc()

	// generate query
	query := dbc.selectReleaseLogsQuery().
		Where(sq.Eq{"a.repo_source": repoSource}).
		Where(sq.Eq{"a.repo_owner": repoOwner}).
		Where(sq.Eq{"a.repo_name": repoName}).
		Where(sq.Eq{"a.release_id": id}).
		OrderBy("a.inserted_at DESC").
		Limit(uint64(1))

	releaseLog = &contracts.ReleaseLog{}

	var stepsData []uint8
	var releaseID int

	// execute query
	row := query.RunWith(dbc.databaseConnection).QueryRow()
	if err = row.Scan(&releaseLog.ID,
		&releaseLog.RepoSource,
		&releaseLog.RepoOwner,
		&releaseLog.RepoName,
		&releaseID,
		&stepsData,
		&releaseLog.InsertedAt); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return
	}

	releaseLog.ReleaseID = strconv.Itoa(releaseID)

	if err = json.Unmarshal(stepsData, &releaseLog.Steps); err != nil {
		return
	}

	return
}

func (dbc *cockroachDBClientImpl) GetBuildsCount(filters map[string][]string) (totalCount int, err error) {

	dbc.PrometheusOutboundAPICallTotals.With(prometheus.Labels{"target": "cockroachdb"}).Inc()

	// generate query
	query :=
		sq.StatementBuilder.PlaceholderFormat(sq.Dollar).
			Select("COUNT(*)").
			From("builds a")

	// dynamically set where clauses for filtering
	query, err = whereClauseGeneratorForAllFilters(query, "a", filters)
	if err != nil {
		return
	}

	// execute query
	row := query.RunWith(dbc.databaseConnection).QueryRow()
	if err = row.Scan(&totalCount); err != nil {
		return
	}

	return
}

func (dbc *cockroachDBClientImpl) GetReleasesCount(filters map[string][]string) (totalCount int, err error) {

	dbc.PrometheusOutboundAPICallTotals.With(prometheus.Labels{"target": "cockroachdb"}).Inc()

	// generate query
	query :=
		sq.StatementBuilder.PlaceholderFormat(sq.Dollar).
			Select("COUNT(*)").
			From("releases a")

	// dynamically set where clauses for filtering
	query, err = whereClauseGeneratorForAllReleaseFilters(query, "a", filters)
	if err != nil {
		return
	}

	// execute query
	row := query.RunWith(dbc.databaseConnection).QueryRow()
	if err = row.Scan(&totalCount); err != nil {
		return
	}

	return
}

func (dbc *cockroachDBClientImpl) GetBuildsDuration(filters map[string][]string) (totalDuration time.Duration, err error) {

	dbc.PrometheusOutboundAPICallTotals.With(prometheus.Labels{"target": "cockroachdb"}).Inc()

	// generate query
	query :=
		sq.StatementBuilder.PlaceholderFormat(sq.Dollar).
			Select("SUM(AGE(updated_at,inserted_at))::string").
			From("builds a")

	// dynamically set where clauses for filtering
	query, err = whereClauseGeneratorForAllFilters(query, "a", filters)
	if err != nil {
		return
	}

	// execute query
	row := query.RunWith(dbc.databaseConnection).QueryRow()

	var totalDurationAsString string

	if err = row.Scan(&totalDurationAsString); err != nil {
		return
	}

	totalDuration, err = time.ParseDuration(totalDurationAsString)
	if err != nil {
		return
	}

	return
}

func (dbc *cockroachDBClientImpl) GetFirstBuildTimes() (buildTimes []time.Time, err error) {

	dbc.PrometheusOutboundAPICallTotals.With(prometheus.Labels{"target": "cockroachdb"}).Inc()

	// generate query
	query :=
		sq.StatementBuilder.PlaceholderFormat(sq.Dollar).
			Select("a.inserted_at").
			From("builds a").
			LeftJoin("builds b ON a.repo_source=b.repo_source AND a.repo_owner=b.repo_owner AND a.repo_name=b.repo_name AND a.inserted_at > b.inserted_at").
			Where("b.id IS NULL").
			OrderBy("a.inserted_at")

	buildTimes = make([]time.Time, 0)

	// execute query
	rows, err := query.RunWith(dbc.databaseConnection).Query()

	if err != nil {
		return
	}

	defer rows.Close()
	for rows.Next() {

		insertedAt := time.Time{}

		if err = rows.Scan(
			&insertedAt); err != nil {
			return
		}

		buildTimes = append(buildTimes, insertedAt)
	}

	return
}

func (dbc *cockroachDBClientImpl) GetFirstReleaseTimes() (releaseTimes []time.Time, err error) {

	dbc.PrometheusOutboundAPICallTotals.With(prometheus.Labels{"target": "cockroachdb"}).Inc()

	// generate query
	query :=
		sq.StatementBuilder.PlaceholderFormat(sq.Dollar).
			Select("a.inserted_at").
			From("releases a").
			LeftJoin("releases b ON a.repo_source=b.repo_source AND a.repo_owner=b.repo_owner AND a.repo_name=b.repo_name AND a.inserted_at > b.inserted_at").
			Where("b.id IS NULL").
			OrderBy("a.inserted_at")

	releaseTimes = make([]time.Time, 0)

	rows, err := query.RunWith(dbc.databaseConnection).Query()

	if err != nil {
		return
	}

	defer rows.Close()
	for rows.Next() {

		insertedAt := time.Time{}

		if err = rows.Scan(
			&insertedAt); err != nil {
			return
		}

		releaseTimes = append(releaseTimes, insertedAt)
	}

	return
}

func whereClauseGeneratorForAllFilters(query sq.SelectBuilder, alias string, filters map[string][]string) (sq.SelectBuilder, error) {

	query, err := whereClauseGeneratorForSinceFilter(query, alias, filters)
	if err != nil {
		return query, err
	}
	query, err = whereClauseGeneratorForStatusFilter(query, alias, filters)
	if err != nil {
		return query, err
	}
	query, err = whereClauseGeneratorForLabelsFilter(query, alias, filters)
	if err != nil {
		return query, err
	}

	return query, nil
}

func whereClauseGeneratorForAllReleaseFilters(query sq.SelectBuilder, alias string, filters map[string][]string) (sq.SelectBuilder, error) {

	query, err := whereClauseGeneratorForReleaseStatusFilter(query, alias, filters)
	if err != nil {
		return query, err
	}
	query, err = whereClauseGeneratorForSinceFilter(query, alias, filters)
	if err != nil {
		return query, err
	}

	return query, nil
}

func whereClauseGeneratorForSinceFilter(query sq.SelectBuilder, alias string, filters map[string][]string) (sq.SelectBuilder, error) {

	if since, ok := filters["since"]; ok && len(since) > 0 && since[0] != "eternity" {
		sinceValue := since[0]
		switch sinceValue {
		case "1d":
			query = query.Where(sq.GtOrEq{fmt.Sprintf("%v.inserted_at", alias): time.Now().AddDate(0, 0, -1)})
		case "1w":
			query = query.Where(sq.GtOrEq{fmt.Sprintf("%v.inserted_at", alias): time.Now().AddDate(0, 0, -7)})
		case "1m":
			query = query.Where(sq.GtOrEq{fmt.Sprintf("%v.inserted_at", alias): time.Now().AddDate(0, -1, 0)})
		case "1y":
			query = query.Where(sq.GtOrEq{fmt.Sprintf("%v.inserted_at", alias): time.Now().AddDate(-1, 0, 0)})
		}
	}

	return query, nil
}

func whereClauseGeneratorForStatusFilter(query sq.SelectBuilder, alias string, filters map[string][]string) (sq.SelectBuilder, error) {

	if statuses, ok := filters["status"]; ok && len(statuses) > 0 && statuses[0] != "all" {
		query = query.Where(sq.Eq{fmt.Sprintf("%v.build_status", alias): statuses})
	}

	return query, nil
}

func whereClauseGeneratorForReleaseStatusFilter(query sq.SelectBuilder, alias string, filters map[string][]string) (sq.SelectBuilder, error) {

	if statuses, ok := filters["status"]; ok && len(statuses) > 0 && statuses[0] != "all" {
		query = query.Where(sq.Eq{fmt.Sprintf("%v.release_status", alias): statuses})
	}

	return query, nil
}

func whereClauseGeneratorForLabelsFilter(query sq.SelectBuilder, alias string, filters map[string][]string) (sq.SelectBuilder, error) {

	if labels, ok := filters["labels"]; ok && len(labels) > 0 {

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

			query = query.Where(fmt.Sprintf("%v.labels @> ?", alias), string(bytes))
		}
	}

	return query, nil
}

func (dbc *cockroachDBClientImpl) scanBuild(row sq.RowScanner) (build *contracts.Build, err error) {

	build = &contracts.Build{}
	var labelsData, releaseTargetsData, commitsData []uint8
	var seconds int

	if err = row.Scan(
		&build.ID,
		&build.RepoSource,
		&build.RepoOwner,
		&build.RepoName,
		&build.RepoBranch,
		&build.RepoRevision,
		&build.BuildVersion,
		&build.BuildStatus,
		&labelsData,
		&releaseTargetsData,
		&build.Manifest,
		&commitsData,
		&build.InsertedAt,
		&build.UpdatedAt,
		&seconds); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return
	}

	build.Duration = time.Duration(seconds) * time.Second

	dbc.setBuildPropertiesFromJSONB(build, labelsData, releaseTargetsData, commitsData)

	// clear some properties for reduced size and improved performance over the network
	build.Manifest = ""
	build.ManifestWithDefaults = ""

	return
}

func (dbc *cockroachDBClientImpl) scanBuilds(rows *sql.Rows) (builds []*contracts.Build, err error) {

	builds = make([]*contracts.Build, 0)

	defer rows.Close()
	for rows.Next() {

		build := contracts.Build{}
		var labelsData, releaseTargetsData, commitsData []uint8
		var seconds int

		if err = rows.Scan(
			&build.ID,
			&build.RepoSource,
			&build.RepoOwner,
			&build.RepoName,
			&build.RepoBranch,
			&build.RepoRevision,
			&build.BuildVersion,
			&build.BuildStatus,
			&labelsData,
			&releaseTargetsData,
			&build.Manifest,
			&commitsData,
			&build.InsertedAt,
			&build.UpdatedAt,
			&seconds); err != nil {
			return
		}

		build.Duration = time.Duration(seconds) * time.Second

		dbc.setBuildPropertiesFromJSONB(&build, labelsData, releaseTargetsData, commitsData)

		// clear some properties for reduced size and improved performance over the network
		build.Manifest = ""
		build.ManifestWithDefaults = ""

		builds = append(builds, &build)
	}

	dbc.enrichBuilds(builds)

	return
}

func (dbc *cockroachDBClientImpl) scanPipeline(row sq.RowScanner) (pipeline *contracts.Pipeline, err error) {

	pipeline = &contracts.Pipeline{}
	var labelsData, releaseTargetsData, commitsData []uint8
	var seconds int

	if err = row.Scan(
		&pipeline.ID,
		&pipeline.RepoSource,
		&pipeline.RepoOwner,
		&pipeline.RepoName,
		&pipeline.RepoBranch,
		&pipeline.RepoRevision,
		&pipeline.BuildVersion,
		&pipeline.BuildStatus,
		&labelsData,
		&releaseTargetsData,
		&pipeline.Manifest,
		&commitsData,
		&pipeline.InsertedAt,
		&pipeline.UpdatedAt,
		&seconds); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return
	}

	pipeline.Duration = time.Duration(seconds) * time.Second

	dbc.setPipelinePropertiesFromJSONB(pipeline, labelsData, releaseTargetsData, commitsData)

	// set released versions
	dbc.enrichPipeline(pipeline)

	// clear some properties for reduced size and improved performance over the network
	pipeline.Manifest = ""
	pipeline.ManifestWithDefaults = ""

	return
}

func (dbc *cockroachDBClientImpl) scanPipelines(rows *sql.Rows) (pipelines []*contracts.Pipeline, err error) {

	pipelines = make([]*contracts.Pipeline, 0)

	defer rows.Close()
	for rows.Next() {

		pipeline := contracts.Pipeline{}
		var labelsData, releaseTargetsData, commitsData []uint8
		var seconds int

		if err = rows.Scan(
			&pipeline.ID,
			&pipeline.RepoSource,
			&pipeline.RepoOwner,
			&pipeline.RepoName,
			&pipeline.RepoBranch,
			&pipeline.RepoRevision,
			&pipeline.BuildVersion,
			&pipeline.BuildStatus,
			&labelsData,
			&releaseTargetsData,
			&pipeline.Manifest,
			&commitsData,
			&pipeline.InsertedAt,
			&pipeline.UpdatedAt,
			&seconds); err != nil {
			return
		}

		pipeline.Duration = time.Duration(seconds) * time.Second

		dbc.setPipelinePropertiesFromJSONB(&pipeline, labelsData, releaseTargetsData, commitsData)

		// clear some properties for reduced size and improved performance over the network
		pipeline.Manifest = ""
		pipeline.ManifestWithDefaults = ""

		pipelines = append(pipelines, &pipeline)
	}

	dbc.enrichPipelines(pipelines)

	return
}

func (dbc *cockroachDBClientImpl) scanRelease(row sq.RowScanner) (release *contracts.Release, err error) {

	release = &contracts.Release{}
	var seconds int
	var id int

	if err = row.Scan(
		&id,
		&release.RepoSource,
		&release.RepoOwner,
		&release.RepoName,
		&release.Name,
		&release.Action,
		&release.ReleaseVersion,
		&release.ReleaseStatus,
		&release.TriggeredBy,
		&release.InsertedAt,
		&release.UpdatedAt,
		&seconds); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return
	}

	duration := time.Duration(seconds) * time.Second
	release.Duration = &duration
	release.ID = strconv.Itoa(id)

	return
}

func (dbc *cockroachDBClientImpl) scanReleases(rows *sql.Rows) (releases []*contracts.Release, err error) {

	releases = make([]*contracts.Release, 0)

	defer rows.Close()
	for rows.Next() {

		release := contracts.Release{}
		var seconds int
		var id int

		if err = rows.Scan(
			&id,
			&release.RepoSource,
			&release.RepoOwner,
			&release.RepoName,
			&release.Name,
			&release.Action,
			&release.ReleaseVersion,
			&release.ReleaseStatus,
			&release.TriggeredBy,
			&release.InsertedAt,
			&release.UpdatedAt,
			&seconds); err != nil {
			return
		}

		duration := time.Duration(seconds) * time.Second
		release.Duration = &duration
		release.ID = strconv.Itoa(id)

		releases = append(releases, &release)
	}

	return
}

func (dbc *cockroachDBClientImpl) selectBuildsQuery() sq.SelectBuilder {
	psql := sq.StatementBuilder.PlaceholderFormat(sq.Dollar)

	return psql.
		Select("a.id, a.repo_source, a.repo_owner, a.repo_name, a.repo_branch, a.repo_revision, a.build_version, a.build_status, a.labels, a.release_targets, a.manifest, a.commits, a.inserted_at, a.updated_at, a.duration::INT").
		From("builds a")
}

func (dbc *cockroachDBClientImpl) selectPipelinesQuery() sq.SelectBuilder {
	psql := sq.StatementBuilder.PlaceholderFormat(sq.Dollar)

	return psql.
		Select("a.id, a.repo_source, a.repo_owner, a.repo_name, a.repo_branch, a.repo_revision, a.build_version, a.build_status, a.labels, a.release_targets, a.manifest, a.commits, a.inserted_at, a.updated_at, a.duration::INT").
		From("builds a")
}

func (dbc *cockroachDBClientImpl) selectReleasesQuery() sq.SelectBuilder {
	psql := sq.StatementBuilder.PlaceholderFormat(sq.Dollar)

	return psql.
		Select("a.id, a.repo_source, a.repo_owner, a.repo_name, a.release, a.release_action, a.release_version, a.release_status, a.triggered_by, a.inserted_at, a.updated_at, a.duration::INT").
		From("releases a")
}

func (dbc *cockroachDBClientImpl) selectBuildLogsQuery() sq.SelectBuilder {
	psql := sq.StatementBuilder.PlaceholderFormat(sq.Dollar)

	return psql.
		Select("a.id, a.repo_source, a.repo_owner, a.repo_name, a.repo_branch, a.repo_revision, a.build_id, a.steps, a.inserted_at").
		From("build_logs_v2 a")
}

func (dbc *cockroachDBClientImpl) selectReleaseLogsQuery() sq.SelectBuilder {
	psql := sq.StatementBuilder.PlaceholderFormat(sq.Dollar)

	return psql.
		Select("a.id, a.repo_source, a.repo_owner, a.repo_name, a.release_id, a.steps, a.inserted_at").
		From("release_logs a")
}

func (dbc *cockroachDBClientImpl) enrichPipelines(pipelines []*contracts.Pipeline) {
	var wg sync.WaitGroup
	wg.Add(len(pipelines))

	for _, p := range pipelines {
		go func(p *contracts.Pipeline) {
			defer wg.Done()
			dbc.enrichPipeline(p)
		}(p)
	}
	wg.Wait()
}

func (dbc *cockroachDBClientImpl) enrichPipeline(pipeline *contracts.Pipeline) {
	dbc.getLatestReleasesForPipeline(pipeline)
}

func (dbc *cockroachDBClientImpl) enrichBuilds(builds []*contracts.Build) {
	var wg sync.WaitGroup
	wg.Add(len(builds))

	for _, b := range builds {
		go func(b *contracts.Build) {
			defer wg.Done()
			dbc.enrichBuild(b)
		}(b)
	}
	wg.Wait()
}

func (dbc *cockroachDBClientImpl) enrichBuild(build *contracts.Build) {
	dbc.getLatestReleasesForBuild(build)
}

func (dbc *cockroachDBClientImpl) setPipelinePropertiesFromJSONB(pipeline *contracts.Pipeline, labelsData, releaseTargetsData, commitsData []uint8) (err error) {

	if len(labelsData) > 0 {
		if err = json.Unmarshal(labelsData, &pipeline.Labels); err != nil {
			return
		}
	}
	if len(releaseTargetsData) > 0 {
		if err = json.Unmarshal(releaseTargetsData, &pipeline.ReleaseTargets); err != nil {
			return
		}
	}
	if len(commitsData) > 0 {
		if err = json.Unmarshal(commitsData, &pipeline.Commits); err != nil {
			return
		}

		// remove all but the first 6 commits
		if len(pipeline.Commits) > 6 {
			pipeline.Commits = pipeline.Commits[:6]
		}
	}

	// unmarshal then marshal manifest to include defaults
	var manifest manifest.EstafetteManifest
	err = yaml.Unmarshal([]byte(pipeline.Manifest), &manifest)
	if err == nil {
		manifestWithDefaultBytes, err := yaml.Marshal(manifest)
		if err == nil {
			pipeline.ManifestWithDefaults = string(manifestWithDefaultBytes)
		} else {
			log.Warn().Err(err).Interface("manifest", manifest).Msgf("Marshalling manifest for %v/%v/%v revision %v failed", pipeline.RepoSource, pipeline.RepoOwner, pipeline.RepoName, pipeline.RepoRevision)
		}
	} else {
		log.Warn().Err(err).Str("manifest", pipeline.Manifest).Msgf("Unmarshalling manifest for %v/%v/%v revision %v failed", pipeline.RepoSource, pipeline.RepoOwner, pipeline.RepoName, pipeline.RepoRevision)
	}

	return
}

func (dbc *cockroachDBClientImpl) setBuildPropertiesFromJSONB(build *contracts.Build, labelsData, releaseTargetsData, commitsData []uint8) (err error) {

	if len(labelsData) > 0 {
		if err = json.Unmarshal(labelsData, &build.Labels); err != nil {
			return
		}
	}
	if len(releaseTargetsData) > 0 {
		if err = json.Unmarshal(releaseTargetsData, &build.ReleaseTargets); err != nil {
			return
		}
	}
	if len(commitsData) > 0 {
		if err = json.Unmarshal(commitsData, &build.Commits); err != nil {
			return
		}

		// remove all but the first 6 commits
		if len(build.Commits) > 6 {
			build.Commits = build.Commits[:6]
		}
	}

	// unmarshal then marshal manifest to include defaults
	var manifest manifest.EstafetteManifest
	err = yaml.Unmarshal([]byte(build.Manifest), &manifest)
	if err == nil {
		manifestWithDefaultBytes, err := yaml.Marshal(manifest)
		if err == nil {
			build.ManifestWithDefaults = string(manifestWithDefaultBytes)
		} else {
			log.Warn().Err(err).Interface("manifest", manifest).Msgf("Marshalling manifest for %v/%v/%v revision %v failed", build.RepoSource, build.RepoOwner, build.RepoName, build.RepoRevision)
		}
	} else {
		log.Warn().Err(err).Str("manifest", build.Manifest).Msgf("Unmarshalling manifest for %v/%v/%v revision %v failed", build.RepoSource, build.RepoOwner, build.RepoName, build.RepoRevision)
	}

	return
}

func (dbc *cockroachDBClientImpl) getLatestReleasesForPipeline(pipeline *contracts.Pipeline) {
	pipeline.ReleaseTargets = dbc.getLatestReleases(pipeline.ReleaseTargets, pipeline.RepoSource, pipeline.RepoOwner, pipeline.RepoName)
}

func (dbc *cockroachDBClientImpl) getLatestReleasesForBuild(build *contracts.Build) {
	build.ReleaseTargets = dbc.getLatestReleases(build.ReleaseTargets, build.RepoSource, build.RepoOwner, build.RepoName)
}

func (dbc *cockroachDBClientImpl) getLatestReleases(releaseTargets []contracts.ReleaseTarget, repoSource, repoOwner, repoName string) []contracts.ReleaseTarget {

	// todo retrieve all latest releases per action

	// set latest release version per release targets
	updatedReleaseTargets := make([]contracts.ReleaseTarget, 0)
	for _, rt := range releaseTargets {

		actions := getActionNamesFromReleaseTarget(rt)
		latestReleases, err := dbc.GetPipelineLastReleasesByName(repoSource, repoOwner, repoName, rt.Name, actions)
		if err != nil {
			log.Error().Err(err).Msgf("Failed retrieving latest release for %v/%v/%v %v", repoSource, repoOwner, repoName, rt.Name)
		} else {
			rt.ActiveReleases = latestReleases
		}
		updatedReleaseTargets = append(updatedReleaseTargets, rt)
	}

	return updatedReleaseTargets
}

func getActionNamesFromReleaseTarget(releaseTarget contracts.ReleaseTarget) (actions []string) {

	actions = []string{}

	if releaseTarget.Actions != nil && len(releaseTarget.Actions) > 0 {
		for _, a := range releaseTarget.Actions {
			actions = append(actions, a.Name)
		}
	}

	return
}

func (dbc *cockroachDBClientImpl) mapBuildToPipeline(build *contracts.Build) (pipeline *contracts.Pipeline) {
	return &contracts.Pipeline{
		ID:                   build.ID,
		RepoSource:           build.RepoSource,
		RepoOwner:            build.RepoOwner,
		RepoName:             build.RepoName,
		RepoBranch:           build.RepoBranch,
		RepoRevision:         build.RepoRevision,
		BuildVersion:         build.BuildVersion,
		BuildStatus:          build.BuildStatus,
		Labels:               build.Labels,
		ReleaseTargets:       build.ReleaseTargets,
		Manifest:             build.Manifest,
		ManifestWithDefaults: build.ManifestWithDefaults,
		Commits:              build.Commits,
		InsertedAt:           build.InsertedAt,
		UpdatedAt:            build.UpdatedAt,
		Duration:             build.Duration,
	}
}
