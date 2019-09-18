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
		publicKey, err := GetCachedIAPJWK("f9R3yg")

		if assert.Nil(t, err) {
			assert.Equal(t, elliptic.P256(), publicKey.Curve)

			expectedX := new(big.Int)
			expectedX, _ = expectedX.SetString("33754992528993959342082873952071099444905807959681776349240807143574023195992", 10)

			if assert.Equal(t, expectedX, publicKey.X) {

				expectedY := new(big.Int)
				expectedY, _ = expectedY.SetString("30017756976983295626595109856839943719662421701617989535808220756803905010317", 10)

				assert.Equal(t, expectedY, publicKey.Y)
			}
		}
	})
}

func TestRetrievingGoogleJSONWebKeys(t *testing.T) {
	t.Run("ReturnsKeyByKeyID", func(t *testing.T) {

		// act (if fails get new kid from https://www.googleapis.com/oauth2/v3/certs and update expectancies until it works)
		publicKey, err := GetCachedGoogleJWK("2bf8418b2963f366f5fefdd127b2cee07c887e65")

		if assert.Nil(t, err) {
			expectedN, _ := new(big.Int).SetString("23235268419750350896479879774659191573169567918849369896536107657036353446658863440858788954827536576575413231263592319108039821728164143132661704276896098114455259786698954221072891772383849652804054509686678787128765717574331384816149020176385994332849203556079122536610124679883967712868758010387631337374974987995929385432252095024159800995698335197590296037258572875687848207973821027051092405054150783043512186661871319341843441378814478780665983372672903095819343873037871776953006134890865358650885931068812652117149823699039043511460975454922196213016913382801039507006105778821319235127830677266979423153599", 10)

			if assert.Equal(t, expectedN, publicKey.N) {

				expectedY := 65537
				assert.Equal(t, expectedY, publicKey.E)
			}
		}
	})
}
