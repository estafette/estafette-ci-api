package cloudsourceapi

import (
	"context"
	"log"
	"os"
	"path/filepath"
	"testing"

	"github.com/estafette/estafette-ci-api/pkg/api"
	"github.com/stretchr/testify/assert"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/option"
	"google.golang.org/api/sourcerepo/v1"
)

func TestGetToken(t *testing.T) {
	if testing.Short() {
		return
	}

	t.Run("ReturnsTokenForRepository", func(t *testing.T) {

		config, tokenSource, service := getTokenSourceAndService()
		client := NewClient(config, tokenSource, service)

		// act
		token, err := client.GetAccessToken(context.Background())

		assert.Nil(t, err)
		assert.NotNil(t, token)
	})
}

func TestGetEstafetteManifest(t *testing.T) {
	if testing.Short() {
		return
	}

	notification := PubSubNotification{
		Name: "projects/test-project/repos/pubsub-test",
		Url:  "https://sourcecode.google.com",
		RefUpdateEvent: &RefUpdateEvent{
			Email: "estafette@estafette.com",
			RefUpdates: map[string]RefUpdate{
				"refs/heads/master": RefUpdate{
					RefName:    "refs/heads/master",
					UpdateType: "UPDATE_FAST_FORWARD",
					OldId:      "c7a28dd5de3403cc384a025",
					NewId:      "f00768887da8de620612102",
				},
			},
		},
	}

	t.Run("ReturnsFalseIfNoManifestExists", func(t *testing.T) {

		config, tokenSource, service := getTokenSourceAndService()
		client := NewClient(config, tokenSource, service)

		ctx := context.Background()
		token, err := client.GetAccessToken(ctx)

		assert.Nil(t, err)
		assert.NotNil(t, token)

		gitClone := func(dir, gitUrl, repoRefName string) error {
			return nil
		}

		// act
		exists, manifest, err := client.GetEstafetteManifest(ctx, token, notification, gitClone)

		assert.False(t, exists)
		assert.Empty(t, manifest)
		assert.Nil(t, err)
	})

	t.Run("ReturnsTrueIfManifestExists", func(t *testing.T) {

		config, tokenSource, service := getTokenSourceAndService()
		client := NewClient(config, tokenSource, service)

		ctx := context.Background()
		token, err := client.GetAccessToken(ctx)

		assert.Nil(t, err)
		assert.NotNil(t, token)

		gitClone := func(dir, gitUrl, repoRefName string) error {
			file, err := os.Create(filepath.Join(dir, ".estafette.yaml"))
			if err != nil {
				return err
			}
			defer file.Close()

			d1 := []byte("hello\ngo\n")
			_, err = file.Write(d1)
			if err != nil {
				return err
			}

			return nil
		}

		// act
		exists, manifest, err := client.GetEstafetteManifest(ctx, token, notification, gitClone)

		assert.True(t, exists)
		assert.NotEmpty(t, manifest)
		assert.Nil(t, err)
	})
}

func getTokenSourceAndService() (*api.APIConfig, oauth2.TokenSource, *sourcerepo.Service) {
	ctx := context.Background()
	tokenSource, err := google.DefaultTokenSource(ctx, sourcerepo.CloudPlatformScope)
	if err != nil {
		log.Fatal("Creating google cloud token source has failed")
	}

	sourcerepoService, err := sourcerepo.NewService(ctx, option.WithHTTPClient(oauth2.NewClient(ctx, tokenSource)))
	if err != nil {
		log.Fatal("Creating google cloud source repo service has failed")
	}

	return &api.APIConfig{Integrations: &api.APIConfigIntegrations{CloudSource: &api.CloudSourceConfig{Enable: true}}}, tokenSource, sourcerepoService
}
