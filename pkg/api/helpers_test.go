package api

import (
	"io"
	"regexp"
	"testing"

	"github.com/sethgrid/pester"
	"github.com/stretchr/testify/assert"
)

func TestRetrievingGoogleJSONWebKeys(t *testing.T) {
	t.Run("ReturnsKeyByKeyID", func(t *testing.T) {

		// get kid from https://www.googleapis.com/oauth2/v3/certs
		response, err := pester.Get("https://www.googleapis.com/oauth2/v3/certs")
		if !assert.Nil(t, err, "Did not expect error %v", err) {
			return
		}

		defer response.Body.Close()

		body, err := io.ReadAll(response.Body)
		if !assert.Nil(t, err, "Did not expect error %v", err) {
			return
		}

		re := regexp.MustCompile(`"kid": "([a-z0-9]+)"`)
		match := re.FindStringSubmatch(string(body))

		if !assert.Equal(t, 2, len(match)) {
			return
		}

		kid := match[1]

		// act
		publicKey, err := GetCachedGoogleJWK(kid)

		if assert.Nil(t, err) {
			expectedY := 65537
			assert.Equal(t, expectedY, publicKey.E)
		}
	})
}
