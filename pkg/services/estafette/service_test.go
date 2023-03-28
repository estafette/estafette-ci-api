package estafette

import (
	"context"
	"testing"

	"github.com/estafette/estafette-ci-api/pkg/api"
	"github.com/estafette/estafette-ci-api/pkg/clients/bitbucketapi"
	"github.com/estafette/estafette-ci-api/pkg/clients/builderapi"
	"github.com/estafette/estafette-ci-api/pkg/clients/cloudsourceapi"
	"github.com/estafette/estafette-ci-api/pkg/clients/cloudstorage"
	"github.com/estafette/estafette-ci-api/pkg/clients/database"
	"github.com/estafette/estafette-ci-api/pkg/clients/githubapi"
	"github.com/estafette/estafette-ci-api/pkg/clients/prometheus"
	contracts "github.com/estafette/estafette-ci-contracts"
	crypt "github.com/estafette/estafette-ci-crypt"
	manifest "github.com/estafette/estafette-ci-manifest"
	gomock "github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
)

func TestCreateBuild(t *testing.T) {

	t.Run("CallsGetAutoIncrementOndatabaseClient", func(t *testing.T) {

		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		ctx := context.Background()

		config := &api.APIConfig{
			Jobs:      &api.JobsConfig{},
			APIServer: &api.APIServerConfig{},
		}

		databaseClient := database.NewMockClient(ctrl)
		secretHelper := crypt.NewSecretHelper("abc", false)
		prometheusClient := prometheus.NewMockClient(ctrl)
		cloudStorageClient := cloudstorage.NewMockClient(ctrl)
		builderapiClient := builderapi.NewMockClient(ctrl)
		githubapiClient := githubapi.NewMockClient(ctrl)
		bitbucketapiClient := bitbucketapi.NewMockClient(ctrl)
		cloudsourceapiClient := cloudsourceapi.NewMockClient(ctrl)

		databaseClient.
			EXPECT().
			GetAutoIncrement(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
			Times(1)

		githubapiClient.EXPECT().JobVarsFunc(gomock.Any()).AnyTimes()
		bitbucketapiClient.EXPECT().JobVarsFunc(gomock.Any()).AnyTimes()
		cloudsourceapiClient.EXPECT().JobVarsFunc(gomock.Any()).AnyTimes()
		databaseClient.EXPECT().GetPipeline(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes()
		githubapiClientJobVarsFunc := func(ctx context.Context, repoSource, repoOwner, repoName string) (token string, err error) {
			return
		}
		bitbucketapiClientJobVarsFunc := func(ctx context.Context, repoSource, repoOwner, repoName string) (token string, err error) {
			return
		}
		cloudsourceapiClientJobVarsFunc := func(ctx context.Context, repoSource, repoOwner, repoName string) (token string, err error) {
			return
		}
		databaseClient.EXPECT().GetPipelineBuildMaxResourceUtilization(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes()
		databaseClient.EXPECT().GetAutoIncrement(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes()
		databaseClient.EXPECT().InsertBuild(gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes()

		service := NewService(config, databaseClient, secretHelper, prometheusClient, cloudStorageClient, builderapiClient, githubapiClientJobVarsFunc, bitbucketapiClientJobVarsFunc, cloudsourceapiClientJobVarsFunc)

		build := contracts.Build{
			RepoSource: "github.com",
			RepoOwner:  "estafette",
			RepoName:   "estafette-ci-api",
			RepoBranch: "master",
			Manifest:   "builder:\n  track: dev\nstages:\n  stage-1:\n    image: extensions/doesnothing:dev",
		}

		// act
		_, _ = service.CreateBuild(ctx, build)
	})

	t.Run("CallsGetPipelineOndatabaseClientIfManifestIsInvalid", func(t *testing.T) {

		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		ctx := context.Background()

		config := &api.APIConfig{
			Jobs:      &api.JobsConfig{},
			APIServer: &api.APIServerConfig{},
		}
		databaseClient := database.NewMockClient(ctrl)
		secretHelper := crypt.NewSecretHelper("abc", false)
		prometheusClient := prometheus.NewMockClient(ctrl)
		cloudStorageClient := cloudstorage.NewMockClient(ctrl)
		builderapiClient := builderapi.NewMockClient(ctrl)
		githubapiClient := githubapi.NewMockClient(ctrl)

		databaseClient.
			EXPECT().
			GetPipeline(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
			Times(1)

		githubapiClient.EXPECT().JobVarsFunc(gomock.Any()).AnyTimes()
		databaseClient.EXPECT().GetPipeline(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes()
		databaseClient.EXPECT().GetPipelineBuildMaxResourceUtilization(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes()
		githubapiClientJobVarsFunc := func(ctx context.Context, repoSource, repoOwner, repoName string) (token string, err error) {
			return
		}
		bitbucketapiClientJobVarsFunc := func(ctx context.Context, repoSource, repoOwner, repoName string) (token string, err error) {
			return
		}
		cloudsourceapiClientJobVarsFunc := func(ctx context.Context, repoSource, repoOwner, repoName string) (token string, err error) {
			return
		}
		databaseClient.EXPECT().GetAutoIncrement(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes()
		databaseClient.EXPECT().InsertBuild(gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes()

		service := NewService(config, databaseClient, secretHelper, prometheusClient, cloudStorageClient, builderapiClient, githubapiClientJobVarsFunc, bitbucketapiClientJobVarsFunc, cloudsourceapiClientJobVarsFunc)

		build := contracts.Build{
			RepoSource: "github.com",
			RepoOwner:  "estafette",
			RepoName:   "estafette-ci-api",
			RepoBranch: "master",
			Manifest:   "builder:\n  track: dev\nstages:\n", // no stages, thus invalid
		}

		// act
		_, _ = service.CreateBuild(ctx, build)
	})

	t.Run("CallsGetPipelineBuildMaxResourceUtilizationOndatabaseClient", func(t *testing.T) {

		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		ctx := context.Background()

		config := &api.APIConfig{
			Jobs:      &api.JobsConfig{},
			APIServer: &api.APIServerConfig{},
		}
		databaseClient := database.NewMockClient(ctrl)
		secretHelper := crypt.NewSecretHelper("abc", false)
		prometheusClient := prometheus.NewMockClient(ctrl)
		cloudStorageClient := cloudstorage.NewMockClient(ctrl)
		builderapiClient := builderapi.NewMockClient(ctrl)

		databaseClient.
			EXPECT().
			GetPipelineBuildMaxResourceUtilization(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
			Times(1)
		githubapiClientJobVarsFunc := func(ctx context.Context, repoSource, repoOwner, repoName string) (token string, err error) {
			return
		}
		bitbucketapiClientJobVarsFunc := func(ctx context.Context, repoSource, repoOwner, repoName string) (token string, err error) {
			return
		}
		cloudsourceapiClientJobVarsFunc := func(ctx context.Context, repoSource, repoOwner, repoName string) (token string, err error) {
			return
		}
		databaseClient.EXPECT().GetPipeline(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes()
		databaseClient.EXPECT().GetAutoIncrement(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes()
		databaseClient.EXPECT().InsertBuild(gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes()

		service := NewService(config, databaseClient, secretHelper, prometheusClient, cloudStorageClient, builderapiClient, githubapiClientJobVarsFunc, bitbucketapiClientJobVarsFunc, cloudsourceapiClientJobVarsFunc)

		build := contracts.Build{
			RepoSource: "github.com",
			RepoOwner:  "estafette",
			RepoName:   "estafette-ci-api",
			RepoBranch: "master",
			Manifest:   "builder:\n  track: dev\nstages:\n  stage-1:\n    image: extensions/doesnothing:dev",
		}

		// act
		_, _ = service.CreateBuild(ctx, build)
	})

	t.Run("CallsInsertBuildOndatabaseClient", func(t *testing.T) {

		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		ctx := context.Background()

		config := &api.APIConfig{
			Jobs:      &api.JobsConfig{},
			APIServer: &api.APIServerConfig{},
		}
		databaseClient := database.NewMockClient(ctrl)
		secretHelper := crypt.NewSecretHelper("abc", false)
		prometheusClient := prometheus.NewMockClient(ctrl)
		cloudStorageClient := cloudstorage.NewMockClient(ctrl)
		builderapiClient := builderapi.NewMockClient(ctrl)

		databaseClient.
			EXPECT().
			InsertBuild(gomock.Any(), gomock.Any(), gomock.Any()).
			Times(1)
		githubapiClientJobVarsFunc := func(ctx context.Context, repoSource, repoOwner, repoName string) (token string, err error) {
			return
		}
		bitbucketapiClientJobVarsFunc := func(ctx context.Context, repoSource, repoOwner, repoName string) (token string, err error) {
			return
		}
		cloudsourceapiClientJobVarsFunc := func(ctx context.Context, repoSource, repoOwner, repoName string) (token string, err error) {
			return
		}
		databaseClient.EXPECT().GetPipeline(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes()
		databaseClient.EXPECT().GetAutoIncrement(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes()
		databaseClient.EXPECT().GetPipelineBuildMaxResourceUtilization(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes()

		service := NewService(config, databaseClient, secretHelper, prometheusClient, cloudStorageClient, builderapiClient, githubapiClientJobVarsFunc, bitbucketapiClientJobVarsFunc, cloudsourceapiClientJobVarsFunc)

		build := contracts.Build{
			RepoSource: "github.com",
			RepoOwner:  "estafette",
			RepoName:   "estafette-ci-api",
			RepoBranch: "master",
			Manifest:   "builder:\n  track: dev\nstages:\n  stage-1:\n    image: extensions/doesnothing:dev",
		}

		// act
		_, _ = service.CreateBuild(ctx, build)
	})

	t.Run("CallsCreateCiBuilderJobOnBuilderapiClient", func(t *testing.T) {

		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		ctx := context.Background()

		config := &api.APIConfig{
			Jobs:      &api.JobsConfig{},
			APIServer: &api.APIServerConfig{},
		}
		databaseClient := database.NewMockClient(ctrl)
		secretHelper := crypt.NewSecretHelper("abc", false)
		prometheusClient := prometheus.NewMockClient(ctrl)
		cloudStorageClient := cloudstorage.NewMockClient(ctrl)
		builderapiClient := builderapi.NewMockClient(ctrl)

		databaseClient.
			EXPECT().
			InsertBuild(gomock.Any(), gomock.Any(), gomock.Any()).
			DoAndReturn(func(ctx context.Context, build contracts.Build, jobResources database.JobResources) (b *contracts.Build, err error) {
				b = &build
				b.ID = "5"
				return
			})
		builderapiClient.
			EXPECT().
			CreateCiBuilderJob(gomock.Any(), gomock.Any()).
			Times(1)
		githubapiClientJobVarsFunc := func(ctx context.Context, repoSource, repoOwner, repoName string) (token string, err error) {
			return
		}
		bitbucketapiClientJobVarsFunc := func(ctx context.Context, repoSource, repoOwner, repoName string) (token string, err error) {
			return
		}
		cloudsourceapiClientJobVarsFunc := func(ctx context.Context, repoSource, repoOwner, repoName string) (token string, err error) {
			return
		}
		databaseClient.EXPECT().GetPipeline(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes()
		databaseClient.EXPECT().GetAutoIncrement(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes()
		databaseClient.EXPECT().GetPipelineBuildMaxResourceUtilization(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes()
		databaseClient.EXPECT().GetPipelineTriggers(gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes()

		service := NewService(config, databaseClient, secretHelper, prometheusClient, cloudStorageClient, builderapiClient, githubapiClientJobVarsFunc, bitbucketapiClientJobVarsFunc, cloudsourceapiClientJobVarsFunc)

		build := contracts.Build{
			RepoSource: "github.com",
			RepoOwner:  "estafette",
			RepoName:   "estafette-ci-api",
			RepoBranch: "master",
			Manifest:   "builder:\n  track: dev\nstages:\n  stage-1:\n    image: extensions/doesnothing:dev",
		}

		// act
		_, err := service.CreateBuild(ctx, build)

		assert.Nil(t, err)
	})

	t.Run("CallsInsertBuildLogOndatabaseClientInsteadOfCreateCiBuilderJobOnBuilderapiClientIfManifestIsInvalid", func(t *testing.T) {

		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		ctx := context.Background()

		config := &api.APIConfig{
			Jobs:      &api.JobsConfig{},
			APIServer: &api.APIServerConfig{},
		}
		databaseClient := database.NewMockClient(ctrl)
		secretHelper := crypt.NewSecretHelper("abc", false)
		prometheusClient := prometheus.NewMockClient(ctrl)
		cloudStorageClient := cloudstorage.NewMockClient(ctrl)
		builderapiClient := builderapi.NewMockClient(ctrl)

		databaseClient.
			EXPECT().
			InsertBuild(gomock.Any(), gomock.Any(), gomock.Any()).
			DoAndReturn(func(ctx context.Context, build contracts.Build, jobResources database.JobResources) (b *contracts.Build, err error) {
				b = &build
				b.ID = "5"
				return
			})
		builderapiClient.
			EXPECT().
			CreateCiBuilderJob(gomock.Any(), gomock.Any()).
			Times(0)

		databaseClient.
			EXPECT().
			InsertBuildLog(gomock.Any(), gomock.Any()).
			Times(1)

		githubapiClientJobVarsFunc := func(ctx context.Context, repoSource, repoOwner, repoName string) (token string, err error) {
			return
		}
		bitbucketapiClientJobVarsFunc := func(ctx context.Context, repoSource, repoOwner, repoName string) (token string, err error) {
			return
		}
		cloudsourceapiClientJobVarsFunc := func(ctx context.Context, repoSource, repoOwner, repoName string) (token string, err error) {
			return
		}
		databaseClient.EXPECT().GetPipeline(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes()
		databaseClient.EXPECT().GetAutoIncrement(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes()
		databaseClient.EXPECT().GetPipelineBuildMaxResourceUtilization(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes()

		service := NewService(config, databaseClient, secretHelper, prometheusClient, cloudStorageClient, builderapiClient, githubapiClientJobVarsFunc, bitbucketapiClientJobVarsFunc, cloudsourceapiClientJobVarsFunc)

		build := contracts.Build{
			RepoSource: "github.com",
			RepoOwner:  "estafette",
			RepoName:   "estafette-ci-api",
			RepoBranch: "master",
			Manifest:   "builder:\n  track: dev\nstages:\n", // no stages, thus invalid
		}

		// act
		_, err := service.CreateBuild(ctx, build)

		assert.Nil(t, err)
	})

	t.Run("CallsInsertBuildLogOnCloudstorageClientInsteadOfCreateCiBuilderJobOnBuilderapiClientIfManifestIsInvalidAndLogWritersConfigContainsCloudstorage", func(t *testing.T) {

		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		ctx := context.Background()

		config := &api.APIConfig{
			Jobs: &api.JobsConfig{},
			APIServer: &api.APIServerConfig{
				LogWriters: []api.LogTarget{api.LogTargetCloudStorage},
			},
		}
		databaseClient := database.NewMockClient(ctrl)
		secretHelper := crypt.NewSecretHelper("abc", false)
		prometheusClient := prometheus.NewMockClient(ctrl)
		cloudStorageClient := cloudstorage.NewMockClient(ctrl)
		builderapiClient := builderapi.NewMockClient(ctrl)

		databaseClient.
			EXPECT().
			InsertBuild(gomock.Any(), gomock.Any(), gomock.Any()).
			DoAndReturn(func(ctx context.Context, build contracts.Build, jobResources database.JobResources) (b *contracts.Build, err error) {
				b = &build
				b.ID = "5"
				return
			})
		builderapiClient.
			EXPECT().
			CreateCiBuilderJob(gomock.Any(), gomock.Any()).
			Times(0)

		cloudStorageClient.
			EXPECT().
			InsertBuildLog(gomock.Any(), gomock.Any()).
			Times(1)

		githubapiClientJobVarsFunc := func(ctx context.Context, repoSource, repoOwner, repoName string) (token string, err error) {
			return
		}
		bitbucketapiClientJobVarsFunc := func(ctx context.Context, repoSource, repoOwner, repoName string) (token string, err error) {
			return
		}
		cloudsourceapiClientJobVarsFunc := func(ctx context.Context, repoSource, repoOwner, repoName string) (token string, err error) {
			return
		}

		databaseClient.EXPECT().GetPipeline(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes()
		databaseClient.EXPECT().GetAutoIncrement(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes()
		databaseClient.EXPECT().GetPipelineBuildMaxResourceUtilization(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes()
		databaseClient.EXPECT().InsertBuildLog(gomock.Any(), gomock.Any()).AnyTimes()

		service := NewService(config, databaseClient, secretHelper, prometheusClient, cloudStorageClient, builderapiClient, githubapiClientJobVarsFunc, bitbucketapiClientJobVarsFunc, cloudsourceapiClientJobVarsFunc)

		build := contracts.Build{
			RepoSource: "github.com",
			RepoOwner:  "estafette",
			RepoName:   "estafette-ci-api",
			RepoBranch: "master",
			Manifest:   "builder:\n  track: dev\nstages:\n", // no stages, thus invalid
		}

		// act
		_, err := service.CreateBuild(ctx, build)

		assert.Nil(t, err)
	})

	t.Run("CallsInsertBuildOndatabaseClientWithBuildVersionGeneratedFromAutoincrement", func(t *testing.T) {

		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		ctx := context.Background()

		config := &api.APIConfig{
			Jobs:      &api.JobsConfig{},
			APIServer: &api.APIServerConfig{},
		}
		databaseClient := database.NewMockClient(ctrl)
		secretHelper := crypt.NewSecretHelper("abc", false)
		prometheusClient := prometheus.NewMockClient(ctrl)
		cloudStorageClient := cloudstorage.NewMockClient(ctrl)
		builderapiClient := builderapi.NewMockClient(ctrl)

		databaseClient.
			EXPECT().
			GetAutoIncrement(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
			DoAndReturn(func(ctx context.Context, shortRepoSource, repoOwner, repoName string) (autoincrement int, err error) {
				return 15, nil
			})

		callCount := 0
		databaseClient.
			EXPECT().
			InsertBuild(gomock.Any(), gomock.Any(), gomock.Any()).
			DoAndReturn(func(ctx context.Context, build contracts.Build, jobResources database.JobResources) (b *contracts.Build, err error) {
				if build.BuildVersion == "0.0.15" {
					callCount++
				}
				return
			})

		githubapiClientJobVarsFunc := func(ctx context.Context, repoSource, repoOwner, repoName string) (token string, err error) {
			return
		}
		bitbucketapiClientJobVarsFunc := func(ctx context.Context, repoSource, repoOwner, repoName string) (token string, err error) {
			return
		}
		cloudsourceapiClientJobVarsFunc := func(ctx context.Context, repoSource, repoOwner, repoName string) (token string, err error) {
			return
		}
		databaseClient.EXPECT().GetPipeline(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes()
		databaseClient.EXPECT().GetAutoIncrement(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes()
		databaseClient.EXPECT().GetPipelineBuildMaxResourceUtilization(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes()

		service := NewService(config, databaseClient, secretHelper, prometheusClient, cloudStorageClient, builderapiClient, githubapiClientJobVarsFunc, bitbucketapiClientJobVarsFunc, cloudsourceapiClientJobVarsFunc)

		build := contracts.Build{
			RepoSource: "github.com",
			RepoOwner:  "estafette",
			RepoName:   "estafette-ci-api",
			RepoBranch: "master",
			Manifest:   "builder:\n  track: dev\nstages:\n  stage-1:\n    image: extensions/doesnothing:dev",
		}

		// act
		_, _ = service.CreateBuild(ctx, build)

		// assert.Nil(t, err)
		assert.Equal(t, 1, callCount)
	})

}

func TestFinishBuild(t *testing.T) {

	t.Run("CallsUpdateBuildStatusOndatabaseClient", func(t *testing.T) {

		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		ctx := context.Background()

		config := &api.APIConfig{
			Jobs:      &api.JobsConfig{},
			APIServer: &api.APIServerConfig{},
		}
		databaseClient := database.NewMockClient(ctrl)
		secretHelper := crypt.NewSecretHelper("abc", false)
		prometheusClient := prometheus.NewMockClient(ctrl)
		cloudStorageClient := cloudstorage.NewMockClient(ctrl)
		builderapiClient := builderapi.NewMockClient(ctrl)

		databaseClient.
			EXPECT().
			UpdateBuildStatus(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
			Times(1)
		githubapiClientJobVarsFunc := func(ctx context.Context, repoSource, repoOwner, repoName string) (token string, err error) {
			return
		}
		bitbucketapiClientJobVarsFunc := func(ctx context.Context, repoSource, repoOwner, repoName string) (token string, err error) {
			return
		}
		cloudsourceapiClientJobVarsFunc := func(ctx context.Context, repoSource, repoOwner, repoName string) (token string, err error) {
			return
		}

		databaseClient.EXPECT().GetPipelineBuildByID(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes()

		service := NewService(config, databaseClient, secretHelper, prometheusClient, cloudStorageClient, builderapiClient, githubapiClientJobVarsFunc, bitbucketapiClientJobVarsFunc, cloudsourceapiClientJobVarsFunc)

		repoSource := "github.com"
		repoOwner := "estafette"
		repoName := "estafette-ci-api"
		buildID := "1557"
		buildStatus := contracts.StatusSucceeded

		// act
		err := service.FinishBuild(ctx, repoSource, repoOwner, repoName, buildID, buildStatus)

		assert.Nil(t, err)
	})
}

func TestCreateRelease(t *testing.T) {

	t.Run("CallsInsertReleaseOndatabaseClient", func(t *testing.T) {

		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		ctx := context.Background()

		config := &api.APIConfig{
			Jobs:      &api.JobsConfig{},
			APIServer: &api.APIServerConfig{},
		}
		databaseClient := database.NewMockClient(ctrl)
		secretHelper := crypt.NewSecretHelper("abc", false)
		prometheusClient := prometheus.NewMockClient(ctrl)
		cloudStorageClient := cloudstorage.NewMockClient(ctrl)
		builderapiClient := builderapi.NewMockClient(ctrl)

		databaseClient.
			EXPECT().
			InsertRelease(gomock.Any(), gomock.Any(), gomock.Any()).
			Return(&contracts.Release{ID: "15"}, nil).
			Times(1)

		githubapiClientJobVarsFunc := func(ctx context.Context, repoSource, repoOwner, repoName string) (token string, err error) {
			return
		}
		bitbucketapiClientJobVarsFunc := func(ctx context.Context, repoSource, repoOwner, repoName string) (token string, err error) {
			return
		}
		cloudsourceapiClientJobVarsFunc := func(ctx context.Context, repoSource, repoOwner, repoName string) (token string, err error) {
			return
		}

		databaseClient.EXPECT().GetPipelineReleaseMaxResourceUtilization(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes()
		databaseClient.EXPECT().GetPipeline(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes()
		builderapiClient.EXPECT().CreateCiBuilderJob(gomock.Any(), gomock.Any()).AnyTimes()
		databaseClient.EXPECT().GetReleaseTriggers(gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes()
		databaseClient.EXPECT().GetPipelineBuildByID(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes()

		service := NewService(config, databaseClient, secretHelper, prometheusClient, cloudStorageClient, builderapiClient, githubapiClientJobVarsFunc, bitbucketapiClientJobVarsFunc, cloudsourceapiClientJobVarsFunc)

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
		_, err := service.CreateRelease(ctx, release, mft, branch, revision)

		assert.Nil(t, err)
	})

	t.Run("CallsCreateCiBuilderJobOnBuilderapiClient", func(t *testing.T) {

		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		ctx := context.Background()

		config := &api.APIConfig{
			Jobs:      &api.JobsConfig{},
			APIServer: &api.APIServerConfig{},
		}
		databaseClient := database.NewMockClient(ctrl)
		secretHelper := crypt.NewSecretHelper("abc", false)
		prometheusClient := prometheus.NewMockClient(ctrl)
		cloudStorageClient := cloudstorage.NewMockClient(ctrl)
		builderapiClient := builderapi.NewMockClient(ctrl)

		databaseClient.
			EXPECT().
			InsertRelease(gomock.Any(), gomock.Any(), gomock.Any()).
			DoAndReturn(func(ctx context.Context, release contracts.Release, jobResources database.JobResources) (r *contracts.Release, err error) {
				r = &release
				r.ID = "5"
				return
			})

		builderapiClient.
			EXPECT().
			CreateCiBuilderJob(gomock.Any(), gomock.Any()).
			Times(1)

		githubapiClientJobVarsFunc := func(ctx context.Context, repoSource, repoOwner, repoName string) (token string, err error) {
			return
		}
		bitbucketapiClientJobVarsFunc := func(ctx context.Context, repoSource, repoOwner, repoName string) (token string, err error) {
			return
		}
		cloudsourceapiClientJobVarsFunc := func(ctx context.Context, repoSource, repoOwner, repoName string) (token string, err error) {
			return
		}

		databaseClient.EXPECT().GetPipelineReleaseMaxResourceUtilization(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes()
		databaseClient.EXPECT().GetPipeline(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes()
		databaseClient.EXPECT().GetPipelineBuildByID(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes()
		databaseClient.EXPECT().GetReleaseTriggers(gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes()

		service := NewService(config, databaseClient, secretHelper, prometheusClient, cloudStorageClient, builderapiClient, githubapiClientJobVarsFunc, bitbucketapiClientJobVarsFunc, cloudsourceapiClientJobVarsFunc)

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
		_, err := service.CreateRelease(ctx, release, mft, branch, revision)

		assert.Nil(t, err)
	})
}

func TestFinishRelease(t *testing.T) {

	t.Run("CallsUpdateReleaseStatusOndatabaseClient", func(t *testing.T) {

		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		ctx := context.Background()

		config := &api.APIConfig{
			Jobs:      &api.JobsConfig{},
			APIServer: &api.APIServerConfig{},
		}
		databaseClient := database.NewMockClient(ctrl)
		secretHelper := crypt.NewSecretHelper("abc", false)
		prometheusClient := prometheus.NewMockClient(ctrl)
		cloudStorageClient := cloudstorage.NewMockClient(ctrl)
		builderapiClient := builderapi.NewMockClient(ctrl)

		databaseClient.
			EXPECT().
			UpdateReleaseStatus(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
			Times(1)

		githubapiClientJobVarsFunc := func(ctx context.Context, repoSource, repoOwner, repoName string) (token string, err error) {
			return
		}
		bitbucketapiClientJobVarsFunc := func(ctx context.Context, repoSource, repoOwner, repoName string) (token string, err error) {
			return
		}
		cloudsourceapiClientJobVarsFunc := func(ctx context.Context, repoSource, repoOwner, repoName string) (token string, err error) {
			return
		}

		databaseClient.EXPECT().GetPipelineRelease(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes()

		service := NewService(config, databaseClient, secretHelper, prometheusClient, cloudStorageClient, builderapiClient, githubapiClientJobVarsFunc, bitbucketapiClientJobVarsFunc, cloudsourceapiClientJobVarsFunc)

		repoSource := "github.com"
		repoOwner := "estafette"
		repoName := "estafette-ci-api"
		buildID := "1557"
		buildStatus := contracts.StatusSucceeded

		// act
		err := service.FinishRelease(ctx, repoSource, repoOwner, repoName, buildID, buildStatus)

		assert.Nil(t, err)
	})
}

func TestRename(t *testing.T) {

	t.Run("CallsRenameOndatabaseClient", func(t *testing.T) {

		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		ctx := context.Background()

		config := &api.APIConfig{
			Jobs:      &api.JobsConfig{},
			APIServer: &api.APIServerConfig{},
		}
		databaseClient := database.NewMockClient(ctrl)
		secretHelper := crypt.NewSecretHelper("abc", false)
		prometheusClient := prometheus.NewMockClient(ctrl)
		cloudStorageClient := cloudstorage.NewMockClient(ctrl)
		builderapiClient := builderapi.NewMockClient(ctrl)

		databaseClient.
			EXPECT().
			Rename(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
			Times(1)

		githubapiClientJobVarsFunc := func(ctx context.Context, repoSource, repoOwner, repoName string) (token string, err error) {
			return
		}
		bitbucketapiClientJobVarsFunc := func(ctx context.Context, repoSource, repoOwner, repoName string) (token string, err error) {
			return
		}
		cloudsourceapiClientJobVarsFunc := func(ctx context.Context, repoSource, repoOwner, repoName string) (token string, err error) {
			return
		}

		service := NewService(config, databaseClient, secretHelper, prometheusClient, cloudStorageClient, builderapiClient, githubapiClientJobVarsFunc, bitbucketapiClientJobVarsFunc, cloudsourceapiClientJobVarsFunc)

		// act
		err := service.Rename(ctx, "github.com", "estafette", "estafette-ci-contracts", "github.com", "estafette", "estafette-ci-protos")

		assert.Nil(t, err)
	})

	t.Run("CallsRenameOnCloudstorageClientIfLogWritersConfigContainsCloudstorage", func(t *testing.T) {

		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		ctx := context.Background()

		config := &api.APIConfig{
			Jobs: &api.JobsConfig{},
			APIServer: &api.APIServerConfig{
				LogWriters: []api.LogTarget{api.LogTargetCloudStorage},
			},
		}
		databaseClient := database.NewMockClient(ctrl)
		secretHelper := crypt.NewSecretHelper("abc", false)
		prometheusClient := prometheus.NewMockClient(ctrl)
		cloudStorageClient := cloudstorage.NewMockClient(ctrl)
		builderapiClient := builderapi.NewMockClient(ctrl)

		cloudStorageClient.
			EXPECT().
			Rename(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
			Times(1)

		githubapiClientJobVarsFunc := func(ctx context.Context, repoSource, repoOwner, repoName string) (token string, err error) {
			return
		}
		bitbucketapiClientJobVarsFunc := func(ctx context.Context, repoSource, repoOwner, repoName string) (token string, err error) {
			return
		}
		cloudsourceapiClientJobVarsFunc := func(ctx context.Context, repoSource, repoOwner, repoName string) (token string, err error) {
			return
		}

		databaseClient.EXPECT().Rename(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes()

		service := NewService(config, databaseClient, secretHelper, prometheusClient, cloudStorageClient, builderapiClient, githubapiClientJobVarsFunc, bitbucketapiClientJobVarsFunc, cloudsourceapiClientJobVarsFunc)

		// act
		err := service.Rename(ctx, "github.com", "estafette", "estafette-ci-contracts", "github.com", "estafette", "estafette-ci-protos")

		assert.Nil(t, err)
	})
}

func TestUpdateBuildStatus(t *testing.T) {
	t.Run("CallsUpdateBuildStatusOndatabaseClientIfBuildIDIsNonEmpty", func(t *testing.T) {

		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		ctx := context.Background()

		config := &api.APIConfig{
			Jobs:      &api.JobsConfig{},
			APIServer: &api.APIServerConfig{},
		}
		databaseClient := database.NewMockClient(ctrl)
		secretHelper := crypt.NewSecretHelper("abc", false)
		prometheusClient := prometheus.NewMockClient(ctrl)
		cloudStorageClient := cloudstorage.NewMockClient(ctrl)
		builderapiClient := builderapi.NewMockClient(ctrl)

		databaseClient.
			EXPECT().
			UpdateBuildStatus(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
			Times(1)

		githubapiClientJobVarsFunc := func(ctx context.Context, repoSource, repoOwner, repoName string) (token string, err error) {
			return
		}
		bitbucketapiClientJobVarsFunc := func(ctx context.Context, repoSource, repoOwner, repoName string) (token string, err error) {
			return
		}
		cloudsourceapiClientJobVarsFunc := func(ctx context.Context, repoSource, repoOwner, repoName string) (token string, err error) {
			return
		}

		databaseClient.EXPECT().GetPipelineBuildByID(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes()
		databaseClient.EXPECT().GetPipelineRelease(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes()

		service := NewService(config, databaseClient, secretHelper, prometheusClient, cloudStorageClient, builderapiClient, githubapiClientJobVarsFunc, bitbucketapiClientJobVarsFunc, cloudsourceapiClientJobVarsFunc)

		event := contracts.EstafetteCiBuilderEvent{
			JobType: contracts.JobTypeBuild,
			Build: &contracts.Build{
				ID:          "123456",
				BuildStatus: contracts.StatusSucceeded,
			},
			Git: &contracts.GitConfig{},
		}

		// act
		err := service.UpdateBuildStatus(ctx, event)

		assert.Nil(t, err)
	})

	t.Run("CallsUpdateReleaseStatusOndatabaseClientIfReleaseIDIsNonEmpty", func(t *testing.T) {

		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		ctx := context.Background()

		config := &api.APIConfig{
			Jobs:      &api.JobsConfig{},
			APIServer: &api.APIServerConfig{},
		}
		databaseClient := database.NewMockClient(ctrl)
		secretHelper := crypt.NewSecretHelper("abc", false)
		prometheusClient := prometheus.NewMockClient(ctrl)
		cloudStorageClient := cloudstorage.NewMockClient(ctrl)
		builderapiClient := builderapi.NewMockClient(ctrl)

		databaseClient.
			EXPECT().
			UpdateReleaseStatus(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
			Times(1)

		githubapiClientJobVarsFunc := func(ctx context.Context, repoSource, repoOwner, repoName string) (token string, err error) {
			return
		}
		bitbucketapiClientJobVarsFunc := func(ctx context.Context, repoSource, repoOwner, repoName string) (token string, err error) {
			return
		}
		cloudsourceapiClientJobVarsFunc := func(ctx context.Context, repoSource, repoOwner, repoName string) (token string, err error) {
			return
		}

		databaseClient.EXPECT().GetPipelineBuildByID(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes()
		databaseClient.EXPECT().GetPipelineRelease(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes()

		service := NewService(config, databaseClient, secretHelper, prometheusClient, cloudStorageClient, builderapiClient, githubapiClientJobVarsFunc, bitbucketapiClientJobVarsFunc, cloudsourceapiClientJobVarsFunc)

		event := contracts.EstafetteCiBuilderEvent{
			JobType: contracts.JobTypeRelease,
			Release: &contracts.Release{
				ID:            "123456",
				ReleaseStatus: contracts.StatusSucceeded,
			},
			Git: &contracts.GitConfig{},
		}

		// act
		err := service.UpdateBuildStatus(ctx, event)

		assert.Nil(t, err)
	})
}

func TestUpdateJobResources(t *testing.T) {
	t.Run("CallsUpdateBuildResourceUtilizationOndatabaseClientIfBuildIDIsNonEmpty", func(t *testing.T) {

		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		ctx := context.Background()

		config := &api.APIConfig{
			Jobs:      &api.JobsConfig{},
			APIServer: &api.APIServerConfig{},
		}
		databaseClient := database.NewMockClient(ctrl)
		secretHelper := crypt.NewSecretHelper("abc", false)
		prometheusClient := prometheus.NewMockClient(ctrl)
		cloudStorageClient := cloudstorage.NewMockClient(ctrl)
		builderapiClient := builderapi.NewMockClient(ctrl)

		databaseClient.
			EXPECT().
			UpdateBuildResourceUtilization(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
			Times(1)

		githubapiClientJobVarsFunc := func(ctx context.Context, repoSource, repoOwner, repoName string) (token string, err error) {
			return
		}
		bitbucketapiClientJobVarsFunc := func(ctx context.Context, repoSource, repoOwner, repoName string) (token string, err error) {
			return
		}
		cloudsourceapiClientJobVarsFunc := func(ctx context.Context, repoSource, repoOwner, repoName string) (token string, err error) {
			return
		}

		prometheusClient.EXPECT().AwaitScrapeInterval(gomock.Any()).AnyTimes()
		prometheusClient.EXPECT().GetMaxCPUByPodName(gomock.Any(), gomock.Any()).AnyTimes()
		prometheusClient.EXPECT().GetMaxMemoryByPodName(gomock.Any(), gomock.Any()).AnyTimes()

		service := NewService(config, databaseClient, secretHelper, prometheusClient, cloudStorageClient, builderapiClient, githubapiClientJobVarsFunc, bitbucketapiClientJobVarsFunc, cloudsourceapiClientJobVarsFunc)

		event := contracts.EstafetteCiBuilderEvent{
			PodName: "build-estafette-estafette-ci-api-123456-mhrzk",
			JobType: contracts.JobTypeBuild,
			Build: &contracts.Build{
				ID:          "123456",
				BuildStatus: contracts.StatusSucceeded,
			},
			Git: &contracts.GitConfig{},
		}

		// act
		err := service.UpdateJobResources(ctx, event)

		assert.Nil(t, err)
	})

	t.Run("CallsUpdateReleaseResourceUtilizationOndatabaseClientIfReleaseIDIsNonEmpty", func(t *testing.T) {

		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		ctx := context.Background()

		config := &api.APIConfig{
			Jobs:      &api.JobsConfig{},
			APIServer: &api.APIServerConfig{},
		}
		databaseClient := database.NewMockClient(ctrl)
		secretHelper := crypt.NewSecretHelper("abc", false)
		prometheusClient := prometheus.NewMockClient(ctrl)
		cloudStorageClient := cloudstorage.NewMockClient(ctrl)
		builderapiClient := builderapi.NewMockClient(ctrl)

		databaseClient.
			EXPECT().
			UpdateReleaseResourceUtilization(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
			Times(1)

		githubapiClientJobVarsFunc := func(ctx context.Context, repoSource, repoOwner, repoName string) (token string, err error) {
			return
		}
		bitbucketapiClientJobVarsFunc := func(ctx context.Context, repoSource, repoOwner, repoName string) (token string, err error) {
			return
		}
		cloudsourceapiClientJobVarsFunc := func(ctx context.Context, repoSource, repoOwner, repoName string) (token string, err error) {
			return
		}

		prometheusClient.EXPECT().AwaitScrapeInterval(gomock.Any()).AnyTimes()
		prometheusClient.EXPECT().GetMaxCPUByPodName(gomock.Any(), gomock.Any()).AnyTimes()
		prometheusClient.EXPECT().GetMaxMemoryByPodName(gomock.Any(), gomock.Any()).AnyTimes()

		service := NewService(config, databaseClient, secretHelper, prometheusClient, cloudStorageClient, builderapiClient, githubapiClientJobVarsFunc, bitbucketapiClientJobVarsFunc, cloudsourceapiClientJobVarsFunc)

		event := contracts.EstafetteCiBuilderEvent{
			PodName: "release-estafette-estafette-ci-api-123456-mhrzk",
			JobType: contracts.JobTypeRelease,
			Release: &contracts.Release{
				ID:            "123456",
				ReleaseStatus: contracts.StatusSucceeded,
			},
			Git: &contracts.GitConfig{},
		}

		// act
		err := service.UpdateJobResources(ctx, event)

		assert.Nil(t, err)
	})
}

func TestGetEventsForJobEnvvars(t *testing.T) {
	t.Run("ReturnsTriggerEventsForEveryNamedTriggerWithAtLeastOneGreenBuild", func(t *testing.T) {

		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		ctx := context.Background()

		config := &api.APIConfig{
			Jobs:      &api.JobsConfig{},
			APIServer: &api.APIServerConfig{},
		}
		databaseClient := database.NewMockClient(ctrl)
		secretHelper := crypt.NewSecretHelper("abc", false)
		prometheusClient := prometheus.NewMockClient(ctrl)
		cloudStorageClient := cloudstorage.NewMockClient(ctrl)
		builderapiClient := builderapi.NewMockClient(ctrl)

		githubapiClientJobVarsFunc := func(ctx context.Context, repoSource, repoOwner, repoName string) (token string, err error) {
			return
		}
		bitbucketapiClientJobVarsFunc := func(ctx context.Context, repoSource, repoOwner, repoName string) (token string, err error) {
			return
		}
		cloudsourceapiClientJobVarsFunc := func(ctx context.Context, repoSource, repoOwner, repoName string) (token string, err error) {
			return
		}

		databaseClient.EXPECT().GetPipelineBuilds(gomock.Any(), gomock.Eq("github.com"), gomock.Eq("estafette"), gomock.Eq("repo1"), gomock.Eq(1), gomock.Eq(1), gomock.Any(), gomock.Any(), gomock.Eq(false)).
			DoAndReturn(func(ctx context.Context, repoSource, repoOwner, repoName string, pageNumber, pageSize int, filters map[api.FilterType][]string, sortings []api.OrderField, optimized bool) (builds []*contracts.Build, err error) {
				return []*contracts.Build{
					{
						BuildStatus:  contracts.StatusSucceeded,
						BuildVersion: "1.0.5",
						RepoSource:   "github.com",
						RepoOwner:    "estafette",
						RepoName:     "repo1",
					},
				}, nil
			})

		databaseClient.EXPECT().GetPipelineBuilds(gomock.Any(), gomock.Eq("github.com"), gomock.Eq("estafette"), gomock.Eq("repo2"), gomock.Eq(1), gomock.Eq(1), gomock.Any(), gomock.Any(), gomock.Eq(false)).
			DoAndReturn(func(ctx context.Context, repoSource, repoOwner, repoName string, pageNumber, pageSize int, filters map[api.FilterType][]string, sortings []api.OrderField, optimized bool) (builds []*contracts.Build, err error) {
				return []*contracts.Build{
					{
						BuildStatus:  contracts.StatusSucceeded,
						BuildVersion: "6.4.3",
						RepoSource:   "github.com",
						RepoOwner:    "estafette",
						RepoName:     "repo2",
					},
				}, nil
			})

		service := NewService(config, databaseClient, secretHelper, prometheusClient, cloudStorageClient, builderapiClient, githubapiClientJobVarsFunc, bitbucketapiClientJobVarsFunc, cloudsourceapiClientJobVarsFunc)

		triggers := []manifest.EstafetteTrigger{
			{
				Name: "trigger1",
				Pipeline: &manifest.EstafettePipelineTrigger{
					Name:   "github.com/estafette/repo1",
					Branch: "main",
					Event:  "finished",
					Status: "succeeded",
				},
				BuildAction: &manifest.EstafetteTriggerBuildAction{
					Branch: "main",
				},
			},
			{
				Name: "trigger2",
				Pipeline: &manifest.EstafettePipelineTrigger{
					Name:   "github.com/estafette/repo2",
					Branch: "main",
					Event:  "finished",
					Status: "succeeded",
				},
				BuildAction: &manifest.EstafetteTriggerBuildAction{
					Branch: "main",
				},
			},
		}

		events := []manifest.EstafetteEvent{}

		// act
		triggersAsEvents, err := service.GetEventsForJobEnvvars(ctx, triggers, events)

		assert.Nil(t, err)
		assert.Equal(t, 2, len(triggersAsEvents))
		assert.Equal(t, "trigger1", triggersAsEvents[0].Name)
		assert.Equal(t, false, triggersAsEvents[0].Fired)
		assert.Equal(t, "1.0.5", triggersAsEvents[0].Pipeline.BuildVersion)
		assert.Equal(t, "trigger2", triggersAsEvents[1].Name)
		assert.Equal(t, false, triggersAsEvents[1].Fired)
		assert.Equal(t, "6.4.3", triggersAsEvents[1].Pipeline.BuildVersion)
	})
}

func Test_isReleaseBlocked(t *testing.T) {
	tests := []struct {
		name, release, repo, branch string
		repositories                map[string]api.RepositoryReleaseControl
		restrictedClusters          api.List
		mft                         manifest.EstafetteManifest
		expected                    bool
	}{
		{
			"ReturnsTrueIfBlocked",
			"prd-cluster",
			"repo1",
			"non-master",
			map[string]api.RepositoryReleaseControl{
				"repo1": {
					Blocked: api.List{"non-master"},
				},
			},
			api.List{"prd-cluster"},
			manifest.EstafetteManifest{},
			true,
		},
		{
			"ReturnsTrueIfBlockedInAllRepositories",
			"prd-cluster",
			"repo1",
			"main2",
			map[string]api.RepositoryReleaseControl{
				api.AllRepositories: {
					Blocked: api.List{"main2"},
				},
			},
			api.List{"prd-cluster"},
			manifest.EstafetteManifest{},
			true,
		},
		{
			"ReturnsTrueIfNotAllowed",
			"prd-cluster",
			"repo1",
			"non-master",
			map[string]api.RepositoryReleaseControl{
				"repo1": {
					Allowed: api.List{"main"},
				},
			},
			api.List{"prd-cluster"},
			manifest.EstafetteManifest{},
			true,
		},
		{
			"ReturnsTrueIfNotAllowedClusterCreds",
			"some-random-release",
			"repo1",
			"non-master",
			map[string]api.RepositoryReleaseControl{
				"repo1": {
					Allowed: api.List{"main"},
				},
			},
			api.List{"prd-cluster"},
			manifest.EstafetteManifest{
				Releases: []*manifest.EstafetteRelease{
					{
						Name: "some-random-release",
						Stages: []*manifest.EstafetteStage{
							{
								CustomProperties: map[string]interface{}{
									"credentials": "gke-prd-cluster",
								},
							},
						},
					},
				},
			},
			true,
		},
		{
			"ReturnsTrueIfNotAllowedClusterCredsInParallelStages",
			"some-random-release",
			"repo1",
			"non-master",
			map[string]api.RepositoryReleaseControl{
				"repo1": {
					Allowed: api.List{"main"},
				},
			},
			api.List{"prd-cluster"},
			manifest.EstafetteManifest{
				Releases: []*manifest.EstafetteRelease{
					{
						Name: "some-random-release",
						Stages: []*manifest.EstafetteStage{
							{
								ParallelStages: []*manifest.EstafetteStage{
									{
										CustomProperties: map[string]interface{}{
											"credentials": "gke-prd-cluster",
										},
									},
								},
							},
						},
					},
				},
			},
			true,
		},
		{
			"ReturnsFalseIfNotAllowedButNonProdCluster",
			"dev-cluster",
			"repo1",
			"non-master",
			map[string]api.RepositoryReleaseControl{
				"repo1": {
					Allowed: api.List{"main"},
				},
			},
			api.List{"prd-cluster"},
			manifest.EstafetteManifest{},
			false,
		},
		{
			"ReturnsFalseIfBlockedButNonProdCluster",
			"dev-cluster",
			"repo1",
			"main1",
			map[string]api.RepositoryReleaseControl{
				"repo1": {
					Allowed: api.List{"main"},
				},
			},
			api.List{"prd-cluster"},
			manifest.EstafetteManifest{},
			false,
		},
		{
			"ReturnsFalseIfAllowed",
			"prd-cluster",
			"repo1",
			"main",
			map[string]api.RepositoryReleaseControl{
				"repo1": {
					Allowed: api.List{"main"},
				},
			},
			api.List{"prd-cluster"},
			manifest.EstafetteManifest{},
			false,
		},
		{
			"ReturnsFalseIfNotConfigured",
			"prd-cluster",
			"repo1",
			"main1",
			map[string]api.RepositoryReleaseControl{},
			api.List{},
			manifest.EstafetteManifest{},
			false,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			s := &service{
				config: &api.APIConfig{
					Jobs:      &api.JobsConfig{},
					APIServer: &api.APIServerConfig{},
					BuildControl: &api.BuildControl{
						Release: &api.ReleaseControl{
							Repositories:       test.repositories,
							RestrictedClusters: test.restrictedClusters,
						},
					},
				},
			}
			release := contracts.Release{
				Name:           test.release,
				RepoSource:     "github.com",
				RepoOwner:      "estafette",
				RepoName:       test.repo,
				ReleaseVersion: "1.0.256",
			}
			actual, _, _ := s.isReleaseBlocked(release, test.mft, test.branch)
			assert.Equal(t, test.expected, actual)
		})
	}
	t.Run("nil release config", func(t *testing.T) {
		s := &service{
			config: &api.APIConfig{
				Jobs:         &api.JobsConfig{},
				APIServer:    &api.APIServerConfig{},
				BuildControl: &api.BuildControl{},
			},
		}
		release := contracts.Release{
			Name:           "prd-cluster",
			RepoSource:     "github.com",
			RepoOwner:      "estafette",
			RepoName:       "repo1",
			ReleaseVersion: "1.0.256",
		}
		mft := manifest.EstafetteManifest{
			Releases: []*manifest.EstafetteRelease{
				{
					Name: "prd-cluster",
					Stages: []*manifest.EstafetteStage{
						{
							ParallelStages: []*manifest.EstafetteStage{
								{
									CustomProperties: map[string]interface{}{
										"credentials": "gke-prd-cluster",
									},
								},
							},
						},
					},
				},
			},
		}
		actual, _, _ := s.isReleaseBlocked(release, mft, "main")
		assert.Equal(t, false, actual)
	})
}
