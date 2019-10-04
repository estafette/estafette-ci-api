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
		publicKey, err := GetCachedGoogleJWK("8c58e138614bd58742172bd5080d197d2b2dd2f3")

		if assert.Nil(t, err) {
			expectedN, _ := new(big.Int).SetString("25499125553942060609012146370546486889172827671420118409283641225903120719117860185057743127679728342322640549464463738761200284233416288905370711621063661311424521372799486180643323823499817852164035086636515528670534305560994952550102156206329812367903536209929853662413368681674475260110893618627060874128630024190720860321546608256594236115732541540972889350913790175419166406016172955665107380289331867394017281891832061391619247959353510801826536397531831795634532880640508334747558015885185329548071105912865621412382270196865961897917183158829718696494708560288824126046783394911850994364074776798161878370083", 10)

			if assert.Equal(t, expectedN, publicKey.N) {

				expectedY := 65537
				assert.Equal(t, expectedY, publicKey.E)
			}
		}
	})
}
