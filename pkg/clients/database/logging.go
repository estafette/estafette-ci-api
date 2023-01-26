package database

import (
	"context"
	"time"

	"github.com/estafette/estafette-ci-api/pkg/api"
	contracts "github.com/estafette/estafette-ci-contracts"
	manifest "github.com/estafette/estafette-ci-manifest"
)

// NewLoggingClient returns a new instance of a logging Client.
func NewLoggingClient(c Client) Client {
	return &loggingClient{c, "cockroachdb"}
}

type loggingClient struct {
	Client Client
	prefix string
}

func (c *loggingClient) Connect(ctx context.Context) (err error) {
	defer func() { api.HandleLogError(c.prefix, "Client", "Connect", err) }()

	return c.Client.Connect(ctx)
}

func (c *loggingClient) ConnectWithDriverAndSource(ctx context.Context, driverName, dataSourceName string) (err error) {
	defer func() { api.HandleLogError(c.prefix, "Client", "ConnectWithDriverAndSource", err) }()

	return c.Client.ConnectWithDriverAndSource(ctx, driverName, dataSourceName)
}

func (c *loggingClient) AwaitDatabaseReadiness(ctx context.Context) (err error) {
	defer func() { api.HandleLogError(c.prefix, "Client", "AwaitDatabaseReadiness", err) }()

	return c.Client.AwaitDatabaseReadiness(ctx)
}

func (c *loggingClient) GetAutoIncrement(ctx context.Context, shortRepoSource, repoOwner, repoName string) (autoincrement int, err error) {
	defer func() { api.HandleLogError(c.prefix, "Client", "GetAutoIncrement", err) }()

	return c.Client.GetAutoIncrement(ctx, shortRepoSource, repoOwner, repoName)
}

func (c *loggingClient) InsertBuild(ctx context.Context, build contracts.Build, jobResources JobResources) (b *contracts.Build, err error) {
	defer func() { api.HandleLogError(c.prefix, "Client", "InsertBuild", err) }()

	return c.Client.InsertBuild(ctx, build, jobResources)
}

func (c *loggingClient) UpdateBuildStatus(ctx context.Context, repoSource, repoOwner, repoName string, buildID string, buildStatus contracts.Status) (err error) {
	defer func() { api.HandleLogError(c.prefix, "Client", "UpdateBuildStatus", err) }()

	return c.Client.UpdateBuildStatus(ctx, repoSource, repoOwner, repoName, buildID, buildStatus)
}

func (c *loggingClient) UpdateBuildResourceUtilization(ctx context.Context, repoSource, repoOwner, repoName string, buildID string, jobResources JobResources) (err error) {
	defer func() { api.HandleLogError(c.prefix, "Client", "UpdateBuildResourceUtilization", err) }()

	return c.Client.UpdateBuildResourceUtilization(ctx, repoSource, repoOwner, repoName, buildID, jobResources)
}

func (c *loggingClient) InsertRelease(ctx context.Context, release contracts.Release, jobResources JobResources) (r *contracts.Release, err error) {
	defer func() { api.HandleLogError(c.prefix, "Client", "InsertRelease", err) }()

	return c.Client.InsertRelease(ctx, release, jobResources)
}

func (c *loggingClient) UpdateReleaseStatus(ctx context.Context, repoSource, repoOwner, repoName string, releaseID string, releaseStatus contracts.Status) (err error) {
	defer func() { api.HandleLogError(c.prefix, "Client", "UpdateReleaseStatus", err) }()

	return c.Client.UpdateReleaseStatus(ctx, repoSource, repoOwner, repoName, releaseID, releaseStatus)
}

func (c *loggingClient) UpdateReleaseResourceUtilization(ctx context.Context, repoSource, repoOwner, repoName string, releaseID string, jobResources JobResources) (err error) {
	defer func() { api.HandleLogError(c.prefix, "Client", "UpdateReleaseResourceUtilization", err) }()

	return c.Client.UpdateReleaseResourceUtilization(ctx, repoSource, repoOwner, repoName, releaseID, jobResources)
}

func (c *loggingClient) InsertBot(ctx context.Context, bot contracts.Bot, jobResources JobResources) (r *contracts.Bot, err error) {
	defer func() { api.HandleLogError(c.prefix, "Client", "InsertBot", err) }()

	return c.Client.InsertBot(ctx, bot, jobResources)
}

func (c *loggingClient) UpdateBotStatus(ctx context.Context, repoSource, repoOwner, repoName string, botID string, botStatus contracts.Status) (err error) {
	defer func() { api.HandleLogError(c.prefix, "Client", "UpdateBotStatus", err) }()

	return c.Client.UpdateBotStatus(ctx, repoSource, repoOwner, repoName, botID, botStatus)
}

func (c *loggingClient) UpdateBotResourceUtilization(ctx context.Context, repoSource, repoOwner, repoName string, botID string, jobResources JobResources) (err error) {
	defer func() { api.HandleLogError(c.prefix, "Client", "UpdateBotResourceUtilization", err) }()

	return c.Client.UpdateBotResourceUtilization(ctx, repoSource, repoOwner, repoName, botID, jobResources)
}

func (c *loggingClient) InsertBuildLog(ctx context.Context, buildLog contracts.BuildLog) (buildlog contracts.BuildLog, err error) {
	defer func() { api.HandleLogError(c.prefix, "Client", "InsertBuildLog", err) }()

	return c.Client.InsertBuildLog(ctx, buildLog)
}

func (c *loggingClient) InsertReleaseLog(ctx context.Context, releaseLog contracts.ReleaseLog) (releaselog contracts.ReleaseLog, err error) {
	defer func() { api.HandleLogError(c.prefix, "Client", "InsertReleaseLog", err) }()

	return c.Client.InsertReleaseLog(ctx, releaseLog)
}

func (c *loggingClient) InsertBotLog(ctx context.Context, botLog contracts.BotLog) (log contracts.BotLog, err error) {
	defer func() { api.HandleLogError(c.prefix, "Client", "InsertBotLog", err) }()

	return c.Client.InsertBotLog(ctx, botLog)
}

func (c *loggingClient) UpdateComputedTables(ctx context.Context, repoSource, repoOwner, repoName string) (err error) {
	defer func() { api.HandleLogError(c.prefix, "Client", "UpdateComputedTables", err) }()

	return c.Client.UpdateComputedTables(ctx, repoSource, repoOwner, repoName)
}

func (c *loggingClient) UpsertComputedPipeline(ctx context.Context, repoSource, repoOwner, repoName string) (err error) {
	defer func() { api.HandleLogError(c.prefix, "Client", "UpsertComputedPipeline", err) }()

	return c.Client.UpsertComputedPipeline(ctx, repoSource, repoOwner, repoName)
}

func (c *loggingClient) UpdateComputedPipelinePermissions(ctx context.Context, pipeline contracts.Pipeline) (err error) {
	defer func() { api.HandleLogError(c.prefix, "Client", "UpdateComputedPipelinePermissions", err) }()

	return c.Client.UpdateComputedPipelinePermissions(ctx, pipeline)
}

func (c *loggingClient) UpdateComputedPipelineFirstInsertedAt(ctx context.Context, repoSource, repoOwner, repoName string) (err error) {
	defer func() { api.HandleLogError(c.prefix, "Client", "UpdateComputedPipelineFirstInsertedAt", err) }()

	return c.Client.UpdateComputedPipelineFirstInsertedAt(ctx, repoSource, repoOwner, repoName)
}

func (c *loggingClient) UpsertComputedRelease(ctx context.Context, repoSource, repoOwner, repoName, releaseName, releaseAction string) (err error) {
	defer func() { api.HandleLogError(c.prefix, "Client", "UpsertComputedRelease", err) }()

	return c.Client.UpsertComputedRelease(ctx, repoSource, repoOwner, repoName, releaseName, releaseAction)
}

func (c *loggingClient) UpdateComputedReleaseFirstInsertedAt(ctx context.Context, repoSource, repoOwner, repoName, releaseName, releaseAction string) (err error) {
	defer func() { api.HandleLogError(c.prefix, "Client", "UpdateComputedReleaseFirstInsertedAt", err) }()

	return c.Client.UpdateComputedReleaseFirstInsertedAt(ctx, repoSource, repoOwner, repoName, releaseName, releaseAction)
}

func (c *loggingClient) ArchiveComputedPipeline(ctx context.Context, repoSource, repoOwner, repoName string) (err error) {
	defer func() { api.HandleLogError(c.prefix, "Client", "ArchiveComputedPipeline", err) }()

	return c.Client.ArchiveComputedPipeline(ctx, repoSource, repoOwner, repoName)
}

func (c *loggingClient) UnarchiveComputedPipeline(ctx context.Context, repoSource, repoOwner, repoName string) (err error) {
	defer func() { api.HandleLogError(c.prefix, "Client", "UnarchiveComputedPipeline", err) }()

	return c.Client.UnarchiveComputedPipeline(ctx, repoSource, repoOwner, repoName)
}

func (c *loggingClient) GetPipelines(ctx context.Context, pageNumber, pageSize int, filters map[api.FilterType][]string, sortings []api.OrderField, optimized bool) (pipelines []*contracts.Pipeline, err error) {
	defer func() { api.HandleLogError(c.prefix, "Client", "GetPipelines", err) }()

	return c.Client.GetPipelines(ctx, pageNumber, pageSize, filters, sortings, optimized)
}

func (c *loggingClient) GetPipelinesByRepoName(ctx context.Context, repoName string, optimized bool) (pipelines []*contracts.Pipeline, err error) {
	defer func() { api.HandleLogError(c.prefix, "Client", "GetPipelinesByRepoName", err) }()

	return c.Client.GetPipelinesByRepoName(ctx, repoName, optimized)
}

func (c *loggingClient) GetPipelinesCount(ctx context.Context, filters map[api.FilterType][]string) (count int, err error) {
	defer func() { api.HandleLogError(c.prefix, "Client", "GetPipelinesCount", err) }()

	return c.Client.GetPipelinesCount(ctx, filters)
}

func (c *loggingClient) GetPipeline(ctx context.Context, repoSource, repoOwner, repoName string, filters map[api.FilterType][]string, optimized bool) (pipeline *contracts.Pipeline, err error) {
	defer func() { api.HandleLogError(c.prefix, "Client", "GetPipeline", err) }()

	return c.Client.GetPipeline(ctx, repoSource, repoOwner, repoName, filters, optimized)
}

func (c *loggingClient) GetPipelineRecentBuilds(ctx context.Context, repoSource, repoOwner, repoName string, optimized bool) (builds []*contracts.Build, err error) {
	defer func() { api.HandleLogError(c.prefix, "Client", "GetPipelineRecentBuilds", err) }()

	return c.Client.GetPipelineRecentBuilds(ctx, repoSource, repoOwner, repoName, optimized)
}

func (c *loggingClient) GetPipelineBuilds(ctx context.Context, repoSource, repoOwner, repoName string, pageNumber, pageSize int, filters map[api.FilterType][]string, sortings []api.OrderField, optimized bool) (builds []*contracts.Build, err error) {
	defer func() { api.HandleLogError(c.prefix, "Client", "GetPipelineBuilds", err) }()

	return c.Client.GetPipelineBuilds(ctx, repoSource, repoOwner, repoName, pageNumber, pageSize, filters, sortings, optimized)
}

func (c *loggingClient) GetPipelineBuildsCount(ctx context.Context, repoSource, repoOwner, repoName string, filters map[api.FilterType][]string) (count int, err error) {
	defer func() { api.HandleLogError(c.prefix, "Client", "GetPipelineBuildsCount", err) }()

	return c.Client.GetPipelineBuildsCount(ctx, repoSource, repoOwner, repoName, filters)
}

func (c *loggingClient) GetPipelineBuild(ctx context.Context, repoSource, repoOwner, repoName, repoRevision string, optimized bool) (build *contracts.Build, err error) {
	defer func() { api.HandleLogError(c.prefix, "Client", "GetPipelineBuild", err) }()

	return c.Client.GetPipelineBuild(ctx, repoSource, repoOwner, repoName, repoRevision, optimized)
}

func (c *loggingClient) GetPipelineBuildByID(ctx context.Context, repoSource, repoOwner, repoName string, buildID string, optimized bool) (build *contracts.Build, err error) {
	defer func() { api.HandleLogError(c.prefix, "Client", "GetPipelineBuildByID", err) }()

	return c.Client.GetPipelineBuildByID(ctx, repoSource, repoOwner, repoName, buildID, optimized)
}

func (c *loggingClient) GetLastPipelineBuild(ctx context.Context, repoSource, repoOwner, repoName string, optimized bool) (build *contracts.Build, err error) {
	defer func() { api.HandleLogError(c.prefix, "Client", "GetLastPipelineBuild", err) }()

	return c.Client.GetLastPipelineBuild(ctx, repoSource, repoOwner, repoName, optimized)
}

func (c *loggingClient) GetFirstPipelineBuild(ctx context.Context, repoSource, repoOwner, repoName string, optimized bool) (build *contracts.Build, err error) {
	defer func() { api.HandleLogError(c.prefix, "Client", "GetFirstPipelineBuild", err) }()

	return c.Client.GetFirstPipelineBuild(ctx, repoSource, repoOwner, repoName, optimized)
}

func (c *loggingClient) GetLastPipelineBuildForBranch(ctx context.Context, repoSource, repoOwner, repoName, branch string) (build *contracts.Build, err error) {
	defer func() { api.HandleLogError(c.prefix, "Client", "GetLastPipelineBuildForBranch", err) }()

	return c.Client.GetLastPipelineBuildForBranch(ctx, repoSource, repoOwner, repoName, branch)
}

func (c *loggingClient) GetLastPipelineReleases(ctx context.Context, repoSource, repoOwner, repoName, releaseName, releaseAction string, pageSize int) (releases []*contracts.Release, err error) {
	defer func() { api.HandleLogError(c.prefix, "Client", "GetLastPipelineReleases", err) }()

	return c.Client.GetLastPipelineReleases(ctx, repoSource, repoOwner, repoName, releaseName, releaseAction, pageSize)
}

func (c *loggingClient) GetFirstPipelineRelease(ctx context.Context, repoSource, repoOwner, repoName, releaseName, releaseAction string) (release *contracts.Release, err error) {
	defer func() { api.HandleLogError(c.prefix, "Client", "GetFirstPipelineRelease", err) }()

	return c.Client.GetFirstPipelineRelease(ctx, repoSource, repoOwner, repoName, releaseName, releaseAction)
}

func (c *loggingClient) GetPipelineBuildsByVersion(ctx context.Context, repoSource, repoOwner, repoName, buildVersion string, statuses []contracts.Status, limit uint64, optimized bool) (builds []*contracts.Build, err error) {
	defer func() { api.HandleLogError(c.prefix, "Client", "GetPipelineBuildsByVersion", err) }()

	return c.Client.GetPipelineBuildsByVersion(ctx, repoSource, repoOwner, repoName, buildVersion, statuses, limit, optimized)
}

func (c *loggingClient) GetPipelineBuildLogs(ctx context.Context, repoSource, repoOwner, repoName, repoBranch, repoRevision, buildID string) (buildlog *contracts.BuildLog, err error) {
	defer func() { api.HandleLogError(c.prefix, "Client", "GetPipelineBuildLogs", err) }()

	return c.Client.GetPipelineBuildLogs(ctx, repoSource, repoOwner, repoName, repoBranch, repoRevision, buildID)
}

func (c *loggingClient) GetPipelineBuildLogsByID(ctx context.Context, repoSource, repoOwner, repoName, buildID, id string) (buildlog *contracts.BuildLog, err error) {
	defer func() { api.HandleLogError(c.prefix, "Client", "GetPipelineBuildLogsByID", err) }()

	return c.Client.GetPipelineBuildLogsByID(ctx, repoSource, repoOwner, repoName, buildID, id)
}

func (c *loggingClient) GetPipelineBuildLogsPerPage(ctx context.Context, repoSource, repoOwner, repoName, repoBranch, repoRevision, buildID string, pageNumber int, pageSize int) (buildLogs []*contracts.BuildLog, err error) {
	defer func() { api.HandleLogError(c.prefix, "Client", "GetPipelineBuildLogsPerPage", err) }()

	return c.Client.GetPipelineBuildLogsPerPage(ctx, repoSource, repoOwner, repoName, repoBranch, repoRevision, buildID, pageNumber, pageSize)
}

func (c *loggingClient) GetPipelineBuildLogsCount(ctx context.Context, repoSource, repoOwner, repoName, repoBranch, repoRevision, buildID string) (count int, err error) {
	defer func() { api.HandleLogError(c.prefix, "Client", "GetPipelineBuildLogsCount", err) }()

	return c.Client.GetPipelineBuildLogsCount(ctx, repoSource, repoOwner, repoName, repoBranch, repoRevision, buildID)
}

func (c *loggingClient) GetPipelineBuildMaxResourceUtilization(ctx context.Context, repoSource, repoOwner, repoName string, lastNRecords int) (jobresources JobResources, count int, err error) {
	defer func() { api.HandleLogError(c.prefix, "Client", "GetPipelineBuildMaxResourceUtilization", err) }()

	return c.Client.GetPipelineBuildMaxResourceUtilization(ctx, repoSource, repoOwner, repoName, lastNRecords)
}

func (c *loggingClient) GetPipelineReleases(ctx context.Context, repoSource, repoOwner, repoName string, pageNumber, pageSize int, filters map[api.FilterType][]string, sortings []api.OrderField) (releases []*contracts.Release, err error) {
	defer func() { api.HandleLogError(c.prefix, "Client", "GetPipelineReleases", err) }()

	return c.Client.GetPipelineReleases(ctx, repoSource, repoOwner, repoName, pageNumber, pageSize, filters, sortings)
}

func (c *loggingClient) GetPipelineReleasesCount(ctx context.Context, repoSource, repoOwner, repoName string, filters map[api.FilterType][]string) (count int, err error) {
	defer func() { api.HandleLogError(c.prefix, "Client", "GetPipelineReleasesCount", err) }()

	return c.Client.GetPipelineReleasesCount(ctx, repoSource, repoOwner, repoName, filters)
}

func (c *loggingClient) GetPipelineRelease(ctx context.Context, repoSource, repoOwner, repoName string, releaseID string) (release *contracts.Release, err error) {
	defer func() { api.HandleLogError(c.prefix, "Client", "GetPipelineRelease", err) }()

	return c.Client.GetPipelineRelease(ctx, repoSource, repoOwner, repoName, releaseID)
}

func (c *loggingClient) GetPipelineLastReleasesByName(ctx context.Context, repoSource, repoOwner, repoName, releaseName string, actions []string) (releases []contracts.Release, err error) {
	defer func() { api.HandleLogError(c.prefix, "Client", "GetPipelineLastReleasesByName", err) }()

	return c.Client.GetPipelineLastReleasesByName(ctx, repoSource, repoOwner, repoName, releaseName, actions)
}

func (c *loggingClient) GetPipelineReleaseLogs(ctx context.Context, repoSource, repoOwner, repoName string, releaseID string) (releaselog *contracts.ReleaseLog, err error) {
	defer func() { api.HandleLogError(c.prefix, "Client", "GetPipelineReleaseLogs", err) }()

	return c.Client.GetPipelineReleaseLogs(ctx, repoSource, repoOwner, repoName, releaseID)
}

func (c *loggingClient) GetPipelineReleaseLogsByID(ctx context.Context, repoSource, repoOwner, repoName string, releaseID string, id string) (releaselog *contracts.ReleaseLog, err error) {
	defer func() { api.HandleLogError(c.prefix, "Client", "GetPipelineReleaseLogsByID", err) }()

	return c.Client.GetPipelineReleaseLogsByID(ctx, repoSource, repoOwner, repoName, releaseID, id)
}

func (c *loggingClient) GetPipelineReleaseLogsPerPage(ctx context.Context, repoSource, repoOwner, repoName string, releaseID string, pageNumber int, pageSize int) (releaselogs []*contracts.ReleaseLog, err error) {
	defer func() { api.HandleLogError(c.prefix, "Client", "GetPipelineReleaseLogsPerPage", err) }()

	return c.Client.GetPipelineReleaseLogsPerPage(ctx, repoSource, repoOwner, repoName, releaseID, pageNumber, pageSize)
}

func (c *loggingClient) GetPipelineReleaseLogsCount(ctx context.Context, repoSource, repoOwner, repoName string, releaseID string) (count int, err error) {
	defer func() { api.HandleLogError(c.prefix, "Client", "GetPipelineReleaseLogsCount", err) }()

	return c.Client.GetPipelineReleaseLogsCount(ctx, repoSource, repoOwner, repoName, releaseID)
}

func (c *loggingClient) GetPipelineReleaseMaxResourceUtilization(ctx context.Context, repoSource, repoOwner, repoName, targetName string, lastNRecords int) (jobresources JobResources, count int, err error) {
	defer func() { api.HandleLogError(c.prefix, "Client", "GetPipelineReleaseMaxResourceUtilization", err) }()

	return c.Client.GetPipelineReleaseMaxResourceUtilization(ctx, repoSource, repoOwner, repoName, targetName, lastNRecords)
}

func (c *loggingClient) GetPipelineBots(ctx context.Context, repoSource, repoOwner, repoName string, pageNumber, pageSize int, filters map[api.FilterType][]string, sortings []api.OrderField) (bots []*contracts.Bot, err error) {
	defer func() { api.HandleLogError(c.prefix, "Client", "GetPipelineBots", err) }()

	return c.Client.GetPipelineBots(ctx, repoSource, repoOwner, repoName, pageNumber, pageSize, filters, sortings)
}

func (c *loggingClient) GetPipelineBotsCount(ctx context.Context, repoSource, repoOwner, repoName string, filters map[api.FilterType][]string) (count int, err error) {
	defer func() { api.HandleLogError(c.prefix, "Client", "GetPipelineBotsCount", err) }()

	return c.Client.GetPipelineBotsCount(ctx, repoSource, repoOwner, repoName, filters)
}

func (c *loggingClient) GetPipelineBot(ctx context.Context, repoSource, repoOwner, repoName string, botID string) (bot *contracts.Bot, err error) {
	defer func() { api.HandleLogError(c.prefix, "Client", "GetPipelineBot", err) }()

	return c.Client.GetPipelineBot(ctx, repoSource, repoOwner, repoName, botID)
}

func (c *loggingClient) GetPipelineBotLogs(ctx context.Context, repoSource, repoOwner, repoName string, botID string) (releaselog *contracts.BotLog, err error) {
	defer func() { api.HandleLogError(c.prefix, "Client", "GetPipelineBotLogs", err) }()

	return c.Client.GetPipelineBotLogs(ctx, repoSource, repoOwner, repoName, botID)
}

func (c *loggingClient) GetPipelineBotLogsByID(ctx context.Context, repoSource, repoOwner, repoName string, botID string, id string) (releaselog *contracts.BotLog, err error) {
	defer func() { api.HandleLogError(c.prefix, "Client", "GetPipelineBotLogsByID", err) }()

	return c.Client.GetPipelineBotLogsByID(ctx, repoSource, repoOwner, repoName, botID, id)
}

func (c *loggingClient) GetPipelineBotLogsPerPage(ctx context.Context, repoSource, repoOwner, repoName string, botID string, pageNumber int, pageSize int) (releaselogs []*contracts.BotLog, err error) {
	defer func() { api.HandleLogError(c.prefix, "Client", "GetPipelineBotLogsPerPage", err) }()

	return c.Client.GetPipelineBotLogsPerPage(ctx, repoSource, repoOwner, repoName, botID, pageNumber, pageSize)
}

func (c *loggingClient) GetPipelineBotLogsCount(ctx context.Context, repoSource, repoOwner, repoName string, botID string) (count int, err error) {
	defer func() { api.HandleLogError(c.prefix, "Client", "GetPipelineBotLogsCount", err) }()

	return c.Client.GetPipelineBotLogsCount(ctx, repoSource, repoOwner, repoName, botID)
}

func (c *loggingClient) GetPipelineBotMaxResourceUtilization(ctx context.Context, repoSource, repoOwner, repoName, targetName string, lastNRecords int) (jobresources JobResources, count int, err error) {
	defer func() { api.HandleLogError(c.prefix, "Client", "GetPipelineBotMaxResourceUtilization", err) }()

	return c.Client.GetPipelineBotMaxResourceUtilization(ctx, repoSource, repoOwner, repoName, targetName, lastNRecords)
}

func (c *loggingClient) GetBuildsCount(ctx context.Context, filters map[api.FilterType][]string) (count int, err error) {
	defer func() { api.HandleLogError(c.prefix, "Client", "GetBuildsCount", err) }()

	return c.Client.GetBuildsCount(ctx, filters)
}

func (c *loggingClient) GetReleasesCount(ctx context.Context, filters map[api.FilterType][]string) (count int, err error) {
	defer func() { api.HandleLogError(c.prefix, "Client", "GetReleasesCount", err) }()

	return c.Client.GetReleasesCount(ctx, filters)
}

func (c *loggingClient) GetBotsCount(ctx context.Context, filters map[api.FilterType][]string) (count int, err error) {
	defer func() { api.HandleLogError(c.prefix, "Client", "GetBotsCount", err) }()

	return c.Client.GetBotsCount(ctx, filters)
}

func (c *loggingClient) GetBuildsDuration(ctx context.Context, filters map[api.FilterType][]string) (duration time.Duration, err error) {
	defer func() { api.HandleLogError(c.prefix, "Client", "GetBuildsDuration", err) }()

	return c.Client.GetBuildsDuration(ctx, filters)
}

func (c *loggingClient) GetFirstBuildTimes(ctx context.Context) (times []time.Time, err error) {
	defer func() { api.HandleLogError(c.prefix, "Client", "GetFirstBuildTimes", err) }()

	return c.Client.GetFirstBuildTimes(ctx)
}

func (c *loggingClient) GetFirstReleaseTimes(ctx context.Context) (times []time.Time, err error) {
	defer func() { api.HandleLogError(c.prefix, "Client", "GetFirstReleaseTimes", err) }()

	return c.Client.GetFirstReleaseTimes(ctx)
}

func (c *loggingClient) GetFirstBotTimes(ctx context.Context) (times []time.Time, err error) {
	defer func() { api.HandleLogError(c.prefix, "Client", "GetFirstBotTimes", err) }()

	return c.Client.GetFirstBotTimes(ctx)
}

func (c *loggingClient) GetPipelineBuildsDurations(ctx context.Context, repoSource, repoOwner, repoName string, filters map[api.FilterType][]string) (durations []map[string]interface{}, err error) {
	defer func() { api.HandleLogError(c.prefix, "Client", "GetPipelineBuildsDurations", err) }()

	return c.Client.GetPipelineBuildsDurations(ctx, repoSource, repoOwner, repoName, filters)
}

func (c *loggingClient) GetPipelineReleasesDurations(ctx context.Context, repoSource, repoOwner, repoName string, filters map[api.FilterType][]string) (durations []map[string]interface{}, err error) {
	defer func() { api.HandleLogError(c.prefix, "Client", "GetPipelineReleasesDurations", err) }()

	return c.Client.GetPipelineReleasesDurations(ctx, repoSource, repoOwner, repoName, filters)
}

func (c *loggingClient) GetPipelineBotsDurations(ctx context.Context, repoSource, repoOwner, repoName string, filters map[api.FilterType][]string) (durations []map[string]interface{}, err error) {
	defer func() { api.HandleLogError(c.prefix, "Client", "GetPipelineBotsDurations", err) }()

	return c.Client.GetPipelineBotsDurations(ctx, repoSource, repoOwner, repoName, filters)
}

func (c *loggingClient) GetPipelineBuildsCPUUsageMeasurements(ctx context.Context, repoSource, repoOwner, repoName string, filters map[api.FilterType][]string) (measurements []map[string]interface{}, err error) {
	defer func() { api.HandleLogError(c.prefix, "Client", "GetPipelineBuildsCPUUsageMeasurements", err) }()

	return c.Client.GetPipelineBuildsCPUUsageMeasurements(ctx, repoSource, repoOwner, repoName, filters)
}

func (c *loggingClient) GetPipelineReleasesCPUUsageMeasurements(ctx context.Context, repoSource, repoOwner, repoName string, filters map[api.FilterType][]string) (measurements []map[string]interface{}, err error) {
	defer func() { api.HandleLogError(c.prefix, "Client", "GetPipelineReleasesCPUUsageMeasurements", err) }()

	return c.Client.GetPipelineReleasesCPUUsageMeasurements(ctx, repoSource, repoOwner, repoName, filters)
}

func (c *loggingClient) GetPipelineBotsCPUUsageMeasurements(ctx context.Context, repoSource, repoOwner, repoName string, filters map[api.FilterType][]string) (measurements []map[string]interface{}, err error) {
	defer func() { api.HandleLogError(c.prefix, "Client", "GetPipelineBotsCPUUsageMeasurements", err) }()

	return c.Client.GetPipelineBotsCPUUsageMeasurements(ctx, repoSource, repoOwner, repoName, filters)
}

func (c *loggingClient) GetPipelineBuildsMemoryUsageMeasurements(ctx context.Context, repoSource, repoOwner, repoName string, filters map[api.FilterType][]string) (measurements []map[string]interface{}, err error) {
	defer func() { api.HandleLogError(c.prefix, "Client", "GetPipelineBuildsMemoryUsageMeasurements", err) }()

	return c.Client.GetPipelineBuildsMemoryUsageMeasurements(ctx, repoSource, repoOwner, repoName, filters)
}

func (c *loggingClient) GetPipelineReleasesMemoryUsageMeasurements(ctx context.Context, repoSource, repoOwner, repoName string, filters map[api.FilterType][]string) (measurements []map[string]interface{}, err error) {
	defer func() { api.HandleLogError(c.prefix, "Client", "GetPipelineReleasesMemoryUsageMeasurements", err) }()

	return c.Client.GetPipelineReleasesMemoryUsageMeasurements(ctx, repoSource, repoOwner, repoName, filters)
}

func (c *loggingClient) GetPipelineBotsMemoryUsageMeasurements(ctx context.Context, repoSource, repoOwner, repoName string, filters map[api.FilterType][]string) (measurements []map[string]interface{}, err error) {
	defer func() { api.HandleLogError(c.prefix, "Client", "GetPipelineBotsMemoryUsageMeasurements", err) }()

	return c.Client.GetPipelineBotsMemoryUsageMeasurements(ctx, repoSource, repoOwner, repoName, filters)
}

func (c *loggingClient) GetLabelValues(ctx context.Context, labelKey string) (labels []map[string]interface{}, err error) {
	defer func() { api.HandleLogError(c.prefix, "Client", "GetLabelValues", err) }()

	return c.Client.GetLabelValues(ctx, labelKey)
}

func (c *loggingClient) GetFrequentLabels(ctx context.Context, pageNumber, pageSize int, filters map[api.FilterType][]string) (measurements []map[string]interface{}, err error) {
	defer func() { api.HandleLogError(c.prefix, "Client", "GetFrequentLabels", err) }()

	return c.Client.GetFrequentLabels(ctx, pageNumber, pageSize, filters)
}

func (c *loggingClient) GetFrequentLabelsCount(ctx context.Context, filters map[api.FilterType][]string) (count int, err error) {
	defer func() { api.HandleLogError(c.prefix, "Client", "GetFrequentLabelsCount", err) }()

	return c.Client.GetFrequentLabelsCount(ctx, filters)
}

func (c *loggingClient) GetPipelinesWithMostBuilds(ctx context.Context, pageNumber, pageSize int, filters map[api.FilterType][]string) (pipelines []map[string]interface{}, err error) {
	defer func() { api.HandleLogError(c.prefix, "Client", "GetPipelinesWithMostBuilds", err) }()

	return c.Client.GetPipelinesWithMostBuilds(ctx, pageNumber, pageSize, filters)
}

func (c *loggingClient) GetPipelinesWithMostBuildsCount(ctx context.Context, filters map[api.FilterType][]string) (count int, err error) {
	defer func() { api.HandleLogError(c.prefix, "Client", "GetPipelinesWithMostBuildsCount", err) }()

	return c.Client.GetPipelinesWithMostBuildsCount(ctx, filters)
}

func (c *loggingClient) GetPipelinesWithMostReleases(ctx context.Context, pageNumber, pageSize int, filters map[api.FilterType][]string) (pipelines []map[string]interface{}, err error) {
	defer func() { api.HandleLogError(c.prefix, "Client", "GetPipelinesWithMostReleases", err) }()

	return c.Client.GetPipelinesWithMostReleases(ctx, pageNumber, pageSize, filters)
}

func (c *loggingClient) GetPipelinesWithMostReleasesCount(ctx context.Context, filters map[api.FilterType][]string) (count int, err error) {
	defer func() { api.HandleLogError(c.prefix, "Client", "GetPipelinesWithMostReleasesCount", err) }()

	return c.Client.GetPipelinesWithMostReleasesCount(ctx, filters)
}

func (c *loggingClient) GetPipelinesWithMostBots(ctx context.Context, pageNumber, pageSize int, filters map[api.FilterType][]string) (pipelines []map[string]interface{}, err error) {
	defer func() { api.HandleLogError(c.prefix, "Client", "GetPipelinesWithMostBots", err) }()

	return c.Client.GetPipelinesWithMostBots(ctx, pageNumber, pageSize, filters)
}

func (c *loggingClient) GetPipelinesWithMostBotsCount(ctx context.Context, filters map[api.FilterType][]string) (count int, err error) {
	defer func() { api.HandleLogError(c.prefix, "Client", "GetPipelinesWithMostBotsCount", err) }()

	return c.Client.GetPipelinesWithMostBotsCount(ctx, filters)
}

func (c *loggingClient) GetTriggers(ctx context.Context, triggerType, identifier, event string) (pipelines []*contracts.Pipeline, err error) {
	defer func() { api.HandleLogError(c.prefix, "Client", "GetTriggers", err) }()

	return c.Client.GetTriggers(ctx, triggerType, identifier, event)
}

func (c *loggingClient) GetGitTriggers(ctx context.Context, gitEvent manifest.EstafetteGitEvent) (pipelines []*contracts.Pipeline, err error) {
	defer func() { api.HandleLogError(c.prefix, "Client", "GetGitTriggers", err) }()

	return c.Client.GetGitTriggers(ctx, gitEvent)
}

func (c *loggingClient) GetPipelineTriggers(ctx context.Context, build contracts.Build, event string) (pipelines []*contracts.Pipeline, err error) {
	defer func() { api.HandleLogError(c.prefix, "Client", "GetPipelineTriggers", err) }()

	return c.Client.GetPipelineTriggers(ctx, build, event)
}

func (c *loggingClient) GetReleaseTriggers(ctx context.Context, release contracts.Release, event string) (pipelines []*contracts.Pipeline, err error) {
	defer func() { api.HandleLogError(c.prefix, "Client", "GetReleaseTriggers", err) }()

	return c.Client.GetReleaseTriggers(ctx, release, event)
}

func (c *loggingClient) GetPubSubTriggers(ctx context.Context) (pipelines []*contracts.Pipeline, err error) {
	defer func() { api.HandleLogError(c.prefix, "Client", "GetPubSubTriggers", err) }()

	return c.Client.GetPubSubTriggers(ctx)
}

func (c *loggingClient) GetCronTriggers(ctx context.Context) (pipelines []*contracts.Pipeline, err error) {
	defer func() { api.HandleLogError(c.prefix, "Client", "GetCronTriggers", err) }()

	return c.Client.GetCronTriggers(ctx)
}

func (c *loggingClient) GetGithubTriggers(ctx context.Context, githubEvent manifest.EstafetteGithubEvent) (pipelines []*contracts.Pipeline, err error) {
	defer func() { api.HandleLogError(c.prefix, "Client", "GetGithubTriggers", err) }()

	return c.Client.GetGithubTriggers(ctx, githubEvent)
}

func (c *loggingClient) GetBitbucketTriggers(ctx context.Context, bitbucketEvent manifest.EstafetteBitbucketEvent) (pipelines []*contracts.Pipeline, err error) {
	defer func() { api.HandleLogError(c.prefix, "Client", "GetBitbucketTriggers", err) }()

	return c.Client.GetBitbucketTriggers(ctx, bitbucketEvent)
}

func (c *loggingClient) Rename(ctx context.Context, shortFromRepoSource, fromRepoSource, fromRepoOwner, fromRepoName, shortToRepoSource, toRepoSource, toRepoOwner, toRepoName string) (err error) {
	defer func() { api.HandleLogError(c.prefix, "Client", "Rename", err) }()

	return c.Client.Rename(ctx, shortFromRepoSource, fromRepoSource, fromRepoOwner, fromRepoName, shortToRepoSource, toRepoSource, toRepoOwner, toRepoName)
}

func (c *loggingClient) RenameBuildVersion(ctx context.Context, shortFromRepoSource, fromRepoOwner, fromRepoName, shortToRepoSource, toRepoOwner, toRepoName string) (err error) {
	defer func() { api.HandleLogError(c.prefix, "Client", "RenameBuildVersion", err) }()

	return c.Client.RenameBuildVersion(ctx, shortFromRepoSource, fromRepoOwner, fromRepoName, shortToRepoSource, toRepoOwner, toRepoName)
}

func (c *loggingClient) RenameBuilds(ctx context.Context, fromRepoSource, fromRepoOwner, fromRepoName, toRepoSource, toRepoOwner, toRepoName string) (err error) {
	defer func() { api.HandleLogError(c.prefix, "Client", "RenameBuilds", err) }()

	return c.Client.RenameBuilds(ctx, fromRepoSource, fromRepoOwner, fromRepoName, toRepoSource, toRepoOwner, toRepoName)
}

func (c *loggingClient) RenameBuildLogs(ctx context.Context, fromRepoSource, fromRepoOwner, fromRepoName, toRepoSource, toRepoOwner, toRepoName string) (err error) {
	defer func() { api.HandleLogError(c.prefix, "Client", "RenameBuildLogs", err) }()

	return c.Client.RenameBuildLogs(ctx, fromRepoSource, fromRepoOwner, fromRepoName, toRepoSource, toRepoOwner, toRepoName)
}

func (c *loggingClient) RenameReleases(ctx context.Context, fromRepoSource, fromRepoOwner, fromRepoName, toRepoSource, toRepoOwner, toRepoName string) (err error) {
	defer func() { api.HandleLogError(c.prefix, "Client", "RenameReleases", err) }()

	return c.Client.RenameReleases(ctx, fromRepoSource, fromRepoOwner, fromRepoName, toRepoSource, toRepoOwner, toRepoName)
}

func (c *loggingClient) RenameReleaseLogs(ctx context.Context, fromRepoSource, fromRepoOwner, fromRepoName, toRepoSource, toRepoOwner, toRepoName string) (err error) {
	defer func() { api.HandleLogError(c.prefix, "Client", "RenameReleaseLogs", err) }()

	return c.Client.RenameReleaseLogs(ctx, fromRepoSource, fromRepoOwner, fromRepoName, toRepoSource, toRepoOwner, toRepoName)
}

func (c *loggingClient) RenameComputedPipelines(ctx context.Context, fromRepoSource, fromRepoOwner, fromRepoName, toRepoSource, toRepoOwner, toRepoName string) (err error) {
	defer func() { api.HandleLogError(c.prefix, "Client", "RenameComputedPipelines", err) }()

	return c.Client.RenameComputedPipelines(ctx, fromRepoSource, fromRepoOwner, fromRepoName, toRepoSource, toRepoOwner, toRepoName)
}

func (c *loggingClient) RenameComputedReleases(ctx context.Context, fromRepoSource, fromRepoOwner, fromRepoName, toRepoSource, toRepoOwner, toRepoName string) (err error) {
	defer func() { api.HandleLogError(c.prefix, "Client", "RenameComputedReleases", err) }()

	return c.Client.RenameComputedReleases(ctx, fromRepoSource, fromRepoOwner, fromRepoName, toRepoSource, toRepoOwner, toRepoName)
}

func (c *loggingClient) InsertUser(ctx context.Context, user contracts.User) (u *contracts.User, err error) {
	defer func() { api.HandleLogError(c.prefix, "Client", "InsertUser", err) }()

	return c.Client.InsertUser(ctx, user)
}

func (c *loggingClient) UpdateUser(ctx context.Context, user contracts.User) (err error) {
	defer func() { api.HandleLogError(c.prefix, "Client", "UpdateUser", err) }()

	return c.Client.UpdateUser(ctx, user)
}

func (c *loggingClient) DeleteUser(ctx context.Context, user contracts.User) (err error) {
	defer func() { api.HandleLogError(c.prefix, "Client", "DeleteUser", err) }()

	return c.Client.DeleteUser(ctx, user)
}

func (c *loggingClient) GetUserByIdentity(ctx context.Context, identity contracts.UserIdentity) (user *contracts.User, err error) {
	defer func() { api.HandleLogError(c.prefix, "Client", "GetUserByIdentity", err, ErrUserNotFound) }()

	return c.Client.GetUserByIdentity(ctx, identity)
}

func (c *loggingClient) GetUserByID(ctx context.Context, id string, filters map[api.FilterType][]string) (user *contracts.User, err error) {
	defer func() { api.HandleLogError(c.prefix, "Client", "GetUserByID", err, ErrUserNotFound) }()

	return c.Client.GetUserByID(ctx, id, filters)
}

func (c *loggingClient) GetUsers(ctx context.Context, pageNumber, pageSize int, filters map[api.FilterType][]string, sortings []api.OrderField) (users []*contracts.User, err error) {
	defer func() { api.HandleLogError(c.prefix, "Client", "GetUsers", err) }()

	return c.Client.GetUsers(ctx, pageNumber, pageSize, filters, sortings)
}

func (c *loggingClient) GetUsersCount(ctx context.Context, filters map[api.FilterType][]string) (count int, err error) {
	defer func() { api.HandleLogError(c.prefix, "Client", "GetUsersCount", err) }()

	return c.Client.GetUsersCount(ctx, filters)
}

func (c *loggingClient) InsertGroup(ctx context.Context, group contracts.Group) (g *contracts.Group, err error) {
	defer func() { api.HandleLogError(c.prefix, "Client", "InsertGroup", err) }()

	return c.Client.InsertGroup(ctx, group)
}

func (c *loggingClient) UpdateGroup(ctx context.Context, group contracts.Group) (err error) {
	defer func() { api.HandleLogError(c.prefix, "Client", "UpdateGroup", err) }()

	return c.Client.UpdateGroup(ctx, group)
}

func (c *loggingClient) DeleteGroup(ctx context.Context, group contracts.Group) (err error) {
	defer func() { api.HandleLogError(c.prefix, "Client", "DeleteGroup", err) }()

	return c.Client.DeleteGroup(ctx, group)
}

func (c *loggingClient) GetGroupByIdentity(ctx context.Context, identity contracts.GroupIdentity) (group *contracts.Group, err error) {
	defer func() { api.HandleLogError(c.prefix, "Client", "GetGroupByIdentity", err, ErrGroupNotFound) }()

	return c.Client.GetGroupByIdentity(ctx, identity)
}

func (c *loggingClient) GetGroupByID(ctx context.Context, id string, filters map[api.FilterType][]string) (group *contracts.Group, err error) {
	defer func() { api.HandleLogError(c.prefix, "Client", "GetGroupByID", err, ErrGroupNotFound) }()

	return c.Client.GetGroupByID(ctx, id, filters)
}

func (c *loggingClient) GetGroups(ctx context.Context, pageNumber, pageSize int, filters map[api.FilterType][]string, sortings []api.OrderField) (groups []*contracts.Group, err error) {
	defer func() { api.HandleLogError(c.prefix, "Client", "GetGroups", err) }()

	return c.Client.GetGroups(ctx, pageNumber, pageSize, filters, sortings)
}

func (c *loggingClient) GetGroupsCount(ctx context.Context, filters map[api.FilterType][]string) (count int, err error) {
	defer func() { api.HandleLogError(c.prefix, "Client", "GetGroupsCount", err) }()

	return c.Client.GetGroupsCount(ctx, filters)
}

func (c *loggingClient) InsertOrganization(ctx context.Context, organization contracts.Organization) (o *contracts.Organization, err error) {
	defer func() { api.HandleLogError(c.prefix, "Client", "InsertOrganization", err) }()

	return c.Client.InsertOrganization(ctx, organization)
}

func (c *loggingClient) UpdateOrganization(ctx context.Context, organization contracts.Organization) (err error) {
	defer func() { api.HandleLogError(c.prefix, "Client", "UpdateOrganization", err) }()

	return c.Client.UpdateOrganization(ctx, organization)
}

func (c *loggingClient) DeleteOrganization(ctx context.Context, organization contracts.Organization) (err error) {
	defer func() { api.HandleLogError(c.prefix, "Client", "DeleteOrganization", err) }()

	return c.Client.DeleteOrganization(ctx, organization)
}

func (c *loggingClient) GetOrganizationByIdentity(ctx context.Context, identity contracts.OrganizationIdentity) (organization *contracts.Organization, err error) {
	defer func() {
		api.HandleLogError(c.prefix, "Client", "GetOrganizationByIdentity", err, ErrOrganizationNotFound)
	}()

	return c.Client.GetOrganizationByIdentity(ctx, identity)
}

func (c *loggingClient) GetOrganizationByID(ctx context.Context, id string) (organization *contracts.Organization, err error) {
	defer func() { api.HandleLogError(c.prefix, "Client", "GetOrganizationByID", err, ErrOrganizationNotFound) }()

	return c.Client.GetOrganizationByID(ctx, id)
}

func (c *loggingClient) GetOrganizationByName(ctx context.Context, name string) (organization *contracts.Organization, err error) {
	defer func() { api.HandleLogError(c.prefix, "Client", "GetOrganizationByName", err, ErrOrganizationNotFound) }()

	return c.Client.GetOrganizationByName(ctx, name)
}

func (c *loggingClient) GetOrganizations(ctx context.Context, pageNumber, pageSize int, filters map[api.FilterType][]string, sortings []api.OrderField) (organizations []*contracts.Organization, err error) {
	defer func() { api.HandleLogError(c.prefix, "Client", "GetGroups", err) }()

	return c.Client.GetOrganizations(ctx, pageNumber, pageSize, filters, sortings)
}

func (c *loggingClient) GetOrganizationsCount(ctx context.Context, filters map[api.FilterType][]string) (count int, err error) {
	defer func() { api.HandleLogError(c.prefix, "Client", "GetGroupsCount", err) }()

	return c.Client.GetOrganizationsCount(ctx, filters)
}

func (c *loggingClient) InsertClient(ctx context.Context, client contracts.Client) (cl *contracts.Client, err error) {
	defer func() { api.HandleLogError(c.prefix, "Client", "InsertClient", err) }()

	return c.Client.InsertClient(ctx, client)
}

func (c *loggingClient) UpdateClient(ctx context.Context, client contracts.Client) (err error) {
	defer func() { api.HandleLogError(c.prefix, "Client", "UpdateClient", err) }()

	return c.Client.UpdateClient(ctx, client)
}

func (c *loggingClient) DeleteClient(ctx context.Context, client contracts.Client) (err error) {
	defer func() { api.HandleLogError(c.prefix, "Client", "DeleteClient", err) }()

	return c.Client.DeleteClient(ctx, client)
}

func (c *loggingClient) GetClientByClientID(ctx context.Context, clientID string) (client *contracts.Client, err error) {
	defer func() { api.HandleLogError(c.prefix, "Client", "GetClientByClientID", err, ErrClientNotFound) }()

	return c.Client.GetClientByClientID(ctx, clientID)
}

func (c *loggingClient) GetClientByID(ctx context.Context, id string) (client *contracts.Client, err error) {
	defer func() { api.HandleLogError(c.prefix, "Client", "GetClientByID", err, ErrClientNotFound) }()

	return c.Client.GetClientByID(ctx, id)
}

func (c *loggingClient) GetClients(ctx context.Context, pageNumber, pageSize int, filters map[api.FilterType][]string, sortings []api.OrderField) (clients []*contracts.Client, err error) {
	defer func() { api.HandleLogError(c.prefix, "Client", "GetClients", err) }()

	return c.Client.GetClients(ctx, pageNumber, pageSize, filters, sortings)
}

func (c *loggingClient) GetClientsCount(ctx context.Context, filters map[api.FilterType][]string) (count int, err error) {
	defer func() { api.HandleLogError(c.prefix, "Client", "GetClientsCount", err) }()

	return c.Client.GetClientsCount(ctx, filters)
}

func (c *loggingClient) InsertCatalogEntity(ctx context.Context, catalogEntity contracts.CatalogEntity) (insertedCatalogEntity *contracts.CatalogEntity, err error) {
	defer func() { api.HandleLogError(c.prefix, "Client", "InsertCatalogEntity", err) }()

	return c.Client.InsertCatalogEntity(ctx, catalogEntity)
}

func (c *loggingClient) UpdateCatalogEntity(ctx context.Context, catalogEntity contracts.CatalogEntity) (err error) {
	defer func() { api.HandleLogError(c.prefix, "Client", "UpdateCatalogEntity", err) }()

	return c.Client.UpdateCatalogEntity(ctx, catalogEntity)
}

func (c *loggingClient) DeleteCatalogEntity(ctx context.Context, id string) (err error) {
	defer func() { api.HandleLogError(c.prefix, "Client", "DeleteCatalogEntity", err) }()

	return c.Client.DeleteCatalogEntity(ctx, id)
}

func (c *loggingClient) GetCatalogEntityByID(ctx context.Context, id string) (catalogEntity *contracts.CatalogEntity, err error) {
	defer func() { api.HandleLogError(c.prefix, "Client", "GetCatalogEntityByID", err) }()

	return c.Client.GetCatalogEntityByID(ctx, id)
}

func (c *loggingClient) GetCatalogEntities(ctx context.Context, pageNumber, pageSize int, filters map[api.FilterType][]string, sortings []api.OrderField) (catalogEntities []*contracts.CatalogEntity, err error) {
	defer func() { api.HandleLogError(c.prefix, "Client", "GetCatalogEntities", err) }()

	return c.Client.GetCatalogEntities(ctx, pageNumber, pageSize, filters, sortings)
}

func (c *loggingClient) GetCatalogEntitiesCount(ctx context.Context, filters map[api.FilterType][]string) (count int, err error) {
	defer func() { api.HandleLogError(c.prefix, "Client", "GetCatalogEntitiesCount", err) }()

	return c.Client.GetCatalogEntitiesCount(ctx, filters)
}

func (c *loggingClient) GetCatalogEntityParentKeys(ctx context.Context, pageNumber, pageSize int, filters map[api.FilterType][]string, sortings []api.OrderField) (keys []map[string]interface{}, err error) {
	defer func() { api.HandleLogError(c.prefix, "Client", "GetCatalogEntityParentKeys", err) }()

	return c.Client.GetCatalogEntityParentKeys(ctx, pageNumber, pageSize, filters, sortings)
}

func (c *loggingClient) GetCatalogEntityParentKeysCount(ctx context.Context, filters map[api.FilterType][]string) (count int, err error) {
	defer func() { api.HandleLogError(c.prefix, "Client", "GetCatalogEntityParentKeysCount", err) }()

	return c.Client.GetCatalogEntityParentKeysCount(ctx, filters)
}

func (c *loggingClient) GetCatalogEntityParentValues(ctx context.Context, pageNumber, pageSize int, filters map[api.FilterType][]string, sortings []api.OrderField) (values []map[string]interface{}, err error) {
	defer func() { api.HandleLogError(c.prefix, "Client", "GetCatalogEntityParentValues", err) }()

	return c.Client.GetCatalogEntityParentValues(ctx, pageNumber, pageSize, filters, sortings)
}

func (c *loggingClient) GetCatalogEntityParentValuesCount(ctx context.Context, filters map[api.FilterType][]string) (count int, err error) {
	defer func() { api.HandleLogError(c.prefix, "Client", "GetCatalogEntityParentValuesCount", err) }()

	return c.Client.GetCatalogEntityParentValuesCount(ctx, filters)
}

func (c *loggingClient) GetCatalogEntityKeys(ctx context.Context, pageNumber, pageSize int, filters map[api.FilterType][]string, sortings []api.OrderField) (keys []map[string]interface{}, err error) {
	defer func() { api.HandleLogError(c.prefix, "Client", "GetCatalogEntityKeys", err) }()

	return c.Client.GetCatalogEntityKeys(ctx, pageNumber, pageSize, filters, sortings)
}

func (c *loggingClient) GetCatalogEntityKeysCount(ctx context.Context, filters map[api.FilterType][]string) (count int, err error) {
	defer func() { api.HandleLogError(c.prefix, "Client", "GetCatalogEntityKeysCount", err) }()

	return c.Client.GetCatalogEntityKeysCount(ctx, filters)
}

func (c *loggingClient) GetCatalogEntityValues(ctx context.Context, pageNumber, pageSize int, filters map[api.FilterType][]string, sortings []api.OrderField) (values []map[string]interface{}, err error) {
	defer func() { api.HandleLogError(c.prefix, "Client", "GetCatalogEntityValues", err) }()

	return c.Client.GetCatalogEntityValues(ctx, pageNumber, pageSize, filters, sortings)
}

func (c *loggingClient) GetCatalogEntityValuesCount(ctx context.Context, filters map[api.FilterType][]string) (count int, err error) {
	defer func() { api.HandleLogError(c.prefix, "Client", "GetCatalogEntityValuesCount", err) }()

	return c.Client.GetCatalogEntityValuesCount(ctx, filters)
}

func (c *loggingClient) GetCatalogEntityLabels(ctx context.Context, pageNumber, pageSize int, filters map[api.FilterType][]string) (labels []map[string]interface{}, err error) {
	defer func() { api.HandleLogError(c.prefix, "Client", "GetCatalogEntityLabels", err) }()

	return c.Client.GetCatalogEntityLabels(ctx, pageNumber, pageSize, filters)
}

func (c *loggingClient) GetCatalogEntityLabelsCount(ctx context.Context, filters map[api.FilterType][]string) (count int, err error) {
	defer func() { api.HandleLogError(c.prefix, "Client", "GetCatalogEntityLabelsCount", err) }()

	return c.Client.GetCatalogEntityLabelsCount(ctx, filters)
}

func (c *loggingClient) GetAllPipelineBuilds(ctx context.Context, pageNumber, pageSize int, filters map[api.FilterType][]string, sortings []api.OrderField, optimized bool) (builds []*contracts.Build, err error) {
	defer func() { api.HandleLogError(c.prefix, "Client", "GetAllPipelineBuilds", err) }()

	return c.Client.GetAllPipelineBuilds(ctx, pageNumber, pageSize, filters, sortings, optimized)
}

func (c *loggingClient) GetAllPipelineBuildsCount(ctx context.Context, filters map[api.FilterType][]string) (count int, err error) {
	defer func() { api.HandleLogError(c.prefix, "Client", "GetAllPipelineBuildsCount", err) }()

	return c.Client.GetAllPipelineBuildsCount(ctx, filters)
}

func (c *loggingClient) GetAllPipelineReleases(ctx context.Context, pageNumber, pageSize int, filters map[api.FilterType][]string, sortings []api.OrderField) (releases []*contracts.Release, err error) {
	defer func() { api.HandleLogError(c.prefix, "Client", "GetAllPipelineReleases", err) }()

	return c.Client.GetAllPipelineReleases(ctx, pageNumber, pageSize, filters, sortings)
}

func (c *loggingClient) GetAllPipelineReleasesCount(ctx context.Context, filters map[api.FilterType][]string) (count int, err error) {
	defer func() { api.HandleLogError(c.prefix, "Client", "GetAllPipelineReleasesCount", err) }()

	return c.Client.GetAllPipelineReleasesCount(ctx, filters)
}

func (c *loggingClient) GetAllPipelineBots(ctx context.Context, pageNumber, pageSize int, filters map[api.FilterType][]string, sortings []api.OrderField) (bots []*contracts.Bot, err error) {
	defer func() { api.HandleLogError(c.prefix, "Client", "GetAllPipelineBots", err) }()

	return c.Client.GetAllPipelineBots(ctx, pageNumber, pageSize, filters, sortings)
}

func (c *loggingClient) GetAllPipelineBotsCount(ctx context.Context, filters map[api.FilterType][]string) (count int, err error) {
	defer func() { api.HandleLogError(c.prefix, "Client", "GetAllPipelineBotsCount", err) }()

	return c.Client.GetAllPipelineBotsCount(ctx, filters)
}

func (c *loggingClient) GetAllNotifications(ctx context.Context, pageNumber, pageSize int, filters map[api.FilterType][]string, sortings []api.OrderField) (notifications []*contracts.NotificationRecord, err error) {
	defer func() { api.HandleLogError(c.prefix, "Client", "GetAllNotifications", err) }()

	return c.Client.GetAllNotifications(ctx, pageNumber, pageSize, filters, sortings)
}

func (c *loggingClient) GetAllNotificationsCount(ctx context.Context, filters map[api.FilterType][]string) (count int, err error) {
	defer func() { api.HandleLogError(c.prefix, "Client", "GetAllNotificationsCount", err) }()

	return c.Client.GetAllNotificationsCount(ctx, filters)
}

func (c *loggingClient) InsertNotification(ctx context.Context, notificationRecord contracts.NotificationRecord) (n *contracts.NotificationRecord, err error) {
	defer func() { api.HandleLogError(c.prefix, "Client", "InsertNotification", err) }()

	return c.Client.InsertNotification(ctx, notificationRecord)
}

func (c *loggingClient) GetReleaseTargets(ctx context.Context, pageNumber, pageSize int, filters map[api.FilterType][]string) (releaseTargets []map[string]interface{}, err error) {
	defer func() { api.HandleLogError(c.prefix, "Client", "GetReleaseTargets", err) }()

	return c.Client.GetReleaseTargets(ctx, pageNumber, pageSize, filters)
}

func (c *loggingClient) GetReleaseTargetsCount(ctx context.Context, filters map[api.FilterType][]string) (count int, err error) {
	defer func() { api.HandleLogError(c.prefix, "Client", "GetReleaseTargetsCount", err) }()

	return c.Client.GetReleaseTargetsCount(ctx, filters)
}

func (c *loggingClient) GetAllPipelinesReleaseTargets(ctx context.Context, pageNumber, pageSize int, filters map[api.FilterType][]string) (releaseTargets []map[string]interface{}, err error) {
	defer func() { api.HandleLogError(c.prefix, "Client", "GetAllPipelinesReleaseTargets", err) }()

	return c.Client.GetAllPipelinesReleaseTargets(ctx, pageNumber, pageSize, filters)
}

func (c *loggingClient) GetAllPipelinesReleaseTargetsCount(ctx context.Context, filters map[api.FilterType][]string) (count int, err error) {
	defer func() { api.HandleLogError(c.prefix, "Client", "GetAllPipelinesReleaseTargetsCount", err) }()

	return c.Client.GetAllPipelinesReleaseTargetsCount(ctx, filters)
}

func (c *loggingClient) GetAllReleasesReleaseTargets(ctx context.Context, pageNumber, pageSize int, filters map[api.FilterType][]string) (releaseTargets []map[string]interface{}, err error) {
	defer func() { api.HandleLogError(c.prefix, "Client", "GetAllReleasesReleaseTargets", err) }()

	return c.Client.GetAllReleasesReleaseTargets(ctx, pageNumber, pageSize, filters)
}

func (c *loggingClient) GetAllReleasesReleaseTargetsCount(ctx context.Context, filters map[api.FilterType][]string) (count int, err error) {
	defer func() { api.HandleLogError(c.prefix, "Client", "GetAllReleasesReleaseTargetsCount", err) }()

	return c.Client.GetAllReleasesReleaseTargetsCount(ctx, filters)
}

func (c *loggingClient) GetPipelineBuildBranches(ctx context.Context, repoSource, repoOwner, repoName string, pageNumber, pageSize int, filters map[api.FilterType][]string) (buildBranches []map[string]interface{}, err error) {
	defer func() { api.HandleLogError(c.prefix, "Client", "GetPipelineBuildBranches", err) }()

	return c.Client.GetPipelineBuildBranches(ctx, repoSource, repoOwner, repoName, pageNumber, pageSize, filters)
}

func (c *loggingClient) GetPipelineBuildBranchesCount(ctx context.Context, repoSource, repoOwner, repoName string, filters map[api.FilterType][]string) (count int, err error) {
	defer func() { api.HandleLogError(c.prefix, "Client", "GetPipelineBuildBranchesCount", err) }()

	return c.Client.GetPipelineBuildBranchesCount(ctx, repoSource, repoOwner, repoName, filters)
}

func (c *loggingClient) GetPipelineBotNames(ctx context.Context, repoSource, repoOwner, repoName string, pageNumber, pageSize int, filters map[api.FilterType][]string) (botNames []map[string]interface{}, err error) {
	defer func() { api.HandleLogError(c.prefix, "Client", "GetPipelineBotNames", err) }()

	return c.Client.GetPipelineBotNames(ctx, repoSource, repoOwner, repoName, pageNumber, pageSize, filters)
}

func (c *loggingClient) GetPipelineBotNamesCount(ctx context.Context, repoSource, repoOwner, repoName string, filters map[api.FilterType][]string) (count int, err error) {
	defer func() { api.HandleLogError(c.prefix, "Client", "GetPipelineBotNamesCount", err) }()

	return c.Client.GetPipelineBotNamesCount(ctx, repoSource, repoOwner, repoName, filters)
}
