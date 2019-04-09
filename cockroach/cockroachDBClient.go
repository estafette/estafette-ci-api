package cockroach

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	sq "github.com/Masterminds/squirrel"
	"github.com/estafette/estafette-ci-api/config"
	contracts "github.com/estafette/estafette-ci-contracts"
	manifest "github.com/estafette/estafette-ci-manifest"
	_ "github.com/lib/pq" // use postgres client library to connect to cockroachdb
	"github.com/prometheus/client_golang/prometheus"
	"github.com/rs/zerolog/log"
	yaml "gopkg.in/yaml.v2"
)

// DBClient is the interface for communicating with CockroachDB
type DBClient interface {
	Connect() error
	ConnectWithDriverAndSource(driverName, dataSourceName string) error
	GetAutoIncrement(shortRepoSource, repoOwner, repoName string) (int, error)

	InsertBuild(contracts.Build) (*contracts.Build, error)
	UpdateBuildStatus(string, string, string, int, string) error
	InsertRelease(contracts.Release) (*contracts.Release, error)
	UpdateReleaseStatus(string, string, string, int, string) error
	InsertBuildLog(contracts.BuildLog) error
	InsertReleaseLog(contracts.ReleaseLog) error

	UpsertComputedPipeline(string, string, string) error
	UpdateComputedPipelineFirstInsertedAt(string, string, string) error
	UpsertComputedRelease(string, string, string, string, string) error
	UpdateComputedReleaseFirstInsertedAt(string, string, string, string, string) error

	GetPipelines(int, int, map[string][]string, bool) ([]*contracts.Pipeline, error)
	GetPipelinesByRepoName(string, bool) ([]*contracts.Pipeline, error)
	GetPipelinesCount(map[string][]string) (int, error)
	GetPipeline(string, string, string, bool) (*contracts.Pipeline, error)
	GetPipelineBuilds(string, string, string, int, int, map[string][]string, bool) ([]*contracts.Build, error)
	GetPipelineBuildsCount(string, string, string, map[string][]string) (int, error)
	GetPipelineBuild(string, string, string, string, bool) (*contracts.Build, error)
	GetPipelineBuildByID(string, string, string, int, bool) (*contracts.Build, error)
	GetLastPipelineBuild(string, string, string, bool) (*contracts.Build, error)
	GetFirstPipelineBuild(string, string, string, bool) (*contracts.Build, error)
	GetLastPipelineBuildForBranch(string, string, string, string) (*contracts.Build, error)
	GetLastPipelineRelease(string, string, string, string, string) (*contracts.Release, error)
	GetFirstPipelineRelease(string, string, string, string, string) (*contracts.Release, error)
	GetPipelineBuildsByVersion(string, string, string, string, bool) ([]*contracts.Build, error)
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
	GetPipelineBuildsDurations(string, string, string, map[string][]string) ([]map[string]interface{}, error)
	GetPipelineReleasesDurations(string, string, string, map[string][]string) ([]map[string]interface{}, error)
	GetFrequentLabels(int, int, map[string][]string) ([]map[string]interface{}, error)
	GetFrequentLabelsCount(map[string][]string) (int, error)

	GetPipelinesWithMostBuilds(pageNumber, pageSize int, filters map[string][]string) ([]map[string]interface{}, error)
	GetPipelinesWithMostBuildsCount(filters map[string][]string) (int, error)
	GetPipelinesWithMostReleases(pageNumber, pageSize int, filters map[string][]string) ([]map[string]interface{}, error)
	GetPipelinesWithMostReleasesCount(filters map[string][]string) (int, error)

	GetTriggers(triggerType, identifier, event string) ([]*contracts.Pipeline, error)
	GetPipelineTriggers(build contracts.Build, event string) ([]*contracts.Pipeline, error)
	GetReleaseTriggers(release contracts.Release, event string) ([]*contracts.Pipeline, error)
	GetCronTriggers() ([]*contracts.Pipeline, error)

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
func (dbc *cockroachDBClientImpl) ConnectWithDriverAndSource(driverName, dataSourceName string) (err error) {

	dbc.databaseConnection, err = sql.Open(driverName, dataSourceName)
	if err != nil {
		return
	}

	return
}

// GetAutoIncrement returns the autoincrement number for a pipeline
func (dbc *cockroachDBClientImpl) GetAutoIncrement(shortRepoSource, repoOwner, repoName string) (autoincrement int, err error) {

	dbc.PrometheusOutboundAPICallTotals.With(prometheus.Labels{"target": "cockroachdb"}).Inc()

	repoFullName := fmt.Sprintf("%v/%v", repoOwner, repoName)

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
		shortRepoSource,
		repoFullName,
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
		shortRepoSource,
		repoFullName,
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

func (dbc *cockroachDBClientImpl) InsertBuild(build contracts.Build) (insertedBuild *contracts.Build, err error) {
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
	triggersBytes, err := json.Marshal(build.Triggers)
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
			commits,
			triggers
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
			$12
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
		triggersBytes,
	)

	insertedBuild = &build

	if err = row.Scan(&insertedBuild.ID); err != nil {
		return
	}

	// update computed tables
	go dbc.UpsertComputedPipeline(insertedBuild.RepoSource, insertedBuild.RepoOwner, insertedBuild.RepoName)

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

	// update computed tables
	go dbc.UpsertComputedPipeline(repoSource, repoOwner, repoName)

	return
}

func (dbc *cockroachDBClientImpl) InsertRelease(release contracts.Release) (insertedRelease *contracts.Release, err error) {
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

	insertedRelease = &release
	if err = rows.Scan(&insertedRelease.ID); err != nil {
		return
	}

	// update computed tables
	go func() {
		dbc.UpsertComputedRelease(insertedRelease.RepoSource, insertedRelease.RepoOwner, insertedRelease.RepoName, insertedRelease.Name, insertedRelease.Action)
		dbc.UpsertComputedPipeline(insertedRelease.RepoSource, insertedRelease.RepoOwner, insertedRelease.RepoName)
	}()

	return
}

func (dbc *cockroachDBClientImpl) UpdateReleaseStatus(repoSource, repoOwner, repoName string, id int, releaseStatus string) (err error) {

	dbc.PrometheusOutboundAPICallTotals.With(prometheus.Labels{"target": "cockroachdb"}).Inc()

	// generate query
	row := dbc.databaseConnection.QueryRow(
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
		RETURNING
			id,
			repo_source,
			repo_owner,
			repo_name,
			release,
			release_action,
			release_version,
			release_status,
			triggered_by,
			inserted_at,
			updated_at,
			duration::INT
		`,
		releaseStatus,
		id,
		repoSource,
		repoOwner,
		repoName,
	)

	// execute query
	insertedRelease, err := dbc.scanRelease(row)
	if err != nil {
		return
	}

	// update computed tables
	go func() {
		dbc.UpsertComputedRelease(insertedRelease.RepoSource, insertedRelease.RepoOwner, insertedRelease.RepoName, insertedRelease.Name, insertedRelease.Action)
		dbc.UpsertComputedPipeline(repoSource, repoOwner, repoName)
	}()

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
				build_logs
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
			build_logs
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

func (dbc *cockroachDBClientImpl) UpsertComputedPipeline(repoSource, repoOwner, repoName string) (err error) {
	dbc.PrometheusOutboundAPICallTotals.With(prometheus.Labels{"target": "cockroachdb"}).Inc()

	// get last build
	lastBuild, err := dbc.GetLastPipelineBuild(repoSource, repoOwner, repoName, false)
	if err != nil {
		log.Error().Err(err).Msgf("Failed getting last build for upserting computed pipeline %v/%v/%v", repoSource, repoOwner, repoName)
		return
	}
	if lastBuild == nil {
		log.Error().Msgf("Failed getting last build for upserting computed pipeline %v/%v/%v", repoSource, repoOwner, repoName)
		return
	}

	upsertedPipeline := dbc.mapBuildToPipeline(lastBuild)

	// add releases
	dbc.enrichPipeline(upsertedPipeline)

	// set LastUpdatedAt from both builds and releases
	upsertedPipeline.LastUpdatedAt = upsertedPipeline.InsertedAt
	if upsertedPipeline.UpdatedAt.After(upsertedPipeline.LastUpdatedAt) {
		upsertedPipeline.LastUpdatedAt = upsertedPipeline.UpdatedAt
	}
	for _, rt := range upsertedPipeline.ReleaseTargets {
		for _, ar := range rt.ActiveReleases {
			if ar.InsertedAt != nil && ar.InsertedAt.After(upsertedPipeline.LastUpdatedAt) {
				upsertedPipeline.LastUpdatedAt = *ar.InsertedAt
			}
			if ar.UpdatedAt != nil && ar.UpdatedAt.After(upsertedPipeline.LastUpdatedAt) {
				upsertedPipeline.LastUpdatedAt = *ar.UpdatedAt
			}
		}
	}

	// sort labels by key
	sort.Slice(upsertedPipeline.Labels, func(i, j int) bool {
		return upsertedPipeline.Labels[i].Key < upsertedPipeline.Labels[j].Key
	})

	labelsBytes, err := json.Marshal(upsertedPipeline.Labels)
	if err != nil {
		log.Error().Err(err).Msgf("Failed upserting computed pipeline %v/%v/%v", upsertedPipeline.RepoSource, upsertedPipeline.RepoOwner, upsertedPipeline.RepoName)
		return
	}
	releaseTargetsBytes, err := json.Marshal(upsertedPipeline.ReleaseTargets)
	if err != nil {
		log.Error().Err(err).Msgf("Failed upserting computed pipeline %v/%v/%v", upsertedPipeline.RepoSource, upsertedPipeline.RepoOwner, upsertedPipeline.RepoName)
		return
	}
	commitsBytes, err := json.Marshal(upsertedPipeline.Commits)
	if err != nil {
		log.Error().Err(err).Msgf("Failed upserting computed pipeline %v/%v/%v", upsertedPipeline.RepoSource, upsertedPipeline.RepoOwner, upsertedPipeline.RepoName)
		return
	}
	triggersBytes, err := json.Marshal(upsertedPipeline.Triggers)
	if err != nil {
		log.Error().Err(err).Msgf("Failed upserting computed pipeline %v/%v/%v", upsertedPipeline.RepoSource, upsertedPipeline.RepoOwner, upsertedPipeline.RepoName)
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
			triggers,
			inserted_at,
			first_inserted_at,
			updated_at,
			duration,
			last_updated_at
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
			$15,
			$16,
			AGE($16,$15),
			$17
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
			triggers = excluded.triggers,
			inserted_at = excluded.inserted_at,
			updated_at = excluded.updated_at,
			duration = AGE(excluded.updated_at,excluded.inserted_at),
			last_updated_at = excluded.last_updated_at
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
		triggersBytes,
		upsertedPipeline.InsertedAt,
		upsertedPipeline.InsertedAt,
		upsertedPipeline.UpdatedAt,
		upsertedPipeline.LastUpdatedAt,
	)
	if err != nil {
		log.Error().Err(err).Msgf("Failed upserting computed pipeline %v/%v/%v", repoSource, repoOwner, repoName)
		return
	}

	return
}

func (dbc *cockroachDBClientImpl) UpdateComputedPipelineFirstInsertedAt(repoSource, repoOwner, repoName string) (err error) {
	dbc.PrometheusOutboundAPICallTotals.With(prometheus.Labels{"target": "cockroachdb"}).Inc()

	// get first build
	firstBuild, err := dbc.GetFirstPipelineBuild(repoSource, repoOwner, repoName, false)
	if err != nil {
		log.Error().Err(err).Msgf("Failed getting first build for updating computed pipeline %v/%v/%v", repoSource, repoOwner, repoName)
		return
	}
	if firstBuild == nil {
		log.Error().Msgf("Failed getting first build for updating computed pipeline %v/%v/%v", repoSource, repoOwner, repoName)
		return
	}

	updatedPipeline := dbc.mapBuildToPipeline(firstBuild)

	// update computed pipeline
	_, err = dbc.databaseConnection.Exec(
		`
		UPDATE
			computed_pipelines
		SET
			first_inserted_at=$1
		WHERE
			repo_source=$2 AND
			repo_owner=$3 AND
			repo_name=$4
		`,
		updatedPipeline.InsertedAt,
		updatedPipeline.RepoSource,
		updatedPipeline.RepoOwner,
		updatedPipeline.RepoName,
	)
	if err != nil {
		log.Error().Err(err).Msgf("Failed updating computed pipeline %v/%v/%v", repoSource, repoOwner, repoName)
		return
	}

	return
}

func (dbc *cockroachDBClientImpl) UpsertComputedRelease(repoSource, repoOwner, repoName, releaseName, releaseAction string) (err error) {
	dbc.PrometheusOutboundAPICallTotals.With(prometheus.Labels{"target": "cockroachdb"}).Inc()

	// get last release
	lastRelease, err := dbc.GetLastPipelineRelease(repoSource, repoOwner, repoName, releaseName, releaseAction)
	if err != nil {
		log.Error().Err(err).Msgf("Failed getting last release for upserting computed release %v/%v/%v/%v/%v", repoSource, repoOwner, repoName, releaseName, releaseAction)
		return
	}
	if lastRelease == nil {
		log.Error().Msgf("Failed getting last release for upserting computed release %v/%v/%v/%v/%v", repoSource, repoOwner, repoName, releaseName, releaseAction)
		return
	}

	// upsert computed release
	_, err = dbc.databaseConnection.Exec(
		`
		INSERT INTO
			computed_releases
		(
			release_id,
			repo_source,
			repo_owner,
			repo_name,
			release,
			release_version,
			release_status,
			inserted_at,
			first_inserted_at,
			updated_at,
			triggered_by,
			duration,
			release_action
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
			$8,
			$9,
			$10,
			AGE($9,$8),
			$11
		)
		ON CONFLICT
		(
			repo_source,
			repo_owner,
			repo_name,
			release,
			release_action
		)
		DO UPDATE SET
			release_id = excluded.release_id,
			release_version = excluded.release_version,
			release_status = excluded.release_status,
			inserted_at = excluded.inserted_at,
			updated_at = excluded.updated_at,
			triggered_by = excluded.triggered_by,
			duration = AGE(excluded.updated_at,excluded.inserted_at)
		`,
		lastRelease.ID,
		lastRelease.RepoSource,
		lastRelease.RepoOwner,
		lastRelease.RepoName,
		lastRelease.Name,
		lastRelease.ReleaseVersion,
		lastRelease.ReleaseStatus,
		lastRelease.InsertedAt,
		lastRelease.UpdatedAt,
		lastRelease.TriggeredBy,
		lastRelease.Action,
	)
	if err != nil {
		log.Error().Err(err).Msgf("Failed upserting computed release %v/%v/%v/%v/%v", repoSource, repoOwner, repoName, releaseName, releaseAction)
		return
	}

	return
}

func (dbc *cockroachDBClientImpl) UpdateComputedReleaseFirstInsertedAt(repoSource, repoOwner, repoName, releaseName, releaseAction string) (err error) {
	dbc.PrometheusOutboundAPICallTotals.With(prometheus.Labels{"target": "cockroachdb"}).Inc()

	// get first release
	firstRelease, err := dbc.GetFirstPipelineRelease(repoSource, repoOwner, repoName, releaseName, releaseAction)
	if err != nil {
		log.Error().Err(err).Msgf("Failed getting first release for updating computed release %v/%v/%v/%v/%v", repoSource, repoOwner, repoName, releaseName, releaseAction)
		return
	}
	if firstRelease == nil {
		log.Error().Msgf("Failed getting first release for updating computed release %v/%v/%v/%v/%v", repoSource, repoOwner, repoName, releaseName, releaseAction)
		return
	}

	// update computed release
	_, err = dbc.databaseConnection.Exec(
		`
		UPDATE
			computed_releases
		SET
			first_inserted_at=$1
		WHERE
			repo_source=$2 AND
			repo_owner=$3 AND
			repo_name=$4 AND
			release=$5 AND
			release_action=$6
		`,
		firstRelease.InsertedAt,
		firstRelease.RepoSource,
		firstRelease.RepoOwner,
		firstRelease.RepoName,
		firstRelease.Name,
		firstRelease.Action,
	)
	if err != nil {
		log.Error().Err(err).Msgf("Failed updating computed release %v/%v/%v/%v/%v", repoSource, repoOwner, repoName, releaseName, releaseAction)
		return
	}

	return
}

func (dbc *cockroachDBClientImpl) GetPipelines(pageNumber, pageSize int, filters map[string][]string, optimized bool) (pipelines []*contracts.Pipeline, err error) {

	dbc.PrometheusOutboundAPICallTotals.With(prometheus.Labels{"target": "cockroachdb"}).Inc()

	// generate query
	query := dbc.selectPipelinesQuery().
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
	if pipelines, err = dbc.scanPipelines(rows, optimized); err != nil {
		return
	}

	return
}

func (dbc *cockroachDBClientImpl) GetPipelinesByRepoName(repoName string, optimized bool) (pipelines []*contracts.Pipeline, err error) {

	dbc.PrometheusOutboundAPICallTotals.With(prometheus.Labels{"target": "cockroachdb"}).Inc()

	// generate query
	query := dbc.selectPipelinesQuery().
		Where(sq.Eq{"a.repo_name": repoName})

	// execute query
	rows, err := query.RunWith(dbc.databaseConnection).Query()
	if err != nil {
		return
	}

	// read rows
	if pipelines, err = dbc.scanPipelines(rows, optimized); err != nil {
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
			From("computed_pipelines a")

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

func (dbc *cockroachDBClientImpl) GetPipeline(repoSource, repoOwner, repoName string, optimized bool) (pipeline *contracts.Pipeline, err error) {

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
	if pipeline, err = dbc.scanPipeline(row, optimized); err != nil {
		return
	}

	return
}

func (dbc *cockroachDBClientImpl) GetPipelineBuilds(repoSource, repoOwner, repoName string, pageNumber, pageSize int, filters map[string][]string, optimized bool) (builds []*contracts.Build, err error) {

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
	if builds, err = dbc.scanBuilds(rows, optimized); err != nil {
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

func (dbc *cockroachDBClientImpl) GetPipelineBuild(repoSource, repoOwner, repoName, repoRevision string, optimized bool) (build *contracts.Build, err error) {

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
	if build, err = dbc.scanBuild(row, optimized, true); err != nil {
		return
	}

	return
}

func (dbc *cockroachDBClientImpl) GetPipelineBuildByID(repoSource, repoOwner, repoName string, id int, optimized bool) (build *contracts.Build, err error) {

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
	if build, err = dbc.scanBuild(row, optimized, true); err != nil {
		return
	}

	return
}

func (dbc *cockroachDBClientImpl) GetLastPipelineBuild(repoSource, repoOwner, repoName string, optimized bool) (build *contracts.Build, err error) {

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
	if build, err = dbc.scanBuild(row, optimized, false); err != nil {
		return
	}

	return
}

func (dbc *cockroachDBClientImpl) GetFirstPipelineBuild(repoSource, repoOwner, repoName string, optimized bool) (build *contracts.Build, err error) {

	dbc.PrometheusOutboundAPICallTotals.With(prometheus.Labels{"target": "cockroachdb"}).Inc()

	// generate query
	query := dbc.selectBuildsQuery().
		Where(sq.Eq{"a.repo_source": repoSource}).
		Where(sq.Eq{"a.repo_owner": repoOwner}).
		Where(sq.Eq{"a.repo_name": repoName}).
		OrderBy("a.inserted_at ASC").
		Limit(uint64(1))

	// execute query
	row := query.RunWith(dbc.databaseConnection).QueryRow()
	if build, err = dbc.scanBuild(row, optimized, false); err != nil {
		return
	}

	return
}

func (dbc *cockroachDBClientImpl) GetLastPipelineBuildForBranch(repoSource, repoOwner, repoName, branch string) (build *contracts.Build, err error) {

	dbc.PrometheusOutboundAPICallTotals.With(prometheus.Labels{"target": "cockroachdb"}).Inc()

	// generate query
	query := dbc.selectBuildsQuery().
		Where(sq.Eq{"a.repo_source": repoSource}).
		Where(sq.Eq{"a.repo_owner": repoOwner}).
		Where(sq.Eq{"a.repo_name": repoName}).
		Where(sq.Eq{"a.repo_branch": branch}).
		OrderBy("a.inserted_at DESC").
		Limit(uint64(1))

	// execute query
	row := query.RunWith(dbc.databaseConnection).QueryRow()
	if build, err = dbc.scanBuild(row, false, false); err != nil {
		return
	}

	return
}

func (dbc *cockroachDBClientImpl) GetLastPipelineRelease(repoSource, repoOwner, repoName, releaseName, releaseAction string) (release *contracts.Release, err error) {

	dbc.PrometheusOutboundAPICallTotals.With(prometheus.Labels{"target": "cockroachdb"}).Inc()

	// generate query
	query := dbc.selectReleasesQuery().
		Where(sq.Eq{"a.repo_source": repoSource}).
		Where(sq.Eq{"a.repo_owner": repoOwner}).
		Where(sq.Eq{"a.repo_name": repoName}).
		Where(sq.Eq{"a.release": releaseName}).
		Where(sq.Eq{"a.release_action": releaseAction}).
		OrderBy("a.inserted_at DESC").
		Limit(uint64(1))

	// execute query
	row := query.RunWith(dbc.databaseConnection).QueryRow()
	if release, err = dbc.scanRelease(row); err != nil {
		return
	}

	return
}

func (dbc *cockroachDBClientImpl) GetFirstPipelineRelease(repoSource, repoOwner, repoName, releaseName, releaseAction string) (release *contracts.Release, err error) {

	dbc.PrometheusOutboundAPICallTotals.With(prometheus.Labels{"target": "cockroachdb"}).Inc()

	// generate query
	query := dbc.selectReleasesQuery().
		Where(sq.Eq{"a.repo_source": repoSource}).
		Where(sq.Eq{"a.repo_owner": repoOwner}).
		Where(sq.Eq{"a.repo_name": repoName}).
		Where(sq.Eq{"a.release": releaseName}).
		Where(sq.Eq{"a.release_action": releaseAction}).
		OrderBy("a.inserted_at ASC").
		Limit(uint64(1))

	// execute query
	row := query.RunWith(dbc.databaseConnection).QueryRow()
	if release, err = dbc.scanRelease(row); err != nil {
		return
	}

	return
}

func (dbc *cockroachDBClientImpl) GetPipelineBuildsByVersion(repoSource, repoOwner, repoName, buildVersion string, optimized bool) (builds []*contracts.Build, err error) {
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
	if builds, err = dbc.scanBuilds(rows, optimized); err != nil {
		return
	}

	return
}

func (dbc *cockroachDBClientImpl) GetPipelineBuildLogs(repoSource, repoOwner, repoName, repoBranch, repoRevision, buildID string) (buildLog *contracts.BuildLog, err error) {

	dbc.PrometheusOutboundAPICallTotals.With(prometheus.Labels{"target": "cockroachdb"}).Inc()

	buildIDAsInt, err := strconv.Atoi(buildID)
	if err != nil {
		return nil, err
	}

	// generate query
	query := dbc.selectBuildLogsQuery().
		Where(sq.Eq{"a.build_id": buildIDAsInt}).
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
	query := dbc.selectComputedReleasesQuery().
		Where(sq.Eq{"a.release": releaseName}).
		Where(sq.Eq{"a.repo_source": repoSource}).
		Where(sq.Eq{"a.repo_owner": repoOwner}).
		Where(sq.Eq{"a.repo_name": repoName}).
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
		Where(sq.Eq{"a.release_id": id}).
		Where(sq.Eq{"a.repo_source": repoSource}).
		Where(sq.Eq{"a.repo_owner": repoOwner}).
		Where(sq.Eq{"a.repo_name": repoName}).
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
			Select("a.first_inserted_at").
			From("computed_pipelines a").
			OrderBy("a.first_inserted_at")

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
			Select("MIN(a.first_inserted_at)").
			From("computed_releases a").
			GroupBy("a.repo_source,a.repo_owner,a.repo_name").
			OrderBy("MIN(a.first_inserted_at)")

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

func (dbc *cockroachDBClientImpl) GetPipelineBuildsDurations(repoSource, repoOwner, repoName string, filters map[string][]string) (durations []map[string]interface{}, err error) {
	dbc.PrometheusOutboundAPICallTotals.With(prometheus.Labels{"target": "cockroachdb"}).Inc()

	// generate query
	innerquery :=
		sq.StatementBuilder.PlaceholderFormat(sq.Dollar).
			Select("a.inserted_at, a.duration::INT").
			From("builds a").
			Where(sq.Eq{"a.repo_source": repoSource}).
			Where(sq.Eq{"a.repo_owner": repoOwner}).
			Where(sq.Eq{"a.repo_name": repoName}).
			OrderBy("a.inserted_at DESC")

	innerquery, err = whereClauseGeneratorForBuildStatusFilter(innerquery, "a", filters)
	if err != nil {
		return
	}

	innerquery, err = limitClauseGeneratorForLastFilter(innerquery, filters)
	if err != nil {
		return
	}

	query :=
		sq.StatementBuilder.PlaceholderFormat(sq.Dollar).
			Select("*").
			FromSelect(innerquery, "a").
			OrderBy("a.duration")

	durations = make([]map[string]interface{}, 0)

	rows, err := query.RunWith(dbc.databaseConnection).Query()

	if err != nil {
		return
	}

	defer rows.Close()
	for rows.Next() {

		var insertedAt time.Time
		var seconds int

		if err = rows.Scan(
			&insertedAt, &seconds); err != nil {
			return
		}

		duration := time.Duration(seconds) * time.Second

		durations = append(durations, map[string]interface{}{
			"insertedAt": insertedAt,
			"duration":   duration,
		})
	}

	return
}

func (dbc *cockroachDBClientImpl) GetPipelineReleasesDurations(repoSource, repoOwner, repoName string, filters map[string][]string) (durations []map[string]interface{}, err error) {
	dbc.PrometheusOutboundAPICallTotals.With(prometheus.Labels{"target": "cockroachdb"}).Inc()

	// generate query
	innerquery :=
		sq.StatementBuilder.PlaceholderFormat(sq.Dollar).
			Select("a.inserted_at, a.release, a.release_action, a.duration::INT").
			From("releases a").
			Where(sq.Eq{"a.repo_source": repoSource}).
			Where(sq.Eq{"a.repo_owner": repoOwner}).
			Where(sq.Eq{"a.repo_name": repoName}).
			OrderBy("a.inserted_at DESC")

	innerquery, err = whereClauseGeneratorForReleaseStatusFilter(innerquery, "a", filters)
	if err != nil {
		return
	}

	innerquery, err = limitClauseGeneratorForLastFilter(innerquery, filters)
	if err != nil {
		return
	}

	query :=
		sq.StatementBuilder.PlaceholderFormat(sq.Dollar).
			Select("*").
			FromSelect(innerquery, "a").
			OrderBy("a.duration")

	durations = make([]map[string]interface{}, 0)

	rows, err := query.RunWith(dbc.databaseConnection).Query()

	if err != nil {
		return
	}

	defer rows.Close()
	for rows.Next() {

		var insertedAt time.Time
		var releaseName, releaseAction string
		var seconds int

		if err = rows.Scan(
			&insertedAt, &releaseName, &releaseAction, &seconds); err != nil {
			return
		}

		duration := time.Duration(seconds) * time.Second

		durations = append(durations, map[string]interface{}{
			"insertedAt": insertedAt,
			"name":       releaseName,
			"action":     releaseAction,
			"duration":   duration,
		})
	}

	return
}

func (dbc *cockroachDBClientImpl) GetFrequentLabels(pageNumber, pageSize int, filters map[string][]string) (labels []map[string]interface{}, err error) {
	dbc.PrometheusOutboundAPICallTotals.With(prometheus.Labels{"target": "cockroachdb"}).Inc()

	// see https://github.com/cockroachdb/cockroach/issues/35848

	// for time being run following query, where the dynamic where clause is in the innermost select query:

	// SELECT
	// 		key, value, nr_computed_pipelines
	// FROM
	// 		(
	// 				SELECT
	// 						key, value, count(DISTINCT id) AS nr_computed_pipelines
	// 				FROM
	// 						(
	// 								SELECT
	// 										l->>'key' AS key, l->>'value' AS value, id
	// 								FROM
	// 										(SELECT id, jsonb_array_elements(labels) AS l FROM computed_pipelines where jsonb_typeof(labels) = 'array')
	// 						)
	// 				GROUP BY
	// 						key, value
	// 		)
	// WHERE
	// 		nr_computed_pipelines > 1
	// ORDER BY
	// 		nr_computed_pipelines DESC, key, value
	// LIMIT 10;

	arrayElementsQuery :=
		sq.StatementBuilder.
			Select("a.id, jsonb_array_elements(a.labels) AS l").
			From("computed_pipelines a").
			Where("jsonb_typeof(labels) = 'array'")

	arrayElementsQuery, err = whereClauseGeneratorForSinceFilter(arrayElementsQuery, "a", filters)
	if err != nil {
		return
	}

	arrayElementsQuery, err = whereClauseGeneratorForBuildStatusFilter(arrayElementsQuery, "a", filters)
	if err != nil {
		return
	}

	arrayElementsQuery, err = whereClauseGeneratorForLabelsFilter(arrayElementsQuery, "a", filters)
	if err != nil {
		return
	}

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
			Limit(uint64(pageSize)).
			Offset(uint64((pageNumber - 1) * pageSize))

	rows, err := query.RunWith(dbc.databaseConnection).Query()

	if err != nil {
		return
	}

	defer rows.Close()

	cols, _ := rows.Columns()
	for rows.Next() {
		// Create a slice of interface{}'s to represent each column,
		// and a second slice to contain pointers to each item in the columns slice.
		columns := make([]interface{}, len(cols))
		columnPointers := make([]interface{}, len(cols))
		for i := range columns {
			columnPointers[i] = &columns[i]
		}

		// Scan the result into the column pointers...
		if err = rows.Scan(columnPointers...); err != nil {
			return nil, err
		}

		// Create our map, and retrieve the value for each column from the pointers slice,
		// storing it in the map with the name of the column as the key.
		m := make(map[string]interface{})
		for i, colName := range cols {
			val := columnPointers[i].(*interface{})
			m[colName] = *val
		}

		labels = append(labels, m)
	}

	return
}

func (dbc *cockroachDBClientImpl) GetFrequentLabelsCount(filters map[string][]string) (totalCount int, err error) {
	dbc.PrometheusOutboundAPICallTotals.With(prometheus.Labels{"target": "cockroachdb"}).Inc()

	// see https://github.com/cockroachdb/cockroach/issues/35848

	// for time being run following query, where the dynamic where clause is in the innermost select query:

	// SELECT
	// 		key, value, nr_computed_pipelines
	// FROM
	// 		(
	// 				SELECT
	// 						key, value, count(DISTINCT id) AS nr_computed_pipelines
	// 				FROM
	// 						(
	// 								SELECT
	// 										l->>'key' AS key, l->>'value' AS value, id
	// 								FROM
	// 										(SELECT id, jsonb_array_elements(labels) AS l FROM computed_pipelines where jsonb_typeof(labels) = 'array')
	// 						)
	// 				GROUP BY
	// 						key, value
	// 		)
	// WHERE
	// 		nr_computed_pipelines > 1
	// ORDER BY
	// 		nr_computed_pipelines DESC, key, value
	// LIMIT 10;

	arrayElementsQuery :=
		sq.StatementBuilder.
			Select("a.id, jsonb_array_elements(a.labels) AS l").
			From("computed_pipelines a").
			Where("jsonb_typeof(labels) = 'array'")

	arrayElementsQuery, err = whereClauseGeneratorForSinceFilter(arrayElementsQuery, "a", filters)
	if err != nil {
		return
	}

	arrayElementsQuery, err = whereClauseGeneratorForBuildStatusFilter(arrayElementsQuery, "a", filters)
	if err != nil {
		return
	}

	arrayElementsQuery, err = whereClauseGeneratorForLabelsFilter(arrayElementsQuery, "a", filters)
	if err != nil {
		return
	}

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
			Select("COUNT(key)").
			FromSelect(groupByQuery, "d").
			Where(sq.Gt{"pipelinesCount": 1})

	// execute query
	row := query.RunWith(dbc.databaseConnection).QueryRow()
	if err = row.Scan(&totalCount); err != nil {
		return
	}

	return
}

func (dbc *cockroachDBClientImpl) GetPipelinesWithMostBuilds(pageNumber, pageSize int, filters map[string][]string) (pipelines []map[string]interface{}, err error) {
	dbc.PrometheusOutboundAPICallTotals.With(prometheus.Labels{"target": "cockroachdb"}).Inc()

	// generate query
	query :=
		sq.StatementBuilder.PlaceholderFormat(sq.Dollar).
			Select("a.repo_source, a.repo_owner, a.repo_name, count(a.id) as nr_records").
			From("builds a").
			GroupBy("a.repo_source, a.repo_owner, a.repo_name").
			OrderBy("nr_records DESC, a.repo_source, a.repo_owner, a.repo_name").
			Limit(uint64(pageSize)).
			Offset(uint64((pageNumber - 1) * pageSize))

	query, err = whereClauseGeneratorForSinceFilter(query, "a", filters)
	if err != nil {
		return
	}

	query, err = whereClauseGeneratorForBuildStatusFilter(query, "a", filters)
	if err != nil {
		return
	}

	query, err = whereClauseGeneratorForLabelsFilter(query, "a", filters)
	if err != nil {
		return
	}

	pipelines = make([]map[string]interface{}, 0)

	rows, err := query.RunWith(dbc.databaseConnection).Query()

	if err != nil {
		return
	}

	defer rows.Close()

	cols, _ := rows.Columns()
	for rows.Next() {
		// Create a slice of interface{}'s to represent each column,
		// and a second slice to contain pointers to each item in the columns slice.
		columns := make([]interface{}, len(cols))
		columnPointers := make([]interface{}, len(cols))
		for i := range columns {
			columnPointers[i] = &columns[i]
		}

		// Scan the result into the column pointers...
		if err = rows.Scan(columnPointers...); err != nil {
			return nil, err
		}

		// Create our map, and retrieve the value for each column from the pointers slice,
		// storing it in the map with the name of the column as the key.
		m := make(map[string]interface{})
		for i, colName := range cols {
			val := columnPointers[i].(*interface{})
			m[colName] = *val
		}

		pipelines = append(pipelines, m)
	}

	return
}

func (dbc *cockroachDBClientImpl) GetPipelinesWithMostBuildsCount(filters map[string][]string) (totalCount int, err error) {
	return dbc.GetPipelinesCount(filters)
}

func (dbc *cockroachDBClientImpl) GetPipelinesWithMostReleases(pageNumber, pageSize int, filters map[string][]string) (pipelines []map[string]interface{}, err error) {
	dbc.PrometheusOutboundAPICallTotals.With(prometheus.Labels{"target": "cockroachdb"}).Inc()

	// generate query
	query :=
		sq.StatementBuilder.PlaceholderFormat(sq.Dollar).
			Select("a.repo_source, a.repo_owner, a.repo_name, count(a.id) as nr_records").
			From("releases a").
			GroupBy("a.repo_source, a.repo_owner, a.repo_name").
			OrderBy("nr_records DESC, a.repo_source, a.repo_owner, a.repo_name").
			Limit(uint64(pageSize)).
			Offset(uint64((pageNumber - 1) * pageSize))

	query, err = whereClauseGeneratorForSinceFilter(query, "a", filters)
	if err != nil {
		return
	}

	query, err = whereClauseGeneratorForReleaseStatusFilter(query, "a", filters)
	if err != nil {
		return
	}

	pipelines = make([]map[string]interface{}, 0)

	rows, err := query.RunWith(dbc.databaseConnection).Query()

	if err != nil {
		return
	}

	defer rows.Close()

	cols, _ := rows.Columns()
	for rows.Next() {
		// Create a slice of interface{}'s to represent each column,
		// and a second slice to contain pointers to each item in the columns slice.
		columns := make([]interface{}, len(cols))
		columnPointers := make([]interface{}, len(cols))
		for i := range columns {
			columnPointers[i] = &columns[i]
		}

		// Scan the result into the column pointers...
		if err = rows.Scan(columnPointers...); err != nil {
			return nil, err
		}

		// Create our map, and retrieve the value for each column from the pointers slice,
		// storing it in the map with the name of the column as the key.
		m := make(map[string]interface{})
		for i, colName := range cols {
			val := columnPointers[i].(*interface{})
			m[colName] = *val
		}

		pipelines = append(pipelines, m)
	}

	return
}

func (dbc *cockroachDBClientImpl) GetPipelinesWithMostReleasesCount(filters map[string][]string) (totalCount int, err error) {

	dbc.PrometheusOutboundAPICallTotals.With(prometheus.Labels{"target": "cockroachdb"}).Inc()

	// generate query
	innerquery :=
		sq.StatementBuilder.PlaceholderFormat(sq.Dollar).
			Select("a.repo_source, a.repo_owner, a.repo_name").
			From("releases a").
			GroupBy("a.repo_source, a.repo_owner, a.repo_name")

	innerquery, err = whereClauseGeneratorForSinceFilter(innerquery, "a", filters)
	if err != nil {
		return
	}

	innerquery, err = whereClauseGeneratorForReleaseStatusFilter(innerquery, "a", filters)
	if err != nil {
		return
	}

	query :=
		sq.StatementBuilder.PlaceholderFormat(sq.Dollar).
			Select("count(*)").
			FromSelect(innerquery, "a")

	// execute query
	row := query.RunWith(dbc.databaseConnection).QueryRow()
	if err = row.Scan(&totalCount); err != nil {
		return
	}

	return
}

func whereClauseGeneratorForAllFilters(query sq.SelectBuilder, alias string, filters map[string][]string) (sq.SelectBuilder, error) {

	query, err := whereClauseGeneratorForSinceFilter(query, alias, filters)
	if err != nil {
		return query, err
	}
	query, err = whereClauseGeneratorForBuildStatusFilter(query, alias, filters)
	if err != nil {
		return query, err
	}
	query, err = whereClauseGeneratorForLabelsFilter(query, alias, filters)
	if err != nil {
		return query, err
	}
	query, err = whereClauseGeneratorForSearchFilter(query, alias, filters)
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
		case "1h":
			query = query.Where(sq.GtOrEq{fmt.Sprintf("%v.inserted_at", alias): time.Now().Add(time.Duration(-1) * time.Hour)})
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

func whereClauseGeneratorForSearchFilter(query sq.SelectBuilder, alias string, filters map[string][]string) (sq.SelectBuilder, error) {

	if search, ok := filters["search"]; ok && len(search) > 0 && search[0] != "" {
		searchValue := search[0]
		query = query.Where(sq.Like{fmt.Sprintf("%v.repo_name", alias): fmt.Sprint("%", searchValue, "%")})
	}

	return query, nil
}

func whereClauseGeneratorForBuildStatusFilter(query sq.SelectBuilder, alias string, filters map[string][]string) (sq.SelectBuilder, error) {

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

func limitClauseGeneratorForLastFilter(query sq.SelectBuilder, filters map[string][]string) (sq.SelectBuilder, error) {

	if last, ok := filters["last"]; ok && len(last) == 1 {
		lastValue := last[0]
		limitSize, err := strconv.ParseUint(lastValue, 10, 64)
		if err != nil {
			return query, err
		}

		query = query.Limit(limitSize)
	}

	return query, nil
}

func (dbc *cockroachDBClientImpl) scanBuild(row sq.RowScanner, optimized, enriched bool) (build *contracts.Build, err error) {

	build = &contracts.Build{}
	var labelsData, releaseTargetsData, commitsData, triggersData []uint8
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
		&triggersData,
		&build.InsertedAt,
		&build.UpdatedAt,
		&seconds); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return
	}

	build.Duration = time.Duration(seconds) * time.Second

	dbc.setBuildPropertiesFromJSONB(build, labelsData, releaseTargetsData, commitsData, triggersData, optimized)

	if enriched {
		dbc.enrichBuild(build)
	}

	if optimized {
		// clear some properties for reduced size and improved performance over the network
		build.Manifest = ""
		build.ManifestWithDefaults = ""
	}

	return
}

func (dbc *cockroachDBClientImpl) scanBuilds(rows *sql.Rows, optimized bool) (builds []*contracts.Build, err error) {

	builds = make([]*contracts.Build, 0)

	defer rows.Close()
	for rows.Next() {

		build := contracts.Build{}
		var labelsData, releaseTargetsData, commitsData, triggersData []uint8
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
			&triggersData,
			&build.InsertedAt,
			&build.UpdatedAt,
			&seconds); err != nil {
			return
		}

		build.Duration = time.Duration(seconds) * time.Second

		dbc.setBuildPropertiesFromJSONB(&build, labelsData, releaseTargetsData, commitsData, triggersData, optimized)

		if optimized {
			// clear some properties for reduced size and improved performance over the network
			build.Manifest = ""
			build.ManifestWithDefaults = ""
		}

		builds = append(builds, &build)
	}

	return
}

func (dbc *cockroachDBClientImpl) scanPipeline(row sq.RowScanner, optimized bool) (pipeline *contracts.Pipeline, err error) {

	pipeline = &contracts.Pipeline{}
	var labelsData, releaseTargetsData, commitsData, triggersData []uint8
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
		&triggersData,
		&pipeline.InsertedAt,
		&pipeline.UpdatedAt,
		&seconds,
		&pipeline.LastUpdatedAt); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return
	}

	pipeline.Duration = time.Duration(seconds) * time.Second

	dbc.setPipelinePropertiesFromJSONB(pipeline, labelsData, releaseTargetsData, commitsData, triggersData, optimized)

	if optimized {
		// clear some properties for reduced size and improved performance over the network
		pipeline.Manifest = ""
		pipeline.ManifestWithDefaults = ""
	}

	return
}

func (dbc *cockroachDBClientImpl) scanPipelines(rows *sql.Rows, optimized bool) (pipelines []*contracts.Pipeline, err error) {

	pipelines = make([]*contracts.Pipeline, 0)

	defer rows.Close()
	for rows.Next() {

		pipeline := contracts.Pipeline{}
		var labelsData, releaseTargetsData, commitsData, triggersData []uint8
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
			&triggersData,
			&pipeline.InsertedAt,
			&pipeline.UpdatedAt,
			&seconds,
			&pipeline.LastUpdatedAt); err != nil {
			return
		}

		pipeline.Duration = time.Duration(seconds) * time.Second

		dbc.setPipelinePropertiesFromJSONB(&pipeline, labelsData, releaseTargetsData, commitsData, triggersData, optimized)

		if optimized {
			// clear some properties for reduced size and improved performance over the network
			pipeline.Manifest = ""
			pipeline.ManifestWithDefaults = ""
		}

		pipelines = append(pipelines, &pipeline)
	}

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

func (dbc *cockroachDBClientImpl) GetTriggers(triggerType, identifier, event string) (pipelines []*contracts.Pipeline, err error) {

	// generate query
	query := dbc.selectPipelinesQuery()

	trigger := manifest.EstafetteTrigger{}

	switch triggerType {
	case "pipeline":

		trigger.Pipeline = &manifest.EstafettePipelineTrigger{
			Event: event,
			Name:  identifier,
		}

		break

	case "release":

		trigger.Release = &manifest.EstafetteReleaseTrigger{
			Event: event,
			Name:  identifier,
		}

		break

	case "cron":

		trigger.Cron = &manifest.EstafetteCronTrigger{}

		break
	}

	bytes, err := json.Marshal([]manifest.EstafetteTrigger{trigger})
	if err != nil {
		return pipelines, err
	}

	query = query.Where("a.triggers @> ?", string(bytes))

	// execute query
	rows, err := query.RunWith(dbc.databaseConnection).Query()
	if err != nil {
		return
	}

	// read rows
	if pipelines, err = dbc.scanPipelines(rows, false); err != nil {
		return
	}

	return
}

func (dbc *cockroachDBClientImpl) GetPipelineTriggers(build contracts.Build, event string) ([]*contracts.Pipeline, error) {

	triggerType := "pipeline"
	name := fmt.Sprintf("%v/%v/%v", build.RepoSource, build.RepoOwner, build.RepoName)

	return dbc.GetTriggers(triggerType, name, event)
}

func (dbc *cockroachDBClientImpl) GetReleaseTriggers(release contracts.Release, event string) ([]*contracts.Pipeline, error) {

	triggerType := "release"
	name := fmt.Sprintf("%v/%v/%v", release.RepoSource, release.RepoOwner, release.RepoName)

	return dbc.GetTriggers(triggerType, name, event)
}

func (dbc *cockroachDBClientImpl) GetCronTriggers() ([]*contracts.Pipeline, error) {

	triggerType := "cron"

	return dbc.GetTriggers(triggerType, "", "")
}

func (dbc *cockroachDBClientImpl) selectBuildsQuery() sq.SelectBuilder {
	psql := sq.StatementBuilder.PlaceholderFormat(sq.Dollar)

	return psql.
		Select("a.id, a.repo_source, a.repo_owner, a.repo_name, a.repo_branch, a.repo_revision, a.build_version, a.build_status, a.labels, a.release_targets, a.manifest, a.commits, a.triggers, a.inserted_at, a.updated_at, a.duration::INT").
		From("builds a")
}

func (dbc *cockroachDBClientImpl) selectPipelinesQuery() sq.SelectBuilder {
	psql := sq.StatementBuilder.PlaceholderFormat(sq.Dollar)

	return psql.
		Select("a.pipeline_id, a.repo_source, a.repo_owner, a.repo_name, a.repo_branch, a.repo_revision, a.build_version, a.build_status, a.labels, a.release_targets, a.manifest, a.commits, a.triggers, a.inserted_at, a.updated_at, a.duration::INT, a.last_updated_at").
		From("computed_pipelines a")
}

func (dbc *cockroachDBClientImpl) selectReleasesQuery() sq.SelectBuilder {
	psql := sq.StatementBuilder.PlaceholderFormat(sq.Dollar)

	return psql.
		Select("a.id, a.repo_source, a.repo_owner, a.repo_name, a.release, a.release_action, a.release_version, a.release_status, a.triggered_by, a.inserted_at, a.updated_at, a.duration::INT").
		From("releases a")
}

func (dbc *cockroachDBClientImpl) selectComputedReleasesQuery() sq.SelectBuilder {
	psql := sq.StatementBuilder.PlaceholderFormat(sq.Dollar)

	return psql.
		Select("a.release_id, a.repo_source, a.repo_owner, a.repo_name, a.release, a.release_action, a.release_version, a.release_status, a.triggered_by, a.inserted_at, a.updated_at, a.duration::INT").
		From("computed_releases a")
}

func (dbc *cockroachDBClientImpl) selectBuildLogsQuery() sq.SelectBuilder {
	psql := sq.StatementBuilder.PlaceholderFormat(sq.Dollar)

	return psql.
		Select("a.id, a.repo_source, a.repo_owner, a.repo_name, a.repo_branch, a.repo_revision, a.build_id, a.steps, a.inserted_at").
		From("build_logs a")
}

func (dbc *cockroachDBClientImpl) selectReleaseLogsQuery() sq.SelectBuilder {
	psql := sq.StatementBuilder.PlaceholderFormat(sq.Dollar)

	return psql.
		Select("a.id, a.repo_source, a.repo_owner, a.repo_name, a.release_id, a.steps, a.inserted_at").
		From("release_logs a")
}

func (dbc *cockroachDBClientImpl) selectPipelineTriggersQuery() sq.SelectBuilder {
	psql := sq.StatementBuilder.PlaceholderFormat(sq.Dollar)

	return psql.
		Select("a.id, a.repo_source, a.repo_owner, a.repo_name, a.trigger_event, a.trigger_filter, a.trigger_run, a.inserted_at, a.updated_at").
		From("pipeline_triggers a")
}

func (dbc *cockroachDBClientImpl) enrichPipeline(pipeline *contracts.Pipeline) {
	dbc.getLatestReleasesForPipeline(pipeline)
}

func (dbc *cockroachDBClientImpl) enrichBuild(build *contracts.Build) {
	dbc.getLatestReleasesForBuild(build)
}

func (dbc *cockroachDBClientImpl) setPipelinePropertiesFromJSONB(pipeline *contracts.Pipeline, labelsData, releaseTargetsData, commitsData, triggersData []uint8, optimized bool) (err error) {

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
	if len(triggersData) > 0 {
		if err = json.Unmarshal(triggersData, &pipeline.Triggers); err != nil {
			return
		}
	}

	if !optimized {
		// unmarshal then marshal manifest to include defaults
		var manifest manifest.EstafetteManifest
		err = yaml.Unmarshal([]byte(pipeline.Manifest), &manifest)
		if err == nil {
			pipeline.ManifestObject = &manifest
			manifestWithDefaultBytes, err := yaml.Marshal(manifest)
			if err == nil {
				pipeline.ManifestWithDefaults = string(manifestWithDefaultBytes)
			} else {
				log.Warn().Err(err).Interface("manifest", manifest).Msgf("Marshalling manifest for %v/%v/%v revision %v failed", pipeline.RepoSource, pipeline.RepoOwner, pipeline.RepoName, pipeline.RepoRevision)
			}
		} else {
			log.Warn().Err(err).Str("manifest", pipeline.Manifest).Msgf("Unmarshalling manifest for %v/%v/%v revision %v failed", pipeline.RepoSource, pipeline.RepoOwner, pipeline.RepoName, pipeline.RepoRevision)
		}
	}

	return
}

func (dbc *cockroachDBClientImpl) setBuildPropertiesFromJSONB(build *contracts.Build, labelsData, releaseTargetsData, commitsData, triggersData []uint8, optimized bool) (err error) {

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
	if len(triggersData) > 0 {
		if err = json.Unmarshal(triggersData, &build.Triggers); err != nil {
			return
		}
	}

	if !optimized {
		// unmarshal then marshal manifest to include defaults
		var manifest manifest.EstafetteManifest
		err = yaml.Unmarshal([]byte(build.Manifest), &manifest)
		if err == nil {
			build.ManifestObject = &manifest
			manifestWithDefaultBytes, err := yaml.Marshal(manifest)
			if err == nil {
				build.ManifestWithDefaults = string(manifestWithDefaultBytes)
			} else {
				log.Warn().Err(err).Interface("manifest", manifest).Msgf("Marshalling manifest for %v/%v/%v revision %v failed", build.RepoSource, build.RepoOwner, build.RepoName, build.RepoRevision)
			}
		} else {
			log.Warn().Err(err).Str("manifest", build.Manifest).Msgf("Unmarshalling manifest for %v/%v/%v revision %v failed", build.RepoSource, build.RepoOwner, build.RepoName, build.RepoRevision)
		}
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
		Triggers:             build.Triggers,
		InsertedAt:           build.InsertedAt,
		UpdatedAt:            build.UpdatedAt,
		Duration:             build.Duration,
	}
}
