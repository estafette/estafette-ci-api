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
	"github.com/stretchr/testify/assert"
)

func TestCreateJobForCloudSourcePush(t *testing.T) {

	t.Run("ReturnsErrNonCloneableEventIfNotificationHasNoRefUpdate", func(t *testing.T) {

		config := &api.APIConfig{
			Integrations: &api.APIConfigIntegrations{
				CloudSource: &api.CloudSourceConfig{},
			},
		}
		cloudsourceapiClient := cloudsourceapi.MockClient{}
		pubsubapiClient := pubsubapi.MockClient{}
		estafetteService := estafette.MockService{}
		service := NewService(config, cloudsourceapiClient, pubsubapiClient, estafetteService, api.NewGitEventTopic("test topic"))

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

		config := &api.APIConfig{
			Integrations: &api.APIConfigIntegrations{
				CloudSource: &api.CloudSourceConfig{},
			},
		}
		cloudsourceapiClient := cloudsourceapi.MockClient{}
		pubsubapiClient := pubsubapi.MockClient{}
		estafetteService := estafette.MockService{}

		getAccessTokenCallCount := 0
		cloudsourceapiClient.GetAccessTokenFunc = func(ctx context.Context) (accesstoken cloudsourceapi.AccessToken, err error) {
			getAccessTokenCallCount++
			return
		}

		service := NewService(config, cloudsourceapiClient, pubsubapiClient, estafetteService, api.NewGitEventTopic("test topic"))

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

		assert.Equal(t, 1, getAccessTokenCallCount)
	})

	t.Run("CallsGetEstafetteManifestOnCloudSourceAPIClient", func(t *testing.T) {

		config := &api.APIConfig{
			Integrations: &api.APIConfigIntegrations{
				CloudSource: &api.CloudSourceConfig{},
			},
		}
		cloudsourceapiClient := cloudsourceapi.MockClient{}
		pubsubapiClient := pubsubapi.MockClient{}
		estafetteService := estafette.MockService{}

		getEstafetteManifestCallCount := 0
		cloudsourceapiClient.GetEstafetteManifestFunc = func(ctx context.Context, accesstoken cloudsourceapi.AccessToken, event cloudsourceapi.PubSubNotification, gitClone func(string, string, string) error) (valid bool, manifest string, err error) {
			getEstafetteManifestCallCount++
			return true, "builder:\n  track: dev\n", nil
		}

		service := NewService(config, cloudsourceapiClient, pubsubapiClient, estafetteService, api.NewGitEventTopic("test topic"))

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
		assert.Equal(t, 1, getEstafetteManifestCallCount)
	})

	t.Run("CallsCreateBuildOnEstafetteService", func(t *testing.T) {

		config := &api.APIConfig{
			Integrations: &api.APIConfigIntegrations{
				CloudSource: &api.CloudSourceConfig{},
			},
		}
		cloudsourceapiClient := cloudsourceapi.MockClient{}
		pubsubapiClient := pubsubapi.MockClient{}
		estafetteService := estafette.MockService{}

		cloudsourceapiClient.GetEstafetteManifestFunc = func(ctx context.Context, accesstoken cloudsourceapi.AccessToken, event cloudsourceapi.PubSubNotification, gitClone func(string, string, string) error) (valid bool, manifest string, err error) {
			return true, "builder:\n  track: dev\n", nil
		}

		createBuildCallCount := 0
		estafetteService.CreateBuildFunc = func(ctx context.Context, build contracts.Build, waitForJobToStart bool) (b *contracts.Build, err error) {
			createBuildCallCount++
			return
		}

		service := NewService(config, cloudsourceapiClient, pubsubapiClient, estafetteService, api.NewGitEventTopic("test topic"))

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
		assert.Equal(t, 1, createBuildCallCount)
	})

	t.Run("PublishesGitTriggersOnTopic", func(t *testing.T) {

		config := &api.APIConfig{
			Integrations: &api.APIConfigIntegrations{
				CloudSource: &api.CloudSourceConfig{},
			},
		}
		cloudsourceapiClient := cloudsourceapi.MockClient{}
		pubsubapiClient := pubsubapi.MockClient{}
		estafetteService := estafette.MockService{}
		gitEventTopic := api.NewGitEventTopic("test topic")
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

		// act
		_ = service.CreateJobForCloudSourcePush(context.Background(), notification)

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
				CloudSource: &api.CloudSourceConfig{},
			},
		}
		cloudsourceapiClient := cloudsourceapi.MockClient{}
		pubsubapiClient := pubsubapi.MockClient{}
		estafetteService := estafette.MockService{}

		cloudsourceapiClient.GetEstafetteManifestFunc = func(ctx context.Context, accesstoken cloudsourceapi.AccessToken, event cloudsourceapi.PubSubNotification, gitClone func(string, string, string) error) (valid bool, manifest string, err error) {
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

		service := NewService(config, cloudsourceapiClient, pubsubapiClient, estafetteService, api.NewGitEventTopic("test topic"))

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

		wg.Wait()

		assert.Nil(t, err)
		assert.Equal(t, 1, subscribeToPubsubTriggersCallCount)
	})
}

func TestIsWhitelistedProject(t *testing.T) {

	t.Run("ReturnsTrueIfWhitelistedProjectsConfigIsEmpty", func(t *testing.T) {

		config := &api.APIConfig{
			Integrations: &api.APIConfigIntegrations{
				CloudSource: &api.CloudSourceConfig{
					ProjectOrganizations: []api.ProjectOrganizations{},
				},
			},
		}
		cloudsourceapiClient := cloudsourceapi.MockClient{}
		pubsubapiClient := pubsubapi.MockClient{}
		estafetteService := estafette.MockService{}
		service := NewService(config, cloudsourceapiClient, pubsubapiClient, estafetteService, api.NewGitEventTopic("test topic"))

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
		isWhitelisted := service.IsWhitelistedProject(notification)

		assert.True(t, isWhitelisted)
	})

	t.Run("ReturnsFalseIfOwnerUsernameIsNotInWhitelistedProjectsConfig", func(t *testing.T) {

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
		cloudsourceapiClient := cloudsourceapi.MockClient{}
		pubsubapiClient := pubsubapi.MockClient{}
		estafetteService := estafette.MockService{}
		service := NewService(config, cloudsourceapiClient, pubsubapiClient, estafetteService, api.NewGitEventTopic("test topic"))

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
		isWhitelisted := service.IsWhitelistedProject(notification)

		assert.False(t, isWhitelisted)
	})

	t.Run("ReturnsTrueIfOwnerUsernameIsInWhitelistedProjectsConfig", func(t *testing.T) {

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
		cloudsourceapiClient := cloudsourceapi.MockClient{}
		pubsubapiClient := pubsubapi.MockClient{}
		estafetteService := estafette.MockService{}
		service := NewService(config, cloudsourceapiClient, pubsubapiClient, estafetteService, api.NewGitEventTopic("test topic"))

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
		isWhitelisted := service.IsWhitelistedProject(notification)

		assert.True(t, isWhitelisted)
	})
}
