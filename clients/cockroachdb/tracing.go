package cockroachdb

import (
	"context"
	"time"

	contracts "github.com/estafette/estafette-ci-contracts"
	manifest "github.com/estafette/estafette-ci-manifest"
	"github.com/opentracing/opentracing-go"
	"github.com/opentracing/opentracing-go/ext"
	"github.com/opentracing/opentracing-go/log"
)

type tracingClient struct {
	Client
}

// NewTracingClient returns a new instance of a tracing Client.
func NewTracingClient(c Client) Client {
	return &tracingClient{c}
}

func (c *tracingClient) Connect(ctx context.Context) error {

	span, ctx := opentracing.StartSpanFromContext(ctx, c.getSpanName("Connect"))
	defer span.Finish()

	return c.handleError(span, c.Client.Connect(ctx))
}

func (c *tracingClient) ConnectWithDriverAndSource(ctx context.Context, driverName, dataSourceName string) error {

	span, ctx := opentracing.StartSpanFromContext(ctx, c.getSpanName("ConnectWithDriverAndSource"))
	defer span.Finish()

	return c.handleError(span, c.Client.ConnectWithDriverAndSource(ctx, driverName, dataSourceName))
}

func (c *tracingClient) GetAutoIncrement(ctx context.Context, shortRepoSource, repoOwner, repoName string) (int, error) {

	span, ctx := opentracing.StartSpanFromContext(ctx, c.getSpanName("GetAutoIncrement"))
	defer span.Finish()

	autoincrement, err := c.Client.GetAutoIncrement(ctx, shortRepoSource, repoOwner, repoName)
	c.handleError(span, err)

	return autoincrement, err
}

func (c *tracingClient) InsertBuild(ctx context.Context, build contracts.Build, jobResources JobResources) (*contracts.Build, error) {

	span, ctx := opentracing.StartSpanFromContext(ctx, c.getSpanName("InsertBuild"))
	defer span.Finish()

	b, err := c.Client.InsertBuild(ctx, build, jobResources)
	c.handleError(span, err)

	return b, err
}

func (c *tracingClient) UpdateBuildStatus(ctx context.Context, repoSource, repoOwner, repoName string, buildID int, buildStatus string) error {

	span, ctx := opentracing.StartSpanFromContext(ctx, c.getSpanName("UpdateBuildStatus"))
	defer span.Finish()

	return c.handleError(span, c.Client.UpdateBuildStatus(ctx, repoSource, repoOwner, repoName, buildID, buildStatus))
}

func (c *tracingClient) UpdateBuildResourceUtilization(ctx context.Context, repoSource, repoOwner, repoName string, buildID int, jobResources JobResources) error {

	span, ctx := opentracing.StartSpanFromContext(ctx, c.getSpanName("UpdateBuildResourceUtilization"))
	defer span.Finish()

	return c.handleError(span, c.Client.UpdateBuildResourceUtilization(ctx, repoSource, repoOwner, repoName, buildID, jobResources))
}

func (c *tracingClient) InsertRelease(ctx context.Context, release contracts.Release, jobResources JobResources) (*contracts.Release, error) {

	span, ctx := opentracing.StartSpanFromContext(ctx, c.getSpanName("InsertRelease"))
	defer span.Finish()

	r, err := c.Client.InsertRelease(ctx, release, jobResources)
	c.handleError(span, err)

	return r, err
}

func (c *tracingClient) UpdateReleaseStatus(ctx context.Context, repoSource, repoOwner, repoName string, id int, releaseStatus string) error {

	span, ctx := opentracing.StartSpanFromContext(ctx, c.getSpanName("UpdateReleaseStatus"))
	defer span.Finish()

	return c.handleError(span, c.Client.UpdateReleaseStatus(ctx, repoSource, repoOwner, repoName, id, releaseStatus))
}

func (c *tracingClient) UpdateReleaseResourceUtilization(ctx context.Context, repoSource, repoOwner, repoName string, id int, jobResources JobResources) error {

	span, ctx := opentracing.StartSpanFromContext(ctx, c.getSpanName("UpdateReleaseResourceUtilization"))
	defer span.Finish()

	return c.handleError(span, c.Client.UpdateReleaseResourceUtilization(ctx, repoSource, repoOwner, repoName, id, jobResources))
}

func (c *tracingClient) InsertBuildLog(ctx context.Context, buildLog contracts.BuildLog, writeLogToDatabase bool) (contracts.BuildLog, error) {

	span, ctx := opentracing.StartSpanFromContext(ctx, c.getSpanName("InsertBuildLog"))
	defer span.Finish()

	buildlog, err := c.Client.InsertBuildLog(ctx, buildLog, writeLogToDatabase)
	c.handleError(span, err)

	return buildlog, err
}

func (c *tracingClient) InsertReleaseLog(ctx context.Context, releaseLog contracts.ReleaseLog, writeLogToDatabase bool) (contracts.ReleaseLog, error) {

	span, ctx := opentracing.StartSpanFromContext(ctx, c.getSpanName("InsertReleaseLog"))
	defer span.Finish()

	releaselog, err := c.Client.InsertReleaseLog(ctx, releaseLog, writeLogToDatabase)
	c.handleError(span, err)

	return releaselog, err
}

func (c *tracingClient) UpsertComputedPipeline(ctx context.Context, repoSource, repoOwner, repoName string) error {

	span, ctx := opentracing.StartSpanFromContext(ctx, c.getSpanName("UpsertComputedPipeline"))
	defer span.Finish()

	return c.handleError(span, c.Client.UpsertComputedPipeline(ctx, repoSource, repoOwner, repoName))
}

func (c *tracingClient) UpdateComputedPipelineFirstInsertedAt(ctx context.Context, repoSource, repoOwner, repoName string) error {

	span, ctx := opentracing.StartSpanFromContext(ctx, c.getSpanName("UpdateComputedPipelineFirstInsertedAt"))
	defer span.Finish()

	return c.handleError(span, c.Client.UpdateComputedPipelineFirstInsertedAt(ctx, repoSource, repoOwner, repoName))
}

func (c *tracingClient) UpsertComputedRelease(ctx context.Context, repoSource, repoOwner, repoName, releaseName, releaseAction string) error {

	span, ctx := opentracing.StartSpanFromContext(ctx, c.getSpanName("UpsertComputedRelease"))
	defer span.Finish()

	return c.handleError(span, c.Client.UpsertComputedRelease(ctx, repoSource, repoOwner, repoName, releaseName, releaseAction))
}

func (c *tracingClient) UpdateComputedReleaseFirstInsertedAt(ctx context.Context, repoSource, repoOwner, repoName, releaseName, releaseAction string) error {

	span, ctx := opentracing.StartSpanFromContext(ctx, c.getSpanName("UpdateComputedReleaseFirstInsertedAt"))
	defer span.Finish()

	return c.handleError(span, c.Client.UpdateComputedReleaseFirstInsertedAt(ctx, repoSource, repoOwner, repoName, releaseName, releaseAction))
}

func (c *tracingClient) GetPipelines(ctx context.Context, pageNumber, pageSize int, filters map[string][]string, optimized bool) ([]*contracts.Pipeline, error) {

	span, ctx := opentracing.StartSpanFromContext(ctx, c.getSpanName("GetPipelines"))
	defer span.Finish()

	pipelines, err := c.Client.GetPipelines(ctx, pageNumber, pageSize, filters, optimized)
	c.handleError(span, err)

	return pipelines, err
}

func (c *tracingClient) GetPipelinesByRepoName(ctx context.Context, repoName string, optimized bool) ([]*contracts.Pipeline, error) {

	span, ctx := opentracing.StartSpanFromContext(ctx, c.getSpanName("GetPipelinesByRepoName"))
	defer span.Finish()

	pipeline, err := c.Client.GetPipelinesByRepoName(ctx, repoName, optimized)
	c.handleError(span, err)

	return pipeline, err
}

func (c *tracingClient) GetPipelinesCount(ctx context.Context, filters map[string][]string) (int, error) {

	span, ctx := opentracing.StartSpanFromContext(ctx, c.getSpanName("GetPipelinesCount"))
	defer span.Finish()

	count, err := c.Client.GetPipelinesCount(ctx, filters)
	c.handleError(span, err)

	return count, err
}

func (c *tracingClient) GetPipeline(ctx context.Context, repoSource, repoOwner, repoName string, optimized bool) (*contracts.Pipeline, error) {

	span, ctx := opentracing.StartSpanFromContext(ctx, c.getSpanName("GetPipeline"))
	defer span.Finish()

	pipeline, err := c.Client.GetPipeline(ctx, repoSource, repoOwner, repoName, optimized)
	c.handleError(span, err)

	return pipeline, err
}

func (c *tracingClient) GetPipelineBuilds(ctx context.Context, repoSource, repoOwner, repoName string, pageNumber, pageSize int, filters map[string][]string, optimized bool) ([]*contracts.Build, error) {

	span, ctx := opentracing.StartSpanFromContext(ctx, c.getSpanName("GetPipelineBuilds"))
	defer span.Finish()

	builds, err := c.Client.GetPipelineBuilds(ctx, repoSource, repoOwner, repoName, pageNumber, pageSize, filters, optimized)
	c.handleError(span, err)

	return builds, err
}

func (c *tracingClient) GetPipelineBuildsCount(ctx context.Context, repoSource, repoOwner, repoName string, filters map[string][]string) (int, error) {

	span, ctx := opentracing.StartSpanFromContext(ctx, c.getSpanName("GetPipelineBuildsCount"))
	defer span.Finish()

	count, err := c.Client.GetPipelineBuildsCount(ctx, repoSource, repoOwner, repoName, filters)
	c.handleError(span, err)

	return count, err
}

func (c *tracingClient) GetPipelineBuild(ctx context.Context, repoSource, repoOwner, repoName, repoRevision string, optimized bool) (*contracts.Build, error) {

	span, ctx := opentracing.StartSpanFromContext(ctx, c.getSpanName("GetPipelineBuild"))
	defer span.Finish()

	build, err := c.Client.GetPipelineBuild(ctx, repoSource, repoOwner, repoName, repoRevision, optimized)
	c.handleError(span, err)

	return build, err
}

func (c *tracingClient) GetPipelineBuildByID(ctx context.Context, repoSource, repoOwner, repoName string, id int, optimized bool) (*contracts.Build, error) {

	span, ctx := opentracing.StartSpanFromContext(ctx, c.getSpanName("GetPipelineBuildByID"))
	defer span.Finish()

	build, err := c.Client.GetPipelineBuildByID(ctx, repoSource, repoOwner, repoName, id, optimized)
	c.handleError(span, err)

	return build, err
}

func (c *tracingClient) GetLastPipelineBuild(ctx context.Context, repoSource, repoOwner, repoName string, optimized bool) (*contracts.Build, error) {

	span, ctx := opentracing.StartSpanFromContext(ctx, c.getSpanName("GetLastPipelineBuild"))
	defer span.Finish()

	build, err := c.Client.GetLastPipelineBuild(ctx, repoSource, repoOwner, repoName, optimized)
	c.handleError(span, err)

	return build, err
}

func (c *tracingClient) GetFirstPipelineBuild(ctx context.Context, repoSource, repoOwner, repoName string, optimized bool) (*contracts.Build, error) {

	span, ctx := opentracing.StartSpanFromContext(ctx, c.getSpanName("GetFirstPipelineBuild"))
	defer span.Finish()

	build, err := c.Client.GetFirstPipelineBuild(ctx, repoSource, repoOwner, repoName, optimized)
	c.handleError(span, err)

	return build, err
}

func (c *tracingClient) GetLastPipelineBuildForBranch(ctx context.Context, repoSource, repoOwner, repoName, branch string) (*contracts.Build, error) {

	span, ctx := opentracing.StartSpanFromContext(ctx, c.getSpanName("GetLastPipelineBuildForBranch"))
	defer span.Finish()

	build, err := c.Client.GetLastPipelineBuildForBranch(ctx, repoSource, repoOwner, repoName, branch)
	c.handleError(span, err)

	return build, err
}

func (c *tracingClient) GetLastPipelineRelease(ctx context.Context, repoSource, repoOwner, repoName, releaseName, releaseAction string) (*contracts.Release, error) {

	span, ctx := opentracing.StartSpanFromContext(ctx, c.getSpanName("GetLastPipelineRelease"))
	defer span.Finish()

	release, err := c.Client.GetLastPipelineRelease(ctx, repoSource, repoOwner, repoName, releaseName, releaseAction)
	c.handleError(span, err)

	return release, err
}

func (c *tracingClient) GetFirstPipelineRelease(ctx context.Context, repoSource, repoOwner, repoName, releaseName, releaseAction string) (*contracts.Release, error) {

	span, ctx := opentracing.StartSpanFromContext(ctx, c.getSpanName("GetFirstPipelineRelease"))
	defer span.Finish()

	release, err := c.Client.GetFirstPipelineRelease(ctx, repoSource, repoOwner, repoName, releaseName, releaseAction)
	c.handleError(span, err)

	return release, err
}

func (c *tracingClient) GetPipelineBuildsByVersion(ctx context.Context, repoSource, repoOwner, repoName, buildVersion string, statuses []string, limit uint64, optimized bool) ([]*contracts.Build, error) {

	span, ctx := opentracing.StartSpanFromContext(ctx, c.getSpanName("GetPipelineBuildsByVersion"))
	defer span.Finish()

	builds, err := c.Client.GetPipelineBuildsByVersion(ctx, repoSource, repoOwner, repoName, buildVersion, statuses, limit, optimized)
	c.handleError(span, err)

	return builds, err
}

func (c *tracingClient) GetPipelineBuildLogs(ctx context.Context, repoSource, repoOwner, repoName, repoBranch, repoRevision, buildID string, readLogFromDatabase bool) (*contracts.BuildLog, error) {

	span, ctx := opentracing.StartSpanFromContext(ctx, c.getSpanName("GetPipelineBuildLogs"))
	defer span.Finish()

	buildlog, err := c.Client.GetPipelineBuildLogs(ctx, repoSource, repoOwner, repoName, repoBranch, repoRevision, buildID, readLogFromDatabase)
	c.handleError(span, err)

	return buildlog, err
}

func (c *tracingClient) GetPipelineBuildLogsPerPage(ctx context.Context, repoSource, repoOwner, repoName string, pageNumber int, pageSize int) ([]*contracts.BuildLog, error) {

	span, ctx := opentracing.StartSpanFromContext(ctx, c.getSpanName("GetPipelineBuildLogsPerPage"))
	defer span.Finish()

	buildlogs, err := c.Client.GetPipelineBuildLogsPerPage(ctx, repoSource, repoOwner, repoName, pageNumber, pageSize)
	c.handleError(span, err)

	return buildlogs, err
}

func (c *tracingClient) GetPipelineBuildMaxResourceUtilization(ctx context.Context, repoSource, repoOwner, repoName string, lastNRecords int) (JobResources, int, error) {

	span, ctx := opentracing.StartSpanFromContext(ctx, c.getSpanName("GetPipelineBuildMaxResourceUtilization"))
	defer span.Finish()

	jobresources, recordCount, err := c.Client.GetPipelineBuildMaxResourceUtilization(ctx, repoSource, repoOwner, repoName, lastNRecords)
	c.handleError(span, err)

	return jobresources, recordCount, err
}

func (c *tracingClient) GetPipelineReleases(ctx context.Context, repoSource, repoOwner, repoName string, pageNumber, pageSize int, filters map[string][]string) ([]*contracts.Release, error) {

	span, ctx := opentracing.StartSpanFromContext(ctx, c.getSpanName("GetPipelineReleases"))
	defer span.Finish()

	releases, err := c.Client.GetPipelineReleases(ctx, repoSource, repoOwner, repoName, pageNumber, pageSize, filters)
	c.handleError(span, err)

	return releases, err
}

func (c *tracingClient) GetPipelineReleasesCount(ctx context.Context, repoSource, repoOwner, repoName string, filters map[string][]string) (int, error) {

	span, ctx := opentracing.StartSpanFromContext(ctx, c.getSpanName("GetPipelineReleasesCount"))
	defer span.Finish()

	count, err := c.Client.GetPipelineReleasesCount(ctx, repoSource, repoOwner, repoName, filters)
	c.handleError(span, err)

	return count, err
}

func (c *tracingClient) GetPipelineRelease(ctx context.Context, repoSource, repoOwner, repoName string, id int) (*contracts.Release, error) {

	span, ctx := opentracing.StartSpanFromContext(ctx, c.getSpanName("GetPipelineRelease"))
	defer span.Finish()

	release, err := c.Client.GetPipelineRelease(ctx, repoSource, repoOwner, repoName, id)
	c.handleError(span, err)

	return release, err
}

func (c *tracingClient) GetPipelineLastReleasesByName(ctx context.Context, repoSource, repoOwner, repoName, releaseName string, actions []string) ([]contracts.Release, error) {

	span, ctx := opentracing.StartSpanFromContext(ctx, c.getSpanName("GetPipelineLastReleasesByName"))
	defer span.Finish()

	releases, err := c.Client.GetPipelineLastReleasesByName(ctx, repoSource, repoOwner, repoName, releaseName, actions)
	c.handleError(span, err)

	return releases, err
}

func (c *tracingClient) GetPipelineReleaseLogs(ctx context.Context, repoSource, repoOwner, repoName string, id int, readLogFromDatabase bool) (*contracts.ReleaseLog, error) {

	span, ctx := opentracing.StartSpanFromContext(ctx, c.getSpanName("GetPipelineReleaseLogs"))
	defer span.Finish()

	releaselog, err := c.Client.GetPipelineReleaseLogs(ctx, repoSource, repoOwner, repoName, id, readLogFromDatabase)
	c.handleError(span, err)

	return releaselog, err
}

func (c *tracingClient) GetPipelineReleaseLogsPerPage(ctx context.Context, repoSource, repoOwner, repoName string, pageNumber int, pageSize int) ([]*contracts.ReleaseLog, error) {

	span, ctx := opentracing.StartSpanFromContext(ctx, c.getSpanName("GetPipelineReleaseLogsPerPage"))
	defer span.Finish()

	releaselogs, err := c.Client.GetPipelineReleaseLogsPerPage(ctx, repoSource, repoOwner, repoName, pageNumber, pageSize)
	c.handleError(span, err)

	return releaselogs, err
}

func (c *tracingClient) GetPipelineReleaseMaxResourceUtilization(ctx context.Context, repoSource, repoOwner, repoName, targetName string, lastNRecords int) (JobResources, int, error) {

	span, ctx := opentracing.StartSpanFromContext(ctx, c.getSpanName("GetPipelineReleaseMaxResourceUtilization"))
	defer span.Finish()

	jobresources, recordCount, err := c.Client.GetPipelineReleaseMaxResourceUtilization(ctx, repoSource, repoOwner, repoName, targetName, lastNRecords)
	c.handleError(span, err)

	return jobresources, recordCount, err
}

func (c *tracingClient) GetBuildsCount(ctx context.Context, filters map[string][]string) (int, error) {

	span, ctx := opentracing.StartSpanFromContext(ctx, c.getSpanName("GetBuildsCount"))
	defer span.Finish()

	count, err := c.Client.GetBuildsCount(ctx, filters)
	c.handleError(span, err)

	return count, err
}

func (c *tracingClient) GetReleasesCount(ctx context.Context, filters map[string][]string) (int, error) {

	span, ctx := opentracing.StartSpanFromContext(ctx, c.getSpanName("GetReleasesCount"))
	defer span.Finish()

	count, err := c.Client.GetReleasesCount(ctx, filters)
	c.handleError(span, err)

	return count, err
}

func (c *tracingClient) GetBuildsDuration(ctx context.Context, filters map[string][]string) (time.Duration, error) {

	span, ctx := opentracing.StartSpanFromContext(ctx, c.getSpanName("GetBuildsDuration"))
	defer span.Finish()

	duration, err := c.Client.GetBuildsDuration(ctx, filters)
	c.handleError(span, err)

	return duration, err
}

func (c *tracingClient) GetFirstBuildTimes(ctx context.Context) ([]time.Time, error) {

	span, ctx := opentracing.StartSpanFromContext(ctx, c.getSpanName("GetFirstBuildTimes"))
	defer span.Finish()

	buildtimes, err := c.Client.GetFirstBuildTimes(ctx)
	c.handleError(span, err)

	return buildtimes, err
}

func (c *tracingClient) GetFirstReleaseTimes(ctx context.Context) ([]time.Time, error) {

	span, ctx := opentracing.StartSpanFromContext(ctx, c.getSpanName("GetFirstReleaseTimes"))
	defer span.Finish()

	releasetimes, err := c.Client.GetFirstReleaseTimes(ctx)
	c.handleError(span, err)

	return releasetimes, err
}

func (c *tracingClient) GetPipelineBuildsDurations(ctx context.Context, repoSource, repoOwner, repoName string, filters map[string][]string) ([]map[string]interface{}, error) {

	span, ctx := opentracing.StartSpanFromContext(ctx, c.getSpanName("GetPipelineBuildsDurations"))
	defer span.Finish()

	durations, err := c.Client.GetPipelineBuildsDurations(ctx, repoSource, repoOwner, repoName, filters)
	c.handleError(span, err)

	return durations, err
}

func (c *tracingClient) GetPipelineReleasesDurations(ctx context.Context, repoSource, repoOwner, repoName string, filters map[string][]string) ([]map[string]interface{}, error) {

	span, ctx := opentracing.StartSpanFromContext(ctx, c.getSpanName("GetPipelineReleasesDurations"))
	defer span.Finish()

	durations, err := c.Client.GetPipelineReleasesDurations(ctx, repoSource, repoOwner, repoName, filters)
	c.handleError(span, err)

	return durations, err
}

func (c *tracingClient) GetPipelineBuildsCPUUsageMeasurements(ctx context.Context, repoSource, repoOwner, repoName string, filters map[string][]string) ([]map[string]interface{}, error) {

	span, ctx := opentracing.StartSpanFromContext(ctx, c.getSpanName("GetPipelineBuildsCPUUsageMeasurements"))
	defer span.Finish()

	measurements, err := c.Client.GetPipelineBuildsCPUUsageMeasurements(ctx, repoSource, repoOwner, repoName, filters)
	c.handleError(span, err)

	return measurements, err
}

func (c *tracingClient) GetPipelineReleasesCPUUsageMeasurements(ctx context.Context, repoSource, repoOwner, repoName string, filters map[string][]string) ([]map[string]interface{}, error) {

	span, ctx := opentracing.StartSpanFromContext(ctx, c.getSpanName("GetPipelineReleasesCPUUsageMeasurements"))
	defer span.Finish()

	measurements, err := c.Client.GetPipelineReleasesCPUUsageMeasurements(ctx, repoSource, repoOwner, repoName, filters)
	c.handleError(span, err)

	return measurements, err
}

func (c *tracingClient) GetPipelineBuildsMemoryUsageMeasurements(ctx context.Context, repoSource, repoOwner, repoName string, filters map[string][]string) ([]map[string]interface{}, error) {

	span, ctx := opentracing.StartSpanFromContext(ctx, c.getSpanName("GetPipelineBuildsMemoryUsageMeasurements"))
	defer span.Finish()

	measurements, err := c.Client.GetPipelineBuildsMemoryUsageMeasurements(ctx, repoSource, repoOwner, repoName, filters)
	c.handleError(span, err)

	return measurements, err
}

func (c *tracingClient) GetPipelineReleasesMemoryUsageMeasurements(ctx context.Context, repoSource, repoOwner, repoName string, filters map[string][]string) ([]map[string]interface{}, error) {

	span, ctx := opentracing.StartSpanFromContext(ctx, c.getSpanName("GetPipelineReleasesMemoryUsageMeasurements"))
	defer span.Finish()

	measurements, err := c.Client.GetPipelineReleasesMemoryUsageMeasurements(ctx, repoSource, repoOwner, repoName, filters)
	c.handleError(span, err)

	return measurements, err
}

func (c *tracingClient) GetFrequentLabels(ctx context.Context, pageNumber, pageSize int, filters map[string][]string) ([]map[string]interface{}, error) {

	span, ctx := opentracing.StartSpanFromContext(ctx, c.getSpanName("GetFrequentLabels"))
	defer span.Finish()

	measurements, err := c.Client.GetFrequentLabels(ctx, pageNumber, pageSize, filters)
	c.handleError(span, err)

	return measurements, err
}

func (c *tracingClient) GetFrequentLabelsCount(ctx context.Context, filters map[string][]string) (int, error) {

	span, ctx := opentracing.StartSpanFromContext(ctx, c.getSpanName("GetFrequentLabelsCount"))
	defer span.Finish()

	count, err := c.Client.GetFrequentLabelsCount(ctx, filters)
	c.handleError(span, err)

	return count, err
}

func (c *tracingClient) GetPipelinesWithMostBuilds(ctx context.Context, pageNumber, pageSize int, filters map[string][]string) ([]map[string]interface{}, error) {

	span, ctx := opentracing.StartSpanFromContext(ctx, c.getSpanName("GetPipelinesWithMostBuilds"))
	defer span.Finish()

	pipelines, err := c.Client.GetPipelinesWithMostBuilds(ctx, pageNumber, pageSize, filters)
	c.handleError(span, err)

	return pipelines, err
}

func (c *tracingClient) GetPipelinesWithMostBuildsCount(ctx context.Context, filters map[string][]string) (int, error) {

	span, ctx := opentracing.StartSpanFromContext(ctx, c.getSpanName("GetPipelinesWithMostBuildsCount"))
	defer span.Finish()

	pipelines, err := c.Client.GetPipelinesWithMostBuildsCount(ctx, filters)
	c.handleError(span, err)

	return pipelines, err
}

func (c *tracingClient) GetPipelinesWithMostReleases(ctx context.Context, pageNumber, pageSize int, filters map[string][]string) ([]map[string]interface{}, error) {

	span, ctx := opentracing.StartSpanFromContext(ctx, c.getSpanName("GetPipelinesWithMostReleases"))
	defer span.Finish()

	pipelines, err := c.Client.GetPipelinesWithMostReleases(ctx, pageNumber, pageSize, filters)
	c.handleError(span, err)

	return pipelines, err
}

func (c *tracingClient) GetPipelinesWithMostReleasesCount(ctx context.Context, filters map[string][]string) (int, error) {

	span, ctx := opentracing.StartSpanFromContext(ctx, c.getSpanName("GetPipelinesWithMostReleasesCount"))
	defer span.Finish()

	pipelines, err := c.Client.GetPipelinesWithMostReleasesCount(ctx, filters)
	c.handleError(span, err)

	return pipelines, err
}

func (c *tracingClient) GetTriggers(ctx context.Context, triggerType, identifier, event string) ([]*contracts.Pipeline, error) {

	span, ctx := opentracing.StartSpanFromContext(ctx, c.getSpanName("GetTriggers"))
	defer span.Finish()

	pipelines, err := c.Client.GetTriggers(ctx, triggerType, identifier, event)
	c.handleError(span, err)

	return pipelines, err
}

func (c *tracingClient) GetGitTriggers(ctx context.Context, gitEvent manifest.EstafetteGitEvent) ([]*contracts.Pipeline, error) {

	span, ctx := opentracing.StartSpanFromContext(ctx, c.getSpanName("GetGitTriggers"))
	defer span.Finish()

	pipelines, err := c.Client.GetGitTriggers(ctx, gitEvent)
	c.handleError(span, err)

	return pipelines, err
}

func (c *tracingClient) GetPipelineTriggers(ctx context.Context, build contracts.Build, event string) ([]*contracts.Pipeline, error) {

	span, ctx := opentracing.StartSpanFromContext(ctx, c.getSpanName("GetPipelineTriggers"))
	defer span.Finish()

	pipelines, err := c.Client.GetPipelineTriggers(ctx, build, event)
	c.handleError(span, err)

	return pipelines, err
}

func (c *tracingClient) GetReleaseTriggers(ctx context.Context, release contracts.Release, event string) ([]*contracts.Pipeline, error) {

	span, ctx := opentracing.StartSpanFromContext(ctx, c.getSpanName("GetReleaseTriggers"))
	defer span.Finish()

	pipelines, err := c.Client.GetReleaseTriggers(ctx, release, event)
	c.handleError(span, err)

	return pipelines, err
}

func (c *tracingClient) GetPubSubTriggers(ctx context.Context, pubsubEvent manifest.EstafettePubSubEvent) ([]*contracts.Pipeline, error) {

	span, ctx := opentracing.StartSpanFromContext(ctx, c.getSpanName("GetPubSubTriggers"))
	defer span.Finish()

	pipelines, err := c.Client.GetPubSubTriggers(ctx, pubsubEvent)
	c.handleError(span, err)

	return pipelines, err
}

func (c *tracingClient) GetCronTriggers(ctx context.Context) ([]*contracts.Pipeline, error) {

	span, ctx := opentracing.StartSpanFromContext(ctx, c.getSpanName("GetCronTriggers"))
	defer span.Finish()

	pipelines, err := c.Client.GetCronTriggers(ctx)
	c.handleError(span, err)

	return pipelines, err
}

func (c *tracingClient) Rename(ctx context.Context, shortFromRepoSource, fromRepoSource, fromRepoOwner, fromRepoName, shortToRepoSource, toRepoSource, toRepoOwner, toRepoName string) error {

	span, ctx := opentracing.StartSpanFromContext(ctx, c.getSpanName("Rename"))
	defer span.Finish()

	return c.handleError(span, c.Client.Rename(ctx, shortFromRepoSource, fromRepoSource, fromRepoOwner, fromRepoName, shortToRepoSource, toRepoSource, toRepoOwner, toRepoName))
}

func (c *tracingClient) RenameBuildVersion(ctx context.Context, shortFromRepoSource, fromRepoOwner, fromRepoName, shortToRepoSource, toRepoOwner, toRepoName string) error {

	span, ctx := opentracing.StartSpanFromContext(ctx, c.getSpanName("RenameBuildVersion"))
	defer span.Finish()

	return c.handleError(span, c.Client.RenameBuildVersion(ctx, shortFromRepoSource, fromRepoOwner, fromRepoName, shortToRepoSource, toRepoOwner, toRepoName))
}

func (c *tracingClient) RenameBuilds(ctx context.Context, fromRepoSource, fromRepoOwner, fromRepoName, toRepoSource, toRepoOwner, toRepoName string) error {

	span, ctx := opentracing.StartSpanFromContext(ctx, c.getSpanName("RenameBuilds"))
	defer span.Finish()

	return c.handleError(span, c.Client.RenameBuilds(ctx, fromRepoSource, fromRepoOwner, fromRepoName, toRepoSource, toRepoOwner, toRepoName))
}

func (c *tracingClient) RenameBuildLogs(ctx context.Context, fromRepoSource, fromRepoOwner, fromRepoName, toRepoSource, toRepoOwner, toRepoName string) error {

	span, ctx := opentracing.StartSpanFromContext(ctx, c.getSpanName("RenameBuildLogs"))
	defer span.Finish()

	return c.handleError(span, c.Client.RenameBuildLogs(ctx, fromRepoSource, fromRepoOwner, fromRepoName, toRepoSource, toRepoOwner, toRepoName))
}

func (c *tracingClient) RenameReleases(ctx context.Context, fromRepoSource, fromRepoOwner, fromRepoName, toRepoSource, toRepoOwner, toRepoName string) error {

	span, ctx := opentracing.StartSpanFromContext(ctx, c.getSpanName("RenameReleases"))
	defer span.Finish()

	return c.handleError(span, c.Client.RenameReleases(ctx, fromRepoSource, fromRepoOwner, fromRepoName, toRepoSource, toRepoOwner, toRepoName))
}

func (c *tracingClient) RenameReleaseLogs(ctx context.Context, fromRepoSource, fromRepoOwner, fromRepoName, toRepoSource, toRepoOwner, toRepoName string) error {

	span, ctx := opentracing.StartSpanFromContext(ctx, c.getSpanName("RenameReleaseLogs"))
	defer span.Finish()

	return c.handleError(span, c.Client.RenameReleaseLogs(ctx, fromRepoSource, fromRepoOwner, fromRepoName, toRepoSource, toRepoOwner, toRepoName))
}

func (c *tracingClient) RenameComputedPipelines(ctx context.Context, fromRepoSource, fromRepoOwner, fromRepoName, toRepoSource, toRepoOwner, toRepoName string) error {

	span, ctx := opentracing.StartSpanFromContext(ctx, c.getSpanName("RenameComputedPipelines"))
	defer span.Finish()

	return c.handleError(span, c.Client.RenameComputedPipelines(ctx, fromRepoSource, fromRepoOwner, fromRepoName, toRepoSource, toRepoOwner, toRepoName))
}

func (c *tracingClient) RenameComputedReleases(ctx context.Context, fromRepoSource, fromRepoOwner, fromRepoName, toRepoSource, toRepoOwner, toRepoName string) error {

	span, ctx := opentracing.StartSpanFromContext(ctx, c.getSpanName("RenameComputedReleases"))
	defer span.Finish()

	return c.handleError(span, c.Client.RenameComputedReleases(ctx, fromRepoSource, fromRepoOwner, fromRepoName, toRepoSource, toRepoOwner, toRepoName))
}

func (c *tracingClient) getSpanName(funcName string) string {
	return "cockroachdb:" + funcName
}

func (c *tracingClient) handleError(span opentracing.Span, err error) error {
	if err != nil {
		ext.Error.Set(span, true)
		span.LogFields(log.Error(err))
	}
	return err
}
