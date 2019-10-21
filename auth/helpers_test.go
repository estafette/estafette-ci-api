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
		publicKey, err := GetCachedGoogleJWK("3db3ed6b9574ee3fcd9f149e59ff0eef4f932153")

		if assert.Nil(t, err) {
			expectedN, _ := new(big.Int).SetString("27349905058855127968386103083394866213815847157226658207829268545600181950377300862481176817912409670858609137610020055997633413328762561137991178831777091094547887792847449826917401187028086955683997109466088616161972480428803322222835274139626412339095605534896733458293924418586435954201369261155962062592044320146667266607982957182509068548596713693647536087590222980421957308422730890043002602024506478770355300469677715328888224830981932761345728416332941464505853097186519865782757670005375943325117046535555363673051705356588671170776656604497778320950692720571503196071281550014910248825942892005693136500731", 10)

			if assert.Equal(t, expectedN, publicKey.N) {

				expectedY := 65537
				assert.Equal(t, expectedY, publicKey.E)
			}
		}
	})
}
