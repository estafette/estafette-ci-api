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
		publicKey, err := GetCachedIAPJsonWebKey("rTlk-g")

		expectedX := new(big.Int)
		expectedX, _ = expectedX.SetString("80705443177100294943977016489350978230449831162147377309904653463161053062518", 10)

		expectedY := new(big.Int)
		expectedY, _ = expectedY.SetString("39386914180697077709177261326057757164855031063540188737223679822281777517038", 10)

		if assert.Nil(t, err) {
			assert.Equal(t, elliptic.P256(), publicKey.Curve)
			assert.Equal(t, expectedX, publicKey.X)
			assert.Equal(t, expectedY, publicKey.Y)
		}
	})
}
