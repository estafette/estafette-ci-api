package cloudsourceapi

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/estafette/estafette-ci-api/config"
	"github.com/stretchr/testify/assert"
)

func TestGetToken(t *testing.T) {

	t.Run("ReturnsTokenForRepository", func(t *testing.T) {

		cloudSourceConfig := config.CloudSourceConfig{
			WhitelistedProjects: []string{
				"estafette",
			},
		}
		client, err := NewClient(cloudSourceConfig)
		assert.Nil(t, err)

		// act
		token, err := client.GetAccessToken(context.Background())

		assert.Nil(t, err)
		assert.NotNil(t, token)
	})
}

func TestGetAuthenticatedRepositoryURL(t *testing.T) {

	t.Run("ReturnsAuthenticatedURLForRepository", func(t *testing.T) {

		cloudSourceConfig := config.CloudSourceConfig{
			WhitelistedProjects: []string{
				"estafette",
			},
		}
		client, err := NewClient(cloudSourceConfig)
		assert.Nil(t, err)

		// act
		ctx := context.Background()
		token, err := client.GetAccessToken(ctx)

		assert.Nil(t, err)
		assert.NotNil(t, token)

		url, err := client.GetAuthenticatedRepositoryURL(ctx, token, fmt.Sprintf("https://%v/p/%v/r/%v", "source.developers.google.com", "playground-varins-idpd0", "test-estafette-support"))
		expectedUrl := "https://estafette:.+@source\\.developers\\.google\\.com/p/playground-varins-idpd0/r/test-estafette-support"
		assert.Nil(t, err)
		assert.NotNil(t, url)
		assert.Regexp(t, expectedUrl, url)
	})
}

func TestGetEstafetteManifest(t *testing.T) {

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

		cloudSourceConfig := config.CloudSourceConfig{
			WhitelistedProjects: []string{
				"estafette",
			},
		}
		client, err := NewClient(cloudSourceConfig)
		assert.Nil(t, err)

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

		cloudSourceConfig := config.CloudSourceConfig{
			WhitelistedProjects: []string{
				"estafette",
			},
		}
		client, err := NewClient(cloudSourceConfig)
		assert.Nil(t, err)

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

			return nil
		}

		// act
		exists, manifest, err := client.GetEstafetteManifest(ctx, token, notification, gitClone)

		assert.True(t, exists)
		assert.NotEmpty(t, manifest)
		assert.Nil(t, err)
	})
}
