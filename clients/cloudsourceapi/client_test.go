package cloudsourceapi

import (
	"context"
	"fmt"
	"testing"

	"github.com/estafette/estafette-ci-api/config"
	"github.com/stretchr/testify/assert"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	sourcerepo "google.golang.org/api/sourcerepo/v1"
)

func TestGetToken(t *testing.T) {

	t.Run("ReturnsTokenForRepository", func(t *testing.T) {

		ctx := context.Background()
		tokenSource, err := google.DefaultTokenSource(ctx, sourcerepo.CloudPlatformScope)
		assert.Nil(t, err)
		sourcerepoService, err := sourcerepo.New(oauth2.NewClient(ctx, tokenSource))
		assert.Nil(t, err)

		cloudSourceConfig := config.CloudSourceConfig{
			PrivateKeyPath: "",
			WhitelistedOwners: []string{
				"estafette",
			},
		}
		client := NewClient(cloudSourceConfig, sourcerepoService, tokenSource)

		// act
		token, err := client.GetAccessToken(ctx)

		assert.Nil(t, err)
		assert.NotNil(t, token)
	})
}

func TestGetAuthenticatedRepositoryURL(t *testing.T) {

	t.Run("ReturnsAuthenticatedURLForRepository", func(t *testing.T) {

		ctx := context.Background()
		tokenSource, err := google.DefaultTokenSource(ctx, sourcerepo.CloudPlatformScope)
		assert.Nil(t, err)
		sourcerepoService, err := sourcerepo.New(oauth2.NewClient(ctx, tokenSource))
		assert.Nil(t, err)

		cloudSourceConfig := config.CloudSourceConfig{
			PrivateKeyPath: "",
			WhitelistedOwners: []string{
				"estafette",
			},
		}
		client := NewClient(cloudSourceConfig, sourcerepoService, tokenSource)

		// act
		token, err := client.GetAccessToken(ctx)

		assert.Nil(t, err)
		assert.NotNil(t, token)

		url, err := client.GetAuthenticatedRepositoryURL(ctx, token, fmt.Sprintf("https://%v/p/%v/r/%v", "source.developers.google.com", "playground-varins-idpd0", "test-estafette-support"))
		expectedUrl := "https://x-token-auth:.+@source\\.developers\\.google\\.com/p/playground-varins-idpd0/r/test-estafette-support"
		assert.Nil(t, err)
		assert.NotNil(t, url)
		assert.Regexp(t, expectedUrl, url)
	})
}
