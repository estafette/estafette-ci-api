package auth

import (
	"crypto/elliptic"
	"math/big"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRetrievingIAPJSONWebKeys(t *testing.T) {
	t.Run("ReturnsKeyByKeyID", func(t *testing.T) {

		// act
		publicKey, err := GetCachedIAPJsonWebKey("s3nVXQ")

		expectedX := new(big.Int)
		expectedX, _ = expectedX.SetString("37309719193135926924790436712650528418311291758574183688166423972208868774419", 10)

		expectedY := new(big.Int)
		expectedY, _ = expectedY.SetString("80216437109621353502341424837685742005288420423223052354044301043659735172060", 10)

		if assert.Nil(t, err) {
			assert.Equal(t, elliptic.P256(), publicKey.Curve)
			assert.Equal(t, expectedX, publicKey.X)
			assert.Equal(t, expectedY, publicKey.Y)
		}
	})
}
