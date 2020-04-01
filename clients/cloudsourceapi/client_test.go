package cloudsourceapi

import (
	"context"
	"fmt"
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
