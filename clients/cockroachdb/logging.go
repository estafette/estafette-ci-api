package cockroachdb

import (
	"context"
	"time"

	"github.com/estafette/estafette-ci-api/helpers"
	contracts "github.com/estafette/estafette-ci-contracts"
	manifest "github.com/estafette/estafette-ci-manifest"
)

// NewLoggingClient returns a new instance of a logging Client.
func NewLoggingClient(c Client) Client {
	return &loggingClient{c, "cockroachdb"}
}

type loggingClient struct {
	Client
	prefix string
}

func (c *loggingClient) Connect(ctx context.Context) (err error) {
	defer func() { helpers.HandleLogError(c.prefix, "Connect", err) }()

	return c.Client.Connect(ctx)
}

func (c *loggingClient) ConnectWithDriverAndSource(ctx context.Context, driverName, dataSourceName string) (err error) {
	defer func() { helpers.HandleLogError(c.prefix, "ConnectWithDriverAndSource", err) }()

	return c.Client.ConnectWithDriverAndSource(ctx, driverName, dataSourceName)
}

func (c *loggingClient) GetAutoIncrement(ctx context.Context, shortRepoSource, repoOwner, repoName string) (autoincrement int, err error) {
	defer func() { helpers.HandleLogError(c.prefix, "GetAutoIncrement", err) }()

	return c.Client.GetAutoIncrement(ctx, shortRepoSource, repoOwner, repoName)
}

func (c *loggingClient) InsertBuild(ctx context.Context, build contracts.Build, jobResources JobResources) (b *contracts.Build, err error) {
	defer func() { helpers.HandleLogError(c.prefix, "InsertBuild", err) }()

	return c.Client.InsertBuild(ctx, build, jobResources)
}

func (c *loggingClient) UpdateBuildStatus(ctx context.Context, repoSource, repoOwner, repoName string, buildID int, buildStatus string) (err error) {
	defer func() { helpers.HandleLogError(c.prefix, "UpdateBuildStatus", err) }()

	return c.Client.UpdateBuildStatus(ctx, repoSource, repoOwner, repoName, buildID, buildStatus)
}

func (c *loggingClient) UpdateBuildResourceUtilization(ctx context.Context, repoSource, repoOwner, repoName string, buildID int, jobResources JobResources) (err error) {
	defer func() { helpers.HandleLogError(c.prefix, "UpdateBuildResourceUtilization", err) }()

	return c.Client.UpdateBuildResourceUtilization(ctx, repoSource, repoOwner, repoName, buildID, jobResources)
}

func (c *loggingClient) InsertRelease(ctx context.Context, release contracts.Release, jobResources JobResources) (r *contracts.Release, err error) {
	defer func() { helpers.HandleLogError(c.prefix, "InsertRelease", err) }()

	return c.Client.InsertRelease(ctx, release, jobResources)
}

func (c *loggingClient) UpdateReleaseStatus(ctx context.Context, repoSource, repoOwner, repoName string, id int, releaseStatus string) (err error) {
	defer func() { helpers.HandleLogError(c.prefix, "UpdateReleaseStatus", err) }()

	return c.Client.UpdateReleaseStatus(ctx, repoSource, repoOwner, repoName, id, releaseStatus)
}

func (c *loggingClient) UpdateReleaseResourceUtilization(ctx context.Context, repoSource, repoOwner, repoName string, id int, jobResources JobResources) (err error) {
	defer func() { helpers.HandleLogError(c.prefix, "UpdateReleaseResourceUtilization", err) }()

	return c.Client.UpdateReleaseResourceUtilization(ctx, repoSource, repoOwner, repoName, id, jobResources)
}

func (c *loggingClient) InsertBuildLog(ctx context.Context, buildLog contracts.BuildLog, writeLogToDatabase bool) (buildlog contracts.BuildLog, err error) {
	defer func() { helpers.HandleLogError(c.prefix, "InsertBuildLog", err) }()

	return c.Client.InsertBuildLog(ctx, buildLog, writeLogToDatabase)
}

func (c *loggingClient) InsertReleaseLog(ctx context.Context, releaseLog contracts.ReleaseLog, writeLogToDatabase bool) (releaselog contracts.ReleaseLog, err error) {
	defer func() { helpers.HandleLogError(c.prefix, "InsertReleaseLog", err) }()

	return c.Client.InsertReleaseLog(ctx, releaseLog, writeLogToDatabase)
}

func (c *loggingClient) UpsertComputedPipeline(ctx context.Context, repoSource, repoOwner, repoName string) (err error) {
	defer func() { helpers.HandleLogError(c.prefix, "UpsertComputedPipeline", err) }()

	return c.Client.UpsertComputedPipeline(ctx, repoSource, repoOwner, repoName)
}

func (c *loggingClient) UpdateComputedPipelineFirstInsertedAt(ctx context.Context, repoSource, repoOwner, repoName string) (err error) {
	defer func() { helpers.HandleLogError(c.prefix, "UpdateComputedPipelineFirstInsertedAt", err) }()

	return c.Client.UpdateComputedPipelineFirstInsertedAt(ctx, repoSource, repoOwner, repoName)
}

func (c *loggingClient) UpsertComputedRelease(ctx context.Context, repoSource, repoOwner, repoName, releaseName, releaseAction string) (err error) {
	defer func() { helpers.HandleLogError(c.prefix, "UpsertComputedRelease", err) }()

	return c.Client.UpsertComputedRelease(ctx, repoSource, repoOwner, repoName, releaseName, releaseAction)
}

func (c *loggingClient) UpdateComputedReleaseFirstInsertedAt(ctx context.Context, repoSource, repoOwner, repoName, releaseName, releaseAction string) (err error) {
	defer func() { helpers.HandleLogError(c.prefix, "UpdateComputedReleaseFirstInsertedAt", err) }()

	return c.Client.UpdateComputedReleaseFirstInsertedAt(ctx, repoSource, repoOwner, repoName, releaseName, releaseAction)
}

func (c *loggingClient) GetPipelines(ctx context.Context, pageNumber, pageSize int, filters map[string][]string, optimized bool) (pipelines []*contracts.Pipeline, err error) {
	defer func() { helpers.HandleLogError(c.prefix, "GetPipelines", err) }()

	return c.Client.GetPipelines(ctx, pageNumber, pageSize, filters, optimized)
}

func (c *loggingClient) GetPipelinesByRepoName(ctx context.Context, repoName string, optimized bool) (pipelines []*contracts.Pipeline, err error) {
	defer func() { helpers.HandleLogError(c.prefix, "GetPipelinesByRepoName", err) }()

	return c.Client.GetPipelinesByRepoName(ctx, repoName, optimized)
}

func (c *loggingClient) GetPipelinesCount(ctx context.Context, filters map[string][]string) (count int, err error) {
	defer func() { helpers.HandleLogError(c.prefix, "GetPipelinesCount", err) }()

	return c.Client.GetPipelinesCount(ctx, filters)
}

func (c *loggingClient) GetPipeline(ctx context.Context, repoSource, repoOwner, repoName string, optimized bool) (pipeline *contracts.Pipeline, err error) {
	defer func() { helpers.HandleLogError(c.prefix, "GetPipeline", err) }()

	return c.Client.GetPipeline(ctx, repoSource, repoOwner, repoName, optimized)
}

func (c *loggingClient) GetPipelineRecentBuilds(ctx context.Context, repoSource, repoOwner, repoName string, optimized bool) (builds []*contracts.Build, err error) {
	defer func() { helpers.HandleLogError(c.prefix, "GetPipelineRecentBuilds", err) }()

	return c.Client.GetPipelineRecentBuilds(ctx, repoSource, repoOwner, repoName, optimized)
}

func (c *loggingClient) GetPipelineBuilds(ctx context.Context, repoSource, repoOwner, repoName string, pageNumber, pageSize int, filters map[string][]string, optimized bool) (builds []*contracts.Build, err error) {
	defer func() { helpers.HandleLogError(c.prefix, "GetPipelineBuilds", err) }()

	return c.Client.GetPipelineBuilds(ctx, repoSource, repoOwner, repoName, pageNumber, pageSize, filters, optimized)
}

func (c *loggingClient) GetPipelineBuildsCount(ctx context.Context, repoSource, repoOwner, repoName string, filters map[string][]string) (count int, err error) {
	defer func() { helpers.HandleLogError(c.prefix, "GetPipelineBuildsCount", err) }()

	return c.Client.GetPipelineBuildsCount(ctx, repoSource, repoOwner, repoName, filters)
}

func (c *loggingClient) GetPipelineBuild(ctx context.Context, repoSource, repoOwner, repoName, repoRevision string, optimized bool) (build *contracts.Build, err error) {
	defer func() { helpers.HandleLogError(c.prefix, "GetPipelineBuild", err) }()

	return c.Client.GetPipelineBuild(ctx, repoSource, repoOwner, repoName, repoRevision, optimized)
}

func (c *loggingClient) GetPipelineBuildByID(ctx context.Context, repoSource, repoOwner, repoName string, id int, optimized bool) (build *contracts.Build, err error) {
	defer func() { helpers.HandleLogError(c.prefix, "GetPipelineBuildByID", err) }()

	return c.Client.GetPipelineBuildByID(ctx, repoSource, repoOwner, repoName, id, optimized)
}

func (c *loggingClient) GetLastPipelineBuild(ctx context.Context, repoSource, repoOwner, repoName string, optimized bool) (build *contracts.Build, err error) {
	defer func() { helpers.HandleLogError(c.prefix, "GetLastPipelineBuild", err) }()

	return c.Client.GetLastPipelineBuild(ctx, repoSource, repoOwner, repoName, optimized)
}

func (c *loggingClient) GetFirstPipelineBuild(ctx context.Context, repoSource, repoOwner, repoName string, optimized bool) (build *contracts.Build, err error) {
	defer func() { helpers.HandleLogError(c.prefix, "GetFirstPipelineBuild", err) }()

	return c.Client.GetFirstPipelineBuild(ctx, repoSource, repoOwner, repoName, optimized)
}

func (c *loggingClient) GetLastPipelineBuildForBranch(ctx context.Context, repoSource, repoOwner, repoName, branch string) (build *contracts.Build, err error) {
	defer func() { helpers.HandleLogError(c.prefix, "GetLastPipelineBuildForBranch", err) }()

	return c.Client.GetLastPipelineBuildForBranch(ctx, repoSource, repoOwner, repoName, branch)
}

func (c *loggingClient) GetLastPipelineRelease(ctx context.Context, repoSource, repoOwner, repoName, releaseName, releaseAction string) (release *contracts.Release, err error) {
	defer func() { helpers.HandleLogError(c.prefix, "GetLastPipelineRelease", err) }()

	return c.Client.GetLastPipelineRelease(ctx, repoSource, repoOwner, repoName, releaseName, releaseAction)
}

func (c *loggingClient) GetFirstPipelineRelease(ctx context.Context, repoSource, repoOwner, repoName, releaseName, releaseAction string) (release *contracts.Release, err error) {
	defer func() { helpers.HandleLogError(c.prefix, "GetFirstPipelineRelease", err) }()

	return c.Client.GetFirstPipelineRelease(ctx, repoSource, repoOwner, repoName, releaseName, releaseAction)
}

func (c *loggingClient) GetPipelineBuildsByVersion(ctx context.Context, repoSource, repoOwner, repoName, buildVersion string, statuses []string, limit uint64, optimized bool) (builds []*contracts.Build, err error) {
	defer func() { helpers.HandleLogError(c.prefix, "GetPipelineBuildsByVersion", err) }()

	return c.Client.GetPipelineBuildsByVersion(ctx, repoSource, repoOwner, repoName, buildVersion, statuses, limit, optimized)
}

func (c *loggingClient) GetPipelineBuildLogs(ctx context.Context, repoSource, repoOwner, repoName, repoBranch, repoRevision, buildID string, readLogFromDatabase bool) (buildlog *contracts.BuildLog, err error) {
	defer func() { helpers.HandleLogError(c.prefix, "GetPipelineBuildLogs", err) }()

	return c.Client.GetPipelineBuildLogs(ctx, repoSource, repoOwner, repoName, repoBranch, repoRevision, buildID, readLogFromDatabase)
}

func (c *loggingClient) GetPipelineBuildLogsPerPage(ctx context.Context, repoSource, repoOwner, repoName string, pageNumber int, pageSize int) (buildlogs []*contracts.BuildLog, err error) {
	defer func() { helpers.HandleLogError(c.prefix, "GetPipelineBuildLogsPerPage", err) }()

	return c.Client.GetPipelineBuildLogsPerPage(ctx, repoSource, repoOwner, repoName, pageNumber, pageSize)
}

func (c *loggingClient) GetPipelineBuildMaxResourceUtilization(ctx context.Context, repoSource, repoOwner, repoName string, lastNRecords int) (jobresources JobResources, count int, err error) {
	defer func() { helpers.HandleLogError(c.prefix, "GetPipelineBuildMaxResourceUtilization", err) }()

	return c.Client.GetPipelineBuildMaxResourceUtilization(ctx, repoSource, repoOwner, repoName, lastNRecords)
}

func (c *loggingClient) GetPipelineReleases(ctx context.Context, repoSource, repoOwner, repoName string, pageNumber, pageSize int, filters map[string][]string) (releases []*contracts.Release, err error) {
	defer func() { helpers.HandleLogError(c.prefix, "GetPipelineReleases", err) }()

	return c.Client.GetPipelineReleases(ctx, repoSource, repoOwner, repoName, pageNumber, pageSize, filters)
}

func (c *loggingClient) GetPipelineReleasesCount(ctx context.Context, repoSource, repoOwner, repoName string, filters map[string][]string) (count int, err error) {
	defer func() { helpers.HandleLogError(c.prefix, "GetPipelineReleasesCount", err) }()

	return c.Client.GetPipelineReleasesCount(ctx, repoSource, repoOwner, repoName, filters)
}

func (c *loggingClient) GetPipelineRelease(ctx context.Context, repoSource, repoOwner, repoName string, id int) (release *contracts.Release, err error) {
	defer func() { helpers.HandleLogError(c.prefix, "GetPipelineRelease", err) }()

	return c.Client.GetPipelineRelease(ctx, repoSource, repoOwner, repoName, id)
}

func (c *loggingClient) GetPipelineLastReleasesByName(ctx context.Context, repoSource, repoOwner, repoName, releaseName string, actions []string) (releases []contracts.Release, err error) {
	defer func() { helpers.HandleLogError(c.prefix, "GetPipelineLastReleasesByName", err) }()

	return c.Client.GetPipelineLastReleasesByName(ctx, repoSource, repoOwner, repoName, releaseName, actions)
}

func (c *loggingClient) GetPipelineReleaseLogs(ctx context.Context, repoSource, repoOwner, repoName string, id int, readLogFromDatabase bool) (releaselog *contracts.ReleaseLog, err error) {
	defer func() { helpers.HandleLogError(c.prefix, "GetPipelineReleaseLogs", err) }()

	return c.Client.GetPipelineReleaseLogs(ctx, repoSource, repoOwner, repoName, id, readLogFromDatabase)
}

func (c *loggingClient) GetPipelineReleaseLogsPerPage(ctx context.Context, repoSource, repoOwner, repoName string, pageNumber int, pageSize int) (releaselogs []*contracts.ReleaseLog, err error) {
	defer func() { helpers.HandleLogError(c.prefix, "GetPipelineReleaseLogsPerPage", err) }()

	return c.Client.GetPipelineReleaseLogsPerPage(ctx, repoSource, repoOwner, repoName, pageNumber, pageSize)
}

func (c *loggingClient) GetPipelineReleaseMaxResourceUtilization(ctx context.Context, repoSource, repoOwner, repoName, targetName string, lastNRecords int) (jobresources JobResources, count int, err error) {
	defer func() { helpers.HandleLogError(c.prefix, "GetPipelineReleaseMaxResourceUtilization", err) }()

	return c.Client.GetPipelineReleaseMaxResourceUtilization(ctx, repoSource, repoOwner, repoName, targetName, lastNRecords)
}

func (c *loggingClient) GetBuildsCount(ctx context.Context, filters map[string][]string) (count int, err error) {
	defer func() { helpers.HandleLogError(c.prefix, "GetBuildsCount", err) }()

	return c.Client.GetBuildsCount(ctx, filters)
}

func (c *loggingClient) GetReleasesCount(ctx context.Context, filters map[string][]string) (count int, err error) {
	defer func() { helpers.HandleLogError(c.prefix, "GetReleasesCount", err) }()

	return c.Client.GetReleasesCount(ctx, filters)
}

func (c *loggingClient) GetBuildsDuration(ctx context.Context, filters map[string][]string) (duration time.Duration, err error) {
	defer func() { helpers.HandleLogError(c.prefix, "GetBuildsDuration", err) }()

	return c.Client.GetBuildsDuration(ctx, filters)
}

func (c *loggingClient) GetFirstBuildTimes(ctx context.Context) (times []time.Time, err error) {
	defer func() { helpers.HandleLogError(c.prefix, "GetFirstBuildTimes", err) }()

	return c.Client.GetFirstBuildTimes(ctx)
}

func (c *loggingClient) GetFirstReleaseTimes(ctx context.Context) (times []time.Time, err error) {
	defer func() { helpers.HandleLogError(c.prefix, "GetFirstReleaseTimes", err) }()

	return c.Client.GetFirstReleaseTimes(ctx)
}

func (c *loggingClient) GetPipelineBuildsDurations(ctx context.Context, repoSource, repoOwner, repoName string, filters map[string][]string) (durations []map[string]interface{}, err error) {
	defer func() { helpers.HandleLogError(c.prefix, "GetPipelineBuildsDurations", err) }()

	return c.Client.GetPipelineBuildsDurations(ctx, repoSource, repoOwner, repoName, filters)
}

func (c *loggingClient) GetPipelineReleasesDurations(ctx context.Context, repoSource, repoOwner, repoName string, filters map[string][]string) (durations []map[string]interface{}, err error) {
	defer func() { helpers.HandleLogError(c.prefix, "GetPipelineReleasesDurations", err) }()

	return c.Client.GetPipelineReleasesDurations(ctx, repoSource, repoOwner, repoName, filters)
}

func (c *loggingClient) GetPipelineBuildsCPUUsageMeasurements(ctx context.Context, repoSource, repoOwner, repoName string, filters map[string][]string) (measurements []map[string]interface{}, err error) {
	defer func() { helpers.HandleLogError(c.prefix, "GetPipelineBuildsCPUUsageMeasurements", err) }()

	return c.Client.GetPipelineBuildsCPUUsageMeasurements(ctx, repoSource, repoOwner, repoName, filters)
}

func (c *loggingClient) GetPipelineReleasesCPUUsageMeasurements(ctx context.Context, repoSource, repoOwner, repoName string, filters map[string][]string) (measurements []map[string]interface{}, err error) {
	defer func() { helpers.HandleLogError(c.prefix, "GetPipelineReleasesCPUUsageMeasurements", err) }()

	return c.Client.GetPipelineReleasesCPUUsageMeasurements(ctx, repoSource, repoOwner, repoName, filters)
}

func (c *loggingClient) GetPipelineBuildsMemoryUsageMeasurements(ctx context.Context, repoSource, repoOwner, repoName string, filters map[string][]string) (measurements []map[string]interface{}, err error) {
	defer func() { helpers.HandleLogError(c.prefix, "GetPipelineBuildsMemoryUsageMeasurements", err) }()

	return c.Client.GetPipelineBuildsMemoryUsageMeasurements(ctx, repoSource, repoOwner, repoName, filters)
}

func (c *loggingClient) GetPipelineReleasesMemoryUsageMeasurements(ctx context.Context, repoSource, repoOwner, repoName string, filters map[string][]string) (measurements []map[string]interface{}, err error) {
	defer func() { helpers.HandleLogError(c.prefix, "GetPipelineReleasesMemoryUsageMeasurements", err) }()

	return c.Client.GetPipelineReleasesMemoryUsageMeasurements(ctx, repoSource, repoOwner, repoName, filters)
}

func (c *loggingClient) GetFrequentLabels(ctx context.Context, pageNumber, pageSize int, filters map[string][]string) (measurements []map[string]interface{}, err error) {
	defer func() { helpers.HandleLogError(c.prefix, "GetFrequentLabels", err) }()

	return c.Client.GetFrequentLabels(ctx, pageNumber, pageSize, filters)
}

func (c *loggingClient) GetFrequentLabelsCount(ctx context.Context, filters map[string][]string) (count int, err error) {
	defer func() { helpers.HandleLogError(c.prefix, "GetFrequentLabelsCount", err) }()

	return c.Client.GetFrequentLabelsCount(ctx, filters)
}

func (c *loggingClient) GetPipelinesWithMostBuilds(ctx context.Context, pageNumber, pageSize int, filters map[string][]string) (pipelines []map[string]interface{}, err error) {
	defer func() { helpers.HandleLogError(c.prefix, "GetPipelinesWithMostBuilds", err) }()

	return c.Client.GetPipelinesWithMostBuilds(ctx, pageNumber, pageSize, filters)
}

func (c *loggingClient) GetPipelinesWithMostBuildsCount(ctx context.Context, filters map[string][]string) (count int, err error) {
	defer func() { helpers.HandleLogError(c.prefix, "GetPipelinesWithMostBuildsCount", err) }()

	return c.Client.GetPipelinesWithMostBuildsCount(ctx, filters)
}

func (c *loggingClient) GetPipelinesWithMostReleases(ctx context.Context, pageNumber, pageSize int, filters map[string][]string) (pipelines []map[string]interface{}, err error) {
	defer func() { helpers.HandleLogError(c.prefix, "GetPipelinesWithMostReleases", err) }()

	return c.Client.GetPipelinesWithMostReleases(ctx, pageNumber, pageSize, filters)
}

func (c *loggingClient) GetPipelinesWithMostReleasesCount(ctx context.Context, filters map[string][]string) (count int, err error) {
	defer func() { helpers.HandleLogError(c.prefix, "GetPipelinesWithMostReleasesCount", err) }()

	return c.Client.GetPipelinesWithMostReleasesCount(ctx, filters)
}

func (c *loggingClient) GetTriggers(ctx context.Context, triggerType, identifier, event string) (pipelines []*contracts.Pipeline, err error) {
	defer func() { helpers.HandleLogError(c.prefix, "GetTriggers", err) }()

	return c.Client.GetTriggers(ctx, triggerType, identifier, event)
}

func (c *loggingClient) GetGitTriggers(ctx context.Context, gitEvent manifest.EstafetteGitEvent) (pipelines []*contracts.Pipeline, err error) {
	defer func() { helpers.HandleLogError(c.prefix, "GetGitTriggers", err) }()

	return c.Client.GetGitTriggers(ctx, gitEvent)
}

func (c *loggingClient) GetPipelineTriggers(ctx context.Context, build contracts.Build, event string) (pipelines []*contracts.Pipeline, err error) {
	defer func() { helpers.HandleLogError(c.prefix, "GetPipelineTriggers", err) }()

	return c.Client.GetPipelineTriggers(ctx, build, event)
}

func (c *loggingClient) GetReleaseTriggers(ctx context.Context, release contracts.Release, event string) (pipelines []*contracts.Pipeline, err error) {
	defer func() { helpers.HandleLogError(c.prefix, "GetReleaseTriggers", err) }()

	return c.Client.GetReleaseTriggers(ctx, release, event)
}

func (c *loggingClient) GetPubSubTriggers(ctx context.Context, pubsubEvent manifest.EstafettePubSubEvent) (pipelines []*contracts.Pipeline, err error) {
	defer func() { helpers.HandleLogError(c.prefix, "GetPubSubTriggers", err) }()

	return c.Client.GetPubSubTriggers(ctx, pubsubEvent)
}

func (c *loggingClient) GetCronTriggers(ctx context.Context) (pipelines []*contracts.Pipeline, err error) {
	defer func() { helpers.HandleLogError(c.prefix, "GetCronTriggers", err) }()

	return c.Client.GetCronTriggers(ctx)
}

func (c *loggingClient) Rename(ctx context.Context, shortFromRepoSource, fromRepoSource, fromRepoOwner, fromRepoName, shortToRepoSource, toRepoSource, toRepoOwner, toRepoName string) (err error) {
	defer func() { helpers.HandleLogError(c.prefix, "Rename", err) }()

	return c.Client.Rename(ctx, shortFromRepoSource, fromRepoSource, fromRepoOwner, fromRepoName, shortToRepoSource, toRepoSource, toRepoOwner, toRepoName)
}

func (c *loggingClient) RenameBuildVersion(ctx context.Context, shortFromRepoSource, fromRepoOwner, fromRepoName, shortToRepoSource, toRepoOwner, toRepoName string) (err error) {
	defer func() { helpers.HandleLogError(c.prefix, "RenameBuildVersion", err) }()

	return c.Client.RenameBuildVersion(ctx, shortFromRepoSource, fromRepoOwner, fromRepoName, shortToRepoSource, toRepoOwner, toRepoName)
}

func (c *loggingClient) RenameBuilds(ctx context.Context, fromRepoSource, fromRepoOwner, fromRepoName, toRepoSource, toRepoOwner, toRepoName string) (err error) {
	defer func() { helpers.HandleLogError(c.prefix, "RenameBuilds", err) }()

	return c.Client.RenameBuilds(ctx, fromRepoSource, fromRepoOwner, fromRepoName, toRepoSource, toRepoOwner, toRepoName)
}

func (c *loggingClient) RenameBuildLogs(ctx context.Context, fromRepoSource, fromRepoOwner, fromRepoName, toRepoSource, toRepoOwner, toRepoName string) (err error) {
	defer func() { helpers.HandleLogError(c.prefix, "RenameBuildLogs", err) }()

	return c.Client.RenameBuildLogs(ctx, fromRepoSource, fromRepoOwner, fromRepoName, toRepoSource, toRepoOwner, toRepoName)
}

func (c *loggingClient) RenameReleases(ctx context.Context, fromRepoSource, fromRepoOwner, fromRepoName, toRepoSource, toRepoOwner, toRepoName string) (err error) {
	defer func() { helpers.HandleLogError(c.prefix, "RenameReleases", err) }()

	return c.Client.RenameReleases(ctx, fromRepoSource, fromRepoOwner, fromRepoName, toRepoSource, toRepoOwner, toRepoName)
}

func (c *loggingClient) RenameReleaseLogs(ctx context.Context, fromRepoSource, fromRepoOwner, fromRepoName, toRepoSource, toRepoOwner, toRepoName string) (err error) {
	defer func() { helpers.HandleLogError(c.prefix, "RenameReleaseLogs", err) }()

	return c.Client.RenameReleaseLogs(ctx, fromRepoSource, fromRepoOwner, fromRepoName, toRepoSource, toRepoOwner, toRepoName)
}

func (c *loggingClient) RenameComputedPipelines(ctx context.Context, fromRepoSource, fromRepoOwner, fromRepoName, toRepoSource, toRepoOwner, toRepoName string) (err error) {
	defer func() { helpers.HandleLogError(c.prefix, "RenameComputedPipelines", err) }()

	return c.Client.RenameComputedPipelines(ctx, fromRepoSource, fromRepoOwner, fromRepoName, toRepoSource, toRepoOwner, toRepoName)
}

func (c *loggingClient) RenameComputedReleases(ctx context.Context, fromRepoSource, fromRepoOwner, fromRepoName, toRepoSource, toRepoOwner, toRepoName string) (err error) {
	defer func() { helpers.HandleLogError(c.prefix, "RenameComputedReleases", err) }()

	return c.Client.RenameComputedReleases(ctx, fromRepoSource, fromRepoOwner, fromRepoName, toRepoSource, toRepoOwner, toRepoName)
}
