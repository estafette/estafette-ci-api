package dockerhubapi

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestGetToken(t *testing.T) {

	t.Run("ReturnsTokenForRepository", func(t *testing.T) {

		client := NewClient()

		// act
		token, err := client.GetToken(context.Background(), "estafette/estafette-ci-builder")

		assert.Nil(t, err)
		assert.NotNil(t, token)
		assert.Equal(t, 300, token.ExpiresIn)
	})
}

func TestGetDigest(t *testing.T) {

	t.Run("ReturnsDigestForRepositoryAndTag", func(t *testing.T) {

		client := NewClient()
		token, err := client.GetToken(context.Background(), "estafette/estafette-ci-builder")
		assert.Nil(t, err)

		// act
		digest, err := client.GetDigest(context.Background(), token, "estafette/estafette-ci-builder", "0.0.245")

		assert.Nil(t, err)
		assert.Equal(t, "sha256:00758c7ba65441b93bd5ecb6fe0242587560af061045bcb7337cd6c618cffe5e", digest.Digest)
	})
}

func TestGetDigestCached(t *testing.T) {

	t.Run("ReturnsDigestForRepositoryAndTag", func(t *testing.T) {

		client := NewClient()

		// act
		_, err := client.GetDigestCached(context.Background(), "estafette/estafette-ci-builder", "0.0.245")
		assert.Nil(t, err)

		digest, err := client.GetDigestCached(context.Background(), "estafette/estafette-ci-builder", "0.0.245")

		assert.Nil(t, err)
		assert.Equal(t, "sha256:00758c7ba65441b93bd5ecb6fe0242587560af061045bcb7337cd6c618cffe5e", digest.Digest)
	})
}

func TestExpiresAt(t *testing.T) {

	t.Run("ReturnsIssuedAtPlusExpiredAtSeconds", func(t *testing.T) {

		token := DockerHubToken{
			ExpiresIn: 300,
			IssuedAt:  time.Date(2017, 11, 18, 21, 03, 0, 0, time.UTC),
		}

		// act
		expiresAt := token.ExpiresAt()

		assert.Equal(t, time.Date(2017, 11, 18, 21, 8, 0, 0, time.UTC), expiresAt)
	})
}

func TestIsExpired(t *testing.T) {

	t.Run("ReturnsTrueIfIssuedLongerThanExpiresInSecondsAgo", func(t *testing.T) {

		token := DockerHubToken{
			ExpiresIn: 300,
			IssuedAt:  time.Now().UTC().Add(time.Duration(-301) * time.Second),
		}

		// act
		isExpired := token.IsExpired()

		assert.True(t, isExpired)
	})

	t.Run("ReturnsFalseIfIssuedLessThanExpiresInSecondsAgo", func(t *testing.T) {

		token := DockerHubToken{
			ExpiresIn: 300,
			IssuedAt:  time.Now().UTC().Add(time.Duration(-299) * time.Second),
		}

		// act
		isExpired := token.IsExpired()

		assert.False(t, isExpired)
	})
}
