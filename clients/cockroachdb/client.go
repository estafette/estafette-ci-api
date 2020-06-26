package cockroachdb

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	sq "github.com/Masterminds/squirrel"
	"github.com/estafette/estafette-ci-api/config"
	"github.com/estafette/estafette-ci-api/helpers"
	contracts "github.com/estafette/estafette-ci-contracts"
	manifest "github.com/estafette/estafette-ci-manifest"
	foundation "github.com/estafette/estafette-foundation"
	_ "github.com/lib/pq" // use postgres client library to connect to cockroachdb
	"github.com/rs/zerolog/log"
	yaml "gopkg.in/yaml.v2"
)

var (
	// ErrUserNotFound is returned if a query for a user returns no results
	ErrUserNotFound = errors.New("The user can't be found")

	// ErrGroupNotFound is returned if a query for a group returns no results
	ErrGroupNotFound = errors.New("The group can't be found")

	// ErrOrganizationNotFound is returned if a query for a organization returns no results
	ErrOrganizationNotFound = errors.New("The organization can't be found")

	// ErrClientNotFound is returned if a query for a client returns no results
	ErrClientNotFound = errors.New("The client can't be found")

	// ErrCatalogEntityNotFound is returned if a query for a catalog entity returns no results
	ErrCatalogEntityNotFound = errors.New("The catalog entity can't be found")
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
	ArchiveComputedPipeline(ctx context.Context, repoSource, repoOwner, repoName string) (err error)
	UnarchiveComputedPipeline(ctx context.Context, repoSource, repoOwner, repoName string) (err error)

	GetPipelines(ctx context.Context, pageNumber, pageSize int, filters map[string][]string, sortings []helpers.OrderField, optimized bool) (pipelines []*contracts.Pipeline, err error)
	GetPipelinesByRepoName(ctx context.Context, repoName string, optimized bool) (pipelines []*contracts.Pipeline, err error)
	GetPipelinesCount(ctx context.Context, filters map[string][]string) (count int, err error)
	GetPipeline(ctx context.Context, repoSource, repoOwner, repoName string, optimized bool) (pipeline *contracts.Pipeline, err error)
	GetPipelineRecentBuilds(ctx context.Context, repoSource, repoOwner, repoName string, optimized bool) (builds []*contracts.Build, err error)
	GetPipelineBuilds(ctx context.Context, repoSource, repoOwner, repoName string, pageNumber, pageSize int, filters map[string][]string, sortings []helpers.OrderField, optimized bool) (builds []*contracts.Build, err error)
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
	GetPipelineReleases(ctx context.Context, repoSource, repoOwner, repoName string, pageNumber, pageSize int, filters map[string][]string, sortings []helpers.OrderField) (releases []*contracts.Release, err error)
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

	InsertUser(ctx context.Context, user contracts.User) (u *contracts.User, err error)
	UpdateUser(ctx context.Context, user contracts.User) (err error)
	GetUserByIdentity(ctx context.Context, identity contracts.UserIdentity) (user *contracts.User, err error)
	GetUserByID(ctx context.Context, id string) (user *contracts.User, err error)
	GetUsers(ctx context.Context, pageNumber, pageSize int, filters map[string][]string, sortings []helpers.OrderField) (users []*contracts.User, err error)
	GetUsersCount(ctx context.Context, filters map[string][]string) (count int, err error)

	InsertGroup(ctx context.Context, group contracts.Group) (g *contracts.Group, err error)
	UpdateGroup(ctx context.Context, group contracts.Group) (err error)
	GetGroupByIdentity(ctx context.Context, identity contracts.GroupIdentity) (group *contracts.Group, err error)
	GetGroupByID(ctx context.Context, id string) (group *contracts.Group, err error)
	GetGroups(ctx context.Context, pageNumber, pageSize int, filters map[string][]string, sortings []helpers.OrderField) (groups []*contracts.Group, err error)
	GetGroupsCount(ctx context.Context, filters map[string][]string) (count int, err error)

	InsertOrganization(ctx context.Context, organization contracts.Organization) (o *contracts.Organization, err error)
	UpdateOrganization(ctx context.Context, organization contracts.Organization) (err error)
	GetOrganizationByIdentity(ctx context.Context, identity contracts.OrganizationIdentity) (organization *contracts.Organization, err error)
	GetOrganizationByID(ctx context.Context, id string) (organization *contracts.Organization, err error)
	GetOrganizationByName(ctx context.Context, name string) (organization *contracts.Organization, err error)
	GetOrganizations(ctx context.Context, pageNumber, pageSize int, filters map[string][]string, sortings []helpers.OrderField) (organizations []*contracts.Organization, err error)
	GetOrganizationsCount(ctx context.Context, filters map[string][]string) (count int, err error)

	InsertClient(ctx context.Context, client contracts.Client) (cl *contracts.Client, err error)
	UpdateClient(ctx context.Context, client contracts.Client) (err error)
	GetClientByClientID(ctx context.Context, clientID string) (client *contracts.Client, err error)
	GetClientByID(ctx context.Context, id string) (client *contracts.Client, err error)
	GetClients(ctx context.Context, pageNumber, pageSize int, filters map[string][]string, sortings []helpers.OrderField) (clients []*contracts.Client, err error)
	GetClientsCount(ctx context.Context, filters map[string][]string) (count int, err error)

	InsertCatalogEntity(ctx context.Context, catalogEntity contracts.CatalogEntity) (insertedCatalogEntity *contracts.CatalogEntity, err error)
	UpdateCatalogEntity(ctx context.Context, catalogEntity contracts.CatalogEntity) (err error)
	DeleteCatalogEntity(ctx context.Context, id string) (err error)
	GetCatalogEntityByID(ctx context.Context, id string) (catalogEntity *contracts.CatalogEntity, err error)
	GetCatalogEntities(ctx context.Context, pageNumber, pageSize int, filters map[string][]string, sortings []helpers.OrderField) (catalogEntities []*contracts.CatalogEntity, err error)
	GetCatalogEntitiesCount(ctx context.Context, filters map[string][]string) (count int, err error)

	GetCatalogEntityParentKeys(ctx context.Context, pageNumber, pageSize int, filters map[string][]string, sortings []helpers.OrderField) (keys []map[string]interface{}, err error)
	GetCatalogEntityParentKeysCount(ctx context.Context, filters map[string][]string) (count int, err error)
	GetCatalogEntityParentValues(ctx context.Context, pageNumber, pageSize int, filters map[string][]string, sortings []helpers.OrderField) (values []map[string]interface{}, err error)
	GetCatalogEntityParentValuesCount(ctx context.Context, filters map[string][]string) (count int, err error)
	GetCatalogEntityKeys(ctx context.Context, pageNumber, pageSize int, filters map[string][]string, sortings []helpers.OrderField) (keys []map[string]interface{}, err error)
	GetCatalogEntityKeysCount(ctx context.Context, filters map[string][]string) (count int, err error)
	GetCatalogEntityValues(ctx context.Context, pageNumber, pageSize int, filters map[string][]string, sortings []helpers.OrderField) (values []map[string]interface{}, err error)
	GetCatalogEntityValuesCount(ctx context.Context, filters map[string][]string) (count int, err error)
	GetCatalogEntityLabels(ctx context.Context, pageNumber, pageSize int, filters map[string][]string) (labels []map[string]interface{}, err error)
	GetCatalogEntityLabelsCount(ctx context.Context, filters map[string][]string) (count int, err error)
}

// NewClient returns a new cockroach.Client
func NewClient(config *config.APIConfig) Client {
	return &client{
		databaseDriver: "postgres",
		config:         config,
	}
}

type client struct {
	databaseDriver     string
	config             *config.APIConfig
	databaseConnection *sql.DB
}

// Connect sets up a connection with CockroachDB
func (c *client) Connect(ctx context.Context) (err error) {

	log.Debug().Msgf("Connecting to database %v on host %v...", c.config.Database.DatabaseName, c.config.Database.Host)

	dataSourceName := ""
	if c.config.Database.Insecure {
		dataSourceName = fmt.Sprintf("postgresql://%v:%v@%v:%v/%v?sslmode=disable", c.config.Database.User, c.config.Database.Password, c.config.Database.Host, c.config.Database.Port, c.config.Database.DatabaseName)
	} else {
		dataSourceName = fmt.Sprintf("postgresql://%v@%v:%v/%v?sslmode=%v&sslrootcert=%v&sslcert=%v/cert&sslkey=%v/key", c.config.Database.User, c.config.Database.Host, c.config.Database.Port, c.config.Database.DatabaseName, "verify-full", "/var/run/secrets/kubernetes.io/serviceaccount/ca.crt", "/cockroach-certs", "/cockroach-certs")
	}

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
		Where(sq.Eq{"id": buildID}).
		Where(sq.Eq{"repo_source": repoSource}).
		Where(sq.Eq{"repo_owner": repoOwner}).
		Where(sq.Eq{"repo_name": repoName}).
		Where(sq.Eq{"build_status": allowedBuildStatusesToTransitionFrom}).
		Suffix("RETURNING id, repo_source, repo_owner, repo_name, repo_branch, repo_revision, build_version, build_status, labels, release_targets, manifest, commits, triggers, inserted_at, started_at, updated_at, age(COALESCE(started_at, inserted_at), inserted_at)::INT, age(updated_at, COALESCE(started_at,inserted_at))::INT, triggered_by_event")

	if buildStatus == "running" {
		query = query.Set("started_at", sq.Expr("now()"))
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
		Where(sq.Eq{"id": id}).
		Where(sq.Eq{"repo_source": repoSource}).
		Where(sq.Eq{"repo_owner": repoOwner}).
		Where(sq.Eq{"repo_name": repoName}).
		Where(sq.Eq{"release_status": allowedReleaseStatusesToTransitionFrom}).
		Suffix("RETURNING id, repo_source, repo_owner, repo_name, release, release_action, release_version, release_status, inserted_at, started_at, updated_at, age(COALESCE(started_at, inserted_at), inserted_at)::INT, age(updated_at, COALESCE(started_at,inserted_at))::INT, triggered_by_event")

	if releaseStatus == "running" {
		query = query.Set("started_at", sq.Expr("now()"))
	}

	// update release status
	row := query.RunWith(c.databaseConnection).QueryRow()
	insertedRelease, err := c.scanRelease(row)
	if err != nil {
		if err == sql.ErrNoRows {
			log.Warn().Err(err).Msgf("Updating release status for %v/%v/%v id %v from %v to %v is not allowed, no records have been updated", repoSource, repoOwner, repoName, id, allowedReleaseStatusesToTransitionFrom, releaseStatus)
			return nil
		}

		return err
	}

	// update computed tables
	go func(insertedRelease *contracts.Release) {
		if insertedRelease != nil {
			c.UpsertComputedRelease(ctx, repoSource, repoOwner, repoName, insertedRelease.Name, insertedRelease.Action)
		} else {
			log.Warn().Msgf("Cannot update computed tables after updating release status for %v/%v/%v id %v from %v to %v", repoSource, repoOwner, repoName, id, allowedReleaseStatusesToTransitionFrom, releaseStatus)
		}
		c.UpsertComputedPipeline(ctx, repoSource, repoOwner, repoName)
	}(insertedRelease)

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

	// get last x builds
	lastBuilds, err := c.GetPipelineBuilds(ctx, repoSource, repoOwner, repoName, 1, 10, map[string][]string{}, []helpers.OrderField{}, false)
	if err != nil {
		log.Error().Err(err).Msgf("Failed getting last build for upserting computed pipeline %v/%v/%v", repoSource, repoOwner, repoName)
		return
	}
	if lastBuilds == nil || len(lastBuilds) == 0 {
		log.Error().Msgf("Failed getting last build for upserting computed pipeline %v/%v/%v", repoSource, repoOwner, repoName)
		return
	}

	upsertedPipeline := c.mapBuildToPipeline(lastBuilds[0])

	// extract recent committers from last builds
	for _, b := range lastBuilds {
		if len(lastBuilds) > 1 && b.InsertedAt.Before(time.Now().UTC().Add(time.Duration(-7*24)*time.Hour)) {
			continue
		}
		for _, c := range b.Commits {
			if c.Author.Email != "" && !foundation.StringArrayContains(upsertedPipeline.RecentCommitters, c.Author.Email) {
				upsertedPipeline.RecentCommitters = append(upsertedPipeline.RecentCommitters, c.Author.Email)
			}
		}
	}

	upsertedPipeline.ExtraInfo = &contracts.PipelineExtraInfo{}

	// get median (pending) build time from last builds
	buildDurations := []time.Duration{}
	buildPendingDurations := []time.Duration{}
	for _, b := range lastBuilds {
		buildDurations = append(buildDurations, b.Duration)
		if b.PendingDuration != nil {
			buildPendingDurations = append(buildPendingDurations, *b.PendingDuration)
		}
	}
	sort.Slice(buildDurations, func(i, j int) bool {
		return buildDurations[i] < buildDurations[j]
	})
	medianDurationIndex := len(buildDurations)/2 - 1
	if medianDurationIndex < 0 {
		medianDurationIndex = 0
	}
	upsertedPipeline.ExtraInfo.MedianDuration = buildDurations[medianDurationIndex]

	if len(buildPendingDurations) > 0 {
		sort.Slice(buildPendingDurations, func(i, j int) bool {
			return buildPendingDurations[i] < buildPendingDurations[j]
		})
		medianPendingDurationIndex := len(buildPendingDurations)/2 - 1
		if medianPendingDurationIndex < 0 {
			medianPendingDurationIndex = 0
		}
		upsertedPipeline.ExtraInfo.MedianPendingDuration = buildPendingDurations[medianPendingDurationIndex]
	} else {
		upsertedPipeline.ExtraInfo.MedianPendingDuration = time.Duration(0)
	}

	// add releases
	c.enrichPipeline(ctx, upsertedPipeline)

	// get last x releases
	lastReleases, err := c.GetPipelineReleases(ctx, repoSource, repoOwner, repoName, 1, 10, map[string][]string{"since": {"1w"}}, []helpers.OrderField{})
	if err != nil {
		log.Error().Err(err).Msgf("Failed getting last releases for upserting computed pipeline %v/%v/%v", repoSource, repoOwner, repoName)
		return
	}

	if lastReleases != nil {
		// extract recent releasers from last releases
		for _, r := range lastReleases {
			for _, e := range r.Events {
				if e.Manual != nil && e.Manual.UserID != "" && !foundation.StringArrayContains(upsertedPipeline.RecentReleasers, e.Manual.UserID) {
					upsertedPipeline.RecentReleasers = append(upsertedPipeline.RecentReleasers, e.Manual.UserID)
				}
			}
		}
	}

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
	recentCommittersBytes, err := json.Marshal(upsertedPipeline.RecentCommitters)
	if err != nil {
		log.Error().Err(err).Msgf("Failed upserting computed pipeline %v/%v/%v", upsertedPipeline.RepoSource, upsertedPipeline.RepoOwner, upsertedPipeline.RepoName)

		return
	}
	recentReleasersBytes, err := json.Marshal(upsertedPipeline.RecentReleasers)
	if err != nil {
		log.Error().Err(err).Msgf("Failed upserting computed pipeline %v/%v/%v", upsertedPipeline.RepoSource, upsertedPipeline.RepoOwner, upsertedPipeline.RepoName)

		return
	}
	extraInfoBytes, err := json.Marshal(upsertedPipeline.ExtraInfo)
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
			started_at,
			updated_at,
			last_updated_at,
			triggered_by_event,
			recent_committers,
			recent_releasers,
			extra_info
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
			$18,
			$19,
			$20,
			$21,
			$22,
			$23
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
			started_at = excluded.started_at,
			updated_at = excluded.updated_at,
			last_updated_at = excluded.last_updated_at,
			triggered_by_event = excluded.triggered_by_event,
			recent_committers = excluded.recent_committers,
			recent_releasers = excluded.recent_releasers,
			extra_info = excluded.extra_info
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
		upsertedPipeline.StartedAt,
		upsertedPipeline.UpdatedAt,
		upsertedPipeline.LastUpdatedAt,
		eventsBytes,
		recentCommittersBytes,
		recentReleasersBytes,
		extraInfoBytes,
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
			started_at,
			updated_at,
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
			$10,
			$11,
			$12
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
			started_at = excluded.started_at,
			updated_at = excluded.updated_at,
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
		lastRelease.StartedAt,
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

func (c *client) ArchiveComputedPipeline(ctx context.Context, repoSource, repoOwner, repoName string) (err error) {

	psql := sq.StatementBuilder.PlaceholderFormat(sq.Dollar)

	query := psql.
		Update("computed_pipelines").
		Set("archived", true).
		Where(sq.Eq{"repo_source": repoSource}).
		Where(sq.Eq{"repo_owner": repoOwner}).
		Where(sq.Eq{"repo_name": repoName}).
		Where(sq.Eq{"archived": false}).
		Limit(uint64(1))

	_, err = query.RunWith(c.databaseConnection).Exec()
	if err != nil {
		return err
	}

	return nil
}

func (c *client) UnarchiveComputedPipeline(ctx context.Context, repoSource, repoOwner, repoName string) (err error) {

	psql := sq.StatementBuilder.PlaceholderFormat(sq.Dollar)

	query := psql.
		Update("computed_pipelines").
		Set("archived", false).
		Where(sq.Eq{"repo_source": repoSource}).
		Where(sq.Eq{"repo_owner": repoOwner}).
		Where(sq.Eq{"repo_name": repoName}).
		Where(sq.Eq{"archived": true}).
		Limit(uint64(1))

	_, err = query.RunWith(c.databaseConnection).Exec()
	if err != nil {
		return err
	}

	return nil
}
func (c *client) GetPipelines(ctx context.Context, pageNumber, pageSize int, filters map[string][]string, sortings []helpers.OrderField, optimized bool) (pipelines []*contracts.Pipeline, err error) {

	// generate query
	query := c.selectPipelinesQuery().
		Where(sq.Eq{"a.archived": false}).
		Limit(uint64(pageSize)).
		Offset(uint64((pageNumber - 1) * pageSize))

	// dynamically set order by clause
	query, err = orderByClauseGeneratorForSortings(query, "a", "a.repo_source,a.repo_owner,a.repo_name", sortings)
	if err != nil {

		return
	}

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
		Select("a.id, a.repo_source, a.repo_owner, a.repo_name, a.repo_branch, a.repo_revision, a.build_version, a.build_status, a.labels, a.release_targets, a.manifest, a.commits, a.triggers, a.inserted_at, a.started_at, a.updated_at, age(COALESCE(a.started_at, a.inserted_at), a.inserted_at)::INT, age(a.updated_at, COALESCE(a.started_at,a.inserted_at))::INT, a.triggered_by_event").
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

func (c *client) GetPipelineBuilds(ctx context.Context, repoSource, repoOwner, repoName string, pageNumber, pageSize int, filters map[string][]string, sortings []helpers.OrderField, optimized bool) (builds []*contracts.Build, err error) {

	// generate query
	query := c.selectBuildsQuery().
		Where(sq.Eq{"a.repo_source": repoSource}).
		Where(sq.Eq{"a.repo_owner": repoOwner}).
		Where(sq.Eq{"a.repo_name": repoName}).
		Limit(uint64(pageSize)).
		Offset(uint64((pageNumber - 1) * pageSize))

	// dynamically set order by clause
	query, err = orderByClauseGeneratorForSortings(query, "a", "a.inserted_at DESC", sortings)
	if err != nil {

		return
	}

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
		if err == sql.ErrNoRows {
			return nil, nil
		}

		return nil, err
	}

	return release, nil
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
		if err == sql.ErrNoRows {
			return nil, nil
		}

		return nil, err
	}

	return release, nil

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

func (c *client) GetPipelineReleases(ctx context.Context, repoSource, repoOwner, repoName string, pageNumber, pageSize int, filters map[string][]string, sortings []helpers.OrderField) (releases []*contracts.Release, err error) {

	// generate query
	query := c.selectReleasesQuery().
		Where(sq.Eq{"a.repo_source": repoSource}).
		Where(sq.Eq{"a.repo_owner": repoOwner}).
		Where(sq.Eq{"a.repo_name": repoName}).
		Limit(uint64(pageSize)).
		Offset(uint64((pageNumber - 1) * pageSize))

		// dynamically set order by clause
	query, err = orderByClauseGeneratorForSortings(query, "a", "a.inserted_at DESC", sortings)
	if err != nil {

		return
	}

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
		if err == sql.ErrNoRows {
			return nil, nil
		}

		return nil, err
	}

	return release, nil
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
			Select("a.inserted_at, age(COALESCE(a.started_at, a.inserted_at), a.inserted_at)::INT AS pending_duration, age(a.updated_at, COALESCE(a.started_at,a.inserted_at))::INT as duration").
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
		var durationPendingSeconds, durationRunningSeconds int

		if err = rows.Scan(
			&insertedAt,
			&durationPendingSeconds,
			&durationRunningSeconds); err != nil {

			return
		}

		pendingDuration := time.Duration(durationPendingSeconds) * time.Second
		runningDuration := time.Duration(durationRunningSeconds) * time.Second

		durations = append(durations, map[string]interface{}{
			"insertedAt":      insertedAt,
			"pendingDuration": pendingDuration,
			"duration":        runningDuration,
		})
	}

	return
}

func (c *client) GetPipelineReleasesDurations(ctx context.Context, repoSource, repoOwner, repoName string, filters map[string][]string) (durations []map[string]interface{}, err error) {

	// generate query
	innerquery :=
		sq.StatementBuilder.PlaceholderFormat(sq.Dollar).
			Select("a.inserted_at, a.release, a.release_action, age(COALESCE(a.started_at, a.inserted_at), a.inserted_at)::INT as pending_duration, age(a.updated_at, COALESCE(a.started_at,a.inserted_at))::INT as duration").
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
		var durationPendingSeconds, durationRunningSeconds int

		if err = rows.Scan(
			&insertedAt,
			&releaseName,
			&releaseAction,
			&durationPendingSeconds,
			&durationRunningSeconds); err != nil {

			return
		}

		pendingDuration := time.Duration(durationPendingSeconds) * time.Second
		runningDuration := time.Duration(durationRunningSeconds) * time.Second

		durations = append(durations, map[string]interface{}{
			"insertedAt":      insertedAt,
			"name":            releaseName,
			"action":          releaseAction,
			"pendingDuration": pendingDuration,
			"duration":        runningDuration,
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

	psql := sq.StatementBuilder.PlaceholderFormat(sq.Dollar)

	arrayElementsQuery :=
		psql.
			Select("a.id, jsonb_array_elements(a.labels) AS l").
			From("computed_pipelines a").
			Where("jsonb_typeof(labels) = 'array'").
			Where(sq.Eq{"a.archived": false})

	selectCountQuery :=
		psql.
			Select("l->>'key' AS key, l->>'value' AS value, id").
			FromSelect(arrayElementsQuery, "b").
			Where(sq.Eq{"l->>'key'": labelKey})

	groupByQuery :=
		psql.
			Select("key, value, count(DISTINCT id) AS pipelinesCount").
			FromSelect(selectCountQuery, "c").
			GroupBy("key, value")

	query :=
		psql.
			Select("key, value, pipelinesCount").
			FromSelect(groupByQuery, "d").
			OrderBy("value")

	rows, err := query.RunWith(c.databaseConnection).Query()
	if err != nil {
		return
	}
	defer rows.Close()

	return c.scanItems(ctx, rows)
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

	psql := sq.StatementBuilder.PlaceholderFormat(sq.Dollar)

	arrayElementsQuery :=
		psql.
			Select("a.id, jsonb_array_elements(a.labels) AS l").
			From("computed_pipelines a").
			Where("jsonb_typeof(labels) = 'array'").
			Where(sq.Eq{"a.archived": false})

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
		psql.
			Select("l->>'key' AS key, l->>'value' AS value, id").
			FromSelect(arrayElementsQuery, "b")

	groupByQuery :=
		psql.
			Select("key, value, count(DISTINCT id) AS pipelinesCount").
			FromSelect(selectCountQuery, "c").
			GroupBy("key, value")

	query :=
		psql.
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

	return c.scanItems(ctx, rows)
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

	psql := sq.StatementBuilder.PlaceholderFormat(sq.Dollar)

	arrayElementsQuery :=
		psql.
			Select("a.id, jsonb_array_elements(a.labels) AS l").
			From("computed_pipelines a").
			Where("jsonb_typeof(labels) = 'array'").
			Where(sq.Eq{"a.archived": false})

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
		psql.
			Select("l->>'key' AS key, l->>'value' AS value, id").
			FromSelect(arrayElementsQuery, "b")

	groupByQuery :=
		psql.
			Select("key, value, count(DISTINCT id) AS pipelinesCount").
			FromSelect(selectCountQuery, "c").
			GroupBy("key, value")

	query :=
		psql.
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

	rows, err := query.RunWith(c.databaseConnection).Query()
	if err != nil {
		return
	}
	defer rows.Close()

	return c.scanItems(ctx, rows)
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

	rows, err := query.RunWith(c.databaseConnection).Query()
	if err != nil {
		return
	}
	defer rows.Close()

	return c.scanItems(ctx, rows)
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

func orderByClauseGeneratorForSortings(query sq.SelectBuilder, alias, defaultOrderBy string, sortings []helpers.OrderField) (sq.SelectBuilder, error) {

	if len(sortings) == 0 {
		return query.OrderBy(defaultOrderBy), nil
	}

	for _, s := range sortings {
		query = query.OrderBy(fmt.Sprintf("%v.%v %v", alias, foundation.ToLowerSnakeCase(s.FieldName), s.Direction))
	}

	return query, nil
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
	query, err = whereClauseGeneratorForRecentCommitterFilter(query, alias, filters)
	if err != nil {
		return query, err
	}
	query, err = whereClauseGeneratorForRecentReleaserFilter(query, alias, filters)
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

func whereClauseGeneratorForRecentCommitterFilter(query sq.SelectBuilder, alias string, filters map[string][]string) (sq.SelectBuilder, error) {

	if user, ok := filters["recent-committer"]; ok && len(user) > 0 {

		userParam := []string{user[0]}

		bytes, err := json.Marshal(userParam)
		if err != nil {
			return query, err
		}

		query = query.Where(fmt.Sprintf("%v.recent_committers @> ?", alias), string(bytes))
	}

	return query, nil
}

func whereClauseGeneratorForRecentReleaserFilter(query sq.SelectBuilder, alias string, filters map[string][]string) (sq.SelectBuilder, error) {

	if user, ok := filters["recent-releaser"]; ok && len(user) > 0 {

		userParam := []string{user[0]}

		bytes, err := json.Marshal(userParam)
		if err != nil {
			return query, err
		}

		query = query.Where(fmt.Sprintf("%v.recent_releasers @> ?", alias), string(bytes))
	}

	return query, nil
}

func whereClauseGeneratorForUserFilters(query sq.SelectBuilder, alias string, filters map[string][]string) (sq.SelectBuilder, error) {

	query, err := whereClauseGeneratorForUserGroupFilters(query, alias, filters)
	if err != nil {
		return query, err
	}

	query, err = whereClauseGeneratorForUserOrganizationFilters(query, alias, filters)
	if err != nil {
		return query, err
	}

	return query, nil
}

func whereClauseGeneratorForUserGroupFilters(query sq.SelectBuilder, alias string, filters map[string][]string) (sq.SelectBuilder, error) {
	if groupIDs, ok := filters["group-id"]; ok && len(groupIDs) > 0 {

		groupID := groupIDs[0]

		filter := struct {
			Groups []struct {
				ID string `json:"id"`
			} `json:"groups"`
		}{
			[]struct {
				ID string `json:"id"`
			}{
				{
					ID: groupID,
				},
			},
		}

		filterBytes, err := json.Marshal(filter)
		if err != nil {
			return query, err
		}

		query = query.
			Where(fmt.Sprintf("%v.user_data @> ?", alias), string(filterBytes))
	}

	return query, nil
}

func whereClauseGeneratorForUserOrganizationFilters(query sq.SelectBuilder, alias string, filters map[string][]string) (sq.SelectBuilder, error) {
	if organizationIDs, ok := filters["organization-id"]; ok && len(organizationIDs) > 0 {

		organizationID := organizationIDs[0]

		filter := struct {
			Organizations []struct {
				ID string `json:"id"`
			} `json:"organizations"`
		}{
			[]struct {
				ID string `json:"id"`
			}{
				{
					ID: organizationID,
				},
			},
		}

		filterBytes, err := json.Marshal(filter)
		if err != nil {
			return query, err
		}

		query = query.
			Where(fmt.Sprintf("%v.user_data @> ?", alias), string(filterBytes))
	}

	return query, nil
}

func whereClauseGeneratorForCatalogEntityFilters(query sq.SelectBuilder, alias string, filters map[string][]string) (sq.SelectBuilder, error) {

	query, err := whereClauseGeneratorForParentFilter(query, alias, filters)
	if err != nil {
		return query, err
	}

	query, err = whereClauseGeneratorForEntityFilter(query, alias, filters)
	if err != nil {
		return query, err
	}

	query, err = whereClauseGeneratorForLinkedPipelineFilter(query, alias, filters)
	if err != nil {
		return query, err
	}

	query, err = whereClauseGeneratorForLabelsFilter(query, alias, filters)
	if err != nil {
		return query, err
	}

	return query, nil
}

func whereClauseGeneratorForParentFilter(query sq.SelectBuilder, alias string, filters map[string][]string) (sq.SelectBuilder, error) {

	if parents, ok := filters["parent"]; ok && len(parents) == 1 {
		keyValuePair := strings.Split(parents[0], "=")
		if len(keyValuePair) > 0 {
			query = query.Where(sq.Eq{fmt.Sprintf("%v.parent_key", alias): keyValuePair[0]})
		}
		if len(keyValuePair) > 1 {
			query = query.Where(sq.Eq{fmt.Sprintf("%v.parent_value", alias): keyValuePair[1]})
		}
	}

	return query, nil
}

func whereClauseGeneratorForEntityFilter(query sq.SelectBuilder, alias string, filters map[string][]string) (sq.SelectBuilder, error) {

	if entities, ok := filters["entity"]; ok && len(entities) == 1 {
		keyValuePair := strings.Split(entities[0], "=")
		if len(keyValuePair) > 0 {
			query = query.Where(sq.Eq{fmt.Sprintf("%v.entity_key", alias): keyValuePair[0]})
		}
		if len(keyValuePair) > 1 {
			query = query.Where(sq.Eq{fmt.Sprintf("%v.entity_value", alias): keyValuePair[1]})
		}
	}

	return query, nil
}

func whereClauseGeneratorForLinkedPipelineFilter(query sq.SelectBuilder, alias string, filters map[string][]string) (sq.SelectBuilder, error) {

	if pipelines, ok := filters["pipeline"]; ok && len(pipelines) == 1 {
		query = query.Where(sq.Eq{fmt.Sprintf("%v.linked_pipeline", alias): pipelines[0]})
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

func (c *client) scanItems(ctx context.Context, rows *sql.Rows) (items []map[string]interface{}, err error) {

	items = make([]map[string]interface{}, 0)

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

		items = append(items, m)
	}

	return items, nil
}

func (c *client) scanBuild(ctx context.Context, row sq.RowScanner, optimized, enriched bool) (build *contracts.Build, err error) {

	build = &contracts.Build{}
	var labelsData, releaseTargetsData, commitsData, triggersData, triggeredByEventsData []uint8
	var durationPendingSeconds, durationRunningSeconds int

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
		&build.StartedAt,
		&build.UpdatedAt,
		&durationPendingSeconds,
		&durationRunningSeconds,
		&triggeredByEventsData); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return
	}

	pendingDuration := time.Duration(durationPendingSeconds) * time.Second
	runningDuration := time.Duration(durationRunningSeconds) * time.Second

	build.PendingDuration = &pendingDuration
	build.Duration = runningDuration

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
		var durationPendingSeconds, durationRunningSeconds int

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
			&build.StartedAt,
			&build.UpdatedAt,
			&durationPendingSeconds,
			&durationRunningSeconds,
			&triggeredByEventsData); err != nil {
			return
		}

		pendingDuration := time.Duration(durationPendingSeconds) * time.Second
		runningDuration := time.Duration(durationRunningSeconds) * time.Second

		build.PendingDuration = &pendingDuration
		build.Duration = runningDuration

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
	var labelsData, releaseTargetsData, commitsData, triggersData, triggeredByEventsData, extraInfoData []uint8
	var durationPendingSeconds, durationRunningSeconds int

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
		&pipeline.StartedAt,
		&pipeline.UpdatedAt,
		&durationPendingSeconds,
		&durationRunningSeconds,
		&pipeline.LastUpdatedAt,
		&triggeredByEventsData,
		&extraInfoData); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return
	}

	pendingDuration := time.Duration(durationPendingSeconds) * time.Second
	runningDuration := time.Duration(durationRunningSeconds) * time.Second

	pipeline.PendingDuration = &pendingDuration
	pipeline.Duration = runningDuration

	c.setPipelinePropertiesFromJSONB(pipeline, labelsData, releaseTargetsData, commitsData, triggersData, triggeredByEventsData, extraInfoData, optimized)

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
		var labelsData, releaseTargetsData, commitsData, triggersData, triggeredByEventsData, extraInfoData []uint8
		var durationPendingSeconds, durationRunningSeconds int

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
			&pipeline.StartedAt,
			&pipeline.UpdatedAt,
			&durationPendingSeconds,
			&durationRunningSeconds,
			&pipeline.LastUpdatedAt,
			&triggeredByEventsData,
			&extraInfoData); err != nil {
			return
		}

		pendingDuration := time.Duration(durationPendingSeconds) * time.Second
		runningDuration := time.Duration(durationRunningSeconds) * time.Second

		pipeline.PendingDuration = &pendingDuration
		pipeline.Duration = runningDuration

		c.setPipelinePropertiesFromJSONB(&pipeline, labelsData, releaseTargetsData, commitsData, triggersData, triggeredByEventsData, extraInfoData, optimized)

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
	var durationPendingSeconds, durationRunningSeconds int
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
		&release.StartedAt,
		&release.UpdatedAt,
		&durationPendingSeconds,
		&durationRunningSeconds,
		&triggeredByEventsData); err != nil {
		return nil, err
	}

	pendingDuration := time.Duration(durationPendingSeconds) * time.Second
	runningDuration := time.Duration(durationRunningSeconds) * time.Second

	release.PendingDuration = &pendingDuration
	release.Duration = &runningDuration
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
		var durationPendingSeconds, durationRunningSeconds int
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
			&release.StartedAt,
			&release.UpdatedAt,
			&durationPendingSeconds,
			&durationRunningSeconds,
			&triggeredByEventsData); err != nil {
			return
		}

		pendingDuration := time.Duration(durationPendingSeconds) * time.Second
		runningDuration := time.Duration(durationRunningSeconds) * time.Second

		release.PendingDuration = &pendingDuration
		release.Duration = &runningDuration
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

func (c *client) InsertUser(ctx context.Context, user contracts.User) (u *contracts.User, err error) {

	userBytes, err := json.Marshal(user)
	if err != nil {
		return nil, err
	}

	// upsert user
	row := c.databaseConnection.QueryRow(
		`
		INSERT INTO
			users
		(
			user_data
		)
		VALUES
		(
			$1
		)
		RETURNING
			id
		`,
		userBytes,
	)

	u = &user

	if err = row.Scan(&u.ID); err != nil {
		return nil, err
	}

	return
}

func (c *client) UpdateUser(ctx context.Context, user contracts.User) (err error) {

	userBytes, err := json.Marshal(user)
	if err != nil {
		return
	}

	userID, err := strconv.Atoi(user.ID)
	if err != nil {
		return
	}

	psql := sq.StatementBuilder.PlaceholderFormat(sq.Dollar)

	query := psql.
		Update("users").
		Set("user_data", userBytes).
		Set("updated_at", sq.Expr("now()")).
		Where(sq.Eq{"id": userID}).
		Limit(uint64(1))

	_, err = query.RunWith(c.databaseConnection).Exec()

	return
}

func (c *client) GetUserByID(ctx context.Context, id string) (user *contracts.User, err error) {

	psql := sq.StatementBuilder.PlaceholderFormat(sq.Dollar)

	query := psql.
		Select("a.id, a.user_data, a.inserted_at").
		From("users a").
		Where(sq.Eq{"a.id": id}).
		Limit(uint64(1))

	// execute query
	row := query.RunWith(c.databaseConnection).QueryRow()
	user, err = c.scanUser(row)
	if err != nil {
		return nil, err
	}

	return user, nil
}

func (c *client) GetUserByIdentity(ctx context.Context, identity contracts.UserIdentity) (user *contracts.User, err error) {

	filter := struct {
		Identities []struct {
			Provider string `json:"provider"`
			Email    string `json:"email"`
		} `json:"identities"`
	}{
		[]struct {
			Provider string `json:"provider"`
			Email    string `json:"email"`
		}{
			{
				Provider: identity.Provider,
				Email:    identity.Email,
			},
		},
	}

	filterBytes, err := json.Marshal(filter)
	if err != nil {
		return nil, err
	}

	psql := sq.StatementBuilder.PlaceholderFormat(sq.Dollar)

	query := psql.
		Select("a.id, a.user_data, a.inserted_at").
		From("users a").
		Where("a.user_data @> ?", string(filterBytes)).
		Limit(uint64(1))

	// execute query
	row := query.RunWith(c.databaseConnection).QueryRow()
	user, err = c.scanUser(row)
	if err != nil {
		return nil, err
	}

	return user, nil
}

func (c *client) GetUsers(ctx context.Context, pageNumber, pageSize int, filters map[string][]string, sortings []helpers.OrderField) (users []*contracts.User, err error) {
	psql := sq.StatementBuilder.PlaceholderFormat(sq.Dollar)

	query := psql.
		Select("a.id, a.user_data, a.inserted_at").
		From("users a").
		Limit(uint64(pageSize)).
		Offset(uint64((pageNumber - 1) * pageSize))

	// wait for https://github.com/cockroachdb/cockroach/issues/35706 to be implemented for sorting jsonb fields

	// // fix sortings for fields inside the user_data jsonb object
	// fixedSortings := []helpers.OrderField{}
	// for _, s := range sortings {
	// 	fieldName := s.FieldName
	// 	direction := s.Direction

	// 	switch s.FieldName {
	// 	case "name":
	// 		fieldName = "user_data-->'name'"
	// 	case "email":
	// 		fieldName = "user_data-->'email'"
	// 	}

	// 	fixedSortings = append(fixedSortings, helpers.OrderField{
	// 		FieldName: fieldName,
	// 		Direction: direction,
	// 	})
	// }

	// // dynamically set order by clause
	// query, err = orderByClauseGeneratorForSortings(query, "a", "a.user_data-->'name'", fixedSortings)
	// if err != nil {
	// 	return
	// }

	query, err = whereClauseGeneratorForUserFilters(query, "a", filters)

	// execute query
	rows, err := query.RunWith(c.databaseConnection).Query()
	if err != nil {
		return
	}

	return c.scanUsers(rows)
}

func (c *client) GetUsersCount(ctx context.Context, filters map[string][]string) (count int, err error) {

	psql := sq.StatementBuilder.PlaceholderFormat(sq.Dollar)

	query := psql.
		Select("COUNT(a.id)").
		From("users a")

	// execute query
	row := query.RunWith(c.databaseConnection).QueryRow()
	if err = row.Scan(&count); err != nil {
		return
	}

	return
}

func (c *client) InsertGroup(ctx context.Context, group contracts.Group) (g *contracts.Group, err error) {

	groupBytes, err := json.Marshal(group)
	if err != nil {
		return nil, err
	}

	// upsert user
	row := c.databaseConnection.QueryRow(
		`
		INSERT INTO
			groups
		(
			group_data
		)
		VALUES
		(
			$1
		)
		RETURNING
			id
		`,
		groupBytes,
	)

	g = &group

	if err = row.Scan(&g.ID); err != nil {
		return nil, err
	}

	return
}

func (c *client) UpdateGroup(ctx context.Context, group contracts.Group) (err error) {
	groupBytes, err := json.Marshal(group)
	if err != nil {
		return
	}

	groupID, err := strconv.Atoi(group.ID)
	if err != nil {
		return
	}

	psql := sq.StatementBuilder.PlaceholderFormat(sq.Dollar)

	query := psql.
		Update("groups").
		Set("group_data", groupBytes).
		Set("updated_at", sq.Expr("now()")).
		Where(sq.Eq{"id": groupID}).
		Limit(uint64(1))

	_, err = query.RunWith(c.databaseConnection).Exec()

	return
}

func (c *client) GetGroupByIdentity(ctx context.Context, identity contracts.GroupIdentity) (group *contracts.Group, err error) {
	filter := struct {
		Identities []struct {
			Provider string `json:"provider"`
			Name     string `json:"name"`
		} `json:"identities"`
	}{
		[]struct {
			Provider string `json:"provider"`
			Name     string `json:"name"`
		}{
			{
				Provider: identity.Provider,
				Name:     identity.Name,
			},
		},
	}

	filterBytes, err := json.Marshal(filter)
	if err != nil {
		return nil, err
	}

	psql := sq.StatementBuilder.PlaceholderFormat(sq.Dollar)

	query := psql.
		Select("a.id, a.group_data, a.inserted_at").
		From("groups a").
		Where("a.group_data @> ?", string(filterBytes)).
		Limit(uint64(1))

	// execute query
	row := query.RunWith(c.databaseConnection).QueryRow()
	group, err = c.scanGroup(row)
	if err != nil {
		return nil, err
	}

	return group, nil
}

func (c *client) GetGroupByID(ctx context.Context, id string) (group *contracts.Group, err error) {
	psql := sq.StatementBuilder.PlaceholderFormat(sq.Dollar)

	query := psql.
		Select("a.id, a.group_data, a.inserted_at").
		From("groups a").
		Where(sq.Eq{"a.id": id}).
		Limit(uint64(1))

	// execute query
	row := query.RunWith(c.databaseConnection).QueryRow()
	group, err = c.scanGroup(row)
	if err != nil {
		return nil, err
	}

	return group, nil
}

func (c *client) GetGroups(ctx context.Context, pageNumber, pageSize int, filters map[string][]string, sortings []helpers.OrderField) (groups []*contracts.Group, err error) {
	psql := sq.StatementBuilder.PlaceholderFormat(sq.Dollar)

	query := psql.
		Select("a.id, a.group_data, a.inserted_at").
		From("groups a").
		Limit(uint64(pageSize)).
		Offset(uint64((pageNumber - 1) * pageSize))

	// execute query
	rows, err := query.RunWith(c.databaseConnection).Query()
	if err != nil {
		return
	}

	return c.scanGroups(rows)
}

func (c *client) GetGroupsCount(ctx context.Context, filters map[string][]string) (count int, err error) {
	psql := sq.StatementBuilder.PlaceholderFormat(sq.Dollar)

	query := psql.
		Select("COUNT(a.id)").
		From("groups a")

	// execute query
	row := query.RunWith(c.databaseConnection).QueryRow()
	if err = row.Scan(&count); err != nil {
		return
	}

	return
}

func (c *client) InsertOrganization(ctx context.Context, organization contracts.Organization) (o *contracts.Organization, err error) {

	organizationBytes, err := json.Marshal(organization)
	if err != nil {
		return nil, err
	}

	// upsert user
	row := c.databaseConnection.QueryRow(
		`
		INSERT INTO
		organizations
		(
			organization_data
		)
		VALUES
		(
			$1
		)
		RETURNING
			id
		`,
		organizationBytes,
	)

	o = &organization

	if err = row.Scan(&o.ID); err != nil {
		return nil, err
	}

	return
}

func (c *client) UpdateOrganization(ctx context.Context, organization contracts.Organization) (err error) {
	organizationBytes, err := json.Marshal(organization)
	if err != nil {
		return
	}

	organizationID, err := strconv.Atoi(organization.ID)
	if err != nil {
		return
	}

	psql := sq.StatementBuilder.PlaceholderFormat(sq.Dollar)

	query := psql.
		Update("organizations").
		Set("organization_data", organizationBytes).
		Set("updated_at", sq.Expr("now()")).
		Where(sq.Eq{"id": organizationID}).
		Limit(uint64(1))

	_, err = query.RunWith(c.databaseConnection).Exec()

	return
}

func (c *client) GetOrganizationByIdentity(ctx context.Context, identity contracts.OrganizationIdentity) (organization *contracts.Organization, err error) {
	filter := struct {
		Identities []struct {
			Provider string `json:"provider"`
			Name     string `json:"name"`
		} `json:"identities"`
	}{
		[]struct {
			Provider string `json:"provider"`
			Name     string `json:"name"`
		}{
			{
				Provider: identity.Provider,
				Name:     identity.Name,
			},
		},
	}

	filterBytes, err := json.Marshal(filter)
	if err != nil {
		return nil, err
	}

	psql := sq.StatementBuilder.PlaceholderFormat(sq.Dollar)

	query := psql.
		Select("a.id, a.organization_data, a.inserted_at").
		From("organizations a").
		Where("a.organization_data @> ?", string(filterBytes)).
		Limit(uint64(1))

	// execute query
	row := query.RunWith(c.databaseConnection).QueryRow()
	organization, err = c.scanOrganization(row)
	if err != nil {
		return nil, err
	}

	return organization, nil
}

func (c *client) GetOrganizationByID(ctx context.Context, id string) (organization *contracts.Organization, err error) {
	psql := sq.StatementBuilder.PlaceholderFormat(sq.Dollar)

	query := psql.
		Select("a.id, a.organization_data, a.inserted_at").
		From("organizations a").
		Where(sq.Eq{"a.id": id}).
		Limit(uint64(1))

	// execute query
	row := query.RunWith(c.databaseConnection).QueryRow()
	organization, err = c.scanOrganization(row)
	if err != nil {
		return nil, err
	}

	return organization, nil
}

func (c *client) GetOrganizationByName(ctx context.Context, name string) (organization *contracts.Organization, err error) {
	psql := sq.StatementBuilder.PlaceholderFormat(sq.Dollar)

	query := psql.
		Select("a.id, a.organization_data, a.inserted_at").
		From("organizations a").
		Where(sq.Eq{"a.organization_data->>'name'": name}).
		Limit(uint64(1))

	// execute query
	row := query.RunWith(c.databaseConnection).QueryRow()
	organization, err = c.scanOrganization(row)
	if err != nil {
		return nil, err
	}

	return organization, nil
}

func (c *client) GetOrganizations(ctx context.Context, pageNumber, pageSize int, filters map[string][]string, sortings []helpers.OrderField) (organizations []*contracts.Organization, err error) {
	psql := sq.StatementBuilder.PlaceholderFormat(sq.Dollar)

	query := psql.
		Select("a.id, a.organization_data, a.inserted_at").
		From("organizations a").
		Limit(uint64(pageSize)).
		Offset(uint64((pageNumber - 1) * pageSize))

	// execute query
	rows, err := query.RunWith(c.databaseConnection).Query()
	if err != nil {
		return
	}

	return c.scanOrganizations(rows)
}

func (c *client) GetOrganizationsCount(ctx context.Context, filters map[string][]string) (count int, err error) {
	psql := sq.StatementBuilder.PlaceholderFormat(sq.Dollar)

	query := psql.
		Select("COUNT(a.id)").
		From("organizations a")

	// execute query
	row := query.RunWith(c.databaseConnection).QueryRow()
	if err = row.Scan(&count); err != nil {
		return
	}

	return
}

func (c *client) InsertClient(ctx context.Context, client contracts.Client) (cl *contracts.Client, err error) {

	clientBytes, err := json.Marshal(client)
	if err != nil {
		return nil, err
	}

	// upsert user
	row := c.databaseConnection.QueryRow(
		`
		INSERT INTO
		clients
		(
			client_data
		)
		VALUES
		(
			$1
		)
		RETURNING
			id
		`,
		clientBytes,
	)

	cl = &client

	if err = row.Scan(&cl.ID); err != nil {
		return nil, err
	}

	return
}

func (c *client) UpdateClient(ctx context.Context, client contracts.Client) (err error) {
	clientBytes, err := json.Marshal(client)
	if err != nil {
		return
	}

	clientID, err := strconv.Atoi(client.ID)
	if err != nil {
		return
	}

	psql := sq.StatementBuilder.PlaceholderFormat(sq.Dollar)

	query := psql.
		Update("clients").
		Set("client_data", clientBytes).
		Set("updated_at", sq.Expr("now()")).
		Where(sq.Eq{"id": clientID}).
		Limit(uint64(1))

	_, err = query.RunWith(c.databaseConnection).Exec()

	return
}

func (c *client) GetClientByClientID(ctx context.Context, clientID string) (client *contracts.Client, err error) {
	psql := sq.StatementBuilder.PlaceholderFormat(sq.Dollar)

	query := psql.
		Select("a.id, a.client_data, a.inserted_at").
		From("clients a").
		Where(sq.Eq{"a.client_data->>'clientID'": clientID}).
		Limit(uint64(1))

	// execute query
	row := query.RunWith(c.databaseConnection).QueryRow()
	client, err = c.scanClient(row)
	if err != nil {
		return nil, err
	}

	return client, nil
}

func (c *client) GetClientByID(ctx context.Context, id string) (client *contracts.Client, err error) {
	psql := sq.StatementBuilder.PlaceholderFormat(sq.Dollar)

	query := psql.
		Select("a.id, a.client_data, a.inserted_at").
		From("clients a").
		Where(sq.Eq{"a.id": id}).
		Limit(uint64(1))

	// execute query
	row := query.RunWith(c.databaseConnection).QueryRow()
	client, err = c.scanClient(row)
	if err != nil {
		return nil, err
	}

	return client, nil
}

func (c *client) GetClients(ctx context.Context, pageNumber, pageSize int, filters map[string][]string, sortings []helpers.OrderField) (clients []*contracts.Client, err error) {
	psql := sq.StatementBuilder.PlaceholderFormat(sq.Dollar)

	query := psql.
		Select("a.id, a.client_data, a.inserted_at").
		From("clients a").
		Limit(uint64(pageSize)).
		Offset(uint64((pageNumber - 1) * pageSize))

	// execute query
	rows, err := query.RunWith(c.databaseConnection).Query()
	if err != nil {
		return
	}

	return c.scanClients(rows)
}

func (c *client) GetClientsCount(ctx context.Context, filters map[string][]string) (count int, err error) {
	psql := sq.StatementBuilder.PlaceholderFormat(sq.Dollar)

	query := psql.
		Select("COUNT(a.id)").
		From("clients a")

	// execute query
	row := query.RunWith(c.databaseConnection).QueryRow()
	if err = row.Scan(&count); err != nil {
		return
	}

	return
}

func (c *client) InsertCatalogEntity(ctx context.Context, catalogEntity contracts.CatalogEntity) (insertedCatalogEntity *contracts.CatalogEntity, err error) {

	labelBytes, err := json.Marshal(catalogEntity.Labels)
	if err != nil {
		return nil, err
	}

	metadataBytes, err := json.Marshal(catalogEntity.Metadata)
	if err != nil {
		return nil, err
	}

	// upsert user
	row := c.databaseConnection.QueryRow(
		`
		INSERT INTO
		catalog_entities
		(
			parent_key,
			parent_value,
			entity_key,
			entity_value,
			linked_pipeline,
			labels,
			entity_metadata
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
		catalogEntity.ParentKey,
		catalogEntity.ParentValue,
		catalogEntity.Key,
		catalogEntity.Value,
		catalogEntity.LinkedPipeline,
		labelBytes,
		metadataBytes,
	)

	insertedCatalogEntity = &catalogEntity

	if err = row.Scan(&insertedCatalogEntity.ID); err != nil {
		return nil, err
	}

	return
}

func (c *client) UpdateCatalogEntity(ctx context.Context, catalogEntity contracts.CatalogEntity) (err error) {
	labelBytes, err := json.Marshal(catalogEntity.Labels)
	if err != nil {
		return
	}

	metadataBytes, err := json.Marshal(catalogEntity.Metadata)
	if err != nil {
		return
	}

	entityID, err := strconv.Atoi(catalogEntity.ID)
	if err != nil {
		return
	}

	psql := sq.StatementBuilder.PlaceholderFormat(sq.Dollar)

	query := psql.
		Update("catalog_entities").
		Set("linked_pipeline", catalogEntity.LinkedPipeline).
		Set("labels", labelBytes).
		Set("entity_metadata", metadataBytes).
		Set("updated_at", sq.Expr("now()")).
		Where(sq.Eq{"id": entityID}).
		Limit(uint64(1))

	_, err = query.RunWith(c.databaseConnection).Exec()

	return
}

func (c *client) DeleteCatalogEntity(ctx context.Context, id string) (err error) {

	psql := sq.StatementBuilder.PlaceholderFormat(sq.Dollar)

	query := psql.
		Delete("catalog_entities a").
		Where(sq.Eq{"a.id": id}).
		Limit(uint64(1))

	_, err = query.RunWith(c.databaseConnection).Exec()
	if err != nil {
		return
	}

	return nil
}

func (c *client) GetCatalogEntityByID(ctx context.Context, id string) (catalogEntity *contracts.CatalogEntity, err error) {

	query := c.selectCatalogEntityQuery().
		Where(sq.Eq{"a.id": id}).
		Limit(uint64(1))

	// execute query
	row := query.RunWith(c.databaseConnection).QueryRow()
	catalogEntity, err = c.scanCatalogEntity(row)
	if err != nil {
		return nil, err
	}

	return catalogEntity, nil
}

func (c *client) GetCatalogEntities(ctx context.Context, pageNumber, pageSize int, filters map[string][]string, sortings []helpers.OrderField) (catalogEntities []*contracts.CatalogEntity, err error) {

	query := c.selectCatalogEntityQuery().
		Limit(uint64(pageSize)).
		Offset(uint64((pageNumber - 1) * pageSize))

	query, err = whereClauseGeneratorForCatalogEntityFilters(query, "a", filters)
	query, err = orderByClauseGeneratorForSortings(query, "a", "a.parent_key, a.parent_value, a.entity_key, a.entity_value", sortings)

	// execute query
	rows, err := query.RunWith(c.databaseConnection).Query()
	if err != nil {
		return
	}

	return c.scanCatalogEntities(rows)
}

func (c *client) GetCatalogEntitiesCount(ctx context.Context, filters map[string][]string) (count int, err error) {
	psql := sq.StatementBuilder.PlaceholderFormat(sq.Dollar)

	query := psql.
		Select("COUNT(a.id)").
		From("catalog_entities a")

	query, err = whereClauseGeneratorForCatalogEntityFilters(query, "a", filters)

	// execute query
	row := query.RunWith(c.databaseConnection).QueryRow()
	if err = row.Scan(&count); err != nil {
		return
	}

	return
}

func (c *client) GetCatalogEntityParentKeys(ctx context.Context, pageNumber, pageSize int, filters map[string][]string, sortings []helpers.OrderField) (keys []map[string]interface{}, err error) {
	return c.getCatalogEntityColumn(ctx, "parent_key", "id", pageNumber, pageSize, filters, sortings)
}

func (c *client) GetCatalogEntityParentKeysCount(ctx context.Context, filters map[string][]string) (count int, err error) {
	return c.getCatalogEntityColumnCount(ctx, "parent_key", filters)
}

func (c *client) GetCatalogEntityParentValues(ctx context.Context, pageNumber, pageSize int, filters map[string][]string, sortings []helpers.OrderField) (values []map[string]interface{}, err error) {
	return c.getCatalogEntityColumn(ctx, "parent_value", "id", pageNumber, pageSize, filters, sortings)
}

func (c *client) GetCatalogEntityParentValuesCount(ctx context.Context, filters map[string][]string) (count int, err error) {
	return c.getCatalogEntityColumnCount(ctx, "parent_value", filters)
}

func (c *client) GetCatalogEntityKeys(ctx context.Context, pageNumber, pageSize int, filters map[string][]string, sortings []helpers.OrderField) (keys []map[string]interface{}, err error) {
	return c.getCatalogEntityColumn(ctx, "entity_key", "id", pageNumber, pageSize, filters, sortings)
}

func (c *client) GetCatalogEntityKeysCount(ctx context.Context, filters map[string][]string) (count int, err error) {
	return c.getCatalogEntityColumnCount(ctx, "entity_key", filters)
}

func (c *client) GetCatalogEntityValues(ctx context.Context, pageNumber, pageSize int, filters map[string][]string, sortings []helpers.OrderField) (values []map[string]interface{}, err error) {
	return c.getCatalogEntityColumn(ctx, "entity_value", "id", pageNumber, pageSize, filters, sortings)
}

func (c *client) GetCatalogEntityValuesCount(ctx context.Context, filters map[string][]string) (count int, err error) {
	return c.getCatalogEntityColumnCount(ctx, "entity_value", filters)
}

func (c *client) getCatalogEntityColumn(ctx context.Context, groupColumn, countColumn string, pageNumber, pageSize int, filters map[string][]string, sortings []helpers.OrderField) (keys []map[string]interface{}, err error) {

	psql := sq.StatementBuilder.PlaceholderFormat(sq.Dollar)

	query :=
		psql.
			Select(fmt.Sprintf("a.%v AS key, COUNT(a.%v) AS count", groupColumn, countColumn)).
			From("catalog_entities a").
			GroupBy(fmt.Sprintf("a.%v", groupColumn)).
			OrderBy("count DESC, key").
			Limit(uint64(pageSize)).
			Offset(uint64((pageNumber - 1) * pageSize))

	query, err = whereClauseGeneratorForCatalogEntityFilters(query, "a", filters)

	rows, err := query.RunWith(c.databaseConnection).Query()
	if err != nil {
		return
	}
	defer rows.Close()

	return c.scanItems(ctx, rows)
}

func (c *client) getCatalogEntityColumnCount(ctx context.Context, groupColumn string, filters map[string][]string) (count int, err error) {

	psql := sq.StatementBuilder.PlaceholderFormat(sq.Dollar)

	query :=
		psql.
			Select(fmt.Sprintf("COUNT(a.%v)", groupColumn)).
			From("catalog_entities a").
			GroupBy(fmt.Sprintf("a.%v", groupColumn))

	query, err = whereClauseGeneratorForCatalogEntityFilters(query, "a", filters)

	// execute query
	row := query.RunWith(c.databaseConnection).QueryRow()
	if err = row.Scan(&count); err != nil {
		return
	}

	return
}

func (c *client) GetCatalogEntityLabels(ctx context.Context, pageNumber, pageSize int, filters map[string][]string) (labels []map[string]interface{}, err error) {
	psql := sq.StatementBuilder.PlaceholderFormat(sq.Dollar)

	arrayElementsQuery :=
		psql.
			Select("a.id, jsonb_array_elements(a.labels) AS l").
			From("catalog_entities a").
			Where("jsonb_typeof(labels) = 'array'")

	arrayElementsQuery, err = whereClauseGeneratorForLabelsFilter(arrayElementsQuery, "a", filters)
	if err != nil {
		return
	}

	selectCountQuery :=
		psql.
			Select("l->>'key' AS key, l->>'value' AS value, id").
			FromSelect(arrayElementsQuery, "b")

	groupByQuery :=
		psql.
			Select("key, value, count(DISTINCT id) AS count").
			FromSelect(selectCountQuery, "c").
			GroupBy("key, value")

	query :=
		psql.
			Select("key, value, count").
			FromSelect(groupByQuery, "d").
			Where(sq.Gt{"count": 1}).
			OrderBy("count DESC, key, value").
			Limit(uint64(pageSize)).
			Offset(uint64((pageNumber - 1) * pageSize))

	rows, err := query.RunWith(c.databaseConnection).Query()
	if err != nil {
		return
	}
	defer rows.Close()

	return c.scanItems(ctx, rows)
}

func (c *client) GetCatalogEntityLabelsCount(ctx context.Context, filters map[string][]string) (count int, err error) {
	psql := sq.StatementBuilder.PlaceholderFormat(sq.Dollar)

	arrayElementsQuery :=
		psql.
			Select("a.id, jsonb_array_elements(a.labels) AS l").
			From("catalog_entities a").
			Where("jsonb_typeof(labels) = 'array'")

	arrayElementsQuery, err = whereClauseGeneratorForLabelsFilter(arrayElementsQuery, "a", filters)
	if err != nil {
		return
	}

	selectCountQuery :=
		psql.
			Select("l->>'key' AS key, l->>'value' AS value, id").
			FromSelect(arrayElementsQuery, "b")

	groupByQuery :=
		psql.
			Select("key, value, count(DISTINCT id) AS count").
			FromSelect(selectCountQuery, "c").
			GroupBy("key, value")

	query :=
		psql.
			Select("COUNT(key)").
			FromSelect(groupByQuery, "d").
			Where(sq.Gt{"count": 1})

	// execute query
	row := query.RunWith(c.databaseConnection).QueryRow()
	if err = row.Scan(&count); err != nil {
		return
	}

	return
}

func (c *client) scanUsers(rows *sql.Rows) (users []*contracts.User, err error) {
	users = make([]*contracts.User, 0)

	defer rows.Close()
	for rows.Next() {

		user := &contracts.User{}
		var id string
		var userData []uint8
		var insertedAt *time.Time

		if err = rows.Scan(
			&id,
			&userData,
			&insertedAt); err != nil {
			return
		}

		if len(userData) > 0 {
			if err = json.Unmarshal(userData, &user); err != nil {
				return nil, err
			}
		}

		user.ID = id
		user.FirstVisit = insertedAt

		users = append(users, user)
	}

	return
}

func (c *client) scanUser(row sq.RowScanner) (user *contracts.User, err error) {

	user = &contracts.User{}
	var id string
	var userData []uint8
	var insertedAt *time.Time

	if err = row.Scan(
		&id,
		&userData,
		&insertedAt); err != nil {
		if err == sql.ErrNoRows {
			return nil, ErrUserNotFound
		}

		return
	}

	if len(userData) > 0 {
		if err = json.Unmarshal(userData, &user); err != nil {
			return nil, err
		}
	}

	user.ID = id
	user.FirstVisit = insertedAt

	return
}

func (c *client) scanGroups(rows *sql.Rows) (groups []*contracts.Group, err error) {
	groups = make([]*contracts.Group, 0)

	defer rows.Close()
	for rows.Next() {

		group := &contracts.Group{}
		var id string
		var groupData []uint8
		var insertedAt *time.Time

		if err = rows.Scan(
			&id,
			&groupData,
			&insertedAt); err != nil {
			return
		}

		if len(groupData) > 0 {
			if err = json.Unmarshal(groupData, &group); err != nil {
				return nil, err
			}
		}

		group.ID = id

		groups = append(groups, group)
	}

	return
}

func (c *client) scanGroup(row sq.RowScanner) (group *contracts.Group, err error) {

	group = &contracts.Group{}
	var id string
	var groupData []uint8
	var insertedAt *time.Time

	if err = row.Scan(
		&id,
		&groupData,
		&insertedAt); err != nil {
		if err == sql.ErrNoRows {
			return nil, ErrGroupNotFound
		}

		return
	}

	if len(groupData) > 0 {
		if err = json.Unmarshal(groupData, &group); err != nil {
			return nil, err
		}
	}

	group.ID = id

	return
}

func (c *client) scanOrganizations(rows *sql.Rows) (organizations []*contracts.Organization, err error) {
	organizations = make([]*contracts.Organization, 0)

	defer rows.Close()
	for rows.Next() {

		organization := &contracts.Organization{}
		var id string
		var organizationData []uint8
		var insertedAt *time.Time

		if err = rows.Scan(
			&id,
			&organizationData,
			&insertedAt); err != nil {
			return
		}

		if len(organizationData) > 0 {
			if err = json.Unmarshal(organizationData, &organization); err != nil {
				return nil, err
			}
		}

		organization.ID = id

		organizations = append(organizations, organization)
	}

	return
}

func (c *client) scanOrganization(row sq.RowScanner) (organization *contracts.Organization, err error) {

	organization = &contracts.Organization{}
	var id string
	var organizationData []uint8
	var insertedAt *time.Time

	if err = row.Scan(
		&id,
		&organizationData,
		&insertedAt); err != nil {
		if err == sql.ErrNoRows {
			return nil, ErrOrganizationNotFound
		}

		return
	}

	if len(organizationData) > 0 {
		if err = json.Unmarshal(organizationData, &organization); err != nil {
			return nil, err
		}
	}

	organization.ID = id

	return
}

func (c *client) scanClients(rows *sql.Rows) (clients []*contracts.Client, err error) {
	clients = make([]*contracts.Client, 0)

	defer rows.Close()
	for rows.Next() {

		client := &contracts.Client{}
		var id string
		var clientData []uint8
		var insertedAt *time.Time

		if err = rows.Scan(
			&id,
			&clientData,
			&insertedAt); err != nil {
			return
		}

		if len(clientData) > 0 {
			if err = json.Unmarshal(clientData, &client); err != nil {
				return nil, err
			}
		}

		client.ID = id

		clients = append(clients, client)
	}

	return
}

func (c *client) scanClient(row sq.RowScanner) (client *contracts.Client, err error) {

	client = &contracts.Client{}
	var id string
	var clientData []uint8
	var insertedAt *time.Time

	if err = row.Scan(
		&id,
		&clientData,
		&insertedAt); err != nil {
		if err == sql.ErrNoRows {
			return nil, ErrClientNotFound
		}

		return
	}

	if len(clientData) > 0 {
		if err = json.Unmarshal(clientData, &client); err != nil {
			return nil, err
		}
	}

	client.ID = id

	return
}

func (c *client) scanCatalogEntities(rows *sql.Rows) (catalogEntities []*contracts.CatalogEntity, err error) {
	catalogEntities = make([]*contracts.CatalogEntity, 0)

	defer rows.Close()
	for rows.Next() {

		catalogEntity := &contracts.CatalogEntity{}
		var labelData, metadataData []uint8

		if err = rows.Scan(
			&catalogEntity.ID,
			&catalogEntity.ParentKey,
			&catalogEntity.ParentValue,
			&catalogEntity.Key,
			&catalogEntity.Value,
			&catalogEntity.LinkedPipeline,
			&labelData,
			&metadataData,
			&catalogEntity.InsertedAt,
			&catalogEntity.UpdatedAt); err != nil {
			return
		}

		if len(labelData) > 0 {
			if err = json.Unmarshal(labelData, &catalogEntity.Labels); err != nil {
				return nil, err
			}
		}

		if len(metadataData) > 0 {
			if err = json.Unmarshal(metadataData, &catalogEntity.Metadata); err != nil {
				return nil, err
			}
		}

		catalogEntities = append(catalogEntities, catalogEntity)
	}

	return
}

func (c *client) scanCatalogEntity(row sq.RowScanner) (catalogEntity *contracts.CatalogEntity, err error) {

	catalogEntity = &contracts.CatalogEntity{}

	var labelData, metadataData []uint8

	if err = row.Scan(
		&catalogEntity.ID,
		&catalogEntity.ParentKey,
		&catalogEntity.ParentValue,
		&catalogEntity.Key,
		&catalogEntity.Value,
		&catalogEntity.LinkedPipeline,
		&labelData,
		&metadataData,
		&catalogEntity.InsertedAt,
		&catalogEntity.UpdatedAt); err != nil {
		if err == sql.ErrNoRows {
			return nil, ErrCatalogEntityNotFound
		}

		return
	}

	if len(labelData) > 0 {
		if err = json.Unmarshal(labelData, &catalogEntity.Labels); err != nil {
			return nil, err
		}
	}

	if len(metadataData) > 0 {
		if err = json.Unmarshal(metadataData, &catalogEntity.Metadata); err != nil {
			return nil, err
		}
	}

	return
}

func (c *client) selectBuildsQuery() sq.SelectBuilder {
	psql := sq.StatementBuilder.PlaceholderFormat(sq.Dollar)

	return psql.
		Select("a.id, a.repo_source, a.repo_owner, a.repo_name, a.repo_branch, a.repo_revision, a.build_version, a.build_status, a.labels, a.release_targets, a.manifest, a.commits, a.triggers, a.inserted_at, a.started_at, a.updated_at, age(COALESCE(a.started_at, a.inserted_at), a.inserted_at)::INT, age(a.updated_at, COALESCE(a.started_at,a.inserted_at))::INT, a.triggered_by_event").
		From("builds a")
}

func (c *client) selectPipelinesQuery() sq.SelectBuilder {
	psql := sq.StatementBuilder.PlaceholderFormat(sq.Dollar)

	return psql.
		Select("a.pipeline_id, a.repo_source, a.repo_owner, a.repo_name, a.repo_branch, a.repo_revision, a.build_version, a.build_status, a.labels, a.release_targets, a.manifest, a.commits, a.triggers, a.archived, a.inserted_at, a.started_at, a.updated_at, age(COALESCE(a.started_at, a.inserted_at), a.inserted_at)::INT, age(a.updated_at, COALESCE(a.started_at,a.inserted_at))::INT, a.last_updated_at, a.triggered_by_event, a.extra_info").
		From("computed_pipelines a")
}

func (c *client) selectReleasesQuery() sq.SelectBuilder {
	psql := sq.StatementBuilder.PlaceholderFormat(sq.Dollar)

	return psql.
		Select("a.id, a.repo_source, a.repo_owner, a.repo_name, a.release, a.release_action, a.release_version, a.release_status, a.inserted_at, a.started_at, a.updated_at, age(COALESCE(a.started_at, a.inserted_at), a.inserted_at)::INT, age(a.updated_at, COALESCE(a.started_at,a.inserted_at))::INT, a.triggered_by_event").
		From("releases a")
}

func (c *client) selectComputedReleasesQuery() sq.SelectBuilder {
	psql := sq.StatementBuilder.PlaceholderFormat(sq.Dollar)

	return psql.
		Select("a.release_id, a.repo_source, a.repo_owner, a.repo_name, a.release, a.release_action, a.release_version, a.release_status, a.inserted_at, a.started_at, a.updated_at, age(COALESCE(a.started_at, a.inserted_at), a.inserted_at)::INT, age(a.updated_at, COALESCE(a.started_at,a.inserted_at))::INT, a.triggered_by_event").
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
		Select("a.id, a.repo_source, a.repo_owner, a.repo_name, a.trigger_event, a.trigger_filter, a.trigger_run, a.inserted_at, a.started_at, a.updated_at").
		From("pipeline_triggers a")
}

func (c *client) selectCatalogEntityQuery() sq.SelectBuilder {
	psql := sq.StatementBuilder.PlaceholderFormat(sq.Dollar)

	return psql.
		Select("a.id, a.parent_key, a.parent_value, a.entity_key, a.entity_value, a.linked_pipeline, a.labels, a.entity_metadata, a.inserted_at, a.updated_at").
		From("catalog_entities a")
}

func (c *client) enrichPipeline(ctx context.Context, pipeline *contracts.Pipeline) {
	c.getLatestReleasesForPipeline(ctx, pipeline)
}

func (c *client) enrichBuild(ctx context.Context, build *contracts.Build) {
	c.getLatestReleasesForBuild(ctx, build)
}

func (c *client) setPipelinePropertiesFromJSONB(pipeline *contracts.Pipeline, labelsData, releaseTargetsData, commitsData, triggersData, triggeredByEventsData, extraInfoData []uint8, optimized bool) (err error) {

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
	if len(extraInfoData) > 0 {
		if err = json.Unmarshal(extraInfoData, &pipeline.ExtraInfo); err != nil {
			return
		}
	}

	if !optimized {
		// unmarshal then marshal manifest to include defaults
		manifest, err := manifest.ReadManifest(c.config.ManifestPreferences, pipeline.Manifest, false)
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
		manifest, err := manifest.ReadManifest(c.config.ManifestPreferences, build.Manifest, false)
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
	mft, err := manifest.ReadManifest(c.config.ManifestPreferences, build.Manifest, false)
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
		StartedAt:            build.StartedAt,
		UpdatedAt:            build.UpdatedAt,
		Duration:             build.Duration,
		Events:               build.Events,
	}
}
