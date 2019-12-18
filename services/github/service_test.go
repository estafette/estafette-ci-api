package github

import (
	"context"
	"errors"
	"sync"
	"testing"

	"github.com/estafette/estafette-ci-api/clients/githubapi"
	"github.com/estafette/estafette-ci-api/clients/pubsubapi"
	"github.com/estafette/estafette-ci-api/config"
	"github.com/estafette/estafette-ci-api/services/estafette"
	contracts "github.com/estafette/estafette-ci-contracts"
	manifest "github.com/estafette/estafette-ci-manifest"
	"github.com/stretchr/testify/assert"
)

func TestCreateJobForGithubPush(t *testing.T) {

	t.Run("ReturnsErrNonCloneableEventIfPushEventHasNoRefsHeadsPrefix", func(t *testing.T) {

		config := config.GithubConfig{}
		githubapiClient := githubapi.MockClient{}
		pubsubapiClient := pubsubapi.MockClient{}
		estafetteService := estafette.MockService{}
		service := NewService(config, githubapiClient, pubsubapiClient, estafetteService)

		pushEvent := githubapi.PushEvent{
			Ref: "refs/noheads",
		}

		// act
		err := service.CreateJobForGithubPush(context.Background(), pushEvent)

		assert.NotNil(t, err)
		assert.True(t, errors.Is(err, ErrNonCloneableEvent))
	})

	t.Run("CallsGetInstallationTokenOnGithubapiClient", func(t *testing.T) {

		config := config.GithubConfig{}
		githubapiClient := githubapi.MockClient{}
		pubsubapiClient := pubsubapi.MockClient{}
		estafetteService := estafette.MockService{}

		getInstallationTokenCallCount := 0
		githubapiClient.GetInstallationTokenFunc = func(ctx context.Context, installationID int) (token githubapi.AccessToken, err error) {
			getInstallationTokenCallCount++
			return
		}

		service := NewService(config, githubapiClient, pubsubapiClient, estafetteService)

		pushEvent := githubapi.PushEvent{
			Ref: "refs/heads/master",
		}

		// act
		_ = service.CreateJobForGithubPush(context.Background(), pushEvent)

		assert.Equal(t, 1, getInstallationTokenCallCount)
	})

	t.Run("CallsGetEstafetteManifestOnGithubapiClient", func(t *testing.T) {

		config := config.GithubConfig{}
		githubapiClient := githubapi.MockClient{}
		pubsubapiClient := pubsubapi.MockClient{}
		estafetteService := estafette.MockService{}

		getEstafetteManifestCallCount := 0
		githubapiClient.GetEstafetteManifestFunc = func(ctx context.Context, accesstoken githubapi.AccessToken, event githubapi.PushEvent) (valid bool, manifest string, err error) {
			getEstafetteManifestCallCount++
			return true, "builder:\n  track: dev\n", nil
		}

		service := NewService(config, githubapiClient, pubsubapiClient, estafetteService)

		pushEvent := githubapi.PushEvent{
			Ref: "refs/heads/master",
		}

		// act
		err := service.CreateJobForGithubPush(context.Background(), pushEvent)

		assert.Nil(t, err)
		assert.Equal(t, 1, getEstafetteManifestCallCount)
	})

	t.Run("CallsCreateBuildOnEstafetteService", func(t *testing.T) {

		config := config.GithubConfig{}
		githubapiClient := githubapi.MockClient{}
		pubsubapiClient := pubsubapi.MockClient{}
		estafetteService := estafette.MockService{}

		githubapiClient.GetEstafetteManifestFunc = func(ctx context.Context, accesstoken githubapi.AccessToken, event githubapi.PushEvent) (valid bool, manifest string, err error) {
			return true, "builder:\n  track: dev\n", nil
		}

		createBuildCallCount := 0
		estafetteService.CreateBuildFunc = func(ctx context.Context, build contracts.Build, waitForJobToStart bool) (b *contracts.Build, err error) {
			createBuildCallCount++
			return
		}

		service := NewService(config, githubapiClient, pubsubapiClient, estafetteService)

		pushEvent := githubapi.PushEvent{
			Ref: "refs/heads/master",
		}

		// act
		err := service.CreateJobForGithubPush(context.Background(), pushEvent)

		assert.Nil(t, err)
		assert.Equal(t, 1, createBuildCallCount)
	})

	t.Run("CallsFireGitTriggersOnEstafetteService", func(t *testing.T) {

		config := config.GithubConfig{}
		githubapiClient := githubapi.MockClient{}
		pubsubapiClient := pubsubapi.MockClient{}
		estafetteService := estafette.MockService{}

		var wg sync.WaitGroup
		wg.Add(1)
		fireGitTriggersCallCount := 0
		estafetteService.FireGitTriggersFunc = func(ctx context.Context, gitEvent manifest.EstafetteGitEvent) (err error) {
			fireGitTriggersCallCount++
			wg.Done()
			return
		}

		service := NewService(config, githubapiClient, pubsubapiClient, estafetteService)

		pushEvent := githubapi.PushEvent{
			Ref: "refs/heads/master",
		}

		// act
		_ = service.CreateJobForGithubPush(context.Background(), pushEvent)

		wg.Wait()

		assert.Equal(t, 1, fireGitTriggersCallCount)
	})

	t.Run("CallsSubscribeToPubsubTriggersOnPubsubAPIClient", func(t *testing.T) {

		config := config.GithubConfig{}
		githubapiClient := githubapi.MockClient{}
		pubsubapiClient := pubsubapi.MockClient{}
		estafetteService := estafette.MockService{}

		githubapiClient.GetEstafetteManifestFunc = func(ctx context.Context, accesstoken githubapi.AccessToken, event githubapi.PushEvent) (valid bool, manifest string, err error) {
			return true, "builder:\n  track: dev\n", nil
		}

		var wg sync.WaitGroup
		wg.Add(1)
		subscribeToPubsubTriggersCallCount := 0
		pubsubapiClient.SubscribeToPubsubTriggersFunc = func(ctx context.Context, manifestString string) (err error) {
			subscribeToPubsubTriggersCallCount++
			wg.Done()
			return
		}

		service := NewService(config, githubapiClient, pubsubapiClient, estafetteService)

		pushEvent := githubapi.PushEvent{
			Ref: "refs/heads/master",
		}

		// act
		err := service.CreateJobForGithubPush(context.Background(), pushEvent)

		wg.Wait()

		assert.Nil(t, err)
		assert.Equal(t, 1, subscribeToPubsubTriggersCallCount)
	})
}

func TestIsWhitelistedInstallation(t *testing.T) {

	t.Run("ReturnsTrueIfWhitelistedInstallationsConfigIsEmpty", func(t *testing.T) {

		config := config.GithubConfig{
			WhitelistedInstallations: []int{},
		}
		githubapiClient := githubapi.MockClient{}
		pubsubapiClient := pubsubapi.MockClient{}
		estafetteService := estafette.MockService{}
		service := NewService(config, githubapiClient, pubsubapiClient, estafetteService)

		installation := githubapi.Installation{
			ID: 513,
		}

		// act
		isWhitelisted := service.IsWhitelistedInstallation(context.Background(), installation)

		assert.True(t, isWhitelisted)
	})

	t.Run("ReturnsFalseIfInstallationIDIsNotInWhitelistedInstallationsConfig", func(t *testing.T) {

		config := config.GithubConfig{
			WhitelistedInstallations: []int{
				236,
			},
		}
		githubapiClient := githubapi.MockClient{}
		pubsubapiClient := pubsubapi.MockClient{}
		estafetteService := estafette.MockService{}
		service := NewService(config, githubapiClient, pubsubapiClient, estafetteService)

		installation := githubapi.Installation{
			ID: 513,
		}

		// act
		isWhitelisted := service.IsWhitelistedInstallation(context.Background(), installation)

		assert.False(t, isWhitelisted)
	})

	t.Run("ReturnsTrueIfInstallationIDIsInWhitelistedInstallationsConfig", func(t *testing.T) {

		config := config.GithubConfig{
			WhitelistedInstallations: []int{
				236,
				513,
			},
		}
		githubapiClient := githubapi.MockClient{}
		pubsubapiClient := pubsubapi.MockClient{}
		estafetteService := estafette.MockService{}
		service := NewService(config, githubapiClient, pubsubapiClient, estafetteService)

		installation := githubapi.Installation{
			ID: 513,
		}

		// act
		isWhitelisted := service.IsWhitelistedInstallation(context.Background(), installation)

		assert.True(t, isWhitelisted)
	})
}

func TestRename(t *testing.T) {

	t.Run("CallsRenameOnEstafetteService", func(t *testing.T) {

		config := config.GithubConfig{
			WhitelistedInstallations: []int{},
		}
		githubapiClient := githubapi.MockClient{}
		pubsubapiClient := pubsubapi.MockClient{}
		estafetteService := estafette.MockService{}

		renameCallCount := 0
		estafetteService.RenameFunc = func(ctx context.Context, fromRepoSource, fromRepoOwner, fromRepoName, toRepoSource, toRepoOwner, toRepoName string) (err error) {
			renameCallCount++
			return
		}

		service := NewService(config, githubapiClient, pubsubapiClient, estafetteService)

		// act
		err := service.Rename(context.Background(), "github.com", "estafette", "estafette-ci-contracts", "github.com", "estafette", "estafette-ci-protos")

		assert.Nil(t, err)
		assert.Equal(t, 1, renameCallCount)
	})
}
