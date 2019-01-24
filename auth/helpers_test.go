package auth

import (
	"crypto/elliptic"
	"math/big"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRetrievingIAPJSONWebKeys(t *testing.T) {
	t.Run("ReturnsKeyByKeyID", func(t *testing.T) {

		// act (if fails get new kid from https://www.gstatic.com/iap/verify/public_key-jwk and update expectancies until it works)
		publicKey, err := GetCachedIAPJsonWebKey("FAWt5w")

		if assert.Nil(t, err) {
			assert.Equal(t, elliptic.P256(), publicKey.Curve)

			expectedX := new(big.Int)
			expectedX, _ = expectedX.SetString("109310548864476541829996403431107199831106596242696403068756948144863290489665", 10)

			if assert.Equal(t, expectedX, publicKey.X) {

				expectedY := new(big.Int)
				expectedY, _ = expectedY.SetString("15007658710064885800843054588135540073471183338568009446226392736133283689534", 10)

				assert.Equal(t, expectedY, publicKey.Y)
			}
		}
	})
}
