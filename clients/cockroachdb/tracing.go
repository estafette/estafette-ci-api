package cockroachdb

import (
	"context"
	"time"

	"github.com/estafette/estafette-ci-api/helpers"
	contracts "github.com/estafette/estafette-ci-contracts"
	manifest "github.com/estafette/estafette-ci-manifest"
	"github.com/opentracing/opentracing-go"
)

// NewTracingClient returns a new instance of a tracing Client.
func NewTracingClient(c Client) Client {
	return &tracingClient{c}
}

type tracingClient struct {
	Client
}

func (c *tracingClient) Connect(ctx context.Context) (err error) {

	span, ctx := opentracing.StartSpanFromContext(ctx, c.getSpanName("Connect"))
	defer func() { helpers.FinishSpanWithError(span, err) }()

	return c.Client.Connect(ctx)
}

func (c *tracingClient) ConnectWithDriverAndSource(ctx context.Context, driverName, dataSourceName string) (err error) {

	span, ctx := opentracing.StartSpanFromContext(ctx, c.getSpanName("ConnectWithDriverAndSource"))
	defer func() { helpers.FinishSpanWithError(span, err) }()

	return c.Client.ConnectWithDriverAndSource(ctx, driverName, dataSourceName)
}

func (c *tracingClient) GetAutoIncrement(ctx context.Context, shortRepoSource, repoOwner, repoName string) (autoincrement int, err error) {

	span, ctx := opentracing.StartSpanFromContext(ctx, c.getSpanName("GetAutoIncrement"))
	defer func() { helpers.FinishSpanWithError(span, err) }()

	return c.Client.GetAutoIncrement(ctx, shortRepoSource, repoOwner, repoName)
}

func (c *tracingClient) InsertBuild(ctx context.Context, build contracts.Build, jobResources JobResources) (b *contracts.Build, err error) {

	span, ctx := opentracing.StartSpanFromContext(ctx, c.getSpanName("InsertBuild"))
	defer func() { helpers.FinishSpanWithError(span, err) }()

	return c.Client.InsertBuild(ctx, build, jobResources)
}

func (c *tracingClient) UpdateBuildStatus(ctx context.Context, repoSource, repoOwner, repoName string, buildID int, buildStatus string) (err error) {

	span, ctx := opentracing.StartSpanFromContext(ctx, c.getSpanName("UpdateBuildStatus"))
	defer func() { helpers.FinishSpanWithError(span, err) }()

	return c.Client.UpdateBuildStatus(ctx, repoSource, repoOwner, repoName, buildID, buildStatus)
}

func (c *tracingClient) UpdateBuildResourceUtilization(ctx context.Context, repoSource, repoOwner, repoName string, buildID int, jobResources JobResources) (err error) {

	span, ctx := opentracing.StartSpanFromContext(ctx, c.getSpanName("UpdateBuildResourceUtilization"))
	defer func() { helpers.FinishSpanWithError(span, err) }()

	return c.Client.UpdateBuildResourceUtilization(ctx, repoSource, repoOwner, repoName, buildID, jobResources)
}

func (c *tracingClient) InsertRelease(ctx context.Context, release contracts.Release, jobResources JobResources) (r *contracts.Release, err error) {

	span, ctx := opentracing.StartSpanFromContext(ctx, c.getSpanName("InsertRelease"))
	defer func() { helpers.FinishSpanWithError(span, err) }()

	return c.Client.InsertRelease(ctx, release, jobResources)
}

func (c *tracingClient) UpdateReleaseStatus(ctx context.Context, repoSource, repoOwner, repoName string, id int, releaseStatus string) (err error) {

	span, ctx := opentracing.StartSpanFromContext(ctx, c.getSpanName("UpdateReleaseStatus"))
	defer func() { helpers.FinishSpanWithError(span, err) }()

	return c.Client.UpdateReleaseStatus(ctx, repoSource, repoOwner, repoName, id, releaseStatus)
}

func (c *tracingClient) UpdateReleaseResourceUtilization(ctx context.Context, repoSource, repoOwner, repoName string, id int, jobResources JobResources) (err error) {

	span, ctx := opentracing.StartSpanFromContext(ctx, c.getSpanName("UpdateReleaseResourceUtilization"))
	defer func() { helpers.FinishSpanWithError(span, err) }()

	return c.Client.UpdateReleaseResourceUtilization(ctx, repoSource, repoOwner, repoName, id, jobResources)
}

func (c *tracingClient) InsertBuildLog(ctx context.Context, buildLog contracts.BuildLog, writeLogToDatabase bool) (buildlog contracts.BuildLog, err error) {

	span, ctx := opentracing.StartSpanFromContext(ctx, c.getSpanName("InsertBuildLog"))
	defer func() { helpers.FinishSpanWithError(span, err) }()

	return c.Client.InsertBuildLog(ctx, buildLog, writeLogToDatabase)
}

func (c *tracingClient) InsertReleaseLog(ctx context.Context, releaseLog contracts.ReleaseLog, writeLogToDatabase bool) (releaselog contracts.ReleaseLog, err error) {

	span, ctx := opentracing.StartSpanFromContext(ctx, c.getSpanName("InsertReleaseLog"))
	defer func() { helpers.FinishSpanWithError(span, err) }()

	return c.Client.InsertReleaseLog(ctx, releaseLog, writeLogToDatabase)
}

func (c *tracingClient) UpsertComputedPipeline(ctx context.Context, repoSource, repoOwner, repoName string) (err error) {

	span, ctx := opentracing.StartSpanFromContext(ctx, c.getSpanName("UpsertComputedPipeline"))
	defer func() { helpers.FinishSpanWithError(span, err) }()

	return c.Client.UpsertComputedPipeline(ctx, repoSource, repoOwner, repoName)
}

func (c *tracingClient) UpdateComputedPipelineFirstInsertedAt(ctx context.Context, repoSource, repoOwner, repoName string) (err error) {

	span, ctx := opentracing.StartSpanFromContext(ctx, c.getSpanName("UpdateComputedPipelineFirstInsertedAt"))
	defer func() { helpers.FinishSpanWithError(span, err) }()

	return c.Client.UpdateComputedPipelineFirstInsertedAt(ctx, repoSource, repoOwner, repoName)
}

func (c *tracingClient) UpsertComputedRelease(ctx context.Context, repoSource, repoOwner, repoName, releaseName, releaseAction string) (err error) {

	span, ctx := opentracing.StartSpanFromContext(ctx, c.getSpanName("UpsertComputedRelease"))
	defer func() { helpers.FinishSpanWithError(span, err) }()

	return c.Client.UpsertComputedRelease(ctx, repoSource, repoOwner, repoName, releaseName, releaseAction)
}

func (c *tracingClient) UpdateComputedReleaseFirstInsertedAt(ctx context.Context, repoSource, repoOwner, repoName, releaseName, releaseAction string) (err error) {

	span, ctx := opentracing.StartSpanFromContext(ctx, c.getSpanName("UpdateComputedReleaseFirstInsertedAt"))
	defer func() { helpers.FinishSpanWithError(span, err) }()

	return c.Client.UpdateComputedReleaseFirstInsertedAt(ctx, repoSource, repoOwner, repoName, releaseName, releaseAction)
}

func (c *tracingClient) GetPipelines(ctx context.Context, pageNumber, pageSize int, filters map[string][]string, optimized bool) (pipelines []*contracts.Pipeline, err error) {

	span, ctx := opentracing.StartSpanFromContext(ctx, c.getSpanName("GetPipelines"))
	defer func() { helpers.FinishSpanWithError(span, err) }()

	return c.Client.GetPipelines(ctx, pageNumber, pageSize, filters, optimized)
}

func (c *tracingClient) GetPipelinesByRepoName(ctx context.Context, repoName string, optimized bool) (pipelines []*contracts.Pipeline, err error) {

	span, ctx := opentracing.StartSpanFromContext(ctx, c.getSpanName("GetPipelinesByRepoName"))
	defer func() { helpers.FinishSpanWithError(span, err) }()

	return c.Client.GetPipelinesByRepoName(ctx, repoName, optimized)
}

func (c *tracingClient) GetPipelinesCount(ctx context.Context, filters map[string][]string) (count int, err error) {

	span, ctx := opentracing.StartSpanFromContext(ctx, c.getSpanName("GetPipelinesCount"))
	defer func() { helpers.FinishSpanWithError(span, err) }()

	return c.Client.GetPipelinesCount(ctx, filters)
}

func (c *tracingClient) GetPipeline(ctx context.Context, repoSource, repoOwner, repoName string, optimized bool) (pipeline *contracts.Pipeline, err error) {

	span, ctx := opentracing.StartSpanFromContext(ctx, c.getSpanName("GetPipeline"))
	defer func() { helpers.FinishSpanWithError(span, err) }()

	return c.Client.GetPipeline(ctx, repoSource, repoOwner, repoName, optimized)
}

func (c *tracingClient) GetPipelineBuilds(ctx context.Context, repoSource, repoOwner, repoName string, pageNumber, pageSize int, filters map[string][]string, optimized bool) (builds []*contracts.Build, err error) {

	span, ctx := opentracing.StartSpanFromContext(ctx, c.getSpanName("GetPipelineBuilds"))
	defer func() { helpers.FinishSpanWithError(span, err) }()

	return c.Client.GetPipelineBuilds(ctx, repoSource, repoOwner, repoName, pageNumber, pageSize, filters, optimized)
}

func (c *tracingClient) GetPipelineBuildsCount(ctx context.Context, repoSource, repoOwner, repoName string, filters map[string][]string) (count int, err error) {

	span, ctx := opentracing.StartSpanFromContext(ctx, c.getSpanName("GetPipelineBuildsCount"))
	defer func() { helpers.FinishSpanWithError(span, err) }()

	return c.Client.GetPipelineBuildsCount(ctx, repoSource, repoOwner, repoName, filters)
}

func (c *tracingClient) GetPipelineBuild(ctx context.Context, repoSource, repoOwner, repoName, repoRevision string, optimized bool) (build *contracts.Build, err error) {

	span, ctx := opentracing.StartSpanFromContext(ctx, c.getSpanName("GetPipelineBuild"))
	defer func() { helpers.FinishSpanWithError(span, err) }()

	return c.Client.GetPipelineBuild(ctx, repoSource, repoOwner, repoName, repoRevision, optimized)
}

func (c *tracingClient) GetPipelineBuildByID(ctx context.Context, repoSource, repoOwner, repoName string, id int, optimized bool) (build *contracts.Build, err error) {

	span, ctx := opentracing.StartSpanFromContext(ctx, c.getSpanName("GetPipelineBuildByID"))
	defer func() { helpers.FinishSpanWithError(span, err) }()

	return c.Client.GetPipelineBuildByID(ctx, repoSource, repoOwner, repoName, id, optimized)
}

func (c *tracingClient) GetLastPipelineBuild(ctx context.Context, repoSource, repoOwner, repoName string, optimized bool) (build *contracts.Build, err error) {

	span, ctx := opentracing.StartSpanFromContext(ctx, c.getSpanName("GetLastPipelineBuild"))
	defer func() { helpers.FinishSpanWithError(span, err) }()

	return c.Client.GetLastPipelineBuild(ctx, repoSource, repoOwner, repoName, optimized)
}

func (c *tracingClient) GetFirstPipelineBuild(ctx context.Context, repoSource, repoOwner, repoName string, optimized bool) (build *contracts.Build, err error) {

	span, ctx := opentracing.StartSpanFromContext(ctx, c.getSpanName("GetFirstPipelineBuild"))
	defer func() { helpers.FinishSpanWithError(span, err) }()

	return c.Client.GetFirstPipelineBuild(ctx, repoSource, repoOwner, repoName, optimized)
}

func (c *tracingClient) GetLastPipelineBuildForBranch(ctx context.Context, repoSource, repoOwner, repoName, branch string) (build *contracts.Build, err error) {

	span, ctx := opentracing.StartSpanFromContext(ctx, c.getSpanName("GetLastPipelineBuildForBranch"))
	defer func() { helpers.FinishSpanWithError(span, err) }()

	return c.Client.GetLastPipelineBuildForBranch(ctx, repoSource, repoOwner, repoName, branch)
}

func (c *tracingClient) GetLastPipelineRelease(ctx context.Context, repoSource, repoOwner, repoName, releaseName, releaseAction string) (release *contracts.Release, err error) {

	span, ctx := opentracing.StartSpanFromContext(ctx, c.getSpanName("GetLastPipelineRelease"))
	defer func() { helpers.FinishSpanWithError(span, err) }()

	return c.Client.GetLastPipelineRelease(ctx, repoSource, repoOwner, repoName, releaseName, releaseAction)
}

func (c *tracingClient) GetFirstPipelineRelease(ctx context.Context, repoSource, repoOwner, repoName, releaseName, releaseAction string) (release *contracts.Release, err error) {

	span, ctx := opentracing.StartSpanFromContext(ctx, c.getSpanName("GetFirstPipelineRelease"))
	defer func() { helpers.FinishSpanWithError(span, err) }()

	return c.Client.GetFirstPipelineRelease(ctx, repoSource, repoOwner, repoName, releaseName, releaseAction)
}

func (c *tracingClient) GetPipelineBuildsByVersion(ctx context.Context, repoSource, repoOwner, repoName, buildVersion string, statuses []string, limit uint64, optimized bool) (builds []*contracts.Build, err error) {

	span, ctx := opentracing.StartSpanFromContext(ctx, c.getSpanName("GetPipelineBuildsByVersion"))
	defer func() { helpers.FinishSpanWithError(span, err) }()

	return c.Client.GetPipelineBuildsByVersion(ctx, repoSource, repoOwner, repoName, buildVersion, statuses, limit, optimized)
}

func (c *tracingClient) GetPipelineBuildLogs(ctx context.Context, repoSource, repoOwner, repoName, repoBranch, repoRevision, buildID string, readLogFromDatabase bool) (buildlog *contracts.BuildLog, err error) {

	span, ctx := opentracing.StartSpanFromContext(ctx, c.getSpanName("GetPipelineBuildLogs"))
	defer func() { helpers.FinishSpanWithError(span, err) }()

	return c.Client.GetPipelineBuildLogs(ctx, repoSource, repoOwner, repoName, repoBranch, repoRevision, buildID, readLogFromDatabase)
}

func (c *tracingClient) GetPipelineBuildLogsPerPage(ctx context.Context, repoSource, repoOwner, repoName string, pageNumber int, pageSize int) (buildlogs []*contracts.BuildLog, err error) {

	span, ctx := opentracing.StartSpanFromContext(ctx, c.getSpanName("GetPipelineBuildLogsPerPage"))
	defer func() { helpers.FinishSpanWithError(span, err) }()

	return c.Client.GetPipelineBuildLogsPerPage(ctx, repoSource, repoOwner, repoName, pageNumber, pageSize)
}

func (c *tracingClient) GetPipelineBuildMaxResourceUtilization(ctx context.Context, repoSource, repoOwner, repoName string, lastNRecords int) (jobresources JobResources, count int, err error) {

	span, ctx := opentracing.StartSpanFromContext(ctx, c.getSpanName("GetPipelineBuildMaxResourceUtilization"))
	defer func() { helpers.FinishSpanWithError(span, err) }()

	return c.Client.GetPipelineBuildMaxResourceUtilization(ctx, repoSource, repoOwner, repoName, lastNRecords)
}

func (c *tracingClient) GetPipelineReleases(ctx context.Context, repoSource, repoOwner, repoName string, pageNumber, pageSize int, filters map[string][]string) (releases []*contracts.Release, err error) {

	span, ctx := opentracing.StartSpanFromContext(ctx, c.getSpanName("GetPipelineReleases"))
	defer func() { helpers.FinishSpanWithError(span, err) }()

	return c.Client.GetPipelineReleases(ctx, repoSource, repoOwner, repoName, pageNumber, pageSize, filters)
}

func (c *tracingClient) GetPipelineReleasesCount(ctx context.Context, repoSource, repoOwner, repoName string, filters map[string][]string) (count int, err error) {

	span, ctx := opentracing.StartSpanFromContext(ctx, c.getSpanName("GetPipelineReleasesCount"))
	defer func() { helpers.FinishSpanWithError(span, err) }()

	return c.Client.GetPipelineReleasesCount(ctx, repoSource, repoOwner, repoName, filters)
}

func (c *tracingClient) GetPipelineRelease(ctx context.Context, repoSource, repoOwner, repoName string, id int) (release *contracts.Release, err error) {

	span, ctx := opentracing.StartSpanFromContext(ctx, c.getSpanName("GetPipelineRelease"))
	defer func() { helpers.FinishSpanWithError(span, err) }()

	return c.Client.GetPipelineRelease(ctx, repoSource, repoOwner, repoName, id)
}

func (c *tracingClient) GetPipelineLastReleasesByName(ctx context.Context, repoSource, repoOwner, repoName, releaseName string, actions []string) (releases []contracts.Release, err error) {

	span, ctx := opentracing.StartSpanFromContext(ctx, c.getSpanName("GetPipelineLastReleasesByName"))
	defer func() { helpers.FinishSpanWithError(span, err) }()

	return c.Client.GetPipelineLastReleasesByName(ctx, repoSource, repoOwner, repoName, releaseName, actions)
}

func (c *tracingClient) GetPipelineReleaseLogs(ctx context.Context, repoSource, repoOwner, repoName string, id int, readLogFromDatabase bool) (releaselog *contracts.ReleaseLog, err error) {

	span, ctx := opentracing.StartSpanFromContext(ctx, c.getSpanName("GetPipelineReleaseLogs"))
	defer func() { helpers.FinishSpanWithError(span, err) }()

	return c.Client.GetPipelineReleaseLogs(ctx, repoSource, repoOwner, repoName, id, readLogFromDatabase)
}

func (c *tracingClient) GetPipelineReleaseLogsPerPage(ctx context.Context, repoSource, repoOwner, repoName string, pageNumber int, pageSize int) (releaselogs []*contracts.ReleaseLog, err error) {

	span, ctx := opentracing.StartSpanFromContext(ctx, c.getSpanName("GetPipelineReleaseLogsPerPage"))
	defer func() { helpers.FinishSpanWithError(span, err) }()

	return c.Client.GetPipelineReleaseLogsPerPage(ctx, repoSource, repoOwner, repoName, pageNumber, pageSize)
}

func (c *tracingClient) GetPipelineReleaseMaxResourceUtilization(ctx context.Context, repoSource, repoOwner, repoName, targetName string, lastNRecords int) (jobresources JobResources, count int, err error) {

	span, ctx := opentracing.StartSpanFromContext(ctx, c.getSpanName("GetPipelineReleaseMaxResourceUtilization"))
	defer func() { helpers.FinishSpanWithError(span, err) }()

	return c.Client.GetPipelineReleaseMaxResourceUtilization(ctx, repoSource, repoOwner, repoName, targetName, lastNRecords)
}

func (c *tracingClient) GetBuildsCount(ctx context.Context, filters map[string][]string) (count int, err error) {

	span, ctx := opentracing.StartSpanFromContext(ctx, c.getSpanName("GetBuildsCount"))
	defer func() { helpers.FinishSpanWithError(span, err) }()

	return c.Client.GetBuildsCount(ctx, filters)
}

func (c *tracingClient) GetReleasesCount(ctx context.Context, filters map[string][]string) (count int, err error) {

	span, ctx := opentracing.StartSpanFromContext(ctx, c.getSpanName("GetReleasesCount"))
	defer func() { helpers.FinishSpanWithError(span, err) }()

	return c.Client.GetReleasesCount(ctx, filters)
}

func (c *tracingClient) GetBuildsDuration(ctx context.Context, filters map[string][]string) (duration time.Duration, err error) {

	span, ctx := opentracing.StartSpanFromContext(ctx, c.getSpanName("GetBuildsDuration"))
	defer func() { helpers.FinishSpanWithError(span, err) }()

	return c.Client.GetBuildsDuration(ctx, filters)
}

func (c *tracingClient) GetFirstBuildTimes(ctx context.Context) (times []time.Time, err error) {

	span, ctx := opentracing.StartSpanFromContext(ctx, c.getSpanName("GetFirstBuildTimes"))
	defer func() { helpers.FinishSpanWithError(span, err) }()

	return c.Client.GetFirstBuildTimes(ctx)
}

func (c *tracingClient) GetFirstReleaseTimes(ctx context.Context) (times []time.Time, err error) {

	span, ctx := opentracing.StartSpanFromContext(ctx, c.getSpanName("GetFirstReleaseTimes"))
	defer func() { helpers.FinishSpanWithError(span, err) }()

	return c.Client.GetFirstReleaseTimes(ctx)
}

func (c *tracingClient) GetPipelineBuildsDurations(ctx context.Context, repoSource, repoOwner, repoName string, filters map[string][]string) (durations []map[string]interface{}, err error) {

	span, ctx := opentracing.StartSpanFromContext(ctx, c.getSpanName("GetPipelineBuildsDurations"))
	defer func() { helpers.FinishSpanWithError(span, err) }()

	return c.Client.GetPipelineBuildsDurations(ctx, repoSource, repoOwner, repoName, filters)
}

func (c *tracingClient) GetPipelineReleasesDurations(ctx context.Context, repoSource, repoOwner, repoName string, filters map[string][]string) (durations []map[string]interface{}, err error) {

	span, ctx := opentracing.StartSpanFromContext(ctx, c.getSpanName("GetPipelineReleasesDurations"))
	defer func() { helpers.FinishSpanWithError(span, err) }()

	return c.Client.GetPipelineReleasesDurations(ctx, repoSource, repoOwner, repoName, filters)
}

func (c *tracingClient) GetPipelineBuildsCPUUsageMeasurements(ctx context.Context, repoSource, repoOwner, repoName string, filters map[string][]string) (measurements []map[string]interface{}, err error) {

	span, ctx := opentracing.StartSpanFromContext(ctx, c.getSpanName("GetPipelineBuildsCPUUsageMeasurements"))
	defer func() { helpers.FinishSpanWithError(span, err) }()

	return c.Client.GetPipelineBuildsCPUUsageMeasurements(ctx, repoSource, repoOwner, repoName, filters)
}

func (c *tracingClient) GetPipelineReleasesCPUUsageMeasurements(ctx context.Context, repoSource, repoOwner, repoName string, filters map[string][]string) (measurements []map[string]interface{}, err error) {

	span, ctx := opentracing.StartSpanFromContext(ctx, c.getSpanName("GetPipelineReleasesCPUUsageMeasurements"))
	defer func() { helpers.FinishSpanWithError(span, err) }()

	return c.Client.GetPipelineReleasesCPUUsageMeasurements(ctx, repoSource, repoOwner, repoName, filters)
}

func (c *tracingClient) GetPipelineBuildsMemoryUsageMeasurements(ctx context.Context, repoSource, repoOwner, repoName string, filters map[string][]string) (measurements []map[string]interface{}, err error) {

	span, ctx := opentracing.StartSpanFromContext(ctx, c.getSpanName("GetPipelineBuildsMemoryUsageMeasurements"))
	defer func() { helpers.FinishSpanWithError(span, err) }()

	return c.Client.GetPipelineBuildsMemoryUsageMeasurements(ctx, repoSource, repoOwner, repoName, filters)
}

func (c *tracingClient) GetPipelineReleasesMemoryUsageMeasurements(ctx context.Context, repoSource, repoOwner, repoName string, filters map[string][]string) (measurements []map[string]interface{}, err error) {

	span, ctx := opentracing.StartSpanFromContext(ctx, c.getSpanName("GetPipelineReleasesMemoryUsageMeasurements"))
	defer func() { helpers.FinishSpanWithError(span, err) }()

	return c.Client.GetPipelineReleasesMemoryUsageMeasurements(ctx, repoSource, repoOwner, repoName, filters)
}

func (c *tracingClient) GetFrequentLabels(ctx context.Context, pageNumber, pageSize int, filters map[string][]string) (measurements []map[string]interface{}, err error) {

	span, ctx := opentracing.StartSpanFromContext(ctx, c.getSpanName("GetFrequentLabels"))
	defer func() { helpers.FinishSpanWithError(span, err) }()

	return c.Client.GetFrequentLabels(ctx, pageNumber, pageSize, filters)
}

func (c *tracingClient) GetFrequentLabelsCount(ctx context.Context, filters map[string][]string) (count int, err error) {

	span, ctx := opentracing.StartSpanFromContext(ctx, c.getSpanName("GetFrequentLabelsCount"))
	defer func() { helpers.FinishSpanWithError(span, err) }()

	return c.Client.GetFrequentLabelsCount(ctx, filters)
}

func (c *tracingClient) GetPipelinesWithMostBuilds(ctx context.Context, pageNumber, pageSize int, filters map[string][]string) (pipelines []map[string]interface{}, err error) {

	span, ctx := opentracing.StartSpanFromContext(ctx, c.getSpanName("GetPipelinesWithMostBuilds"))
	defer func() { helpers.FinishSpanWithError(span, err) }()

	return c.Client.GetPipelinesWithMostBuilds(ctx, pageNumber, pageSize, filters)
}

func (c *tracingClient) GetPipelinesWithMostBuildsCount(ctx context.Context, filters map[string][]string) (count int, err error) {

	span, ctx := opentracing.StartSpanFromContext(ctx, c.getSpanName("GetPipelinesWithMostBuildsCount"))
	defer func() { helpers.FinishSpanWithError(span, err) }()

	return c.Client.GetPipelinesWithMostBuildsCount(ctx, filters)
}

func (c *tracingClient) GetPipelinesWithMostReleases(ctx context.Context, pageNumber, pageSize int, filters map[string][]string) (pipelines []map[string]interface{}, err error) {

	span, ctx := opentracing.StartSpanFromContext(ctx, c.getSpanName("GetPipelinesWithMostReleases"))
	defer func() { helpers.FinishSpanWithError(span, err) }()

	return c.Client.GetPipelinesWithMostReleases(ctx, pageNumber, pageSize, filters)
}

func (c *tracingClient) GetPipelinesWithMostReleasesCount(ctx context.Context, filters map[string][]string) (count int, err error) {

	span, ctx := opentracing.StartSpanFromContext(ctx, c.getSpanName("GetPipelinesWithMostReleasesCount"))
	defer func() { helpers.FinishSpanWithError(span, err) }()

	return c.Client.GetPipelinesWithMostReleasesCount(ctx, filters)
}

func (c *tracingClient) GetTriggers(ctx context.Context, triggerType, identifier, event string) (pipelines []*contracts.Pipeline, err error) {

	span, ctx := opentracing.StartSpanFromContext(ctx, c.getSpanName("GetTriggers"))
	defer func() { helpers.FinishSpanWithError(span, err) }()

	return c.Client.GetTriggers(ctx, triggerType, identifier, event)
}

func (c *tracingClient) GetGitTriggers(ctx context.Context, gitEvent manifest.EstafetteGitEvent) (pipelines []*contracts.Pipeline, err error) {

	span, ctx := opentracing.StartSpanFromContext(ctx, c.getSpanName("GetGitTriggers"))
	defer func() { helpers.FinishSpanWithError(span, err) }()

	return c.Client.GetGitTriggers(ctx, gitEvent)
}

func (c *tracingClient) GetPipelineTriggers(ctx context.Context, build contracts.Build, event string) (pipelines []*contracts.Pipeline, err error) {

	span, ctx := opentracing.StartSpanFromContext(ctx, c.getSpanName("GetPipelineTriggers"))
	defer func() { helpers.FinishSpanWithError(span, err) }()

	return c.Client.GetPipelineTriggers(ctx, build, event)
}

func (c *tracingClient) GetReleaseTriggers(ctx context.Context, release contracts.Release, event string) (pipelines []*contracts.Pipeline, err error) {

	span, ctx := opentracing.StartSpanFromContext(ctx, c.getSpanName("GetReleaseTriggers"))
	defer func() { helpers.FinishSpanWithError(span, err) }()

	return c.Client.GetReleaseTriggers(ctx, release, event)
}

func (c *tracingClient) GetPubSubTriggers(ctx context.Context, pubsubEvent manifest.EstafettePubSubEvent) (pipelines []*contracts.Pipeline, err error) {

	span, ctx := opentracing.StartSpanFromContext(ctx, c.getSpanName("GetPubSubTriggers"))
	defer func() { helpers.FinishSpanWithError(span, err) }()

	return c.Client.GetPubSubTriggers(ctx, pubsubEvent)
}

func (c *tracingClient) GetCronTriggers(ctx context.Context) (pipelines []*contracts.Pipeline, err error) {

	span, ctx := opentracing.StartSpanFromContext(ctx, c.getSpanName("GetCronTriggers"))
	defer func() { helpers.FinishSpanWithError(span, err) }()

	return c.Client.GetCronTriggers(ctx)
}

func (c *tracingClient) Rename(ctx context.Context, shortFromRepoSource, fromRepoSource, fromRepoOwner, fromRepoName, shortToRepoSource, toRepoSource, toRepoOwner, toRepoName string) (err error) {

	span, ctx := opentracing.StartSpanFromContext(ctx, c.getSpanName("Rename"))
	defer func() { helpers.FinishSpanWithError(span, err) }()

	return c.Client.Rename(ctx, shortFromRepoSource, fromRepoSource, fromRepoOwner, fromRepoName, shortToRepoSource, toRepoSource, toRepoOwner, toRepoName)
}

func (c *tracingClient) RenameBuildVersion(ctx context.Context, shortFromRepoSource, fromRepoOwner, fromRepoName, shortToRepoSource, toRepoOwner, toRepoName string) (err error) {

	span, ctx := opentracing.StartSpanFromContext(ctx, c.getSpanName("RenameBuildVersion"))
	defer func() { helpers.FinishSpanWithError(span, err) }()

	return c.Client.RenameBuildVersion(ctx, shortFromRepoSource, fromRepoOwner, fromRepoName, shortToRepoSource, toRepoOwner, toRepoName)
}

func (c *tracingClient) RenameBuilds(ctx context.Context, fromRepoSource, fromRepoOwner, fromRepoName, toRepoSource, toRepoOwner, toRepoName string) (err error) {

	span, ctx := opentracing.StartSpanFromContext(ctx, c.getSpanName("RenameBuilds"))
	defer func() { helpers.FinishSpanWithError(span, err) }()

	return c.Client.RenameBuilds(ctx, fromRepoSource, fromRepoOwner, fromRepoName, toRepoSource, toRepoOwner, toRepoName)
}

func (c *tracingClient) RenameBuildLogs(ctx context.Context, fromRepoSource, fromRepoOwner, fromRepoName, toRepoSource, toRepoOwner, toRepoName string) (err error) {

	span, ctx := opentracing.StartSpanFromContext(ctx, c.getSpanName("RenameBuildLogs"))
	defer func() { helpers.FinishSpanWithError(span, err) }()

	return c.Client.RenameBuildLogs(ctx, fromRepoSource, fromRepoOwner, fromRepoName, toRepoSource, toRepoOwner, toRepoName)
}

func (c *tracingClient) RenameReleases(ctx context.Context, fromRepoSource, fromRepoOwner, fromRepoName, toRepoSource, toRepoOwner, toRepoName string) (err error) {

	span, ctx := opentracing.StartSpanFromContext(ctx, c.getSpanName("RenameReleases"))
	defer func() { helpers.FinishSpanWithError(span, err) }()

	return c.Client.RenameReleases(ctx, fromRepoSource, fromRepoOwner, fromRepoName, toRepoSource, toRepoOwner, toRepoName)
}

func (c *tracingClient) RenameReleaseLogs(ctx context.Context, fromRepoSource, fromRepoOwner, fromRepoName, toRepoSource, toRepoOwner, toRepoName string) (err error) {

	span, ctx := opentracing.StartSpanFromContext(ctx, c.getSpanName("RenameReleaseLogs"))
	defer func() { helpers.FinishSpanWithError(span, err) }()

	return c.Client.RenameReleaseLogs(ctx, fromRepoSource, fromRepoOwner, fromRepoName, toRepoSource, toRepoOwner, toRepoName)
}

func (c *tracingClient) RenameComputedPipelines(ctx context.Context, fromRepoSource, fromRepoOwner, fromRepoName, toRepoSource, toRepoOwner, toRepoName string) (err error) {

	span, ctx := opentracing.StartSpanFromContext(ctx, c.getSpanName("RenameComputedPipelines"))
	defer func() { helpers.FinishSpanWithError(span, err) }()

	return c.Client.RenameComputedPipelines(ctx, fromRepoSource, fromRepoOwner, fromRepoName, toRepoSource, toRepoOwner, toRepoName)
}

func (c *tracingClient) RenameComputedReleases(ctx context.Context, fromRepoSource, fromRepoOwner, fromRepoName, toRepoSource, toRepoOwner, toRepoName string) (err error) {

	span, ctx := opentracing.StartSpanFromContext(ctx, c.getSpanName("RenameComputedReleases"))
	defer func() { helpers.FinishSpanWithError(span, err) }()

	return c.Client.RenameComputedReleases(ctx, fromRepoSource, fromRepoOwner, fromRepoName, toRepoSource, toRepoOwner, toRepoName)
}

func (c *tracingClient) getSpanName(funcName string) string {
	return "cockroachdb:" + funcName
}
