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
	return &tracingClient{c, "cockroachdb"}
}

type tracingClient struct {
	Client
	prefix string
}

func (c *tracingClient) Connect(ctx context.Context) (err error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, helpers.GetSpanName(c.prefix, "Connect"))
	defer func() { helpers.FinishSpanWithError(span, err) }()

	return c.Client.Connect(ctx)
}

func (c *tracingClient) ConnectWithDriverAndSource(ctx context.Context, driverName, dataSourceName string) (err error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, helpers.GetSpanName(c.prefix, "ConnectWithDriverAndSource"))
	defer func() { helpers.FinishSpanWithError(span, err) }()

	return c.Client.ConnectWithDriverAndSource(ctx, driverName, dataSourceName)
}

func (c *tracingClient) GetAutoIncrement(ctx context.Context, shortRepoSource, repoOwner, repoName string) (autoincrement int, err error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, helpers.GetSpanName(c.prefix, "GetAutoIncrement"))
	defer func() { helpers.FinishSpanWithError(span, err) }()

	return c.Client.GetAutoIncrement(ctx, shortRepoSource, repoOwner, repoName)
}

func (c *tracingClient) InsertBuild(ctx context.Context, build contracts.Build, jobResources JobResources) (b *contracts.Build, err error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, helpers.GetSpanName(c.prefix, "InsertBuild"))
	defer func() { helpers.FinishSpanWithError(span, err) }()

	return c.Client.InsertBuild(ctx, build, jobResources)
}

func (c *tracingClient) UpdateBuildStatus(ctx context.Context, repoSource, repoOwner, repoName string, buildID int, buildStatus string) (err error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, helpers.GetSpanName(c.prefix, "UpdateBuildStatus"))
	defer func() { helpers.FinishSpanWithError(span, err) }()

	return c.Client.UpdateBuildStatus(ctx, repoSource, repoOwner, repoName, buildID, buildStatus)
}

func (c *tracingClient) UpdateBuildResourceUtilization(ctx context.Context, repoSource, repoOwner, repoName string, buildID int, jobResources JobResources) (err error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, helpers.GetSpanName(c.prefix, "UpdateBuildResourceUtilization"))
	defer func() { helpers.FinishSpanWithError(span, err) }()

	return c.Client.UpdateBuildResourceUtilization(ctx, repoSource, repoOwner, repoName, buildID, jobResources)
}

func (c *tracingClient) InsertRelease(ctx context.Context, release contracts.Release, jobResources JobResources) (r *contracts.Release, err error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, helpers.GetSpanName(c.prefix, "InsertRelease"))
	defer func() { helpers.FinishSpanWithError(span, err) }()

	return c.Client.InsertRelease(ctx, release, jobResources)
}

func (c *tracingClient) UpdateReleaseStatus(ctx context.Context, repoSource, repoOwner, repoName string, id int, releaseStatus string) (err error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, helpers.GetSpanName(c.prefix, "UpdateReleaseStatus"))
	defer func() { helpers.FinishSpanWithError(span, err) }()

	return c.Client.UpdateReleaseStatus(ctx, repoSource, repoOwner, repoName, id, releaseStatus)
}

func (c *tracingClient) UpdateReleaseResourceUtilization(ctx context.Context, repoSource, repoOwner, repoName string, id int, jobResources JobResources) (err error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, helpers.GetSpanName(c.prefix, "UpdateReleaseResourceUtilization"))
	defer func() { helpers.FinishSpanWithError(span, err) }()

	return c.Client.UpdateReleaseResourceUtilization(ctx, repoSource, repoOwner, repoName, id, jobResources)
}

func (c *tracingClient) InsertBuildLog(ctx context.Context, buildLog contracts.BuildLog, writeLogToDatabase bool) (buildlog contracts.BuildLog, err error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, helpers.GetSpanName(c.prefix, "InsertBuildLog"))
	defer func() { helpers.FinishSpanWithError(span, err) }()

	return c.Client.InsertBuildLog(ctx, buildLog, writeLogToDatabase)
}

func (c *tracingClient) InsertReleaseLog(ctx context.Context, releaseLog contracts.ReleaseLog, writeLogToDatabase bool) (releaselog contracts.ReleaseLog, err error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, helpers.GetSpanName(c.prefix, "InsertReleaseLog"))
	defer func() { helpers.FinishSpanWithError(span, err) }()

	return c.Client.InsertReleaseLog(ctx, releaseLog, writeLogToDatabase)
}

func (c *tracingClient) UpsertComputedPipeline(ctx context.Context, repoSource, repoOwner, repoName string) (err error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, helpers.GetSpanName(c.prefix, "UpsertComputedPipeline"))
	defer func() { helpers.FinishSpanWithError(span, err) }()

	return c.Client.UpsertComputedPipeline(ctx, repoSource, repoOwner, repoName)
}

func (c *tracingClient) UpdateComputedPipelineFirstInsertedAt(ctx context.Context, repoSource, repoOwner, repoName string) (err error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, helpers.GetSpanName(c.prefix, "UpdateComputedPipelineFirstInsertedAt"))
	defer func() { helpers.FinishSpanWithError(span, err) }()

	return c.Client.UpdateComputedPipelineFirstInsertedAt(ctx, repoSource, repoOwner, repoName)
}

func (c *tracingClient) UpsertComputedRelease(ctx context.Context, repoSource, repoOwner, repoName, releaseName, releaseAction string) (err error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, helpers.GetSpanName(c.prefix, "UpsertComputedRelease"))
	defer func() { helpers.FinishSpanWithError(span, err) }()

	return c.Client.UpsertComputedRelease(ctx, repoSource, repoOwner, repoName, releaseName, releaseAction)
}

func (c *tracingClient) UpdateComputedReleaseFirstInsertedAt(ctx context.Context, repoSource, repoOwner, repoName, releaseName, releaseAction string) (err error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, helpers.GetSpanName(c.prefix, "UpdateComputedReleaseFirstInsertedAt"))
	defer func() { helpers.FinishSpanWithError(span, err) }()

	return c.Client.UpdateComputedReleaseFirstInsertedAt(ctx, repoSource, repoOwner, repoName, releaseName, releaseAction)
}

func (c *tracingClient) ArchiveComputedPipeline(ctx context.Context, repoSource, repoOwner, repoName string) (err error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, helpers.GetSpanName(c.prefix, "ArchiveComputedPipeline"))
	defer func() { helpers.FinishSpanWithError(span, err) }()

	return c.Client.ArchiveComputedPipeline(ctx, repoSource, repoOwner, repoName)
}

func (c *tracingClient) UnarchiveComputedPipeline(ctx context.Context, repoSource, repoOwner, repoName string) (err error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, helpers.GetSpanName(c.prefix, "UnarchiveComputedPipeline"))
	defer func() { helpers.FinishSpanWithError(span, err) }()

	return c.Client.UnarchiveComputedPipeline(ctx, repoSource, repoOwner, repoName)
}

func (c *tracingClient) GetPipelines(ctx context.Context, pageNumber, pageSize int, filters map[string][]string, sortings []helpers.OrderField, optimized bool) (pipelines []*contracts.Pipeline, err error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, helpers.GetSpanName(c.prefix, "GetPipelines"))
	defer func() { helpers.FinishSpanWithError(span, err) }()

	return c.Client.GetPipelines(ctx, pageNumber, pageSize, filters, sortings, optimized)
}

func (c *tracingClient) GetPipelinesByRepoName(ctx context.Context, repoName string, optimized bool) (pipelines []*contracts.Pipeline, err error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, helpers.GetSpanName(c.prefix, "GetPipelinesByRepoName"))
	defer func() { helpers.FinishSpanWithError(span, err) }()

	return c.Client.GetPipelinesByRepoName(ctx, repoName, optimized)
}

func (c *tracingClient) GetPipelinesCount(ctx context.Context, filters map[string][]string) (count int, err error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, helpers.GetSpanName(c.prefix, "GetPipelinesCount"))
	defer func() { helpers.FinishSpanWithError(span, err) }()

	return c.Client.GetPipelinesCount(ctx, filters)
}

func (c *tracingClient) GetPipeline(ctx context.Context, repoSource, repoOwner, repoName string, optimized bool) (pipeline *contracts.Pipeline, err error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, helpers.GetSpanName(c.prefix, "GetPipeline"))
	defer func() { helpers.FinishSpanWithError(span, err) }()

	return c.Client.GetPipeline(ctx, repoSource, repoOwner, repoName, optimized)
}

func (c *tracingClient) GetPipelineRecentBuilds(ctx context.Context, repoSource, repoOwner, repoName string, optimized bool) (builds []*contracts.Build, err error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, helpers.GetSpanName(c.prefix, "GetPipelineRecentBuilds"))
	defer func() { helpers.FinishSpanWithError(span, err) }()

	return c.Client.GetPipelineRecentBuilds(ctx, repoSource, repoOwner, repoName, optimized)
}

func (c *tracingClient) GetPipelineBuilds(ctx context.Context, repoSource, repoOwner, repoName string, pageNumber, pageSize int, filters map[string][]string, sortings []helpers.OrderField, optimized bool) (builds []*contracts.Build, err error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, helpers.GetSpanName(c.prefix, "GetPipelineBuilds"))
	defer func() { helpers.FinishSpanWithError(span, err) }()

	return c.Client.GetPipelineBuilds(ctx, repoSource, repoOwner, repoName, pageNumber, pageSize, filters, sortings, optimized)
}

func (c *tracingClient) GetPipelineBuildsCount(ctx context.Context, repoSource, repoOwner, repoName string, filters map[string][]string) (count int, err error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, helpers.GetSpanName(c.prefix, "GetPipelineBuildsCount"))
	defer func() { helpers.FinishSpanWithError(span, err) }()

	return c.Client.GetPipelineBuildsCount(ctx, repoSource, repoOwner, repoName, filters)
}

func (c *tracingClient) GetPipelineBuild(ctx context.Context, repoSource, repoOwner, repoName, repoRevision string, optimized bool) (build *contracts.Build, err error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, helpers.GetSpanName(c.prefix, "GetPipelineBuild"))
	defer func() { helpers.FinishSpanWithError(span, err) }()

	return c.Client.GetPipelineBuild(ctx, repoSource, repoOwner, repoName, repoRevision, optimized)
}

func (c *tracingClient) GetPipelineBuildByID(ctx context.Context, repoSource, repoOwner, repoName string, id int, optimized bool) (build *contracts.Build, err error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, helpers.GetSpanName(c.prefix, "GetPipelineBuildByID"))
	defer func() { helpers.FinishSpanWithError(span, err) }()

	return c.Client.GetPipelineBuildByID(ctx, repoSource, repoOwner, repoName, id, optimized)
}

func (c *tracingClient) GetLastPipelineBuild(ctx context.Context, repoSource, repoOwner, repoName string, optimized bool) (build *contracts.Build, err error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, helpers.GetSpanName(c.prefix, "GetLastPipelineBuild"))
	defer func() { helpers.FinishSpanWithError(span, err) }()

	return c.Client.GetLastPipelineBuild(ctx, repoSource, repoOwner, repoName, optimized)
}

func (c *tracingClient) GetFirstPipelineBuild(ctx context.Context, repoSource, repoOwner, repoName string, optimized bool) (build *contracts.Build, err error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, helpers.GetSpanName(c.prefix, "GetFirstPipelineBuild"))
	defer func() { helpers.FinishSpanWithError(span, err) }()

	return c.Client.GetFirstPipelineBuild(ctx, repoSource, repoOwner, repoName, optimized)
}

func (c *tracingClient) GetLastPipelineBuildForBranch(ctx context.Context, repoSource, repoOwner, repoName, branch string) (build *contracts.Build, err error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, helpers.GetSpanName(c.prefix, "GetLastPipelineBuildForBranch"))
	defer func() { helpers.FinishSpanWithError(span, err) }()

	return c.Client.GetLastPipelineBuildForBranch(ctx, repoSource, repoOwner, repoName, branch)
}

func (c *tracingClient) GetLastPipelineRelease(ctx context.Context, repoSource, repoOwner, repoName, releaseName, releaseAction string) (release *contracts.Release, err error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, helpers.GetSpanName(c.prefix, "GetLastPipelineRelease"))
	defer func() { helpers.FinishSpanWithError(span, err) }()

	return c.Client.GetLastPipelineRelease(ctx, repoSource, repoOwner, repoName, releaseName, releaseAction)
}

func (c *tracingClient) GetFirstPipelineRelease(ctx context.Context, repoSource, repoOwner, repoName, releaseName, releaseAction string) (release *contracts.Release, err error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, helpers.GetSpanName(c.prefix, "GetFirstPipelineRelease"))
	defer func() { helpers.FinishSpanWithError(span, err) }()

	return c.Client.GetFirstPipelineRelease(ctx, repoSource, repoOwner, repoName, releaseName, releaseAction)
}

func (c *tracingClient) GetPipelineBuildsByVersion(ctx context.Context, repoSource, repoOwner, repoName, buildVersion string, statuses []string, limit uint64, optimized bool) (builds []*contracts.Build, err error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, helpers.GetSpanName(c.prefix, "GetPipelineBuildsByVersion"))
	defer func() { helpers.FinishSpanWithError(span, err) }()

	return c.Client.GetPipelineBuildsByVersion(ctx, repoSource, repoOwner, repoName, buildVersion, statuses, limit, optimized)
}

func (c *tracingClient) GetPipelineBuildLogs(ctx context.Context, repoSource, repoOwner, repoName, repoBranch, repoRevision, buildID string, readLogFromDatabase bool) (buildlog *contracts.BuildLog, err error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, helpers.GetSpanName(c.prefix, "GetPipelineBuildLogs"))
	defer func() { helpers.FinishSpanWithError(span, err) }()

	return c.Client.GetPipelineBuildLogs(ctx, repoSource, repoOwner, repoName, repoBranch, repoRevision, buildID, readLogFromDatabase)
}

func (c *tracingClient) GetPipelineBuildLogsPerPage(ctx context.Context, repoSource, repoOwner, repoName string, pageNumber int, pageSize int) (buildlogs []*contracts.BuildLog, err error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, helpers.GetSpanName(c.prefix, "GetPipelineBuildLogsPerPage"))
	defer func() { helpers.FinishSpanWithError(span, err) }()

	return c.Client.GetPipelineBuildLogsPerPage(ctx, repoSource, repoOwner, repoName, pageNumber, pageSize)
}

func (c *tracingClient) GetPipelineBuildMaxResourceUtilization(ctx context.Context, repoSource, repoOwner, repoName string, lastNRecords int) (jobresources JobResources, count int, err error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, helpers.GetSpanName(c.prefix, "GetPipelineBuildMaxResourceUtilization"))
	defer func() { helpers.FinishSpanWithError(span, err) }()

	return c.Client.GetPipelineBuildMaxResourceUtilization(ctx, repoSource, repoOwner, repoName, lastNRecords)
}

func (c *tracingClient) GetPipelineReleases(ctx context.Context, repoSource, repoOwner, repoName string, pageNumber, pageSize int, filters map[string][]string, sortings []helpers.OrderField) (releases []*contracts.Release, err error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, helpers.GetSpanName(c.prefix, "GetPipelineReleases"))
	defer func() { helpers.FinishSpanWithError(span, err) }()

	return c.Client.GetPipelineReleases(ctx, repoSource, repoOwner, repoName, pageNumber, pageSize, filters, sortings)
}

func (c *tracingClient) GetPipelineReleasesCount(ctx context.Context, repoSource, repoOwner, repoName string, filters map[string][]string) (count int, err error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, helpers.GetSpanName(c.prefix, "GetPipelineReleasesCount"))
	defer func() { helpers.FinishSpanWithError(span, err) }()

	return c.Client.GetPipelineReleasesCount(ctx, repoSource, repoOwner, repoName, filters)
}

func (c *tracingClient) GetPipelineRelease(ctx context.Context, repoSource, repoOwner, repoName string, id int) (release *contracts.Release, err error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, helpers.GetSpanName(c.prefix, "GetPipelineRelease"))
	defer func() { helpers.FinishSpanWithError(span, err) }()

	return c.Client.GetPipelineRelease(ctx, repoSource, repoOwner, repoName, id)
}

func (c *tracingClient) GetPipelineLastReleasesByName(ctx context.Context, repoSource, repoOwner, repoName, releaseName string, actions []string) (releases []contracts.Release, err error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, helpers.GetSpanName(c.prefix, "GetPipelineLastReleasesByName"))
	defer func() { helpers.FinishSpanWithError(span, err) }()

	return c.Client.GetPipelineLastReleasesByName(ctx, repoSource, repoOwner, repoName, releaseName, actions)
}

func (c *tracingClient) GetPipelineReleaseLogs(ctx context.Context, repoSource, repoOwner, repoName string, id int, readLogFromDatabase bool) (releaselog *contracts.ReleaseLog, err error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, helpers.GetSpanName(c.prefix, "GetPipelineReleaseLogs"))
	defer func() { helpers.FinishSpanWithError(span, err) }()

	return c.Client.GetPipelineReleaseLogs(ctx, repoSource, repoOwner, repoName, id, readLogFromDatabase)
}

func (c *tracingClient) GetPipelineReleaseLogsPerPage(ctx context.Context, repoSource, repoOwner, repoName string, pageNumber int, pageSize int) (releaselogs []*contracts.ReleaseLog, err error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, helpers.GetSpanName(c.prefix, "GetPipelineReleaseLogsPerPage"))
	defer func() { helpers.FinishSpanWithError(span, err) }()

	return c.Client.GetPipelineReleaseLogsPerPage(ctx, repoSource, repoOwner, repoName, pageNumber, pageSize)
}

func (c *tracingClient) GetPipelineReleaseMaxResourceUtilization(ctx context.Context, repoSource, repoOwner, repoName, targetName string, lastNRecords int) (jobresources JobResources, count int, err error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, helpers.GetSpanName(c.prefix, "GetPipelineReleaseMaxResourceUtilization"))
	defer func() { helpers.FinishSpanWithError(span, err) }()

	return c.Client.GetPipelineReleaseMaxResourceUtilization(ctx, repoSource, repoOwner, repoName, targetName, lastNRecords)
}

func (c *tracingClient) GetBuildsCount(ctx context.Context, filters map[string][]string) (count int, err error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, helpers.GetSpanName(c.prefix, "GetBuildsCount"))
	defer func() { helpers.FinishSpanWithError(span, err) }()

	return c.Client.GetBuildsCount(ctx, filters)
}

func (c *tracingClient) GetReleasesCount(ctx context.Context, filters map[string][]string) (count int, err error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, helpers.GetSpanName(c.prefix, "GetReleasesCount"))
	defer func() { helpers.FinishSpanWithError(span, err) }()

	return c.Client.GetReleasesCount(ctx, filters)
}

func (c *tracingClient) GetBuildsDuration(ctx context.Context, filters map[string][]string) (duration time.Duration, err error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, helpers.GetSpanName(c.prefix, "GetBuildsDuration"))
	defer func() { helpers.FinishSpanWithError(span, err) }()

	return c.Client.GetBuildsDuration(ctx, filters)
}

func (c *tracingClient) GetFirstBuildTimes(ctx context.Context) (times []time.Time, err error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, helpers.GetSpanName(c.prefix, "GetFirstBuildTimes"))
	defer func() { helpers.FinishSpanWithError(span, err) }()

	return c.Client.GetFirstBuildTimes(ctx)
}

func (c *tracingClient) GetFirstReleaseTimes(ctx context.Context) (times []time.Time, err error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, helpers.GetSpanName(c.prefix, "GetFirstReleaseTimes"))
	defer func() { helpers.FinishSpanWithError(span, err) }()

	return c.Client.GetFirstReleaseTimes(ctx)
}

func (c *tracingClient) GetPipelineBuildsDurations(ctx context.Context, repoSource, repoOwner, repoName string, filters map[string][]string) (durations []map[string]interface{}, err error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, helpers.GetSpanName(c.prefix, "GetPipelineBuildsDurations"))
	defer func() { helpers.FinishSpanWithError(span, err) }()

	return c.Client.GetPipelineBuildsDurations(ctx, repoSource, repoOwner, repoName, filters)
}

func (c *tracingClient) GetPipelineReleasesDurations(ctx context.Context, repoSource, repoOwner, repoName string, filters map[string][]string) (durations []map[string]interface{}, err error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, helpers.GetSpanName(c.prefix, "GetPipelineReleasesDurations"))
	defer func() { helpers.FinishSpanWithError(span, err) }()

	return c.Client.GetPipelineReleasesDurations(ctx, repoSource, repoOwner, repoName, filters)
}

func (c *tracingClient) GetPipelineBuildsCPUUsageMeasurements(ctx context.Context, repoSource, repoOwner, repoName string, filters map[string][]string) (measurements []map[string]interface{}, err error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, helpers.GetSpanName(c.prefix, "GetPipelineBuildsCPUUsageMeasurements"))
	defer func() { helpers.FinishSpanWithError(span, err) }()

	return c.Client.GetPipelineBuildsCPUUsageMeasurements(ctx, repoSource, repoOwner, repoName, filters)
}

func (c *tracingClient) GetPipelineReleasesCPUUsageMeasurements(ctx context.Context, repoSource, repoOwner, repoName string, filters map[string][]string) (measurements []map[string]interface{}, err error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, helpers.GetSpanName(c.prefix, "GetPipelineReleasesCPUUsageMeasurements"))
	defer func() { helpers.FinishSpanWithError(span, err) }()

	return c.Client.GetPipelineReleasesCPUUsageMeasurements(ctx, repoSource, repoOwner, repoName, filters)
}

func (c *tracingClient) GetPipelineBuildsMemoryUsageMeasurements(ctx context.Context, repoSource, repoOwner, repoName string, filters map[string][]string) (measurements []map[string]interface{}, err error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, helpers.GetSpanName(c.prefix, "GetPipelineBuildsMemoryUsageMeasurements"))
	defer func() { helpers.FinishSpanWithError(span, err) }()

	return c.Client.GetPipelineBuildsMemoryUsageMeasurements(ctx, repoSource, repoOwner, repoName, filters)
}

func (c *tracingClient) GetPipelineReleasesMemoryUsageMeasurements(ctx context.Context, repoSource, repoOwner, repoName string, filters map[string][]string) (measurements []map[string]interface{}, err error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, helpers.GetSpanName(c.prefix, "GetPipelineReleasesMemoryUsageMeasurements"))
	defer func() { helpers.FinishSpanWithError(span, err) }()

	return c.Client.GetPipelineReleasesMemoryUsageMeasurements(ctx, repoSource, repoOwner, repoName, filters)
}

func (c *tracingClient) GetLabelValues(ctx context.Context, labelKey string) (labels []map[string]interface{}, err error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, helpers.GetSpanName(c.prefix, "GetLabelValues"))
	defer func() { helpers.FinishSpanWithError(span, err) }()

	return c.Client.GetLabelValues(ctx, labelKey)
}

func (c *tracingClient) GetFrequentLabels(ctx context.Context, pageNumber, pageSize int, filters map[string][]string) (measurements []map[string]interface{}, err error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, helpers.GetSpanName(c.prefix, "GetFrequentLabels"))
	defer func() { helpers.FinishSpanWithError(span, err) }()

	return c.Client.GetFrequentLabels(ctx, pageNumber, pageSize, filters)
}

func (c *tracingClient) GetFrequentLabelsCount(ctx context.Context, filters map[string][]string) (count int, err error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, helpers.GetSpanName(c.prefix, "GetFrequentLabelsCount"))
	defer func() { helpers.FinishSpanWithError(span, err) }()

	return c.Client.GetFrequentLabelsCount(ctx, filters)
}

func (c *tracingClient) GetPipelinesWithMostBuilds(ctx context.Context, pageNumber, pageSize int, filters map[string][]string) (pipelines []map[string]interface{}, err error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, helpers.GetSpanName(c.prefix, "GetPipelinesWithMostBuilds"))
	defer func() { helpers.FinishSpanWithError(span, err) }()

	return c.Client.GetPipelinesWithMostBuilds(ctx, pageNumber, pageSize, filters)
}

func (c *tracingClient) GetPipelinesWithMostBuildsCount(ctx context.Context, filters map[string][]string) (count int, err error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, helpers.GetSpanName(c.prefix, "GetPipelinesWithMostBuildsCount"))
	defer func() { helpers.FinishSpanWithError(span, err) }()

	return c.Client.GetPipelinesWithMostBuildsCount(ctx, filters)
}

func (c *tracingClient) GetPipelinesWithMostReleases(ctx context.Context, pageNumber, pageSize int, filters map[string][]string) (pipelines []map[string]interface{}, err error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, helpers.GetSpanName(c.prefix, "GetPipelinesWithMostReleases"))
	defer func() { helpers.FinishSpanWithError(span, err) }()

	return c.Client.GetPipelinesWithMostReleases(ctx, pageNumber, pageSize, filters)
}

func (c *tracingClient) GetPipelinesWithMostReleasesCount(ctx context.Context, filters map[string][]string) (count int, err error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, helpers.GetSpanName(c.prefix, "GetPipelinesWithMostReleasesCount"))
	defer func() { helpers.FinishSpanWithError(span, err) }()

	return c.Client.GetPipelinesWithMostReleasesCount(ctx, filters)
}

func (c *tracingClient) GetTriggers(ctx context.Context, triggerType, identifier, event string) (pipelines []*contracts.Pipeline, err error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, helpers.GetSpanName(c.prefix, "GetTriggers"))
	defer func() { helpers.FinishSpanWithError(span, err) }()

	return c.Client.GetTriggers(ctx, triggerType, identifier, event)
}

func (c *tracingClient) GetGitTriggers(ctx context.Context, gitEvent manifest.EstafetteGitEvent) (pipelines []*contracts.Pipeline, err error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, helpers.GetSpanName(c.prefix, "GetGitTriggers"))
	defer func() { helpers.FinishSpanWithError(span, err) }()

	return c.Client.GetGitTriggers(ctx, gitEvent)
}

func (c *tracingClient) GetPipelineTriggers(ctx context.Context, build contracts.Build, event string) (pipelines []*contracts.Pipeline, err error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, helpers.GetSpanName(c.prefix, "GetPipelineTriggers"))
	defer func() { helpers.FinishSpanWithError(span, err) }()

	return c.Client.GetPipelineTriggers(ctx, build, event)
}

func (c *tracingClient) GetReleaseTriggers(ctx context.Context, release contracts.Release, event string) (pipelines []*contracts.Pipeline, err error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, helpers.GetSpanName(c.prefix, "GetReleaseTriggers"))
	defer func() { helpers.FinishSpanWithError(span, err) }()

	return c.Client.GetReleaseTriggers(ctx, release, event)
}

func (c *tracingClient) GetPubSubTriggers(ctx context.Context, pubsubEvent manifest.EstafettePubSubEvent) (pipelines []*contracts.Pipeline, err error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, helpers.GetSpanName(c.prefix, "GetPubSubTriggers"))
	defer func() { helpers.FinishSpanWithError(span, err) }()

	return c.Client.GetPubSubTriggers(ctx, pubsubEvent)
}

func (c *tracingClient) GetCronTriggers(ctx context.Context) (pipelines []*contracts.Pipeline, err error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, helpers.GetSpanName(c.prefix, "GetCronTriggers"))
	defer func() { helpers.FinishSpanWithError(span, err) }()

	return c.Client.GetCronTriggers(ctx)
}

func (c *tracingClient) Rename(ctx context.Context, shortFromRepoSource, fromRepoSource, fromRepoOwner, fromRepoName, shortToRepoSource, toRepoSource, toRepoOwner, toRepoName string) (err error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, helpers.GetSpanName(c.prefix, "Rename"))
	defer func() { helpers.FinishSpanWithError(span, err) }()

	return c.Client.Rename(ctx, shortFromRepoSource, fromRepoSource, fromRepoOwner, fromRepoName, shortToRepoSource, toRepoSource, toRepoOwner, toRepoName)
}

func (c *tracingClient) RenameBuildVersion(ctx context.Context, shortFromRepoSource, fromRepoOwner, fromRepoName, shortToRepoSource, toRepoOwner, toRepoName string) (err error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, helpers.GetSpanName(c.prefix, "RenameBuildVersion"))
	defer func() { helpers.FinishSpanWithError(span, err) }()

	return c.Client.RenameBuildVersion(ctx, shortFromRepoSource, fromRepoOwner, fromRepoName, shortToRepoSource, toRepoOwner, toRepoName)
}

func (c *tracingClient) RenameBuilds(ctx context.Context, fromRepoSource, fromRepoOwner, fromRepoName, toRepoSource, toRepoOwner, toRepoName string) (err error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, helpers.GetSpanName(c.prefix, "RenameBuilds"))
	defer func() { helpers.FinishSpanWithError(span, err) }()

	return c.Client.RenameBuilds(ctx, fromRepoSource, fromRepoOwner, fromRepoName, toRepoSource, toRepoOwner, toRepoName)
}

func (c *tracingClient) RenameBuildLogs(ctx context.Context, fromRepoSource, fromRepoOwner, fromRepoName, toRepoSource, toRepoOwner, toRepoName string) (err error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, helpers.GetSpanName(c.prefix, "RenameBuildLogs"))
	defer func() { helpers.FinishSpanWithError(span, err) }()

	return c.Client.RenameBuildLogs(ctx, fromRepoSource, fromRepoOwner, fromRepoName, toRepoSource, toRepoOwner, toRepoName)
}

func (c *tracingClient) RenameReleases(ctx context.Context, fromRepoSource, fromRepoOwner, fromRepoName, toRepoSource, toRepoOwner, toRepoName string) (err error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, helpers.GetSpanName(c.prefix, "RenameReleases"))
	defer func() { helpers.FinishSpanWithError(span, err) }()

	return c.Client.RenameReleases(ctx, fromRepoSource, fromRepoOwner, fromRepoName, toRepoSource, toRepoOwner, toRepoName)
}

func (c *tracingClient) RenameReleaseLogs(ctx context.Context, fromRepoSource, fromRepoOwner, fromRepoName, toRepoSource, toRepoOwner, toRepoName string) (err error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, helpers.GetSpanName(c.prefix, "RenameReleaseLogs"))
	defer func() { helpers.FinishSpanWithError(span, err) }()

	return c.Client.RenameReleaseLogs(ctx, fromRepoSource, fromRepoOwner, fromRepoName, toRepoSource, toRepoOwner, toRepoName)
}

func (c *tracingClient) RenameComputedPipelines(ctx context.Context, fromRepoSource, fromRepoOwner, fromRepoName, toRepoSource, toRepoOwner, toRepoName string) (err error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, helpers.GetSpanName(c.prefix, "RenameComputedPipelines"))
	defer func() { helpers.FinishSpanWithError(span, err) }()

	return c.Client.RenameComputedPipelines(ctx, fromRepoSource, fromRepoOwner, fromRepoName, toRepoSource, toRepoOwner, toRepoName)
}

func (c *tracingClient) RenameComputedReleases(ctx context.Context, fromRepoSource, fromRepoOwner, fromRepoName, toRepoSource, toRepoOwner, toRepoName string) (err error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, helpers.GetSpanName(c.prefix, "RenameComputedReleases"))
	defer func() { helpers.FinishSpanWithError(span, err) }()

	return c.Client.RenameComputedReleases(ctx, fromRepoSource, fromRepoOwner, fromRepoName, toRepoSource, toRepoOwner, toRepoName)
}

func (c *tracingClient) InsertUser(ctx context.Context, user contracts.User) (u *contracts.User, err error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, helpers.GetSpanName(c.prefix, "InsertUser"))
	defer func() { helpers.FinishSpanWithError(span, err) }()

	return c.Client.InsertUser(ctx, user)
}

func (c *tracingClient) UpdateUser(ctx context.Context, user contracts.User) (err error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, helpers.GetSpanName(c.prefix, "UpdateUser"))
	defer func() { helpers.FinishSpanWithError(span, err) }()

	return c.Client.UpdateUser(ctx, user)
}

func (c *tracingClient) GetUserByIdentity(ctx context.Context, identity contracts.UserIdentity) (user *contracts.User, err error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, helpers.GetSpanName(c.prefix, "GetUserByIdentity"))
	defer func() { helpers.FinishSpanWithError(span, err) }()

	return c.Client.GetUserByIdentity(ctx, identity)
}

func (c *tracingClient) GetUserByID(ctx context.Context, id string) (user *contracts.User, err error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, helpers.GetSpanName(c.prefix, "GetUserByID"))
	defer func() { helpers.FinishSpanWithError(span, err) }()

	return c.Client.GetUserByID(ctx, id)
}

func (c *tracingClient) GetUsers(ctx context.Context, pageNumber, pageSize int, filters map[string][]string, sortings []helpers.OrderField) (users []*contracts.User, err error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, helpers.GetSpanName(c.prefix, "GetUsers"))
	defer func() { helpers.FinishSpanWithError(span, err) }()

	return c.Client.GetUsers(ctx, pageNumber, pageSize, filters, sortings)
}

func (c *tracingClient) GetUsersCount(ctx context.Context, filters map[string][]string) (count int, err error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, helpers.GetSpanName(c.prefix, "GetUsersCount"))
	defer func() { helpers.FinishSpanWithError(span, err) }()

	return c.Client.GetUsersCount(ctx, filters)
}

func (c *tracingClient) InsertGroup(ctx context.Context, group contracts.Group) (g *contracts.Group, err error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, helpers.GetSpanName(c.prefix, "InsertGroup"))
	defer func() { helpers.FinishSpanWithError(span, err) }()

	return c.Client.InsertGroup(ctx, group)
}

func (c *tracingClient) UpdateGroup(ctx context.Context, group contracts.Group) (err error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, helpers.GetSpanName(c.prefix, "UpdateGroup"))
	defer func() { helpers.FinishSpanWithError(span, err) }()

	return c.Client.UpdateGroup(ctx, group)
}

func (c *tracingClient) GetGroupByIdentity(ctx context.Context, identity contracts.GroupIdentity) (group *contracts.Group, err error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, helpers.GetSpanName(c.prefix, "GetGroupByIdentity"))
	defer func() { helpers.FinishSpanWithError(span, err) }()

	return c.Client.GetGroupByIdentity(ctx, identity)
}

func (c *tracingClient) GetGroupByID(ctx context.Context, id string) (group *contracts.Group, err error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, helpers.GetSpanName(c.prefix, "GetGroupByID"))
	defer func() { helpers.FinishSpanWithError(span, err) }()

	return c.Client.GetGroupByID(ctx, id)
}

func (c *tracingClient) GetGroups(ctx context.Context, pageNumber, pageSize int, filters map[string][]string, sortings []helpers.OrderField) (groups []*contracts.Group, err error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, helpers.GetSpanName(c.prefix, "GetGroups"))
	defer func() { helpers.FinishSpanWithError(span, err) }()

	return c.Client.GetGroups(ctx, pageNumber, pageSize, filters, sortings)
}

func (c *tracingClient) GetGroupsCount(ctx context.Context, filters map[string][]string) (count int, err error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, helpers.GetSpanName(c.prefix, "GetGroupsCount"))
	defer func() { helpers.FinishSpanWithError(span, err) }()

	return c.Client.GetGroupsCount(ctx, filters)
}

func (c *tracingClient) InsertOrganization(ctx context.Context, organization contracts.Organization) (o *contracts.Organization, err error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, helpers.GetSpanName(c.prefix, "InsertOrganization"))
	defer func() { helpers.FinishSpanWithError(span, err) }()

	return c.Client.InsertOrganization(ctx, organization)
}

func (c *tracingClient) UpdateOrganization(ctx context.Context, organization contracts.Organization) (err error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, helpers.GetSpanName(c.prefix, "UpdateOrganization"))
	defer func() { helpers.FinishSpanWithError(span, err) }()

	return c.Client.UpdateOrganization(ctx, organization)
}

func (c *tracingClient) GetOrganizationByIdentity(ctx context.Context, identity contracts.OrganizationIdentity) (organization *contracts.Organization, err error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, helpers.GetSpanName(c.prefix, "GetOrganizationByIdentity"))
	defer func() { helpers.FinishSpanWithError(span, err) }()

	return c.Client.GetOrganizationByIdentity(ctx, identity)
}

func (c *tracingClient) GetOrganizationByID(ctx context.Context, id string) (organization *contracts.Organization, err error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, helpers.GetSpanName(c.prefix, "GetOrganizationByID"))
	defer func() { helpers.FinishSpanWithError(span, err) }()

	return c.Client.GetOrganizationByID(ctx, id)
}

func (c *tracingClient) GetOrganizationByName(ctx context.Context, name string) (organization *contracts.Organization, err error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, helpers.GetSpanName(c.prefix, "GetOrganizationByName"))
	defer func() { helpers.FinishSpanWithError(span, err) }()

	return c.Client.GetOrganizationByName(ctx, name)
}

func (c *tracingClient) GetOrganizations(ctx context.Context, pageNumber, pageSize int, filters map[string][]string, sortings []helpers.OrderField) (organizations []*contracts.Organization, err error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, helpers.GetSpanName(c.prefix, "GetOrganizations"))
	defer func() { helpers.FinishSpanWithError(span, err) }()

	return c.Client.GetOrganizations(ctx, pageNumber, pageSize, filters, sortings)
}

func (c *tracingClient) GetOrganizationsCount(ctx context.Context, filters map[string][]string) (count int, err error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, helpers.GetSpanName(c.prefix, "GetOrganizationsCount"))
	defer func() { helpers.FinishSpanWithError(span, err) }()

	return c.Client.GetOrganizationsCount(ctx, filters)
}

func (c *tracingClient) InsertClient(ctx context.Context, client contracts.Client) (cl *contracts.Client, err error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, helpers.GetSpanName(c.prefix, "InsertClient"))
	defer func() { helpers.FinishSpanWithError(span, err) }()

	return c.Client.InsertClient(ctx, client)
}

func (c *tracingClient) UpdateClient(ctx context.Context, client contracts.Client) (err error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, helpers.GetSpanName(c.prefix, "UpdateClient"))
	defer func() { helpers.FinishSpanWithError(span, err) }()

	return c.Client.UpdateClient(ctx, client)
}

func (c *tracingClient) GetClientByClientID(ctx context.Context, clientID string) (client *contracts.Client, err error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, helpers.GetSpanName(c.prefix, "GetClientByClientID"))
	defer func() { helpers.FinishSpanWithError(span, err) }()

	return c.Client.GetClientByClientID(ctx, clientID)
}

func (c *tracingClient) GetClientByID(ctx context.Context, id string) (client *contracts.Client, err error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, helpers.GetSpanName(c.prefix, "GetClientByID"))
	defer func() { helpers.FinishSpanWithError(span, err) }()

	return c.Client.GetClientByID(ctx, id)
}

func (c *tracingClient) GetClients(ctx context.Context, pageNumber, pageSize int, filters map[string][]string, sortings []helpers.OrderField) (clients []*contracts.Client, err error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, helpers.GetSpanName(c.prefix, "GetClients"))
	defer func() { helpers.FinishSpanWithError(span, err) }()

	return c.Client.GetClients(ctx, pageNumber, pageSize, filters, sortings)
}

func (c *tracingClient) GetClientsCount(ctx context.Context, filters map[string][]string) (count int, err error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, helpers.GetSpanName(c.prefix, "GetClientsCount"))
	defer func() { helpers.FinishSpanWithError(span, err) }()

	return c.Client.GetClientsCount(ctx, filters)
}
