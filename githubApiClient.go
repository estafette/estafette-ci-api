package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"strings"
	"time"

	"github.com/dgrijalva/jwt-go"
	"github.com/rs/zerolog/log"
)

// GithubAPIClient is the object to perform Github api calls with
type GithubAPIClient struct {
	githubAppPrivateKeyPath    string
	githubAppID                string
	githubAppOAuthClientID     string
	githubAppOAuthClientSecret string
}

// CreateGithubAPIClient returns an initialized APIClient
func CreateGithubAPIClient(githubAppPrivateKeyPath, githubAppID, githubAppOAuthClientID, githubAppOAuthClientSecret string) *GithubAPIClient {

	return &GithubAPIClient{
		githubAppPrivateKeyPath:    githubAppPrivateKeyPath,
		githubAppID:                githubAppID,
		githubAppOAuthClientID:     githubAppOAuthClientID,
		githubAppOAuthClientSecret: githubAppOAuthClientSecret,
	}
}

func (gh *GithubAPIClient) getGithubAppToken() (githubAppToken string, err error) {

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

func (gh *GithubAPIClient) getGithubAppDetails() (err error) {

	githubAppToken, err := gh.getGithubAppToken()
	if err != nil {
		return
	}

	// curl -i -H "Authorization: Bearer $JWT" -H "Accept: application/vnd.github.machine-man-preview+json" https://api.github.com/app
	_, err = callGithubAPI("GET", "https://api.github.com/app", nil, "Bearer", githubAppToken)
	if err != nil {
		return
	}

	return
}

func (gh *GithubAPIClient) getInstallationToken(installationID int) (accessToken GithubAccessToken, err error) {

	githubAppToken, err := gh.getGithubAppToken()
	if err != nil {
		return
	}

	body, err := callGithubAPI("POST", fmt.Sprintf("https://api.github.com/installations/%v/access_tokens", installationID), nil, "Bearer", githubAppToken)

	// unmarshal json body
	err = json.Unmarshal(body, &accessToken)
	if err != nil {
		return
	}

	return
}

func (gh *GithubAPIClient) getInstallationRepositories(installationID int) (err error) {

	installationToken, err := gh.getInstallationToken(installationID)
	if err != nil {
		return
	}

	_, err = callGithubAPI("GET", "https://api.github.com/installation/repositories", nil, "Token", installationToken.Token)
	if err != nil {
		return
	}

	return
}

func (gh *GithubAPIClient) getAuthenticatedRepositoryURL(installationID int, htmlURL string) (url string, err error) {

	installationToken, err := gh.getInstallationToken(installationID)
	if err != nil {
		return
	}

	url = strings.Replace(htmlURL, "https://github.com", fmt.Sprintf("https://x-access-token:%v@github.com", installationToken.Token), -1)

	return
}

func callGithubAPI(method, url string, params interface{}, authorizationType, token string) (body []byte, err error) {

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
