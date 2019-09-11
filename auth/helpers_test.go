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
		publicKey, err := GetCachedGoogleJWK("05a02649a5b45c90fdfe4da1ebefa9c079ab593e")

		if assert.Nil(t, err) {
			expectedN, _ := new(big.Int).SetString("26386640196372426380437575256004548936078072361252547077903440302506233434026751142864687086939606473921399288790210323600409676795193267305585389059810176539488639444390947099699907424596002414905393427842778094187859764992449565396383471129542818197816195695120715225228023515249902749246068703546189510800383607006572535126780003955237938919642024842060843141184329176556409221492538479319048970969157596490567720452340615746813764533701900891564417052816935374290624864997316451761737845169579480665181743826245198966407270954746461442427988790235075562783746664947912733632469577829222861610766205251772685370823", 10)

			if assert.Equal(t, expectedN, publicKey.N) {

				expectedY := 65537
				assert.Equal(t, expectedY, publicKey.E)
			}
		}
	})
}
