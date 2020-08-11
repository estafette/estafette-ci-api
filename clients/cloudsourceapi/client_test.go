package cloudsourceapi

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	sourcerepo "google.golang.org/api/sourcerepo/v1"
	stdsourcerepo "google.golang.org/api/sourcerepo/v1"
)

func TestGetToken(t *testing.T) {

	t.Run("ReturnsTokenForRepository", func(t *testing.T) {

		tokenSource, service := getTokenSourceAndService()
		client, err := NewClient(tokenSource, service)
		assert.Nil(t, err)

		// act
		token, err := client.GetAccessToken(context.Background())

		assert.Nil(t, err)
		assert.NotNil(t, token)
	})
}

func TestGetAuthenticatedRepositoryURL(t *testing.T) {

	t.Run("ReturnsAuthenticatedURLForRepository", func(t *testing.T) {

		tokenSource, service := getTokenSourceAndService()
		client, err := NewClient(tokenSource, service)
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

		tokenSource, service := getTokenSourceAndService()
		client, err := NewClient(tokenSource, service)
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

		tokenSource, service := getTokenSourceAndService()
		client, err := NewClient(tokenSource, service)
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

func getTokenSourceAndService() (oauth2.TokenSource, *sourcerepo.Service) {
	ctx := context.Background()
	tokenSource, err := google.DefaultTokenSource(ctx, sourcerepo.CloudPlatformScope)
	if err != nil {
		log.Fatal("Creating google cloud token source has failed")
	}
	sourcerepoService, err := stdsourcerepo.New(oauth2.NewClient(ctx, tokenSource))
	if err != nil {
		log.Fatal("Creating google cloud source repo service has failed")
	}

	return tokenSource, sourcerepoService
}
