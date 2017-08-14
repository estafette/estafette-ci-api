package main

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"
)

// BitbucketAPIClient is the object to perform Bitbucket api calls with
type BitbucketAPIClient struct {
	bitbucketAPIKey         string
	bitbucketAppOAuthKey    string
	bitbucketAppOAuthSecret string
}

// CreateBitbucketAPIClient returns an initialized APIClient
func CreateBitbucketAPIClient(bitbucketAPIKey, bitbucketAppOAuthKey, bitbucketAppOAuthSecret string) *BitbucketAPIClient {

	return &BitbucketAPIClient{
		bitbucketAPIKey:         bitbucketAPIKey,
		bitbucketAppOAuthKey:    bitbucketAppOAuthKey,
		bitbucketAppOAuthSecret: bitbucketAppOAuthSecret,
	}
}

func (bb *BitbucketAPIClient) getAccessToken() (accessToken BitbucketAccessToken, err error) {

	//curl -X POST -u "cEHbmCvrCbL54afgxF:6wwX7yvXxcrPVkEChebzcCTNPtdKmsp7" https://bitbucket.org/site/oauth2/access_token -d grant_type=client_credentials

	// {"access_token": "dxHqk3Z8-XurpbRbheGTfNgADuChMtY6hE78-nnfHWH8vWnPBfRjbM3LjwolZW0MxZNVG1YtjBffwMOfgEs=", "scopes": "pipeline:variable webhook snippet:write wiki issue:write pullrequest:write repositor
	// y:delete repository:admin project:write team:write account:write", "expires_in": 3600, "refresh_toke
	// n": "BfjCrRUm4vB4Ld2AJc", "token_type": "bearer"}

	basicAuthenticationToken := base64.StdEncoding.EncodeToString([]byte(fmt.Sprintf("%v:%v", bb.bitbucketAppOAuthKey, bb.bitbucketAppOAuthSecret)))

	// form values
	data := url.Values{}
	data.Set("grant_type", "client_credentials")

	// create client, in order to add headers
	client := &http.Client{}
	request, err := http.NewRequest("POST", "https://bitbucket.org/site/oauth2/access_token", bytes.NewBufferString(data.Encode()))
	if err != nil {
		return
	}

	// add headers
	request.Header.Add("Authorization", fmt.Sprintf("%v %v", "Basic", basicAuthenticationToken))
	request.Header.Add("Content-Type", "application/x-www-form-urlencoded")

	// perform actual request
	response, err := client.Do(request)
	if err != nil {
		return
	}

	defer response.Body.Close()

	body, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return
	}

	// unmarshal json body
	err = json.Unmarshal(body, &accessToken)
	if err != nil {
		return
	}

	return
}

func (bb *BitbucketAPIClient) getAuthenticatedRepositoryURL(htmlURL string) (url string, err error) {

	accessToken, err := bb.getAccessToken()
	if err != nil {
		return
	}

	url = strings.Replace(htmlURL, "https://bitbucket.org", fmt.Sprintf("https://x-token-auth:%v@bitbucket.org", accessToken.AccessToken), -1)

	return
}
