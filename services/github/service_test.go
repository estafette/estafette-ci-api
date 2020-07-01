package github

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/estafette/estafette-ci-api/api"
	"github.com/estafette/estafette-ci-api/clients/githubapi"
	"github.com/estafette/estafette-ci-api/clients/pubsubapi"
	"github.com/estafette/estafette-ci-api/services/estafette"
	contracts "github.com/estafette/estafette-ci-contracts"
	"github.com/stretchr/testify/assert"
)

func TestCreateJobForGithubPush(t *testing.T) {

	t.Run("ReturnsErrNonCloneableEventIfPushEventHasNoRefsHeadsPrefix", func(t *testing.T) {

		config := &api.APIConfig{
			Integrations: &api.APIConfigIntegrations{
				Github: &api.GithubConfig{},
			},
		}
		githubapiClient := githubapi.MockClient{}
		pubsubapiClient := pubsubapi.MockClient{}
		estafetteService := estafette.MockService{}
		service := NewService(config, githubapiClient, pubsubapiClient, estafetteService, api.NewGitEventTopic("test topic"))

		pushEvent := githubapi.PushEvent{
			Ref: "refs/noheads",
		}

		// act
		err := service.CreateJobForGithubPush(context.Background(), pushEvent)

		assert.NotNil(t, err)
		assert.True(t, errors.Is(err, ErrNonCloneableEvent))
	})

	t.Run("CallsGetInstallationTokenOnGithubapiClient", func(t *testing.T) {

		config := &api.APIConfig{
			Integrations: &api.APIConfigIntegrations{
				Github: &api.GithubConfig{},
			},
		}
		githubapiClient := githubapi.MockClient{}
		pubsubapiClient := pubsubapi.MockClient{}
		estafetteService := estafette.MockService{}

		getInstallationTokenCallCount := 0
		githubapiClient.GetInstallationTokenFunc = func(ctx context.Context, installationID int) (token githubapi.AccessToken, err error) {
			getInstallationTokenCallCount++
			return
		}

		service := NewService(config, githubapiClient, pubsubapiClient, estafetteService, api.NewGitEventTopic("test topic"))

		pushEvent := githubapi.PushEvent{
			Ref: "refs/heads/master",
		}

		// act
		_ = service.CreateJobForGithubPush(context.Background(), pushEvent)

		assert.Equal(t, 1, getInstallationTokenCallCount)
	})

	t.Run("CallsGetEstafetteManifestOnGithubapiClient", func(t *testing.T) {

		config := &api.APIConfig{
			Integrations: &api.APIConfigIntegrations{
				Github: &api.GithubConfig{},
			},
		}
		githubapiClient := githubapi.MockClient{}
		pubsubapiClient := pubsubapi.MockClient{}
		estafetteService := estafette.MockService{}

		getEstafetteManifestCallCount := 0
		githubapiClient.GetEstafetteManifestFunc = func(ctx context.Context, accesstoken githubapi.AccessToken, event githubapi.PushEvent) (valid bool, manifest string, err error) {
			getEstafetteManifestCallCount++
			return true, "builder:\n  track: dev\n", nil
		}

		service := NewService(config, githubapiClient, pubsubapiClient, estafetteService, api.NewGitEventTopic("test topic"))

		pushEvent := githubapi.PushEvent{
			Ref: "refs/heads/master",
		}

		// act
		err := service.CreateJobForGithubPush(context.Background(), pushEvent)

		assert.Nil(t, err)
		assert.Equal(t, 1, getEstafetteManifestCallCount)
	})

	t.Run("CallsCreateBuildOnEstafetteService", func(t *testing.T) {

		config := &api.APIConfig{
			Integrations: &api.APIConfigIntegrations{
				Github: &api.GithubConfig{},
			},
		}
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

		service := NewService(config, githubapiClient, pubsubapiClient, estafetteService, api.NewGitEventTopic("test topic"))

		pushEvent := githubapi.PushEvent{
			Ref: "refs/heads/master",
		}

		// act
		err := service.CreateJobForGithubPush(context.Background(), pushEvent)

		assert.Nil(t, err)
		assert.Equal(t, 1, createBuildCallCount)
	})

	t.Run("PublishesGitTriggersOnTopic", func(t *testing.T) {

		config := &api.APIConfig{
			Integrations: &api.APIConfigIntegrations{
				Github: &api.GithubConfig{},
			},
		}
		githubapiClient := githubapi.MockClient{}
		pubsubapiClient := pubsubapi.MockClient{}
		estafetteService := estafette.MockService{}

		gitEventTopic := api.NewGitEventTopic("test topic")
		defer gitEventTopic.Close()
		subscriptionChannel := gitEventTopic.Subscribe("PublishesGitTriggersOnTopic")

		service := NewService(config, githubapiClient, pubsubapiClient, estafetteService, gitEventTopic)

		pushEvent := githubapi.PushEvent{
			Ref: "refs/heads/master",
		}

		// act
		_ = service.CreateJobForGithubPush(context.Background(), pushEvent)

		select {
		case message, ok := <-subscriptionChannel:
			assert.True(t, ok)
			assert.Equal(t, "master", message.Event.Branch)

		case <-time.After(10 * time.Second):
			assert.Fail(t, "subscription timed out after 10 seconds")
		}
	})

	t.Run("CallsSubscribeToPubsubTriggersOnPubsubAPIClient", func(t *testing.T) {

		config := &api.APIConfig{
			Integrations: &api.APIConfigIntegrations{
				Github: &api.GithubConfig{},
			},
		}
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

		service := NewService(config, githubapiClient, pubsubapiClient, estafetteService, api.NewGitEventTopic("test topic"))

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

		config := &api.APIConfig{
			Integrations: &api.APIConfigIntegrations{
				Github: &api.GithubConfig{
					WhitelistedInstallations: []int{},
				},
			},
		}
		githubapiClient := githubapi.MockClient{}
		pubsubapiClient := pubsubapi.MockClient{}
		estafetteService := estafette.MockService{}
		service := NewService(config, githubapiClient, pubsubapiClient, estafetteService, api.NewGitEventTopic("test topic"))

		installation := githubapi.Installation{
			ID: 513,
		}

		// act
		isWhitelisted := service.IsWhitelistedInstallation(context.Background(), installation)

		assert.True(t, isWhitelisted)
	})

	t.Run("ReturnsFalseIfInstallationIDIsNotInWhitelistedInstallationsConfig", func(t *testing.T) {

		config := &api.APIConfig{
			Integrations: &api.APIConfigIntegrations{
				Github: &api.GithubConfig{
					WhitelistedInstallations: []int{
						236,
					},
				},
			},
		}
		githubapiClient := githubapi.MockClient{}
		pubsubapiClient := pubsubapi.MockClient{}
		estafetteService := estafette.MockService{}
		service := NewService(config, githubapiClient, pubsubapiClient, estafetteService, api.NewGitEventTopic("test topic"))

		installation := githubapi.Installation{
			ID: 513,
		}

		// act
		isWhitelisted := service.IsWhitelistedInstallation(context.Background(), installation)

		assert.False(t, isWhitelisted)
	})

	t.Run("ReturnsTrueIfInstallationIDIsInWhitelistedInstallationsConfig", func(t *testing.T) {

		config := &api.APIConfig{
			Integrations: &api.APIConfigIntegrations{
				Github: &api.GithubConfig{
					WhitelistedInstallations: []int{
						236,
						513,
					},
				},
			},
		}
		githubapiClient := githubapi.MockClient{}
		pubsubapiClient := pubsubapi.MockClient{}
		estafetteService := estafette.MockService{}
		service := NewService(config, githubapiClient, pubsubapiClient, estafetteService, api.NewGitEventTopic("test topic"))

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

		config := &api.APIConfig{
			Integrations: &api.APIConfigIntegrations{
				Github: &api.GithubConfig{
					WhitelistedInstallations: []int{},
				},
			},
		}

		githubapiClient := githubapi.MockClient{}
		pubsubapiClient := pubsubapi.MockClient{}
		estafetteService := estafette.MockService{}

		renameCallCount := 0
		estafetteService.RenameFunc = func(ctx context.Context, fromRepoSource, fromRepoOwner, fromRepoName, toRepoSource, toRepoOwner, toRepoName string) (err error) {
			renameCallCount++
			return
		}

		service := NewService(config, githubapiClient, pubsubapiClient, estafetteService, api.NewGitEventTopic("test topic"))

		// act
		err := service.Rename(context.Background(), "github.com", "estafette", "estafette-ci-contracts", "github.com", "estafette", "estafette-ci-protos")

		assert.Nil(t, err)
		assert.Equal(t, 1, renameCallCount)
	})
}

func TestHasValidSignature(t *testing.T) {

	t.Run("ReturnsFalseIfSignatureDoesNotMatchExpectedSignature", func(t *testing.T) {

		config := &api.APIConfig{
			Integrations: &api.APIConfigIntegrations{
				Github: &api.GithubConfig{
					WebhookSecret: "m1gw5wmje424dmfvpb72ny6vjnubw79jvi7dlw2h",
				},
			},
		}

		githubapiClient := githubapi.MockClient{}
		pubsubapiClient := pubsubapi.MockClient{}
		estafetteService := estafette.MockService{}

		service := NewService(config, githubapiClient, pubsubapiClient, estafetteService, api.NewGitEventTopic("test topic"))

		body := []byte(`{"action": "opened","issue": {"url": "https://api.github.com/repos/octocat/Hello-World/issues/1347","number": 1347,...},"repository" : {"id": 1296269,"full_name": "octocat/Hello-World","owner": {"login": "octocat","id": 1,...},...},"sender": {"login": "octocat","id": 1,...}}`)
		signatureHeader := "sha1=7d38cdd689735b008b3c702edd92eea23791c5f6"

		// act
		validSignature, err := service.HasValidSignature(context.Background(), body, signatureHeader)

		assert.Nil(t, err)
		assert.False(t, validSignature)
	})

	t.Run("ReturnTrueIfSignatureMatchesExpectedSignature", func(t *testing.T) {

		config := &api.APIConfig{
			Integrations: &api.APIConfigIntegrations{
				Github: &api.GithubConfig{
					WebhookSecret: "m1gw5wmje424dmfvpb72ny6vjnubw79jvi7dlw2h",
				},
			},
		}

		githubapiClient := githubapi.MockClient{}
		pubsubapiClient := pubsubapi.MockClient{}
		estafetteService := estafette.MockService{}

		service := NewService(config, githubapiClient, pubsubapiClient, estafetteService, api.NewGitEventTopic("test topic"))

		body := []byte(`{"action": "opened","issue": {"url": "https://api.github.com/repos/octocat/Hello-World/issues/1347","number": 1347,...},"repository" : {"id": 1296269,"full_name": "octocat/Hello-World","owner": {"login": "octocat","id": 1,...},...},"sender": {"login": "octocat","id": 1,...}}`)
		signatureHeader := "sha1=765539562e575982123574d8325a636e16e0efba"

		// act
		validSignature, err := service.HasValidSignature(context.Background(), body, signatureHeader)

		assert.Nil(t, err)
		assert.True(t, validSignature)
	})
}
