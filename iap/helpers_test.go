package iap

import (
	"crypto/elliptic"
	"math/big"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRetrievingIAPJSONWebKeys(t *testing.T) {
	t.Run("ReturnsKeyByKeyID", func(t *testing.T) {

		// act
		publicKey, err := GetCachedIAPJsonWebKey("BOIWdQ")

		expectedX := new(big.Int)
		expectedX, _ = expectedX.SetString("51848729579695172483464161482242606434602712503912468683154292353327863710431", 10)

		expectedY := new(big.Int)
		expectedY, _ = expectedY.SetString("50055884315100024408193525825372861832752941579251404418416297609341596837855", 10)

		assert.Nil(t, err)
		assert.Equal(t, elliptic.P256(), publicKey.Curve)
		assert.Equal(t, expectedX, publicKey.X)
		assert.Equal(t, expectedY, publicKey.Y)
	})
}
