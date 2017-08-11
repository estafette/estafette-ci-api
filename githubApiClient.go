package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"time"

	"github.com/dgrijalva/jwt-go"
	"github.com/rs/zerolog/log"
)

// GithubAPIClient is the object to perform Github api calls with
type GithubAPIClient struct {
	githubAppPrivateKeyPath    string
	githubAppOAuthClientID     string
	githubAppOAuthClientSecret string
}

// CreateGithubAPIClient returns an initialized APIClient
func CreateGithubAPIClient(githubAppPrivateKeyPath, githubAppOAuthClientID, githubAppOAuthClientSecret string) *GithubAPIClient {

	return &GithubAPIClient{
		githubAppPrivateKeyPath:    githubAppPrivateKeyPath,
		githubAppOAuthClientID:     githubAppOAuthClientID,
		githubAppOAuthClientSecret: githubAppOAuthClientSecret,
	}
}

func (gh *GithubAPIClient) getGithubAppToken() (githubAppToken string, err error) {

	// https://developer.github.com/apps/building-integrations/setting-up-and-registering-github-apps/about-authentication-options-for-github-apps/

	// load private key from pem file
	log.Debug().Msgf("Reading pem file at %v...", gh.githubAppPrivateKeyPath)
	pemFileByteArray, err := ioutil.ReadFile(gh.githubAppPrivateKeyPath)
	if err != nil {
		return
	}

	log.Debug().Msg("Reading private key from pem file...")
	privateKey, err := jwt.ParseRSAPrivateKeyFromPEM(pemFileByteArray)
	if err != nil {
		return
	}

	// create a new token object, specifying signing method and the claims you would like it to contain.
	log.Debug().Msg("Creating json web token...")
	epoch := time.Now().Unix()
	token := jwt.NewWithClaims(jwt.SigningMethodRS256, jwt.MapClaims{
		// issued at time
		"iat": epoch,
		// JWT expiration time (10 minute maximum)
		"exp": epoch + 600,
		// GitHub App's identifier
		"iss": githubAppOAuthClientID,
	})

	// sign and get the complete encoded token as a string using the private key
	log.Debug().Msg("Signing json web token...")
	githubAppToken, err = token.SignedString(&privateKey)
	if err != nil {
		return
	}

	return
}

func (gh *GithubAPIClient) getGithubAppDetails() {

	githubAppToken, err := gh.getGithubAppToken()
	if err != nil {
		log.Error().Err(err).
			Msg("Creating Github App web token failed")

		return
	}

	// curl -i -H "Authorization: Bearer $JWT" -H "Accept: application/vnd.github.machine-man-preview+json" https://api.github.com/app
	callGithubAPI("GET", "https://api.github.com/app", nil, githubAppToken)
}

func (gh *GithubAPIClient) getInstallationToken(installationID int) (installationToken string, err error) {

	githubAppToken, err := gh.getGithubAppToken()
	if err != nil {
		log.Error().Err(err).
			Msg("Creating Github App web token failed")

		return
	}

	callGithubAPI("GET", fmt.Sprintf("https://api.github.com/installations/%v/access_tokens", installationID), nil, githubAppToken)

	return
}

func (gh *GithubAPIClient) getInstallationRepositories(installationID int) {

	installationToken, err := gh.getInstallationToken(installationID)
	if err != nil {
		log.Error().Err(err).
			Msg("Retrieving Github App installation web token failed")

		return
	}

	callGithubAPI("GET", "https://api.github.com/installation/repositories", nil, installationToken)
}

func (gh *GithubAPIClient) getAuthenticatedRepositoryURL(installationID int, htmlURL string) (url string, err error) {

	// installationToken, err := gh.getInstallationToken(installationID)
	// if err != nil {
	// 	return
	// }

	// git clone https://x-access-token:<token>@github.com/owner/repo.git

	return
}

func callGithubAPI(method, url string, params interface{}, token string) (body []byte, err error) {

	// convert params to json if they're present
	var requestBody io.Reader
	if params != nil {
		data, err := json.Marshal(params)
		if err != nil {
			return body, err
		}
		requestBody = bytes.NewReader(data)
	}

	// create client, in order to add headers
	client := &http.Client{}
	request, err := http.NewRequest(method, url, requestBody)
	if err != nil {
		log.Error().Err(err).
			Msg("Creating http client failed")

		return
	}

	// add headers
	request.Header.Add("Authorization", token)
	request.Header.Add("Accept", "application/vnd.github.machine-man-preview+json")

	// perform actual request
	response, err := client.Do(request)
	if err != nil {
		log.Error().Err(err).
			Msg("Performing Github api call failed")

		return
	}

	defer response.Body.Close()

	body, err = ioutil.ReadAll(response.Body)
	if err != nil {
		log.Error().Err(err).
			Msg("Reading Github api call response failed")

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
