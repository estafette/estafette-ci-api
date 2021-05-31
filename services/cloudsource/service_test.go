package cloudsource

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/estafette/estafette-ci-api/api"
	"github.com/estafette/estafette-ci-api/clients/cloudsourceapi"
	"github.com/estafette/estafette-ci-api/clients/pubsubapi"
	"github.com/estafette/estafette-ci-api/services/estafette"
	contracts "github.com/estafette/estafette-ci-contracts"
	gomock "github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
)

func TestCreateJobForCloudSourcePush(t *testing.T) {

	t.Run("ReturnsErrNonCloneableEventIfNotificationHasNoRefUpdate", func(t *testing.T) {

		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		config := &api.APIConfig{
			Integrations: &api.APIConfigIntegrations{
				CloudSource: &api.CloudSourceConfig{},
			},
		}
		cloudsourceapiClient := cloudsourceapi.NewMockClient(ctrl)
		pubsubapiClient := pubsubapi.NewMockClient(ctrl)
		estafetteService := estafette.NewMockService(ctrl)
		service := NewService(config, cloudsourceapiClient, pubsubapiClient, estafetteService, api.NewEventTopic("test topic"))

		notification := cloudsourceapi.PubSubNotification{
			Name:      "test-pubsub",
			Url:       "https://sourcecode.google.com",
			EventTime: time.Now(),
		}

		// act
		err := service.CreateJobForCloudSourcePush(context.Background(), notification)

		assert.NotNil(t, err)
		assert.True(t, errors.Is(err, ErrNonCloneableEvent))
	})

	t.Run("CallsGetAccessTokenOnCloudSourceAPIClient", func(t *testing.T) {

		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		config := &api.APIConfig{
			Integrations: &api.APIConfigIntegrations{
				CloudSource: &api.CloudSourceConfig{},
			},
		}
		cloudsourceapiClient := cloudsourceapi.NewMockClient(ctrl)
		pubsubapiClient := pubsubapi.NewMockClient(ctrl)
		estafetteService := estafette.NewMockService(ctrl)

		cloudsourceapiClient.
			EXPECT().
			GetAccessToken(gomock.Any()).
			Times(1)

		cloudsourceapiClient.EXPECT().GetEstafetteManifest(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes()

		service := NewService(config, cloudsourceapiClient, pubsubapiClient, estafetteService, api.NewEventTopic("test topic"))

		notification := cloudsourceapi.PubSubNotification{
			Name:      "test-pubsub",
			Url:       "https://sourcecode.google.com",
			EventTime: time.Now(),
			RefUpdateEvent: &cloudsourceapi.RefUpdateEvent{
				Email: "estafette@estafette.com",
				RefUpdates: map[string]cloudsourceapi.RefUpdate{
					"refs/heads/master": cloudsourceapi.RefUpdate{
						RefName:    "refs/heads/master",
						UpdateType: "UPDATE_FAST_FORWARD",
						OldId:      "c7a28dd5de3403cc384a025",
						NewId:      "f00768887da8de620612102",
					},
				},
			},
		}

		// act
		_ = service.CreateJobForCloudSourcePush(context.Background(), notification)
	})

	t.Run("CallsGetEstafetteManifestOnCloudSourceAPIClient", func(t *testing.T) {

		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		config := &api.APIConfig{
			Integrations: &api.APIConfigIntegrations{
				CloudSource: &api.CloudSourceConfig{},
			},
		}
		cloudsourceapiClient := cloudsourceapi.NewMockClient(ctrl)
		pubsubapiClient := pubsubapi.NewMockClient(ctrl)
		estafetteService := estafette.NewMockService(ctrl)

		cloudsourceapiClient.
			EXPECT().
			GetEstafetteManifest(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
			DoAndReturn(func(ctx context.Context, accesstoken cloudsourceapi.AccessToken, event cloudsourceapi.PubSubNotification, gitClone func(string, string, string) error) (valid bool, manifest string, err error) {
				return true, "builder:\n  track: dev\n", nil
			}).
			Times(1)

		cloudsourceapiClient.EXPECT().GetAccessToken(gomock.Any()).AnyTimes()
		estafetteService.EXPECT().CreateBuild(gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes()
		pubsubapiClient.EXPECT().SubscribeToPubsubTriggers(gomock.Any(), gomock.Any()).AnyTimes()

		service := NewService(config, cloudsourceapiClient, pubsubapiClient, estafetteService, api.NewEventTopic("test topic"))

		notification := cloudsourceapi.PubSubNotification{
			Name:      "projects/test-project/repos/pubsub-test",
			Url:       "https://sourcecode.google.com",
			EventTime: time.Now(),
			RefUpdateEvent: &cloudsourceapi.RefUpdateEvent{
				Email: "estafette@estafette.com",
				RefUpdates: map[string]cloudsourceapi.RefUpdate{
					"refs/heads/master": cloudsourceapi.RefUpdate{
						RefName:    "refs/heads/master",
						UpdateType: "UPDATE_FAST_FORWARD",
						OldId:      "c7a28dd5de3403cc384a025",
						NewId:      "f00768887da8de620612102",
					},
				},
			},
		}

		// act
		err := service.CreateJobForCloudSourcePush(context.Background(), notification)

		assert.Nil(t, err)
	})

	t.Run("CallsCreateBuildOnEstafetteService", func(t *testing.T) {

		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		config := &api.APIConfig{
			Integrations: &api.APIConfigIntegrations{
				CloudSource: &api.CloudSourceConfig{},
			},
		}
		cloudsourceapiClient := cloudsourceapi.NewMockClient(ctrl)
		pubsubapiClient := pubsubapi.NewMockClient(ctrl)
		estafetteService := estafette.NewMockService(ctrl)

		cloudsourceapiClient.
			EXPECT().
			GetEstafetteManifest(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
			DoAndReturn(func(ctx context.Context, accesstoken cloudsourceapi.AccessToken, event cloudsourceapi.PubSubNotification, gitClone func(string, string, string) error) (valid bool, manifest string, err error) {
				return true, "builder:\n  track: dev\n", nil
			})

		estafetteService.
			EXPECT().
			CreateBuild(gomock.Any(), gomock.Any(), gomock.Any()).
			DoAndReturn(func(ctx context.Context, build contracts.Build, waitForJobToStart bool) (b *contracts.Build, err error) {
				return
			}).
			Times(1)

		cloudsourceapiClient.EXPECT().GetAccessToken(gomock.Any()).AnyTimes()
		pubsubapiClient.EXPECT().SubscribeToPubsubTriggers(gomock.Any(), gomock.Any()).AnyTimes()

		service := NewService(config, cloudsourceapiClient, pubsubapiClient, estafetteService, api.NewEventTopic("test topic"))

		notification := cloudsourceapi.PubSubNotification{
			Name:      "projects/test-project/repos/pubsub-test",
			Url:       "https://sourcecode.google.com",
			EventTime: time.Now(),
			RefUpdateEvent: &cloudsourceapi.RefUpdateEvent{
				Email: "estafette@estafette.com",
				RefUpdates: map[string]cloudsourceapi.RefUpdate{
					"refs/heads/master": cloudsourceapi.RefUpdate{
						RefName:    "refs/heads/master",
						UpdateType: "UPDATE_FAST_FORWARD",
						OldId:      "c7a28dd5de3403cc384a025",
						NewId:      "f00768887da8de620612102",
					},
				},
			},
		}

		// act
		err := service.CreateJobForCloudSourcePush(context.Background(), notification)

		assert.Nil(t, err)
	})

	t.Run("PublishesGitTriggersOnTopic", func(t *testing.T) {

		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		config := &api.APIConfig{
			Integrations: &api.APIConfigIntegrations{
				CloudSource: &api.CloudSourceConfig{},
			},
		}
		cloudsourceapiClient := cloudsourceapi.NewMockClient(ctrl)
		pubsubapiClient := pubsubapi.NewMockClient(ctrl)
		estafetteService := estafette.NewMockService(ctrl)
		gitEventTopic := api.NewEventTopic("test topic")
		defer gitEventTopic.Close()
		subscriptionChannel := gitEventTopic.Subscribe("PublishesGitTriggersOnTopic")

		service := NewService(config, cloudsourceapiClient, pubsubapiClient, estafetteService, gitEventTopic)

		notification := cloudsourceapi.PubSubNotification{
			Name:      "projects/test-project/repos/pubsub-test",
			Url:       "https://sourcecode.google.com",
			EventTime: time.Now(),
			RefUpdateEvent: &cloudsourceapi.RefUpdateEvent{
				Email: "estafette@estafette.com",
				RefUpdates: map[string]cloudsourceapi.RefUpdate{
					"refs/heads/master": cloudsourceapi.RefUpdate{
						RefName:    "refs/heads/master",
						UpdateType: "UPDATE_FAST_FORWARD",
						OldId:      "c7a28dd5de3403cc384a025",
						NewId:      "f00768887da8de620612102",
					},
				},
			},
		}

		cloudsourceapiClient.EXPECT().GetAccessToken(gomock.Any()).AnyTimes()
		cloudsourceapiClient.EXPECT().GetEstafetteManifest(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes()
		pubsubapiClient.EXPECT().SubscribeToPubsubTriggers(gomock.Any(), gomock.Any()).AnyTimes()

		// act
		_ = service.CreateJobForCloudSourcePush(context.Background(), notification)

		select {
		case message, ok := <-subscriptionChannel:
			assert.True(t, ok)
			assert.Equal(t, "master", message.Event.Git.Branch)

		case <-time.After(10 * time.Second):
			assert.Fail(t, "subscription timed out after 10 seconds")
		}
	})

	t.Run("CallsSubscribeToPubsubTriggersOnPubsubAPIClient", func(t *testing.T) {

		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		config := &api.APIConfig{
			Integrations: &api.APIConfigIntegrations{
				CloudSource: &api.CloudSourceConfig{},
			},
		}
		cloudsourceapiClient := cloudsourceapi.NewMockClient(ctrl)
		pubsubapiClient := pubsubapi.NewMockClient(ctrl)
		estafetteService := estafette.NewMockService(ctrl)

		cloudsourceapiClient.
			EXPECT().
			GetEstafetteManifest(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
			DoAndReturn(func(ctx context.Context, accesstoken cloudsourceapi.AccessToken, event cloudsourceapi.PubSubNotification, gitClone func(string, string, string) error) (valid bool, manifest string, err error) {
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

		cloudsourceapiClient.EXPECT().GetAccessToken(gomock.Any()).AnyTimes()
		cloudsourceapiClient.EXPECT().GetEstafetteManifest(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes()
		estafetteService.EXPECT().CreateBuild(gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes()

		service := NewService(config, cloudsourceapiClient, pubsubapiClient, estafetteService, api.NewEventTopic("test topic"))

		notification := cloudsourceapi.PubSubNotification{
			Name:      "projects/test-project/repos/pubsub-test",
			Url:       "https://sourcecode.google.com",
			EventTime: time.Now(),
			RefUpdateEvent: &cloudsourceapi.RefUpdateEvent{
				Email: "estafette@estafette.com",
				RefUpdates: map[string]cloudsourceapi.RefUpdate{
					"refs/heads/master": cloudsourceapi.RefUpdate{
						RefName:    "refs/heads/master",
						UpdateType: "UPDATE_FAST_FORWARD",
						OldId:      "c7a28dd5de3403cc384a025",
						NewId:      "f00768887da8de620612102",
					},
				},
			},
		}

		// act
		err := service.CreateJobForCloudSourcePush(context.Background(), notification)

		assert.Nil(t, err)
	})
}

func TestIsAllowedProject(t *testing.T) {

	t.Run("ReturnsTrueIfAllowedProjectsConfigIsEmpty", func(t *testing.T) {

		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		config := &api.APIConfig{
			Integrations: &api.APIConfigIntegrations{
				CloudSource: &api.CloudSourceConfig{
					ProjectOrganizations: []api.ProjectOrganizations{},
				},
			},
		}
		cloudsourceapiClient := cloudsourceapi.NewMockClient(ctrl)
		pubsubapiClient := pubsubapi.NewMockClient(ctrl)
		estafetteService := estafette.NewMockService(ctrl)
		service := NewService(config, cloudsourceapiClient, pubsubapiClient, estafetteService, api.NewEventTopic("test topic"))

		notification := cloudsourceapi.PubSubNotification{
			Name:      "projects/test-project/repos/pubsub-test",
			Url:       "https://sourcecode.google.com",
			EventTime: time.Now(),
			RefUpdateEvent: &cloudsourceapi.RefUpdateEvent{
				Email: "estafette@estafette.com",
				RefUpdates: map[string]cloudsourceapi.RefUpdate{
					"refs/heads/master": cloudsourceapi.RefUpdate{
						RefName:    "refs/heads/master",
						UpdateType: "UPDATE_FAST_FORWARD",
						OldId:      "c7a28dd5de3403cc384a025",
						NewId:      "f00768887da8de620612102",
					},
				},
			},
		}

		// act
		isAllowed, _ := service.IsAllowedProject(notification)

		assert.True(t, isAllowed)
	})

	t.Run("ReturnsFalseIfOwnerUsernameIsNotInAllowedProjectsConfig", func(t *testing.T) {

		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		config := &api.APIConfig{
			Integrations: &api.APIConfigIntegrations{
				CloudSource: &api.CloudSourceConfig{
					ProjectOrganizations: []api.ProjectOrganizations{
						{
							Project: "someone-else",
						},
					},
				},
			},
		}
		cloudsourceapiClient := cloudsourceapi.NewMockClient(ctrl)
		pubsubapiClient := pubsubapi.NewMockClient(ctrl)
		estafetteService := estafette.NewMockService(ctrl)
		service := NewService(config, cloudsourceapiClient, pubsubapiClient, estafetteService, api.NewEventTopic("test topic"))

		notification := cloudsourceapi.PubSubNotification{
			Name:      "projects/test-project/repos/pubsub-test",
			Url:       "https://sourcecode.google.com",
			EventTime: time.Now(),
			RefUpdateEvent: &cloudsourceapi.RefUpdateEvent{
				Email: "estafette@estafette.com",
				RefUpdates: map[string]cloudsourceapi.RefUpdate{
					"refs/heads/master": cloudsourceapi.RefUpdate{
						RefName:    "refs/heads/master",
						UpdateType: "UPDATE_FAST_FORWARD",
						OldId:      "c7a28dd5de3403cc384a025",
						NewId:      "f00768887da8de620612102",
					},
				},
			},
		}

		// act
		isAllowed, _ := service.IsAllowedProject(notification)

		assert.False(t, isAllowed)
	})

	t.Run("ReturnsTrueIfOwnerUsernameIsInAllowedProjectsConfig", func(t *testing.T) {

		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		config := &api.APIConfig{
			Integrations: &api.APIConfigIntegrations{
				CloudSource: &api.CloudSourceConfig{
					ProjectOrganizations: []api.ProjectOrganizations{
						{
							Project: "someone-else",
						},
						{
							Project: "estafette-in-cloudsource",
						},
					},
				},
			},
		}
		cloudsourceapiClient := cloudsourceapi.NewMockClient(ctrl)
		pubsubapiClient := pubsubapi.NewMockClient(ctrl)
		estafetteService := estafette.NewMockService(ctrl)
		service := NewService(config, cloudsourceapiClient, pubsubapiClient, estafetteService, api.NewEventTopic("test topic"))

		notification := cloudsourceapi.PubSubNotification{
			Name:      "projects/estafette-in-cloudsource/repos/pubsub",
			Url:       "https://sourcecode.google.com",
			EventTime: time.Now(),
			RefUpdateEvent: &cloudsourceapi.RefUpdateEvent{
				Email: "estafette@estafette.com",
				RefUpdates: map[string]cloudsourceapi.RefUpdate{
					"refs/heads/master": cloudsourceapi.RefUpdate{
						RefName:    "refs/heads/master",
						UpdateType: "UPDATE_FAST_FORWARD",
						OldId:      "c7a28dd5de3403cc384a0258",
						NewId:      "f00768887da8de6206121029",
					},
				},
			},
		}

		// act
		isAllowed, _ := service.IsAllowedProject(notification)

		assert.True(t, isAllowed)
	})
}
