package auth

import (
	"encoding/base64"
	"encoding/json"
	"math/big"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestUnmarshalJWK(t *testing.T) {
	t.Run("UnmarshalsJWKResponse", func(t *testing.T) {

		// response from https://www.gstatic.com/iap/verify/public_key-jwk
		jsonResponse := `{ "keys" : [ { "alg" : "ES256", "crv" : "P-256", "kid" : "BOIWdQ", "kty" : "EC", "use" : "sig", "x" : "cqFXSp-TUxZN3uuNFKsz82GwmCs3y-d4ZBr74btAQt8", "y" : "bqqgY8vyllWut5IfjRpWXx8n413PNRONorSFbl93p98" }, { "alg" : "ES256", "crv" : "P-256", "kid" : "5PuDqQ", "kty" : "EC", "use" : "sig", "x" : "i18fVvEF4SW1EAHabO7lYAbOtJeTuxXv1IQOSdQE_Hg", "y" : "92cL23LzmfAH8EPfgySZqoDhxoSxJYekuF2CsWaxiIY" }, { "alg" : "ES256", "crv" : "P-256", "kid" : "s3nVXQ", "kty" : "EC", "use" : "sig", "x" : "UnyNfx2yYjT7IQUQxcV1HhZ_2qKAacAvvQCslOga0hM", "y" : "sVjk-yHj0JzGyIbbkbHPxUy9QbbnNsRVRwKiZjrWA9w" }, { "alg" : "ES256", "crv" : "P-256", "kid" : "rTlk-g", "kty" : "EC", "use" : "sig", "x" : "sm2pfkO5SYLW7vSv3e-XkKBH6SLrxPL5a0Z2MwWfJXY", "y" : "VxQ0E1s8hMLSAAzJkvN4adV6jee1XLtzZreyL4Z6ke4" }, { "alg" : "ES256", "crv" : "P-256", "kid" : "FAWt5w", "kty" : "EC", "use" : "sig", "x" : "8auUAdTS54HmUuIabrTKvWawxmbs81kdbzQMV_Tae0E", "y" : "IS4Ip_KpyeJZJSa8RM5LF4OTrQbsxOtgyI_gnjzdtD4" } ] }`

		// act
		var keysResponse JWKResponse

		// unmarshal json body
		err := json.Unmarshal([]byte(jsonResponse), &keysResponse)

		assert.Nil(t, err)
		assert.Equal(t, 5, len(keysResponse.Keys))
		assert.Equal(t, "ES256", keysResponse.Keys[0].Algorithm)
		assert.Equal(t, "P-256", keysResponse.Keys[0].Curve)
		assert.Equal(t, "BOIWdQ", keysResponse.Keys[0].KeyID)
		assert.Equal(t, "EC", keysResponse.Keys[0].KeyType)
		assert.Equal(t, "sig", keysResponse.Keys[0].PublicKeyUse)
		assert.Equal(t, "cqFXSp-TUxZN3uuNFKsz82GwmCs3y-d4ZBr74btAQt8", keysResponse.Keys[0].X)
		assert.Equal(t, "bqqgY8vyllWut5IfjRpWXx8n413PNRONorSFbl93p98", keysResponse.Keys[0].Y)

		xdata, err := base64.RawURLEncoding.DecodeString(keysResponse.Keys[0].X)

		x := new(big.Int)
		x.SetBytes(xdata)

		xpected := new(big.Int)
		xpected.SetString("51848729579695172483464161482242606434602712503912468683154292353327863710431", 10)

		assert.Nil(t, err)
		assert.Equal(t, xpected, x)

		ydata, err := base64.RawURLEncoding.DecodeString(keysResponse.Keys[0].Y)

		y := new(big.Int)
		y.SetBytes(ydata)

		ypected := new(big.Int)
		ypected.SetString("50055884315100024408193525825372861832752941579251404418416297609341596837855", 10)

		assert.Nil(t, err)
		assert.Equal(t, ypected, y)

	})
}
