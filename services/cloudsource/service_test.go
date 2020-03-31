package cloudsource

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/estafette/estafette-ci-api/clients/cloudsourceapi"
	"github.com/estafette/estafette-ci-api/clients/pubsubapi"
	"github.com/estafette/estafette-ci-api/config"
	"github.com/estafette/estafette-ci-api/services/estafette"
	contracts "github.com/estafette/estafette-ci-contracts"
	manifest "github.com/estafette/estafette-ci-manifest"
	"github.com/stretchr/testify/assert"
)

func TestCreateJobForCloudSourcePush(t *testing.T) {

	t.Run("ReturnsErrNonCloneableEventIfNotificationHasNoRefUpdate", func(t *testing.T) {

		config := config.CloudSourceConfig{}
		cloudsourceapiClient := cloudsourceapi.MockClient{}
		pubsubapiClient := pubsubapi.MockClient{}
		estafetteService := estafette.MockService{}
		service := NewService(config, cloudsourceapiClient, pubsubapiClient, estafetteService)

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

		config := config.CloudSourceConfig{}
		cloudsourceapiClient := cloudsourceapi.MockClient{}
		pubsubapiClient := pubsubapi.MockClient{}
		estafetteService := estafette.MockService{}

		getAccessTokenCallCount := 0
		cloudsourceapiClient.GetAccessTokenFunc = func(ctx context.Context) (accesstoken cloudsourceapi.AccessToken, err error) {
			getAccessTokenCallCount++
			return
		}

		service := NewService(config, cloudsourceapiClient, pubsubapiClient, estafetteService)

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

		config := config.CloudSourceConfig{}
		cloudsourceapiClient := cloudsourceapi.MockClient{}
		pubsubapiClient := pubsubapi.MockClient{}
		estafetteService := estafette.MockService{}

		getEstafetteManifestCallCount := 0
		cloudsourceapiClient.GetEstafetteManifestFunc = func(ctx context.Context, accesstoken cloudsourceapi.AccessToken, event cloudsourceapi.PubSubNotification) (valid bool, manifest string, err error) {
			getEstafetteManifestCallCount++
			return true, "builder:\n  track: dev\n", nil
		}

		service := NewService(config, cloudsourceapiClient, pubsubapiClient, estafetteService)

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

		config := config.CloudSourceConfig{}
		cloudsourceapiClient := cloudsourceapi.MockClient{}
		pubsubapiClient := pubsubapi.MockClient{}
		estafetteService := estafette.MockService{}

		cloudsourceapiClient.GetEstafetteManifestFunc = func(ctx context.Context, accesstoken cloudsourceapi.AccessToken, event cloudsourceapi.PubSubNotification) (valid bool, manifest string, err error) {
			return true, "builder:\n  track: dev\n", nil
		}

		createBuildCallCount := 0
		estafetteService.CreateBuildFunc = func(ctx context.Context, build contracts.Build, waitForJobToStart bool) (b *contracts.Build, err error) {
			createBuildCallCount++
			return
		}

		service := NewService(config, cloudsourceapiClient, pubsubapiClient, estafetteService)

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

	t.Run("CallsFireGitTriggersOnEstafetteService", func(t *testing.T) {

		config := config.CloudSourceConfig{}
		cloudsourceapiClient := cloudsourceapi.MockClient{}
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

		service := NewService(config, cloudsourceapiClient, pubsubapiClient, estafetteService)

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

		wg.Wait()

		assert.Equal(t, 1, fireGitTriggersCallCount)
	})

	t.Run("CallsSubscribeToPubsubTriggersOnPubsubAPIClient", func(t *testing.T) {

		config := config.CloudSourceConfig{}
		cloudsourceapiClient := cloudsourceapi.MockClient{}
		pubsubapiClient := pubsubapi.MockClient{}
		estafetteService := estafette.MockService{}

		cloudsourceapiClient.GetEstafetteManifestFunc = func(ctx context.Context, accesstoken cloudsourceapi.AccessToken, event cloudsourceapi.PubSubNotification) (valid bool, manifest string, err error) {
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

		service := NewService(config, cloudsourceapiClient, pubsubapiClient, estafetteService)

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

func TestIsWhitelistedOwner(t *testing.T) {

	t.Run("ReturnsTrueIfWhitelistedOwnersConfigIsEmpty", func(t *testing.T) {

		config := config.CloudSourceConfig{
			WhitelistedOwners: []string{},
		}
		cloudsourceapiClient := cloudsourceapi.MockClient{}
		pubsubapiClient := pubsubapi.MockClient{}
		estafetteService := estafette.MockService{}
		service := NewService(config, cloudsourceapiClient, pubsubapiClient, estafetteService)

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
		isWhitelisted := service.IsWhitelistedOwner(notification)

		assert.True(t, isWhitelisted)
	})

	t.Run("ReturnsFalseIfOwnerUsernameIsNotInWhitelistedOwnersConfig", func(t *testing.T) {

		config := config.CloudSourceConfig{
			WhitelistedOwners: []string{
				"someone-else",
			},
		}
		cloudsourceapiClient := cloudsourceapi.MockClient{}
		pubsubapiClient := pubsubapi.MockClient{}
		estafetteService := estafette.MockService{}
		service := NewService(config, cloudsourceapiClient, pubsubapiClient, estafetteService)

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
		isWhitelisted := service.IsWhitelistedOwner(notification)

		assert.False(t, isWhitelisted)
	})

	t.Run("ReturnsTrueIfOwnerUsernameIsInWhitelistedOwnersConfig", func(t *testing.T) {

		config := config.CloudSourceConfig{
			WhitelistedOwners: []string{
				"someone-else",
				"estafette-in-cloudsource",
			},
		}
		cloudsourceapiClient := cloudsourceapi.MockClient{}
		pubsubapiClient := pubsubapi.MockClient{}
		estafetteService := estafette.MockService{}
		service := NewService(config, cloudsourceapiClient, pubsubapiClient, estafetteService)

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
		isWhitelisted := service.IsWhitelistedOwner(notification)

		assert.True(t, isWhitelisted)
	})
}
