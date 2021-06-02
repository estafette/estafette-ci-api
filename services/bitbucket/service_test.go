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
		service := NewService(config, bitbucketapiClient, pubsubapiClient, estafetteService, api.NewEventTopic("test topic"))

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
		service := NewService(config, bitbucketapiClient, pubsubapiClient, estafetteService, api.NewEventTopic("test topic"))

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
		service := NewService(config, bitbucketapiClient, pubsubapiClient, estafetteService, api.NewEventTopic("test topic"))

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
		service := NewService(config, bitbucketapiClient, pubsubapiClient, estafetteService, api.NewEventTopic("test topic"))

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

		bitbucketapiClient.
			EXPECT().
			GetAccessToken(gomock.Any()).
			Times(1)

		bitbucketapiClient.EXPECT().GetEstafetteManifest(gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes()
		pubsubapiClient.EXPECT().SubscribeToPubsubTriggers(gomock.Any(), gomock.Any()).AnyTimes()

		service := NewService(config, bitbucketapiClient, pubsubapiClient, estafetteService, api.NewEventTopic("test topic"))

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

		bitbucketapiClient.
			EXPECT().
			GetEstafetteManifest(gomock.Any(), gomock.Any(), gomock.Any()).
			DoAndReturn(func(ctx context.Context, accesstoken bitbucketapi.AccessToken, event bitbucketapi.RepositoryPushEvent) (valid bool, manifest string, err error) {
				return true, "builder:\n  track: dev\n", nil
			}).Times(1)

		bitbucketapiClient.EXPECT().GetAccessToken(gomock.Any()).AnyTimes()
		estafetteService.EXPECT().CreateBuild(gomock.Any(), gomock.Any()).AnyTimes()
		pubsubapiClient.EXPECT().SubscribeToPubsubTriggers(gomock.Any(), gomock.Any()).AnyTimes()

		service := NewService(config, bitbucketapiClient, pubsubapiClient, estafetteService, api.NewEventTopic("test topic"))

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

		estafetteService.
			EXPECT().
			CreateBuild(gomock.Any(), gomock.Any()).
			Times(1)
		bitbucketapiClient.EXPECT().GetAccessToken(gomock.Any()).AnyTimes()
		bitbucketapiClient.EXPECT().GetEstafetteManifest(gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes()
		pubsubapiClient.EXPECT().SubscribeToPubsubTriggers(gomock.Any(), gomock.Any()).AnyTimes()

		service := NewService(config, bitbucketapiClient, pubsubapiClient, estafetteService, api.NewEventTopic("test topic"))

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

		gitEventTopic := api.NewEventTopic("test topic")
		defer gitEventTopic.Close()
		subscriptionChannel := gitEventTopic.Subscribe("PublishesGitTriggersOnTopic")

		bitbucketapiClient.EXPECT().GetAccessToken(gomock.Any()).AnyTimes()
		bitbucketapiClient.EXPECT().GetEstafetteManifest(gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes()
		pubsubapiClient.EXPECT().SubscribeToPubsubTriggers(gomock.Any(), gomock.Any()).AnyTimes()

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
			assert.Equal(t, "bitbucket.org/", message.Event.Git.Repository)

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
		estafetteService.EXPECT().CreateBuild(gomock.Any(), gomock.Any()).AnyTimes()

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

		service := NewService(config, bitbucketapiClient, pubsubapiClient, estafetteService, api.NewEventTopic("test topic"))

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
		service := NewService(config, bitbucketapiClient, pubsubapiClient, estafetteService, api.NewEventTopic("test topic"))

		repository := bitbucketapi.Repository{
			Owner: bitbucketapi.Owner{
				UserName: "anyone",
			},
		}

		// act
		isAllowed, _ := service.IsAllowedOwner(&repository)

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
		service := NewService(config, bitbucketapiClient, pubsubapiClient, estafetteService, api.NewEventTopic("test topic"))

		repository := bitbucketapi.Repository{
			Owner: bitbucketapi.Owner{
				UserName: "estafette-in-bitbucket",
			},
		}

		// act
		isAllowed, _ := service.IsAllowedOwner(&repository)

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
		service := NewService(config, bitbucketapiClient, pubsubapiClient, estafetteService, api.NewEventTopic("test topic"))

		repository := bitbucketapi.Repository{
			Owner: bitbucketapi.Owner{
				UserName: "estafette-in-bitbucket",
			},
		}

		// act
		isAllowed, _ := service.IsAllowedOwner(&repository)

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

		estafetteService.
			EXPECT().
			Rename(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
			Times(1)

		service := NewService(config, bitbucketapiClient, pubsubapiClient, estafetteService, api.NewEventTopic("test topic"))

		// act
		err := service.Rename(context.Background(), "bitbucket.org", "estafette", "estafette-ci-contracts", "bitbucket.org", "estafette", "estafette-ci-protos")

		assert.Nil(t, err)
	})
}
