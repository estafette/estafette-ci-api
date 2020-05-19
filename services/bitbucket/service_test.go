package bitbucket

import (
	"context"
	"errors"
	"sync"
	"testing"

	"github.com/estafette/estafette-ci-api/clients/bitbucketapi"
	"github.com/estafette/estafette-ci-api/clients/pubsubapi"
	"github.com/estafette/estafette-ci-api/config"
	"github.com/estafette/estafette-ci-api/services/estafette"
	contracts "github.com/estafette/estafette-ci-contracts"
	manifest "github.com/estafette/estafette-ci-manifest"
	"github.com/stretchr/testify/assert"
)

func TestCreateJobForBitbucketPush(t *testing.T) {

	t.Run("ReturnsErrNonCloneableEventIfPushEventHasNoChanges", func(t *testing.T) {

		config := &config.APIConfig{
			Integrations: &config.APIConfigIntegrations{
				Bitbucket: &config.BitbucketConfig{},
			},
		}
		bitbucketapiClient := bitbucketapi.MockClient{}
		pubsubapiClient := pubsubapi.MockClient{}
		estafetteService := estafette.MockService{}
		service := NewService(config, bitbucketapiClient, pubsubapiClient, estafetteService)

		pushEvent := bitbucketapi.RepositoryPushEvent{}

		// act
		err := service.CreateJobForBitbucketPush(context.Background(), pushEvent)

		assert.NotNil(t, err)
		assert.True(t, errors.Is(err, ErrNonCloneableEvent))
	})

	t.Run("ReturnsErrNonCloneableEventIfPushEventChangeHasNoNewObject", func(t *testing.T) {

		config := &config.APIConfig{
			Integrations: &config.APIConfigIntegrations{
				Bitbucket: &config.BitbucketConfig{},
			},
		}
		bitbucketapiClient := bitbucketapi.MockClient{}
		pubsubapiClient := pubsubapi.MockClient{}
		estafetteService := estafette.MockService{}
		service := NewService(config, bitbucketapiClient, pubsubapiClient, estafetteService)

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

		config := &config.APIConfig{
			Integrations: &config.APIConfigIntegrations{
				Bitbucket: &config.BitbucketConfig{},
			},
		}
		bitbucketapiClient := bitbucketapi.MockClient{}
		pubsubapiClient := pubsubapi.MockClient{}
		estafetteService := estafette.MockService{}
		service := NewService(config, bitbucketapiClient, pubsubapiClient, estafetteService)

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

		config := &config.APIConfig{
			Integrations: &config.APIConfigIntegrations{
				Bitbucket: &config.BitbucketConfig{},
			},
		}
		bitbucketapiClient := bitbucketapi.MockClient{}
		pubsubapiClient := pubsubapi.MockClient{}
		estafetteService := estafette.MockService{}
		service := NewService(config, bitbucketapiClient, pubsubapiClient, estafetteService)

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

		config := &config.APIConfig{
			Integrations: &config.APIConfigIntegrations{
				Bitbucket: &config.BitbucketConfig{},
			},
		}
		bitbucketapiClient := bitbucketapi.MockClient{}
		pubsubapiClient := pubsubapi.MockClient{}
		estafetteService := estafette.MockService{}

		getAccessTokenCallCount := 0
		bitbucketapiClient.GetAccessTokenFunc = func(ctx context.Context) (accesstoken bitbucketapi.AccessToken, err error) {
			getAccessTokenCallCount++
			return
		}

		service := NewService(config, bitbucketapiClient, pubsubapiClient, estafetteService)

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

		config := &config.APIConfig{
			Integrations: &config.APIConfigIntegrations{
				Bitbucket: &config.BitbucketConfig{},
			},
		}
		bitbucketapiClient := bitbucketapi.MockClient{}
		pubsubapiClient := pubsubapi.MockClient{}
		estafetteService := estafette.MockService{}

		getEstafetteManifestCallCount := 0
		bitbucketapiClient.GetEstafetteManifestFunc = func(ctx context.Context, accesstoken bitbucketapi.AccessToken, event bitbucketapi.RepositoryPushEvent) (valid bool, manifest string, err error) {
			getEstafetteManifestCallCount++
			return true, "builder:\n  track: dev\n", nil
		}

		service := NewService(config, bitbucketapiClient, pubsubapiClient, estafetteService)

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

		config := &config.APIConfig{
			Integrations: &config.APIConfigIntegrations{
				Bitbucket: &config.BitbucketConfig{},
			},
		}
		bitbucketapiClient := bitbucketapi.MockClient{}
		pubsubapiClient := pubsubapi.MockClient{}
		estafetteService := estafette.MockService{}

		bitbucketapiClient.GetEstafetteManifestFunc = func(ctx context.Context, accesstoken bitbucketapi.AccessToken, event bitbucketapi.RepositoryPushEvent) (valid bool, manifest string, err error) {
			return true, "builder:\n  track: dev\n", nil
		}

		createBuildCallCount := 0
		estafetteService.CreateBuildFunc = func(ctx context.Context, build contracts.Build, waitForJobToStart bool) (b *contracts.Build, err error) {
			createBuildCallCount++
			return
		}

		service := NewService(config, bitbucketapiClient, pubsubapiClient, estafetteService)

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

	t.Run("CallsFireGitTriggersOnEstafetteService", func(t *testing.T) {

		config := &config.APIConfig{
			Integrations: &config.APIConfigIntegrations{
				Bitbucket: &config.BitbucketConfig{},
			},
		}
		bitbucketapiClient := bitbucketapi.MockClient{}
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

		service := NewService(config, bitbucketapiClient, pubsubapiClient, estafetteService)

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

		wg.Wait()

		assert.Equal(t, 1, fireGitTriggersCallCount)
	})

	t.Run("CallsSubscribeToPubsubTriggersOnPubsubAPIClient", func(t *testing.T) {

		config := &config.APIConfig{
			Integrations: &config.APIConfigIntegrations{
				Bitbucket: &config.BitbucketConfig{},
			},
		}
		bitbucketapiClient := bitbucketapi.MockClient{}
		pubsubapiClient := pubsubapi.MockClient{}
		estafetteService := estafette.MockService{}

		bitbucketapiClient.GetEstafetteManifestFunc = func(ctx context.Context, accesstoken bitbucketapi.AccessToken, event bitbucketapi.RepositoryPushEvent) (valid bool, manifest string, err error) {
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

		service := NewService(config, bitbucketapiClient, pubsubapiClient, estafetteService)

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

func TestIsWhitelistedOwner(t *testing.T) {

	t.Run("ReturnsTrueIfWhitelistedOwnersConfigIsEmpty", func(t *testing.T) {

		config := &config.APIConfig{
			Integrations: &config.APIConfigIntegrations{
				Bitbucket: &config.BitbucketConfig{
					WhitelistedOwners: []string{},
				},
			},
		}
		bitbucketapiClient := bitbucketapi.MockClient{}
		pubsubapiClient := pubsubapi.MockClient{}
		estafetteService := estafette.MockService{}
		service := NewService(config, bitbucketapiClient, pubsubapiClient, estafetteService)

		repository := bitbucketapi.Repository{
			Owner: bitbucketapi.Owner{
				UserName: "anyone",
			},
		}

		// act
		isWhitelisted := service.IsWhitelistedOwner(repository)

		assert.True(t, isWhitelisted)
	})

	t.Run("ReturnsFalseIfOwnerUsernameIsNotInWhitelistedOwnersConfig", func(t *testing.T) {

		config := &config.APIConfig{
			Integrations: &config.APIConfigIntegrations{
				Bitbucket: &config.BitbucketConfig{
					WhitelistedOwners: []string{
						"someone-else",
					},
				},
			},
		}
		bitbucketapiClient := bitbucketapi.MockClient{}
		pubsubapiClient := pubsubapi.MockClient{}
		estafetteService := estafette.MockService{}
		service := NewService(config, bitbucketapiClient, pubsubapiClient, estafetteService)

		repository := bitbucketapi.Repository{
			Owner: bitbucketapi.Owner{
				UserName: "estafette-in-bitbucket",
			},
		}

		// act
		isWhitelisted := service.IsWhitelistedOwner(repository)

		assert.False(t, isWhitelisted)
	})

	t.Run("ReturnsTrueIfOwnerUsernameIsInWhitelistedOwnersConfig", func(t *testing.T) {

		config := &config.APIConfig{
			Integrations: &config.APIConfigIntegrations{
				Bitbucket: &config.BitbucketConfig{
					WhitelistedOwners: []string{
						"someone-else",
						"estafette-in-bitbucket",
					},
				},
			},
		}
		bitbucketapiClient := bitbucketapi.MockClient{}
		pubsubapiClient := pubsubapi.MockClient{}
		estafetteService := estafette.MockService{}
		service := NewService(config, bitbucketapiClient, pubsubapiClient, estafetteService)

		repository := bitbucketapi.Repository{
			Owner: bitbucketapi.Owner{
				UserName: "estafette-in-bitbucket",
			},
		}

		// act
		isWhitelisted := service.IsWhitelistedOwner(repository)

		assert.True(t, isWhitelisted)
	})
}

func TestRename(t *testing.T) {

	t.Run("CallsRenameOnEstafetteService", func(t *testing.T) {

		config := &config.APIConfig{
			Integrations: &config.APIConfigIntegrations{
				Bitbucket: &config.BitbucketConfig{},
			},
		}
		bitbucketapiClient := bitbucketapi.MockClient{}
		pubsubapiClient := pubsubapi.MockClient{}
		estafetteService := estafette.MockService{}

		renameCallCount := 0
		estafetteService.RenameFunc = func(ctx context.Context, fromRepoSource, fromRepoOwner, fromRepoName, toRepoSource, toRepoOwner, toRepoName string) (err error) {
			renameCallCount++
			return
		}

		service := NewService(config, bitbucketapiClient, pubsubapiClient, estafetteService)

		// act
		err := service.Rename(context.Background(), "bitbucket.org", "estafette", "estafette-ci-contracts", "bitbucket.org", "estafette", "estafette-ci-protos")

		assert.Nil(t, err)
		assert.Equal(t, 1, renameCallCount)
	})
}
