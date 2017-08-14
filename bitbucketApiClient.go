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

	"github.com/prometheus/client_golang/prometheus"
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

	outgoingAPIRequestTotal.With(prometheus.Labels{"target": "bitbucket"}).Inc()

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
