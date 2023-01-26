package database

import (
	"context"
	"time"

	"github.com/estafette/estafette-ci-api/pkg/api"
	contracts "github.com/estafette/estafette-ci-contracts"
	manifest "github.com/estafette/estafette-ci-manifest"
	"github.com/go-kit/kit/metrics"
)

// NewMetricsClient returns a new instance of a metrics Client.
func NewMetricsClient(c Client, requestCount metrics.Counter, requestLatency metrics.Histogram) Client {
	return &metricsClient{c, requestCount, requestLatency}
}

type metricsClient struct {
	Client         Client
	requestCount   metrics.Counter
	requestLatency metrics.Histogram
}

func (c *metricsClient) Connect(ctx context.Context) (err error) {
	defer func(begin time.Time) { api.UpdateMetrics(c.requestCount, c.requestLatency, "Connect", begin) }(time.Now())

	return c.Client.Connect(ctx)
}

func (c *metricsClient) ConnectWithDriverAndSource(ctx context.Context, driverName, dataSourceName string) (err error) {
	defer func(begin time.Time) {
		api.UpdateMetrics(c.requestCount, c.requestLatency, "ConnectWithDriverAndSource", begin)
	}(time.Now())

	return c.Client.ConnectWithDriverAndSource(ctx, driverName, dataSourceName)
}

func (c *metricsClient) AwaitDatabaseReadiness(ctx context.Context) (err error) {
	defer func(begin time.Time) {
		api.UpdateMetrics(c.requestCount, c.requestLatency, "AwaitDatabaseReadiness", begin)
	}(time.Now())

	return c.Client.AwaitDatabaseReadiness(ctx)
}

func (c *metricsClient) GetAutoIncrement(ctx context.Context, shortRepoSource, repoOwner, repoName string) (autoincrement int, err error) {
	defer func(begin time.Time) {
		api.UpdateMetrics(c.requestCount, c.requestLatency, "GetAutoIncrement", begin)
	}(time.Now())

	return c.Client.GetAutoIncrement(ctx, shortRepoSource, repoOwner, repoName)
}

func (c *metricsClient) InsertBuild(ctx context.Context, build contracts.Build, jobResources JobResources) (b *contracts.Build, err error) {
	defer func(begin time.Time) { api.UpdateMetrics(c.requestCount, c.requestLatency, "InsertBuild", begin) }(time.Now())

	return c.Client.InsertBuild(ctx, build, jobResources)
}

func (c *metricsClient) UpdateBuildStatus(ctx context.Context, repoSource, repoOwner, repoName string, buildID string, buildStatus contracts.Status) (err error) {
	defer func(begin time.Time) {
		api.UpdateMetrics(c.requestCount, c.requestLatency, "UpdateBuildStatus", begin)
	}(time.Now())

	return c.Client.UpdateBuildStatus(ctx, repoSource, repoOwner, repoName, buildID, buildStatus)
}

func (c *metricsClient) UpdateBuildResourceUtilization(ctx context.Context, repoSource, repoOwner, repoName string, buildID string, jobResources JobResources) (err error) {
	defer func(begin time.Time) {
		api.UpdateMetrics(c.requestCount, c.requestLatency, "UpdateBuildResourceUtilization", begin)
	}(time.Now())

	return c.Client.UpdateBuildResourceUtilization(ctx, repoSource, repoOwner, repoName, buildID, jobResources)
}

func (c *metricsClient) InsertRelease(ctx context.Context, release contracts.Release, jobResources JobResources) (r *contracts.Release, err error) {
	defer func(begin time.Time) { api.UpdateMetrics(c.requestCount, c.requestLatency, "InsertRelease", begin) }(time.Now())

	return c.Client.InsertRelease(ctx, release, jobResources)
}

func (c *metricsClient) UpdateReleaseStatus(ctx context.Context, repoSource, repoOwner, repoName string, releaseID string, releaseStatus contracts.Status) (err error) {
	defer func(begin time.Time) {
		api.UpdateMetrics(c.requestCount, c.requestLatency, "UpdateReleaseStatus", begin)
	}(time.Now())

	return c.Client.UpdateReleaseStatus(ctx, repoSource, repoOwner, repoName, releaseID, releaseStatus)
}

func (c *metricsClient) UpdateReleaseResourceUtilization(ctx context.Context, repoSource, repoOwner, repoName string, releaseID string, jobResources JobResources) (err error) {
	defer func(begin time.Time) {
		api.UpdateMetrics(c.requestCount, c.requestLatency, "UpdateReleaseResourceUtilization", begin)
	}(time.Now())

	return c.Client.UpdateReleaseResourceUtilization(ctx, repoSource, repoOwner, repoName, releaseID, jobResources)
}

func (c *metricsClient) InsertBot(ctx context.Context, bot contracts.Bot, jobResources JobResources) (r *contracts.Bot, err error) {
	defer func(begin time.Time) {
		api.UpdateMetrics(c.requestCount, c.requestLatency, "InsertBot", begin)
	}(time.Now())

	return c.Client.InsertBot(ctx, bot, jobResources)
}

func (c *metricsClient) UpdateBotStatus(ctx context.Context, repoSource, repoOwner, repoName string, botID string, botStatus contracts.Status) (err error) {
	defer func(begin time.Time) {
		api.UpdateMetrics(c.requestCount, c.requestLatency, "UpdateBotStatus", begin)
	}(time.Now())

	return c.Client.UpdateBotStatus(ctx, repoSource, repoOwner, repoName, botID, botStatus)
}

func (c *metricsClient) UpdateBotResourceUtilization(ctx context.Context, repoSource, repoOwner, repoName string, botID string, jobResources JobResources) (err error) {
	defer func(begin time.Time) {
		api.UpdateMetrics(c.requestCount, c.requestLatency, "UpdateBotResourceUtilization", begin)
	}(time.Now())

	return c.Client.UpdateBotResourceUtilization(ctx, repoSource, repoOwner, repoName, botID, jobResources)
}

func (c *metricsClient) InsertBuildLog(ctx context.Context, buildLog contracts.BuildLog) (buildlog contracts.BuildLog, err error) {
	defer func(begin time.Time) {
		api.UpdateMetrics(c.requestCount, c.requestLatency, "InsertBuildLog", begin)
	}(time.Now())

	return c.Client.InsertBuildLog(ctx, buildLog)
}

func (c *metricsClient) InsertReleaseLog(ctx context.Context, releaseLog contracts.ReleaseLog) (releaselog contracts.ReleaseLog, err error) {
	defer func(begin time.Time) {
		api.UpdateMetrics(c.requestCount, c.requestLatency, "InsertReleaseLog", begin)
	}(time.Now())

	return c.Client.InsertReleaseLog(ctx, releaseLog)
}

func (c *metricsClient) InsertBotLog(ctx context.Context, botLog contracts.BotLog) (log contracts.BotLog, err error) {
	defer func(begin time.Time) {
		api.UpdateMetrics(c.requestCount, c.requestLatency, "InsertBotLog", begin)
	}(time.Now())

	return c.Client.InsertBotLog(ctx, botLog)
}

func (c *metricsClient) UpdateComputedTables(ctx context.Context, repoSource, repoOwner, repoName string) (err error) {
	defer func(begin time.Time) {
		api.UpdateMetrics(c.requestCount, c.requestLatency, "UpdateComputedTables", begin)
	}(time.Now())

	return c.Client.UpdateComputedTables(ctx, repoSource, repoOwner, repoName)
}

func (c *metricsClient) UpsertComputedPipeline(ctx context.Context, repoSource, repoOwner, repoName string) (err error) {
	defer func(begin time.Time) {
		api.UpdateMetrics(c.requestCount, c.requestLatency, "UpsertComputedPipeline", begin)
	}(time.Now())

	return c.Client.UpsertComputedPipeline(ctx, repoSource, repoOwner, repoName)
}

func (c *metricsClient) UpdateComputedPipelinePermissions(ctx context.Context, pipeline contracts.Pipeline) (err error) {
	defer func(begin time.Time) {
		api.UpdateMetrics(c.requestCount, c.requestLatency, "UpdateComputedPipelinePermissions", begin)
	}(time.Now())

	return c.Client.UpdateComputedPipelinePermissions(ctx, pipeline)
}

func (c *metricsClient) UpdateComputedPipelineFirstInsertedAt(ctx context.Context, repoSource, repoOwner, repoName string) (err error) {
	defer func(begin time.Time) {
		api.UpdateMetrics(c.requestCount, c.requestLatency, "UpdateComputedPipelineFirstInsertedAt", begin)
	}(time.Now())

	return c.Client.UpdateComputedPipelineFirstInsertedAt(ctx, repoSource, repoOwner, repoName)
}

func (c *metricsClient) UpsertComputedRelease(ctx context.Context, repoSource, repoOwner, repoName, releaseName, releaseAction string) (err error) {
	defer func(begin time.Time) {
		api.UpdateMetrics(c.requestCount, c.requestLatency, "UpsertComputedRelease", begin)
	}(time.Now())

	return c.Client.UpsertComputedRelease(ctx, repoSource, repoOwner, repoName, releaseName, releaseAction)
}

func (c *metricsClient) UpdateComputedReleaseFirstInsertedAt(ctx context.Context, repoSource, repoOwner, repoName, releaseName, releaseAction string) (err error) {
	defer func(begin time.Time) {
		api.UpdateMetrics(c.requestCount, c.requestLatency, "UpdateComputedReleaseFirstInsertedAt", begin)
	}(time.Now())

	return c.Client.UpdateComputedReleaseFirstInsertedAt(ctx, repoSource, repoOwner, repoName, releaseName, releaseAction)
}

func (c *metricsClient) ArchiveComputedPipeline(ctx context.Context, repoSource, repoOwner, repoName string) (err error) {
	defer func(begin time.Time) {
		api.UpdateMetrics(c.requestCount, c.requestLatency, "ArchiveComputedPipeline", begin)
	}(time.Now())

	return c.Client.ArchiveComputedPipeline(ctx, repoSource, repoOwner, repoName)
}

func (c *metricsClient) UnarchiveComputedPipeline(ctx context.Context, repoSource, repoOwner, repoName string) (err error) {
	defer func(begin time.Time) {
		api.UpdateMetrics(c.requestCount, c.requestLatency, "UnarchiveComputedPipeline", begin)
	}(time.Now())

	return c.Client.UnarchiveComputedPipeline(ctx, repoSource, repoOwner, repoName)
}

func (c *metricsClient) GetPipelines(ctx context.Context, pageNumber, pageSize int, filters map[api.FilterType][]string, sortings []api.OrderField, optimized bool) (pipelines []*contracts.Pipeline, err error) {
	defer func(begin time.Time) { api.UpdateMetrics(c.requestCount, c.requestLatency, "GetPipelines", begin) }(time.Now())

	return c.Client.GetPipelines(ctx, pageNumber, pageSize, filters, sortings, optimized)
}

func (c *metricsClient) GetPipelinesByRepoName(ctx context.Context, repoName string, optimized bool) (pipelines []*contracts.Pipeline, err error) {
	defer func(begin time.Time) {
		api.UpdateMetrics(c.requestCount, c.requestLatency, "GetPipelinesByRepoName", begin)
	}(time.Now())

	return c.Client.GetPipelinesByRepoName(ctx, repoName, optimized)
}

func (c *metricsClient) GetPipelinesCount(ctx context.Context, filters map[api.FilterType][]string) (count int, err error) {
	defer func(begin time.Time) {
		api.UpdateMetrics(c.requestCount, c.requestLatency, "GetPipelinesCount", begin)
	}(time.Now())

	return c.Client.GetPipelinesCount(ctx, filters)
}

func (c *metricsClient) GetPipeline(ctx context.Context, repoSource, repoOwner, repoName string, filters map[api.FilterType][]string, optimized bool) (pipeline *contracts.Pipeline, err error) {
	defer func(begin time.Time) { api.UpdateMetrics(c.requestCount, c.requestLatency, "GetPipeline", begin) }(time.Now())

	return c.Client.GetPipeline(ctx, repoSource, repoOwner, repoName, filters, optimized)
}

func (c *metricsClient) GetPipelineRecentBuilds(ctx context.Context, repoSource, repoOwner, repoName string, optimized bool) (builds []*contracts.Build, err error) {
	defer func(begin time.Time) {
		api.UpdateMetrics(c.requestCount, c.requestLatency, "GetPipelineRecentBuilds", begin)
	}(time.Now())

	return c.Client.GetPipelineRecentBuilds(ctx, repoSource, repoOwner, repoName, optimized)
}

func (c *metricsClient) GetPipelineBuilds(ctx context.Context, repoSource, repoOwner, repoName string, pageNumber, pageSize int, filters map[api.FilterType][]string, sortings []api.OrderField, optimized bool) (builds []*contracts.Build, err error) {
	defer func(begin time.Time) {
		api.UpdateMetrics(c.requestCount, c.requestLatency, "GetPipelineBuilds", begin)
	}(time.Now())

	return c.Client.GetPipelineBuilds(ctx, repoSource, repoOwner, repoName, pageNumber, pageSize, filters, sortings, optimized)
}

func (c *metricsClient) GetPipelineBuildsCount(ctx context.Context, repoSource, repoOwner, repoName string, filters map[api.FilterType][]string) (count int, err error) {
	defer func(begin time.Time) {
		api.UpdateMetrics(c.requestCount, c.requestLatency, "GetPipelineBuildsCount", begin)
	}(time.Now())

	return c.Client.GetPipelineBuildsCount(ctx, repoSource, repoOwner, repoName, filters)
}

func (c *metricsClient) GetPipelineBuild(ctx context.Context, repoSource, repoOwner, repoName, repoRevision string, optimized bool) (build *contracts.Build, err error) {
	defer func(begin time.Time) {
		api.UpdateMetrics(c.requestCount, c.requestLatency, "GetPipelineBuild", begin)
	}(time.Now())

	return c.Client.GetPipelineBuild(ctx, repoSource, repoOwner, repoName, repoRevision, optimized)
}

func (c *metricsClient) GetPipelineBuildByID(ctx context.Context, repoSource, repoOwner, repoName string, buildID string, optimized bool) (build *contracts.Build, err error) {
	defer func(begin time.Time) {
		api.UpdateMetrics(c.requestCount, c.requestLatency, "GetPipelineBuildByID", begin)
	}(time.Now())

	return c.Client.GetPipelineBuildByID(ctx, repoSource, repoOwner, repoName, buildID, optimized)
}

func (c *metricsClient) GetLastPipelineBuild(ctx context.Context, repoSource, repoOwner, repoName string, optimized bool) (build *contracts.Build, err error) {
	defer func(begin time.Time) {
		api.UpdateMetrics(c.requestCount, c.requestLatency, "GetLastPipelineBuild", begin)
	}(time.Now())

	return c.Client.GetLastPipelineBuild(ctx, repoSource, repoOwner, repoName, optimized)
}

func (c *metricsClient) GetFirstPipelineBuild(ctx context.Context, repoSource, repoOwner, repoName string, optimized bool) (build *contracts.Build, err error) {
	defer func(begin time.Time) {
		api.UpdateMetrics(c.requestCount, c.requestLatency, "GetFirstPipelineBuild", begin)
	}(time.Now())

	return c.Client.GetFirstPipelineBuild(ctx, repoSource, repoOwner, repoName, optimized)
}

func (c *metricsClient) GetLastPipelineBuildForBranch(ctx context.Context, repoSource, repoOwner, repoName, branch string) (build *contracts.Build, err error) {
	defer func(begin time.Time) {
		api.UpdateMetrics(c.requestCount, c.requestLatency, "GetLastPipelineBuildForBranch", begin)
	}(time.Now())

	return c.Client.GetLastPipelineBuildForBranch(ctx, repoSource, repoOwner, repoName, branch)
}

func (c *metricsClient) GetLastPipelineReleases(ctx context.Context, repoSource, repoOwner, repoName, releaseName, releaseAction string, pageSize int) (releases []*contracts.Release, err error) {
	defer func(begin time.Time) {
		api.UpdateMetrics(c.requestCount, c.requestLatency, "GetLastPipelineReleases", begin)
	}(time.Now())

	return c.Client.GetLastPipelineReleases(ctx, repoSource, repoOwner, repoName, releaseName, releaseAction, pageSize)
}

func (c *metricsClient) GetFirstPipelineRelease(ctx context.Context, repoSource, repoOwner, repoName, releaseName, releaseAction string) (release *contracts.Release, err error) {
	defer func(begin time.Time) {
		api.UpdateMetrics(c.requestCount, c.requestLatency, "GetFirstPipelineRelease", begin)
	}(time.Now())

	return c.Client.GetFirstPipelineRelease(ctx, repoSource, repoOwner, repoName, releaseName, releaseAction)
}

func (c *metricsClient) GetPipelineBuildsByVersion(ctx context.Context, repoSource, repoOwner, repoName, buildVersion string, statuses []contracts.Status, limit uint64, optimized bool) (builds []*contracts.Build, err error) {
	defer func(begin time.Time) {
		api.UpdateMetrics(c.requestCount, c.requestLatency, "GetPipelineBuildsByVersion", begin)
	}(time.Now())

	return c.Client.GetPipelineBuildsByVersion(ctx, repoSource, repoOwner, repoName, buildVersion, statuses, limit, optimized)
}

func (c *metricsClient) GetPipelineBuildLogs(ctx context.Context, repoSource, repoOwner, repoName, repoBranch, repoRevision, buildID string) (buildlog *contracts.BuildLog, err error) {
	defer func(begin time.Time) {
		api.UpdateMetrics(c.requestCount, c.requestLatency, "GetPipelineBuildLogs", begin)
	}(time.Now())

	return c.Client.GetPipelineBuildLogs(ctx, repoSource, repoOwner, repoName, repoBranch, repoRevision, buildID)
}

func (c *metricsClient) GetPipelineBuildLogsByID(ctx context.Context, repoSource, repoOwner, repoName, buildID, id string) (buildlog *contracts.BuildLog, err error) {
	defer func(begin time.Time) {
		api.UpdateMetrics(c.requestCount, c.requestLatency, "GetPipelineBuildLogsByID", begin)
	}(time.Now())

	return c.Client.GetPipelineBuildLogsByID(ctx, repoSource, repoOwner, repoName, buildID, id)
}

func (c *metricsClient) GetPipelineBuildLogsPerPage(ctx context.Context, repoSource, repoOwner, repoName, repoBranch, repoRevision, buildID string, pageNumber int, pageSize int) (buildLogs []*contracts.BuildLog, err error) {
	defer func(begin time.Time) {
		api.UpdateMetrics(c.requestCount, c.requestLatency, "GetPipelineBuildLogsPerPage", begin)
	}(time.Now())

	return c.Client.GetPipelineBuildLogsPerPage(ctx, repoSource, repoOwner, repoName, repoBranch, repoRevision, buildID, pageNumber, pageSize)
}

func (c *metricsClient) GetPipelineBuildLogsCount(ctx context.Context, repoSource, repoOwner, repoName, repoBranch, repoRevision, buildID string) (count int, err error) {
	defer func(begin time.Time) {
		api.UpdateMetrics(c.requestCount, c.requestLatency, "GetPipelineBuildLogsCount", begin)
	}(time.Now())

	return c.Client.GetPipelineBuildLogsCount(ctx, repoSource, repoOwner, repoName, repoBranch, repoRevision, buildID)
}

func (c *metricsClient) GetPipelineBuildMaxResourceUtilization(ctx context.Context, repoSource, repoOwner, repoName string, lastNRecords int) (jobresources JobResources, count int, err error) {
	defer func(begin time.Time) {
		api.UpdateMetrics(c.requestCount, c.requestLatency, "GetPipelineBuildMaxResourceUtilization", begin)
	}(time.Now())

	return c.Client.GetPipelineBuildMaxResourceUtilization(ctx, repoSource, repoOwner, repoName, lastNRecords)
}

func (c *metricsClient) GetPipelineReleases(ctx context.Context, repoSource, repoOwner, repoName string, pageNumber, pageSize int, filters map[api.FilterType][]string, sortings []api.OrderField) (releases []*contracts.Release, err error) {
	defer func(begin time.Time) {
		api.UpdateMetrics(c.requestCount, c.requestLatency, "GetPipelineReleases", begin)
	}(time.Now())

	return c.Client.GetPipelineReleases(ctx, repoSource, repoOwner, repoName, pageNumber, pageSize, filters, sortings)
}

func (c *metricsClient) GetPipelineReleasesCount(ctx context.Context, repoSource, repoOwner, repoName string, filters map[api.FilterType][]string) (count int, err error) {
	defer func(begin time.Time) {
		api.UpdateMetrics(c.requestCount, c.requestLatency, "GetPipelineReleasesCount", begin)
	}(time.Now())

	return c.Client.GetPipelineReleasesCount(ctx, repoSource, repoOwner, repoName, filters)
}

func (c *metricsClient) GetPipelineRelease(ctx context.Context, repoSource, repoOwner, repoName string, releaseID string) (release *contracts.Release, err error) {
	defer func(begin time.Time) {
		api.UpdateMetrics(c.requestCount, c.requestLatency, "GetPipelineRelease", begin)
	}(time.Now())

	return c.Client.GetPipelineRelease(ctx, repoSource, repoOwner, repoName, releaseID)
}

func (c *metricsClient) GetPipelineLastReleasesByName(ctx context.Context, repoSource, repoOwner, repoName, releaseName string, actions []string) (releases []contracts.Release, err error) {
	defer func(begin time.Time) {
		api.UpdateMetrics(c.requestCount, c.requestLatency, "GetPipelineLastReleasesByName", begin)
	}(time.Now())

	return c.Client.GetPipelineLastReleasesByName(ctx, repoSource, repoOwner, repoName, releaseName, actions)
}

func (c *metricsClient) GetPipelineReleaseLogs(ctx context.Context, repoSource, repoOwner, repoName string, releaseID string) (releaselog *contracts.ReleaseLog, err error) {
	defer func(begin time.Time) {
		api.UpdateMetrics(c.requestCount, c.requestLatency, "GetPipelineReleaseLogs", begin)
	}(time.Now())

	return c.Client.GetPipelineReleaseLogs(ctx, repoSource, repoOwner, repoName, releaseID)
}

func (c *metricsClient) GetPipelineReleaseLogsByID(ctx context.Context, repoSource, repoOwner, repoName string, releaseID string, id string) (releaselog *contracts.ReleaseLog, err error) {
	defer func(begin time.Time) {
		api.UpdateMetrics(c.requestCount, c.requestLatency, "GetPipelineReleaseLogsByID", begin)
	}(time.Now())

	return c.Client.GetPipelineReleaseLogsByID(ctx, repoSource, repoOwner, repoName, releaseID, id)
}

func (c *metricsClient) GetPipelineReleaseLogsPerPage(ctx context.Context, repoSource, repoOwner, repoName string, releaseID string, pageNumber int, pageSize int) (releaselogs []*contracts.ReleaseLog, err error) {
	defer func(begin time.Time) {
		api.UpdateMetrics(c.requestCount, c.requestLatency, "GetPipelineReleaseLogsPerPage", begin)
	}(time.Now())

	return c.Client.GetPipelineReleaseLogsPerPage(ctx, repoSource, repoOwner, repoName, releaseID, pageNumber, pageSize)
}

func (c *metricsClient) GetPipelineReleaseLogsCount(ctx context.Context, repoSource, repoOwner, repoName string, releaseID string) (count int, err error) {
	defer func(begin time.Time) {
		api.UpdateMetrics(c.requestCount, c.requestLatency, "GetPipelineReleaseLogsCount", begin)
	}(time.Now())

	return c.Client.GetPipelineReleaseLogsCount(ctx, repoSource, repoOwner, repoName, releaseID)
}

func (c *metricsClient) GetPipelineReleaseMaxResourceUtilization(ctx context.Context, repoSource, repoOwner, repoName, targetName string, lastNRecords int) (jobresources JobResources, count int, err error) {
	defer func(begin time.Time) {
		api.UpdateMetrics(c.requestCount, c.requestLatency, "GetPipelineReleaseMaxResourceUtilization", begin)
	}(time.Now())

	return c.Client.GetPipelineReleaseMaxResourceUtilization(ctx, repoSource, repoOwner, repoName, targetName, lastNRecords)
}

func (c *metricsClient) GetPipelineBots(ctx context.Context, repoSource, repoOwner, repoName string, pageNumber, pageSize int, filters map[api.FilterType][]string, sortings []api.OrderField) (bots []*contracts.Bot, err error) {
	defer func(begin time.Time) {
		api.UpdateMetrics(c.requestCount, c.requestLatency, "GetPipelineBots", begin)
	}(time.Now())

	return c.Client.GetPipelineBots(ctx, repoSource, repoOwner, repoName, pageNumber, pageSize, filters, sortings)
}

func (c *metricsClient) GetPipelineBotsCount(ctx context.Context, repoSource, repoOwner, repoName string, filters map[api.FilterType][]string) (count int, err error) {
	defer func(begin time.Time) {
		api.UpdateMetrics(c.requestCount, c.requestLatency, "GetPipelineBotsCount", begin)
	}(time.Now())

	return c.Client.GetPipelineBotsCount(ctx, repoSource, repoOwner, repoName, filters)
}

func (c *metricsClient) GetPipelineBot(ctx context.Context, repoSource, repoOwner, repoName string, botID string) (bot *contracts.Bot, err error) {
	defer func(begin time.Time) {
		api.UpdateMetrics(c.requestCount, c.requestLatency, "GetPipelineBot", begin)
	}(time.Now())

	return c.Client.GetPipelineBot(ctx, repoSource, repoOwner, repoName, botID)
}

func (c *metricsClient) GetPipelineBotLogs(ctx context.Context, repoSource, repoOwner, repoName string, botID string) (releaselog *contracts.BotLog, err error) {
	defer func(begin time.Time) {
		api.UpdateMetrics(c.requestCount, c.requestLatency, "GetPipelineBotLogs", begin)
	}(time.Now())

	return c.Client.GetPipelineBotLogs(ctx, repoSource, repoOwner, repoName, botID)
}

func (c *metricsClient) GetPipelineBotLogsByID(ctx context.Context, repoSource, repoOwner, repoName string, botID string, id string) (releaselog *contracts.BotLog, err error) {
	defer func(begin time.Time) {
		api.UpdateMetrics(c.requestCount, c.requestLatency, "GetPipelineBotLogsByID", begin)
	}(time.Now())

	return c.Client.GetPipelineBotLogsByID(ctx, repoSource, repoOwner, repoName, botID, id)
}

func (c *metricsClient) GetPipelineBotLogsPerPage(ctx context.Context, repoSource, repoOwner, repoName string, botID string, pageNumber int, pageSize int) (releaselogs []*contracts.BotLog, err error) {
	defer func(begin time.Time) {
		api.UpdateMetrics(c.requestCount, c.requestLatency, "GetPipelineBotLogsPerPage", begin)
	}(time.Now())

	return c.Client.GetPipelineBotLogsPerPage(ctx, repoSource, repoOwner, repoName, botID, pageNumber, pageSize)
}

func (c *metricsClient) GetPipelineBotLogsCount(ctx context.Context, repoSource, repoOwner, repoName string, botID string) (count int, err error) {
	defer func(begin time.Time) {
		api.UpdateMetrics(c.requestCount, c.requestLatency, "GetPipelineBotLogsCount", begin)
	}(time.Now())

	return c.Client.GetPipelineBotLogsCount(ctx, repoSource, repoOwner, repoName, botID)
}

func (c *metricsClient) GetPipelineBotMaxResourceUtilization(ctx context.Context, repoSource, repoOwner, repoName, targetName string, lastNRecords int) (jobresources JobResources, count int, err error) {
	defer func(begin time.Time) {
		api.UpdateMetrics(c.requestCount, c.requestLatency, "GetPipelineBotMaxResourceUtilization", begin)
	}(time.Now())

	return c.Client.GetPipelineBotMaxResourceUtilization(ctx, repoSource, repoOwner, repoName, targetName, lastNRecords)
}

func (c *metricsClient) GetBuildsCount(ctx context.Context, filters map[api.FilterType][]string) (count int, err error) {
	defer func(begin time.Time) {
		api.UpdateMetrics(c.requestCount, c.requestLatency, "GetBuildsCount", begin)
	}(time.Now())

	return c.Client.GetBuildsCount(ctx, filters)
}

func (c *metricsClient) GetReleasesCount(ctx context.Context, filters map[api.FilterType][]string) (count int, err error) {
	defer func(begin time.Time) {
		api.UpdateMetrics(c.requestCount, c.requestLatency, "GetReleasesCount", begin)
	}(time.Now())

	return c.Client.GetReleasesCount(ctx, filters)
}

func (c *metricsClient) GetBotsCount(ctx context.Context, filters map[api.FilterType][]string) (count int, err error) {
	defer func(begin time.Time) {
		api.UpdateMetrics(c.requestCount, c.requestLatency, "GetBotsCount", begin)
	}(time.Now())

	return c.Client.GetBotsCount(ctx, filters)
}

func (c *metricsClient) GetBuildsDuration(ctx context.Context, filters map[api.FilterType][]string) (duration time.Duration, err error) {
	defer func(begin time.Time) {
		api.UpdateMetrics(c.requestCount, c.requestLatency, "GetBuildsDuration", begin)
	}(time.Now())

	return c.Client.GetBuildsDuration(ctx, filters)
}

func (c *metricsClient) GetFirstBuildTimes(ctx context.Context) (times []time.Time, err error) {
	defer func(begin time.Time) {
		api.UpdateMetrics(c.requestCount, c.requestLatency, "GetFirstBuildTimes", begin)
	}(time.Now())

	return c.Client.GetFirstBuildTimes(ctx)
}

func (c *metricsClient) GetFirstReleaseTimes(ctx context.Context) (times []time.Time, err error) {
	defer func(begin time.Time) {
		api.UpdateMetrics(c.requestCount, c.requestLatency, "GetFirstReleaseTimes", begin)
	}(time.Now())

	return c.Client.GetFirstReleaseTimes(ctx)
}

func (c *metricsClient) GetFirstBotTimes(ctx context.Context) (times []time.Time, err error) {
	defer func(begin time.Time) {
		api.UpdateMetrics(c.requestCount, c.requestLatency, "GetFirstBotTimes", begin)
	}(time.Now())

	return c.Client.GetFirstBotTimes(ctx)
}

func (c *metricsClient) GetPipelineBuildsDurations(ctx context.Context, repoSource, repoOwner, repoName string, filters map[api.FilterType][]string) (durations []map[string]interface{}, err error) {
	defer func(begin time.Time) {
		api.UpdateMetrics(c.requestCount, c.requestLatency, "GetPipelineBuildsDurations", begin)
	}(time.Now())

	return c.Client.GetPipelineBuildsDurations(ctx, repoSource, repoOwner, repoName, filters)
}

func (c *metricsClient) GetPipelineReleasesDurations(ctx context.Context, repoSource, repoOwner, repoName string, filters map[api.FilterType][]string) (durations []map[string]interface{}, err error) {
	defer func(begin time.Time) {
		api.UpdateMetrics(c.requestCount, c.requestLatency, "GetPipelineReleasesDurations", begin)
	}(time.Now())

	return c.Client.GetPipelineReleasesDurations(ctx, repoSource, repoOwner, repoName, filters)
}

func (c *metricsClient) GetPipelineBotsDurations(ctx context.Context, repoSource, repoOwner, repoName string, filters map[api.FilterType][]string) (durations []map[string]interface{}, err error) {
	defer func(begin time.Time) {
		api.UpdateMetrics(c.requestCount, c.requestLatency, "GetPipelineBotsDurations", begin)
	}(time.Now())

	return c.Client.GetPipelineBotsDurations(ctx, repoSource, repoOwner, repoName, filters)
}

func (c *metricsClient) GetPipelineBuildsCPUUsageMeasurements(ctx context.Context, repoSource, repoOwner, repoName string, filters map[api.FilterType][]string) (measurements []map[string]interface{}, err error) {
	defer func(begin time.Time) {
		api.UpdateMetrics(c.requestCount, c.requestLatency, "GetPipelineBuildsCPUUsageMeasurements", begin)
	}(time.Now())

	return c.Client.GetPipelineBuildsCPUUsageMeasurements(ctx, repoSource, repoOwner, repoName, filters)
}

func (c *metricsClient) GetPipelineReleasesCPUUsageMeasurements(ctx context.Context, repoSource, repoOwner, repoName string, filters map[api.FilterType][]string) (measurements []map[string]interface{}, err error) {
	defer func(begin time.Time) {
		api.UpdateMetrics(c.requestCount, c.requestLatency, "GetPipelineReleasesCPUUsageMeasurements", begin)
	}(time.Now())

	return c.Client.GetPipelineReleasesCPUUsageMeasurements(ctx, repoSource, repoOwner, repoName, filters)
}

func (c *metricsClient) GetPipelineBotsCPUUsageMeasurements(ctx context.Context, repoSource, repoOwner, repoName string, filters map[api.FilterType][]string) (measurements []map[string]interface{}, err error) {
	defer func(begin time.Time) {
		api.UpdateMetrics(c.requestCount, c.requestLatency, "GetPipelineBotsCPUUsageMeasurements", begin)
	}(time.Now())

	return c.Client.GetPipelineBotsCPUUsageMeasurements(ctx, repoSource, repoOwner, repoName, filters)
}

func (c *metricsClient) GetPipelineBuildsMemoryUsageMeasurements(ctx context.Context, repoSource, repoOwner, repoName string, filters map[api.FilterType][]string) (measurements []map[string]interface{}, err error) {
	defer func(begin time.Time) {
		api.UpdateMetrics(c.requestCount, c.requestLatency, "GetPipelineBuildsMemoryUsageMeasurements", begin)
	}(time.Now())

	return c.Client.GetPipelineBuildsMemoryUsageMeasurements(ctx, repoSource, repoOwner, repoName, filters)
}

func (c *metricsClient) GetPipelineReleasesMemoryUsageMeasurements(ctx context.Context, repoSource, repoOwner, repoName string, filters map[api.FilterType][]string) (measurements []map[string]interface{}, err error) {
	defer func(begin time.Time) {
		api.UpdateMetrics(c.requestCount, c.requestLatency, "GetPipelineReleasesMemoryUsageMeasurements", begin)
	}(time.Now())

	return c.Client.GetPipelineReleasesMemoryUsageMeasurements(ctx, repoSource, repoOwner, repoName, filters)
}

func (c *metricsClient) GetPipelineBotsMemoryUsageMeasurements(ctx context.Context, repoSource, repoOwner, repoName string, filters map[api.FilterType][]string) (measurements []map[string]interface{}, err error) {
	defer func(begin time.Time) {
		api.UpdateMetrics(c.requestCount, c.requestLatency, "GetPipelineBotsMemoryUsageMeasurements", begin)
	}(time.Now())

	return c.Client.GetPipelineBotsMemoryUsageMeasurements(ctx, repoSource, repoOwner, repoName, filters)
}

func (c *metricsClient) GetLabelValues(ctx context.Context, labelKey string) (labels []map[string]interface{}, err error) {
	defer func(begin time.Time) {
		api.UpdateMetrics(c.requestCount, c.requestLatency, "GetLabelValues", begin)
	}(time.Now())

	return c.Client.GetLabelValues(ctx, labelKey)
}

func (c *metricsClient) GetFrequentLabels(ctx context.Context, pageNumber, pageSize int, filters map[api.FilterType][]string) (measurements []map[string]interface{}, err error) {
	defer func(begin time.Time) {
		api.UpdateMetrics(c.requestCount, c.requestLatency, "GetFrequentLabels", begin)
	}(time.Now())

	return c.Client.GetFrequentLabels(ctx, pageNumber, pageSize, filters)
}

func (c *metricsClient) GetFrequentLabelsCount(ctx context.Context, filters map[api.FilterType][]string) (count int, err error) {
	defer func(begin time.Time) {
		api.UpdateMetrics(c.requestCount, c.requestLatency, "GetFrequentLabelsCount", begin)
	}(time.Now())

	return c.Client.GetFrequentLabelsCount(ctx, filters)
}

func (c *metricsClient) GetPipelinesWithMostBuilds(ctx context.Context, pageNumber, pageSize int, filters map[api.FilterType][]string) (pipelines []map[string]interface{}, err error) {
	defer func(begin time.Time) {
		api.UpdateMetrics(c.requestCount, c.requestLatency, "GetPipelinesWithMostBuilds", begin)
	}(time.Now())

	return c.Client.GetPipelinesWithMostBuilds(ctx, pageNumber, pageSize, filters)
}

func (c *metricsClient) GetPipelinesWithMostBuildsCount(ctx context.Context, filters map[api.FilterType][]string) (count int, err error) {
	defer func(begin time.Time) {
		api.UpdateMetrics(c.requestCount, c.requestLatency, "GetPipelinesWithMostBuildsCount", begin)
	}(time.Now())

	return c.Client.GetPipelinesWithMostBuildsCount(ctx, filters)
}

func (c *metricsClient) GetPipelinesWithMostReleases(ctx context.Context, pageNumber, pageSize int, filters map[api.FilterType][]string) (pipelines []map[string]interface{}, err error) {
	defer func(begin time.Time) {
		api.UpdateMetrics(c.requestCount, c.requestLatency, "GetPipelinesWithMostReleases", begin)
	}(time.Now())

	return c.Client.GetPipelinesWithMostReleases(ctx, pageNumber, pageSize, filters)
}

func (c *metricsClient) GetPipelinesWithMostReleasesCount(ctx context.Context, filters map[api.FilterType][]string) (count int, err error) {
	defer func(begin time.Time) {
		api.UpdateMetrics(c.requestCount, c.requestLatency, "GetPipelinesWithMostReleasesCount", begin)
	}(time.Now())

	return c.Client.GetPipelinesWithMostReleasesCount(ctx, filters)
}

func (c *metricsClient) GetPipelinesWithMostBots(ctx context.Context, pageNumber, pageSize int, filters map[api.FilterType][]string) (pipelines []map[string]interface{}, err error) {
	defer func(begin time.Time) {
		api.UpdateMetrics(c.requestCount, c.requestLatency, "GetPipelinesWithMostBots", begin)
	}(time.Now())

	return c.Client.GetPipelinesWithMostBots(ctx, pageNumber, pageSize, filters)
}

func (c *metricsClient) GetPipelinesWithMostBotsCount(ctx context.Context, filters map[api.FilterType][]string) (count int, err error) {
	defer func(begin time.Time) {
		api.UpdateMetrics(c.requestCount, c.requestLatency, "GetPipelinesWithMostBotsCount", begin)
	}(time.Now())

	return c.Client.GetPipelinesWithMostBotsCount(ctx, filters)
}

func (c *metricsClient) GetTriggers(ctx context.Context, triggerType, identifier, event string) (pipelines []*contracts.Pipeline, err error) {
	defer func(begin time.Time) { api.UpdateMetrics(c.requestCount, c.requestLatency, "GetTriggers", begin) }(time.Now())

	return c.Client.GetTriggers(ctx, triggerType, identifier, event)
}

func (c *metricsClient) GetGitTriggers(ctx context.Context, gitEvent manifest.EstafetteGitEvent) (pipelines []*contracts.Pipeline, err error) {
	defer func(begin time.Time) {
		api.UpdateMetrics(c.requestCount, c.requestLatency, "GetGitTriggers", begin)
	}(time.Now())

	return c.Client.GetGitTriggers(ctx, gitEvent)
}

func (c *metricsClient) GetPipelineTriggers(ctx context.Context, build contracts.Build, event string) (pipelines []*contracts.Pipeline, err error) {
	defer func(begin time.Time) {
		api.UpdateMetrics(c.requestCount, c.requestLatency, "GetPipelineTriggers", begin)
	}(time.Now())

	return c.Client.GetPipelineTriggers(ctx, build, event)
}

func (c *metricsClient) GetReleaseTriggers(ctx context.Context, release contracts.Release, event string) (pipelines []*contracts.Pipeline, err error) {
	defer func(begin time.Time) {
		api.UpdateMetrics(c.requestCount, c.requestLatency, "GetReleaseTriggers", begin)
	}(time.Now())

	return c.Client.GetReleaseTriggers(ctx, release, event)
}

func (c *metricsClient) GetPubSubTriggers(ctx context.Context) (pipelines []*contracts.Pipeline, err error) {
	defer func(begin time.Time) {
		api.UpdateMetrics(c.requestCount, c.requestLatency, "GetPubSubTriggers", begin)
	}(time.Now())

	return c.Client.GetPubSubTriggers(ctx)
}

func (c *metricsClient) GetCronTriggers(ctx context.Context) (pipelines []*contracts.Pipeline, err error) {
	defer func(begin time.Time) {
		api.UpdateMetrics(c.requestCount, c.requestLatency, "GetCronTriggers", begin)
	}(time.Now())

	return c.Client.GetCronTriggers(ctx)
}

func (c *metricsClient) GetGithubTriggers(ctx context.Context, githubEvent manifest.EstafetteGithubEvent) (pipelines []*contracts.Pipeline, err error) {
	defer func(begin time.Time) {
		api.UpdateMetrics(c.requestCount, c.requestLatency, "GetGithubTriggers", begin)
	}(time.Now())

	return c.Client.GetGithubTriggers(ctx, githubEvent)
}

func (c *metricsClient) GetBitbucketTriggers(ctx context.Context, bitbucketEvent manifest.EstafetteBitbucketEvent) (pipelines []*contracts.Pipeline, err error) {
	defer func(begin time.Time) {
		api.UpdateMetrics(c.requestCount, c.requestLatency, "GetBitbucketTriggers", begin)
	}(time.Now())

	return c.Client.GetBitbucketTriggers(ctx, bitbucketEvent)
}

func (c *metricsClient) Rename(ctx context.Context, shortFromRepoSource, fromRepoSource, fromRepoOwner, fromRepoName, shortToRepoSource, toRepoSource, toRepoOwner, toRepoName string) (err error) {
	defer func(begin time.Time) { api.UpdateMetrics(c.requestCount, c.requestLatency, "Rename", begin) }(time.Now())

	return c.Client.Rename(ctx, shortFromRepoSource, fromRepoSource, fromRepoOwner, fromRepoName, shortToRepoSource, toRepoSource, toRepoOwner, toRepoName)
}

func (c *metricsClient) RenameBuildVersion(ctx context.Context, shortFromRepoSource, fromRepoOwner, fromRepoName, shortToRepoSource, toRepoOwner, toRepoName string) (err error) {
	defer func(begin time.Time) {
		api.UpdateMetrics(c.requestCount, c.requestLatency, "RenameBuildVersion", begin)
	}(time.Now())

	return c.Client.RenameBuildVersion(ctx, shortFromRepoSource, fromRepoOwner, fromRepoName, shortToRepoSource, toRepoOwner, toRepoName)
}

func (c *metricsClient) RenameBuilds(ctx context.Context, fromRepoSource, fromRepoOwner, fromRepoName, toRepoSource, toRepoOwner, toRepoName string) (err error) {
	defer func(begin time.Time) { api.UpdateMetrics(c.requestCount, c.requestLatency, "RenameBuilds", begin) }(time.Now())

	return c.Client.RenameBuilds(ctx, fromRepoSource, fromRepoOwner, fromRepoName, toRepoSource, toRepoOwner, toRepoName)
}

func (c *metricsClient) RenameBuildLogs(ctx context.Context, fromRepoSource, fromRepoOwner, fromRepoName, toRepoSource, toRepoOwner, toRepoName string) (err error) {
	defer func(begin time.Time) {
		api.UpdateMetrics(c.requestCount, c.requestLatency, "RenameBuildLogs", begin)
	}(time.Now())

	return c.Client.RenameBuildLogs(ctx, fromRepoSource, fromRepoOwner, fromRepoName, toRepoSource, toRepoOwner, toRepoName)
}

func (c *metricsClient) RenameReleases(ctx context.Context, fromRepoSource, fromRepoOwner, fromRepoName, toRepoSource, toRepoOwner, toRepoName string) (err error) {
	defer func(begin time.Time) {
		api.UpdateMetrics(c.requestCount, c.requestLatency, "RenameReleases", begin)
	}(time.Now())

	return c.Client.RenameReleases(ctx, fromRepoSource, fromRepoOwner, fromRepoName, toRepoSource, toRepoOwner, toRepoName)
}

func (c *metricsClient) RenameReleaseLogs(ctx context.Context, fromRepoSource, fromRepoOwner, fromRepoName, toRepoSource, toRepoOwner, toRepoName string) (err error) {
	defer func(begin time.Time) {
		api.UpdateMetrics(c.requestCount, c.requestLatency, "RenameReleaseLogs", begin)
	}(time.Now())

	return c.Client.RenameReleaseLogs(ctx, fromRepoSource, fromRepoOwner, fromRepoName, toRepoSource, toRepoOwner, toRepoName)
}

func (c *metricsClient) RenameComputedPipelines(ctx context.Context, fromRepoSource, fromRepoOwner, fromRepoName, toRepoSource, toRepoOwner, toRepoName string) (err error) {
	defer func(begin time.Time) {
		api.UpdateMetrics(c.requestCount, c.requestLatency, "RenameComputedPipelines", begin)
	}(time.Now())

	return c.Client.RenameComputedPipelines(ctx, fromRepoSource, fromRepoOwner, fromRepoName, toRepoSource, toRepoOwner, toRepoName)
}

func (c *metricsClient) RenameComputedReleases(ctx context.Context, fromRepoSource, fromRepoOwner, fromRepoName, toRepoSource, toRepoOwner, toRepoName string) (err error) {
	defer func(begin time.Time) {
		api.UpdateMetrics(c.requestCount, c.requestLatency, "RenameComputedReleases", begin)
	}(time.Now())

	return c.Client.RenameComputedReleases(ctx, fromRepoSource, fromRepoOwner, fromRepoName, toRepoSource, toRepoOwner, toRepoName)
}

func (c *metricsClient) InsertUser(ctx context.Context, user contracts.User) (u *contracts.User, err error) {
	defer func(begin time.Time) {
		api.UpdateMetrics(c.requestCount, c.requestLatency, "InsertUser", begin)
	}(time.Now())

	return c.Client.InsertUser(ctx, user)
}

func (c *metricsClient) UpdateUser(ctx context.Context, user contracts.User) (err error) {
	defer func(begin time.Time) {
		api.UpdateMetrics(c.requestCount, c.requestLatency, "UpdateUser", begin)
	}(time.Now())

	return c.Client.UpdateUser(ctx, user)
}

func (c *metricsClient) DeleteUser(ctx context.Context, user contracts.User) (err error) {
	defer func(begin time.Time) {
		api.UpdateMetrics(c.requestCount, c.requestLatency, "DeleteUser", begin)
	}(time.Now())

	return c.Client.DeleteUser(ctx, user)
}

func (c *metricsClient) GetUserByIdentity(ctx context.Context, identity contracts.UserIdentity) (user *contracts.User, err error) {
	defer func(begin time.Time) {
		api.UpdateMetrics(c.requestCount, c.requestLatency, "GetUserByIdentity", begin)
	}(time.Now())

	return c.Client.GetUserByIdentity(ctx, identity)
}

func (c *metricsClient) GetUserByID(ctx context.Context, id string, filters map[api.FilterType][]string) (user *contracts.User, err error) {
	defer func(begin time.Time) {
		api.UpdateMetrics(c.requestCount, c.requestLatency, "GetUserByID", begin)
	}(time.Now())

	return c.Client.GetUserByID(ctx, id, filters)
}

func (c *metricsClient) GetUsers(ctx context.Context, pageNumber, pageSize int, filters map[api.FilterType][]string, sortings []api.OrderField) (users []*contracts.User, err error) {
	defer func(begin time.Time) {
		api.UpdateMetrics(c.requestCount, c.requestLatency, "GetUsers", begin)
	}(time.Now())

	return c.Client.GetUsers(ctx, pageNumber, pageSize, filters, sortings)
}

func (c *metricsClient) GetUsersCount(ctx context.Context, filters map[api.FilterType][]string) (count int, err error) {
	defer func(begin time.Time) {
		api.UpdateMetrics(c.requestCount, c.requestLatency, "GetUsersCount", begin)
	}(time.Now())

	return c.Client.GetUsersCount(ctx, filters)
}

func (c *metricsClient) InsertGroup(ctx context.Context, group contracts.Group) (g *contracts.Group, err error) {
	defer func(begin time.Time) {
		api.UpdateMetrics(c.requestCount, c.requestLatency, "InsertGroup", begin)
	}(time.Now())

	return c.Client.InsertGroup(ctx, group)
}

func (c *metricsClient) UpdateGroup(ctx context.Context, group contracts.Group) (err error) {
	defer func(begin time.Time) {
		api.UpdateMetrics(c.requestCount, c.requestLatency, "UpdateGroup", begin)
	}(time.Now())

	return c.Client.UpdateGroup(ctx, group)
}

func (c *metricsClient) DeleteGroup(ctx context.Context, group contracts.Group) (err error) {
	defer func(begin time.Time) {
		api.UpdateMetrics(c.requestCount, c.requestLatency, "DeleteGroup", begin)
	}(time.Now())

	return c.Client.DeleteGroup(ctx, group)
}

func (c *metricsClient) GetGroupByIdentity(ctx context.Context, identity contracts.GroupIdentity) (group *contracts.Group, err error) {
	defer func(begin time.Time) {
		api.UpdateMetrics(c.requestCount, c.requestLatency, "GetGroupByIdentity", begin)
	}(time.Now())

	return c.Client.GetGroupByIdentity(ctx, identity)
}

func (c *metricsClient) GetGroupByID(ctx context.Context, id string, filters map[api.FilterType][]string) (group *contracts.Group, err error) {
	defer func(begin time.Time) {
		api.UpdateMetrics(c.requestCount, c.requestLatency, "GetGroupByID", begin)
	}(time.Now())

	return c.Client.GetGroupByID(ctx, id, filters)
}

func (c *metricsClient) GetGroups(ctx context.Context, pageNumber, pageSize int, filters map[api.FilterType][]string, sortings []api.OrderField) (groups []*contracts.Group, err error) {
	defer func(begin time.Time) {
		api.UpdateMetrics(c.requestCount, c.requestLatency, "GetGroups", begin)
	}(time.Now())

	return c.Client.GetGroups(ctx, pageNumber, pageSize, filters, sortings)
}

func (c *metricsClient) GetGroupsCount(ctx context.Context, filters map[api.FilterType][]string) (count int, err error) {
	defer func(begin time.Time) {
		api.UpdateMetrics(c.requestCount, c.requestLatency, "GetGroupsCount", begin)
	}(time.Now())

	return c.Client.GetGroupsCount(ctx, filters)
}

func (c *metricsClient) InsertOrganization(ctx context.Context, organization contracts.Organization) (o *contracts.Organization, err error) {
	defer func(begin time.Time) {
		api.UpdateMetrics(c.requestCount, c.requestLatency, "InsertOrganization", begin)
	}(time.Now())

	return c.Client.InsertOrganization(ctx, organization)
}

func (c *metricsClient) UpdateOrganization(ctx context.Context, organization contracts.Organization) (err error) {
	defer func(begin time.Time) {
		api.UpdateMetrics(c.requestCount, c.requestLatency, "UpdateOrganization", begin)
	}(time.Now())

	return c.Client.UpdateOrganization(ctx, organization)
}

func (c *metricsClient) DeleteOrganization(ctx context.Context, organization contracts.Organization) (err error) {
	defer func(begin time.Time) {
		api.UpdateMetrics(c.requestCount, c.requestLatency, "DeleteOrganization", begin)
	}(time.Now())

	return c.Client.DeleteOrganization(ctx, organization)
}

func (c *metricsClient) GetOrganizationByIdentity(ctx context.Context, identity contracts.OrganizationIdentity) (organization *contracts.Organization, err error) {
	defer func(begin time.Time) {
		api.UpdateMetrics(c.requestCount, c.requestLatency, "GetOrganizationByIdentity", begin)
	}(time.Now())

	return c.Client.GetOrganizationByIdentity(ctx, identity)
}

func (c *metricsClient) GetOrganizationByID(ctx context.Context, id string) (organization *contracts.Organization, err error) {
	defer func(begin time.Time) {
		api.UpdateMetrics(c.requestCount, c.requestLatency, "GetOrganizationByID", begin)
	}(time.Now())

	return c.Client.GetOrganizationByID(ctx, id)
}

func (c *metricsClient) GetOrganizationByName(ctx context.Context, name string) (organization *contracts.Organization, err error) {
	defer func(begin time.Time) {
		api.UpdateMetrics(c.requestCount, c.requestLatency, "GetOrganizationByName", begin)
	}(time.Now())

	return c.Client.GetOrganizationByName(ctx, name)
}

func (c *metricsClient) GetOrganizations(ctx context.Context, pageNumber, pageSize int, filters map[api.FilterType][]string, sortings []api.OrderField) (organizations []*contracts.Organization, err error) {
	defer func(begin time.Time) {
		api.UpdateMetrics(c.requestCount, c.requestLatency, "GetOrganizations", begin)
	}(time.Now())

	return c.Client.GetOrganizations(ctx, pageNumber, pageSize, filters, sortings)
}

func (c *metricsClient) GetOrganizationsCount(ctx context.Context, filters map[api.FilterType][]string) (count int, err error) {
	defer func(begin time.Time) {
		api.UpdateMetrics(c.requestCount, c.requestLatency, "GetOrganizationsCount", begin)
	}(time.Now())

	return c.Client.GetOrganizationsCount(ctx, filters)
}

func (c *metricsClient) InsertClient(ctx context.Context, client contracts.Client) (cl *contracts.Client, err error) {
	defer func(begin time.Time) {
		api.UpdateMetrics(c.requestCount, c.requestLatency, "InsertClient", begin)
	}(time.Now())

	return c.Client.InsertClient(ctx, client)
}

func (c *metricsClient) UpdateClient(ctx context.Context, client contracts.Client) (err error) {
	defer func(begin time.Time) {
		api.UpdateMetrics(c.requestCount, c.requestLatency, "UpdateClient", begin)
	}(time.Now())

	return c.Client.UpdateClient(ctx, client)
}

func (c *metricsClient) DeleteClient(ctx context.Context, client contracts.Client) (err error) {
	defer func(begin time.Time) {
		api.UpdateMetrics(c.requestCount, c.requestLatency, "DeleteClient", begin)
	}(time.Now())

	return c.Client.DeleteClient(ctx, client)
}

func (c *metricsClient) GetClientByClientID(ctx context.Context, clientID string) (client *contracts.Client, err error) {
	defer func(begin time.Time) {
		api.UpdateMetrics(c.requestCount, c.requestLatency, "GetClientByClientID", begin)
	}(time.Now())

	return c.Client.GetClientByClientID(ctx, clientID)
}

func (c *metricsClient) GetClientByID(ctx context.Context, id string) (client *contracts.Client, err error) {
	defer func(begin time.Time) {
		api.UpdateMetrics(c.requestCount, c.requestLatency, "GetClientByID", begin)
	}(time.Now())

	return c.Client.GetClientByID(ctx, id)
}

func (c *metricsClient) GetClients(ctx context.Context, pageNumber, pageSize int, filters map[api.FilterType][]string, sortings []api.OrderField) (clients []*contracts.Client, err error) {
	defer func(begin time.Time) {
		api.UpdateMetrics(c.requestCount, c.requestLatency, "GetClients", begin)
	}(time.Now())

	return c.Client.GetClients(ctx, pageNumber, pageSize, filters, sortings)
}

func (c *metricsClient) GetClientsCount(ctx context.Context, filters map[api.FilterType][]string) (count int, err error) {
	defer func(begin time.Time) {
		api.UpdateMetrics(c.requestCount, c.requestLatency, "GetClientsCount", begin)
	}(time.Now())

	return c.Client.GetClientsCount(ctx, filters)
}

func (c *metricsClient) InsertCatalogEntity(ctx context.Context, catalogEntity contracts.CatalogEntity) (insertedCatalogEntity *contracts.CatalogEntity, err error) {
	defer func(begin time.Time) {
		api.UpdateMetrics(c.requestCount, c.requestLatency, "InsertCatalogEntity", begin)
	}(time.Now())

	return c.Client.InsertCatalogEntity(ctx, catalogEntity)
}

func (c *metricsClient) UpdateCatalogEntity(ctx context.Context, catalogEntity contracts.CatalogEntity) (err error) {
	defer func(begin time.Time) {
		api.UpdateMetrics(c.requestCount, c.requestLatency, "UpdateCatalogEntity", begin)
	}(time.Now())

	return c.Client.UpdateCatalogEntity(ctx, catalogEntity)
}

func (c *metricsClient) DeleteCatalogEntity(ctx context.Context, id string) (err error) {
	defer func(begin time.Time) {
		api.UpdateMetrics(c.requestCount, c.requestLatency, "DeleteCatalogEntity", begin)
	}(time.Now())

	return c.Client.DeleteCatalogEntity(ctx, id)
}

func (c *metricsClient) GetCatalogEntityByID(ctx context.Context, id string) (catalogEntity *contracts.CatalogEntity, err error) {
	defer func(begin time.Time) {
		api.UpdateMetrics(c.requestCount, c.requestLatency, "GetCatalogEntityByID", begin)
	}(time.Now())

	return c.Client.GetCatalogEntityByID(ctx, id)
}

func (c *metricsClient) GetCatalogEntities(ctx context.Context, pageNumber, pageSize int, filters map[api.FilterType][]string, sortings []api.OrderField) (catalogEntities []*contracts.CatalogEntity, err error) {
	defer func(begin time.Time) {
		api.UpdateMetrics(c.requestCount, c.requestLatency, "GetCatalogEntities", begin)
	}(time.Now())

	return c.Client.GetCatalogEntities(ctx, pageNumber, pageSize, filters, sortings)
}

func (c *metricsClient) GetCatalogEntitiesCount(ctx context.Context, filters map[api.FilterType][]string) (count int, err error) {
	defer func(begin time.Time) {
		api.UpdateMetrics(c.requestCount, c.requestLatency, "GetCatalogEntitiesCount", begin)
	}(time.Now())

	return c.Client.GetCatalogEntitiesCount(ctx, filters)
}

func (c *metricsClient) GetCatalogEntityParentKeys(ctx context.Context, pageNumber, pageSize int, filters map[api.FilterType][]string, sortings []api.OrderField) (keys []map[string]interface{}, err error) {
	defer func(begin time.Time) {
		api.UpdateMetrics(c.requestCount, c.requestLatency, "GetCatalogEntityParentKeys", begin)
	}(time.Now())

	return c.Client.GetCatalogEntityParentKeys(ctx, pageNumber, pageSize, filters, sortings)
}

func (c *metricsClient) GetCatalogEntityParentKeysCount(ctx context.Context, filters map[api.FilterType][]string) (count int, err error) {
	defer func(begin time.Time) {
		api.UpdateMetrics(c.requestCount, c.requestLatency, "GetCatalogEntityParentKeysCount", begin)
	}(time.Now())

	return c.Client.GetCatalogEntityParentKeysCount(ctx, filters)
}

func (c *metricsClient) GetCatalogEntityParentValues(ctx context.Context, pageNumber, pageSize int, filters map[api.FilterType][]string, sortings []api.OrderField) (values []map[string]interface{}, err error) {
	defer func(begin time.Time) {
		api.UpdateMetrics(c.requestCount, c.requestLatency, "GetCatalogEntityParentValues", begin)
	}(time.Now())

	return c.Client.GetCatalogEntityParentValues(ctx, pageNumber, pageSize, filters, sortings)
}

func (c *metricsClient) GetCatalogEntityParentValuesCount(ctx context.Context, filters map[api.FilterType][]string) (count int, err error) {
	defer func(begin time.Time) {
		api.UpdateMetrics(c.requestCount, c.requestLatency, "GetCatalogEntityParentValuesCount", begin)
	}(time.Now())

	return c.Client.GetCatalogEntityParentValuesCount(ctx, filters)
}

func (c *metricsClient) GetCatalogEntityKeys(ctx context.Context, pageNumber, pageSize int, filters map[api.FilterType][]string, sortings []api.OrderField) (keys []map[string]interface{}, err error) {
	defer func(begin time.Time) {
		api.UpdateMetrics(c.requestCount, c.requestLatency, "GetCatalogEntityKeys", begin)
	}(time.Now())

	return c.Client.GetCatalogEntityKeys(ctx, pageNumber, pageSize, filters, sortings)
}

func (c *metricsClient) GetCatalogEntityKeysCount(ctx context.Context, filters map[api.FilterType][]string) (count int, err error) {
	defer func(begin time.Time) {
		api.UpdateMetrics(c.requestCount, c.requestLatency, "GetCatalogEntityKeysCount", begin)
	}(time.Now())

	return c.Client.GetCatalogEntityKeysCount(ctx, filters)
}

func (c *metricsClient) GetCatalogEntityValues(ctx context.Context, pageNumber, pageSize int, filters map[api.FilterType][]string, sortings []api.OrderField) (values []map[string]interface{}, err error) {
	defer func(begin time.Time) {
		api.UpdateMetrics(c.requestCount, c.requestLatency, "GetCatalogEntityValues", begin)
	}(time.Now())

	return c.Client.GetCatalogEntityValues(ctx, pageNumber, pageSize, filters, sortings)
}

func (c *metricsClient) GetCatalogEntityValuesCount(ctx context.Context, filters map[api.FilterType][]string) (count int, err error) {
	defer func(begin time.Time) {
		api.UpdateMetrics(c.requestCount, c.requestLatency, "GetCatalogEntityValuesCount", begin)
	}(time.Now())

	return c.Client.GetCatalogEntityValuesCount(ctx, filters)
}

func (c *metricsClient) GetCatalogEntityLabels(ctx context.Context, pageNumber, pageSize int, filters map[api.FilterType][]string) (labels []map[string]interface{}, err error) {
	defer func(begin time.Time) {
		api.UpdateMetrics(c.requestCount, c.requestLatency, "GetCatalogEntityLabels", begin)
	}(time.Now())

	return c.Client.GetCatalogEntityLabels(ctx, pageNumber, pageSize, filters)
}

func (c *metricsClient) GetCatalogEntityLabelsCount(ctx context.Context, filters map[api.FilterType][]string) (count int, err error) {
	defer func(begin time.Time) {
		api.UpdateMetrics(c.requestCount, c.requestLatency, "GetCatalogEntityLabelsCount", begin)
	}(time.Now())

	return c.Client.GetCatalogEntityLabelsCount(ctx, filters)
}

func (c *metricsClient) GetAllPipelineBuilds(ctx context.Context, pageNumber, pageSize int, filters map[api.FilterType][]string, sortings []api.OrderField, optimized bool) (builds []*contracts.Build, err error) {
	defer func(begin time.Time) {
		api.UpdateMetrics(c.requestCount, c.requestLatency, "GetAllPipelineBuilds", begin)
	}(time.Now())

	return c.Client.GetAllPipelineBuilds(ctx, pageNumber, pageSize, filters, sortings, optimized)
}

func (c *metricsClient) GetAllPipelineBuildsCount(ctx context.Context, filters map[api.FilterType][]string) (count int, err error) {
	defer func(begin time.Time) {
		api.UpdateMetrics(c.requestCount, c.requestLatency, "GetAllPipelineBuildsCount", begin)
	}(time.Now())

	return c.Client.GetAllPipelineBuildsCount(ctx, filters)
}

func (c *metricsClient) GetAllPipelineReleases(ctx context.Context, pageNumber, pageSize int, filters map[api.FilterType][]string, sortings []api.OrderField) (releases []*contracts.Release, err error) {
	defer func(begin time.Time) {
		api.UpdateMetrics(c.requestCount, c.requestLatency, "GetAllPipelineReleases", begin)
	}(time.Now())

	return c.Client.GetAllPipelineReleases(ctx, pageNumber, pageSize, filters, sortings)
}

func (c *metricsClient) GetAllPipelineReleasesCount(ctx context.Context, filters map[api.FilterType][]string) (count int, err error) {
	defer func(begin time.Time) {
		api.UpdateMetrics(c.requestCount, c.requestLatency, "GetAllPipelineReleasesCount", begin)
	}(time.Now())

	return c.Client.GetAllPipelineReleasesCount(ctx, filters)
}

func (c *metricsClient) GetAllPipelineBots(ctx context.Context, pageNumber, pageSize int, filters map[api.FilterType][]string, sortings []api.OrderField) (bots []*contracts.Bot, err error) {
	defer func(begin time.Time) {
		api.UpdateMetrics(c.requestCount, c.requestLatency, "GetAllPipelineBots", begin)
	}(time.Now())

	return c.Client.GetAllPipelineBots(ctx, pageNumber, pageSize, filters, sortings)
}

func (c *metricsClient) GetAllPipelineBotsCount(ctx context.Context, filters map[api.FilterType][]string) (count int, err error) {
	defer func(begin time.Time) {
		api.UpdateMetrics(c.requestCount, c.requestLatency, "GetAllPipelineBotsCount", begin)
	}(time.Now())

	return c.Client.GetAllPipelineBotsCount(ctx, filters)
}

func (c *metricsClient) GetAllNotifications(ctx context.Context, pageNumber, pageSize int, filters map[api.FilterType][]string, sortings []api.OrderField) (notifications []*contracts.NotificationRecord, err error) {
	defer func(begin time.Time) {
		api.UpdateMetrics(c.requestCount, c.requestLatency, "GetAllNotifications", begin)
	}(time.Now())

	return c.Client.GetAllNotifications(ctx, pageNumber, pageSize, filters, sortings)
}

func (c *metricsClient) GetAllNotificationsCount(ctx context.Context, filters map[api.FilterType][]string) (count int, err error) {
	defer func(begin time.Time) {
		api.UpdateMetrics(c.requestCount, c.requestLatency, "GetAllNotificationsCount", begin)
	}(time.Now())

	return c.Client.GetAllNotificationsCount(ctx, filters)
}

func (c *metricsClient) InsertNotification(ctx context.Context, notificationRecord contracts.NotificationRecord) (n *contracts.NotificationRecord, err error) {
	defer func(begin time.Time) {
		api.UpdateMetrics(c.requestCount, c.requestLatency, "InsertNotification", begin)
	}(time.Now())

	return c.Client.InsertNotification(ctx, notificationRecord)
}

func (c *metricsClient) GetReleaseTargets(ctx context.Context, pageNumber, pageSize int, filters map[api.FilterType][]string) (releaseTargets []map[string]interface{}, err error) {
	defer func(begin time.Time) {
		api.UpdateMetrics(c.requestCount, c.requestLatency, "GetReleaseTargets", begin)
	}(time.Now())

	return c.Client.GetReleaseTargets(ctx, pageNumber, pageSize, filters)
}

func (c *metricsClient) GetReleaseTargetsCount(ctx context.Context, filters map[api.FilterType][]string) (count int, err error) {
	defer func(begin time.Time) {
		api.UpdateMetrics(c.requestCount, c.requestLatency, "GetReleaseTargetsCount", begin)
	}(time.Now())

	return c.Client.GetReleaseTargetsCount(ctx, filters)
}

func (c *metricsClient) GetAllPipelinesReleaseTargets(ctx context.Context, pageNumber, pageSize int, filters map[api.FilterType][]string) (releaseTargets []map[string]interface{}, err error) {
	defer func(begin time.Time) {
		api.UpdateMetrics(c.requestCount, c.requestLatency, "GetAllPipelinesReleaseTargets", begin)
	}(time.Now())

	return c.Client.GetAllPipelinesReleaseTargets(ctx, pageNumber, pageSize, filters)
}

func (c *metricsClient) GetAllPipelinesReleaseTargetsCount(ctx context.Context, filters map[api.FilterType][]string) (count int, err error) {
	defer func(begin time.Time) {
		api.UpdateMetrics(c.requestCount, c.requestLatency, "GetAllPipelinesReleaseTargetsCount", begin)
	}(time.Now())

	return c.Client.GetAllPipelinesReleaseTargetsCount(ctx, filters)
}

func (c *metricsClient) GetAllReleasesReleaseTargets(ctx context.Context, pageNumber, pageSize int, filters map[api.FilterType][]string) (releaseTargets []map[string]interface{}, err error) {
	defer func(begin time.Time) {
		api.UpdateMetrics(c.requestCount, c.requestLatency, "GetAllReleasesReleaseTargets", begin)
	}(time.Now())

	return c.Client.GetAllReleasesReleaseTargets(ctx, pageNumber, pageSize, filters)
}

func (c *metricsClient) GetAllReleasesReleaseTargetsCount(ctx context.Context, filters map[api.FilterType][]string) (count int, err error) {
	defer func(begin time.Time) {
		api.UpdateMetrics(c.requestCount, c.requestLatency, "GetAllReleasesReleaseTargetsCount", begin)
	}(time.Now())

	return c.Client.GetAllReleasesReleaseTargetsCount(ctx, filters)
}

func (c *metricsClient) GetPipelineBuildBranches(ctx context.Context, repoSource, repoOwner, repoName string, pageNumber, pageSize int, filters map[api.FilterType][]string) (buildBranches []map[string]interface{}, err error) {
	defer func(begin time.Time) {
		api.UpdateMetrics(c.requestCount, c.requestLatency, "GetPipelineBuildBranches", begin)
	}(time.Now())

	return c.Client.GetPipelineBuildBranches(ctx, repoSource, repoOwner, repoName, pageNumber, pageSize, filters)
}

func (c *metricsClient) GetPipelineBuildBranchesCount(ctx context.Context, repoSource, repoOwner, repoName string, filters map[api.FilterType][]string) (count int, err error) {
	defer func(begin time.Time) {
		api.UpdateMetrics(c.requestCount, c.requestLatency, "GetPipelineBuildBranchesCount", begin)
	}(time.Now())

	return c.Client.GetPipelineBuildBranchesCount(ctx, repoSource, repoOwner, repoName, filters)
}

func (c *metricsClient) GetPipelineBotNames(ctx context.Context, repoSource, repoOwner, repoName string, pageNumber, pageSize int, filters map[api.FilterType][]string) (botNames []map[string]interface{}, err error) {
	defer func(begin time.Time) {
		api.UpdateMetrics(c.requestCount, c.requestLatency, "GetPipelineBotNames", begin)
	}(time.Now())

	return c.Client.GetPipelineBotNames(ctx, repoSource, repoOwner, repoName, pageNumber, pageSize, filters)
}

func (c *metricsClient) GetPipelineBotNamesCount(ctx context.Context, repoSource, repoOwner, repoName string, filters map[api.FilterType][]string) (count int, err error) {
	defer func(begin time.Time) {
		api.UpdateMetrics(c.requestCount, c.requestLatency, "GetPipelineBotNamesCount", begin)
	}(time.Now())

	return c.Client.GetPipelineBotNamesCount(ctx, repoSource, repoOwner, repoName, filters)
}
