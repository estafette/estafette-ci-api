package estafette

import (
	"context"
	"testing"

	"github.com/estafette/estafette-ci-api/api"
	"github.com/estafette/estafette-ci-api/clients/bitbucketapi"
	"github.com/estafette/estafette-ci-api/clients/builderapi"
	"github.com/estafette/estafette-ci-api/clients/cloudsourceapi"
	"github.com/estafette/estafette-ci-api/clients/cloudstorage"
	"github.com/estafette/estafette-ci-api/clients/cockroachdb"
	"github.com/estafette/estafette-ci-api/clients/githubapi"
	"github.com/estafette/estafette-ci-api/clients/prometheus"
	contracts "github.com/estafette/estafette-ci-contracts"
	manifest "github.com/estafette/estafette-ci-manifest"
	gomock "github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	batchv1 "k8s.io/api/batch/v1"
)

func TestCreateBuild(t *testing.T) {

	t.Run("CallsGetAutoIncrementOnCockroachdbClient", func(t *testing.T) {

		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		ctx := context.Background()

		config := &api.APIConfig{
			Jobs:      &api.JobsConfig{},
			APIServer: &api.APIServerConfig{},
		}

		cockroachdbClient := cockroachdb.NewMockClient(ctrl)
		prometheusClient := prometheus.NewMockClient(ctrl)
		cloudStorageClient := cloudstorage.NewMockClient(ctrl)
		builderapiClient := builderapi.NewMockClient(ctrl)
		githubapiClient := githubapi.NewMockClient(ctrl)
		bitbucketapiClient := bitbucketapi.NewMockClient(ctrl)
		cloudsourceapiClient := cloudsourceapi.NewMockClient(ctrl)

		callCount := 0
		cockroachdbClient.
			EXPECT().
			GetAutoIncrement(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
			DoAndReturn(func(ctx context.Context, shortRepoSource, repoOwner, repoName string) (autoincrement int, err error) {
				callCount++
				return
			})

		service := NewService(config, cockroachdbClient, prometheusClient, cloudStorageClient, builderapiClient, githubapiClient.JobVarsFunc(ctx), bitbucketapiClient.JobVarsFunc(ctx), cloudsourceapiClient.JobVarsFunc(ctx))

		build := contracts.Build{
			RepoSource: "github.com",
			RepoOwner:  "estafette",
			RepoName:   "estafette-ci-api",
			RepoBranch: "master",
			Manifest:   "builder:\n  track: dev\nstages:\n  stage-1:\n    image: extensions/doesnothing:dev",
		}

		// act
		_, _ = service.CreateBuild(context.Background(), build, true)

		// assert.Nil(t, err)
		assert.Equal(t, 1, callCount)
	})

	t.Run("CallsGetPipelineOnCockroachdbClientIfManifestIsInvalid", func(t *testing.T) {

		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		ctx := context.Background()

		config := &api.APIConfig{
			Jobs:      &api.JobsConfig{},
			APIServer: &api.APIServerConfig{},
		}
		cockroachdbClient := cockroachdb.NewMockClient(ctrl)
		prometheusClient := prometheus.NewMockClient(ctrl)
		cloudStorageClient := cloudstorage.NewMockClient(ctrl)
		builderapiClient := builderapi.NewMockClient(ctrl)
		githubapiClient := githubapi.NewMockClient(ctrl)
		bitbucketapiClient := bitbucketapi.NewMockClient(ctrl)
		cloudsourceapiClient := cloudsourceapi.NewMockClient(ctrl)

		callCount := 0
		cockroachdbClient.
			EXPECT().
			GetPipeline(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
			DoAndReturn(func(ctx context.Context, repoSource, repoOwner, repoName string, filters map[api.FilterType][]string, optimized bool) (pipeline *contracts.Pipeline, err error) {
				callCount++
				return
			})

		service := NewService(config, cockroachdbClient, prometheusClient, cloudStorageClient, builderapiClient, githubapiClient.JobVarsFunc(ctx), bitbucketapiClient.JobVarsFunc(ctx), cloudsourceapiClient.JobVarsFunc(ctx))

		build := contracts.Build{
			RepoSource: "github.com",
			RepoOwner:  "estafette",
			RepoName:   "estafette-ci-api",
			RepoBranch: "master",
			Manifest:   "builder:\n  track: dev\nstages:\n", // no stages, thus invalid
		}

		// act
		_, _ = service.CreateBuild(context.Background(), build, true)

		assert.Equal(t, 1, callCount)
	})

	t.Run("CallsGetPipelineBuildMaxResourceUtilizationOnCockroachdbClient", func(t *testing.T) {

		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		ctx := context.Background()

		config := &api.APIConfig{
			Jobs:      &api.JobsConfig{},
			APIServer: &api.APIServerConfig{},
		}
		cockroachdbClient := cockroachdb.NewMockClient(ctrl)
		prometheusClient := prometheus.NewMockClient(ctrl)
		cloudStorageClient := cloudstorage.NewMockClient(ctrl)
		builderapiClient := builderapi.NewMockClient(ctrl)
		githubapiClient := githubapi.NewMockClient(ctrl)
		bitbucketapiClient := bitbucketapi.NewMockClient(ctrl)
		cloudsourceapiClient := cloudsourceapi.NewMockClient(ctrl)

		callCount := 0
		cockroachdbClient.
			EXPECT().
			GetPipelineBuildMaxResourceUtilization(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
			DoAndReturn(func(ctx context.Context, repoSource, repoOwner, repoName string, lastNRecords int) (jobresources cockroachdb.JobResources, count int, err error) {
				callCount++
				return
			})

		service := NewService(config, cockroachdbClient, prometheusClient, cloudStorageClient, builderapiClient, githubapiClient.JobVarsFunc(ctx), bitbucketapiClient.JobVarsFunc(ctx), cloudsourceapiClient.JobVarsFunc(ctx))

		build := contracts.Build{
			RepoSource: "github.com",
			RepoOwner:  "estafette",
			RepoName:   "estafette-ci-api",
			RepoBranch: "master",
			Manifest:   "builder:\n  track: dev\nstages:\n  stage-1:\n    image: extensions/doesnothing:dev",
		}

		// act
		_, _ = service.CreateBuild(context.Background(), build, true)

		// assert.Nil(t, err)
		assert.Equal(t, 1, callCount)
	})

	t.Run("CallsInsertBuildOnCockroachdbClient", func(t *testing.T) {

		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		ctx := context.Background()

		config := &api.APIConfig{
			Jobs:      &api.JobsConfig{},
			APIServer: &api.APIServerConfig{},
		}
		cockroachdbClient := cockroachdb.NewMockClient(ctrl)
		prometheusClient := prometheus.NewMockClient(ctrl)
		cloudStorageClient := cloudstorage.NewMockClient(ctrl)
		builderapiClient := builderapi.NewMockClient(ctrl)
		githubapiClient := githubapi.NewMockClient(ctrl)
		bitbucketapiClient := bitbucketapi.NewMockClient(ctrl)
		cloudsourceapiClient := cloudsourceapi.NewMockClient(ctrl)

		callCount := 0
		cockroachdbClient.
			EXPECT().
			InsertBuild(gomock.Any(), gomock.Any(), gomock.Any()).
			DoAndReturn(func(ctx context.Context, build contracts.Build, jobResources cockroachdb.JobResources) (b *contracts.Build, err error) {
				callCount++
				return
			})

		service := NewService(config, cockroachdbClient, prometheusClient, cloudStorageClient, builderapiClient, githubapiClient.JobVarsFunc(ctx), bitbucketapiClient.JobVarsFunc(ctx), cloudsourceapiClient.JobVarsFunc(ctx))

		build := contracts.Build{
			RepoSource: "github.com",
			RepoOwner:  "estafette",
			RepoName:   "estafette-ci-api",
			RepoBranch: "master",
			Manifest:   "builder:\n  track: dev\nstages:\n  stage-1:\n    image: extensions/doesnothing:dev",
		}

		// act
		_, _ = service.CreateBuild(context.Background(), build, true)

		// assert.Nil(t, err)
		assert.Equal(t, 1, callCount)
	})

	t.Run("CallsCreateCiBuilderJobOnBuilderapiClient", func(t *testing.T) {

		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		ctx := context.Background()

		config := &api.APIConfig{
			Jobs:      &api.JobsConfig{},
			APIServer: &api.APIServerConfig{},
		}
		cockroachdbClient := cockroachdb.NewMockClient(ctrl)
		prometheusClient := prometheus.NewMockClient(ctrl)
		cloudStorageClient := cloudstorage.NewMockClient(ctrl)
		builderapiClient := builderapi.NewMockClient(ctrl)
		githubapiClient := githubapi.NewMockClient(ctrl)
		bitbucketapiClient := bitbucketapi.NewMockClient(ctrl)
		cloudsourceapiClient := cloudsourceapi.NewMockClient(ctrl)

		cockroachdbClient.
			EXPECT().
			InsertBuild(gomock.Any(), gomock.Any(), gomock.Any()).
			DoAndReturn(func(ctx context.Context, build contracts.Build, jobResources cockroachdb.JobResources) (b *contracts.Build, err error) {
				b = &build
				b.ID = "5"
				return
			})
		callCount := 0
		builderapiClient.
			EXPECT().
			CreateCiBuilderJob(gomock.Any(), gomock.Any()).
			DoAndReturn(func(ctx context.Context, params builderapi.CiBuilderParams) (job *batchv1.Job, err error) {
				callCount++
				return
			})

		service := NewService(config, cockroachdbClient, prometheusClient, cloudStorageClient, builderapiClient, githubapiClient.JobVarsFunc(ctx), bitbucketapiClient.JobVarsFunc(ctx), cloudsourceapiClient.JobVarsFunc(ctx))

		build := contracts.Build{
			RepoSource: "github.com",
			RepoOwner:  "estafette",
			RepoName:   "estafette-ci-api",
			RepoBranch: "master",
			Manifest:   "builder:\n  track: dev\nstages:\n  stage-1:\n    image: extensions/doesnothing:dev",
		}

		// act
		_, err := service.CreateBuild(context.Background(), build, true)

		assert.Nil(t, err)
		assert.Equal(t, 1, callCount)
	})

	t.Run("CallsInsertBuildLogOnCockroachdbClientInsteadOfCreateCiBuilderJobOnBuilderapiClientIfManifestIsInvalid", func(t *testing.T) {

		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		ctx := context.Background()

		config := &api.APIConfig{
			Jobs:      &api.JobsConfig{},
			APIServer: &api.APIServerConfig{},
		}
		cockroachdbClient := cockroachdb.NewMockClient(ctrl)
		prometheusClient := prometheus.NewMockClient(ctrl)
		cloudStorageClient := cloudstorage.NewMockClient(ctrl)
		builderapiClient := builderapi.NewMockClient(ctrl)
		githubapiClient := githubapi.NewMockClient(ctrl)
		bitbucketapiClient := bitbucketapi.NewMockClient(ctrl)
		cloudsourceapiClient := cloudsourceapi.NewMockClient(ctrl)

		cockroachdbClient.
			EXPECT().
			InsertBuild(gomock.Any(), gomock.Any(), gomock.Any()).
			DoAndReturn(func(ctx context.Context, build contracts.Build, jobResources cockroachdb.JobResources) (b *contracts.Build, err error) {
				b = &build
				b.ID = "5"
				return
			})
		createcibuilderjobCallCount := 0
		builderapiClient.
			EXPECT().
			CreateCiBuilderJob(gomock.Any(), gomock.Any()).
			DoAndReturn(func(ctx context.Context, params builderapi.CiBuilderParams) (job *batchv1.Job, err error) {
				createcibuilderjobCallCount++
				return
			})

		insertbuildlogCallCount := 0
		cockroachdbClient.
			EXPECT().
			InsertBuildLog(gomock.Any(), gomock.Any(), gomock.Any()).
			DoAndReturn(func(ctx context.Context, buildLog contracts.BuildLog, writeLogToDatabase bool) (log contracts.BuildLog, err error) {
				insertbuildlogCallCount++
				return
			})

		service := NewService(config, cockroachdbClient, prometheusClient, cloudStorageClient, builderapiClient, githubapiClient.JobVarsFunc(ctx), bitbucketapiClient.JobVarsFunc(ctx), cloudsourceapiClient.JobVarsFunc(ctx))

		build := contracts.Build{
			RepoSource: "github.com",
			RepoOwner:  "estafette",
			RepoName:   "estafette-ci-api",
			RepoBranch: "master",
			Manifest:   "builder:\n  track: dev\nstages:\n", // no stages, thus invalid
		}

		// act
		_, err := service.CreateBuild(context.Background(), build, true)

		assert.Nil(t, err)
		assert.Equal(t, 0, createcibuilderjobCallCount)
		assert.Equal(t, 1, insertbuildlogCallCount)
	})

	t.Run("CallsInsertBuildLogOnCloudstorageClientInsteadOfCreateCiBuilderJobOnBuilderapiClientIfManifestIsInvalidAndLogWritersConfigContainsCloudstorage", func(t *testing.T) {

		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		ctx := context.Background()

		config := &api.APIConfig{
			Jobs: &api.JobsConfig{},
			APIServer: &api.APIServerConfig{
				LogWriters: []string{"cloudstorage"},
			},
		}
		cockroachdbClient := cockroachdb.NewMockClient(ctrl)
		prometheusClient := prometheus.NewMockClient(ctrl)
		cloudStorageClient := cloudstorage.NewMockClient(ctrl)
		builderapiClient := builderapi.NewMockClient(ctrl)
		githubapiClient := githubapi.NewMockClient(ctrl)
		bitbucketapiClient := bitbucketapi.NewMockClient(ctrl)
		cloudsourceapiClient := cloudsourceapi.NewMockClient(ctrl)

		cockroachdbClient.
			EXPECT().
			InsertBuild(gomock.Any(), gomock.Any(), gomock.Any()).
			DoAndReturn(func(ctx context.Context, build contracts.Build, jobResources cockroachdb.JobResources) (b *contracts.Build, err error) {
				b = &build
				b.ID = "5"
				return
			})
		createcibuilderjobCallCount := 0
		builderapiClient.
			EXPECT().
			CreateCiBuilderJob(gomock.Any(), gomock.Any()).
			DoAndReturn(func(ctx context.Context, params builderapi.CiBuilderParams) (job *batchv1.Job, err error) {
				createcibuilderjobCallCount++
				return
			})

		insertbuildlogCallCount := 0
		cloudStorageClient.
			EXPECT().
			InsertBuildLog(gomock.Any(), gomock.Any()).
			DoAndReturn(func(ctx context.Context, buildLog contracts.BuildLog) (err error) {
				insertbuildlogCallCount++
				return
			})

		service := NewService(config, cockroachdbClient, prometheusClient, cloudStorageClient, builderapiClient, githubapiClient.JobVarsFunc(ctx), bitbucketapiClient.JobVarsFunc(ctx), cloudsourceapiClient.JobVarsFunc(ctx))

		build := contracts.Build{
			RepoSource: "github.com",
			RepoOwner:  "estafette",
			RepoName:   "estafette-ci-api",
			RepoBranch: "master",
			Manifest:   "builder:\n  track: dev\nstages:\n", // no stages, thus invalid
		}

		// act
		_, err := service.CreateBuild(context.Background(), build, true)

		assert.Nil(t, err)
		assert.Equal(t, 0, createcibuilderjobCallCount)
		assert.Equal(t, 1, insertbuildlogCallCount)
	})

	t.Run("CallsInsertBuildOnCockroachdbClientWithBuildVersionGeneratedFromAutoincrement", func(t *testing.T) {

		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		ctx := context.Background()

		config := &api.APIConfig{
			Jobs:      &api.JobsConfig{},
			APIServer: &api.APIServerConfig{},
		}
		cockroachdbClient := cockroachdb.NewMockClient(ctrl)
		prometheusClient := prometheus.NewMockClient(ctrl)
		cloudStorageClient := cloudstorage.NewMockClient(ctrl)
		builderapiClient := builderapi.NewMockClient(ctrl)
		githubapiClient := githubapi.NewMockClient(ctrl)
		bitbucketapiClient := bitbucketapi.NewMockClient(ctrl)
		cloudsourceapiClient := cloudsourceapi.NewMockClient(ctrl)

		cockroachdbClient.
			EXPECT().
			GetAutoIncrement(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
			DoAndReturn(func(ctx context.Context, shortRepoSource, repoOwner, repoName string) (autoincrement int, err error) {
				return 15, nil
			})

		callCount := 0
		cockroachdbClient.
			EXPECT().
			InsertBuild(gomock.Any(), gomock.Any(), gomock.Any()).
			DoAndReturn(func(ctx context.Context, build contracts.Build, jobResources cockroachdb.JobResources) (b *contracts.Build, err error) {
				if build.BuildVersion == "0.0.15" {
					callCount++
				}
				return
			})

		service := NewService(config, cockroachdbClient, prometheusClient, cloudStorageClient, builderapiClient, githubapiClient.JobVarsFunc(ctx), bitbucketapiClient.JobVarsFunc(ctx), cloudsourceapiClient.JobVarsFunc(ctx))

		build := contracts.Build{
			RepoSource: "github.com",
			RepoOwner:  "estafette",
			RepoName:   "estafette-ci-api",
			RepoBranch: "master",
			Manifest:   "builder:\n  track: dev\nstages:\n  stage-1:\n    image: extensions/doesnothing:dev",
		}

		// act
		_, _ = service.CreateBuild(context.Background(), build, true)

		// assert.Nil(t, err)
		assert.Equal(t, 1, callCount)
	})

}

func TestFinishBuild(t *testing.T) {

	t.Run("CallsUpdateBuildStatusOnCockroachdbClient", func(t *testing.T) {

		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		ctx := context.Background()

		config := &api.APIConfig{
			Jobs:      &api.JobsConfig{},
			APIServer: &api.APIServerConfig{},
		}
		cockroachdbClient := cockroachdb.NewMockClient(ctrl)
		prometheusClient := prometheus.NewMockClient(ctrl)
		cloudStorageClient := cloudstorage.NewMockClient(ctrl)
		builderapiClient := builderapi.NewMockClient(ctrl)
		githubapiClient := githubapi.NewMockClient(ctrl)
		bitbucketapiClient := bitbucketapi.NewMockClient(ctrl)
		cloudsourceapiClient := cloudsourceapi.NewMockClient(ctrl)

		callCount := 0
		cockroachdbClient.
			EXPECT().
			UpdateBuildStatus(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
			DoAndReturn(func(ctx context.Context, repoSource, repoOwner, repoName string, buildID int, buildStatus contracts.Status) (err error) {
				callCount++
				return
			})

		service := NewService(config, cockroachdbClient, prometheusClient, cloudStorageClient, builderapiClient, githubapiClient.JobVarsFunc(ctx), bitbucketapiClient.JobVarsFunc(ctx), cloudsourceapiClient.JobVarsFunc(ctx))

		repoSource := "github.com"
		repoOwner := "estafette"
		repoName := "estafette-ci-api"
		buildID := 1557
		buildStatus := contracts.StatusSucceeded

		// act
		err := service.FinishBuild(context.Background(), repoSource, repoOwner, repoName, buildID, buildStatus)

		assert.Nil(t, err)
		assert.Equal(t, 1, callCount)
	})
}

func TestCreateRelease(t *testing.T) {

	t.Run("CallsInsertBuildOnCockroachdbClient", func(t *testing.T) {

		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		ctx := context.Background()

		config := &api.APIConfig{
			Jobs:      &api.JobsConfig{},
			APIServer: &api.APIServerConfig{},
		}
		cockroachdbClient := cockroachdb.NewMockClient(ctrl)
		prometheusClient := prometheus.NewMockClient(ctrl)
		cloudStorageClient := cloudstorage.NewMockClient(ctrl)
		builderapiClient := builderapi.NewMockClient(ctrl)
		githubapiClient := githubapi.NewMockClient(ctrl)
		bitbucketapiClient := bitbucketapi.NewMockClient(ctrl)
		cloudsourceapiClient := cloudsourceapi.NewMockClient(ctrl)

		callCount := 0
		cockroachdbClient.
			EXPECT().
			InsertRelease(gomock.Any(), gomock.Any(), gomock.Any()).
			DoAndReturn(func(ctx context.Context, release contracts.Release, jobResources cockroachdb.JobResources) (r *contracts.Release, err error) {
				callCount++
				return
			})

		service := NewService(config, cockroachdbClient, prometheusClient, cloudStorageClient, builderapiClient, githubapiClient.JobVarsFunc(ctx), bitbucketapiClient.JobVarsFunc(ctx), cloudsourceapiClient.JobVarsFunc(ctx))

		release := contracts.Release{
			RepoSource:     "github.com",
			RepoOwner:      "estafette",
			RepoName:       "estafette-ci-api",
			ReleaseVersion: "1.0.256",
		}
		mft := manifest.EstafetteManifest{}
		branch := "master"
		revision := "f0677f01cc6d54a5b042224a9eb374e98f979985"

		// act
		_, _ = service.CreateRelease(context.Background(), release, mft, branch, revision, true)

		// assert.Nil(t, err)
		assert.Equal(t, 1, callCount)
	})

	t.Run("CallsCreateCiBuilderJobOnBuilderapiClient", func(t *testing.T) {

		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		ctx := context.Background()

		config := &api.APIConfig{
			Jobs:      &api.JobsConfig{},
			APIServer: &api.APIServerConfig{},
		}
		cockroachdbClient := cockroachdb.NewMockClient(ctrl)
		prometheusClient := prometheus.NewMockClient(ctrl)
		cloudStorageClient := cloudstorage.NewMockClient(ctrl)
		builderapiClient := builderapi.NewMockClient(ctrl)
		githubapiClient := githubapi.NewMockClient(ctrl)
		bitbucketapiClient := bitbucketapi.NewMockClient(ctrl)
		cloudsourceapiClient := cloudsourceapi.NewMockClient(ctrl)

		cockroachdbClient.
			EXPECT().
			InsertRelease(gomock.Any(), gomock.Any(), gomock.Any()).
			DoAndReturn(func(ctx context.Context, release contracts.Release, jobResources cockroachdb.JobResources) (r *contracts.Release, err error) {
				r = &release
				r.ID = "5"
				return
			})
		callCount := 0
		builderapiClient.
			EXPECT().
			CreateCiBuilderJob(gomock.Any(), gomock.Any()).
			DoAndReturn(func(ctx context.Context, params builderapi.CiBuilderParams) (job *batchv1.Job, err error) {
				callCount++
				return
			})

		service := NewService(config, cockroachdbClient, prometheusClient, cloudStorageClient, builderapiClient, githubapiClient.JobVarsFunc(ctx), bitbucketapiClient.JobVarsFunc(ctx), cloudsourceapiClient.JobVarsFunc(ctx))

		release := contracts.Release{
			RepoSource:     "github.com",
			RepoOwner:      "estafette",
			RepoName:       "estafette-ci-api",
			ReleaseVersion: "1.0.256",
		}
		mft := manifest.EstafetteManifest{}
		branch := "master"
		revision := "f0677f01cc6d54a5b042224a9eb374e98f979985"

		// act
		_, err := service.CreateRelease(context.Background(), release, mft, branch, revision, true)

		assert.Nil(t, err)
		assert.Equal(t, 1, callCount)
	})
}

func TestFinishRelease(t *testing.T) {

	t.Run("CallsUpdateReleaseStatusOnCockroachdbClient", func(t *testing.T) {

		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		ctx := context.Background()

		config := &api.APIConfig{
			Jobs:      &api.JobsConfig{},
			APIServer: &api.APIServerConfig{},
		}
		cockroachdbClient := cockroachdb.NewMockClient(ctrl)
		prometheusClient := prometheus.NewMockClient(ctrl)
		cloudStorageClient := cloudstorage.NewMockClient(ctrl)
		builderapiClient := builderapi.NewMockClient(ctrl)
		githubapiClient := githubapi.NewMockClient(ctrl)
		bitbucketapiClient := bitbucketapi.NewMockClient(ctrl)
		cloudsourceapiClient := cloudsourceapi.NewMockClient(ctrl)

		callCount := 0
		cockroachdbClient.
			EXPECT().
			UpdateReleaseStatus(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
			DoAndReturn(func(ctx context.Context, repoSource, repoOwner, repoName string, buildID int, buildStatus contracts.Status) (err error) {
				callCount++
				return
			})

		service := NewService(config, cockroachdbClient, prometheusClient, cloudStorageClient, builderapiClient, githubapiClient.JobVarsFunc(ctx), bitbucketapiClient.JobVarsFunc(ctx), cloudsourceapiClient.JobVarsFunc(ctx))

		repoSource := "github.com"
		repoOwner := "estafette"
		repoName := "estafette-ci-api"
		buildID := 1557
		buildStatus := contracts.StatusSucceeded

		// act
		err := service.FinishRelease(context.Background(), repoSource, repoOwner, repoName, buildID, buildStatus)

		assert.Nil(t, err)
		assert.Equal(t, 1, callCount)
	})
}

func TestRename(t *testing.T) {

	t.Run("CallsRenameOnCockroachdbClient", func(t *testing.T) {

		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		ctx := context.Background()

		config := &api.APIConfig{
			Jobs:      &api.JobsConfig{},
			APIServer: &api.APIServerConfig{},
		}
		cockroachdbClient := cockroachdb.NewMockClient(ctrl)
		prometheusClient := prometheus.NewMockClient(ctrl)
		cloudStorageClient := cloudstorage.NewMockClient(ctrl)
		builderapiClient := builderapi.NewMockClient(ctrl)

		githubapiClient := githubapi.NewMockClient(ctrl)
		bitbucketapiClient := bitbucketapi.NewMockClient(ctrl)
		cloudsourceapiClient := cloudsourceapi.NewMockClient(ctrl)

		renameOnCockroachdbClientCallCount := 0
		cockroachdbClient.
			EXPECT().
			Rename(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
			DoAndReturn(func(ctx context.Context, shortFromRepoSource, fromRepoSource, fromRepoOwner, fromRepoName, shortToRepoSource, toRepoSource, toRepoOwner, toRepoName string) (err error) {
				renameOnCockroachdbClientCallCount++
				return
			})

		service := NewService(config, cockroachdbClient, prometheusClient, cloudStorageClient, builderapiClient, githubapiClient.JobVarsFunc(ctx), bitbucketapiClient.JobVarsFunc(ctx), cloudsourceapiClient.JobVarsFunc(ctx))

		// act
		err := service.Rename(context.Background(), "github.com", "estafette", "estafette-ci-contracts", "github.com", "estafette", "estafette-ci-protos")

		assert.Nil(t, err)
		assert.Equal(t, 1, renameOnCockroachdbClientCallCount)
	})

	t.Run("CallsRenameOnCloudstorageClientIfLogWritersConfigContainsCloudstorage", func(t *testing.T) {

		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		ctx := context.Background()

		config := &api.APIConfig{
			Jobs: &api.JobsConfig{},
			APIServer: &api.APIServerConfig{
				LogWriters: []string{"cloudstorage"},
			},
		}
		cockroachdbClient := cockroachdb.NewMockClient(ctrl)
		prometheusClient := prometheus.NewMockClient(ctrl)
		cloudStorageClient := cloudstorage.NewMockClient(ctrl)
		builderapiClient := builderapi.NewMockClient(ctrl)

		githubapiClient := githubapi.NewMockClient(ctrl)
		bitbucketapiClient := bitbucketapi.NewMockClient(ctrl)
		cloudsourceapiClient := cloudsourceapi.NewMockClient(ctrl)

		renameOnCloudstorageClientCallCount := 0
		cloudStorageClient.
			EXPECT().
			Rename(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
			DoAndReturn(func(ctx context.Context, fromRepoSource, fromRepoOwner, fromRepoName, toRepoSource, toRepoOwner, toRepoName string) (err error) {
				renameOnCloudstorageClientCallCount++
				return
			})

		service := NewService(config, cockroachdbClient, prometheusClient, cloudStorageClient, builderapiClient, githubapiClient.JobVarsFunc(ctx), bitbucketapiClient.JobVarsFunc(ctx), cloudsourceapiClient.JobVarsFunc(ctx))

		// act
		err := service.Rename(context.Background(), "github.com", "estafette", "estafette-ci-contracts", "github.com", "estafette", "estafette-ci-protos")

		assert.Nil(t, err)
		assert.Equal(t, 1, renameOnCloudstorageClientCallCount)
	})
}

func TestUpdateBuildStatus(t *testing.T) {
	t.Run("CallsUpdateBuildStatusOnCockroachdbClientIfBuildIDIsNonEmpty", func(t *testing.T) {

		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		ctx := context.Background()

		config := &api.APIConfig{
			Jobs:      &api.JobsConfig{},
			APIServer: &api.APIServerConfig{},
		}
		cockroachdbClient := cockroachdb.NewMockClient(ctrl)
		prometheusClient := prometheus.NewMockClient(ctrl)
		cloudStorageClient := cloudstorage.NewMockClient(ctrl)
		builderapiClient := builderapi.NewMockClient(ctrl)

		githubapiClient := githubapi.NewMockClient(ctrl)
		bitbucketapiClient := bitbucketapi.NewMockClient(ctrl)
		cloudsourceapiClient := cloudsourceapi.NewMockClient(ctrl)

		callCount := 0
		cockroachdbClient.
			EXPECT().
			UpdateBuildStatus(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
			DoAndReturn(func(ctx context.Context, repoSource, repoOwner, repoName string, buildID int, buildStatus contracts.Status) (err error) {
				callCount++
				return
			})

		service := NewService(config, cockroachdbClient, prometheusClient, cloudStorageClient, builderapiClient, githubapiClient.JobVarsFunc(ctx), bitbucketapiClient.JobVarsFunc(ctx), cloudsourceapiClient.JobVarsFunc(ctx))

		event := builderapi.CiBuilderEvent{
			BuildID:     "123456",
			BuildStatus: "succeeeded",
		}

		// act
		err := service.UpdateBuildStatus(context.Background(), event)

		assert.Nil(t, err)
		assert.Equal(t, 1, callCount)
	})

	t.Run("CallsUpdateReleaseStatusOnCockroachdbClientIfReleaseIDIsNonEmpty", func(t *testing.T) {

		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		ctx := context.Background()

		config := &api.APIConfig{
			Jobs:      &api.JobsConfig{},
			APIServer: &api.APIServerConfig{},
		}
		cockroachdbClient := cockroachdb.NewMockClient(ctrl)
		prometheusClient := prometheus.NewMockClient(ctrl)
		cloudStorageClient := cloudstorage.NewMockClient(ctrl)
		builderapiClient := builderapi.NewMockClient(ctrl)

		githubapiClient := githubapi.NewMockClient(ctrl)
		bitbucketapiClient := bitbucketapi.NewMockClient(ctrl)
		cloudsourceapiClient := cloudsourceapi.NewMockClient(ctrl)

		callCount := 0
		cockroachdbClient.
			EXPECT().
			UpdateReleaseStatus(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
			DoAndReturn(func(ctx context.Context, repoSource, repoOwner, repoName string, buildID int, buildStatus contracts.Status) (err error) {
				callCount++
				return
			})

		service := NewService(config, cockroachdbClient, prometheusClient, cloudStorageClient, builderapiClient, githubapiClient.JobVarsFunc(ctx), bitbucketapiClient.JobVarsFunc(ctx), cloudsourceapiClient.JobVarsFunc(ctx))

		event := builderapi.CiBuilderEvent{
			ReleaseID:   "123456",
			BuildStatus: "succeeeded",
		}

		// act
		err := service.UpdateBuildStatus(context.Background(), event)

		assert.Nil(t, err)
		assert.Equal(t, 1, callCount)
	})
}

func TestUpdateJobResources(t *testing.T) {
	t.Run("CallsUpdateBuildResourceUtilizationOnCockroachdbClientIfBuildIDIsNonEmpty", func(t *testing.T) {

		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		ctx := context.Background()

		config := &api.APIConfig{
			Jobs:      &api.JobsConfig{},
			APIServer: &api.APIServerConfig{},
		}
		cockroachdbClient := cockroachdb.NewMockClient(ctrl)
		prometheusClient := prometheus.NewMockClient(ctrl)
		cloudStorageClient := cloudstorage.NewMockClient(ctrl)
		builderapiClient := builderapi.NewMockClient(ctrl)

		githubapiClient := githubapi.NewMockClient(ctrl)
		bitbucketapiClient := bitbucketapi.NewMockClient(ctrl)
		cloudsourceapiClient := cloudsourceapi.NewMockClient(ctrl)

		callCount := 0
		cockroachdbClient.
			EXPECT().
			UpdateBuildResourceUtilization(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
			DoAndReturn(func(ctx context.Context, repoSource, repoOwner, repoName string, buildID int, jobResources cockroachdb.JobResources) (err error) {
				callCount++
				return
			})

		service := NewService(config, cockroachdbClient, prometheusClient, cloudStorageClient, builderapiClient, githubapiClient.JobVarsFunc(ctx), bitbucketapiClient.JobVarsFunc(ctx), cloudsourceapiClient.JobVarsFunc(ctx))

		event := builderapi.CiBuilderEvent{
			PodName:     "build-estafette-estafette-ci-api-123456-mhrzk",
			BuildID:     "123456",
			BuildStatus: "succeeeded",
		}

		// act
		err := service.UpdateJobResources(context.Background(), event)

		assert.Nil(t, err)
		assert.Equal(t, 1, callCount)
	})

	t.Run("CallsUpdateReleaseResourceUtilizationOnCockroachdbClientIfReleaseIDIsNonEmpty", func(t *testing.T) {

		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		ctx := context.Background()

		config := &api.APIConfig{
			Jobs:      &api.JobsConfig{},
			APIServer: &api.APIServerConfig{},
		}
		cockroachdbClient := cockroachdb.NewMockClient(ctrl)
		prometheusClient := prometheus.NewMockClient(ctrl)
		cloudStorageClient := cloudstorage.NewMockClient(ctrl)
		builderapiClient := builderapi.NewMockClient(ctrl)

		githubapiClient := githubapi.NewMockClient(ctrl)
		bitbucketapiClient := bitbucketapi.NewMockClient(ctrl)
		cloudsourceapiClient := cloudsourceapi.NewMockClient(ctrl)

		callCount := 0
		cockroachdbClient.
			EXPECT().
			UpdateReleaseResourceUtilization(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
			DoAndReturn(func(ctx context.Context, repoSource, repoOwner, repoName string, buildID int, jobResources cockroachdb.JobResources) (err error) {
				callCount++
				return
			})

		service := NewService(config, cockroachdbClient, prometheusClient, cloudStorageClient, builderapiClient, githubapiClient.JobVarsFunc(ctx), bitbucketapiClient.JobVarsFunc(ctx), cloudsourceapiClient.JobVarsFunc(ctx))

		event := builderapi.CiBuilderEvent{
			PodName:     "release-estafette-estafette-ci-api-123456-mhrzk",
			ReleaseID:   "123456",
			BuildStatus: "succeeeded",
		}

		// act
		err := service.UpdateJobResources(context.Background(), event)

		assert.Nil(t, err)
		assert.Equal(t, 1, callCount)
	})
}

func TestGetEventsForJobEnvvars(t *testing.T) {
	t.Run("ReturnsTriggereEventsForEveryNamedTriggerWithAtLeastOneGreenBuild", func(t *testing.T) {

		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		ctx := context.Background()

		config := &api.APIConfig{
			Jobs:      &api.JobsConfig{},
			APIServer: &api.APIServerConfig{},
		}
		cockroachdbClient := cockroachdb.NewMockClient(ctrl)
		prometheusClient := prometheus.NewMockClient(ctrl)
		cloudStorageClient := cloudstorage.NewMockClient(ctrl)
		builderapiClient := builderapi.NewMockClient(ctrl)

		githubapiClient := githubapi.NewMockClient(ctrl)
		bitbucketapiClient := bitbucketapi.NewMockClient(ctrl)
		cloudsourceapiClient := cloudsourceapi.NewMockClient(ctrl)

		callCount := 0
		cockroachdbClient.
			EXPECT().
			UpdateBuildResourceUtilization(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
			DoAndReturn(func(ctx context.Context, repoSource, repoOwner, repoName string, buildID int, jobResources cockroachdb.JobResources) (err error) {
				callCount++
				return
			})

		service := NewService(config, cockroachdbClient, prometheusClient, cloudStorageClient, builderapiClient, githubapiClient.JobVarsFunc(ctx), bitbucketapiClient.JobVarsFunc(ctx), cloudsourceapiClient.JobVarsFunc(ctx))

		event := builderapi.CiBuilderEvent{
			PodName:     "build-estafette-estafette-ci-api-123456-mhrzk",
			BuildID:     "123456",
			BuildStatus: "succeeeded",
		}

		// act
		err := service.UpdateJobResources(context.Background(), event)

		assert.Nil(t, err)
		assert.Equal(t, 1, callCount)
	})
}
