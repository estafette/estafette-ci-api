package cockroachdb

import (
	"context"
	"time"

	contracts "github.com/estafette/estafette-ci-contracts"
	manifest "github.com/estafette/estafette-ci-manifest"
	"github.com/opentracing/opentracing-go"
)

type tracingClient struct {
	Client
}

// NewTracingClient returns a new instance of a tracing Client.
func NewTracingClient(c Client) Client {
	return &tracingClient{c}
}

func (s *tracingClient) Connect(ctx context.Context) error {

	span, ctx := opentracing.StartSpanFromContext(ctx, s.getSpanName("Connect"))
	defer span.Finish()

	return s.Client.Connect(ctx)
}

func (s *tracingClient) ConnectWithDriverAndSource(ctx context.Context, driverName, dataSourceName string) error {

	span, ctx := opentracing.StartSpanFromContext(ctx, s.getSpanName("ConnectWithDriverAndSource"))
	defer span.Finish()

	return s.Client.ConnectWithDriverAndSource(ctx, driverName, dataSourceName)
}

func (s *tracingClient) GetAutoIncrement(ctx context.Context, shortRepoSource, repoOwner, repoName string) (int, error) {

	span, ctx := opentracing.StartSpanFromContext(ctx, s.getSpanName("GetAutoIncrement"))
	defer span.Finish()

	return s.Client.GetAutoIncrement(ctx, shortRepoSource, repoOwner, repoName)
}

func (s *tracingClient) InsertBuild(ctx context.Context, build contracts.Build, jobResources JobResources) (*contracts.Build, error) {

	span, ctx := opentracing.StartSpanFromContext(ctx, s.getSpanName("InsertBuild"))
	defer span.Finish()

	return s.Client.InsertBuild(ctx, build, jobResources)
}

func (s *tracingClient) UpdateBuildStatus(ctx context.Context, repoSource, repoOwner, repoName string, buildID int, buildStatus string) error {

	span, ctx := opentracing.StartSpanFromContext(ctx, s.getSpanName("UpdateBuildStatus"))
	defer span.Finish()

	return s.Client.UpdateBuildStatus(ctx, repoSource, repoOwner, repoName, buildID, buildStatus)
}

func (s *tracingClient) UpdateBuildResourceUtilization(ctx context.Context, repoSource, repoOwner, repoName string, buildID int, jobResources JobResources) error {

	span, ctx := opentracing.StartSpanFromContext(ctx, s.getSpanName("UpdateBuildResourceUtilization"))
	defer span.Finish()

	return s.Client.UpdateBuildResourceUtilization(ctx, repoSource, repoOwner, repoName, buildID, jobResources)
}

func (s *tracingClient) InsertRelease(ctx context.Context, release contracts.Release, jobResources JobResources) (*contracts.Release, error) {

	span, ctx := opentracing.StartSpanFromContext(ctx, s.getSpanName("InsertRelease"))
	defer span.Finish()

	return s.Client.InsertRelease(ctx, release, jobResources)
}

func (s *tracingClient) UpdateReleaseStatus(ctx context.Context, repoSource, repoOwner, repoName string, id int, releaseStatus string) error {

	span, ctx := opentracing.StartSpanFromContext(ctx, s.getSpanName("UpdateReleaseStatus"))
	defer span.Finish()

	return s.Client.UpdateReleaseStatus(ctx, repoSource, repoOwner, repoName, id, releaseStatus)
}

func (s *tracingClient) UpdateReleaseResourceUtilization(ctx context.Context, repoSource, repoOwner, repoName string, id int, jobResources JobResources) error {

	span, ctx := opentracing.StartSpanFromContext(ctx, s.getSpanName("UpdateReleaseResourceUtilization"))
	defer span.Finish()

	return s.Client.UpdateReleaseResourceUtilization(ctx, repoSource, repoOwner, repoName, id, jobResources)
}

func (s *tracingClient) InsertBuildLog(ctx context.Context, buildLog contracts.BuildLog, writeLogToDatabase bool) (contracts.BuildLog, error) {

	span, ctx := opentracing.StartSpanFromContext(ctx, s.getSpanName("InsertBuildLog"))
	defer span.Finish()

	return s.Client.InsertBuildLog(ctx, buildLog, writeLogToDatabase)
}

func (s *tracingClient) InsertReleaseLog(ctx context.Context, releaseLog contracts.ReleaseLog, writeLogToDatabase bool) (contracts.ReleaseLog, error) {

	span, ctx := opentracing.StartSpanFromContext(ctx, s.getSpanName("InsertReleaseLog"))
	defer span.Finish()

	return s.Client.InsertReleaseLog(ctx, releaseLog, writeLogToDatabase)
}

func (s *tracingClient) UpsertComputedPipeline(ctx context.Context, repoSource, repoOwner, repoName string) error {

	span, ctx := opentracing.StartSpanFromContext(ctx, s.getSpanName("UpsertComputedPipeline"))
	defer span.Finish()

	return s.Client.UpsertComputedPipeline(ctx, repoSource, repoOwner, repoName)
}

func (s *tracingClient) UpdateComputedPipelineFirstInsertedAt(ctx context.Context, repoSource, repoOwner, repoName string) error {

	span, ctx := opentracing.StartSpanFromContext(ctx, s.getSpanName("UpdateComputedPipelineFirstInsertedAt"))
	defer span.Finish()

	return s.Client.UpdateComputedPipelineFirstInsertedAt(ctx, repoSource, repoOwner, repoName)
}

func (s *tracingClient) UpsertComputedRelease(ctx context.Context, repoSource, repoOwner, repoName, releaseName, releaseAction string) error {

	span, ctx := opentracing.StartSpanFromContext(ctx, s.getSpanName("UpsertComputedRelease"))
	defer span.Finish()

	return s.Client.UpsertComputedRelease(ctx, repoSource, repoOwner, repoName, releaseName, releaseAction)
}

func (s *tracingClient) UpdateComputedReleaseFirstInsertedAt(ctx context.Context, repoSource, repoOwner, repoName, releaseName, releaseAction string) error {

	span, ctx := opentracing.StartSpanFromContext(ctx, s.getSpanName("UpdateComputedReleaseFirstInsertedAt"))
	defer span.Finish()

	return s.Client.UpdateComputedReleaseFirstInsertedAt(ctx, repoSource, repoOwner, repoName, releaseName, releaseAction)
}

func (s *tracingClient) GetPipelines(ctx context.Context, pageNumber, pageSize int, filters map[string][]string, optimized bool) ([]*contracts.Pipeline, error) {

	span, ctx := opentracing.StartSpanFromContext(ctx, s.getSpanName("GetPipelines"))
	defer span.Finish()

	return s.Client.GetPipelines(ctx, pageNumber, pageSize, filters, optimized)
}

func (s *tracingClient) GetPipelinesByRepoName(ctx context.Context, repoName string, optimized bool) ([]*contracts.Pipeline, error) {

	span, ctx := opentracing.StartSpanFromContext(ctx, s.getSpanName("GetPipelinesByRepoName"))
	defer span.Finish()

	return s.Client.GetPipelinesByRepoName(ctx, repoName, optimized)
}

func (s *tracingClient) GetPipelinesCount(ctx context.Context, filters map[string][]string) (int, error) {

	span, ctx := opentracing.StartSpanFromContext(ctx, s.getSpanName("GetPipelinesCount"))
	defer span.Finish()

	return s.Client.GetPipelinesCount(ctx, filters)
}

func (s *tracingClient) GetPipeline(ctx context.Context, repoSource, repoOwner, repoName string, optimized bool) (*contracts.Pipeline, error) {

	span, ctx := opentracing.StartSpanFromContext(ctx, s.getSpanName("GetPipeline"))
	defer span.Finish()

	return s.Client.GetPipeline(ctx, repoSource, repoOwner, repoName, optimized)
}

func (s *tracingClient) GetPipelineBuilds(ctx context.Context, repoSource, repoOwner, repoName string, pageNumber, pageSize int, filters map[string][]string, optimized bool) ([]*contracts.Build, error) {

	span, ctx := opentracing.StartSpanFromContext(ctx, s.getSpanName("GetPipelineBuilds"))
	defer span.Finish()

	return s.Client.GetPipelineBuilds(ctx, repoSource, repoOwner, repoName, pageNumber, pageSize, filters, optimized)
}

func (s *tracingClient) GetPipelineBuildsCount(ctx context.Context, repoSource, repoOwner, repoName string, filters map[string][]string) (int, error) {

	span, ctx := opentracing.StartSpanFromContext(ctx, s.getSpanName("GetPipelineBuildsCount"))
	defer span.Finish()

	return s.Client.GetPipelineBuildsCount(ctx, repoSource, repoOwner, repoName, filters)
}

func (s *tracingClient) GetPipelineBuild(ctx context.Context, repoSource, repoOwner, repoName, repoRevision string, optimized bool) (*contracts.Build, error) {

	span, ctx := opentracing.StartSpanFromContext(ctx, s.getSpanName("GetPipelineBuild"))
	defer span.Finish()

	return s.Client.GetPipelineBuild(ctx, repoSource, repoOwner, repoName, repoRevision, optimized)
}

func (s *tracingClient) GetPipelineBuildByID(ctx context.Context, repoSource, repoOwner, repoName string, id int, optimized bool) (*contracts.Build, error) {

	span, ctx := opentracing.StartSpanFromContext(ctx, s.getSpanName("GetPipelineBuildByID"))
	defer span.Finish()

	return s.Client.GetPipelineBuildByID(ctx, repoSource, repoOwner, repoName, id, optimized)
}

func (s *tracingClient) GetLastPipelineBuild(ctx context.Context, repoSource, repoOwner, repoName string, optimized bool) (*contracts.Build, error) {

	span, ctx := opentracing.StartSpanFromContext(ctx, s.getSpanName("GetLastPipelineBuild"))
	defer span.Finish()

	return s.Client.GetLastPipelineBuild(ctx, repoSource, repoOwner, repoName, optimized)
}

func (s *tracingClient) GetFirstPipelineBuild(ctx context.Context, repoSource, repoOwner, repoName string, optimized bool) (*contracts.Build, error) {

	span, ctx := opentracing.StartSpanFromContext(ctx, s.getSpanName("GetFirstPipelineBuild"))
	defer span.Finish()

	return s.Client.GetFirstPipelineBuild(ctx, repoSource, repoOwner, repoName, optimized)
}

func (s *tracingClient) GetLastPipelineBuildForBranch(ctx context.Context, repoSource, repoOwner, repoName, branch string) (*contracts.Build, error) {

	span, ctx := opentracing.StartSpanFromContext(ctx, s.getSpanName("GetLastPipelineBuildForBranch"))
	defer span.Finish()

	return s.Client.GetLastPipelineBuildForBranch(ctx, repoSource, repoOwner, repoName, branch)
}

func (s *tracingClient) GetLastPipelineRelease(ctx context.Context, repoSource, repoOwner, repoName, releaseName, releaseAction string) (*contracts.Release, error) {

	span, ctx := opentracing.StartSpanFromContext(ctx, s.getSpanName("GetLastPipelineRelease"))
	defer span.Finish()

	return s.Client.GetLastPipelineRelease(ctx, repoSource, repoOwner, repoName, releaseName, releaseAction)
}

func (s *tracingClient) GetFirstPipelineRelease(ctx context.Context, repoSource, repoOwner, repoName, releaseName, releaseAction string) (*contracts.Release, error) {

	span, ctx := opentracing.StartSpanFromContext(ctx, s.getSpanName("GetFirstPipelineRelease"))
	defer span.Finish()

	return s.Client.GetFirstPipelineRelease(ctx, repoSource, repoOwner, repoName, releaseName, releaseAction)
}

func (s *tracingClient) GetPipelineBuildsByVersion(ctx context.Context, repoSource, repoOwner, repoName, buildVersion string, statuses []string, limit uint64, optimized bool) ([]*contracts.Build, error) {

	span, ctx := opentracing.StartSpanFromContext(ctx, s.getSpanName("GetPipelineBuildsByVersion"))
	defer span.Finish()

	return s.Client.GetPipelineBuildsByVersion(ctx, repoSource, repoOwner, repoName, buildVersion, statuses, limit, optimized)
}

func (s *tracingClient) GetPipelineBuildLogs(ctx context.Context, repoSource, repoOwner, repoName, repoBranch, repoRevision, buildID string, readLogFromDatabase bool) (*contracts.BuildLog, error) {

	span, ctx := opentracing.StartSpanFromContext(ctx, s.getSpanName("GetPipelineBuildLogs"))
	defer span.Finish()

	return s.Client.GetPipelineBuildLogs(ctx, repoSource, repoOwner, repoName, repoBranch, repoRevision, buildID, readLogFromDatabase)
}

func (s *tracingClient) GetPipelineBuildLogsPerPage(ctx context.Context, repoSource, repoOwner, repoName string, pageNumber int, pageSize int) ([]*contracts.BuildLog, error) {

	span, ctx := opentracing.StartSpanFromContext(ctx, s.getSpanName("GetPipelineBuildLogsPerPage"))
	defer span.Finish()

	return s.Client.GetPipelineBuildLogsPerPage(ctx, repoSource, repoOwner, repoName, pageNumber, pageSize)
}

func (s *tracingClient) GetPipelineBuildMaxResourceUtilization(ctx context.Context, repoSource, repoOwner, repoName string, lastNRecords int) (JobResources, int, error) {

	span, ctx := opentracing.StartSpanFromContext(ctx, s.getSpanName("GetPipelineBuildMaxResourceUtilization"))
	defer span.Finish()

	return s.Client.GetPipelineBuildMaxResourceUtilization(ctx, repoSource, repoOwner, repoName, lastNRecords)
}

func (s *tracingClient) GetPipelineReleases(ctx context.Context, repoSource, repoOwner, repoName string, pageNumber, pageSize int, filters map[string][]string) ([]*contracts.Release, error) {

	span, ctx := opentracing.StartSpanFromContext(ctx, s.getSpanName("GetPipelineReleases"))
	defer span.Finish()

	return s.Client.GetPipelineReleases(ctx, repoSource, repoOwner, repoName, pageNumber, pageSize, filters)
}

func (s *tracingClient) GetPipelineReleasesCount(ctx context.Context, repoSource, repoOwner, repoName string, filters map[string][]string) (int, error) {

	span, ctx := opentracing.StartSpanFromContext(ctx, s.getSpanName("GetPipelineReleasesCount"))
	defer span.Finish()

	return s.Client.GetPipelineReleasesCount(ctx, repoSource, repoOwner, repoName, filters)
}

func (s *tracingClient) GetPipelineRelease(ctx context.Context, repoSource, repoOwner, repoName string, id int) (*contracts.Release, error) {

	span, ctx := opentracing.StartSpanFromContext(ctx, s.getSpanName("GetPipelineRelease"))
	defer span.Finish()

	return s.Client.GetPipelineRelease(ctx, repoSource, repoOwner, repoName, id)
}

func (s *tracingClient) GetPipelineLastReleasesByName(ctx context.Context, repoSource, repoOwner, repoName, releaseName string, actions []string) ([]contracts.Release, error) {

	span, ctx := opentracing.StartSpanFromContext(ctx, s.getSpanName("GetPipelineLastReleasesByName"))
	defer span.Finish()

	return s.Client.GetPipelineLastReleasesByName(ctx, repoSource, repoOwner, repoName, releaseName, actions)
}

func (s *tracingClient) GetPipelineReleaseLogs(ctx context.Context, repoSource, repoOwner, repoName string, id int, readLogFromDatabase bool) (*contracts.ReleaseLog, error) {

	span, ctx := opentracing.StartSpanFromContext(ctx, s.getSpanName("GetPipelineReleaseLogs"))
	defer span.Finish()

	return s.Client.GetPipelineReleaseLogs(ctx, repoSource, repoOwner, repoName, id, readLogFromDatabase)
}

func (s *tracingClient) GetPipelineReleaseLogsPerPage(ctx context.Context, repoSource, repoOwner, repoName string, pageNumber int, pageSize int) ([]*contracts.ReleaseLog, error) {

	span, ctx := opentracing.StartSpanFromContext(ctx, s.getSpanName("GetPipelineReleaseLogsPerPage"))
	defer span.Finish()

	return s.Client.GetPipelineReleaseLogsPerPage(ctx, repoSource, repoOwner, repoName, pageNumber, pageSize)
}

func (s *tracingClient) GetPipelineReleaseMaxResourceUtilization(ctx context.Context, repoSource, repoOwner, repoName, targetName string, lastNRecords int) (JobResources, int, error) {

	span, ctx := opentracing.StartSpanFromContext(ctx, s.getSpanName("GetPipelineReleaseMaxResourceUtilization"))
	defer span.Finish()

	return s.Client.GetPipelineReleaseMaxResourceUtilization(ctx, repoSource, repoOwner, repoName, targetName, lastNRecords)
}

func (s *tracingClient) GetBuildsCount(ctx context.Context, filters map[string][]string) (int, error) {

	span, ctx := opentracing.StartSpanFromContext(ctx, s.getSpanName("GetBuildsCount"))
	defer span.Finish()

	return s.Client.GetBuildsCount(ctx, filters)
}

func (s *tracingClient) GetReleasesCount(ctx context.Context, filters map[string][]string) (int, error) {

	span, ctx := opentracing.StartSpanFromContext(ctx, s.getSpanName("GetReleasesCount"))
	defer span.Finish()

	return s.Client.GetReleasesCount(ctx, filters)
}

func (s *tracingClient) GetBuildsDuration(ctx context.Context, filters map[string][]string) (time.Duration, error) {

	span, ctx := opentracing.StartSpanFromContext(ctx, s.getSpanName("GetBuildsDuration"))
	defer span.Finish()

	return s.Client.GetBuildsDuration(ctx, filters)
}

func (s *tracingClient) GetFirstBuildTimes(ctx context.Context) ([]time.Time, error) {

	span, ctx := opentracing.StartSpanFromContext(ctx, s.getSpanName("GetFirstBuildTimes"))
	defer span.Finish()

	return s.Client.GetFirstBuildTimes(ctx)
}

func (s *tracingClient) GetFirstReleaseTimes(ctx context.Context) ([]time.Time, error) {

	span, ctx := opentracing.StartSpanFromContext(ctx, s.getSpanName("GetFirstReleaseTimes"))
	defer span.Finish()

	return s.Client.GetFirstReleaseTimes(ctx)
}

func (s *tracingClient) GetPipelineBuildsDurations(ctx context.Context, repoSource, repoOwner, repoName string, filters map[string][]string) ([]map[string]interface{}, error) {

	span, ctx := opentracing.StartSpanFromContext(ctx, s.getSpanName("GetPipelineBuildsDurations"))
	defer span.Finish()

	return s.Client.GetPipelineBuildsDurations(ctx, repoSource, repoOwner, repoName, filters)
}

func (s *tracingClient) GetPipelineReleasesDurations(ctx context.Context, repoSource, repoOwner, repoName string, filters map[string][]string) ([]map[string]interface{}, error) {

	span, ctx := opentracing.StartSpanFromContext(ctx, s.getSpanName("GetPipelineReleasesDurations"))
	defer span.Finish()

	return s.Client.GetPipelineReleasesDurations(ctx, repoSource, repoOwner, repoName, filters)
}

func (s *tracingClient) GetPipelineBuildsCPUUsageMeasurements(ctx context.Context, repoSource, repoOwner, repoName string, filters map[string][]string) ([]map[string]interface{}, error) {

	span, ctx := opentracing.StartSpanFromContext(ctx, s.getSpanName("GetPipelineBuildsCPUUsageMeasurements"))
	defer span.Finish()

	return s.Client.GetPipelineBuildsCPUUsageMeasurements(ctx, repoSource, repoOwner, repoName, filters)
}

func (s *tracingClient) GetPipelineReleasesCPUUsageMeasurements(ctx context.Context, repoSource, repoOwner, repoName string, filters map[string][]string) ([]map[string]interface{}, error) {

	span, ctx := opentracing.StartSpanFromContext(ctx, s.getSpanName("GetPipelineReleasesCPUUsageMeasurements"))
	defer span.Finish()

	return s.Client.GetPipelineReleasesCPUUsageMeasurements(ctx, repoSource, repoOwner, repoName, filters)
}

func (s *tracingClient) GetPipelineBuildsMemoryUsageMeasurements(ctx context.Context, repoSource, repoOwner, repoName string, filters map[string][]string) ([]map[string]interface{}, error) {

	span, ctx := opentracing.StartSpanFromContext(ctx, s.getSpanName("GetPipelineBuildsMemoryUsageMeasurements"))
	defer span.Finish()

	return s.Client.GetPipelineBuildsMemoryUsageMeasurements(ctx, repoSource, repoOwner, repoName, filters)
}

func (s *tracingClient) GetPipelineReleasesMemoryUsageMeasurements(ctx context.Context, repoSource, repoOwner, repoName string, filters map[string][]string) ([]map[string]interface{}, error) {

	span, ctx := opentracing.StartSpanFromContext(ctx, s.getSpanName("GetPipelineReleasesMemoryUsageMeasurements"))
	defer span.Finish()

	return s.Client.GetPipelineReleasesMemoryUsageMeasurements(ctx, repoSource, repoOwner, repoName, filters)
}

func (s *tracingClient) GetFrequentLabels(ctx context.Context, pageNumber, pageSize int, filters map[string][]string) ([]map[string]interface{}, error) {

	span, ctx := opentracing.StartSpanFromContext(ctx, s.getSpanName("GetFrequentLabels"))
	defer span.Finish()

	return s.Client.GetFrequentLabels(ctx, pageNumber, pageSize, filters)
}

func (s *tracingClient) GetFrequentLabelsCount(ctx context.Context, filters map[string][]string) (int, error) {

	span, ctx := opentracing.StartSpanFromContext(ctx, s.getSpanName("GetFrequentLabelsCount"))
	defer span.Finish()

	return s.Client.GetFrequentLabelsCount(ctx, filters)
}

func (s *tracingClient) GetPipelinesWithMostBuilds(ctx context.Context, pageNumber, pageSize int, filters map[string][]string) ([]map[string]interface{}, error) {

	span, ctx := opentracing.StartSpanFromContext(ctx, s.getSpanName("GetPipelinesWithMostBuilds"))
	defer span.Finish()

	return s.Client.GetPipelinesWithMostBuilds(ctx, pageNumber, pageSize, filters)
}

func (s *tracingClient) GetPipelinesWithMostBuildsCount(ctx context.Context, filters map[string][]string) (int, error) {

	span, ctx := opentracing.StartSpanFromContext(ctx, s.getSpanName("GetPipelinesWithMostBuildsCount"))
	defer span.Finish()

	return s.Client.GetPipelinesWithMostBuildsCount(ctx, filters)
}

func (s *tracingClient) GetPipelinesWithMostReleases(ctx context.Context, pageNumber, pageSize int, filters map[string][]string) ([]map[string]interface{}, error) {

	span, ctx := opentracing.StartSpanFromContext(ctx, s.getSpanName("GetPipelinesWithMostReleases"))
	defer span.Finish()

	return s.Client.GetPipelinesWithMostReleases(ctx, pageNumber, pageSize, filters)
}

func (s *tracingClient) GetPipelinesWithMostReleasesCount(ctx context.Context, filters map[string][]string) (int, error) {

	span, ctx := opentracing.StartSpanFromContext(ctx, s.getSpanName("GetPipelinesWithMostReleasesCount"))
	defer span.Finish()

	return s.Client.GetPipelinesWithMostReleasesCount(ctx, filters)
}

func (s *tracingClient) GetTriggers(ctx context.Context, triggerType, identifier, event string) ([]*contracts.Pipeline, error) {

	span, ctx := opentracing.StartSpanFromContext(ctx, s.getSpanName("GetTriggers"))
	defer span.Finish()

	return s.Client.GetTriggers(ctx, triggerType, identifier, event)
}

func (s *tracingClient) GetGitTriggers(ctx context.Context, gitEvent manifest.EstafetteGitEvent) ([]*contracts.Pipeline, error) {

	span, ctx := opentracing.StartSpanFromContext(ctx, s.getSpanName("GetGitTriggers"))
	defer span.Finish()

	return s.Client.GetGitTriggers(ctx, gitEvent)
}

func (s *tracingClient) GetPipelineTriggers(ctx context.Context, build contracts.Build, event string) ([]*contracts.Pipeline, error) {

	span, ctx := opentracing.StartSpanFromContext(ctx, s.getSpanName("GetPipelineTriggers"))
	defer span.Finish()

	return s.Client.GetPipelineTriggers(ctx, build, event)
}

func (s *tracingClient) GetReleaseTriggers(ctx context.Context, release contracts.Release, event string) ([]*contracts.Pipeline, error) {

	span, ctx := opentracing.StartSpanFromContext(ctx, s.getSpanName("GetReleaseTriggers"))
	defer span.Finish()

	return s.Client.GetReleaseTriggers(ctx, release, event)
}

func (s *tracingClient) GetPubSubTriggers(ctx context.Context, pubsubEvent manifest.EstafettePubSubEvent) ([]*contracts.Pipeline, error) {

	span, ctx := opentracing.StartSpanFromContext(ctx, s.getSpanName("GetPubSubTriggers"))
	defer span.Finish()

	return s.Client.GetPubSubTriggers(ctx, pubsubEvent)
}

func (s *tracingClient) GetCronTriggers(ctx context.Context) ([]*contracts.Pipeline, error) {

	span, ctx := opentracing.StartSpanFromContext(ctx, s.getSpanName("GetCronTriggers"))
	defer span.Finish()

	return s.Client.GetCronTriggers(ctx)
}

func (s *tracingClient) Rename(ctx context.Context, shortFromRepoSource, fromRepoSource, fromRepoOwner, fromRepoName, shortToRepoSource, toRepoSource, toRepoOwner, toRepoName string) error {

	span, ctx := opentracing.StartSpanFromContext(ctx, s.getSpanName("Rename"))
	defer span.Finish()

	return s.Client.Rename(ctx, shortFromRepoSource, fromRepoSource, fromRepoOwner, fromRepoName, shortToRepoSource, toRepoSource, toRepoOwner, toRepoName)
}

func (s *tracingClient) RenameBuildVersion(ctx context.Context, shortFromRepoSource, fromRepoOwner, fromRepoName, shortToRepoSource, toRepoOwner, toRepoName string) error {

	span, ctx := opentracing.StartSpanFromContext(ctx, s.getSpanName("RenameBuildVersion"))
	defer span.Finish()

	return s.Client.RenameBuildVersion(ctx, shortFromRepoSource, fromRepoOwner, fromRepoName, shortToRepoSource, toRepoOwner, toRepoName)
}

func (s *tracingClient) RenameBuilds(ctx context.Context, fromRepoSource, fromRepoOwner, fromRepoName, toRepoSource, toRepoOwner, toRepoName string) error {

	span, ctx := opentracing.StartSpanFromContext(ctx, s.getSpanName("RenameBuilds"))
	defer span.Finish()

	return s.Client.RenameBuilds(ctx, fromRepoSource, fromRepoOwner, fromRepoName, toRepoSource, toRepoOwner, toRepoName)
}

func (s *tracingClient) RenameBuildLogs(ctx context.Context, fromRepoSource, fromRepoOwner, fromRepoName, toRepoSource, toRepoOwner, toRepoName string) error {

	span, ctx := opentracing.StartSpanFromContext(ctx, s.getSpanName("RenameBuildLogs"))
	defer span.Finish()

	return s.Client.RenameBuildLogs(ctx, fromRepoSource, fromRepoOwner, fromRepoName, toRepoSource, toRepoOwner, toRepoName)
}

func (s *tracingClient) RenameReleases(ctx context.Context, fromRepoSource, fromRepoOwner, fromRepoName, toRepoSource, toRepoOwner, toRepoName string) error {

	span, ctx := opentracing.StartSpanFromContext(ctx, s.getSpanName("RenameReleases"))
	defer span.Finish()

	return s.Client.RenameReleases(ctx, fromRepoSource, fromRepoOwner, fromRepoName, toRepoSource, toRepoOwner, toRepoName)
}

func (s *tracingClient) RenameReleaseLogs(ctx context.Context, fromRepoSource, fromRepoOwner, fromRepoName, toRepoSource, toRepoOwner, toRepoName string) error {

	span, ctx := opentracing.StartSpanFromContext(ctx, s.getSpanName("RenameReleaseLogs"))
	defer span.Finish()

	return s.Client.RenameReleaseLogs(ctx, fromRepoSource, fromRepoOwner, fromRepoName, toRepoSource, toRepoOwner, toRepoName)
}

func (s *tracingClient) RenameComputedPipelines(ctx context.Context, fromRepoSource, fromRepoOwner, fromRepoName, toRepoSource, toRepoOwner, toRepoName string) error {

	span, ctx := opentracing.StartSpanFromContext(ctx, s.getSpanName("RenameComputedPipelines"))
	defer span.Finish()

	return s.Client.RenameComputedPipelines(ctx, fromRepoSource, fromRepoOwner, fromRepoName, toRepoSource, toRepoOwner, toRepoName)
}

func (s *tracingClient) RenameComputedReleases(ctx context.Context, fromRepoSource, fromRepoOwner, fromRepoName, toRepoSource, toRepoOwner, toRepoName string) error {

	span, ctx := opentracing.StartSpanFromContext(ctx, s.getSpanName("RenameComputedReleases"))
	defer span.Finish()

	return s.Client.RenameComputedReleases(ctx, fromRepoSource, fromRepoOwner, fromRepoName, toRepoSource, toRepoOwner, toRepoName)
}

func (s *tracingClient) getSpanName(funcName string) string {
	return "cockroachdb:" + funcName
}
