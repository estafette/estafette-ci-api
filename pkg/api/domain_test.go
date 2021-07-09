package api

import (
	"encoding/base64"
	"encoding/json"
	"math/big"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestUnmarshalIAPJWK(t *testing.T) {
	t.Run("UnmarshalsIAPJWKResponse", func(t *testing.T) {

		// response from https://www.gstatic.com/iap/verify/public_key-jwk
		jsonResponse := `{ "keys" : [ { "alg" : "ES256", "crv" : "P-256", "kid" : "BOIWdQ", "kty" : "EC", "use" : "sig", "x" : "cqFXSp-TUxZN3uuNFKsz82GwmCs3y-d4ZBr74btAQt8", "y" : "bqqgY8vyllWut5IfjRpWXx8n413PNRONorSFbl93p98" }, { "alg" : "ES256", "crv" : "P-256", "kid" : "5PuDqQ", "kty" : "EC", "use" : "sig", "x" : "i18fVvEF4SW1EAHabO7lYAbOtJeTuxXv1IQOSdQE_Hg", "y" : "92cL23LzmfAH8EPfgySZqoDhxoSxJYekuF2CsWaxiIY" }, { "alg" : "ES256", "crv" : "P-256", "kid" : "s3nVXQ", "kty" : "EC", "use" : "sig", "x" : "UnyNfx2yYjT7IQUQxcV1HhZ_2qKAacAvvQCslOga0hM", "y" : "sVjk-yHj0JzGyIbbkbHPxUy9QbbnNsRVRwKiZjrWA9w" }, { "alg" : "ES256", "crv" : "P-256", "kid" : "rTlk-g", "kty" : "EC", "use" : "sig", "x" : "sm2pfkO5SYLW7vSv3e-XkKBH6SLrxPL5a0Z2MwWfJXY", "y" : "VxQ0E1s8hMLSAAzJkvN4adV6jee1XLtzZreyL4Z6ke4" }, { "alg" : "ES256", "crv" : "P-256", "kid" : "FAWt5w", "kty" : "EC", "use" : "sig", "x" : "8auUAdTS54HmUuIabrTKvWawxmbs81kdbzQMV_Tae0E", "y" : "IS4Ip_KpyeJZJSa8RM5LF4OTrQbsxOtgyI_gnjzdtD4" } ] }`

		// act
		var keysResponse IAPJWKResponse

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

		x, err := base64.RawURLEncoding.DecodeString(keysResponse.Keys[0].X)
		xpected, _ := new(big.Int).SetString("51848729579695172483464161482242606434602712503912468683154292353327863710431", 10)

		assert.Nil(t, err)
		assert.Equal(t, xpected, new(big.Int).SetBytes(x))

		y, err := base64.RawURLEncoding.DecodeString(keysResponse.Keys[0].Y)
		ypected, _ := new(big.Int).SetString("50055884315100024408193525825372861832752941579251404418416297609341596837855", 10)

		assert.Nil(t, err)
		assert.Equal(t, ypected, new(big.Int).SetBytes(y))

	})
}

func TestUnmarshalGoogleJWK(t *testing.T) {
	t.Run("UnmarshalsGoogleJWKResponse", func(t *testing.T) {

		// response from https://www.gstatic.com/iap/verify/public_key-jwk
		jsonResponse := `{
			"keys": [
			  {
				"kid": "05a02649a5b45c90fdfe4da1ebefa9c079ab593e",
				"e": "AQAB",
				"kty": "RSA",
				"alg": "RS256",
				"n": "0QW_fsq8WFtNPeOp8cJO1zoToB_E2HBs1Y4ceJB_3qgJmATBCffGwTm7waYEgIlQbJ7fqP1ttgdab-5yQTDGrE51_KS1_3jlB_EDYZPciT3uzHo69BE0v4h9A29fG2MTR1iwkjqDuWE-JN1TNQUeYZ554WYktX1d0qnaiOhM8jNLcuU948LW9d-9xwd7NwnKD_PakCOWRUqXZVYnS7EsTMG4aZpZk0ZB-695tsH-NmwqISPfXI7sEjINRd2PdD9mvs2xAfp-T7eaCV-C3fTfoHDGB3Vwkfn1rG2p-hFB57vzUYB8vEdRgR8ehhEgWndLU6fovvVToWFnPcvkm-ZFxw",
				"use": "sig"
			  },
			  {
				"kid": "2bf8418b2963f366f5fefdd127b2cee07c887e65",
				"e": "AQAB",
				"kty": "RSA",
				"alg": "RS256",
				"n": "uA8MNzSvgJTB_eHWj_HnpGzgtfBzsO4PjdNkEvdETRHEvyqIyqQnACRUNQ9KACnPV3R1M_1VlkJGRJ-xI3By3uylEOh4VagVRlCjLbsmuXburlOLn3TZSkwR7XE3pvVqcypq4nLPdu6foV__wcrLkZPMJzq654vepbOIegx5iVIvV2ilfdqs7VTwHRAUQU6nYfa8jaUwj1H_-zlaaHK-vxm-lWdGjAyiv-xBj5UmY24WtkTuX-MWLvOgbrqcYzMpzEm-LCdBbZR4qjQbWEatRISp4QW31xBZxF2FwMK2YWXDUW_GhXy0hgbsSyX-6jziTDSsulk9SNstSXmYCTZtvw",
				"use": "sig"
			  }
			]
		  }`

		// act
		var keysResponse GoogleJWKResponse

		// unmarshal json body
		err := json.Unmarshal([]byte(jsonResponse), &keysResponse)

		assert.Nil(t, err)
		assert.Equal(t, 2, len(keysResponse.Keys))
		assert.Equal(t, "05a02649a5b45c90fdfe4da1ebefa9c079ab593e", keysResponse.Keys[0].KeyID)
		assert.Equal(t, "AQAB", keysResponse.Keys[0].E)
		assert.Equal(t, "RSA", keysResponse.Keys[0].KeyType)
		assert.Equal(t, "RS256", keysResponse.Keys[0].Algorithm)
		assert.Equal(t, "0QW_fsq8WFtNPeOp8cJO1zoToB_E2HBs1Y4ceJB_3qgJmATBCffGwTm7waYEgIlQbJ7fqP1ttgdab-5yQTDGrE51_KS1_3jlB_EDYZPciT3uzHo69BE0v4h9A29fG2MTR1iwkjqDuWE-JN1TNQUeYZ554WYktX1d0qnaiOhM8jNLcuU948LW9d-9xwd7NwnKD_PakCOWRUqXZVYnS7EsTMG4aZpZk0ZB-695tsH-NmwqISPfXI7sEjINRd2PdD9mvs2xAfp-T7eaCV-C3fTfoHDGB3Vwkfn1rG2p-hFB57vzUYB8vEdRgR8ehhEgWndLU6fovvVToWFnPcvkm-ZFxw", keysResponse.Keys[0].N)
		assert.Equal(t, "sig", keysResponse.Keys[0].PublicKeyUse)

		n, err := base64.RawURLEncoding.DecodeString(keysResponse.Keys[0].N)
		npected, _ := new(big.Int).SetString("26386640196372426380437575256004548936078072361252547077903440302506233434026751142864687086939606473921399288790210323600409676795193267305585389059810176539488639444390947099699907424596002414905393427842778094187859764992449565396383471129542818197816195695120715225228023515249902749246068703546189510800383607006572535126780003955237938919642024842060843141184329176556409221492538479319048970969157596490567720452340615746813764533701900891564417052816935374290624864997316451761737845169579480665181743826245198966407270954746461442427988790235075562783746664947912733632469577829222861610766205251772685370823", 10)

		assert.Nil(t, err)
		assert.Equal(t, npected, new(big.Int).SetBytes(n))

		e := 0
		if keysResponse.Keys[0].E == "AQAB" || keysResponse.Keys[0].E == "AAEAAQ" {
			e = 65537
		}

		epected := 65537
		assert.Equal(t, epected, e)
	})
}

func TestRolesToString(t *testing.T) {
	t.Run("AllRolesCanBeConvertedToString", func(t *testing.T) {

		roles := Roles()

		for i := 0; i < len(roles); i++ {
			r := Role(i)
			roleAsString := r.String()

			assert.Equal(t, roles[i], roleAsString)
		}
	})

	t.Run("AllRolesCanBeConvertedToRole", func(t *testing.T) {

		roles := Roles()

		for _, r := range roles {
			role := ToRole(r)
			assert.NotNil(t, role)
			assert.Equal(t, r, role.String())
		}
	})
}

func TestPermissionsToString(t *testing.T) {
	t.Run("AllPermissionsCanBeConvertedToString", func(t *testing.T) {

		permissions := Permissions()

		for i := 0; i < len(permissions); i++ {
			p := Permission(i)
			permissionAsString := p.String()

			assert.Equal(t, permissions[i], permissionAsString)
		}
	})

	t.Run("AllPermissionsCanBeConvertedToPermission", func(t *testing.T) {

		permissions := Permissions()

		for _, p := range permissions {
			permission := ToPermission(p)
			assert.NotNil(t, permission)
			assert.Equal(t, p, permission.String())
		}
	})
}

func TestFiltersToString(t *testing.T) {
	t.Run("AllFiltersCanBeConvertedToString", func(t *testing.T) {

		filters := Filters()

		for i := 0; i < len(filters); i++ {
			r := FilterType(i)
			roleAsString := r.String()

			assert.Equal(t, filters[i], roleAsString)
		}
	})

	t.Run("AllFiltersCanBeConvertedToFilterType", func(t *testing.T) {

		filters := Filters()

		for _, f := range filters {
			filter := ToFilter(f)
			assert.NotNil(t, filter)
			assert.Equal(t, f, filter.String())
		}
	})
}
