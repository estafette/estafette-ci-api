package cockroachdb

import (
	"context"
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
	contracts "github.com/estafette/estafette-ci-contracts"
	manifest "github.com/estafette/estafette-ci-manifest"
	_ "github.com/lib/pq" // use postgres client library to connect to cockroachdb
	"github.com/rs/zerolog/log"
	yaml "gopkg.in/yaml.v2"
)

// Client is the interface for communicating with CockroachDB
type Client interface {
	Connect(ctx context.Context) (err error)
	ConnectWithDriverAndSource(ctx context.Context, driverName, dataSourceName string) (err error)

	GetAutoIncrement(ctx context.Context, shortRepoSource, repoOwner, repoName string) (autoincrement int, err error)
	InsertBuild(ctx context.Context, build contracts.Build, jobResources JobResources) (b *contracts.Build, err error)
	UpdateBuildStatus(ctx context.Context, repoSource, repoOwner, repoName string, buildID int, buildStatus string) (err error)
	UpdateBuildResourceUtilization(ctx context.Context, repoSource, repoOwner, repoName string, buildID int, jobResources JobResources) (err error)
	InsertRelease(ctx context.Context, release contracts.Release, jobResources JobResources) (r *contracts.Release, err error)
	UpdateReleaseStatus(ctx context.Context, repoSource, repoOwner, repoName string, id int, releaseStatus string) (err error)
	UpdateReleaseResourceUtilization(ctx context.Context, repoSource, repoOwner, repoName string, id int, jobResources JobResources) (err error)
	InsertBuildLog(ctx context.Context, buildLog contracts.BuildLog, writeLogToDatabase bool) (log contracts.BuildLog, err error)
	InsertReleaseLog(ctx context.Context, releaseLog contracts.ReleaseLog, writeLogToDatabase bool) (log contracts.ReleaseLog, err error)

	UpsertComputedPipeline(ctx context.Context, repoSource, repoOwner, repoName string) (err error)
	UpdateComputedPipelineFirstInsertedAt(ctx context.Context, repoSource, repoOwner, repoName string) (err error)
	UpsertComputedRelease(ctx context.Context, repoSource, repoOwner, repoName, releaseName, releaseAction string) (err error)
	UpdateComputedReleaseFirstInsertedAt(ctx context.Context, repoSource, repoOwner, repoName, releaseName, releaseAction string) (err error)

	GetPipelines(ctx context.Context, pageNumber, pageSize int, filters map[string][]string, optimized bool) (pipelines []*contracts.Pipeline, err error)
	GetPipelinesByRepoName(ctx context.Context, repoName string, optimized bool) (pipelines []*contracts.Pipeline, err error)
	GetPipelinesCount(ctx context.Context, filters map[string][]string) (count int, err error)
	GetPipeline(ctx context.Context, repoSource, repoOwner, repoName string, optimized bool) (pipeline *contracts.Pipeline, err error)
	GetPipelineRecentBuilds(ctx context.Context, repoSource, repoOwner, repoName string, optimized bool) (builds []*contracts.Build, err error)
	GetPipelineBuilds(ctx context.Context, repoSource, repoOwner, repoName string, pageNumber, pageSize int, filters map[string][]string, optimized bool) (builds []*contracts.Build, err error)
	GetPipelineBuildsCount(ctx context.Context, repoSource, repoOwner, repoName string, filters map[string][]string) (count int, err error)
	GetPipelineBuild(ctx context.Context, repoSource, repoOwner, repoName, repoRevision string, optimized bool) (build *contracts.Build, err error)
	GetPipelineBuildByID(ctx context.Context, repoSource, repoOwner, repoName string, id int, optimized bool) (build *contracts.Build, err error)
	GetLastPipelineBuild(ctx context.Context, repoSource, repoOwner, repoName string, optimized bool) (build *contracts.Build, err error)
	GetFirstPipelineBuild(ctx context.Context, repoSource, repoOwner, repoName string, optimized bool) (build *contracts.Build, err error)
	GetLastPipelineBuildForBranch(ctx context.Context, repoSource, repoOwner, repoName, branch string) (build *contracts.Build, err error)
	GetLastPipelineRelease(ctx context.Context, repoSource, repoOwner, repoName, releaseName, releaseAction string) (release *contracts.Release, err error)
	GetFirstPipelineRelease(ctx context.Context, repoSource, repoOwner, repoName, releaseName, releaseAction string) (release *contracts.Release, err error)
	GetPipelineBuildsByVersion(ctx context.Context, repoSource, repoOwner, repoName, buildVersion string, statuses []string, limit uint64, optimized bool) (builds []*contracts.Build, err error)
	GetPipelineBuildLogs(ctx context.Context, repoSource, repoOwner, repoName, repoBranch, repoRevision, buildID string, readLogFromDatabase bool) (buildlog *contracts.BuildLog, err error)
	GetPipelineBuildLogsPerPage(ctx context.Context, repoSource, repoOwner, repoName string, pageNumber int, pageSize int) (buildLogs []*contracts.BuildLog, err error)
	GetPipelineBuildMaxResourceUtilization(ctx context.Context, repoSource, repoOwner, repoName string, lastNRecords int) (jobresources JobResources, count int, err error)
	GetPipelineReleases(ctx context.Context, repoSource, repoOwner, repoName string, pageNumber, pageSize int, filters map[string][]string) (releases []*contracts.Release, err error)
	GetPipelineReleasesCount(ctx context.Context, repoSource, repoOwner, repoName string, filters map[string][]string) (count int, err error)
	GetPipelineRelease(ctx context.Context, repoSource, repoOwner, repoName string, id int) (release *contracts.Release, err error)
	GetPipelineLastReleasesByName(ctx context.Context, repoSource, repoOwner, repoName, releaseName string, actions []string) (releases []contracts.Release, err error)
	GetPipelineReleaseLogs(ctx context.Context, repoSource, repoOwner, repoName string, id int, readLogFromDatabase bool) (releaselog *contracts.ReleaseLog, err error)
	GetPipelineReleaseLogsPerPage(ctx context.Context, repoSource, repoOwner, repoName string, pageNumber int, pageSize int) (releaselogs []*contracts.ReleaseLog, err error)
	GetPipelineReleaseMaxResourceUtilization(ctx context.Context, repoSource, repoOwner, repoName, targetName string, lastNRecords int) (jobresources JobResources, count int, err error)
	GetBuildsCount(ctx context.Context, filters map[string][]string) (count int, err error)
	GetReleasesCount(ctx context.Context, filters map[string][]string) (count int, err error)
	GetBuildsDuration(ctx context.Context, filters map[string][]string) (duration time.Duration, err error)
	GetFirstBuildTimes(ctx context.Context) (times []time.Time, err error)
	GetFirstReleaseTimes(ctx context.Context) (times []time.Time, err error)
	GetPipelineBuildsDurations(ctx context.Context, repoSource, repoOwner, repoName string, filters map[string][]string) (durations []map[string]interface{}, err error)
	GetPipelineReleasesDurations(ctx context.Context, repoSource, repoOwner, repoName string, filters map[string][]string) (durations []map[string]interface{}, err error)
	GetPipelineBuildsCPUUsageMeasurements(ctx context.Context, repoSource, repoOwner, repoName string, filters map[string][]string) (measurements []map[string]interface{}, err error)
	GetPipelineReleasesCPUUsageMeasurements(ctx context.Context, repoSource, repoOwner, repoName string, filters map[string][]string) (measurements []map[string]interface{}, err error)
	GetPipelineBuildsMemoryUsageMeasurements(ctx context.Context, repoSource, repoOwner, repoName string, filters map[string][]string) (measurements []map[string]interface{}, err error)
	GetPipelineReleasesMemoryUsageMeasurements(ctx context.Context, repoSource, repoOwner, repoName string, filters map[string][]string) (measurements []map[string]interface{}, err error)

	GetLabelValues(ctx context.Context, labelKey string) (labels []map[string]interface{}, err error)
	GetFrequentLabels(ctx context.Context, pageNumber, pageSize int, filters map[string][]string) (labels []map[string]interface{}, err error)
	GetFrequentLabelsCount(ctx context.Context, filters map[string][]string) (count int, err error)

	GetPipelinesWithMostBuilds(ctx context.Context, pageNumber, pageSize int, filters map[string][]string) (pipelines []map[string]interface{}, err error)
	GetPipelinesWithMostBuildsCount(ctx context.Context, filters map[string][]string) (count int, err error)
	GetPipelinesWithMostReleases(ctx context.Context, pageNumber, pageSize int, filters map[string][]string) (pipelines []map[string]interface{}, err error)
	GetPipelinesWithMostReleasesCount(ctx context.Context, filters map[string][]string) (count int, err error)

	GetTriggers(ctx context.Context, triggerType, identifier, event string) (pipelines []*contracts.Pipeline, err error)
	GetGitTriggers(ctx context.Context, gitEvent manifest.EstafetteGitEvent) (pipelines []*contracts.Pipeline, err error)
	GetPipelineTriggers(ctx context.Context, build contracts.Build, event string) (pipelines []*contracts.Pipeline, err error)
	GetReleaseTriggers(ctx context.Context, release contracts.Release, event string) (pipelines []*contracts.Pipeline, err error)
	GetPubSubTriggers(ctx context.Context, pubsubEvent manifest.EstafettePubSubEvent) (pipelines []*contracts.Pipeline, err error)
	GetCronTriggers(ctx context.Context) (pipelines []*contracts.Pipeline, err error)

	Rename(ctx context.Context, shortFromRepoSource, fromRepoSource, fromRepoOwner, fromRepoName, shortToRepoSource, toRepoSource, toRepoOwner, toRepoName string) (err error)
	RenameBuildVersion(ctx context.Context, shortFromRepoSource, fromRepoOwner, fromRepoName, shortToRepoSource, toRepoOwner, toRepoName string) (err error)
	RenameBuilds(ctx context.Context, fromRepoSource, fromRepoOwner, fromRepoName, toRepoSource, toRepoOwner, toRepoName string) (err error)
	RenameBuildLogs(ctx context.Context, fromRepoSource, fromRepoOwner, fromRepoName, toRepoSource, toRepoOwner, toRepoName string) (err error)
	RenameReleases(ctx context.Context, fromRepoSource, fromRepoOwner, fromRepoName, toRepoSource, toRepoOwner, toRepoName string) (err error)
	RenameReleaseLogs(ctx context.Context, fromRepoSource, fromRepoOwner, fromRepoName, toRepoSource, toRepoOwner, toRepoName string) (err error)
	RenameComputedPipelines(ctx context.Context, fromRepoSource, fromRepoOwner, fromRepoName, toRepoSource, toRepoOwner, toRepoName string) (err error)
	RenameComputedReleases(ctx context.Context, fromRepoSource, fromRepoOwner, fromRepoName, toRepoSource, toRepoOwner, toRepoName string) (err error)
}

// NewClient returns a new cockroach.Client
func NewClient(config config.DatabaseConfig, manifestPreferences manifest.EstafetteManifestPreferences) Client {

	return &client{
		databaseDriver:      "postgres",
		config:              config,
		manifestPreferences: manifestPreferences,
	}
}

type client struct {
	databaseDriver      string
	config              config.DatabaseConfig
	manifestPreferences manifest.EstafetteManifestPreferences
	databaseConnection  *sql.DB
}

// Connect sets up a connection with CockroachDB
func (c *client) Connect(ctx context.Context) (err error) {

	log.Debug().Msgf("Connecting to database %v on host %v...", c.config.DatabaseName, c.config.Host)

	sslMode := ""
	if c.config.Insecure {
		sslMode = "?sslmode=disable"
	}

	dataSourceName := fmt.Sprintf("postgresql://%v:%v@%v:%v/%v%v", c.config.User, c.config.Password, c.config.Host, c.config.Port, c.config.DatabaseName, sslMode)

	return c.ConnectWithDriverAndSource(ctx, c.databaseDriver, dataSourceName)
}

// ConnectWithDriverAndSource set up a connection with any database
func (c *client) ConnectWithDriverAndSource(ctx context.Context, driverName, dataSourceName string) (err error) {

	c.databaseConnection, err = sql.Open(driverName, dataSourceName)
	if err != nil {
		return
	}

	return
}

// GetAutoIncrement returns the autoincrement number for a pipeline
func (c *client) GetAutoIncrement(ctx context.Context, shortRepoSource, repoOwner, repoName string) (autoincrement int, err error) {

	repoFullName := fmt.Sprintf("%v/%v", repoOwner, repoName)

	// insert or increment if record for repo_source and repo_full_name combination already exists
	_, err = c.databaseConnection.Exec(
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

	// fetching auto_increment value, because RETURNING is not supported with UPSERT / INSERT ON CONFLICT (see issue https://github.com/cockroachdb/cockroach/issues/6637)
	rows, err := c.databaseConnection.Query(
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

func (c *client) InsertBuild(ctx context.Context, build contracts.Build, jobResources JobResources) (insertedBuild *contracts.Build, err error) {

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
	eventsBytes, err := json.Marshal(build.Events)
	if err != nil {

		return
	}

	// insert logs
	row := c.databaseConnection.QueryRow(
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
			triggers,
			triggered_by_event,
			cpu_request,
			cpu_limit,
			memory_request,
			memory_limit
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
			$17
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
		eventsBytes,
		jobResources.CPURequest,
		jobResources.CPULimit,
		jobResources.MemoryRequest,
		jobResources.MemoryLimit,
	)

	insertedBuild = &build

	if err = row.Scan(&insertedBuild.ID); err != nil {

		return
	}

	// update computed tables
	go c.UpsertComputedPipeline(ctx, insertedBuild.RepoSource, insertedBuild.RepoOwner, insertedBuild.RepoName)

	return
}

func (c *client) UpdateBuildStatus(ctx context.Context, repoSource, repoOwner, repoName string, buildID int, buildStatus string) (err error) {

	allowedBuildStatusesToTransitionFrom := []string{}
	switch buildStatus {
	case "running":
		allowedBuildStatusesToTransitionFrom = []string{"pending"}
		break
	case "succeeded",
		"failed",
		"canceling":
		allowedBuildStatusesToTransitionFrom = []string{"running"}
		break
	case "canceled":
		allowedBuildStatusesToTransitionFrom = []string{"pending", "canceling"}
		break
	}

	psql := sq.StatementBuilder.PlaceholderFormat(sq.Dollar)

	query := psql.
		Update("builds").
		Set("build_status", buildStatus).
		Set("updated_at", sq.Expr("now()")).
		Set("duration", sq.Expr("age(now(), inserted_at)")).
		Where(sq.Eq{"id": buildID}).
		Where(sq.Eq{"repo_source": repoSource}).
		Where(sq.Eq{"repo_owner": repoOwner}).
		Where(sq.Eq{"repo_name": repoName}).
		Where(sq.Eq{"build_status": allowedBuildStatusesToTransitionFrom}).
		Suffix("RETURNING id, repo_source, repo_owner, repo_name, repo_branch, repo_revision, build_version, build_status, labels, release_targets, manifest, commits, triggers, inserted_at, updated_at, duration::INT, triggered_by_event")

	if buildStatus == "running" {
		query = query.Set("time_to_running", sq.Expr("age(now(), inserted_at)"))
	}

	// update build status
	row := query.RunWith(c.databaseConnection).QueryRow()

	_, err = c.scanBuild(ctx, row, false, false)
	if err != nil && err != sql.ErrNoRows {

		return
	} else if err != nil {
		log.Warn().Err(err).Msgf("Updating build status for %v/%v/%v id %v from %v to %v is not allowed, no records have been updated", repoSource, repoOwner, repoName, buildStatus, allowedBuildStatusesToTransitionFrom, buildStatus)
		return
	}

	// update computed tables
	go c.UpsertComputedPipeline(ctx, repoSource, repoOwner, repoName)

	return
}

func (c *client) UpdateBuildResourceUtilization(ctx context.Context, repoSource, repoOwner, repoName string, buildID int, jobResources JobResources) (err error) {

	psql := sq.StatementBuilder.PlaceholderFormat(sq.Dollar)

	query := psql.
		Update("builds").
		Set("cpu_max_usage", jobResources.CPUMaxUsage).
		Set("memory_max_usage", jobResources.MemoryMaxUsage).
		Where(sq.Eq{"id": buildID}).
		Where(sq.Eq{"repo_source": repoSource}).
		Where(sq.Eq{"repo_owner": repoOwner}).
		Where(sq.Eq{"repo_name": repoName})

	// update build resources
	_, err = query.RunWith(c.databaseConnection).Exec()
	if err != nil {

		return
	}

	return
}

func (c *client) InsertRelease(ctx context.Context, release contracts.Release, jobResources JobResources) (insertedRelease *contracts.Release, err error) {

	eventsBytes, err := json.Marshal(release.Events)
	if err != nil {

		return
	}

	// insert logs
	rows, err := c.databaseConnection.Query(
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
			triggered_by_event,
			cpu_request,
			cpu_limit,
			memory_request,
			memory_limit
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
		release.RepoSource,
		release.RepoOwner,
		release.RepoName,
		release.Name,
		release.Action,
		release.ReleaseVersion,
		release.ReleaseStatus,
		eventsBytes,
		jobResources.CPURequest,
		jobResources.CPULimit,
		jobResources.MemoryRequest,
		jobResources.MemoryLimit,
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
		c.UpsertComputedRelease(ctx, insertedRelease.RepoSource, insertedRelease.RepoOwner, insertedRelease.RepoName, insertedRelease.Name, insertedRelease.Action)
		c.UpsertComputedPipeline(ctx, insertedRelease.RepoSource, insertedRelease.RepoOwner, insertedRelease.RepoName)
	}()

	return
}

func (c *client) UpdateReleaseStatus(ctx context.Context, repoSource, repoOwner, repoName string, id int, releaseStatus string) (err error) {

	allowedReleaseStatusesToTransitionFrom := []string{}
	switch releaseStatus {
	case "running":
		allowedReleaseStatusesToTransitionFrom = []string{"pending"}
		break
	case "succeeded",
		"failed",
		"canceling":
		allowedReleaseStatusesToTransitionFrom = []string{"running"}
		break
	case "canceled":
		allowedReleaseStatusesToTransitionFrom = []string{"pending", "canceling"}
		break
	}

	psql := sq.StatementBuilder.PlaceholderFormat(sq.Dollar)

	query := psql.
		Update("releases").
		Set("release_status", releaseStatus).
		Set("updated_at", sq.Expr("now()")).
		Set("duration", sq.Expr("age(now(), inserted_at)")).
		Where(sq.Eq{"id": id}).
		Where(sq.Eq{"repo_source": repoSource}).
		Where(sq.Eq{"repo_owner": repoOwner}).
		Where(sq.Eq{"repo_name": repoName}).
		Where(sq.Eq{"release_status": allowedReleaseStatusesToTransitionFrom}).
		Suffix("RETURNING id, repo_source, repo_owner, repo_name, release, release_action, release_version, release_status, inserted_at, updated_at, duration::INT, triggered_by_event")

	if releaseStatus == "running" {
		query = query.Set("time_to_running", sq.Expr("age(now(), inserted_at)"))
	}

	// update release status
	row := query.RunWith(c.databaseConnection).QueryRow()
	insertedRelease, err := c.scanRelease(row)
	if err != nil && err != sql.ErrNoRows {
		return
	} else if err != nil {
		log.Warn().Err(err).Msgf("Updating release status for %v/%v/%v id %v from %v to %v is not allowed, no records have been updated", repoSource, repoOwner, repoName, id, allowedReleaseStatusesToTransitionFrom, releaseStatus)
		return
	}

	// update computed tables
	go func() {
		if insertedRelease != nil {
			c.UpsertComputedRelease(ctx, repoSource, repoOwner, repoName, insertedRelease.Name, insertedRelease.Action)
		} else {
			log.Warn().Msgf("Cannot update computed tables after updating release status for %v/%v/%v id %v from %v to %v", repoSource, repoOwner, repoName, id, allowedReleaseStatusesToTransitionFrom, releaseStatus)
		}
		c.UpsertComputedPipeline(ctx, repoSource, repoOwner, repoName)
	}()

	return
}

func (c *client) UpdateReleaseResourceUtilization(ctx context.Context, repoSource, repoOwner, repoName string, id int, jobResources JobResources) (err error) {

	psql := sq.StatementBuilder.PlaceholderFormat(sq.Dollar)

	query := psql.
		Update("releases").
		Set("cpu_max_usage", jobResources.CPUMaxUsage).
		Set("memory_max_usage", jobResources.MemoryMaxUsage).
		Where(sq.Eq{"id": id}).
		Where(sq.Eq{"repo_source": repoSource}).
		Where(sq.Eq{"repo_owner": repoOwner}).
		Where(sq.Eq{"repo_name": repoName})

	// update release resources
	_, err = query.RunWith(c.databaseConnection).Exec()
	if err != nil {

		return
	}

	return
}

func (c *client) InsertBuildLog(ctx context.Context, buildLog contracts.BuildLog, writeLogToDatabase bool) (insertedBuildLog contracts.BuildLog, err error) {

	insertedBuildLog = buildLog

	var bytes []byte
	if writeLogToDatabase {
		bytes, err = json.Marshal(buildLog.Steps)
		if err != nil {

			return
		}
	} else {
		var steps []*contracts.BuildLogStep
		bytes, err = json.Marshal(steps)
	}

	buildID, err := strconv.Atoi(buildLog.BuildID)
	if err != nil {
		// insert logs
		row := c.databaseConnection.QueryRow(
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
			RETURNING
				id
			`,
			buildLog.RepoSource,
			buildLog.RepoOwner,
			buildLog.RepoName,
			buildLog.RepoBranch,
			buildLog.RepoRevision,
			bytes,
		)

		if err = row.Scan(&insertedBuildLog.ID); err != nil {
			// log extra detail for filing a ticket regarding 'pq: command is too large: xxx bytes (max: 67108864)' issue
			nrLines := 0
			for _, s := range buildLog.Steps {
				nrLines += len(s.LogLines)
			}
			log.Error().Msgf("INSERT INTO build_logs: failed for %v/%v/%v/%v (%v steps, %v lines, %v bytes)", buildLog.RepoSource, buildLog.RepoOwner, buildLog.RepoName, buildLog.RepoRevision, len(buildLog.Steps), nrLines, len(bytes))

			return
		}

		return
	}

	// insert logs
	row := c.databaseConnection.QueryRow(
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
		RETURNING
			id
		`,
		buildLog.RepoSource,
		buildLog.RepoOwner,
		buildLog.RepoName,
		buildLog.RepoBranch,
		buildLog.RepoRevision,
		buildID,
		bytes,
	)

	if err = row.Scan(&insertedBuildLog.ID); err != nil {
		// log extra detail for filing a ticket regarding 'pq: command is too large: xxx bytes (max: 67108864)' issue
		nrLines := 0
		for _, s := range buildLog.Steps {
			nrLines += len(s.LogLines)
		}
		log.Error().Msgf("INSERT INTO build_logs: failed for %v/%v/%v/%v (%v steps, %v lines, %v bytes)", buildLog.RepoSource, buildLog.RepoOwner, buildLog.RepoName, buildLog.RepoRevision, len(buildLog.Steps), nrLines, len(bytes))

		return
	}

	return
}

func (c *client) InsertReleaseLog(ctx context.Context, releaseLog contracts.ReleaseLog, writeLogToDatabase bool) (insertedReleaseLog contracts.ReleaseLog, err error) {

	insertedReleaseLog = releaseLog

	var bytes []byte
	if writeLogToDatabase {
		bytes, err = json.Marshal(releaseLog.Steps)
		if err != nil {

			return
		}
	} else {
		var steps []*contracts.BuildLogStep
		bytes, err = json.Marshal(steps)
	}

	releaseID, err := strconv.Atoi(releaseLog.ReleaseID)
	if err != nil {

		return insertedReleaseLog, err
	}

	// insert logs
	row := c.databaseConnection.QueryRow(
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
		RETURNING
			id		
		`,
		releaseLog.RepoSource,
		releaseLog.RepoOwner,
		releaseLog.RepoName,
		releaseID,
		bytes,
	)

	if err = row.Scan(&insertedReleaseLog.ID); err != nil {
		// log extra detail for filing a ticket regarding 'pq: command is too large: xxx bytes (max: 67108864)' issue
		nrLines := 0
		for _, s := range releaseLog.Steps {
			nrLines += len(s.LogLines)
		}
		log.Error().Msgf("INSERT INTO build_logs: failed for %v/%v/%v/%v (%v steps, %v lines, %v bytes)", releaseLog.RepoSource, releaseLog.RepoOwner, releaseLog.RepoName, releaseLog.ReleaseID, len(releaseLog.Steps), nrLines, len(bytes))

		return
	}

	return
}

func (c *client) UpsertComputedPipeline(ctx context.Context, repoSource, repoOwner, repoName string) (err error) {

	// get last build
	lastBuild, err := c.GetLastPipelineBuild(ctx, repoSource, repoOwner, repoName, false)
	if err != nil {
		log.Error().Err(err).Msgf("Failed getting last build for upserting computed pipeline %v/%v/%v", repoSource, repoOwner, repoName)

		return
	}
	if lastBuild == nil {
		log.Error().Msgf("Failed getting last build for upserting computed pipeline %v/%v/%v", repoSource, repoOwner, repoName)
		return
	}

	upsertedPipeline := c.mapBuildToPipeline(lastBuild)

	// add releases
	c.enrichPipeline(ctx, upsertedPipeline)

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
	eventsBytes, err := json.Marshal(upsertedPipeline.Events)
	if err != nil {
		log.Error().Err(err).Msgf("Failed upserting computed pipeline %v/%v/%v", upsertedPipeline.RepoSource, upsertedPipeline.RepoOwner, upsertedPipeline.RepoName)

		return
	}

	// upsert computed pipeline
	_, err = c.databaseConnection.Exec(
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
			archived,
			inserted_at,
			first_inserted_at,
			updated_at,
			duration,
			last_updated_at,
			triggered_by_event
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
			$17,
			AGE($17,$16),
			$18,
			$19
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
			archived = excluded.archived,
			inserted_at = excluded.inserted_at,
			updated_at = excluded.updated_at,
			duration = AGE(excluded.updated_at,excluded.inserted_at),
			last_updated_at = excluded.last_updated_at,
			triggered_by_event = excluded.triggered_by_event
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
		upsertedPipeline.Archived,
		upsertedPipeline.InsertedAt,
		upsertedPipeline.InsertedAt,
		upsertedPipeline.UpdatedAt,
		upsertedPipeline.LastUpdatedAt,
		eventsBytes,
	)
	if err != nil {
		log.Error().Err(err).Msgf("Failed upserting computed pipeline %v/%v/%v", repoSource, repoOwner, repoName)

		return
	}

	return
}

func (c *client) UpdateComputedPipelineFirstInsertedAt(ctx context.Context, repoSource, repoOwner, repoName string) (err error) {

	// get first build
	firstBuild, err := c.GetFirstPipelineBuild(ctx, repoSource, repoOwner, repoName, false)
	if err != nil {
		log.Error().Err(err).Msgf("Failed getting first build for updating computed pipeline %v/%v/%v", repoSource, repoOwner, repoName)

		return
	}
	if firstBuild == nil {
		log.Error().Msgf("Failed getting first build for updating computed pipeline %v/%v/%v", repoSource, repoOwner, repoName)
		return
	}

	updatedPipeline := c.mapBuildToPipeline(firstBuild)

	// update computed pipeline
	_, err = c.databaseConnection.Exec(
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

func (c *client) UpsertComputedRelease(ctx context.Context, repoSource, repoOwner, repoName, releaseName, releaseAction string) (err error) {

	// get last release
	lastRelease, err := c.GetLastPipelineRelease(ctx, repoSource, repoOwner, repoName, releaseName, releaseAction)
	if err != nil {
		log.Error().Err(err).Msgf("Failed getting last release for upserting computed release %v/%v/%v/%v/%v", repoSource, repoOwner, repoName, releaseName, releaseAction)

		return
	}
	if lastRelease == nil {
		log.Error().Msgf("Failed getting last release for upserting computed release %v/%v/%v/%v/%v", repoSource, repoOwner, repoName, releaseName, releaseAction)
		return
	}

	eventsBytes, err := json.Marshal(lastRelease.Events)
	if err != nil {

		return
	}

	// upsert computed release
	_, err = c.databaseConnection.Exec(
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
			duration,
			release_action,
			triggered_by_event
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
			AGE($9,$8),
			$10,
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
			duration = AGE(excluded.updated_at,excluded.inserted_at),
			triggered_by_event = excluded.triggered_by_event
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
		lastRelease.Action,
		eventsBytes,
	)
	if err != nil {
		log.Error().Err(err).Msgf("Failed upserting computed release %v/%v/%v/%v/%v", repoSource, repoOwner, repoName, releaseName, releaseAction)

		return
	}

	return
}

func (c *client) UpdateComputedReleaseFirstInsertedAt(ctx context.Context, repoSource, repoOwner, repoName, releaseName, releaseAction string) (err error) {

	// get first release
	firstRelease, err := c.GetFirstPipelineRelease(ctx, repoSource, repoOwner, repoName, releaseName, releaseAction)
	if err != nil {
		log.Error().Err(err).Msgf("Failed getting first release for updating computed release %v/%v/%v/%v/%v", repoSource, repoOwner, repoName, releaseName, releaseAction)

		return
	}
	if firstRelease == nil {
		log.Error().Msgf("Failed getting first release for updating computed release %v/%v/%v/%v/%v", repoSource, repoOwner, repoName, releaseName, releaseAction)
		return
	}

	// update computed release
	_, err = c.databaseConnection.Exec(
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

func (c *client) GetPipelines(ctx context.Context, pageNumber, pageSize int, filters map[string][]string, optimized bool) (pipelines []*contracts.Pipeline, err error) {

	// generate query
	query := c.selectPipelinesQuery().
		OrderBy("a.repo_source,a.repo_owner,a.repo_name").
		Where(sq.Eq{"a.archived": false}).
		Limit(uint64(pageSize)).
		Offset(uint64((pageNumber - 1) * pageSize))

	// dynamically set where clauses for filtering
	query, err = whereClauseGeneratorForAllFilters(query, "a", "last_updated_at", filters)
	if err != nil {

		return
	}

	// execute query
	rows, err := query.RunWith(c.databaseConnection).Query()
	if err != nil {

		return
	}

	// read rows
	if pipelines, err = c.scanPipelines(rows, optimized); err != nil {

		return
	}

	return
}

func (c *client) GetPipelinesByRepoName(ctx context.Context, repoName string, optimized bool) (pipelines []*contracts.Pipeline, err error) {

	// generate query
	query := c.selectPipelinesQuery().
		Where(sq.Eq{"a.repo_name": repoName})

	// execute query
	rows, err := query.RunWith(c.databaseConnection).Query()
	if err != nil {

		return
	}

	// read rows
	if pipelines, err = c.scanPipelines(rows, optimized); err != nil {

		return
	}

	return
}

func (c *client) GetPipelinesCount(ctx context.Context, filters map[string][]string) (totalCount int, err error) {

	// generate query
	query :=
		sq.StatementBuilder.PlaceholderFormat(sq.Dollar).
			Select("COUNT(a.id)").
			From("computed_pipelines a").
			Where(sq.Eq{"a.archived": false})

	// dynamically set where clauses for filtering
	query, err = whereClauseGeneratorForAllFilters(query, "a", "last_updated_at", filters)
	if err != nil {

		return
	}

	// execute query
	row := query.RunWith(c.databaseConnection).QueryRow()
	if err = row.Scan(&totalCount); err != nil {

		return
	}

	return
}

func (c *client) GetPipeline(ctx context.Context, repoSource, repoOwner, repoName string, optimized bool) (pipeline *contracts.Pipeline, err error) {

	// generate query
	query := c.selectPipelinesQuery().
		Where(sq.Eq{"a.repo_source": repoSource}).
		Where(sq.Eq{"a.repo_owner": repoOwner}).
		Where(sq.Eq{"a.repo_name": repoName}).
		OrderBy("a.inserted_at DESC").
		Limit(uint64(1))

	// execute query
	row := query.RunWith(c.databaseConnection).QueryRow()
	if pipeline, err = c.scanPipeline(row, optimized); err != nil {

		return
	}

	return
}

func (c *client) GetPipelineRecentBuilds(ctx context.Context, repoSource, repoOwner, repoName string, optimized bool) (builds []*contracts.Build, err error) {

	psql := sq.StatementBuilder.PlaceholderFormat(sq.Dollar)

	innerquery := psql.
		Select("a.*, ROW_NUMBER() OVER (PARTITION BY a.repo_branch ORDER BY a.inserted_at DESC) AS rn").
		From("builds a").
		Where(sq.Eq{"a.repo_source": repoSource}).
		Where(sq.Eq{"a.repo_owner": repoOwner}).
		Where(sq.Eq{"a.repo_name": repoName}).
		OrderBy("a.inserted_at DESC").
		Limit(uint64(50))

	query := psql.
		Select("a.id, a.repo_source, a.repo_owner, a.repo_name, a.repo_branch, a.repo_revision, a.build_version, a.build_status, a.labels, a.release_targets, a.manifest, a.commits, a.triggers, a.inserted_at, a.updated_at, a.duration::INT, a.triggered_by_event").
		Prefix("WITH ranked_builds AS (?)", innerquery).
		From("ranked_builds a").
		Where("a.rn = 1").
		OrderBy("a.inserted_at DESC").
		Limit(uint64(5))

	// execute query
	rows, err := query.RunWith(c.databaseConnection).Query()
	if err != nil {

		return
	}

	// read rows
	if builds, err = c.scanBuilds(rows, optimized); err != nil {

		return
	}

	return
}

func (c *client) GetPipelineBuilds(ctx context.Context, repoSource, repoOwner, repoName string, pageNumber, pageSize int, filters map[string][]string, optimized bool) (builds []*contracts.Build, err error) {

	// generate query
	query := c.selectBuildsQuery().
		Where(sq.Eq{"a.repo_source": repoSource}).
		Where(sq.Eq{"a.repo_owner": repoOwner}).
		Where(sq.Eq{"a.repo_name": repoName}).
		OrderBy("a.inserted_at DESC").
		Limit(uint64(pageSize)).
		Offset(uint64((pageNumber - 1) * pageSize))

	// dynamically set where clauses for filtering
	query, err = whereClauseGeneratorForAllFilters(query, "a", "inserted_at", filters)
	if err != nil {

		return
	}

	// execute query
	rows, err := query.RunWith(c.databaseConnection).Query()
	if err != nil {

		return
	}

	// read rows
	if builds, err = c.scanBuilds(rows, optimized); err != nil {

		return
	}

	return
}

func (c *client) GetPipelineBuildsCount(ctx context.Context, repoSource, repoOwner, repoName string, filters map[string][]string) (totalCount int, err error) {

	// generate query
	query :=
		sq.StatementBuilder.PlaceholderFormat(sq.Dollar).
			Select("COUNT(*)").
			From("builds a").
			Where(sq.Eq{"a.repo_source": repoSource}).
			Where(sq.Eq{"a.repo_owner": repoOwner}).
			Where(sq.Eq{"a.repo_name": repoName})

	// dynamically set where clauses for filtering
	query, err = whereClauseGeneratorForAllFilters(query, "a", "inserted_at", filters)
	if err != nil {

		return
	}

	// execute query
	row := query.RunWith(c.databaseConnection).QueryRow()
	if err = row.Scan(&totalCount); err != nil {

		return
	}

	return
}

func (c *client) GetPipelineBuild(ctx context.Context, repoSource, repoOwner, repoName, repoRevision string, optimized bool) (build *contracts.Build, err error) {

	// generate query
	query := c.selectBuildsQuery().
		Where(sq.Eq{"a.repo_source": repoSource}).
		Where(sq.Eq{"a.repo_owner": repoOwner}).
		Where(sq.Eq{"a.repo_name": repoName}).
		Where(sq.Eq{"a.repo_revision": repoRevision}).
		OrderBy("a.inserted_at DESC").
		Limit(uint64(1))

	// execute query
	row := query.RunWith(c.databaseConnection).QueryRow()
	if build, err = c.scanBuild(ctx, row, optimized, true); err != nil {

		return
	}

	return
}

func (c *client) GetPipelineBuildByID(ctx context.Context, repoSource, repoOwner, repoName string, id int, optimized bool) (build *contracts.Build, err error) {

	// generate query
	query := c.selectBuildsQuery().
		Where(sq.Eq{"a.id": id}).
		Where(sq.Eq{"a.repo_source": repoSource}).
		Where(sq.Eq{"a.repo_owner": repoOwner}).
		Where(sq.Eq{"a.repo_name": repoName}).
		OrderBy("a.inserted_at DESC").
		Limit(uint64(1))

	// execute query
	row := query.RunWith(c.databaseConnection).QueryRow()
	if build, err = c.scanBuild(ctx, row, optimized, true); err != nil {

		return
	}

	return
}

func (c *client) GetLastPipelineBuild(ctx context.Context, repoSource, repoOwner, repoName string, optimized bool) (build *contracts.Build, err error) {

	// generate query
	query := c.selectBuildsQuery().
		Where(sq.Eq{"a.repo_source": repoSource}).
		Where(sq.Eq{"a.repo_owner": repoOwner}).
		Where(sq.Eq{"a.repo_name": repoName}).
		OrderBy("a.inserted_at DESC").
		Limit(uint64(1))

	// execute query
	row := query.RunWith(c.databaseConnection).QueryRow()
	if build, err = c.scanBuild(ctx, row, optimized, false); err != nil {

		return
	}

	return
}

func (c *client) GetFirstPipelineBuild(ctx context.Context, repoSource, repoOwner, repoName string, optimized bool) (build *contracts.Build, err error) {

	// generate query
	query := c.selectBuildsQuery().
		Where(sq.Eq{"a.repo_source": repoSource}).
		Where(sq.Eq{"a.repo_owner": repoOwner}).
		Where(sq.Eq{"a.repo_name": repoName}).
		OrderBy("a.inserted_at ASC").
		Limit(uint64(1))

	// execute query
	row := query.RunWith(c.databaseConnection).QueryRow()
	if build, err = c.scanBuild(ctx, row, optimized, false); err != nil {

		return
	}

	return
}

func (c *client) GetLastPipelineBuildForBranch(ctx context.Context, repoSource, repoOwner, repoName, branch string) (build *contracts.Build, err error) {

	// generate query
	query := c.selectBuildsQuery().
		Where(sq.Eq{"a.repo_source": repoSource}).
		Where(sq.Eq{"a.repo_owner": repoOwner}).
		Where(sq.Eq{"a.repo_name": repoName}).
		Where(sq.Eq{"a.repo_branch": branch}).
		OrderBy("a.inserted_at DESC").
		Limit(uint64(1))

	// execute query
	row := query.RunWith(c.databaseConnection).QueryRow()
	if build, err = c.scanBuild(ctx, row, false, false); err != nil {

		return
	}

	return
}

func (c *client) GetLastPipelineRelease(ctx context.Context, repoSource, repoOwner, repoName, releaseName, releaseAction string) (release *contracts.Release, err error) {

	// generate query
	query := c.selectReleasesQuery().
		Where(sq.Eq{"a.repo_source": repoSource}).
		Where(sq.Eq{"a.repo_owner": repoOwner}).
		Where(sq.Eq{"a.repo_name": repoName}).
		Where(sq.Eq{"a.release": releaseName}).
		Where(sq.Eq{"a.release_action": releaseAction}).
		OrderBy("a.inserted_at DESC").
		Limit(uint64(1))

	// execute query
	row := query.RunWith(c.databaseConnection).QueryRow()
	if release, err = c.scanRelease(row); err != nil {

		return
	}

	return
}

func (c *client) GetFirstPipelineRelease(ctx context.Context, repoSource, repoOwner, repoName, releaseName, releaseAction string) (release *contracts.Release, err error) {

	// generate query
	query := c.selectReleasesQuery().
		Where(sq.Eq{"a.repo_source": repoSource}).
		Where(sq.Eq{"a.repo_owner": repoOwner}).
		Where(sq.Eq{"a.repo_name": repoName}).
		Where(sq.Eq{"a.release": releaseName}).
		Where(sq.Eq{"a.release_action": releaseAction}).
		OrderBy("a.inserted_at ASC").
		Limit(uint64(1))

	// execute query
	row := query.RunWith(c.databaseConnection).QueryRow()
	if release, err = c.scanRelease(row); err != nil {

		return
	}

	return
}

func (c *client) GetPipelineBuildsByVersion(ctx context.Context, repoSource, repoOwner, repoName, buildVersion string, statuses []string, limit uint64, optimized bool) (builds []*contracts.Build, err error) {

	// generate query
	query := c.selectBuildsQuery().
		Where(sq.Eq{"a.repo_source": repoSource}).
		Where(sq.Eq{"a.repo_owner": repoOwner}).
		Where(sq.Eq{"a.repo_name": repoName}).
		Where(sq.Eq{"a.build_version": buildVersion}).
		Where(sq.Eq{"a.build_status": statuses}).
		OrderBy("a.inserted_at DESC").
		Limit(limit)

	// execute query
	rows, err := query.RunWith(c.databaseConnection).Query()
	if err != nil {

		return
	}

	// read rows
	if builds, err = c.scanBuilds(rows, optimized); err != nil {

		return
	}

	return
}

func (c *client) GetPipelineBuildLogs(ctx context.Context, repoSource, repoOwner, repoName, repoBranch, repoRevision, buildID string, readLogFromDatabase bool) (buildLog *contracts.BuildLog, err error) {

	buildIDAsInt, err := strconv.Atoi(buildID)
	if err != nil {

		return nil, err
	}

	// generate query
	query := c.selectBuildLogsQuery(readLogFromDatabase).
		Where(sq.Eq{"a.build_id": buildIDAsInt}).
		Where(sq.Eq{"a.repo_source": repoSource}).
		Where(sq.Eq{"a.repo_owner": repoOwner}).
		Where(sq.Eq{"a.repo_name": repoName}).
		Where(sq.Eq{"a.repo_branch": repoBranch}).
		Where(sq.Eq{"a.repo_revision": repoRevision}).
		OrderBy("a.inserted_at DESC").
		Limit(uint64(1))

	buildLog = &contracts.BuildLog{}
	var rowBuildID sql.NullInt64

	// execute query
	row := query.RunWith(c.databaseConnection).QueryRow()
	if readLogFromDatabase {

		var stepsData []uint8
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

	} else {
		if err = row.Scan(&buildLog.ID,
			&buildLog.RepoSource,
			&buildLog.RepoOwner,
			&buildLog.RepoName,
			&buildLog.RepoBranch,
			&buildLog.RepoRevision,
			&rowBuildID,
			&buildLog.InsertedAt); err != nil {
			if err == sql.ErrNoRows {
				return nil, nil
			}

			return
		}
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

func (c *client) GetPipelineBuildLogsPerPage(ctx context.Context, repoSource, repoOwner, repoName string, pageNumber int, pageSize int) (buildLogs []*contracts.BuildLog, err error) {

	buildLogs = make([]*contracts.BuildLog, 0)

	// generate query
	query := c.selectBuildLogsQuery(true).
		Where(sq.Eq{"a.repo_source": repoSource}).
		Where(sq.Eq{"a.repo_owner": repoOwner}).
		Where(sq.Eq{"a.repo_name": repoName}).
		OrderBy("a.id").
		Limit(uint64(pageSize)).
		Offset(uint64((pageNumber - 1) * pageSize))

	sqlquery, _, _ := query.ToSql()
	log.Debug().Str("sql", sqlquery).Int("pageNumber", pageNumber).Int("pageSize", pageSize).Msgf("Query for GetPipelineBuildLogsPerPage")

	rows, err := query.RunWith(c.databaseConnection).Query()
	if err != nil {
		return buildLogs, err
	}

	defer rows.Close()
	for rows.Next() {

		buildLog := &contracts.BuildLog{}
		var stepsData []uint8
		var rowBuildID sql.NullInt64

		// execute query
		if err = rows.Scan(&buildLog.ID,
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

		buildLogs = append(buildLogs, buildLog)
	}

	return buildLogs, nil
}

func (c *client) GetPipelineBuildMaxResourceUtilization(ctx context.Context, repoSource, repoOwner, repoName string, lastNRecords int) (jobResources JobResources, recordCount int, err error) {

	// generate query
	psql := sq.StatementBuilder.PlaceholderFormat(sq.Dollar)

	innerquery := psql.
		Select("cpu_max_usage, memory_max_usage").
		From("builds").
		Where(sq.Eq{"repo_source": repoSource}).
		Where(sq.Eq{"repo_owner": repoOwner}).
		Where(sq.Eq{"repo_name": repoName}).
		Where(sq.NotEq{"cpu_max_usage": nil}).
		Where(sq.NotEq{"memory_max_usage": nil}).
		OrderBy("inserted_at DESC").
		Limit(uint64(lastNRecords))

	query := psql.Select("COALESCE(MAX(a.cpu_max_usage),0) AS max_cpu_max_usage, COALESCE(MAX(a.memory_max_usage),0) AS max_memory_max_usage, COUNT(a.*) AS nr_records").
		FromSelect(innerquery, "a")

	// execute query
	row := query.RunWith(c.databaseConnection).QueryRow()
	if err = row.Scan(&jobResources.CPUMaxUsage, &jobResources.MemoryMaxUsage, &recordCount); err != nil {

		return
	}

	return
}

func (c *client) GetPipelineReleases(ctx context.Context, repoSource, repoOwner, repoName string, pageNumber, pageSize int, filters map[string][]string) (releases []*contracts.Release, err error) {

	// generate query
	query := c.selectReleasesQuery().
		Where(sq.Eq{"a.repo_source": repoSource}).
		Where(sq.Eq{"a.repo_owner": repoOwner}).
		Where(sq.Eq{"a.repo_name": repoName}).
		OrderBy("a.inserted_at DESC").
		Limit(uint64(pageSize)).
		Offset(uint64((pageNumber - 1) * pageSize))

	// dynamically set where clauses for filtering
	query, err = whereClauseGeneratorForAllReleaseFilters(query, "a", "inserted_at", filters)
	if err != nil {

		return
	}

	// execute query
	rows, err := query.RunWith(c.databaseConnection).Query()
	if err != nil {

		return
	}

	// read rows
	if releases, err = c.scanReleases(rows); err != nil {

		return
	}

	return
}

func (c *client) GetPipelineReleasesCount(ctx context.Context, repoSource, repoOwner, repoName string, filters map[string][]string) (totalCount int, err error) {

	// generate query
	query :=
		sq.StatementBuilder.PlaceholderFormat(sq.Dollar).
			Select("COUNT(*)").
			From("releases a").
			Where(sq.Eq{"a.repo_source": repoSource}).
			Where(sq.Eq{"a.repo_owner": repoOwner}).
			Where(sq.Eq{"a.repo_name": repoName})

	// dynamically set where clauses for filtering
	query, err = whereClauseGeneratorForAllReleaseFilters(query, "a", "inserted_at", filters)
	if err != nil {

		return
	}

	// execute query
	row := query.RunWith(c.databaseConnection).QueryRow()
	if err = row.Scan(&totalCount); err != nil {

		return
	}

	return
}

func (c *client) GetPipelineRelease(ctx context.Context, repoSource, repoOwner, repoName string, id int) (release *contracts.Release, err error) {

	// generate query
	query := c.selectReleasesQuery().
		Where(sq.Eq{"a.id": id}).
		Where(sq.Eq{"a.repo_source": repoSource}).
		Where(sq.Eq{"a.repo_owner": repoOwner}).
		Where(sq.Eq{"a.repo_name": repoName}).
		OrderBy("a.inserted_at DESC").
		Limit(uint64(1))

	// execute query
	row := query.RunWith(c.databaseConnection).QueryRow()
	if release, err = c.scanRelease(row); err != nil {

		return
	}

	return
}

func (c *client) GetPipelineLastReleasesByName(ctx context.Context, repoSource, repoOwner, repoName, releaseName string, actions []string) (releases []contracts.Release, err error) {

	// generate query
	query := c.selectComputedReleasesQuery().
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
	rows, err := query.RunWith(c.databaseConnection).Query()
	if err != nil {

		return
	}

	// read rows
	releasesPointers := []*contracts.Release{}
	if releasesPointers, err = c.scanReleases(rows); err != nil {

		return releases, err
	}

	// copy pointer values
	releases = make([]contracts.Release, 0)
	for _, r := range releasesPointers {
		releases = append(releases, *r)
	}

	return
}

func (c *client) GetPipelineReleaseLogs(ctx context.Context, repoSource, repoOwner, repoName string, id int, readLogFromDatabase bool) (releaseLog *contracts.ReleaseLog, err error) {

	// generate query
	query := c.selectReleaseLogsQuery(readLogFromDatabase).
		Where(sq.Eq{"a.release_id": id}).
		Where(sq.Eq{"a.repo_source": repoSource}).
		Where(sq.Eq{"a.repo_owner": repoOwner}).
		Where(sq.Eq{"a.repo_name": repoName}).
		OrderBy("a.inserted_at DESC").
		Limit(uint64(1))

	releaseLog = &contracts.ReleaseLog{}

	var releaseID int

	// execute query
	row := query.RunWith(c.databaseConnection).QueryRow()
	if readLogFromDatabase {
		var stepsData []uint8
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

		if err = json.Unmarshal(stepsData, &releaseLog.Steps); err != nil {

			return
		}
	} else {
		if err = row.Scan(&releaseLog.ID,
			&releaseLog.RepoSource,
			&releaseLog.RepoOwner,
			&releaseLog.RepoName,
			&releaseID,
			&releaseLog.InsertedAt); err != nil {
			if err == sql.ErrNoRows {
				return nil, nil
			}

			return
		}
	}

	releaseLog.ReleaseID = strconv.Itoa(releaseID)

	return
}

func (c *client) GetPipelineReleaseLogsPerPage(ctx context.Context, repoSource, repoOwner, repoName string, pageNumber int, pageSize int) (releaseLogs []*contracts.ReleaseLog, err error) {

	releaseLogs = make([]*contracts.ReleaseLog, 0)

	// generate query
	query := c.selectReleaseLogsQuery(true).
		Where(sq.Eq{"a.repo_source": repoSource}).
		Where(sq.Eq{"a.repo_owner": repoOwner}).
		Where(sq.Eq{"a.repo_name": repoName}).
		OrderBy("a.id").
		Limit(uint64(pageSize)).
		Offset(uint64((pageNumber - 1) * pageSize))

	sqlquery, _, _ := query.ToSql()
	log.Debug().Str("sql", sqlquery).Int("pageNumber", pageNumber).Int("pageSize", pageSize).Msgf("Query for GetPipelineReleaseLogsPerPage")

	rows, err := query.RunWith(c.databaseConnection).Query()
	if err != nil {
		return releaseLogs, err
	}

	defer rows.Close()
	for rows.Next() {

		releaseLog := &contracts.ReleaseLog{}
		var stepsData []uint8
		var releaseID int

		if err = rows.Scan(&releaseLog.ID,
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

		releaseLogs = append(releaseLogs, releaseLog)
	}

	return releaseLogs, nil
}

func (c *client) GetPipelineReleaseMaxResourceUtilization(ctx context.Context, repoSource, repoOwner, repoName, targetName string, lastNRecords int) (jobResources JobResources, recordCount int, err error) {

	// generate query
	psql := sq.StatementBuilder.PlaceholderFormat(sq.Dollar)

	innerquery := psql.
		Select("cpu_max_usage, memory_max_usage").
		From("releases").
		Where(sq.Eq{"repo_source": repoSource}).
		Where(sq.Eq{"repo_owner": repoOwner}).
		Where(sq.Eq{"repo_name": repoName}).
		Where(sq.Eq{"release": targetName}).
		Where(sq.NotEq{"cpu_max_usage": nil}).
		Where(sq.NotEq{"memory_max_usage": nil}).
		OrderBy("inserted_at DESC").
		Limit(uint64(lastNRecords))

	query := psql.Select("COALESCE(MAX(a.cpu_max_usage),0) AS max_cpu_max_usage, COALESCE(MAX(a.memory_max_usage),0) AS max_memory_max_usage, COUNT(a.*) AS nr_records").
		FromSelect(innerquery, "a")

	// execute query
	row := query.RunWith(c.databaseConnection).QueryRow()
	if err = row.Scan(&jobResources.CPUMaxUsage, &jobResources.MemoryMaxUsage, &recordCount); err != nil {

		return
	}

	return
}

func (c *client) GetBuildsCount(ctx context.Context, filters map[string][]string) (totalCount int, err error) {

	// generate query
	query :=
		sq.StatementBuilder.PlaceholderFormat(sq.Dollar).
			Select("COUNT(*)").
			From("builds a")

	// dynamically set where clauses for filtering
	query, err = whereClauseGeneratorForAllFilters(query, "a", "inserted_at", filters)
	if err != nil {

		return
	}

	// execute query
	row := query.RunWith(c.databaseConnection).QueryRow()
	if err = row.Scan(&totalCount); err != nil {

		return
	}

	return
}

func (c *client) GetReleasesCount(ctx context.Context, filters map[string][]string) (totalCount int, err error) {

	// generate query
	query :=
		sq.StatementBuilder.PlaceholderFormat(sq.Dollar).
			Select("COUNT(*)").
			From("releases a")

	// dynamically set where clauses for filtering
	query, err = whereClauseGeneratorForAllReleaseFilters(query, "a", "inserted_at", filters)
	if err != nil {

		return
	}

	// execute query
	row := query.RunWith(c.databaseConnection).QueryRow()
	if err = row.Scan(&totalCount); err != nil {

		return
	}

	return
}

func (c *client) GetBuildsDuration(ctx context.Context, filters map[string][]string) (totalDuration time.Duration, err error) {

	// generate query
	query :=
		sq.StatementBuilder.PlaceholderFormat(sq.Dollar).
			Select("SUM(AGE(updated_at,inserted_at))::string").
			From("builds a")

	// dynamically set where clauses for filtering
	query, err = whereClauseGeneratorForAllFilters(query, "a", "inserted_at", filters)
	if err != nil {

		return
	}

	// execute query
	row := query.RunWith(c.databaseConnection).QueryRow()

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

func (c *client) GetFirstBuildTimes(ctx context.Context) (buildTimes []time.Time, err error) {

	// generate query
	query :=
		sq.StatementBuilder.PlaceholderFormat(sq.Dollar).
			Select("a.first_inserted_at").
			From("computed_pipelines a").
			OrderBy("a.first_inserted_at")

	buildTimes = make([]time.Time, 0)

	// execute query
	rows, err := query.RunWith(c.databaseConnection).Query()

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

func (c *client) GetFirstReleaseTimes(ctx context.Context) (releaseTimes []time.Time, err error) {

	// generate query
	query :=
		sq.StatementBuilder.PlaceholderFormat(sq.Dollar).
			Select("MIN(a.first_inserted_at)").
			From("computed_releases a").
			GroupBy("a.repo_source,a.repo_owner,a.repo_name").
			OrderBy("MIN(a.first_inserted_at)")

	releaseTimes = make([]time.Time, 0)

	rows, err := query.RunWith(c.databaseConnection).Query()

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

func (c *client) GetPipelineBuildsDurations(ctx context.Context, repoSource, repoOwner, repoName string, filters map[string][]string) (durations []map[string]interface{}, err error) {

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

	rows, err := query.RunWith(c.databaseConnection).Query()

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

func (c *client) GetPipelineReleasesDurations(ctx context.Context, repoSource, repoOwner, repoName string, filters map[string][]string) (durations []map[string]interface{}, err error) {

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

	rows, err := query.RunWith(c.databaseConnection).Query()

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

func (c *client) GetPipelineBuildsCPUUsageMeasurements(ctx context.Context, repoSource, repoOwner, repoName string, filters map[string][]string) (measurements []map[string]interface{}, err error) {

	// generate query
	innerquery :=
		sq.StatementBuilder.PlaceholderFormat(sq.Dollar).
			Select("a.inserted_at, a.cpu_max_usage").
			From("builds a").
			Where(sq.Eq{"a.repo_source": repoSource}).
			Where(sq.Eq{"a.repo_owner": repoOwner}).
			Where(sq.Eq{"a.repo_name": repoName}).
			Where(sq.NotEq{"a.cpu_max_usage": nil}).
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
			OrderBy("a.inserted_at")

	measurements = make([]map[string]interface{}, 0)

	rows, err := query.RunWith(c.databaseConnection).Query()

	if err != nil {

		return
	}

	defer rows.Close()
	for rows.Next() {

		var insertedAt time.Time
		var cpuMaxUsage float64

		if err = rows.Scan(
			&insertedAt, &cpuMaxUsage); err != nil {

			return
		}

		measurements = append(measurements, map[string]interface{}{
			"insertedAt":  insertedAt,
			"cpuMaxUsage": cpuMaxUsage,
		})
	}

	return
}

func (c *client) GetPipelineReleasesCPUUsageMeasurements(ctx context.Context, repoSource, repoOwner, repoName string, filters map[string][]string) (measurements []map[string]interface{}, err error) {

	// generate query
	innerquery :=
		sq.StatementBuilder.PlaceholderFormat(sq.Dollar).
			Select("a.inserted_at, a.release, a.release_action, a.cpu_max_usage").
			From("releases a").
			Where(sq.Eq{"a.repo_source": repoSource}).
			Where(sq.Eq{"a.repo_owner": repoOwner}).
			Where(sq.Eq{"a.repo_name": repoName}).
			Where(sq.NotEq{"a.cpu_max_usage": nil}).
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
			OrderBy("a.inserted_at")

	measurements = make([]map[string]interface{}, 0)

	rows, err := query.RunWith(c.databaseConnection).Query()

	if err != nil {

		return
	}

	defer rows.Close()
	for rows.Next() {

		var insertedAt time.Time
		var releaseName, releaseAction string
		var cpuMaxUsage float64

		if err = rows.Scan(
			&insertedAt, &releaseName, &releaseAction, &cpuMaxUsage); err != nil {

			return
		}

		measurements = append(measurements, map[string]interface{}{
			"insertedAt":  insertedAt,
			"name":        releaseName,
			"action":      releaseAction,
			"cpuMaxUsage": cpuMaxUsage,
		})
	}

	return
}

func (c *client) GetPipelineBuildsMemoryUsageMeasurements(ctx context.Context, repoSource, repoOwner, repoName string, filters map[string][]string) (measurements []map[string]interface{}, err error) {

	// generate query
	innerquery :=
		sq.StatementBuilder.PlaceholderFormat(sq.Dollar).
			Select("a.inserted_at, a.memory_max_usage").
			From("builds a").
			Where(sq.Eq{"a.repo_source": repoSource}).
			Where(sq.Eq{"a.repo_owner": repoOwner}).
			Where(sq.Eq{"a.repo_name": repoName}).
			Where(sq.NotEq{"a.memory_max_usage": nil}).
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
			OrderBy("a.inserted_at")

	measurements = make([]map[string]interface{}, 0)

	rows, err := query.RunWith(c.databaseConnection).Query()

	if err != nil {

		return
	}

	defer rows.Close()
	for rows.Next() {

		var insertedAt time.Time
		var memoryMaxUsage float64

		if err = rows.Scan(
			&insertedAt, &memoryMaxUsage); err != nil {

			return
		}

		measurements = append(measurements, map[string]interface{}{
			"insertedAt":     insertedAt,
			"memoryMaxUsage": memoryMaxUsage,
		})
	}

	return
}

func (c *client) GetPipelineReleasesMemoryUsageMeasurements(ctx context.Context, repoSource, repoOwner, repoName string, filters map[string][]string) (measurements []map[string]interface{}, err error) {

	// generate query
	innerquery :=
		sq.StatementBuilder.PlaceholderFormat(sq.Dollar).
			Select("a.inserted_at, a.release, a.release_action, a.memory_max_usage").
			From("releases a").
			Where(sq.Eq{"a.repo_source": repoSource}).
			Where(sq.Eq{"a.repo_owner": repoOwner}).
			Where(sq.Eq{"a.repo_name": repoName}).
			Where(sq.NotEq{"a.memory_max_usage": nil}).
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
			OrderBy("a.inserted_at")

	measurements = make([]map[string]interface{}, 0)

	rows, err := query.RunWith(c.databaseConnection).Query()

	if err != nil {

		return
	}

	defer rows.Close()
	for rows.Next() {

		var insertedAt time.Time
		var releaseName, releaseAction string
		var memoryMaxUsage float64

		if err = rows.Scan(
			&insertedAt, &releaseName, &releaseAction, &memoryMaxUsage); err != nil {

			return
		}

		measurements = append(measurements, map[string]interface{}{
			"insertedAt":     insertedAt,
			"name":           releaseName,
			"action":         releaseAction,
			"memoryMaxUsage": memoryMaxUsage,
		})
	}

	return
}

func (c *client) GetLabelValues(ctx context.Context, labelKey string) (labels []map[string]interface{}, err error) {

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
	//                WHERE
	//                  l->>'key' = 'type'
	// 						)
	// 				GROUP BY
	// 						key, value
	// 		)
	// ORDER BY
	// 		value

	arrayElementsQuery :=
		sq.StatementBuilder.
			Select("a.id, jsonb_array_elements(a.labels) AS l").
			From("computed_pipelines a").
			Where("jsonb_typeof(labels) = 'array'")

	selectCountQuery :=
		sq.StatementBuilder.PlaceholderFormat(sq.Dollar).
			Select("l->>'key' AS key, l->>'value' AS value, id").
			FromSelect(arrayElementsQuery, "b").
			Where(sq.Eq{"l->>'key'": labelKey})

	groupByQuery :=
		sq.StatementBuilder.
			Select("key, value, count(DISTINCT id) AS pipelinesCount").
			FromSelect(selectCountQuery, "c").
			GroupBy("key, value")

	query :=
		sq.StatementBuilder.PlaceholderFormat(sq.Dollar).
			Select("key, value, pipelinesCount").
			FromSelect(groupByQuery, "d").
			OrderBy("value")

	rows, err := query.RunWith(c.databaseConnection).Query()

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

func (c *client) GetFrequentLabels(ctx context.Context, pageNumber, pageSize int, filters map[string][]string) (labels []map[string]interface{}, err error) {

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

	arrayElementsQuery, err = whereClauseGeneratorForSinceFilter(arrayElementsQuery, "a", "last_updated_at", filters)
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

	rows, err := query.RunWith(c.databaseConnection).Query()

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

func (c *client) GetFrequentLabelsCount(ctx context.Context, filters map[string][]string) (totalCount int, err error) {

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

	arrayElementsQuery, err = whereClauseGeneratorForSinceFilter(arrayElementsQuery, "a", "last_updated_at", filters)
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
	row := query.RunWith(c.databaseConnection).QueryRow()
	if err = row.Scan(&totalCount); err != nil {

		return
	}

	return
}

func (c *client) GetPipelinesWithMostBuilds(ctx context.Context, pageNumber, pageSize int, filters map[string][]string) (pipelines []map[string]interface{}, err error) {

	// generate query
	query :=
		sq.StatementBuilder.PlaceholderFormat(sq.Dollar).
			Select("a.repo_source, a.repo_owner, a.repo_name, count(a.id) as nr_records").
			From("builds a").
			GroupBy("a.repo_source, a.repo_owner, a.repo_name").
			OrderBy("nr_records DESC, a.repo_source, a.repo_owner, a.repo_name").
			Limit(uint64(pageSize)).
			Offset(uint64((pageNumber - 1) * pageSize))

	query, err = whereClauseGeneratorForSinceFilter(query, "a", "inserted_at", filters)
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

	rows, err := query.RunWith(c.databaseConnection).Query()

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

func (c *client) GetPipelinesWithMostBuildsCount(ctx context.Context, filters map[string][]string) (totalCount int, err error) {
	return c.GetPipelinesCount(ctx, filters)
}

func (c *client) GetPipelinesWithMostReleases(ctx context.Context, pageNumber, pageSize int, filters map[string][]string) (pipelines []map[string]interface{}, err error) {

	// generate query
	query :=
		sq.StatementBuilder.PlaceholderFormat(sq.Dollar).
			Select("a.repo_source, a.repo_owner, a.repo_name, count(a.id) as nr_records").
			From("releases a").
			GroupBy("a.repo_source, a.repo_owner, a.repo_name").
			OrderBy("nr_records DESC, a.repo_source, a.repo_owner, a.repo_name").
			Limit(uint64(pageSize)).
			Offset(uint64((pageNumber - 1) * pageSize))

	query, err = whereClauseGeneratorForSinceFilter(query, "a", "inserted_at", filters)
	if err != nil {

		return
	}

	query, err = whereClauseGeneratorForReleaseStatusFilter(query, "a", filters)
	if err != nil {

		return
	}

	pipelines = make([]map[string]interface{}, 0)

	rows, err := query.RunWith(c.databaseConnection).Query()

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

func (c *client) GetPipelinesWithMostReleasesCount(ctx context.Context, filters map[string][]string) (totalCount int, err error) {

	// generate query
	innerquery :=
		sq.StatementBuilder.PlaceholderFormat(sq.Dollar).
			Select("a.repo_source, a.repo_owner, a.repo_name").
			From("releases a").
			GroupBy("a.repo_source, a.repo_owner, a.repo_name")

	innerquery, err = whereClauseGeneratorForSinceFilter(innerquery, "a", "inserted_at", filters)
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
	row := query.RunWith(c.databaseConnection).QueryRow()
	if err = row.Scan(&totalCount); err != nil {

		return
	}

	return
}

func whereClauseGeneratorForAllFilters(query sq.SelectBuilder, alias, sinceColumn string, filters map[string][]string) (sq.SelectBuilder, error) {

	query, err := whereClauseGeneratorForSinceFilter(query, alias, sinceColumn, filters)
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

func whereClauseGeneratorForAllReleaseFilters(query sq.SelectBuilder, alias, sinceColumn string, filters map[string][]string) (sq.SelectBuilder, error) {

	query, err := whereClauseGeneratorForReleaseStatusFilter(query, alias, filters)
	if err != nil {
		return query, err
	}
	query, err = whereClauseGeneratorForSinceFilter(query, alias, sinceColumn, filters)
	if err != nil {
		return query, err
	}

	return query, nil
}

func whereClauseGeneratorForSinceFilter(query sq.SelectBuilder, alias, sinceColumn string, filters map[string][]string) (sq.SelectBuilder, error) {

	if since, ok := filters["since"]; ok && len(since) > 0 && since[0] != "eternity" {
		sinceValue := since[0]
		switch sinceValue {
		case "1h":
			query = query.Where(sq.GtOrEq{fmt.Sprintf("%v.%v", alias, sinceColumn): time.Now().Add(time.Duration(-1) * time.Hour)})
		case "1d":
			query = query.Where(sq.GtOrEq{fmt.Sprintf("%v.%v", alias, sinceColumn): time.Now().AddDate(0, 0, -1)})
		case "1w":
			query = query.Where(sq.GtOrEq{fmt.Sprintf("%v.%v", alias, sinceColumn): time.Now().AddDate(0, 0, -7)})
		case "1m":
			query = query.Where(sq.GtOrEq{fmt.Sprintf("%v.%v", alias, sinceColumn): time.Now().AddDate(0, -1, 0)})
		case "1y":
			query = query.Where(sq.GtOrEq{fmt.Sprintf("%v.%v", alias, sinceColumn): time.Now().AddDate(-1, 0, 0)})
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

func (c *client) scanBuild(ctx context.Context, row sq.RowScanner, optimized, enriched bool) (build *contracts.Build, err error) {

	build = &contracts.Build{}
	var labelsData, releaseTargetsData, commitsData, triggersData, triggeredByEventsData []uint8
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
		&seconds,
		&triggeredByEventsData); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return
	}

	build.Duration = time.Duration(seconds) * time.Second

	c.setBuildPropertiesFromJSONB(build, labelsData, releaseTargetsData, commitsData, triggersData, triggeredByEventsData, optimized)

	if enriched {
		c.enrichBuild(ctx, build)
	}

	if optimized {
		// clear some properties for reduced size and improved performance over the network
		build.Manifest = ""
		build.ManifestWithDefaults = ""
	}

	return
}

func (c *client) scanBuilds(rows *sql.Rows, optimized bool) (builds []*contracts.Build, err error) {

	builds = make([]*contracts.Build, 0)

	defer rows.Close()
	for rows.Next() {

		build := contracts.Build{}
		var labelsData, releaseTargetsData, commitsData, triggersData, triggeredByEventsData []uint8
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
			&seconds,
			&triggeredByEventsData); err != nil {
			return
		}

		build.Duration = time.Duration(seconds) * time.Second

		c.setBuildPropertiesFromJSONB(&build, labelsData, releaseTargetsData, commitsData, triggersData, triggeredByEventsData, optimized)

		if optimized {
			// clear some properties for reduced size and improved performance over the network
			build.Manifest = ""
			build.ManifestWithDefaults = ""
		}

		builds = append(builds, &build)
	}

	return
}

func (c *client) scanPipeline(row sq.RowScanner, optimized bool) (pipeline *contracts.Pipeline, err error) {

	pipeline = &contracts.Pipeline{}
	var labelsData, releaseTargetsData, commitsData, triggersData, triggeredByEventsData []uint8
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
		&pipeline.Archived,
		&pipeline.InsertedAt,
		&pipeline.UpdatedAt,
		&seconds,
		&pipeline.LastUpdatedAt,
		&triggeredByEventsData); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return
	}

	pipeline.Duration = time.Duration(seconds) * time.Second

	c.setPipelinePropertiesFromJSONB(pipeline, labelsData, releaseTargetsData, commitsData, triggersData, triggeredByEventsData, optimized)

	if optimized {
		// clear some properties for reduced size and improved performance over the network
		pipeline.Manifest = ""
		pipeline.ManifestWithDefaults = ""
	}

	return
}

func (c *client) scanPipelines(rows *sql.Rows, optimized bool) (pipelines []*contracts.Pipeline, err error) {

	pipelines = make([]*contracts.Pipeline, 0)

	defer rows.Close()
	for rows.Next() {

		pipeline := contracts.Pipeline{}
		var labelsData, releaseTargetsData, commitsData, triggersData, triggeredByEventsData []uint8
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
			&pipeline.Archived,
			&pipeline.InsertedAt,
			&pipeline.UpdatedAt,
			&seconds,
			&pipeline.LastUpdatedAt,
			&triggeredByEventsData); err != nil {
			return
		}

		pipeline.Duration = time.Duration(seconds) * time.Second

		c.setPipelinePropertiesFromJSONB(&pipeline, labelsData, releaseTargetsData, commitsData, triggersData, triggeredByEventsData, optimized)

		if optimized {
			// clear some properties for reduced size and improved performance over the network
			pipeline.Manifest = ""
			pipeline.ManifestWithDefaults = ""
		}

		pipelines = append(pipelines, &pipeline)
	}

	return
}

func (c *client) scanRelease(row sq.RowScanner) (release *contracts.Release, err error) {

	release = &contracts.Release{}
	var seconds int
	var id int
	var triggeredByEventsData []uint8

	if err = row.Scan(
		&id,
		&release.RepoSource,
		&release.RepoOwner,
		&release.RepoName,
		&release.Name,
		&release.Action,
		&release.ReleaseVersion,
		&release.ReleaseStatus,
		&release.InsertedAt,
		&release.UpdatedAt,
		&seconds,
		&triggeredByEventsData); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return
	}

	duration := time.Duration(seconds) * time.Second
	release.Duration = &duration
	release.ID = strconv.Itoa(id)

	if len(triggeredByEventsData) > 0 {
		if err = json.Unmarshal(triggeredByEventsData, &release.Events); err != nil {
			return
		}
	}

	return
}

func (c *client) scanReleases(rows *sql.Rows) (releases []*contracts.Release, err error) {

	releases = make([]*contracts.Release, 0)

	defer rows.Close()
	for rows.Next() {

		release := contracts.Release{}
		var seconds int
		var id int
		var triggeredByEventsData []uint8

		if err = rows.Scan(
			&id,
			&release.RepoSource,
			&release.RepoOwner,
			&release.RepoName,
			&release.Name,
			&release.Action,
			&release.ReleaseVersion,
			&release.ReleaseStatus,
			&release.InsertedAt,
			&release.UpdatedAt,
			&seconds,
			&triggeredByEventsData); err != nil {
			return
		}

		duration := time.Duration(seconds) * time.Second
		release.Duration = &duration
		release.ID = strconv.Itoa(id)

		if len(triggeredByEventsData) > 0 {
			if err = json.Unmarshal(triggeredByEventsData, &release.Events); err != nil {
				return
			}
		}

		releases = append(releases, &release)
	}

	return
}

func (c *client) GetTriggers(ctx context.Context, triggerType, identifier, event string) (pipelines []*contracts.Pipeline, err error) {

	// generate query
	query := c.selectPipelinesQuery().
		Where(sq.Eq{"a.archived": false})

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

	case "git":

		trigger.Git = &manifest.EstafetteGitTrigger{
			Event:      event,
			Repository: identifier,
		}

		break

	case "pubsub":

		trigger.PubSub = &manifest.EstafettePubSubTrigger{}

		break

	default:

		return pipelines, fmt.Errorf("Trigger type %v is not supported", triggerType)
	}

	bytes, err := json.Marshal([]manifest.EstafetteTrigger{trigger})
	if err != nil {

		return pipelines, err
	}

	query = query.Where("a.triggers @> ?", string(bytes))

	// execute query
	rows, err := query.RunWith(c.databaseConnection).Query()
	if err != nil {

		return
	}

	// read rows
	if pipelines, err = c.scanPipelines(rows, false); err != nil {

		return
	}

	return
}

func (c *client) GetGitTriggers(ctx context.Context, gitEvent manifest.EstafetteGitEvent) ([]*contracts.Pipeline, error) {

	triggerType := "git"
	name := gitEvent.Repository
	event := gitEvent.Event

	return c.GetTriggers(ctx, triggerType, name, event)
}

func (c *client) GetPipelineTriggers(ctx context.Context, build contracts.Build, event string) ([]*contracts.Pipeline, error) {

	triggerType := "pipeline"
	name := fmt.Sprintf("%v/%v/%v", build.RepoSource, build.RepoOwner, build.RepoName)

	return c.GetTriggers(ctx, triggerType, name, event)
}

func (c *client) GetReleaseTriggers(ctx context.Context, release contracts.Release, event string) ([]*contracts.Pipeline, error) {

	triggerType := "release"
	name := fmt.Sprintf("%v/%v/%v", release.RepoSource, release.RepoOwner, release.RepoName)

	return c.GetTriggers(ctx, triggerType, name, event)
}

func (c *client) GetPubSubTriggers(ctx context.Context, pubsubEvent manifest.EstafettePubSubEvent) ([]*contracts.Pipeline, error) {

	triggerType := "pubsub"

	return c.GetTriggers(ctx, triggerType, "", "")
}

func (c *client) GetCronTriggers(ctx context.Context) ([]*contracts.Pipeline, error) {

	triggerType := "cron"

	return c.GetTriggers(ctx, triggerType, "", "")
}

func (c *client) Rename(ctx context.Context, shortFromRepoSource, fromRepoSource, fromRepoOwner, fromRepoName, shortToRepoSource, toRepoSource, toRepoOwner, toRepoName string) error {

	nrOfQueries := 7
	var wg sync.WaitGroup
	wg.Add(nrOfQueries)

	errors := make(chan error, nrOfQueries)

	go func(wg *sync.WaitGroup, ctx context.Context, shortFromRepoSource, fromRepoOwner, fromRepoName, shortToRepoSource, toRepoOwner, toRepoName string) {
		defer wg.Done()
		err := c.RenameBuildVersion(ctx, shortFromRepoSource, fromRepoOwner, fromRepoName, shortToRepoSource, toRepoOwner, toRepoName)
		if err != nil {
			errors <- err
		}
	}(&wg, ctx, shortFromRepoSource, fromRepoOwner, fromRepoName, shortToRepoSource, toRepoOwner, toRepoName)

	go func(wg *sync.WaitGroup, ctx context.Context, fromRepoSource, fromRepoOwner, fromRepoName, toRepoSource, toRepoOwner, toRepoName string) {
		defer wg.Done()
		err := c.RenameBuilds(ctx, fromRepoSource, fromRepoOwner, fromRepoName, toRepoSource, toRepoOwner, toRepoName)
		if err != nil {
			errors <- err
		}
	}(&wg, ctx, fromRepoSource, fromRepoOwner, fromRepoName, toRepoSource, toRepoOwner, toRepoName)

	go func(wg *sync.WaitGroup, ctx context.Context, fromRepoSource, fromRepoOwner, fromRepoName, toRepoSource, toRepoOwner, toRepoName string) {
		defer wg.Done()
		err := c.RenameBuildLogs(ctx, fromRepoSource, fromRepoOwner, fromRepoName, toRepoSource, toRepoOwner, toRepoName)
		if err != nil {
			errors <- err
		}
	}(&wg, ctx, fromRepoSource, fromRepoOwner, fromRepoName, toRepoSource, toRepoOwner, toRepoName)

	go func(wg *sync.WaitGroup, ctx context.Context, fromRepoSource, fromRepoOwner, fromRepoName, toRepoSource, toRepoOwner, toRepoName string) {
		defer wg.Done()
		err := c.RenameReleases(ctx, fromRepoSource, fromRepoOwner, fromRepoName, toRepoSource, toRepoOwner, toRepoName)
		if err != nil {
			errors <- err
		}
	}(&wg, ctx, fromRepoSource, fromRepoOwner, fromRepoName, toRepoSource, toRepoOwner, toRepoName)

	go func(wg *sync.WaitGroup, ctx context.Context, fromRepoSource, fromRepoOwner, fromRepoName, toRepoSource, toRepoOwner, toRepoName string) {
		defer wg.Done()
		err := c.RenameReleaseLogs(ctx, fromRepoSource, fromRepoOwner, fromRepoName, toRepoSource, toRepoOwner, toRepoName)
		if err != nil {
			errors <- err
		}
	}(&wg, ctx, fromRepoSource, fromRepoOwner, fromRepoName, toRepoSource, toRepoOwner, toRepoName)

	go func(wg *sync.WaitGroup, ctx context.Context, fromRepoSource, fromRepoOwner, fromRepoName, toRepoSource, toRepoOwner, toRepoName string) {
		defer wg.Done()
		err := c.RenameComputedPipelines(ctx, fromRepoSource, fromRepoOwner, fromRepoName, toRepoSource, toRepoOwner, toRepoName)
		if err != nil {
			errors <- err
		}
	}(&wg, ctx, fromRepoSource, fromRepoOwner, fromRepoName, toRepoSource, toRepoOwner, toRepoName)

	go func(wg *sync.WaitGroup, ctx context.Context, fromRepoSource, fromRepoOwner, fromRepoName, toRepoSource, toRepoOwner, toRepoName string) {
		defer wg.Done()
		err := c.RenameComputedReleases(ctx, fromRepoSource, fromRepoOwner, fromRepoName, toRepoSource, toRepoOwner, toRepoName)
		if err != nil {
			errors <- err
		}
	}(&wg, ctx, fromRepoSource, fromRepoOwner, fromRepoName, toRepoSource, toRepoOwner, toRepoName)

	wg.Wait()

	close(errors)
	for e := range errors {
		return e
	}

	// update computed tables
	err := c.UpsertComputedPipeline(ctx, toRepoSource, toRepoOwner, toRepoName)
	if err != nil {

		return err
	}

	return nil
}

func (c *client) RenameBuildVersion(ctx context.Context, shortFromRepoSource, fromRepoOwner, fromRepoName, shortToRepoSource, toRepoOwner, toRepoName string) error {

	psql := sq.StatementBuilder.PlaceholderFormat(sq.Dollar)

	query := psql.
		Update("build_versions").
		Set("repo_source", shortToRepoSource).
		Set("repo_full_name", fmt.Sprintf("%v/%v", toRepoOwner, toRepoName)).
		Where(sq.Eq{"repo_source": shortFromRepoSource}).
		Where(sq.Eq{"repo_full_name": fmt.Sprintf("%v/%v", fromRepoOwner, fromRepoName)}).
		Limit(uint64(1))

	_, err := query.RunWith(c.databaseConnection).Exec()
	if err != nil {

		return err
	}

	return nil
}

func (c *client) RenameBuilds(ctx context.Context, fromRepoSource, fromRepoOwner, fromRepoName, toRepoSource, toRepoOwner, toRepoName string) error {

	psql := sq.StatementBuilder.PlaceholderFormat(sq.Dollar)

	query := psql.
		Update("builds").
		Set("repo_source", toRepoSource).
		Set("repo_owner", toRepoOwner).
		Set("repo_name", toRepoName).
		Where(sq.Eq{"repo_source": fromRepoSource}).
		Where(sq.Eq{"repo_owner": fromRepoOwner}).
		Where(sq.Eq{"repo_name": fromRepoName})

	_, err := query.RunWith(c.databaseConnection).Exec()
	if err != nil {

		return err
	}

	return nil
}

func (c *client) RenameBuildLogs(ctx context.Context, fromRepoSource, fromRepoOwner, fromRepoName, toRepoSource, toRepoOwner, toRepoName string) error {

	psql := sq.StatementBuilder.PlaceholderFormat(sq.Dollar)

	query := psql.
		Update("build_logs").
		Set("repo_source", toRepoSource).
		Set("repo_owner", toRepoOwner).
		Set("repo_name", toRepoName).
		Where(sq.Eq{"repo_source": fromRepoSource}).
		Where(sq.Eq{"repo_owner": fromRepoOwner}).
		Where(sq.Eq{"repo_name": fromRepoName})

	_, err := query.RunWith(c.databaseConnection).Exec()
	if err != nil {

		return err
	}

	return nil
}

func (c *client) RenameReleases(ctx context.Context, fromRepoSource, fromRepoOwner, fromRepoName, toRepoSource, toRepoOwner, toRepoName string) error {

	psql := sq.StatementBuilder.PlaceholderFormat(sq.Dollar)

	query := psql.
		Update("releases").
		Set("repo_source", toRepoSource).
		Set("repo_owner", toRepoOwner).
		Set("repo_name", toRepoName).
		Where(sq.Eq{"repo_source": fromRepoSource}).
		Where(sq.Eq{"repo_owner": fromRepoOwner}).
		Where(sq.Eq{"repo_name": fromRepoName})

	_, err := query.RunWith(c.databaseConnection).Exec()
	if err != nil {

		return err
	}

	return nil
}

func (c *client) RenameReleaseLogs(ctx context.Context, fromRepoSource, fromRepoOwner, fromRepoName, toRepoSource, toRepoOwner, toRepoName string) error {

	psql := sq.StatementBuilder.PlaceholderFormat(sq.Dollar)

	query := psql.
		Update("release_logs").
		Set("repo_source", toRepoSource).
		Set("repo_owner", toRepoOwner).
		Set("repo_name", toRepoName).
		Where(sq.Eq{"repo_source": fromRepoSource}).
		Where(sq.Eq{"repo_owner": fromRepoOwner}).
		Where(sq.Eq{"repo_name": fromRepoName})

	_, err := query.RunWith(c.databaseConnection).Exec()
	if err != nil {

		return err
	}

	return nil
}

func (c *client) RenameComputedPipelines(ctx context.Context, fromRepoSource, fromRepoOwner, fromRepoName, toRepoSource, toRepoOwner, toRepoName string) error {

	psql := sq.StatementBuilder.PlaceholderFormat(sq.Dollar)

	query := psql.
		Update("computed_pipelines").
		Set("repo_source", toRepoSource).
		Set("repo_owner", toRepoOwner).
		Set("repo_name", toRepoName).
		Where(sq.Eq{"repo_source": fromRepoSource}).
		Where(sq.Eq{"repo_owner": fromRepoOwner}).
		Where(sq.Eq{"repo_name": fromRepoName})

	_, err := query.RunWith(c.databaseConnection).Exec()
	if err != nil {

		return err
	}

	return nil
}

func (c *client) RenameComputedReleases(ctx context.Context, fromRepoSource, fromRepoOwner, fromRepoName, toRepoSource, toRepoOwner, toRepoName string) error {

	psql := sq.StatementBuilder.PlaceholderFormat(sq.Dollar)

	query := psql.
		Update("computed_releases").
		Set("repo_source", toRepoSource).
		Set("repo_owner", toRepoOwner).
		Set("repo_name", toRepoName).
		Where(sq.Eq{"repo_source": fromRepoSource}).
		Where(sq.Eq{"repo_owner": fromRepoOwner}).
		Where(sq.Eq{"repo_name": fromRepoName})

	_, err := query.RunWith(c.databaseConnection).Exec()
	if err != nil {

		return err
	}

	return nil
}

func (c *client) selectBuildsQuery() sq.SelectBuilder {
	psql := sq.StatementBuilder.PlaceholderFormat(sq.Dollar)

	return psql.
		Select("a.id, a.repo_source, a.repo_owner, a.repo_name, a.repo_branch, a.repo_revision, a.build_version, a.build_status, a.labels, a.release_targets, a.manifest, a.commits, a.triggers, a.inserted_at, a.updated_at, a.duration::INT, a.triggered_by_event").
		From("builds a")
}

func (c *client) selectPipelinesQuery() sq.SelectBuilder {
	psql := sq.StatementBuilder.PlaceholderFormat(sq.Dollar)

	return psql.
		Select("a.pipeline_id, a.repo_source, a.repo_owner, a.repo_name, a.repo_branch, a.repo_revision, a.build_version, a.build_status, a.labels, a.release_targets, a.manifest, a.commits, a.triggers, a.archived, a.inserted_at, a.updated_at, a.duration::INT, a.last_updated_at, a.triggered_by_event").
		From("computed_pipelines a")
}

func (c *client) selectReleasesQuery() sq.SelectBuilder {
	psql := sq.StatementBuilder.PlaceholderFormat(sq.Dollar)

	return psql.
		Select("a.id, a.repo_source, a.repo_owner, a.repo_name, a.release, a.release_action, a.release_version, a.release_status, a.inserted_at, a.updated_at, a.duration::INT, a.triggered_by_event").
		From("releases a")
}

func (c *client) selectComputedReleasesQuery() sq.SelectBuilder {
	psql := sq.StatementBuilder.PlaceholderFormat(sq.Dollar)

	return psql.
		Select("a.release_id, a.repo_source, a.repo_owner, a.repo_name, a.release, a.release_action, a.release_version, a.release_status, a.inserted_at, a.updated_at, a.duration::INT, a.triggered_by_event").
		From("computed_releases a")
}

func (c *client) selectBuildLogsQuery(readLogFromDatabase bool) sq.SelectBuilder {
	psql := sq.StatementBuilder.PlaceholderFormat(sq.Dollar)

	if readLogFromDatabase {
		return psql.
			Select("a.id, a.repo_source, a.repo_owner, a.repo_name, a.repo_branch, a.repo_revision, a.build_id, a.steps, a.inserted_at").
			From("build_logs a")
	}

	return psql.
		Select("a.id, a.repo_source, a.repo_owner, a.repo_name, a.repo_branch, a.repo_revision, a.build_id, a.inserted_at").
		From("build_logs a")
}

func (c *client) selectReleaseLogsQuery(readLogFromDatabase bool) sq.SelectBuilder {
	psql := sq.StatementBuilder.PlaceholderFormat(sq.Dollar)

	if readLogFromDatabase {
		return psql.
			Select("a.id, a.repo_source, a.repo_owner, a.repo_name, a.release_id, a.steps, a.inserted_at").
			From("release_logs a")
	}

	return psql.
		Select("a.id, a.repo_source, a.repo_owner, a.repo_name, a.release_id, a.inserted_at").
		From("release_logs a")
}

func (c *client) selectPipelineTriggersQuery() sq.SelectBuilder {
	psql := sq.StatementBuilder.PlaceholderFormat(sq.Dollar)

	return psql.
		Select("a.id, a.repo_source, a.repo_owner, a.repo_name, a.trigger_event, a.trigger_filter, a.trigger_run, a.inserted_at, a.updated_at").
		From("pipeline_triggers a")
}

func (c *client) enrichPipeline(ctx context.Context, pipeline *contracts.Pipeline) {
	c.getLatestReleasesForPipeline(ctx, pipeline)
}

func (c *client) enrichBuild(ctx context.Context, build *contracts.Build) {
	c.getLatestReleasesForBuild(ctx, build)
}

func (c *client) setPipelinePropertiesFromJSONB(pipeline *contracts.Pipeline, labelsData, releaseTargetsData, commitsData, triggersData, triggeredByEventsData []uint8, optimized bool) (err error) {

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
	if len(triggeredByEventsData) > 0 {
		if err = json.Unmarshal(triggeredByEventsData, &pipeline.Events); err != nil {
			return
		}
	}

	if !optimized {
		// unmarshal then marshal manifest to include defaults
		manifest, err := manifest.ReadManifest(&c.manifestPreferences, pipeline.Manifest)
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

func (c *client) setBuildPropertiesFromJSONB(build *contracts.Build, labelsData, releaseTargetsData, commitsData, triggersData, triggeredByEventsData []uint8, optimized bool) (err error) {

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
	if len(triggeredByEventsData) > 0 {
		if err = json.Unmarshal(triggeredByEventsData, &build.Events); err != nil {
			return
		}
	}

	if !optimized {
		// unmarshal then marshal manifest to include defaults
		manifest, err := manifest.ReadManifest(&c.manifestPreferences, build.Manifest)
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

func (c *client) getLatestReleasesForPipeline(ctx context.Context, pipeline *contracts.Pipeline) {
	pipeline.ReleaseTargets = c.getLatestReleases(ctx, pipeline.ReleaseTargets, pipeline.RepoSource, pipeline.RepoOwner, pipeline.RepoName)
}

func (c *client) getLatestReleasesForBuild(ctx context.Context, build *contracts.Build) {
	build.ReleaseTargets = c.getLatestReleases(ctx, build.ReleaseTargets, build.RepoSource, build.RepoOwner, build.RepoName)
}

func (c *client) getLatestReleases(ctx context.Context, releaseTargets []contracts.ReleaseTarget, repoSource, repoOwner, repoName string) []contracts.ReleaseTarget {

	// set latest release version per release targets
	updatedReleaseTargets := make([]contracts.ReleaseTarget, 0)
	for _, rt := range releaseTargets {

		actions := getActionNamesFromReleaseTarget(rt)
		latestReleases, err := c.GetPipelineLastReleasesByName(ctx, repoSource, repoOwner, repoName, rt.Name, actions)
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

func (c *client) mapBuildToPipeline(build *contracts.Build) (pipeline *contracts.Pipeline) {

	// get archived value from manifest
	mft, err := manifest.ReadManifest(&c.manifestPreferences, build.Manifest)
	archived := false
	if err == nil {
		archived = mft.Archived
	}

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
		Archived:             archived,
		InsertedAt:           build.InsertedAt,
		UpdatedAt:            build.UpdatedAt,
		Duration:             build.Duration,
		Events:               build.Events,
	}
}
