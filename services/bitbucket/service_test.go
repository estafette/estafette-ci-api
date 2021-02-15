package bitbucket

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/estafette/estafette-ci-api/api"
	"github.com/estafette/estafette-ci-api/clients/bitbucketapi"
	"github.com/estafette/estafette-ci-api/clients/pubsubapi"
	"github.com/estafette/estafette-ci-api/services/estafette"
	contracts "github.com/estafette/estafette-ci-contracts"
	gomock "github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
)

func TestCreateJobForBitbucketPush(t *testing.T) {

	t.Run("ReturnsErrNonCloneableEventIfPushEventHasNoChanges", func(t *testing.T) {

		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		config := &api.APIConfig{
			Integrations: &api.APIConfigIntegrations{
				Bitbucket: &api.BitbucketConfig{},
			},
		}
		bitbucketapiClient := bitbucketapi.NewMockClient(ctrl)
		pubsubapiClient := pubsubapi.NewMockClient(ctrl)
		estafetteService := estafette.NewMockService(ctrl)
		service := NewService(config, bitbucketapiClient, pubsubapiClient, estafetteService, api.NewGitEventTopic("test topic"))

		pushEvent := bitbucketapi.RepositoryPushEvent{}

		// act
		err := service.CreateJobForBitbucketPush(context.Background(), pushEvent)

		assert.NotNil(t, err)
		assert.True(t, errors.Is(err, ErrNonCloneableEvent))
	})

	t.Run("ReturnsErrNonCloneableEventIfPushEventChangeHasNoNewObject", func(t *testing.T) {

		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		config := &api.APIConfig{
			Integrations: &api.APIConfigIntegrations{
				Bitbucket: &api.BitbucketConfig{},
			},
		}
		bitbucketapiClient := bitbucketapi.NewMockClient(ctrl)
		pubsubapiClient := pubsubapi.NewMockClient(ctrl)
		estafetteService := estafette.NewMockService(ctrl)
		service := NewService(config, bitbucketapiClient, pubsubapiClient, estafetteService, api.NewGitEventTopic("test topic"))

		pushEvent := bitbucketapi.RepositoryPushEvent{
			Push: bitbucketapi.PushEvent{
				Changes: []bitbucketapi.PushEventChange{
					bitbucketapi.PushEventChange{
						New: nil,
					},
				},
			},
		}

		// act
		err := service.CreateJobForBitbucketPush(context.Background(), pushEvent)

		assert.NotNil(t, err)
		assert.True(t, errors.Is(err, ErrNonCloneableEvent))
	})

	t.Run("ReturnsErrNonCloneableEventIfPushEventNewTypeDoesNotEqualBranch", func(t *testing.T) {

		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		config := &api.APIConfig{
			Integrations: &api.APIConfigIntegrations{
				Bitbucket: &api.BitbucketConfig{},
			},
		}
		bitbucketapiClient := bitbucketapi.NewMockClient(ctrl)
		pubsubapiClient := pubsubapi.NewMockClient(ctrl)
		estafetteService := estafette.NewMockService(ctrl)
		service := NewService(config, bitbucketapiClient, pubsubapiClient, estafetteService, api.NewGitEventTopic("test topic"))

		pushEvent := bitbucketapi.RepositoryPushEvent{
			Push: bitbucketapi.PushEvent{
				Changes: []bitbucketapi.PushEventChange{
					bitbucketapi.PushEventChange{
						New: &bitbucketapi.PushEventChangeObject{
							Type: "notbranch",
						},
					},
				},
			},
		}

		// act
		err := service.CreateJobForBitbucketPush(context.Background(), pushEvent)

		assert.NotNil(t, err)
		assert.True(t, errors.Is(err, ErrNonCloneableEvent))
	})

	t.Run("ReturnsErrNonCloneableEventIfPushEventNewTargetHasNoHash", func(t *testing.T) {

		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		config := &api.APIConfig{
			Integrations: &api.APIConfigIntegrations{
				Bitbucket: &api.BitbucketConfig{},
			},
		}
		bitbucketapiClient := bitbucketapi.NewMockClient(ctrl)
		pubsubapiClient := pubsubapi.NewMockClient(ctrl)
		estafetteService := estafette.NewMockService(ctrl)
		service := NewService(config, bitbucketapiClient, pubsubapiClient, estafetteService, api.NewGitEventTopic("test topic"))

		pushEvent := bitbucketapi.RepositoryPushEvent{
			Push: bitbucketapi.PushEvent{
				Changes: []bitbucketapi.PushEventChange{
					bitbucketapi.PushEventChange{
						New: &bitbucketapi.PushEventChangeObject{
							Type: "branch",
							Target: bitbucketapi.PushEventChangeObjectTarget{
								Hash: "",
							},
						},
					},
				},
			},
		}

		// act
		err := service.CreateJobForBitbucketPush(context.Background(), pushEvent)

		assert.NotNil(t, err)
		assert.True(t, errors.Is(err, ErrNonCloneableEvent))
	})

	t.Run("CallsGetAccessTokenOnBitbucketAPIClient", func(t *testing.T) {

		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		config := &api.APIConfig{
			Integrations: &api.APIConfigIntegrations{
				Bitbucket: &api.BitbucketConfig{},
			},
		}
		bitbucketapiClient := bitbucketapi.NewMockClient(ctrl)
		pubsubapiClient := pubsubapi.NewMockClient(ctrl)
		estafetteService := estafette.NewMockService(ctrl)

		getAccessTokenCallCount := 0
		bitbucketapiClient.
			EXPECT().
			GetAccessToken(gomock.Any()).
			DoAndReturn(func(ctx context.Context) (accesstoken bitbucketapi.AccessToken, err error) {
				getAccessTokenCallCount++
				return
			})
		bitbucketapiClient.EXPECT().GetEstafetteManifest(gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes()

		service := NewService(config, bitbucketapiClient, pubsubapiClient, estafetteService, api.NewGitEventTopic("test topic"))

		pushEvent := bitbucketapi.RepositoryPushEvent{
			Push: bitbucketapi.PushEvent{
				Changes: []bitbucketapi.PushEventChange{
					bitbucketapi.PushEventChange{
						New: &bitbucketapi.PushEventChangeObject{
							Type: "branch",
							Target: bitbucketapi.PushEventChangeObjectTarget{
								Hash: "f0677f01cc6d54a5b042224a9eb374e98f979985",
							},
						},
					},
				},
			},
		}

		// act
		_ = service.CreateJobForBitbucketPush(context.Background(), pushEvent)

		assert.Equal(t, 1, getAccessTokenCallCount)
	})

	t.Run("CallsGetEstafetteManifestOnBitbucketAPIClient", func(t *testing.T) {

		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		config := &api.APIConfig{
			Integrations: &api.APIConfigIntegrations{
				Bitbucket: &api.BitbucketConfig{},
			},
		}
		bitbucketapiClient := bitbucketapi.NewMockClient(ctrl)
		pubsubapiClient := pubsubapi.NewMockClient(ctrl)
		estafetteService := estafette.NewMockService(ctrl)

		getEstafetteManifestCallCount := 0
		bitbucketapiClient.
			EXPECT().
			GetEstafetteManifest(gomock.Any(), gomock.Any(), gomock.Any()).
			DoAndReturn(func(ctx context.Context, accesstoken bitbucketapi.AccessToken, event bitbucketapi.RepositoryPushEvent) (valid bool, manifest string, err error) {
				getEstafetteManifestCallCount++
				return true, "builder:\n  track: dev\n", nil
			})
		bitbucketapiClient.EXPECT().GetAccessToken(gomock.Any()).AnyTimes()
		estafetteService.EXPECT().CreateBuild(gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes()
		pubsubapiClient.EXPECT().SubscribeToPubsubTriggers(gomock.Any(), gomock.Any()).AnyTimes()

		service := NewService(config, bitbucketapiClient, pubsubapiClient, estafetteService, api.NewGitEventTopic("test topic"))

		pushEvent := bitbucketapi.RepositoryPushEvent{
			Push: bitbucketapi.PushEvent{
				Changes: []bitbucketapi.PushEventChange{
					bitbucketapi.PushEventChange{
						New: &bitbucketapi.PushEventChangeObject{
							Type: "branch",
							Target: bitbucketapi.PushEventChangeObjectTarget{
								Hash: "f0677f01cc6d54a5b042224a9eb374e98f979985",
							},
						},
					},
				},
			},
			Repository: bitbucketapi.Repository{
				FullName: "estafette/estafette-in-bitbucket",
			},
		}

		// act
		err := service.CreateJobForBitbucketPush(context.Background(), pushEvent)

		assert.Nil(t, err)
		assert.Equal(t, 1, getEstafetteManifestCallCount)
	})

	t.Run("CallsCreateBuildOnEstafetteService", func(t *testing.T) {

		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		config := &api.APIConfig{
			Integrations: &api.APIConfigIntegrations{
				Bitbucket: &api.BitbucketConfig{},
			},
		}
		bitbucketapiClient := bitbucketapi.NewMockClient(ctrl)
		pubsubapiClient := pubsubapi.NewMockClient(ctrl)
		estafetteService := estafette.NewMockService(ctrl)

		bitbucketapiClient.
			EXPECT().
			GetEstafetteManifest(gomock.Any(), gomock.Any(), gomock.Any()).
			DoAndReturn(func(ctx context.Context, accesstoken bitbucketapi.AccessToken, event bitbucketapi.RepositoryPushEvent) (valid bool, manifest string, err error) {
				return true, "builder:\n  track: dev\n", nil
			})

		createBuildCallCount := 0
		estafetteService.
			EXPECT().
			CreateBuild(gomock.Any(), gomock.Any(), gomock.Any()).
			DoAndReturn(func(ctx context.Context, build contracts.Build, waitForJobToStart bool) (b *contracts.Build, err error) {
				createBuildCallCount++
				return
			})
		bitbucketapiClient.EXPECT().GetAccessToken(gomock.Any()).AnyTimes()
		bitbucketapiClient.EXPECT().GetEstafetteManifest(gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes()
		pubsubapiClient.EXPECT().SubscribeToPubsubTriggers(gomock.Any(), gomock.Any()).AnyTimes()

		service := NewService(config, bitbucketapiClient, pubsubapiClient, estafetteService, api.NewGitEventTopic("test topic"))

		pushEvent := bitbucketapi.RepositoryPushEvent{
			Push: bitbucketapi.PushEvent{
				Changes: []bitbucketapi.PushEventChange{
					bitbucketapi.PushEventChange{
						New: &bitbucketapi.PushEventChangeObject{
							Type: "branch",
							Target: bitbucketapi.PushEventChangeObjectTarget{
								Hash: "f0677f01cc6d54a5b042224a9eb374e98f979985",
							},
						},
					},
				},
			},
			Repository: bitbucketapi.Repository{
				FullName: "estafette/estafette-in-bitbucket",
			},
		}

		// act
		err := service.CreateJobForBitbucketPush(context.Background(), pushEvent)

		assert.Nil(t, err)
		assert.Equal(t, 1, createBuildCallCount)
	})

	t.Run("PublishesGitTriggersOnTopic", func(t *testing.T) {

		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		config := &api.APIConfig{
			Integrations: &api.APIConfigIntegrations{
				Bitbucket: &api.BitbucketConfig{},
			},
		}
		bitbucketapiClient := bitbucketapi.NewMockClient(ctrl)
		pubsubapiClient := pubsubapi.NewMockClient(ctrl)
		estafetteService := estafette.NewMockService(ctrl)

		gitEventTopic := api.NewGitEventTopic("test topic")
		defer gitEventTopic.Close()
		subscriptionChannel := gitEventTopic.Subscribe("PublishesGitTriggersOnTopic")

		bitbucketapiClient.EXPECT().GetAccessToken(gomock.Any()).AnyTimes()
		bitbucketapiClient.EXPECT().GetEstafetteManifest(gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes()

		service := NewService(config, bitbucketapiClient, pubsubapiClient, estafetteService, gitEventTopic)

		pushEvent := bitbucketapi.RepositoryPushEvent{
			Push: bitbucketapi.PushEvent{
				Changes: []bitbucketapi.PushEventChange{
					bitbucketapi.PushEventChange{
						New: &bitbucketapi.PushEventChangeObject{
							Type: "branch",
							Target: bitbucketapi.PushEventChangeObjectTarget{
								Hash: "f0677f01cc6d54a5b042224a9eb374e98f979985",
							},
						},
					},
				},
			},
		}

		// act
		_ = service.CreateJobForBitbucketPush(context.Background(), pushEvent)

		select {
		case message, ok := <-subscriptionChannel:
			assert.True(t, ok)
			assert.Equal(t, "bitbucket.org/", message.Event.Repository)

		case <-time.After(10 * time.Second):
			assert.Fail(t, "subscription timed out after 10 seconds")
		}
	})

	t.Run("CallsSubscribeToPubsubTriggersOnPubsubAPIClient", func(t *testing.T) {

		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		config := &api.APIConfig{
			Integrations: &api.APIConfigIntegrations{
				Bitbucket: &api.BitbucketConfig{},
			},
		}
		bitbucketapiClient := bitbucketapi.NewMockClient(ctrl)
		pubsubapiClient := pubsubapi.NewMockClient(ctrl)
		estafetteService := estafette.NewMockService(ctrl)

		bitbucketapiClient.
			EXPECT().
			GetEstafetteManifest(gomock.Any(), gomock.Any(), gomock.Any()).
			DoAndReturn(func(ctx context.Context, accesstoken bitbucketapi.AccessToken, event bitbucketapi.RepositoryPushEvent) (valid bool, manifest string, err error) {
				return true, "builder:\n  track: dev\n", nil
			})
		bitbucketapiClient.EXPECT().GetAccessToken(gomock.Any()).AnyTimes()
		pubsubapiClient.EXPECT().SubscribeToPubsubTriggers(gomock.Any(), gomock.Any()).AnyTimes()
		estafetteService.EXPECT().CreateBuild(gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes()

		var wg sync.WaitGroup
		wg.Add(1)
		subscribeToPubsubTriggersCallCount := 0
		pubsubapiClient.
			EXPECT().
			SubscribeToPubsubTriggers(gomock.Any(), gomock.Any()).
			DoAndReturn(func(ctx context.Context, fromRepoSource, fromRepoOwner, fromRepoName, toRepoSource, toRepoOwner, toRepoName string) (err error) {
				subscribeToPubsubTriggersCallCount++
				wg.Done()
				return
			})

		service := NewService(config, bitbucketapiClient, pubsubapiClient, estafetteService, api.NewGitEventTopic("test topic"))

		pushEvent := bitbucketapi.RepositoryPushEvent{
			Push: bitbucketapi.PushEvent{
				Changes: []bitbucketapi.PushEventChange{
					bitbucketapi.PushEventChange{
						New: &bitbucketapi.PushEventChangeObject{
							Type: "branch",
							Target: bitbucketapi.PushEventChangeObjectTarget{
								Hash: "f0677f01cc6d54a5b042224a9eb374e98f979985",
							},
						},
					},
				},
			},
			Repository: bitbucketapi.Repository{
				FullName: "estafette/estafette-in-bitbucket",
			},
		}

		// act
		err := service.CreateJobForBitbucketPush(context.Background(), pushEvent)

		wg.Wait()

		assert.Nil(t, err)
		assert.Equal(t, 1, subscribeToPubsubTriggersCallCount)
	})
}

func TestIsAllowedOwner(t *testing.T) {

	t.Run("ReturnsTrueIfAllowedOwnersConfigIsEmpty", func(t *testing.T) {

		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		config := &api.APIConfig{
			Integrations: &api.APIConfigIntegrations{
				Bitbucket: &api.BitbucketConfig{
					OwnerOrganizations: []api.OwnerOrganizations{},
				},
			},
		}
		bitbucketapiClient := bitbucketapi.NewMockClient(ctrl)
		pubsubapiClient := pubsubapi.NewMockClient(ctrl)
		estafetteService := estafette.NewMockService(ctrl)
		service := NewService(config, bitbucketapiClient, pubsubapiClient, estafetteService, api.NewGitEventTopic("test topic"))

		repository := bitbucketapi.Repository{
			Owner: bitbucketapi.Owner{
				UserName: "anyone",
			},
		}

		// act
		isAllowed, _ := service.IsAllowedOwner(repository)

		assert.True(t, isAllowed)
	})

	t.Run("ReturnsFalseIfOwnerUsernameIsNotInAllowedOwnersConfig", func(t *testing.T) {

		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		config := &api.APIConfig{
			Integrations: &api.APIConfigIntegrations{
				Bitbucket: &api.BitbucketConfig{
					OwnerOrganizations: []api.OwnerOrganizations{
						{
							Owner: "someone-else",
						},
					},
				},
			},
		}
		bitbucketapiClient := bitbucketapi.NewMockClient(ctrl)
		pubsubapiClient := pubsubapi.NewMockClient(ctrl)
		estafetteService := estafette.NewMockService(ctrl)
		service := NewService(config, bitbucketapiClient, pubsubapiClient, estafetteService, api.NewGitEventTopic("test topic"))

		repository := bitbucketapi.Repository{
			Owner: bitbucketapi.Owner{
				UserName: "estafette-in-bitbucket",
			},
		}

		// act
		isAllowed, _ := service.IsAllowedOwner(repository)

		assert.False(t, isAllowed)
	})

	t.Run("ReturnsTrueIfOwnerUsernameIsInAllowedOwnersConfig", func(t *testing.T) {

		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		config := &api.APIConfig{
			Integrations: &api.APIConfigIntegrations{
				Bitbucket: &api.BitbucketConfig{
					OwnerOrganizations: []api.OwnerOrganizations{
						{
							Owner: "someone-else",
						},
						{
							Owner: "estafette-in-bitbucket",
						},
					},
				},
			},
		}
		bitbucketapiClient := bitbucketapi.NewMockClient(ctrl)
		pubsubapiClient := pubsubapi.NewMockClient(ctrl)
		estafetteService := estafette.NewMockService(ctrl)
		service := NewService(config, bitbucketapiClient, pubsubapiClient, estafetteService, api.NewGitEventTopic("test topic"))

		repository := bitbucketapi.Repository{
			Owner: bitbucketapi.Owner{
				UserName: "estafette-in-bitbucket",
			},
		}

		// act
		isAllowed, _ := service.IsAllowedOwner(repository)

		assert.True(t, isAllowed)
	})
}

func TestRename(t *testing.T) {

	t.Run("CallsRenameOnEstafetteService", func(t *testing.T) {

		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		config := &api.APIConfig{
			Integrations: &api.APIConfigIntegrations{
				Bitbucket: &api.BitbucketConfig{},
			},
		}
		bitbucketapiClient := bitbucketapi.NewMockClient(ctrl)
		pubsubapiClient := pubsubapi.NewMockClient(ctrl)
		estafetteService := estafette.NewMockService(ctrl)

		renameCallCount := 0

		estafetteService.
			EXPECT().
			Rename(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
			DoAndReturn(func(ctx context.Context, fromRepoSource, fromRepoOwner, fromRepoName, toRepoSource, toRepoOwner, toRepoName string) (err error) {
				renameCallCount++
				return
			})

		service := NewService(config, bitbucketapiClient, pubsubapiClient, estafetteService, api.NewGitEventTopic("test topic"))

		// act
		err := service.Rename(context.Background(), "bitbucket.org", "estafette", "estafette-ci-contracts", "bitbucket.org", "estafette", "estafette-ci-protos")

		assert.Nil(t, err)
		assert.Equal(t, 1, renameCallCount)
	})
}
