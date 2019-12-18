package estafette

import (
	"context"
	"testing"

	"github.com/estafette/estafette-ci-api/clients/bitbucketapi"
	"github.com/estafette/estafette-ci-api/clients/builderapi"
	"github.com/estafette/estafette-ci-api/clients/cloudstorage"
	"github.com/estafette/estafette-ci-api/clients/cockroachdb"
	"github.com/estafette/estafette-ci-api/clients/githubapi"
	"github.com/estafette/estafette-ci-api/clients/prometheus"
	"github.com/estafette/estafette-ci-api/config"
	"github.com/stretchr/testify/assert"
)

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
