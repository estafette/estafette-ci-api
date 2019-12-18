package estafette

import (
	"context"
	"testing"

	batchv1 "github.com/ericchiang/k8s/apis/batch/v1"
	"github.com/estafette/estafette-ci-api/clients/bitbucketapi"
	"github.com/estafette/estafette-ci-api/clients/builderapi"
	"github.com/estafette/estafette-ci-api/clients/cloudstorage"
	"github.com/estafette/estafette-ci-api/clients/cockroachdb"
	"github.com/estafette/estafette-ci-api/clients/githubapi"
	"github.com/estafette/estafette-ci-api/clients/prometheus"
	"github.com/estafette/estafette-ci-api/config"
	contracts "github.com/estafette/estafette-ci-contracts"
	"github.com/stretchr/testify/assert"
)

func TestCreateBuild(t *testing.T) {

	t.Run("CallsGetAutoIncrementOnCockroachdbClient", func(t *testing.T) {

		ctx := context.Background()

		jobsConfig := config.JobsConfig{}
		apiServerConfig := config.APIServerConfig{}
		cockroachdbClient := cockroachdb.MockClient{}
		prometheusClient := prometheus.MockClient{}
		cloudStorageClient := cloudstorage.MockClient{}
		builderapiClient := builderapi.MockClient{}
		githubapiClient := githubapi.MockClient{}
		bitbucketapiClient := bitbucketapi.MockClient{}

		callCount := 0
		cockroachdbClient.GetAutoIncrementFunc = func(ctx context.Context, shortRepoSource, repoOwner, repoName string) (autoincrement int, err error) {
			callCount++
			return
		}

		service := NewService(jobsConfig, apiServerConfig, cockroachdbClient, prometheusClient, cloudStorageClient, builderapiClient, githubapiClient.JobVarsFunc(ctx), bitbucketapiClient.JobVarsFunc(ctx))

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

	t.Run("CallsGetPipelineBuildMaxResourceUtilizationOnCockroachdbClient", func(t *testing.T) {

		ctx := context.Background()

		jobsConfig := config.JobsConfig{}
		apiServerConfig := config.APIServerConfig{}
		cockroachdbClient := cockroachdb.MockClient{}
		prometheusClient := prometheus.MockClient{}
		cloudStorageClient := cloudstorage.MockClient{}
		builderapiClient := builderapi.MockClient{}
		githubapiClient := githubapi.MockClient{}
		bitbucketapiClient := bitbucketapi.MockClient{}

		callCount := 0
		cockroachdbClient.GetPipelineBuildMaxResourceUtilizationFunc = func(ctx context.Context, repoSource, repoOwner, repoName string, lastNRecords int) (jobresources cockroachdb.JobResources, count int, err error) {
			callCount++
			return
		}

		service := NewService(jobsConfig, apiServerConfig, cockroachdbClient, prometheusClient, cloudStorageClient, builderapiClient, githubapiClient.JobVarsFunc(ctx), bitbucketapiClient.JobVarsFunc(ctx))

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

		ctx := context.Background()

		jobsConfig := config.JobsConfig{}
		apiServerConfig := config.APIServerConfig{}
		cockroachdbClient := cockroachdb.MockClient{}
		prometheusClient := prometheus.MockClient{}
		cloudStorageClient := cloudstorage.MockClient{}
		builderapiClient := builderapi.MockClient{}
		githubapiClient := githubapi.MockClient{}
		bitbucketapiClient := bitbucketapi.MockClient{}

		callCount := 0
		cockroachdbClient.InsertBuildFunc = func(ctx context.Context, build contracts.Build, jobResources cockroachdb.JobResources) (b *contracts.Build, err error) {
			callCount++
			return
		}

		service := NewService(jobsConfig, apiServerConfig, cockroachdbClient, prometheusClient, cloudStorageClient, builderapiClient, githubapiClient.JobVarsFunc(ctx), bitbucketapiClient.JobVarsFunc(ctx))

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

		ctx := context.Background()

		jobsConfig := config.JobsConfig{}
		apiServerConfig := config.APIServerConfig{}
		cockroachdbClient := cockroachdb.MockClient{}
		prometheusClient := prometheus.MockClient{}
		cloudStorageClient := cloudstorage.MockClient{}
		builderapiClient := builderapi.MockClient{}
		githubapiClient := githubapi.MockClient{}
		bitbucketapiClient := bitbucketapi.MockClient{}

		cockroachdbClient.InsertBuildFunc = func(ctx context.Context, build contracts.Build, jobResources cockroachdb.JobResources) (b *contracts.Build, err error) {
			b = &build
			b.ID = "5"
			return
		}
		callCount := 0
		builderapiClient.CreateCiBuilderJobFunc = func(ctx context.Context, params builderapi.CiBuilderParams) (job *batchv1.Job, err error) {
			callCount++
			return
		}

		service := NewService(jobsConfig, apiServerConfig, cockroachdbClient, prometheusClient, cloudStorageClient, builderapiClient, githubapiClient.JobVarsFunc(ctx), bitbucketapiClient.JobVarsFunc(ctx))

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

		ctx := context.Background()

		jobsConfig := config.JobsConfig{}
		apiServerConfig := config.APIServerConfig{}
		cockroachdbClient := cockroachdb.MockClient{}
		prometheusClient := prometheus.MockClient{}
		cloudStorageClient := cloudstorage.MockClient{}
		builderapiClient := builderapi.MockClient{}
		githubapiClient := githubapi.MockClient{}
		bitbucketapiClient := bitbucketapi.MockClient{}

		cockroachdbClient.InsertBuildFunc = func(ctx context.Context, build contracts.Build, jobResources cockroachdb.JobResources) (b *contracts.Build, err error) {
			b = &build
			b.ID = "5"
			return
		}
		createcibuilderjobCallCount := 0
		builderapiClient.CreateCiBuilderJobFunc = func(ctx context.Context, params builderapi.CiBuilderParams) (job *batchv1.Job, err error) {
			createcibuilderjobCallCount++
			return
		}

		insertbuildlogCallCount := 0
		cockroachdbClient.InsertBuildLogFunc = func(ctx context.Context, buildLog contracts.BuildLog, writeLogToDatabase bool) (log contracts.BuildLog, err error) {
			insertbuildlogCallCount++
			return
		}

		service := NewService(jobsConfig, apiServerConfig, cockroachdbClient, prometheusClient, cloudStorageClient, builderapiClient, githubapiClient.JobVarsFunc(ctx), bitbucketapiClient.JobVarsFunc(ctx))

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

		ctx := context.Background()

		jobsConfig := config.JobsConfig{}
		apiServerConfig := config.APIServerConfig{
			LogWriters: []string{"cloudstorage"},
		}
		cockroachdbClient := cockroachdb.MockClient{}
		prometheusClient := prometheus.MockClient{}
		cloudStorageClient := cloudstorage.MockClient{}
		builderapiClient := builderapi.MockClient{}
		githubapiClient := githubapi.MockClient{}
		bitbucketapiClient := bitbucketapi.MockClient{}

		cockroachdbClient.InsertBuildFunc = func(ctx context.Context, build contracts.Build, jobResources cockroachdb.JobResources) (b *contracts.Build, err error) {
			b = &build
			b.ID = "5"
			return
		}
		createcibuilderjobCallCount := 0
		builderapiClient.CreateCiBuilderJobFunc = func(ctx context.Context, params builderapi.CiBuilderParams) (job *batchv1.Job, err error) {
			createcibuilderjobCallCount++
			return
		}

		insertbuildlogCallCount := 0
		cloudStorageClient.InsertBuildLogFunc = func(ctx context.Context, buildLog contracts.BuildLog) (err error) {
			insertbuildlogCallCount++
			return
		}

		service := NewService(jobsConfig, apiServerConfig, cockroachdbClient, prometheusClient, cloudStorageClient, builderapiClient, githubapiClient.JobVarsFunc(ctx), bitbucketapiClient.JobVarsFunc(ctx))

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
}

func TestRename(t *testing.T) {

	t.Run("CallsRenameOnCockroachdbClient", func(t *testing.T) {

		ctx := context.Background()

		jobsConfig := config.JobsConfig{}
		apiServerConfig := config.APIServerConfig{}
		cockroachdbClient := cockroachdb.MockClient{}
		prometheusClient := prometheus.MockClient{}
		cloudStorageClient := cloudstorage.MockClient{}
		builderapiClient := builderapi.MockClient{}

		githubapiClient := githubapi.MockClient{}
		bitbucketapiClient := bitbucketapi.MockClient{}

		renameOnCockroachdbClientCallCount := 0
		cockroachdbClient.RenameFunc = func(ctx context.Context, shortFromRepoSource, fromRepoSource, fromRepoOwner, fromRepoName, shortToRepoSource, toRepoSource, toRepoOwner, toRepoName string) (err error) {
			renameOnCockroachdbClientCallCount++
			return
		}

		service := NewService(jobsConfig, apiServerConfig, cockroachdbClient, prometheusClient, cloudStorageClient, builderapiClient, githubapiClient.JobVarsFunc(ctx), bitbucketapiClient.JobVarsFunc(ctx))

		// act
		err := service.Rename(context.Background(), "github.com", "estafette", "estafette-ci-contracts", "github.com", "estafette", "estafette-ci-protos")

		assert.Nil(t, err)
		assert.Equal(t, 1, renameOnCockroachdbClientCallCount)
	})

	t.Run("CallsRenameOnCloudstorageClientIfLogWritersConfigContainsCloudstorage", func(t *testing.T) {

		ctx := context.Background()

		jobsConfig := config.JobsConfig{}
		apiServerConfig := config.APIServerConfig{
			LogWriters: []string{"cloudstorage"},
		}
		cockroachdbClient := cockroachdb.MockClient{}
		prometheusClient := prometheus.MockClient{}
		cloudStorageClient := cloudstorage.MockClient{}
		builderapiClient := builderapi.MockClient{}

		githubapiClient := githubapi.MockClient{}
		bitbucketapiClient := bitbucketapi.MockClient{}

		renameOnCloudstorageClientCallCount := 0
		cloudStorageClient.RenameFunc = func(ctx context.Context, fromRepoSource, fromRepoOwner, fromRepoName, toRepoSource, toRepoOwner, toRepoName string) (err error) {
			renameOnCloudstorageClientCallCount++
			return
		}

		service := NewService(jobsConfig, apiServerConfig, cockroachdbClient, prometheusClient, cloudStorageClient, builderapiClient, githubapiClient.JobVarsFunc(ctx), bitbucketapiClient.JobVarsFunc(ctx))

		// act
		err := service.Rename(context.Background(), "github.com", "estafette", "estafette-ci-contracts", "github.com", "estafette", "estafette-ci-protos")

		assert.Nil(t, err)
		assert.Equal(t, 1, renameOnCloudstorageClientCallCount)
	})
}
