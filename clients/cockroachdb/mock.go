package cockroachdb

import (
	"context"
	"time"

	"github.com/estafette/estafette-ci-api/api"
	contracts "github.com/estafette/estafette-ci-contracts"
	manifest "github.com/estafette/estafette-ci-manifest"
)

type MockClient struct {
	ConnectFunc                                    func(ctx context.Context) (err error)
	ConnectWithDriverAndSourceFunc                 func(ctx context.Context, driverName, dataSourceName string) (err error)
	GetAutoIncrementFunc                           func(ctx context.Context, shortRepoSource, repoOwner, repoName string) (autoincrement int, err error)
	InsertBuildFunc                                func(ctx context.Context, build contracts.Build, jobResources JobResources) (b *contracts.Build, err error)
	UpdateBuildStatusFunc                          func(ctx context.Context, repoSource, repoOwner, repoName string, buildID int, buildStatus contracts.Status) (err error)
	UpdateBuildResourceUtilizationFunc             func(ctx context.Context, repoSource, repoOwner, repoName string, buildID int, jobResources JobResources) (err error)
	InsertReleaseFunc                              func(ctx context.Context, release contracts.Release, jobResources JobResources) (r *contracts.Release, err error)
	UpdateReleaseStatusFunc                        func(ctx context.Context, repoSource, repoOwner, repoName string, id int, releaseStatus contracts.Status) (err error)
	UpdateReleaseResourceUtilizationFunc           func(ctx context.Context, repoSource, repoOwner, repoName string, id int, jobResources JobResources) (err error)
	InsertBuildLogFunc                             func(ctx context.Context, buildLog contracts.BuildLog, writeLogToDatabase bool) (log contracts.BuildLog, err error)
	InsertReleaseLogFunc                           func(ctx context.Context, releaseLog contracts.ReleaseLog, writeLogToDatabase bool) (log contracts.ReleaseLog, err error)
	UpsertComputedPipelineFunc                     func(ctx context.Context, repoSource, repoOwner, repoName string) (err error)
	UpdateComputedPipelinePermissionsFunc          func(ctx context.Context, pipeline contracts.Pipeline) (err error)
	UpdateComputedPipelineFirstInsertedAtFunc      func(ctx context.Context, repoSource, repoOwner, repoName string) (err error)
	UpsertComputedReleaseFunc                      func(ctx context.Context, repoSource, repoOwner, repoName, releaseName, releaseAction string) (err error)
	UpdateComputedReleaseFirstInsertedAtFunc       func(ctx context.Context, repoSource, repoOwner, repoName, releaseName, releaseAction string) (err error)
	ArchiveComputedPipelineFunc                    func(ctx context.Context, repoSource, repoOwner, repoName string) (err error)
	UnarchiveComputedPipelineFunc                  func(ctx context.Context, repoSource, repoOwner, repoName string) (err error)
	GetPipelinesFunc                               func(ctx context.Context, pageNumber, pageSize int, filters map[api.FilterType][]string, sortings []api.OrderField, optimized bool) (pipelines []*contracts.Pipeline, err error)
	GetPipelinesByRepoNameFunc                     func(ctx context.Context, repoName string, optimized bool) (pipelines []*contracts.Pipeline, err error)
	GetPipelinesCountFunc                          func(ctx context.Context, filters map[api.FilterType][]string) (count int, err error)
	GetPipelineFunc                                func(ctx context.Context, repoSource, repoOwner, repoName string, filters map[api.FilterType][]string, optimized bool) (pipeline *contracts.Pipeline, err error)
	GetPipelineRecentBuildsFunc                    func(ctx context.Context, repoSource, repoOwner, repoName string, optimized bool) (builds []*contracts.Build, err error)
	GetPipelineBuildsFunc                          func(ctx context.Context, repoSource, repoOwner, repoName string, pageNumber, pageSize int, filters map[api.FilterType][]string, sortings []api.OrderField, optimized bool) (builds []*contracts.Build, err error)
	GetPipelineBuildsCountFunc                     func(ctx context.Context, repoSource, repoOwner, repoName string, filters map[api.FilterType][]string) (count int, err error)
	GetPipelineBuildFunc                           func(ctx context.Context, repoSource, repoOwner, repoName, repoRevision string, optimized bool) (build *contracts.Build, err error)
	GetPipelineBuildByIDFunc                       func(ctx context.Context, repoSource, repoOwner, repoName string, id int, optimized bool) (build *contracts.Build, err error)
	GetLastPipelineBuildFunc                       func(ctx context.Context, repoSource, repoOwner, repoName string, optimized bool) (build *contracts.Build, err error)
	GetFirstPipelineBuildFunc                      func(ctx context.Context, repoSource, repoOwner, repoName string, optimized bool) (build *contracts.Build, err error)
	GetLastPipelineBuildForBranchFunc              func(ctx context.Context, repoSource, repoOwner, repoName, branch string) (build *contracts.Build, err error)
	GetLastPipelineReleasesFunc                    func(ctx context.Context, repoSource, repoOwner, repoName, releaseName, releaseAction string, pageSize int) (releases []*contracts.Release, err error)
	GetFirstPipelineReleaseFunc                    func(ctx context.Context, repoSource, repoOwner, repoName, releaseName, releaseAction string) (release *contracts.Release, err error)
	GetPipelineBuildsByVersionFunc                 func(ctx context.Context, repoSource, repoOwner, repoName, buildVersion string, statuses []contracts.Status, limit uint64, optimized bool) (builds []*contracts.Build, err error)
	GetPipelineBuildLogsFunc                       func(ctx context.Context, repoSource, repoOwner, repoName, repoBranch, repoRevision, buildID string, readLogFromDatabase bool) (buildlog *contracts.BuildLog, err error)
	GetPipelineBuildLogsPerPageFunc                func(ctx context.Context, repoSource, repoOwner, repoName, repoBranch, repoRevision, buildID string, readLogFromDatabase bool, pageNumber int, pageSize int) (buildLogs []*contracts.BuildLog, err error)
	GetPipelineBuildLogsCountFunc                  func(ctx context.Context, repoSource, repoOwner, repoName, repoBranch, repoRevision, buildID string) (count int, err error)
	GetPipelineBuildMaxResourceUtilizationFunc     func(ctx context.Context, repoSource, repoOwner, repoName string, lastNRecords int) (jobresources JobResources, count int, err error)
	GetPipelineReleasesFunc                        func(ctx context.Context, repoSource, repoOwner, repoName string, pageNumber, pageSize int, filters map[api.FilterType][]string, sortings []api.OrderField) (releases []*contracts.Release, err error)
	GetPipelineReleasesCountFunc                   func(ctx context.Context, repoSource, repoOwner, repoName string, filters map[api.FilterType][]string) (count int, err error)
	GetPipelineReleaseFunc                         func(ctx context.Context, repoSource, repoOwner, repoName string, id int) (release *contracts.Release, err error)
	GetPipelineLastReleasesByNameFunc              func(ctx context.Context, repoSource, repoOwner, repoName, releaseName string, actions []string) (releases []contracts.Release, err error)
	GetPipelineReleaseLogsFunc                     func(ctx context.Context, repoSource, repoOwner, repoName string, releaseID int, readLogFromDatabase bool) (releaselog *contracts.ReleaseLog, err error)
	GetPipelineReleaseLogsPerPageFunc              func(ctx context.Context, repoSource, repoOwner, repoName string, releaseID int, readLogFromDatabase bool, pageNumber int, pageSize int) (releaselogs []*contracts.ReleaseLog, err error)
	GetPipelineReleaseLogsCountFunc                func(ctx context.Context, repoSource, repoOwner, repoName string, releaseID int) (count int, err error)
	GetPipelineReleaseMaxResourceUtilizationFunc   func(ctx context.Context, repoSource, repoOwner, repoName, targetName string, lastNRecords int) (jobresources JobResources, count int, err error)
	GetBuildsCountFunc                             func(ctx context.Context, filters map[api.FilterType][]string) (count int, err error)
	GetReleasesCountFunc                           func(ctx context.Context, filters map[api.FilterType][]string) (count int, err error)
	GetBuildsDurationFunc                          func(ctx context.Context, filters map[api.FilterType][]string) (duration time.Duration, err error)
	GetFirstBuildTimesFunc                         func(ctx context.Context) (times []time.Time, err error)
	GetFirstReleaseTimesFunc                       func(ctx context.Context) (times []time.Time, err error)
	GetPipelineBuildsDurationsFunc                 func(ctx context.Context, repoSource, repoOwner, repoName string, filters map[api.FilterType][]string) (durations []map[string]interface{}, err error)
	GetPipelineReleasesDurationsFunc               func(ctx context.Context, repoSource, repoOwner, repoName string, filters map[api.FilterType][]string) (durations []map[string]interface{}, err error)
	GetPipelineBuildsCPUUsageMeasurementsFunc      func(ctx context.Context, repoSource, repoOwner, repoName string, filters map[api.FilterType][]string) (measurements []map[string]interface{}, err error)
	GetPipelineReleasesCPUUsageMeasurementsFunc    func(ctx context.Context, repoSource, repoOwner, repoName string, filters map[api.FilterType][]string) (measurements []map[string]interface{}, err error)
	GetPipelineBuildsMemoryUsageMeasurementsFunc   func(ctx context.Context, repoSource, repoOwner, repoName string, filters map[api.FilterType][]string) (measurements []map[string]interface{}, err error)
	GetPipelineReleasesMemoryUsageMeasurementsFunc func(ctx context.Context, repoSource, repoOwner, repoName string, filters map[api.FilterType][]string) (measurements []map[string]interface{}, err error)
	GetLabelValuesFunc                             func(ctx context.Context, labelKey string) (labels []map[string]interface{}, err error)
	GetFrequentLabelsFunc                          func(ctx context.Context, pageNumber, pageSize int, filters map[api.FilterType][]string) (labels []map[string]interface{}, err error)
	GetFrequentLabelsCountFunc                     func(ctx context.Context, filters map[api.FilterType][]string) (count int, err error)
	GetPipelinesWithMostBuildsFunc                 func(ctx context.Context, pageNumber, pageSize int, filters map[api.FilterType][]string) (pipelines []map[string]interface{}, err error)
	GetPipelinesWithMostBuildsCountFunc            func(ctx context.Context, filters map[api.FilterType][]string) (count int, err error)
	GetPipelinesWithMostReleasesFunc               func(ctx context.Context, pageNumber, pageSize int, filters map[api.FilterType][]string) (pipelines []map[string]interface{}, err error)
	GetPipelinesWithMostReleasesCountFunc          func(ctx context.Context, filters map[api.FilterType][]string) (count int, err error)
	GetTriggersFunc                                func(ctx context.Context, triggerType, identifier, event string) (pipelines []*contracts.Pipeline, err error)
	GetGitTriggersFunc                             func(ctx context.Context, gitEvent manifest.EstafetteGitEvent) (pipelines []*contracts.Pipeline, err error)
	GetPipelineTriggersFunc                        func(ctx context.Context, build contracts.Build, event string) (pipelines []*contracts.Pipeline, err error)
	GetReleaseTriggersFunc                         func(ctx context.Context, release contracts.Release, event string) (pipelines []*contracts.Pipeline, err error)
	GetPubSubTriggersFunc                          func(ctx context.Context, pubsubEvent manifest.EstafettePubSubEvent) (pipelines []*contracts.Pipeline, err error)
	GetCronTriggersFunc                            func(ctx context.Context) (pipelines []*contracts.Pipeline, err error)
	RenameFunc                                     func(ctx context.Context, shortFromRepoSource, fromRepoSource, fromRepoOwner, fromRepoName, shortToRepoSource, toRepoSource, toRepoOwner, toRepoName string) (err error)
	RenameBuildVersionFunc                         func(ctx context.Context, shortFromRepoSource, fromRepoOwner, fromRepoName, shortToRepoSource, toRepoOwner, toRepoName string) (err error)
	RenameBuildsFunc                               func(ctx context.Context, fromRepoSource, fromRepoOwner, fromRepoName, toRepoSource, toRepoOwner, toRepoName string) (err error)
	RenameBuildLogsFunc                            func(ctx context.Context, fromRepoSource, fromRepoOwner, fromRepoName, toRepoSource, toRepoOwner, toRepoName string) (err error)
	RenameReleasesFunc                             func(ctx context.Context, fromRepoSource, fromRepoOwner, fromRepoName, toRepoSource, toRepoOwner, toRepoName string) (err error)
	RenameReleaseLogsFunc                          func(ctx context.Context, fromRepoSource, fromRepoOwner, fromRepoName, toRepoSource, toRepoOwner, toRepoName string) (err error)
	RenameComputedPipelinesFunc                    func(ctx context.Context, fromRepoSource, fromRepoOwner, fromRepoName, toRepoSource, toRepoOwner, toRepoName string) (err error)
	RenameComputedReleasesFunc                     func(ctx context.Context, fromRepoSource, fromRepoOwner, fromRepoName, toRepoSource, toRepoOwner, toRepoName string) (err error)

	InsertUserFunc        func(ctx context.Context, user contracts.User) (u *contracts.User, err error)
	UpdateUserFunc        func(ctx context.Context, user contracts.User) (err error)
	DeleteUserFunc        func(ctx context.Context, user contracts.User) (err error)
	GetUserByIdentityFunc func(ctx context.Context, identity contracts.UserIdentity) (user *contracts.User, err error)
	GetUserByIDFunc       func(ctx context.Context, id string, filters map[api.FilterType][]string) (user *contracts.User, err error)
	GetUsersFunc          func(ctx context.Context, pageNumber, pageSize int, filters map[api.FilterType][]string, sortings []api.OrderField) (users []*contracts.User, err error)
	GetUsersCountFunc     func(ctx context.Context, filters map[api.FilterType][]string) (count int, err error)

	InsertGroupFunc        func(ctx context.Context, group contracts.Group) (g *contracts.Group, err error)
	UpdateGroupFunc        func(ctx context.Context, group contracts.Group) (err error)
	DeleteGroupFunc        func(ctx context.Context, group contracts.Group) (err error)
	GetGroupByIdentityFunc func(ctx context.Context, identity contracts.GroupIdentity) (group *contracts.Group, err error)
	GetGroupByIDFunc       func(ctx context.Context, id string, filters map[api.FilterType][]string) (group *contracts.Group, err error)
	GetGroupsFunc          func(ctx context.Context, pageNumber, pageSize int, filters map[api.FilterType][]string, sortings []api.OrderField) (groups []*contracts.Group, err error)
	GetGroupsCountFunc     func(ctx context.Context, filters map[api.FilterType][]string) (count int, err error)

	InsertOrganizationFunc        func(ctx context.Context, organization contracts.Organization) (o *contracts.Organization, err error)
	UpdateOrganizationFunc        func(ctx context.Context, organization contracts.Organization) (err error)
	DeleteOrganizationFunc        func(ctx context.Context, organization contracts.Organization) (err error)
	GetOrganizationByIdentityFunc func(ctx context.Context, identity contracts.OrganizationIdentity) (organization *contracts.Organization, err error)
	GetOrganizationByIDFunc       func(ctx context.Context, id string) (organization *contracts.Organization, err error)
	GetOrganizationByNameFunc     func(ctx context.Context, name string) (organization *contracts.Organization, err error)
	GetOrganizationsFunc          func(ctx context.Context, pageNumber, pageSize int, filters map[api.FilterType][]string, sortings []api.OrderField) (organizations []*contracts.Organization, err error)
	GetOrganizationsCountFunc     func(ctx context.Context, filters map[api.FilterType][]string) (count int, err error)

	InsertClientFunc        func(ctx context.Context, client contracts.Client) (cl *contracts.Client, err error)
	UpdateClientFunc        func(ctx context.Context, client contracts.Client) (err error)
	DeleteClientFunc        func(ctx context.Context, client contracts.Client) (err error)
	GetClientByClientIDFunc func(ctx context.Context, clientID string) (client *contracts.Client, err error)
	GetClientByIDFunc       func(ctx context.Context, id string) (client *contracts.Client, err error)
	GetClientsFunc          func(ctx context.Context, pageNumber, pageSize int, filters map[api.FilterType][]string, sortings []api.OrderField) (clients []*contracts.Client, err error)
	GetClientsCountFunc     func(ctx context.Context, filters map[api.FilterType][]string) (count int, err error)

	InsertCatalogEntityFunc     func(ctx context.Context, catalogEntity contracts.CatalogEntity) (insertedCatalogEntity *contracts.CatalogEntity, err error)
	UpdateCatalogEntityFunc     func(ctx context.Context, catalogEntity contracts.CatalogEntity) (err error)
	DeleteCatalogEntityFunc     func(ctx context.Context, id string) (err error)
	GetCatalogEntityByIDFunc    func(ctx context.Context, id string) (catalogEntity *contracts.CatalogEntity, err error)
	GetCatalogEntitiesFunc      func(ctx context.Context, pageNumber, pageSize int, filters map[api.FilterType][]string, sortings []api.OrderField) (catalogEntities []*contracts.CatalogEntity, err error)
	GetCatalogEntitiesCountFunc func(ctx context.Context, filters map[api.FilterType][]string) (count int, err error)

	GetCatalogEntityParentKeysFunc        func(ctx context.Context, pageNumber, pageSize int, filters map[api.FilterType][]string, sortings []api.OrderField) (keys []map[string]interface{}, err error)
	GetCatalogEntityParentKeysCountFunc   func(ctx context.Context, filters map[api.FilterType][]string) (count int, err error)
	GetCatalogEntityParentValuesFunc      func(ctx context.Context, pageNumber, pageSize int, filters map[api.FilterType][]string, sortings []api.OrderField) (values []map[string]interface{}, err error)
	GetCatalogEntityParentValuesCountFunc func(ctx context.Context, filters map[api.FilterType][]string) (count int, err error)
	GetCatalogEntityKeysFunc              func(ctx context.Context, pageNumber, pageSize int, filters map[api.FilterType][]string, sortings []api.OrderField) (keys []map[string]interface{}, err error)
	GetCatalogEntityKeysCountFunc         func(ctx context.Context, filters map[api.FilterType][]string) (count int, err error)
	GetCatalogEntityValuesFunc            func(ctx context.Context, pageNumber, pageSize int, filters map[api.FilterType][]string, sortings []api.OrderField) (values []map[string]interface{}, err error)
	GetCatalogEntityValuesCountFunc       func(ctx context.Context, filters map[api.FilterType][]string) (count int, err error)
	GetCatalogEntityLabelsFunc            func(ctx context.Context, pageNumber, pageSize int, filters map[api.FilterType][]string) (labels []map[string]interface{}, err error)
	GetCatalogEntityLabelsCountFunc       func(ctx context.Context, filters map[api.FilterType][]string) (count int, err error)
}

func (c MockClient) Connect(ctx context.Context) (err error) {
	if c.ConnectFunc == nil {
		return
	}
	return c.ConnectFunc(ctx)
}

func (c MockClient) ConnectWithDriverAndSource(ctx context.Context, driverName, dataSourceName string) (err error) {
	if c.ConnectWithDriverAndSourceFunc == nil {
		return
	}
	return c.ConnectWithDriverAndSourceFunc(ctx, driverName, dataSourceName)
}

func (c MockClient) GetAutoIncrement(ctx context.Context, shortRepoSource, repoOwner, repoName string) (autoincrement int, err error) {
	if c.GetAutoIncrementFunc == nil {
		return
	}
	return c.GetAutoIncrementFunc(ctx, shortRepoSource, repoOwner, repoName)
}

func (c MockClient) InsertBuild(ctx context.Context, build contracts.Build, jobResources JobResources) (b *contracts.Build, err error) {
	if c.InsertBuildFunc == nil {
		return
	}
	return c.InsertBuildFunc(ctx, build, jobResources)
}

func (c MockClient) UpdateBuildStatus(ctx context.Context, repoSource, repoOwner, repoName string, buildID int, buildStatus contracts.Status) (err error) {
	if c.UpdateBuildStatusFunc == nil {
		return
	}
	return c.UpdateBuildStatusFunc(ctx, repoSource, repoOwner, repoName, buildID, buildStatus)
}

func (c MockClient) UpdateBuildResourceUtilization(ctx context.Context, repoSource, repoOwner, repoName string, buildID int, jobResources JobResources) (err error) {
	if c.UpdateBuildResourceUtilizationFunc == nil {
		return
	}
	return c.UpdateBuildResourceUtilizationFunc(ctx, repoSource, repoOwner, repoName, buildID, jobResources)
}

func (c MockClient) InsertRelease(ctx context.Context, release contracts.Release, jobResources JobResources) (r *contracts.Release, err error) {
	if c.InsertReleaseFunc == nil {
		return
	}
	return c.InsertReleaseFunc(ctx, release, jobResources)
}

func (c MockClient) UpdateReleaseStatus(ctx context.Context, repoSource, repoOwner, repoName string, id int, releaseStatus contracts.Status) (err error) {
	if c.UpdateReleaseStatusFunc == nil {
		return
	}
	return c.UpdateReleaseStatusFunc(ctx, repoSource, repoOwner, repoName, id, releaseStatus)
}

func (c MockClient) UpdateReleaseResourceUtilization(ctx context.Context, repoSource, repoOwner, repoName string, id int, jobResources JobResources) (err error) {
	if c.UpdateReleaseResourceUtilizationFunc == nil {
		return
	}
	return c.UpdateReleaseResourceUtilizationFunc(ctx, repoSource, repoOwner, repoName, id, jobResources)
}

func (c MockClient) InsertBuildLog(ctx context.Context, buildLog contracts.BuildLog, writeLogToDatabase bool) (log contracts.BuildLog, err error) {
	if c.InsertBuildLogFunc == nil {
		return
	}
	return c.InsertBuildLogFunc(ctx, buildLog, writeLogToDatabase)
}

func (c MockClient) InsertReleaseLog(ctx context.Context, releaseLog contracts.ReleaseLog, writeLogToDatabase bool) (log contracts.ReleaseLog, err error) {
	if c.InsertReleaseLogFunc == nil {
		return
	}
	return c.InsertReleaseLogFunc(ctx, releaseLog, writeLogToDatabase)
}

func (c MockClient) UpsertComputedPipeline(ctx context.Context, repoSource, repoOwner, repoName string) (err error) {
	if c.UpsertComputedPipelineFunc == nil {
		return
	}
	return c.UpsertComputedPipelineFunc(ctx, repoSource, repoOwner, repoName)
}

func (c MockClient) UpdateComputedPipelinePermissions(ctx context.Context, pipeline contracts.Pipeline) (err error) {
	if c.UpdateComputedPipelinePermissionsFunc == nil {
		return
	}
	return c.UpdateComputedPipelinePermissionsFunc(ctx, pipeline)
}

func (c MockClient) UpdateComputedPipelineFirstInsertedAt(ctx context.Context, repoSource, repoOwner, repoName string) (err error) {
	if c.UpdateComputedPipelineFirstInsertedAtFunc == nil {
		return
	}
	return c.UpdateComputedPipelineFirstInsertedAtFunc(ctx, repoSource, repoOwner, repoName)
}

func (c MockClient) UpsertComputedRelease(ctx context.Context, repoSource, repoOwner, repoName, releaseName, releaseAction string) (err error) {
	if c.UpsertComputedReleaseFunc == nil {
		return
	}
	return c.UpsertComputedReleaseFunc(ctx, repoSource, repoOwner, repoName, releaseName, releaseAction)
}

func (c MockClient) UpdateComputedReleaseFirstInsertedAt(ctx context.Context, repoSource, repoOwner, repoName, releaseName, releaseAction string) (err error) {
	if c.UpdateComputedReleaseFirstInsertedAtFunc == nil {
		return
	}
	return c.UpdateComputedReleaseFirstInsertedAtFunc(ctx, repoSource, repoOwner, repoName, releaseName, releaseAction)
}

func (c MockClient) ArchiveComputedPipeline(ctx context.Context, repoSource, repoOwner, repoName string) (err error) {
	if c.ArchiveComputedPipelineFunc == nil {
		return
	}
	return c.ArchiveComputedPipelineFunc(ctx, repoSource, repoOwner, repoName)
}

func (c MockClient) UnarchiveComputedPipeline(ctx context.Context, repoSource, repoOwner, repoName string) (err error) {
	if c.UnarchiveComputedPipelineFunc == nil {
		return
	}
	return c.UnarchiveComputedPipelineFunc(ctx, repoSource, repoOwner, repoName)
}

func (c MockClient) GetPipelines(ctx context.Context, pageNumber, pageSize int, filters map[api.FilterType][]string, sortings []api.OrderField, optimized bool) (pipelines []*contracts.Pipeline, err error) {
	if c.GetPipelinesFunc == nil {
		return
	}
	return c.GetPipelinesFunc(ctx, pageNumber, pageSize, filters, sortings, optimized)
}

func (c MockClient) GetPipelinesByRepoName(ctx context.Context, repoName string, optimized bool) (pipelines []*contracts.Pipeline, err error) {
	if c.GetPipelinesByRepoNameFunc == nil {
		return
	}
	return c.GetPipelinesByRepoNameFunc(ctx, repoName, optimized)
}

func (c MockClient) GetPipelinesCount(ctx context.Context, filters map[api.FilterType][]string) (count int, err error) {
	if c.GetPipelinesCountFunc == nil {
		return
	}
	return c.GetPipelinesCountFunc(ctx, filters)
}

func (c MockClient) GetPipeline(ctx context.Context, repoSource, repoOwner, repoName string, filters map[api.FilterType][]string, optimized bool) (pipeline *contracts.Pipeline, err error) {
	if c.GetPipelineFunc == nil {
		return
	}
	return c.GetPipelineFunc(ctx, repoSource, repoOwner, repoName, filters, optimized)
}

func (c MockClient) GetPipelineRecentBuilds(ctx context.Context, repoSource, repoOwner, repoName string, optimized bool) (builds []*contracts.Build, err error) {
	if c.GetPipelineRecentBuildsFunc == nil {
		return
	}

	return c.GetPipelineRecentBuildsFunc(ctx, repoSource, repoOwner, repoName, optimized)
}

func (c MockClient) GetPipelineBuilds(ctx context.Context, repoSource, repoOwner, repoName string, pageNumber, pageSize int, filters map[api.FilterType][]string, sortings []api.OrderField, optimized bool) (builds []*contracts.Build, err error) {
	if c.GetPipelineBuildsFunc == nil {
		return
	}
	return c.GetPipelineBuildsFunc(ctx, repoSource, repoOwner, repoName, pageNumber, pageSize, filters, sortings, optimized)
}

func (c MockClient) GetPipelineBuildsCount(ctx context.Context, repoSource, repoOwner, repoName string, filters map[api.FilterType][]string) (count int, err error) {
	if c.GetPipelineBuildsCountFunc == nil {
		return
	}
	return c.GetPipelineBuildsCountFunc(ctx, repoSource, repoOwner, repoName, filters)
}

func (c MockClient) GetPipelineBuild(ctx context.Context, repoSource, repoOwner, repoName, repoRevision string, optimized bool) (build *contracts.Build, err error) {
	if c.GetPipelineBuildFunc == nil {
		return
	}
	return c.GetPipelineBuildFunc(ctx, repoSource, repoOwner, repoName, repoRevision, optimized)
}

func (c MockClient) GetPipelineBuildByID(ctx context.Context, repoSource, repoOwner, repoName string, id int, optimized bool) (build *contracts.Build, err error) {
	if c.GetPipelineBuildByIDFunc == nil {
		return
	}
	return c.GetPipelineBuildByIDFunc(ctx, repoSource, repoOwner, repoName, id, optimized)
}

func (c MockClient) GetLastPipelineBuild(ctx context.Context, repoSource, repoOwner, repoName string, optimized bool) (build *contracts.Build, err error) {
	if c.GetLastPipelineBuildFunc == nil {
		return
	}
	return c.GetLastPipelineBuildFunc(ctx, repoSource, repoOwner, repoName, optimized)
}

func (c MockClient) GetFirstPipelineBuild(ctx context.Context, repoSource, repoOwner, repoName string, optimized bool) (build *contracts.Build, err error) {
	if c.GetFirstPipelineBuildFunc == nil {
		return
	}
	return c.GetFirstPipelineBuildFunc(ctx, repoSource, repoOwner, repoName, optimized)
}

func (c MockClient) GetLastPipelineBuildForBranch(ctx context.Context, repoSource, repoOwner, repoName, branch string) (build *contracts.Build, err error) {
	if c.GetLastPipelineBuildForBranchFunc == nil {
		return
	}
	return c.GetLastPipelineBuildForBranchFunc(ctx, repoSource, repoOwner, repoName, branch)
}

func (c MockClient) GetLastPipelineReleases(ctx context.Context, repoSource, repoOwner, repoName, releaseName, releaseAction string, pageSize int) (releases []*contracts.Release, err error) {
	if c.GetLastPipelineReleasesFunc == nil {
		return
	}
	return c.GetLastPipelineReleasesFunc(ctx, repoSource, repoOwner, repoName, releaseName, releaseAction, pageSize)
}

func (c MockClient) GetFirstPipelineRelease(ctx context.Context, repoSource, repoOwner, repoName, releaseName, releaseAction string) (release *contracts.Release, err error) {
	if c.GetFirstPipelineReleaseFunc == nil {
		return
	}
	return c.GetFirstPipelineReleaseFunc(ctx, repoSource, repoOwner, repoName, releaseName, releaseAction)
}

func (c MockClient) GetPipelineBuildsByVersion(ctx context.Context, repoSource, repoOwner, repoName, buildVersion string, statuses []contracts.Status, limit uint64, optimized bool) (builds []*contracts.Build, err error) {
	if c.GetPipelineBuildsByVersionFunc == nil {
		return
	}
	return c.GetPipelineBuildsByVersionFunc(ctx, repoSource, repoOwner, repoName, buildVersion, statuses, limit, optimized)
}

func (c MockClient) GetPipelineBuildLogs(ctx context.Context, repoSource, repoOwner, repoName, repoBranch, repoRevision, buildID string, readLogFromDatabase bool) (buildlog *contracts.BuildLog, err error) {
	if c.GetPipelineBuildLogsFunc == nil {
		return
	}
	return c.GetPipelineBuildLogsFunc(ctx, repoSource, repoOwner, repoName, repoBranch, repoRevision, buildID, readLogFromDatabase)
}

func (c MockClient) GetPipelineBuildLogsPerPage(ctx context.Context, repoSource, repoOwner, repoName, repoBranch, repoRevision, buildID string, readLogFromDatabase bool, pageNumber int, pageSize int) (buildLogs []*contracts.BuildLog, err error) {
	if c.GetPipelineBuildLogsPerPageFunc == nil {
		return
	}
	return c.GetPipelineBuildLogsPerPageFunc(ctx, repoSource, repoOwner, repoName, repoBranch, repoRevision, buildID, readLogFromDatabase, pageNumber, pageSize)
}

func (c MockClient) GetPipelineBuildLogsCount(ctx context.Context, repoSource, repoOwner, repoName, repoBranch, repoRevision, buildID string) (count int, err error) {
	if c.GetPipelineBuildLogsCountFunc == nil {
		return
	}
	return c.GetPipelineBuildLogsCountFunc(ctx, repoSource, repoOwner, repoName, repoBranch, repoRevision, buildID)
}

func (c MockClient) GetPipelineBuildMaxResourceUtilization(ctx context.Context, repoSource, repoOwner, repoName string, lastNRecords int) (jobresources JobResources, count int, err error) {
	if c.GetPipelineBuildMaxResourceUtilizationFunc == nil {
		return
	}
	return c.GetPipelineBuildMaxResourceUtilizationFunc(ctx, repoSource, repoOwner, repoName, lastNRecords)
}

func (c MockClient) GetPipelineReleases(ctx context.Context, repoSource, repoOwner, repoName string, pageNumber, pageSize int, filters map[api.FilterType][]string, sortings []api.OrderField) (releases []*contracts.Release, err error) {
	if c.GetPipelineReleasesFunc == nil {
		return
	}
	return c.GetPipelineReleasesFunc(ctx, repoSource, repoOwner, repoName, pageNumber, pageSize, filters, sortings)
}

func (c MockClient) GetPipelineReleasesCount(ctx context.Context, repoSource, repoOwner, repoName string, filters map[api.FilterType][]string) (count int, err error) {
	if c.GetPipelineReleasesCountFunc == nil {
		return
	}
	return c.GetPipelineReleasesCountFunc(ctx, repoSource, repoOwner, repoName, filters)
}

func (c MockClient) GetPipelineRelease(ctx context.Context, repoSource, repoOwner, repoName string, id int) (release *contracts.Release, err error) {
	if c.GetPipelineReleaseFunc == nil {
		return
	}
	return c.GetPipelineReleaseFunc(ctx, repoSource, repoOwner, repoName, id)
}

func (c MockClient) GetPipelineLastReleasesByName(ctx context.Context, repoSource, repoOwner, repoName, releaseName string, actions []string) (releases []contracts.Release, err error) {
	if c.GetPipelineLastReleasesByNameFunc == nil {
		return
	}
	return c.GetPipelineLastReleasesByNameFunc(ctx, repoSource, repoOwner, repoName, releaseName, actions)
}

func (c MockClient) GetPipelineReleaseLogs(ctx context.Context, repoSource, repoOwner, repoName string, releaseID int, readLogFromDatabase bool) (releaselog *contracts.ReleaseLog, err error) {
	if c.GetPipelineReleaseLogsFunc == nil {
		return
	}
	return c.GetPipelineReleaseLogsFunc(ctx, repoSource, repoOwner, repoName, releaseID, readLogFromDatabase)
}

func (c MockClient) GetPipelineReleaseLogsPerPage(ctx context.Context, repoSource, repoOwner, repoName string, releaseID int, readLogFromDatabase bool, pageNumber int, pageSize int) (releaselogs []*contracts.ReleaseLog, err error) {
	if c.GetPipelineReleaseLogsPerPageFunc == nil {
		return
	}
	return c.GetPipelineReleaseLogsPerPageFunc(ctx, repoSource, repoOwner, repoName, releaseID, readLogFromDatabase, pageNumber, pageSize)
}

func (c MockClient) GetPipelineReleaseLogsCount(ctx context.Context, repoSource, repoOwner, repoName string, releaseID int) (count int, err error) {
	if c.GetPipelineReleaseLogsCountFunc == nil {
		return
	}
	return c.GetPipelineReleaseLogsCountFunc(ctx, repoSource, repoOwner, repoName, releaseID)
}

func (c MockClient) GetPipelineReleaseMaxResourceUtilization(ctx context.Context, repoSource, repoOwner, repoName, targetName string, lastNRecords int) (jobresources JobResources, count int, err error) {
	if c.GetPipelineReleaseMaxResourceUtilizationFunc == nil {
		return
	}
	return c.GetPipelineReleaseMaxResourceUtilizationFunc(ctx, repoSource, repoOwner, repoName, targetName, lastNRecords)
}

func (c MockClient) GetBuildsCount(ctx context.Context, filters map[api.FilterType][]string) (count int, err error) {
	if c.GetBuildsCountFunc == nil {
		return
	}
	return c.GetBuildsCountFunc(ctx, filters)
}

func (c MockClient) GetReleasesCount(ctx context.Context, filters map[api.FilterType][]string) (count int, err error) {
	if c.GetReleasesCountFunc == nil {
		return
	}
	return c.GetReleasesCountFunc(ctx, filters)
}

func (c MockClient) GetBuildsDuration(ctx context.Context, filters map[api.FilterType][]string) (duration time.Duration, err error) {
	if c.GetBuildsDurationFunc == nil {
		return
	}
	return c.GetBuildsDurationFunc(ctx, filters)
}

func (c MockClient) GetFirstBuildTimes(ctx context.Context) (times []time.Time, err error) {
	if c.GetFirstBuildTimesFunc == nil {
		return
	}
	return c.GetFirstBuildTimesFunc(ctx)
}

func (c MockClient) GetFirstReleaseTimes(ctx context.Context) (times []time.Time, err error) {
	if c.GetFirstReleaseTimesFunc == nil {
		return
	}
	return c.GetFirstReleaseTimesFunc(ctx)
}

func (c MockClient) GetPipelineBuildsDurations(ctx context.Context, repoSource, repoOwner, repoName string, filters map[api.FilterType][]string) (durations []map[string]interface{}, err error) {
	if c.GetPipelineBuildsDurationsFunc == nil {
		return
	}
	return c.GetPipelineBuildsDurationsFunc(ctx, repoSource, repoOwner, repoName, filters)
}

func (c MockClient) GetPipelineReleasesDurations(ctx context.Context, repoSource, repoOwner, repoName string, filters map[api.FilterType][]string) (durations []map[string]interface{}, err error) {
	if c.GetPipelineReleasesDurationsFunc == nil {
		return
	}
	return c.GetPipelineReleasesDurationsFunc(ctx, repoSource, repoOwner, repoName, filters)
}

func (c MockClient) GetPipelineBuildsCPUUsageMeasurements(ctx context.Context, repoSource, repoOwner, repoName string, filters map[api.FilterType][]string) (measurements []map[string]interface{}, err error) {
	if c.GetPipelineBuildsCPUUsageMeasurementsFunc == nil {
		return
	}
	return c.GetPipelineBuildsCPUUsageMeasurementsFunc(ctx, repoSource, repoOwner, repoName, filters)
}

func (c MockClient) GetPipelineReleasesCPUUsageMeasurements(ctx context.Context, repoSource, repoOwner, repoName string, filters map[api.FilterType][]string) (measurements []map[string]interface{}, err error) {
	if c.GetPipelineReleasesCPUUsageMeasurementsFunc == nil {
		return
	}
	return c.GetPipelineReleasesCPUUsageMeasurementsFunc(ctx, repoSource, repoOwner, repoName, filters)
}

func (c MockClient) GetPipelineBuildsMemoryUsageMeasurements(ctx context.Context, repoSource, repoOwner, repoName string, filters map[api.FilterType][]string) (measurements []map[string]interface{}, err error) {
	if c.GetPipelineBuildsMemoryUsageMeasurementsFunc == nil {
		return
	}
	return c.GetPipelineBuildsMemoryUsageMeasurementsFunc(ctx, repoSource, repoOwner, repoName, filters)
}

func (c MockClient) GetPipelineReleasesMemoryUsageMeasurements(ctx context.Context, repoSource, repoOwner, repoName string, filters map[api.FilterType][]string) (measurements []map[string]interface{}, err error) {
	if c.GetPipelineReleasesMemoryUsageMeasurementsFunc == nil {
		return
	}
	return c.GetPipelineReleasesMemoryUsageMeasurementsFunc(ctx, repoSource, repoOwner, repoName, filters)
}

func (c MockClient) GetLabelValues(ctx context.Context, labelKey string) (labels []map[string]interface{}, err error) {
	if c.GetLabelValuesFunc == nil {
		return
	}
	return c.GetLabelValuesFunc(ctx, labelKey)
}

func (c MockClient) GetFrequentLabels(ctx context.Context, pageNumber, pageSize int, filters map[api.FilterType][]string) (labels []map[string]interface{}, err error) {
	if c.GetFrequentLabelsFunc == nil {
		return
	}
	return c.GetFrequentLabelsFunc(ctx, pageNumber, pageSize, filters)
}

func (c MockClient) GetFrequentLabelsCount(ctx context.Context, filters map[api.FilterType][]string) (count int, err error) {
	if c.GetFrequentLabelsCountFunc == nil {
		return
	}
	return c.GetFrequentLabelsCountFunc(ctx, filters)
}

func (c MockClient) GetPipelinesWithMostBuilds(ctx context.Context, pageNumber, pageSize int, filters map[api.FilterType][]string) (pipelines []map[string]interface{}, err error) {
	if c.GetPipelinesWithMostBuildsFunc == nil {
		return
	}
	return c.GetPipelinesWithMostBuildsFunc(ctx, pageNumber, pageSize, filters)
}

func (c MockClient) GetPipelinesWithMostBuildsCount(ctx context.Context, filters map[api.FilterType][]string) (count int, err error) {
	if c.GetPipelinesWithMostBuildsCountFunc == nil {
		return
	}
	return c.GetPipelinesWithMostBuildsCountFunc(ctx, filters)
}

func (c MockClient) GetPipelinesWithMostReleases(ctx context.Context, pageNumber, pageSize int, filters map[api.FilterType][]string) (pipelines []map[string]interface{}, err error) {
	if c.GetPipelinesWithMostReleasesFunc == nil {
		return
	}
	return c.GetPipelinesWithMostReleasesFunc(ctx, pageNumber, pageSize, filters)
}

func (c MockClient) GetPipelinesWithMostReleasesCount(ctx context.Context, filters map[api.FilterType][]string) (count int, err error) {
	if c.GetPipelinesWithMostReleasesCountFunc == nil {
		return
	}
	return c.GetPipelinesWithMostReleasesCountFunc(ctx, filters)
}

func (c MockClient) GetTriggers(ctx context.Context, triggerType, identifier, event string) (pipelines []*contracts.Pipeline, err error) {
	if c.GetTriggersFunc == nil {
		return
	}
	return c.GetTriggersFunc(ctx, triggerType, identifier, event)
}

func (c MockClient) GetGitTriggers(ctx context.Context, gitEvent manifest.EstafetteGitEvent) (pipelines []*contracts.Pipeline, err error) {
	if c.GetGitTriggersFunc == nil {
		return
	}
	return c.GetGitTriggersFunc(ctx, gitEvent)
}

func (c MockClient) GetPipelineTriggers(ctx context.Context, build contracts.Build, event string) (pipelines []*contracts.Pipeline, err error) {
	if c.GetPipelineTriggersFunc == nil {
		return
	}
	return c.GetPipelineTriggersFunc(ctx, build, event)
}

func (c MockClient) GetReleaseTriggers(ctx context.Context, release contracts.Release, event string) (pipelines []*contracts.Pipeline, err error) {
	if c.GetReleaseTriggersFunc == nil {
		return
	}
	return c.GetReleaseTriggersFunc(ctx, release, event)
}

func (c MockClient) GetPubSubTriggers(ctx context.Context, pubsubEvent manifest.EstafettePubSubEvent) (pipelines []*contracts.Pipeline, err error) {
	if c.GetPubSubTriggersFunc == nil {
		return
	}
	return c.GetPubSubTriggersFunc(ctx, pubsubEvent)
}

func (c MockClient) GetCronTriggers(ctx context.Context) (pipelines []*contracts.Pipeline, err error) {
	if c.GetCronTriggersFunc == nil {
		return
	}
	return c.GetCronTriggersFunc(ctx)
}

func (c MockClient) Rename(ctx context.Context, shortFromRepoSource, fromRepoSource, fromRepoOwner, fromRepoName, shortToRepoSource, toRepoSource, toRepoOwner, toRepoName string) (err error) {
	if c.RenameFunc == nil {
		return
	}
	return c.RenameFunc(ctx, shortFromRepoSource, fromRepoSource, fromRepoOwner, fromRepoName, shortToRepoSource, toRepoSource, toRepoOwner, toRepoName)
}

func (c MockClient) RenameBuildVersion(ctx context.Context, shortFromRepoSource, fromRepoOwner, fromRepoName, shortToRepoSource, toRepoOwner, toRepoName string) (err error) {
	if c.RenameBuildVersionFunc == nil {
		return
	}
	return c.RenameBuildVersionFunc(ctx, shortFromRepoSource, fromRepoOwner, fromRepoName, shortToRepoSource, toRepoOwner, toRepoName)
}

func (c MockClient) RenameBuilds(ctx context.Context, fromRepoSource, fromRepoOwner, fromRepoName, toRepoSource, toRepoOwner, toRepoName string) (err error) {
	if c.RenameBuildsFunc == nil {
		return
	}
	return c.RenameBuildsFunc(ctx, fromRepoSource, fromRepoOwner, fromRepoName, toRepoSource, toRepoOwner, toRepoName)
}

func (c MockClient) RenameBuildLogs(ctx context.Context, fromRepoSource, fromRepoOwner, fromRepoName, toRepoSource, toRepoOwner, toRepoName string) (err error) {
	if c.RenameBuildLogsFunc == nil {
		return
	}
	return c.RenameBuildLogsFunc(ctx, fromRepoSource, fromRepoOwner, fromRepoName, toRepoSource, toRepoOwner, toRepoName)
}

func (c MockClient) RenameReleases(ctx context.Context, fromRepoSource, fromRepoOwner, fromRepoName, toRepoSource, toRepoOwner, toRepoName string) (err error) {
	if c.RenameReleasesFunc == nil {
		return
	}
	return c.RenameReleasesFunc(ctx, fromRepoSource, fromRepoOwner, fromRepoName, toRepoSource, toRepoOwner, toRepoName)
}

func (c MockClient) RenameReleaseLogs(ctx context.Context, fromRepoSource, fromRepoOwner, fromRepoName, toRepoSource, toRepoOwner, toRepoName string) (err error) {
	if c.RenameReleaseLogsFunc == nil {
		return
	}
	return c.RenameReleaseLogsFunc(ctx, fromRepoSource, fromRepoOwner, fromRepoName, toRepoSource, toRepoOwner, toRepoName)
}

func (c MockClient) RenameComputedPipelines(ctx context.Context, fromRepoSource, fromRepoOwner, fromRepoName, toRepoSource, toRepoOwner, toRepoName string) (err error) {
	if c.RenameComputedPipelinesFunc == nil {
		return
	}
	return c.RenameComputedPipelinesFunc(ctx, fromRepoSource, fromRepoOwner, fromRepoName, toRepoSource, toRepoOwner, toRepoName)
}

func (c MockClient) RenameComputedReleases(ctx context.Context, fromRepoSource, fromRepoOwner, fromRepoName, toRepoSource, toRepoOwner, toRepoName string) (err error) {
	if c.RenameComputedReleasesFunc == nil {
		return
	}
	return c.RenameComputedReleasesFunc(ctx, fromRepoSource, fromRepoOwner, fromRepoName, toRepoSource, toRepoOwner, toRepoName)
}

func (c MockClient) InsertUser(ctx context.Context, user contracts.User) (u *contracts.User, err error) {
	if c.InsertUserFunc == nil {
		return
	}
	return c.InsertUserFunc(ctx, user)
}

func (c MockClient) UpdateUser(ctx context.Context, user contracts.User) (err error) {
	if c.UpdateUserFunc == nil {
		return
	}
	return c.UpdateUserFunc(ctx, user)
}

func (c MockClient) DeleteUser(ctx context.Context, user contracts.User) (err error) {
	if c.DeleteUserFunc == nil {
		return
	}
	return c.DeleteUserFunc(ctx, user)
}

func (c MockClient) GetUserByIdentity(ctx context.Context, identity contracts.UserIdentity) (user *contracts.User, err error) {
	if c.GetUserByIdentityFunc == nil {
		return
	}
	return c.GetUserByIdentityFunc(ctx, identity)
}

func (c MockClient) GetUserByID(ctx context.Context, id string, filters map[api.FilterType][]string) (user *contracts.User, err error) {
	if c.GetUserByIDFunc == nil {
		return
	}
	return c.GetUserByIDFunc(ctx, id, filters)
}

func (c MockClient) GetUsers(ctx context.Context, pageNumber, pageSize int, filters map[api.FilterType][]string, sortings []api.OrderField) (users []*contracts.User, err error) {
	if c.GetUsersFunc == nil {
		return
	}
	return c.GetUsersFunc(ctx, pageNumber, pageSize, filters, sortings)
}

func (c MockClient) GetUsersCount(ctx context.Context, filters map[api.FilterType][]string) (count int, err error) {
	if c.GetUsersCountFunc == nil {
		return
	}
	return c.GetUsersCountFunc(ctx, filters)
}

func (c MockClient) InsertGroup(ctx context.Context, group contracts.Group) (g *contracts.Group, err error) {
	if c.InsertGroupFunc == nil {
		return
	}
	return c.InsertGroupFunc(ctx, group)
}

func (c MockClient) UpdateGroup(ctx context.Context, group contracts.Group) (err error) {
	if c.UpdateGroupFunc == nil {
		return
	}
	return c.UpdateGroupFunc(ctx, group)
}

func (c MockClient) DeleteGroup(ctx context.Context, group contracts.Group) (err error) {
	if c.DeleteGroupFunc == nil {
		return
	}
	return c.DeleteGroupFunc(ctx, group)
}

func (c MockClient) GetGroupByIdentity(ctx context.Context, identity contracts.GroupIdentity) (group *contracts.Group, err error) {
	if c.GetGroupByIdentityFunc == nil {
		return
	}
	return c.GetGroupByIdentityFunc(ctx, identity)
}

func (c MockClient) GetGroupByID(ctx context.Context, id string, filters map[api.FilterType][]string) (group *contracts.Group, err error) {
	if c.GetGroupByIDFunc == nil {
		return
	}
	return c.GetGroupByIDFunc(ctx, id, filters)
}

func (c MockClient) GetGroups(ctx context.Context, pageNumber, pageSize int, filters map[api.FilterType][]string, sortings []api.OrderField) (groups []*contracts.Group, err error) {
	if c.GetGroupsFunc == nil {
		return
	}
	return c.GetGroupsFunc(ctx, pageNumber, pageSize, filters, sortings)
}

func (c MockClient) GetGroupsCount(ctx context.Context, filters map[api.FilterType][]string) (count int, err error) {
	if c.GetGroupsCountFunc == nil {
		return
	}
	return c.GetGroupsCountFunc(ctx, filters)
}

func (c MockClient) InsertOrganization(ctx context.Context, organization contracts.Organization) (o *contracts.Organization, err error) {
	if c.InsertOrganizationFunc == nil {
		return
	}
	return c.InsertOrganizationFunc(ctx, organization)
}

func (c MockClient) UpdateOrganization(ctx context.Context, organization contracts.Organization) (err error) {
	if c.UpdateOrganizationFunc == nil {
		return
	}
	return c.UpdateOrganizationFunc(ctx, organization)
}

func (c MockClient) DeleteOrganization(ctx context.Context, organization contracts.Organization) (err error) {
	if c.DeleteOrganizationFunc == nil {
		return
	}
	return c.DeleteOrganizationFunc(ctx, organization)
}

func (c MockClient) GetOrganizationByIdentity(ctx context.Context, identity contracts.OrganizationIdentity) (organization *contracts.Organization, err error) {
	if c.GetOrganizationByIdentityFunc == nil {
		return
	}
	return c.GetOrganizationByIdentityFunc(ctx, identity)
}

func (c MockClient) GetOrganizationByID(ctx context.Context, id string) (organization *contracts.Organization, err error) {
	if c.GetOrganizationByIDFunc == nil {
		return
	}
	return c.GetOrganizationByIDFunc(ctx, id)
}

func (c MockClient) GetOrganizationByName(ctx context.Context, name string) (organization *contracts.Organization, err error) {
	if c.GetOrganizationByNameFunc == nil {
		return
	}
	return c.GetOrganizationByNameFunc(ctx, name)
}

func (c MockClient) GetOrganizations(ctx context.Context, pageNumber, pageSize int, filters map[api.FilterType][]string, sortings []api.OrderField) (organizations []*contracts.Organization, err error) {
	if c.GetOrganizationsFunc == nil {
		return
	}
	return c.GetOrganizationsFunc(ctx, pageNumber, pageSize, filters, sortings)
}

func (c MockClient) GetOrganizationsCount(ctx context.Context, filters map[api.FilterType][]string) (count int, err error) {
	if c.GetOrganizationsCountFunc == nil {
		return
	}
	return c.GetOrganizationsCountFunc(ctx, filters)
}

func (c MockClient) InsertClient(ctx context.Context, client contracts.Client) (cl *contracts.Client, err error) {
	if c.InsertClientFunc == nil {
		return
	}
	return c.InsertClientFunc(ctx, client)
}

func (c MockClient) UpdateClient(ctx context.Context, client contracts.Client) (err error) {
	if c.UpdateClientFunc == nil {
		return
	}
	return c.UpdateClientFunc(ctx, client)
}

func (c MockClient) DeleteClient(ctx context.Context, client contracts.Client) (err error) {
	if c.DeleteClientFunc == nil {
		return
	}
	return c.DeleteClientFunc(ctx, client)
}

func (c MockClient) GetClientByClientID(ctx context.Context, clientID string) (client *contracts.Client, err error) {
	if c.GetClientByClientIDFunc == nil {
		return
	}
	return c.GetClientByClientIDFunc(ctx, clientID)
}

func (c MockClient) GetClientByID(ctx context.Context, id string) (client *contracts.Client, err error) {
	if c.GetClientByIDFunc == nil {
		return
	}
	return c.GetClientByIDFunc(ctx, id)
}

func (c MockClient) GetClients(ctx context.Context, pageNumber, pageSize int, filters map[api.FilterType][]string, sortings []api.OrderField) (clients []*contracts.Client, err error) {
	if c.GetClientsFunc == nil {
		return
	}
	return c.GetClientsFunc(ctx, pageNumber, pageSize, filters, sortings)
}

func (c MockClient) GetClientsCount(ctx context.Context, filters map[api.FilterType][]string) (count int, err error) {
	if c.GetClientsCountFunc == nil {
		return
	}
	return c.GetClientsCountFunc(ctx, filters)
}

func (c MockClient) InsertCatalogEntity(ctx context.Context, catalogEntity contracts.CatalogEntity) (insertedCatalogEntity *contracts.CatalogEntity, err error) {
	if c.InsertCatalogEntityFunc == nil {
		return
	}
	return c.InsertCatalogEntityFunc(ctx, catalogEntity)
}

func (c MockClient) UpdateCatalogEntity(ctx context.Context, catalogEntity contracts.CatalogEntity) (err error) {
	if c.UpdateCatalogEntityFunc == nil {
		return
	}
	return c.UpdateCatalogEntityFunc(ctx, catalogEntity)
}

func (c MockClient) DeleteCatalogEntity(ctx context.Context, id string) (err error) {
	if c.DeleteCatalogEntityFunc == nil {
		return
	}
	return c.DeleteCatalogEntityFunc(ctx, id)
}

func (c MockClient) GetCatalogEntityByID(ctx context.Context, id string) (catalogEntity *contracts.CatalogEntity, err error) {
	if c.GetCatalogEntityByIDFunc == nil {
		return
	}
	return c.GetCatalogEntityByIDFunc(ctx, id)
}

func (c MockClient) GetCatalogEntities(ctx context.Context, pageNumber, pageSize int, filters map[api.FilterType][]string, sortings []api.OrderField) (catalogEntities []*contracts.CatalogEntity, err error) {
	if c.GetCatalogEntitiesFunc == nil {
		return
	}
	return c.GetCatalogEntitiesFunc(ctx, pageNumber, pageSize, filters, sortings)
}

func (c MockClient) GetCatalogEntitiesCount(ctx context.Context, filters map[api.FilterType][]string) (count int, err error) {
	if c.GetCatalogEntitiesCountFunc == nil {
		return
	}
	return c.GetCatalogEntitiesCountFunc(ctx, filters)
}

func (c MockClient) GetCatalogEntityParentKeys(ctx context.Context, pageNumber, pageSize int, filters map[api.FilterType][]string, sortings []api.OrderField) (keys []map[string]interface{}, err error) {
	if c.GetCatalogEntityParentKeysFunc == nil {
		return
	}
	return c.GetCatalogEntityParentKeysFunc(ctx, pageNumber, pageSize, filters, sortings)
}

func (c MockClient) GetCatalogEntityParentKeysCount(ctx context.Context, filters map[api.FilterType][]string) (count int, err error) {
	if c.GetCatalogEntityParentKeysCountFunc == nil {
		return
	}
	return c.GetCatalogEntityParentKeysCountFunc(ctx, filters)
}

func (c MockClient) GetCatalogEntityParentValues(ctx context.Context, pageNumber, pageSize int, filters map[api.FilterType][]string, sortings []api.OrderField) (values []map[string]interface{}, err error) {
	if c.GetCatalogEntityParentValuesFunc == nil {
		return
	}
	return c.GetCatalogEntityParentValuesFunc(ctx, pageNumber, pageSize, filters, sortings)
}

func (c MockClient) GetCatalogEntityParentValuesCount(ctx context.Context, filters map[api.FilterType][]string) (count int, err error) {
	if c.GetCatalogEntityParentValuesCountFunc == nil {
		return
	}
	return c.GetCatalogEntityParentValuesCountFunc(ctx, filters)
}

func (c MockClient) GetCatalogEntityKeys(ctx context.Context, pageNumber, pageSize int, filters map[api.FilterType][]string, sortings []api.OrderField) (keys []map[string]interface{}, err error) {
	if c.GetCatalogEntityKeysFunc == nil {
		return
	}
	return c.GetCatalogEntityKeysFunc(ctx, pageNumber, pageSize, filters, sortings)
}

func (c MockClient) GetCatalogEntityKeysCount(ctx context.Context, filters map[api.FilterType][]string) (count int, err error) {
	if c.GetCatalogEntityKeysCountFunc == nil {
		return
	}
	return c.GetCatalogEntityKeysCountFunc(ctx, filters)
}

func (c MockClient) GetCatalogEntityValues(ctx context.Context, pageNumber, pageSize int, filters map[api.FilterType][]string, sortings []api.OrderField) (values []map[string]interface{}, err error) {
	if c.GetCatalogEntityValuesFunc == nil {
		return
	}
	return c.GetCatalogEntityValuesFunc(ctx, pageNumber, pageSize, filters, sortings)
}

func (c MockClient) GetCatalogEntityValuesCount(ctx context.Context, filters map[api.FilterType][]string) (count int, err error) {
	if c.GetCatalogEntityValuesCountFunc == nil {
		return
	}
	return c.GetCatalogEntityValuesCountFunc(ctx, filters)
}

func (c MockClient) GetCatalogEntityLabels(ctx context.Context, pageNumber, pageSize int, filters map[api.FilterType][]string) (labels []map[string]interface{}, err error) {
	if c.GetCatalogEntityLabelsFunc == nil {
		return
	}
	return c.GetCatalogEntityLabelsFunc(ctx, pageNumber, pageSize, filters)
}

func (c MockClient) GetCatalogEntityLabelsCount(ctx context.Context, filters map[api.FilterType][]string) (count int, err error) {
	if c.GetCatalogEntityLabelsCountFunc == nil {
		return
	}
	return c.GetCatalogEntityLabelsCountFunc(ctx, filters)
}
