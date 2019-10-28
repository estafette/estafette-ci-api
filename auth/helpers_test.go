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
		publicKey, err := GetCachedGoogleJWK("8a63fe71e53067524cbbc6a3a58463b3864c0787")

		if assert.Nil(t, err) {
			expectedN, _ := new(big.Int).SetString("25462559455305279680930073421592071197511990287506517669360420279184745872586657725021452515269520146350819564451481782481370877474015012644315401094281618027294450648211279315323383963323280881195199616541480725506185280537045392579172217469708416229939638263143347431343065229798478771677971367582487218074544084254020686612325960018034590870438044226067081637870371374187583678628509611893006399564569618872788960399498828632291292980618664016645484666281281482487267313176806989149990379724322487728594770747718601853180225819982327160867532914586273610030333603460178770482455846057212938295965297224366206842521", 10)

			if assert.Equal(t, expectedN, publicKey.N) {

				expectedY := 65537
				assert.Equal(t, expectedY, publicKey.E)
			}
		}
	})
}
