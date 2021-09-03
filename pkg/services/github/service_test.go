package github

import (
	"context"
	"errors"
	"sync"
	"testing"

	"github.com/estafette/estafette-ci-api/pkg/api"
	"github.com/estafette/estafette-ci-api/pkg/clients/githubapi"
	"github.com/estafette/estafette-ci-api/pkg/clients/pubsubapi"
	"github.com/estafette/estafette-ci-api/pkg/services/estafette"
	"github.com/estafette/estafette-ci-api/pkg/services/queue"
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
			GetAppAndInstallationByID(gomock.Any(), gomock.Any()).
			Return(&githubapi.GithubApp{}, &githubapi.GithubInstallation{}, nil).
			Times(1)

		githubapiClient.
			EXPECT().
			GetInstallationToken(gomock.Any(), gomock.Any(), gomock.Any()).
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

		githubapiClient.EXPECT().GetAppAndInstallationByID(gomock.Any(), gomock.Any()).Return(&githubapi.GithubApp{}, &githubapi.GithubInstallation{}, nil).AnyTimes()
		githubapiClient.EXPECT().GetInstallationToken(gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes()
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

		githubapiClient.EXPECT().GetAppAndInstallationByID(gomock.Any(), gomock.Any()).Return(&githubapi.GithubApp{}, &githubapi.GithubInstallation{}, nil).AnyTimes()
		githubapiClient.EXPECT().GetInstallationToken(gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes()
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

		githubapiClient.EXPECT().GetAppAndInstallationByID(gomock.Any(), gomock.Any()).Return(&githubapi.GithubApp{}, &githubapi.GithubInstallation{}, nil).AnyTimes()
		githubapiClient.EXPECT().GetInstallationToken(gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes()
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

		githubapiClient.EXPECT().GetAppAndInstallationByID(gomock.Any(), gomock.Any()).Return(&githubapi.GithubApp{}, &githubapi.GithubInstallation{}, nil).AnyTimes()
		githubapiClient.EXPECT().GetInstallationToken(gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes()
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

	t.Run("ReturnsTrueIfInstallationIsKnown", func(t *testing.T) {

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

		githubapiClient.EXPECT().GetAppAndInstallationByID(gomock.Any(), 513).Return(&githubapi.GithubApp{}, &githubapi.GithubInstallation{}, nil)

		service := NewService(config, githubapiClient, pubsubapiClient, estafetteService, queueService)

		installationID := 513

		// act
		isAllowed, _ := service.IsAllowedInstallation(context.Background(), installationID)

		assert.True(t, isAllowed)
	})

	t.Run("ReturnsFalseIfInstallationIDIsUnknown", func(t *testing.T) {

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

		githubapiClient.EXPECT().GetAppAndInstallationByID(gomock.Any(), 513).Return(nil, nil, githubapi.ErrMissingInstallation)

		service := NewService(config, githubapiClient, pubsubapiClient, estafetteService, queueService)

		installationID := 513

		// act
		isAllowed, _ := service.IsAllowedInstallation(context.Background(), installationID)

		assert.False(t, isAllowed)
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
				Github: &api.GithubConfig{},
			},
		}

		githubapiClient := githubapi.NewMockClient(ctrl)
		pubsubapiClient := pubsubapi.NewMockClient(ctrl)
		estafetteService := estafette.NewMockService(ctrl)
		queueService := queue.NewMockService(ctrl)

		githubapiClient.
			EXPECT().
			GetAppByID(gomock.Any(), 15).
			Return(&githubapi.GithubApp{
				ID:            15,
				WebhookSecret: "m1gw5wmje424dmfvpb72ny6vjnubw79jvi7dlw2h",
			}, nil)

		service := NewService(config, githubapiClient, pubsubapiClient, estafetteService, queueService)

		body := []byte(`{"action": "opened","issue": {"url": "https://api.github.com/repos/octocat/Hello-World/issues/1347","number": 1347,...},"repository" : {"id": 1296269,"full_name": "octocat/Hello-World","owner": {"login": "octocat","id": 1,...},...},"sender": {"login": "octocat","id": 1,...}}`)
		signatureHeader := "sha1=7d38cdd689735b008b3c702edd92eea23791c5f6"

		// act
		validSignature, err := service.HasValidSignature(context.Background(), body, "15", signatureHeader)

		assert.Nil(t, err)
		assert.False(t, validSignature)
	})

	t.Run("ReturnTrueIfSignatureMatchesExpectedSignature", func(t *testing.T) {

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
			GetAppByID(gomock.Any(), 15).
			Return(&githubapi.GithubApp{
				ID:            15,
				WebhookSecret: "m1gw5wmje424dmfvpb72ny6vjnubw79jvi7dlw2h",
			}, nil)

		service := NewService(config, githubapiClient, pubsubapiClient, estafetteService, queueService)

		body := []byte(`{"action": "opened","issue": {"url": "https://api.github.com/repos/octocat/Hello-World/issues/1347","number": 1347,...},"repository" : {"id": 1296269,"full_name": "octocat/Hello-World","owner": {"login": "octocat","id": 1,...},...},"sender": {"login": "octocat","id": 1,...}}`)
		signatureHeader := "sha1=765539562e575982123574d8325a636e16e0efba"

		// act
		validSignature, err := service.HasValidSignature(context.Background(), body, "15", signatureHeader)

		assert.Nil(t, err)
		assert.True(t, validSignature)
	})
}
