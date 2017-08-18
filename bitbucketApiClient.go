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

// BitbucketAPIClient is the interface for running kubernetes commands specific to this application
type BitbucketAPIClient interface {
	GetAccessToken() (BitbucketAccessToken, error)
	GetAuthenticatedRepositoryURL(BitbucketAccessToken, string) (string, error)
	GetEstafetteManifest(BitbucketAccessToken, BitbucketRepositoryPushEvent) (bool, string, error)
}

type bitbucketAPIClientImpl struct {
	bitbucketAPIKey         string
	bitbucketAppOAuthKey    string
	bitbucketAppOAuthSecret string
}

func newBitbucketAPIClient(bitbucketAPIKey, bitbucketAppOAuthKey, bitbucketAppOAuthSecret string) BitbucketAPIClient {
	return &bitbucketAPIClientImpl{
		bitbucketAPIKey:         bitbucketAPIKey,
		bitbucketAppOAuthKey:    bitbucketAppOAuthKey,
		bitbucketAppOAuthSecret: bitbucketAppOAuthSecret,
	}
}

// GetAccessToken returns an access token to access the Bitbucket api
func (bb *bitbucketAPIClientImpl) GetAccessToken() (accessToken BitbucketAccessToken, err error) {

	// track call via prometheus
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

// GetAuthenticatedRepositoryURL returns a repository url with a time-limited access token embedded
func (bb *bitbucketAPIClientImpl) GetAuthenticatedRepositoryURL(accessToken BitbucketAccessToken, htmlURL string) (url string, err error) {

	url = strings.Replace(htmlURL, "https://bitbucket.org", fmt.Sprintf("https://x-token-auth:%v@bitbucket.org", accessToken.AccessToken), -1)

	return
}

func (bb *bitbucketAPIClientImpl) GetEstafetteManifest(accessToken BitbucketAccessToken, pushEvent BitbucketRepositoryPushEvent) (exists bool, manifest string, err error) {

	// track call via prometheus
	outgoingAPIRequestTotal.With(prometheus.Labels{"target": "bitbucket"}).Inc()

	// create client, in order to add headers
	client := &http.Client{}
	request, err := http.NewRequest("GET", fmt.Sprintf("https://api.bitbucket.org/1.0/repositories/%v/raw/%v/.estafette.yaml", pushEvent.Repository.FullName, pushEvent.Push.Changes[0].New.Target.Hash), nil)
	if err != nil {
		return
	}

	// add headers
	request.Header.Add("Authorization", fmt.Sprintf("Bearer %v", accessToken.AccessToken))

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

	if response.StatusCode != http.StatusNotFound {
		exists = true
		manifest = string(body)
	}

	return
}
