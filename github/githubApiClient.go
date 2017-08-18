package github

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"strings"
	"time"

	"github.com/dgrijalva/jwt-go"
	"github.com/rs/zerolog/log"

	"github.com/prometheus/client_golang/prometheus"
)

// APIClient is the interface for running kubernetes commands specific to this application
type APIClient interface {
	GetGithubAppToken() (string, error)
	GetInstallationToken(int) (AccessToken, error)
	GetAuthenticatedRepositoryURL(AccessToken, string) (string, error)
	GetEstafetteManifest(AccessToken, PushEvent) (bool, string, error)
	callGithubAPI(string, string, interface{}, string, string) (int, []byte, error)
}

type apiClientImpl struct {
	githubAppPrivateKeyPath         string
	githubAppID                     string
	githubAppOAuthClientID          string
	githubAppOAuthClientSecret      string
	PrometheusOutboundAPICallTotals *prometheus.CounterVec
}

// NewGithubAPIClient creates an github.APIClient to communicate with the Github api
func NewGithubAPIClient(githubAppPrivateKeyPath, githubAppID, githubAppOAuthClientID, githubAppOAuthClientSecret string, prometheusOutboundAPICallTotals *prometheus.CounterVec) APIClient {
	return &apiClientImpl{
		githubAppPrivateKeyPath:         githubAppPrivateKeyPath,
		githubAppID:                     githubAppID,
		githubAppOAuthClientID:          githubAppOAuthClientID,
		githubAppOAuthClientSecret:      githubAppOAuthClientSecret,
		PrometheusOutboundAPICallTotals: prometheusOutboundAPICallTotals,
	}
}

// GetGithubAppToken returns a Github app token with which to retrieve an installation token
func (gh *apiClientImpl) GetGithubAppToken() (githubAppToken string, err error) {

	// https://developer.github.com/apps/building-integrations/setting-up-and-registering-github-apps/about-authentication-options-for-github-apps/

	// load private key from pem file
	pemFileByteArray, err := ioutil.ReadFile(gh.githubAppPrivateKeyPath)
	if err != nil {
		return
	}
	privateKey, err := jwt.ParseRSAPrivateKeyFromPEM(pemFileByteArray)
	if err != nil {
		return
	}

	// create a new token object, specifying signing method and the claims you would like it to contain.
	epoch := time.Now().Unix()
	token := jwt.NewWithClaims(jwt.SigningMethodRS256, jwt.MapClaims{
		// issued at time
		"iat": epoch,
		// JWT expiration time (10 minute maximum)
		"exp": epoch + 500,
		// GitHub App's identifier
		"iss": gh.githubAppID,
	})

	// sign and get the complete encoded token as a string using the private key
	githubAppToken, err = token.SignedString(privateKey)
	if err != nil {
		return
	}

	return
}

// GetInstallationToken returns an access token for an installation of a Github app
func (gh *apiClientImpl) GetInstallationToken(installationID int) (accessToken AccessToken, err error) {

	githubAppToken, err := gh.GetGithubAppToken()
	if err != nil {
		return
	}

	_, body, err := gh.callGithubAPI("POST", fmt.Sprintf("https://api.github.com/installations/%v/access_tokens", installationID), nil, "Bearer", githubAppToken)

	// unmarshal json body
	err = json.Unmarshal(body, &accessToken)
	if err != nil {
		return
	}

	return
}

// GetAuthenticatedRepositoryURL returns a repository url with a time-limited access token embedded
func (gh *apiClientImpl) GetAuthenticatedRepositoryURL(accessToken AccessToken, htmlURL string) (url string, err error) {

	url = strings.Replace(htmlURL, "https://github.com", fmt.Sprintf("https://x-access-token:%v@github.com", accessToken.Token), -1)

	return
}

func (gh *apiClientImpl) GetEstafetteManifest(accessToken AccessToken, pushEvent PushEvent) (exists bool, manifest string, err error) {

	// https://developer.github.com/v3/repos/contents/

	statusCode, body, err := gh.callGithubAPI("GET", fmt.Sprintf("https://api.github.com/repos/%v/contents/.estafette.yaml?ref=%v", pushEvent.Repository.FullName, pushEvent.After), nil, "token", accessToken.Token)
	if err != nil {
		return
	}

	if statusCode == http.StatusNotFound {
		return
	}

	var content RepositoryContent

	// unmarshal json body
	err = json.Unmarshal(body, &content)
	if err != nil {
		return
	}

	if content.Type == "file" && content.Encoding == "base64" {
		manifest = content.Content

		data, err := base64.StdEncoding.DecodeString(content.Content)
		if err != nil {
			return false, "", err
		}
		exists = true
		manifest = string(data)
	}

	return
}

func (gh *apiClientImpl) callGithubAPI(method, url string, params interface{}, authorizationType, token string) (statusCode int, body []byte, err error) {

	// track call via prometheus
	gh.PrometheusOutboundAPICallTotals.With(prometheus.Labels{"target": "github"}).Inc()

	// convert params to json if they're present
	var requestBody io.Reader
	if params != nil {
		data, err := json.Marshal(params)
		if err != nil {
			return 0, body, err
		}
		requestBody = bytes.NewReader(data)
	}

	// create client, in order to add headers
	client := &http.Client{}
	request, err := http.NewRequest(method, url, requestBody)
	if err != nil {
		return
	}

	// add headers
	request.Header.Add("Authorization", fmt.Sprintf("%v %v", authorizationType, token))
	request.Header.Add("Accept", "application/vnd.github.machine-man-preview+json")

	// perform actual request
	response, err := client.Do(request)
	if err != nil {
		return
	}

	defer response.Body.Close()

	statusCode = response.StatusCode

	body, err = ioutil.ReadAll(response.Body)
	if err != nil {
		return
	}

	// unmarshal json body
	var b interface{}
	err = json.Unmarshal(body, &b)
	if err != nil {
		log.Error().Err(err).
			Str("url", url).
			Str("requestMethod", method).
			Interface("requestBody", params).
			Interface("requestHeaders", request.Header).
			Interface("responseHeaders", response.Header).
			Str("responseBody", string(body)).
			Msg("Deserializing response for '%v' Github api call failed")

		return
	}

	log.Debug().
		Str("url", url).
		Str("requestMethod", method).
		Interface("requestBody", params).
		Interface("requestHeaders", request.Header).
		Interface("responseHeaders", response.Header).
		Interface("responseBody", b).
		Msgf("Received response for '%v' Github api call...", url)

	return
}
