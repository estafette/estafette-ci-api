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
		publicKey, err := GetCachedGoogleJWK("ee4dbd06c06683cb48dddca6b88c3e473b6915b9")

		if assert.Nil(t, err) {
			expectedN, _ := new(big.Int).SetString("23332795212837634416668316788618293514624977485046284523355291319719316459744992776027918124607598983066115641182391433293502505952852935695233060783240152404385233804803324409751209054928504411371186897563514270555393844016136651982022364486299909573192301135335078760522324583698875401836998572033790723155309518047958323029121104580650003753076668613065417138394045360491893917054974866470981561787999574559867625761176549619742599423736127776280961965814832756155123603780206912576225078865143248801623624336929337599651210938477999271538277078084581517783006642773197285233845846634527942398306300897527907351269", 10)

			if assert.Equal(t, expectedN, publicKey.N) {

				expectedY := 65537
				assert.Equal(t, expectedY, publicKey.E)
			}
		}
	})
}
