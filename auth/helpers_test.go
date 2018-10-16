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
		publicKey, err := GetCachedIAPJsonWebKey("5PuDqQ")

		expectedX := new(big.Int)
		expectedX, _ = expectedX.SetString("63039552722302819348206200606933896063831791935654903087537299012735652396152", 10)

		expectedY := new(big.Int)
		expectedY, _ = expectedY.SetString("111903340683282919336642547020912254592535012546741170292622780461811063949446", 10)

		if assert.Nil(t, err) {
			assert.Equal(t, elliptic.P256(), publicKey.Curve)
			assert.Equal(t, expectedX, publicKey.X)
			assert.Equal(t, expectedY, publicKey.Y)
		}
	})
}
