package github

import (
	"context"
	"errors"
	"sync"
	"testing"

	"github.com/estafette/estafette-ci-api/api"
	"github.com/estafette/estafette-ci-api/clients/githubapi"
	"github.com/estafette/estafette-ci-api/clients/pubsubapi"
	"github.com/estafette/estafette-ci-api/services/estafette"
	"github.com/estafette/estafette-ci-api/services/queue"
	manifest "github.com/estafette/estafette-ci-manifest"
	gomock "github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
)

func TestCreateJobForGithubPush(t *testing.T) {

	t.Run("ReturnsErrNonCloneableEventIfPushEventHasNoRefsHeadsPrefix", func(t *testing.T) {

		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		config := &api.APIConfig{
			Integrations: &api.APIConfigIntegrations{
				Github: &api.GithubConfig{},
			},
		}
		githubapiClient := githubapi.NewMockClient(ctrl)
		pubsubapiClient := pubsubapi.NewMockClient(ctrl)
		estafetteService := estafette.NewMockService(ctrl)
		queueService := queue.NewMockService(ctrl)
		service := NewService(config, githubapiClient, pubsubapiClient, estafetteService, queueService)

		pushEvent := githubapi.PushEvent{
			Ref: "refs/noheads",
		}

		// act
		err := service.CreateJobForGithubPush(context.Background(), pushEvent)

		assert.NotNil(t, err)
		assert.True(t, errors.Is(err, ErrNonCloneableEvent))
	})

	t.Run("CallsGetInstallationTokenOnGithubapiClient", func(t *testing.T) {

		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		config := &api.APIConfig{
			Integrations: &api.APIConfigIntegrations{
				Github: &api.GithubConfig{},
			},
		}
		githubapiClient := githubapi.NewMockClient(ctrl)
		pubsubapiClient := pubsubapi.NewMockClient(ctrl)
		estafetteService := estafette.NewMockService(ctrl)
		queueService := queue.NewMockService(ctrl)

		githubapiClient.
			EXPECT().
			GetInstallationToken(gomock.Any(), gomock.Any()).
			Times(1)

		githubapiClient.EXPECT().GetEstafetteManifest(gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes()
		queueService.EXPECT().PublishGitEvent(gomock.Any(), gomock.Eq(manifest.EstafetteGitEvent{Event: "push", Repository: "github.com/", Branch: "master"})).AnyTimes()

		service := NewService(config, githubapiClient, pubsubapiClient, estafetteService, queueService)

		pushEvent := githubapi.PushEvent{
			Ref: "refs/heads/master",
		}

		// act
		_ = service.CreateJobForGithubPush(context.Background(), pushEvent)
	})

	t.Run("CallsGetEstafetteManifestOnGithubapiClient", func(t *testing.T) {

		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		config := &api.APIConfig{
			Integrations: &api.APIConfigIntegrations{
				Github: &api.GithubConfig{},
			},
		}
		githubapiClient := githubapi.NewMockClient(ctrl)
		pubsubapiClient := pubsubapi.NewMockClient(ctrl)
		estafetteService := estafette.NewMockService(ctrl)
		queueService := queue.NewMockService(ctrl)

		githubapiClient.
			EXPECT().
			GetEstafetteManifest(gomock.Any(), gomock.Any(), gomock.Any()).
			DoAndReturn(func(ctx context.Context, accesstoken githubapi.AccessToken, event githubapi.PushEvent) (valid bool, manifest string, err error) {
				return true, "builder:\n  track: dev\n", nil
			}).
			Times(1)

		githubapiClient.EXPECT().GetInstallationToken(gomock.Any(), gomock.Any()).AnyTimes()
		estafetteService.EXPECT().CreateBuild(gomock.Any(), gomock.Any()).AnyTimes()
		pubsubapiClient.EXPECT().SubscribeToPubsubTriggers(gomock.Any(), gomock.Any()).AnyTimes()
		queueService.EXPECT().PublishGitEvent(gomock.Any(), gomock.Eq(manifest.EstafetteGitEvent{Event: "push", Repository: "github.com/", Branch: "master"})).AnyTimes()

		service := NewService(config, githubapiClient, pubsubapiClient, estafetteService, queueService)

		pushEvent := githubapi.PushEvent{
			Ref: "refs/heads/master",
		}

		// act
		err := service.CreateJobForGithubPush(context.Background(), pushEvent)

		assert.Nil(t, err)
	})

	t.Run("CallsCreateBuildOnEstafetteService", func(t *testing.T) {

		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		config := &api.APIConfig{
			Integrations: &api.APIConfigIntegrations{
				Github: &api.GithubConfig{},
			},
		}
		githubapiClient := githubapi.NewMockClient(ctrl)
		pubsubapiClient := pubsubapi.NewMockClient(ctrl)
		estafetteService := estafette.NewMockService(ctrl)
		queueService := queue.NewMockService(ctrl)

		githubapiClient.
			EXPECT().
			GetEstafetteManifest(gomock.Any(), gomock.Any(), gomock.Any()).
			DoAndReturn(func(ctx context.Context, accesstoken githubapi.AccessToken, event githubapi.PushEvent) (valid bool, manifest string, err error) {
				return true, "builder:\n  track: dev\n", nil
			})

		estafetteService.
			EXPECT().
			CreateBuild(gomock.Any(), gomock.Any()).
			Times(1)

		githubapiClient.EXPECT().GetInstallationToken(gomock.Any(), gomock.Any()).AnyTimes()
		pubsubapiClient.EXPECT().SubscribeToPubsubTriggers(gomock.Any(), gomock.Any()).AnyTimes()
		queueService.EXPECT().PublishGitEvent(gomock.Any(), gomock.Eq(manifest.EstafetteGitEvent{Event: "push", Repository: "github.com/", Branch: "master"})).AnyTimes()

		service := NewService(config, githubapiClient, pubsubapiClient, estafetteService, queueService)

		pushEvent := githubapi.PushEvent{
			Ref: "refs/heads/master",
		}

		// act
		err := service.CreateJobForGithubPush(context.Background(), pushEvent)

		assert.Nil(t, err)
	})

	t.Run("PublishesGitTriggersOnTopic", func(t *testing.T) {

		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		config := &api.APIConfig{
			Integrations: &api.APIConfigIntegrations{
				Github: &api.GithubConfig{},
			},
		}
		githubapiClient := githubapi.NewMockClient(ctrl)
		pubsubapiClient := pubsubapi.NewMockClient(ctrl)
		estafetteService := estafette.NewMockService(ctrl)
		queueService := queue.NewMockService(ctrl)

		service := NewService(config, githubapiClient, pubsubapiClient, estafetteService, queueService)

		pushEvent := githubapi.PushEvent{
			Ref: "refs/heads/master",
		}

		githubapiClient.EXPECT().GetInstallationToken(gomock.Any(), gomock.Any()).AnyTimes()
		githubapiClient.EXPECT().GetEstafetteManifest(gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes()
		queueService.EXPECT().PublishGitEvent(gomock.Any(), gomock.Eq(manifest.EstafetteGitEvent{Event: "push", Repository: "github.com/", Branch: "master"})).AnyTimes()

		// act
		_ = service.CreateJobForGithubPush(context.Background(), pushEvent)
	})

	t.Run("CallsSubscribeToPubsubTriggersOnPubsubAPIClient", func(t *testing.T) {

		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		config := &api.APIConfig{
			Integrations: &api.APIConfigIntegrations{
				Github: &api.GithubConfig{},
			},
		}
		githubapiClient := githubapi.NewMockClient(ctrl)
		pubsubapiClient := pubsubapi.NewMockClient(ctrl)
		estafetteService := estafette.NewMockService(ctrl)
		queueService := queue.NewMockService(ctrl)

		githubapiClient.
			EXPECT().
			GetEstafetteManifest(gomock.Any(), gomock.Any(), gomock.Any()).
			DoAndReturn(func(ctx context.Context, accesstoken githubapi.AccessToken, event githubapi.PushEvent) (valid bool, manifest string, err error) {
				return true, "builder:\n  track: dev\n", nil
			})

		var wg sync.WaitGroup
		wg.Add(1)
		defer wg.Wait()
		pubsubapiClient.
			EXPECT().
			SubscribeToPubsubTriggers(gomock.Any(), gomock.Any()).
			DoAndReturn(func(ctx context.Context, manifestString string) (err error) {
				wg.Done()
				return
			}).
			Times(1)

		githubapiClient.EXPECT().GetInstallationToken(gomock.Any(), gomock.Any()).AnyTimes()
		githubapiClient.EXPECT().GetEstafetteManifest(gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes()
		estafetteService.EXPECT().CreateBuild(gomock.Any(), gomock.Any()).AnyTimes()
		queueService.EXPECT().PublishGitEvent(gomock.Any(), gomock.Eq(manifest.EstafetteGitEvent{Event: "push", Repository: "github.com/", Branch: "master"})).AnyTimes()

		service := NewService(config, githubapiClient, pubsubapiClient, estafetteService, queueService)

		pushEvent := githubapi.PushEvent{
			Ref: "refs/heads/master",
		}

		// act
		err := service.CreateJobForGithubPush(context.Background(), pushEvent)

		wg.Wait()

		assert.Nil(t, err)
	})
}

func TestIsAllowedInstallation(t *testing.T) {

	t.Run("ReturnsTrueIfAllowedInstallationsConfigIsEmpty", func(t *testing.T) {

		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		config := &api.APIConfig{
			Integrations: &api.APIConfigIntegrations{
				Github: &api.GithubConfig{
					InstallationOrganizations: []api.InstallationOrganizations{},
				},
			},
		}
		githubapiClient := githubapi.NewMockClient(ctrl)
		pubsubapiClient := pubsubapi.NewMockClient(ctrl)
		estafetteService := estafette.NewMockService(ctrl)
		queueService := queue.NewMockService(ctrl)

		service := NewService(config, githubapiClient, pubsubapiClient, estafetteService, queueService)

		installation := githubapi.Installation{
			ID: 513,
		}

		// act
		isAllowed, _ := service.IsAllowedInstallation(context.Background(), installation)

		assert.True(t, isAllowed)
	})

	t.Run("ReturnsFalseIfInstallationIDIsNotInAllowedInstallationsConfig", func(t *testing.T) {

		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		config := &api.APIConfig{
			Integrations: &api.APIConfigIntegrations{
				Github: &api.GithubConfig{
					InstallationOrganizations: []api.InstallationOrganizations{
						{
							Installation: 236,
						},
					},
				},
			},
		}
		githubapiClient := githubapi.NewMockClient(ctrl)
		pubsubapiClient := pubsubapi.NewMockClient(ctrl)
		estafetteService := estafette.NewMockService(ctrl)
		queueService := queue.NewMockService(ctrl)
		service := NewService(config, githubapiClient, pubsubapiClient, estafetteService, queueService)

		installation := githubapi.Installation{
			ID: 513,
		}

		// act
		isAllowed, _ := service.IsAllowedInstallation(context.Background(), installation)

		assert.False(t, isAllowed)
	})

	t.Run("ReturnsTrueIfInstallationIDIsInAllowedInstallationsConfig", func(t *testing.T) {

		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		config := &api.APIConfig{
			Integrations: &api.APIConfigIntegrations{
				Github: &api.GithubConfig{
					InstallationOrganizations: []api.InstallationOrganizations{
						{
							Installation: 236,
						},
						{
							Installation: 513,
						},
					},
				},
			},
		}
		githubapiClient := githubapi.NewMockClient(ctrl)
		pubsubapiClient := pubsubapi.NewMockClient(ctrl)
		estafetteService := estafette.NewMockService(ctrl)
		queueService := queue.NewMockService(ctrl)
		service := NewService(config, githubapiClient, pubsubapiClient, estafetteService, queueService)

		installation := githubapi.Installation{
			ID: 513,
		}

		// act
		isAllowed, _ := service.IsAllowedInstallation(context.Background(), installation)

		assert.True(t, isAllowed)
	})
}

func TestRename(t *testing.T) {

	t.Run("CallsRenameOnEstafetteService", func(t *testing.T) {

		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		config := &api.APIConfig{
			Integrations: &api.APIConfigIntegrations{
				Github: &api.GithubConfig{},
			},
		}

		githubapiClient := githubapi.NewMockClient(ctrl)
		pubsubapiClient := pubsubapi.NewMockClient(ctrl)
		estafetteService := estafette.NewMockService(ctrl)
		queueService := queue.NewMockService(ctrl)

		estafetteService.
			EXPECT().
			Rename(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
			Times(1)

		service := NewService(config, githubapiClient, pubsubapiClient, estafetteService, queueService)

		// act
		err := service.Rename(context.Background(), "github.com", "estafette", "estafette-ci-contracts", "github.com", "estafette", "estafette-ci-protos")

		assert.Nil(t, err)
	})
}

func TestHasValidSignature(t *testing.T) {

	t.Run("ReturnsFalseIfSignatureDoesNotMatchExpectedSignature", func(t *testing.T) {

		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		config := &api.APIConfig{
			Integrations: &api.APIConfigIntegrations{
				Github: &api.GithubConfig{
					WebhookSecret: "m1gw5wmje424dmfvpb72ny6vjnubw79jvi7dlw2h",
				},
			},
		}

		githubapiClient := githubapi.NewMockClient(ctrl)
		pubsubapiClient := pubsubapi.NewMockClient(ctrl)
		estafetteService := estafette.NewMockService(ctrl)
		queueService := queue.NewMockService(ctrl)

		service := NewService(config, githubapiClient, pubsubapiClient, estafetteService, queueService)

		body := []byte(`{"action": "opened","issue": {"url": "https://api.github.com/repos/octocat/Hello-World/issues/1347","number": 1347,...},"repository" : {"id": 1296269,"full_name": "octocat/Hello-World","owner": {"login": "octocat","id": 1,...},...},"sender": {"login": "octocat","id": 1,...}}`)
		signatureHeader := "sha1=7d38cdd689735b008b3c702edd92eea23791c5f6"

		// act
		validSignature, err := service.HasValidSignature(context.Background(), body, signatureHeader)

		assert.Nil(t, err)
		assert.False(t, validSignature)
	})

	t.Run("ReturnTrueIfSignatureMatchesExpectedSignature", func(t *testing.T) {

		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		config := &api.APIConfig{
			Integrations: &api.APIConfigIntegrations{
				Github: &api.GithubConfig{
					WebhookSecret: "m1gw5wmje424dmfvpb72ny6vjnubw79jvi7dlw2h",
				},
			},
		}

		githubapiClient := githubapi.NewMockClient(ctrl)
		pubsubapiClient := pubsubapi.NewMockClient(ctrl)
		estafetteService := estafette.NewMockService(ctrl)
		queueService := queue.NewMockService(ctrl)

		service := NewService(config, githubapiClient, pubsubapiClient, estafetteService, queueService)

		body := []byte(`{"action": "opened","issue": {"url": "https://api.github.com/repos/octocat/Hello-World/issues/1347","number": 1347,...},"repository" : {"id": 1296269,"full_name": "octocat/Hello-World","owner": {"login": "octocat","id": 1,...},...},"sender": {"login": "octocat","id": 1,...}}`)
		signatureHeader := "sha1=765539562e575982123574d8325a636e16e0efba"

		// act
		validSignature, err := service.HasValidSignature(context.Background(), body, signatureHeader)

		assert.Nil(t, err)
		assert.True(t, validSignature)
	})
}
