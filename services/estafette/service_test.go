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
	crypt "github.com/estafette/estafette-ci-crypt"
	manifest "github.com/estafette/estafette-ci-manifest"
	gomock "github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
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
		secretHelper := crypt.NewSecretHelper("abc", false)
		prometheusClient := prometheus.NewMockClient(ctrl)
		cloudStorageClient := cloudstorage.NewMockClient(ctrl)
		builderapiClient := builderapi.NewMockClient(ctrl)
		githubapiClient := githubapi.NewMockClient(ctrl)
		bitbucketapiClient := bitbucketapi.NewMockClient(ctrl)
		cloudsourceapiClient := cloudsourceapi.NewMockClient(ctrl)

		cockroachdbClient.
			EXPECT().
			GetAutoIncrement(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
			Times(1)

		githubapiClient.EXPECT().JobVarsFunc(gomock.Any()).AnyTimes()
		bitbucketapiClient.EXPECT().JobVarsFunc(gomock.Any()).AnyTimes()
		cloudsourceapiClient.EXPECT().JobVarsFunc(gomock.Any()).AnyTimes()
		cockroachdbClient.EXPECT().GetPipeline(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes()
		githubapiClientJobVarsFunc := func(ctx context.Context, repoSource, repoOwner, repoName string) (token string, url string, err error) {
			return
		}
		bitbucketapiClientJobVarsFunc := func(ctx context.Context, repoSource, repoOwner, repoName string) (token string, url string, err error) {
			return
		}
		cloudsourceapiClientJobVarsFunc := func(ctx context.Context, repoSource, repoOwner, repoName string) (token string, url string, err error) {
			return
		}
		cockroachdbClient.EXPECT().GetPipelineBuildMaxResourceUtilization(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes()
		cockroachdbClient.EXPECT().GetAutoIncrement(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes()
		cockroachdbClient.EXPECT().InsertBuild(gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes()

		service := NewService(config, cockroachdbClient, secretHelper, prometheusClient, cloudStorageClient, builderapiClient, githubapiClientJobVarsFunc, bitbucketapiClientJobVarsFunc, cloudsourceapiClientJobVarsFunc)

		build := contracts.Build{
			RepoSource: "github.com",
			RepoOwner:  "estafette",
			RepoName:   "estafette-ci-api",
			RepoBranch: "master",
			Manifest:   "builder:\n  track: dev\nstages:\n  stage-1:\n    image: extensions/doesnothing:dev",
		}

		// act
		_, _ = service.CreateBuild(ctx, build, true)
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
		secretHelper := crypt.NewSecretHelper("abc", false)
		prometheusClient := prometheus.NewMockClient(ctrl)
		cloudStorageClient := cloudstorage.NewMockClient(ctrl)
		builderapiClient := builderapi.NewMockClient(ctrl)
		githubapiClient := githubapi.NewMockClient(ctrl)

		cockroachdbClient.
			EXPECT().
			GetPipeline(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
			Times(1)

		githubapiClient.EXPECT().JobVarsFunc(gomock.Any()).AnyTimes()
		cockroachdbClient.EXPECT().GetPipeline(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes()
		cockroachdbClient.EXPECT().GetPipelineBuildMaxResourceUtilization(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes()
		githubapiClientJobVarsFunc := func(ctx context.Context, repoSource, repoOwner, repoName string) (token string, url string, err error) {
			return
		}
		bitbucketapiClientJobVarsFunc := func(ctx context.Context, repoSource, repoOwner, repoName string) (token string, url string, err error) {
			return
		}
		cloudsourceapiClientJobVarsFunc := func(ctx context.Context, repoSource, repoOwner, repoName string) (token string, url string, err error) {
			return
		}
		cockroachdbClient.EXPECT().GetAutoIncrement(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes()
		cockroachdbClient.EXPECT().InsertBuild(gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes()

		service := NewService(config, cockroachdbClient, secretHelper, prometheusClient, cloudStorageClient, builderapiClient, githubapiClientJobVarsFunc, bitbucketapiClientJobVarsFunc, cloudsourceapiClientJobVarsFunc)

		build := contracts.Build{
			RepoSource: "github.com",
			RepoOwner:  "estafette",
			RepoName:   "estafette-ci-api",
			RepoBranch: "master",
			Manifest:   "builder:\n  track: dev\nstages:\n", // no stages, thus invalid
		}

		// act
		_, _ = service.CreateBuild(ctx, build, true)
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
		secretHelper := crypt.NewSecretHelper("abc", false)
		prometheusClient := prometheus.NewMockClient(ctrl)
		cloudStorageClient := cloudstorage.NewMockClient(ctrl)
		builderapiClient := builderapi.NewMockClient(ctrl)

		cockroachdbClient.
			EXPECT().
			GetPipelineBuildMaxResourceUtilization(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
			Times(1)
		githubapiClientJobVarsFunc := func(ctx context.Context, repoSource, repoOwner, repoName string) (token string, url string, err error) {
			return
		}
		bitbucketapiClientJobVarsFunc := func(ctx context.Context, repoSource, repoOwner, repoName string) (token string, url string, err error) {
			return
		}
		cloudsourceapiClientJobVarsFunc := func(ctx context.Context, repoSource, repoOwner, repoName string) (token string, url string, err error) {
			return
		}
		cockroachdbClient.EXPECT().GetPipeline(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes()
		cockroachdbClient.EXPECT().GetAutoIncrement(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes()
		cockroachdbClient.EXPECT().InsertBuild(gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes()

		service := NewService(config, cockroachdbClient, secretHelper, prometheusClient, cloudStorageClient, builderapiClient, githubapiClientJobVarsFunc, bitbucketapiClientJobVarsFunc, cloudsourceapiClientJobVarsFunc)

		build := contracts.Build{
			RepoSource: "github.com",
			RepoOwner:  "estafette",
			RepoName:   "estafette-ci-api",
			RepoBranch: "master",
			Manifest:   "builder:\n  track: dev\nstages:\n  stage-1:\n    image: extensions/doesnothing:dev",
		}

		// act
		_, _ = service.CreateBuild(ctx, build, true)
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
		secretHelper := crypt.NewSecretHelper("abc", false)
		prometheusClient := prometheus.NewMockClient(ctrl)
		cloudStorageClient := cloudstorage.NewMockClient(ctrl)
		builderapiClient := builderapi.NewMockClient(ctrl)

		cockroachdbClient.
			EXPECT().
			InsertBuild(gomock.Any(), gomock.Any(), gomock.Any()).
			Times(1)
		githubapiClientJobVarsFunc := func(ctx context.Context, repoSource, repoOwner, repoName string) (token string, url string, err error) {
			return
		}
		bitbucketapiClientJobVarsFunc := func(ctx context.Context, repoSource, repoOwner, repoName string) (token string, url string, err error) {
			return
		}
		cloudsourceapiClientJobVarsFunc := func(ctx context.Context, repoSource, repoOwner, repoName string) (token string, url string, err error) {
			return
		}
		cockroachdbClient.EXPECT().GetPipeline(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes()
		cockroachdbClient.EXPECT().GetAutoIncrement(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes()
		cockroachdbClient.EXPECT().GetPipelineBuildMaxResourceUtilization(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes()

		service := NewService(config, cockroachdbClient, secretHelper, prometheusClient, cloudStorageClient, builderapiClient, githubapiClientJobVarsFunc, bitbucketapiClientJobVarsFunc, cloudsourceapiClientJobVarsFunc)

		build := contracts.Build{
			RepoSource: "github.com",
			RepoOwner:  "estafette",
			RepoName:   "estafette-ci-api",
			RepoBranch: "master",
			Manifest:   "builder:\n  track: dev\nstages:\n  stage-1:\n    image: extensions/doesnothing:dev",
		}

		// act
		_, _ = service.CreateBuild(ctx, build, true)
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
		secretHelper := crypt.NewSecretHelper("abc", false)
		prometheusClient := prometheus.NewMockClient(ctrl)
		cloudStorageClient := cloudstorage.NewMockClient(ctrl)
		builderapiClient := builderapi.NewMockClient(ctrl)

		cockroachdbClient.
			EXPECT().
			InsertBuild(gomock.Any(), gomock.Any(), gomock.Any()).
			DoAndReturn(func(ctx context.Context, build contracts.Build, jobResources cockroachdb.JobResources) (b *contracts.Build, err error) {
				b = &build
				b.ID = "5"
				return
			})
		builderapiClient.
			EXPECT().
			CreateCiBuilderJob(gomock.Any(), gomock.Any()).
			Times(1)
		githubapiClientJobVarsFunc := func(ctx context.Context, repoSource, repoOwner, repoName string) (token string, url string, err error) {
			return
		}
		bitbucketapiClientJobVarsFunc := func(ctx context.Context, repoSource, repoOwner, repoName string) (token string, url string, err error) {
			return
		}
		cloudsourceapiClientJobVarsFunc := func(ctx context.Context, repoSource, repoOwner, repoName string) (token string, url string, err error) {
			return
		}
		cockroachdbClient.EXPECT().GetPipeline(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes()
		cockroachdbClient.EXPECT().GetAutoIncrement(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes()
		cockroachdbClient.EXPECT().GetPipelineBuildMaxResourceUtilization(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes()
		cockroachdbClient.EXPECT().GetPipelineTriggers(gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes()

		service := NewService(config, cockroachdbClient, secretHelper, prometheusClient, cloudStorageClient, builderapiClient, githubapiClientJobVarsFunc, bitbucketapiClientJobVarsFunc, cloudsourceapiClientJobVarsFunc)

		build := contracts.Build{
			RepoSource: "github.com",
			RepoOwner:  "estafette",
			RepoName:   "estafette-ci-api",
			RepoBranch: "master",
			Manifest:   "builder:\n  track: dev\nstages:\n  stage-1:\n    image: extensions/doesnothing:dev",
		}

		// act
		_, err := service.CreateBuild(ctx, build, true)

		assert.Nil(t, err)
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
		secretHelper := crypt.NewSecretHelper("abc", false)
		prometheusClient := prometheus.NewMockClient(ctrl)
		cloudStorageClient := cloudstorage.NewMockClient(ctrl)
		builderapiClient := builderapi.NewMockClient(ctrl)

		cockroachdbClient.
			EXPECT().
			InsertBuild(gomock.Any(), gomock.Any(), gomock.Any()).
			DoAndReturn(func(ctx context.Context, build contracts.Build, jobResources cockroachdb.JobResources) (b *contracts.Build, err error) {
				b = &build
				b.ID = "5"
				return
			})
		builderapiClient.
			EXPECT().
			CreateCiBuilderJob(gomock.Any(), gomock.Any()).
			Times(0)

		cockroachdbClient.
			EXPECT().
			InsertBuildLog(gomock.Any(), gomock.Any(), gomock.Any()).
			Times(1)

		githubapiClientJobVarsFunc := func(ctx context.Context, repoSource, repoOwner, repoName string) (token string, url string, err error) {
			return
		}
		bitbucketapiClientJobVarsFunc := func(ctx context.Context, repoSource, repoOwner, repoName string) (token string, url string, err error) {
			return
		}
		cloudsourceapiClientJobVarsFunc := func(ctx context.Context, repoSource, repoOwner, repoName string) (token string, url string, err error) {
			return
		}
		cockroachdbClient.EXPECT().GetPipeline(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes()
		cockroachdbClient.EXPECT().GetAutoIncrement(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes()
		cockroachdbClient.EXPECT().GetPipelineBuildMaxResourceUtilization(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes()

		service := NewService(config, cockroachdbClient, secretHelper, prometheusClient, cloudStorageClient, builderapiClient, githubapiClientJobVarsFunc, bitbucketapiClientJobVarsFunc, cloudsourceapiClientJobVarsFunc)

		build := contracts.Build{
			RepoSource: "github.com",
			RepoOwner:  "estafette",
			RepoName:   "estafette-ci-api",
			RepoBranch: "master",
			Manifest:   "builder:\n  track: dev\nstages:\n", // no stages, thus invalid
		}

		// act
		_, err := service.CreateBuild(ctx, build, true)

		assert.Nil(t, err)
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
		secretHelper := crypt.NewSecretHelper("abc", false)
		prometheusClient := prometheus.NewMockClient(ctrl)
		cloudStorageClient := cloudstorage.NewMockClient(ctrl)
		builderapiClient := builderapi.NewMockClient(ctrl)

		cockroachdbClient.
			EXPECT().
			InsertBuild(gomock.Any(), gomock.Any(), gomock.Any()).
			DoAndReturn(func(ctx context.Context, build contracts.Build, jobResources cockroachdb.JobResources) (b *contracts.Build, err error) {
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

		githubapiClientJobVarsFunc := func(ctx context.Context, repoSource, repoOwner, repoName string) (token string, url string, err error) {
			return
		}
		bitbucketapiClientJobVarsFunc := func(ctx context.Context, repoSource, repoOwner, repoName string) (token string, url string, err error) {
			return
		}
		cloudsourceapiClientJobVarsFunc := func(ctx context.Context, repoSource, repoOwner, repoName string) (token string, url string, err error) {
			return
		}

		cockroachdbClient.EXPECT().GetPipeline(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes()
		cockroachdbClient.EXPECT().GetAutoIncrement(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes()
		cockroachdbClient.EXPECT().GetPipelineBuildMaxResourceUtilization(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes()
		cockroachdbClient.EXPECT().InsertBuildLog(gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes()

		service := NewService(config, cockroachdbClient, secretHelper, prometheusClient, cloudStorageClient, builderapiClient, githubapiClientJobVarsFunc, bitbucketapiClientJobVarsFunc, cloudsourceapiClientJobVarsFunc)

		build := contracts.Build{
			RepoSource: "github.com",
			RepoOwner:  "estafette",
			RepoName:   "estafette-ci-api",
			RepoBranch: "master",
			Manifest:   "builder:\n  track: dev\nstages:\n", // no stages, thus invalid
		}

		// act
		_, err := service.CreateBuild(ctx, build, true)

		assert.Nil(t, err)
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
		secretHelper := crypt.NewSecretHelper("abc", false)
		prometheusClient := prometheus.NewMockClient(ctrl)
		cloudStorageClient := cloudstorage.NewMockClient(ctrl)
		builderapiClient := builderapi.NewMockClient(ctrl)

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

		githubapiClientJobVarsFunc := func(ctx context.Context, repoSource, repoOwner, repoName string) (token string, url string, err error) {
			return
		}
		bitbucketapiClientJobVarsFunc := func(ctx context.Context, repoSource, repoOwner, repoName string) (token string, url string, err error) {
			return
		}
		cloudsourceapiClientJobVarsFunc := func(ctx context.Context, repoSource, repoOwner, repoName string) (token string, url string, err error) {
			return
		}
		cockroachdbClient.EXPECT().GetPipeline(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes()
		cockroachdbClient.EXPECT().GetAutoIncrement(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes()
		cockroachdbClient.EXPECT().GetPipelineBuildMaxResourceUtilization(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes()

		service := NewService(config, cockroachdbClient, secretHelper, prometheusClient, cloudStorageClient, builderapiClient, githubapiClientJobVarsFunc, bitbucketapiClientJobVarsFunc, cloudsourceapiClientJobVarsFunc)

		build := contracts.Build{
			RepoSource: "github.com",
			RepoOwner:  "estafette",
			RepoName:   "estafette-ci-api",
			RepoBranch: "master",
			Manifest:   "builder:\n  track: dev\nstages:\n  stage-1:\n    image: extensions/doesnothing:dev",
		}

		// act
		_, _ = service.CreateBuild(ctx, build, true)

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
		secretHelper := crypt.NewSecretHelper("abc", false)
		prometheusClient := prometheus.NewMockClient(ctrl)
		cloudStorageClient := cloudstorage.NewMockClient(ctrl)
		builderapiClient := builderapi.NewMockClient(ctrl)

		cockroachdbClient.
			EXPECT().
			UpdateBuildStatus(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
			Times(1)
		githubapiClientJobVarsFunc := func(ctx context.Context, repoSource, repoOwner, repoName string) (token string, url string, err error) {
			return
		}
		bitbucketapiClientJobVarsFunc := func(ctx context.Context, repoSource, repoOwner, repoName string) (token string, url string, err error) {
			return
		}
		cloudsourceapiClientJobVarsFunc := func(ctx context.Context, repoSource, repoOwner, repoName string) (token string, url string, err error) {
			return
		}

		cockroachdbClient.EXPECT().GetPipelineBuildByID(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes()

		service := NewService(config, cockroachdbClient, secretHelper, prometheusClient, cloudStorageClient, builderapiClient, githubapiClientJobVarsFunc, bitbucketapiClientJobVarsFunc, cloudsourceapiClientJobVarsFunc)

		repoSource := "github.com"
		repoOwner := "estafette"
		repoName := "estafette-ci-api"
		buildID := 1557
		buildStatus := contracts.StatusSucceeded

		// act
		err := service.FinishBuild(ctx, repoSource, repoOwner, repoName, buildID, buildStatus)

		assert.Nil(t, err)
	})
}

func TestCreateRelease(t *testing.T) {

	t.Run("CallsInsertReleaseOnCockroachdbClient", func(t *testing.T) {

		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		ctx := context.Background()

		config := &api.APIConfig{
			Jobs:      &api.JobsConfig{},
			APIServer: &api.APIServerConfig{},
		}
		cockroachdbClient := cockroachdb.NewMockClient(ctrl)
		secretHelper := crypt.NewSecretHelper("abc", false)
		prometheusClient := prometheus.NewMockClient(ctrl)
		cloudStorageClient := cloudstorage.NewMockClient(ctrl)
		builderapiClient := builderapi.NewMockClient(ctrl)

		cockroachdbClient.
			EXPECT().
			InsertRelease(gomock.Any(), gomock.Any(), gomock.Any()).
			Return(&contracts.Release{ID: "15"}, nil).
			Times(1)

		githubapiClientJobVarsFunc := func(ctx context.Context, repoSource, repoOwner, repoName string) (token string, url string, err error) {
			return
		}
		bitbucketapiClientJobVarsFunc := func(ctx context.Context, repoSource, repoOwner, repoName string) (token string, url string, err error) {
			return
		}
		cloudsourceapiClientJobVarsFunc := func(ctx context.Context, repoSource, repoOwner, repoName string) (token string, url string, err error) {
			return
		}

		cockroachdbClient.EXPECT().GetPipelineReleaseMaxResourceUtilization(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes()
		cockroachdbClient.EXPECT().GetPipeline(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes()
		builderapiClient.EXPECT().CreateCiBuilderJob(gomock.Any(), gomock.Any()).AnyTimes()
		cockroachdbClient.EXPECT().GetReleaseTriggers(gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes()
		cockroachdbClient.EXPECT().GetPipelineBuildByID(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes()

		service := NewService(config, cockroachdbClient, secretHelper, prometheusClient, cloudStorageClient, builderapiClient, githubapiClientJobVarsFunc, bitbucketapiClientJobVarsFunc, cloudsourceapiClientJobVarsFunc)

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
		_, err := service.CreateRelease(ctx, release, mft, branch, revision, true)

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
		cockroachdbClient := cockroachdb.NewMockClient(ctrl)
		secretHelper := crypt.NewSecretHelper("abc", false)
		prometheusClient := prometheus.NewMockClient(ctrl)
		cloudStorageClient := cloudstorage.NewMockClient(ctrl)
		builderapiClient := builderapi.NewMockClient(ctrl)

		cockroachdbClient.
			EXPECT().
			InsertRelease(gomock.Any(), gomock.Any(), gomock.Any()).
			DoAndReturn(func(ctx context.Context, release contracts.Release, jobResources cockroachdb.JobResources) (r *contracts.Release, err error) {
				r = &release
				r.ID = "5"
				return
			})

		builderapiClient.
			EXPECT().
			CreateCiBuilderJob(gomock.Any(), gomock.Any()).
			Times(1)

		githubapiClientJobVarsFunc := func(ctx context.Context, repoSource, repoOwner, repoName string) (token string, url string, err error) {
			return
		}
		bitbucketapiClientJobVarsFunc := func(ctx context.Context, repoSource, repoOwner, repoName string) (token string, url string, err error) {
			return
		}
		cloudsourceapiClientJobVarsFunc := func(ctx context.Context, repoSource, repoOwner, repoName string) (token string, url string, err error) {
			return
		}

		cockroachdbClient.EXPECT().GetPipelineReleaseMaxResourceUtilization(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes()
		cockroachdbClient.EXPECT().GetPipeline(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes()
		cockroachdbClient.EXPECT().GetPipelineBuildByID(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes()
		cockroachdbClient.EXPECT().GetReleaseTriggers(gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes()

		service := NewService(config, cockroachdbClient, secretHelper, prometheusClient, cloudStorageClient, builderapiClient, githubapiClientJobVarsFunc, bitbucketapiClientJobVarsFunc, cloudsourceapiClientJobVarsFunc)

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
		_, err := service.CreateRelease(ctx, release, mft, branch, revision, true)

		assert.Nil(t, err)
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
		secretHelper := crypt.NewSecretHelper("abc", false)
		prometheusClient := prometheus.NewMockClient(ctrl)
		cloudStorageClient := cloudstorage.NewMockClient(ctrl)
		builderapiClient := builderapi.NewMockClient(ctrl)

		cockroachdbClient.
			EXPECT().
			UpdateReleaseStatus(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
			Times(1)

		githubapiClientJobVarsFunc := func(ctx context.Context, repoSource, repoOwner, repoName string) (token string, url string, err error) {
			return
		}
		bitbucketapiClientJobVarsFunc := func(ctx context.Context, repoSource, repoOwner, repoName string) (token string, url string, err error) {
			return
		}
		cloudsourceapiClientJobVarsFunc := func(ctx context.Context, repoSource, repoOwner, repoName string) (token string, url string, err error) {
			return
		}

		cockroachdbClient.EXPECT().GetPipelineRelease(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes()

		service := NewService(config, cockroachdbClient, secretHelper, prometheusClient, cloudStorageClient, builderapiClient, githubapiClientJobVarsFunc, bitbucketapiClientJobVarsFunc, cloudsourceapiClientJobVarsFunc)

		repoSource := "github.com"
		repoOwner := "estafette"
		repoName := "estafette-ci-api"
		buildID := 1557
		buildStatus := contracts.StatusSucceeded

		// act
		err := service.FinishRelease(ctx, repoSource, repoOwner, repoName, buildID, buildStatus)

		assert.Nil(t, err)
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
		secretHelper := crypt.NewSecretHelper("abc", false)
		prometheusClient := prometheus.NewMockClient(ctrl)
		cloudStorageClient := cloudstorage.NewMockClient(ctrl)
		builderapiClient := builderapi.NewMockClient(ctrl)

		cockroachdbClient.
			EXPECT().
			Rename(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
			Times(1)

		githubapiClientJobVarsFunc := func(ctx context.Context, repoSource, repoOwner, repoName string) (token string, url string, err error) {
			return
		}
		bitbucketapiClientJobVarsFunc := func(ctx context.Context, repoSource, repoOwner, repoName string) (token string, url string, err error) {
			return
		}
		cloudsourceapiClientJobVarsFunc := func(ctx context.Context, repoSource, repoOwner, repoName string) (token string, url string, err error) {
			return
		}

		service := NewService(config, cockroachdbClient, secretHelper, prometheusClient, cloudStorageClient, builderapiClient, githubapiClientJobVarsFunc, bitbucketapiClientJobVarsFunc, cloudsourceapiClientJobVarsFunc)

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
				LogWriters: []string{"cloudstorage"},
			},
		}
		cockroachdbClient := cockroachdb.NewMockClient(ctrl)
		secretHelper := crypt.NewSecretHelper("abc", false)
		prometheusClient := prometheus.NewMockClient(ctrl)
		cloudStorageClient := cloudstorage.NewMockClient(ctrl)
		builderapiClient := builderapi.NewMockClient(ctrl)

		cloudStorageClient.
			EXPECT().
			Rename(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
			Times(1)

		githubapiClientJobVarsFunc := func(ctx context.Context, repoSource, repoOwner, repoName string) (token string, url string, err error) {
			return
		}
		bitbucketapiClientJobVarsFunc := func(ctx context.Context, repoSource, repoOwner, repoName string) (token string, url string, err error) {
			return
		}
		cloudsourceapiClientJobVarsFunc := func(ctx context.Context, repoSource, repoOwner, repoName string) (token string, url string, err error) {
			return
		}

		cockroachdbClient.EXPECT().Rename(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes()

		service := NewService(config, cockroachdbClient, secretHelper, prometheusClient, cloudStorageClient, builderapiClient, githubapiClientJobVarsFunc, bitbucketapiClientJobVarsFunc, cloudsourceapiClientJobVarsFunc)

		// act
		err := service.Rename(ctx, "github.com", "estafette", "estafette-ci-contracts", "github.com", "estafette", "estafette-ci-protos")

		assert.Nil(t, err)
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
		secretHelper := crypt.NewSecretHelper("abc", false)
		prometheusClient := prometheus.NewMockClient(ctrl)
		cloudStorageClient := cloudstorage.NewMockClient(ctrl)
		builderapiClient := builderapi.NewMockClient(ctrl)

		cockroachdbClient.
			EXPECT().
			UpdateBuildStatus(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
			Times(1)

		githubapiClientJobVarsFunc := func(ctx context.Context, repoSource, repoOwner, repoName string) (token string, url string, err error) {
			return
		}
		bitbucketapiClientJobVarsFunc := func(ctx context.Context, repoSource, repoOwner, repoName string) (token string, url string, err error) {
			return
		}
		cloudsourceapiClientJobVarsFunc := func(ctx context.Context, repoSource, repoOwner, repoName string) (token string, url string, err error) {
			return
		}

		cockroachdbClient.EXPECT().GetPipelineBuildByID(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes()
		cockroachdbClient.EXPECT().GetPipelineRelease(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes()

		service := NewService(config, cockroachdbClient, secretHelper, prometheusClient, cloudStorageClient, builderapiClient, githubapiClientJobVarsFunc, bitbucketapiClientJobVarsFunc, cloudsourceapiClientJobVarsFunc)

		event := builderapi.CiBuilderEvent{
			BuildID:     "123456",
			BuildStatus: "succeeeded",
		}

		// act
		err := service.UpdateBuildStatus(ctx, event)

		assert.Nil(t, err)
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
		secretHelper := crypt.NewSecretHelper("abc", false)
		prometheusClient := prometheus.NewMockClient(ctrl)
		cloudStorageClient := cloudstorage.NewMockClient(ctrl)
		builderapiClient := builderapi.NewMockClient(ctrl)

		cockroachdbClient.
			EXPECT().
			UpdateReleaseStatus(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
			Times(1)

		githubapiClientJobVarsFunc := func(ctx context.Context, repoSource, repoOwner, repoName string) (token string, url string, err error) {
			return
		}
		bitbucketapiClientJobVarsFunc := func(ctx context.Context, repoSource, repoOwner, repoName string) (token string, url string, err error) {
			return
		}
		cloudsourceapiClientJobVarsFunc := func(ctx context.Context, repoSource, repoOwner, repoName string) (token string, url string, err error) {
			return
		}

		cockroachdbClient.EXPECT().GetPipelineBuildByID(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes()
		cockroachdbClient.EXPECT().GetPipelineRelease(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes()

		service := NewService(config, cockroachdbClient, secretHelper, prometheusClient, cloudStorageClient, builderapiClient, githubapiClientJobVarsFunc, bitbucketapiClientJobVarsFunc, cloudsourceapiClientJobVarsFunc)

		event := builderapi.CiBuilderEvent{
			ReleaseID:   "123456",
			BuildStatus: "succeeeded",
		}

		// act
		err := service.UpdateBuildStatus(ctx, event)

		assert.Nil(t, err)
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
		secretHelper := crypt.NewSecretHelper("abc", false)
		prometheusClient := prometheus.NewMockClient(ctrl)
		cloudStorageClient := cloudstorage.NewMockClient(ctrl)
		builderapiClient := builderapi.NewMockClient(ctrl)

		cockroachdbClient.
			EXPECT().
			UpdateBuildResourceUtilization(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
			Times(1)

		githubapiClientJobVarsFunc := func(ctx context.Context, repoSource, repoOwner, repoName string) (token string, url string, err error) {
			return
		}
		bitbucketapiClientJobVarsFunc := func(ctx context.Context, repoSource, repoOwner, repoName string) (token string, url string, err error) {
			return
		}
		cloudsourceapiClientJobVarsFunc := func(ctx context.Context, repoSource, repoOwner, repoName string) (token string, url string, err error) {
			return
		}

		prometheusClient.EXPECT().AwaitScrapeInterval(gomock.Any()).AnyTimes()
		prometheusClient.EXPECT().GetMaxCPUByPodName(gomock.Any(), gomock.Any()).AnyTimes()
		prometheusClient.EXPECT().GetMaxMemoryByPodName(gomock.Any(), gomock.Any()).AnyTimes()

		service := NewService(config, cockroachdbClient, secretHelper, prometheusClient, cloudStorageClient, builderapiClient, githubapiClientJobVarsFunc, bitbucketapiClientJobVarsFunc, cloudsourceapiClientJobVarsFunc)

		event := builderapi.CiBuilderEvent{
			PodName:     "build-estafette-estafette-ci-api-123456-mhrzk",
			BuildID:     "123456",
			BuildStatus: "succeeeded",
		}

		// act
		err := service.UpdateJobResources(ctx, event)

		assert.Nil(t, err)
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
		secretHelper := crypt.NewSecretHelper("abc", false)
		prometheusClient := prometheus.NewMockClient(ctrl)
		cloudStorageClient := cloudstorage.NewMockClient(ctrl)
		builderapiClient := builderapi.NewMockClient(ctrl)

		cockroachdbClient.
			EXPECT().
			UpdateReleaseResourceUtilization(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
			Times(1)

		githubapiClientJobVarsFunc := func(ctx context.Context, repoSource, repoOwner, repoName string) (token string, url string, err error) {
			return
		}
		bitbucketapiClientJobVarsFunc := func(ctx context.Context, repoSource, repoOwner, repoName string) (token string, url string, err error) {
			return
		}
		cloudsourceapiClientJobVarsFunc := func(ctx context.Context, repoSource, repoOwner, repoName string) (token string, url string, err error) {
			return
		}

		prometheusClient.EXPECT().AwaitScrapeInterval(gomock.Any()).AnyTimes()
		prometheusClient.EXPECT().GetMaxCPUByPodName(gomock.Any(), gomock.Any()).AnyTimes()
		prometheusClient.EXPECT().GetMaxMemoryByPodName(gomock.Any(), gomock.Any()).AnyTimes()

		service := NewService(config, cockroachdbClient, secretHelper, prometheusClient, cloudStorageClient, builderapiClient, githubapiClientJobVarsFunc, bitbucketapiClientJobVarsFunc, cloudsourceapiClientJobVarsFunc)

		event := builderapi.CiBuilderEvent{
			PodName:     "release-estafette-estafette-ci-api-123456-mhrzk",
			ReleaseID:   "123456",
			BuildStatus: "succeeeded",
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
		cockroachdbClient := cockroachdb.NewMockClient(ctrl)
		secretHelper := crypt.NewSecretHelper("abc", false)
		prometheusClient := prometheus.NewMockClient(ctrl)
		cloudStorageClient := cloudstorage.NewMockClient(ctrl)
		builderapiClient := builderapi.NewMockClient(ctrl)

		githubapiClientJobVarsFunc := func(ctx context.Context, repoSource, repoOwner, repoName string) (token string, url string, err error) {
			return
		}
		bitbucketapiClientJobVarsFunc := func(ctx context.Context, repoSource, repoOwner, repoName string) (token string, url string, err error) {
			return
		}
		cloudsourceapiClientJobVarsFunc := func(ctx context.Context, repoSource, repoOwner, repoName string) (token string, url string, err error) {
			return
		}

		cockroachdbClient.EXPECT().GetPipelineBuilds(gomock.Any(), gomock.Eq("github.com"), gomock.Eq("estafette"), gomock.Eq("repo1"), gomock.Eq(1), gomock.Eq(1), gomock.Any(), gomock.Any(), gomock.Eq(false)).
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

		cockroachdbClient.EXPECT().GetPipelineBuilds(gomock.Any(), gomock.Eq("github.com"), gomock.Eq("estafette"), gomock.Eq("repo2"), gomock.Eq(1), gomock.Eq(1), gomock.Any(), gomock.Any(), gomock.Eq(false)).
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

		service := NewService(config, cockroachdbClient, secretHelper, prometheusClient, cloudStorageClient, builderapiClient, githubapiClientJobVarsFunc, bitbucketapiClientJobVarsFunc, cloudsourceapiClientJobVarsFunc)

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
