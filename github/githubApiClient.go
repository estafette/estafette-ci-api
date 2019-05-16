package github

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"strings"
	"time"

	"github.com/dgrijalva/jwt-go"
	"github.com/estafette/estafette-ci-api/config"
	ghcontracts "github.com/estafette/estafette-ci-api/github/contracts"
	"github.com/opentracing/opentracing-go"
	"github.com/rs/zerolog/log"
	"github.com/sethgrid/pester"

	"github.com/prometheus/client_golang/prometheus"
)

// APIClient is the interface for running kubernetes commands specific to this application
type APIClient interface {
	GetGithubAppToken(ctx context.Context) (string, error)
	GetInstallationID(context.Context, string) (int, error)
	GetInstallationToken(context.Context, int) (ghcontracts.AccessToken, error)
	GetAuthenticatedRepositoryURL(ghcontracts.AccessToken, string) (string, error)
	GetEstafetteManifest(context.Context, ghcontracts.AccessToken, ghcontracts.PushEvent) (bool, string, error)
	callGithubAPI(string, string, interface{}, string, string) (int, []byte, error)

	JobVarsFunc() func(context.Context, string, string, string) (string, string, error)
}

type apiClientImpl struct {
	config                          config.GithubConfig
	prometheusOutboundAPICallTotals *prometheus.CounterVec
	tracer                          opentracing.Tracer
}

// NewGithubAPIClient creates an github.APIClient to communicate with the Github api
func NewGithubAPIClient(config config.GithubConfig, prometheusOutboundAPICallTotals *prometheus.CounterVec) APIClient {
	return &apiClientImpl{
		config:                          config,
		prometheusOutboundAPICallTotals: prometheusOutboundAPICallTotals,
	}
}

// GetGithubAppToken returns a Github app token with which to retrieve an installation token
func (gh *apiClientImpl) GetGithubAppToken(ctx context.Context) (githubAppToken string, err error) {

	span, ctx := opentracing.StartSpanFromContext(ctx, "GithubApi::GetGithubAppToken")
	defer span.Finish()

	// https://developer.github.com/apps/building-integrations/setting-up-and-registering-github-apps/about-authentication-options-for-github-apps/

	// load private key from pem file
	pemFileByteArray, err := ioutil.ReadFile(gh.config.PrivateKeyPath)
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
		"iss": gh.config.AppID,
	})

	// sign and get the complete encoded token as a string using the private key
	githubAppToken, err = token.SignedString(privateKey)
	if err != nil {
		return
	}

	return
}

// GetInstallationID returns the id for an installation of a Github app
func (gh *apiClientImpl) GetInstallationID(ctx context.Context, repoOwner string) (installationID int, err error) {

	span, ctx := opentracing.StartSpanFromContext(ctx, "GithubApi::GetInstallationID")
	defer span.Finish()

	githubAppToken, err := gh.GetGithubAppToken(ctx)
	if err != nil {
		return
	}

	type InstallationAccount struct {
		Login string `json:"login"`
	}

	type InstallationResponse struct {
		ID      int                 `json:"id"`
		Account InstallationAccount `json:"account"`
	}

	_, body, err := gh.callGithubAPI("GET", "https://api.github.com/app/installations", nil, "Bearer", githubAppToken)

	var installations []InstallationResponse

	// unmarshal json body
	err = json.Unmarshal(body, &installations)
	if err != nil {
		return
	}

	// find installation matching repoOwner
	for _, installation := range installations {
		if installation.Account.Login == repoOwner {
			installationID = installation.ID
			return
		}
	}

	return installationID, fmt.Errorf("Github installation of app %v with account login %v can't be found", gh.config.AppID, repoOwner)
}

// GetInstallationToken returns an access token for an installation of a Github app
func (gh *apiClientImpl) GetInstallationToken(ctx context.Context, installationID int) (accessToken ghcontracts.AccessToken, err error) {

	span, ctx := opentracing.StartSpanFromContext(ctx, "GithubApi::GetInstallationToken")
	defer span.Finish()

	githubAppToken, err := gh.GetGithubAppToken(ctx)
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
func (gh *apiClientImpl) GetAuthenticatedRepositoryURL(accessToken ghcontracts.AccessToken, htmlURL string) (url string, err error) {

	url = strings.Replace(htmlURL, "https://github.com", fmt.Sprintf("https://x-access-token:%v@github.com", accessToken.Token), -1)

	return
}

func (gh *apiClientImpl) GetEstafetteManifest(ctx context.Context, accessToken ghcontracts.AccessToken, pushEvent ghcontracts.PushEvent) (exists bool, manifest string, err error) {

	span, ctx := opentracing.StartSpanFromContext(ctx, "GithubApi::GetEstafetteManifest")
	defer span.Finish()

	// https://developer.github.com/v3/repos/contents/

	statusCode, body, err := gh.callGithubAPI("GET", fmt.Sprintf("https://api.github.com/repos/%v/contents/.estafette.yaml?ref=%v", pushEvent.Repository.FullName, pushEvent.After), nil, "token", accessToken.Token)
	if err != nil {
		return
	}

	if statusCode == http.StatusNotFound {
		return
	}

	var content ghcontracts.RepositoryContent

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

// JobVarsFunc returns a function that can get an access token and authenticated url for a repository
func (gh *apiClientImpl) JobVarsFunc() func(context.Context, string, string, string) (string, string, error) {
	return func(ctx context.Context, repoSource, repoOwner, repoName string) (token string, url string, err error) {
		// get installation id with just the repo owner
		installationID, err := gh.GetInstallationID(ctx, repoOwner)
		if err != nil {
			return "", "", err
		}

		// get access token
		accessToken, err := gh.GetInstallationToken(ctx, installationID)
		if err != nil {
			return "", "", err
		}

		// get authenticated url for the repository
		url, err = gh.GetAuthenticatedRepositoryURL(accessToken, fmt.Sprintf("https://%v/%v/%v", repoSource, repoOwner, repoName))
		if err != nil {
			return "", "", err
		}

		return accessToken.Token, url, nil
	}
}

func (gh *apiClientImpl) callGithubAPI(method, url string, params interface{}, authorizationType, token string) (statusCode int, body []byte, err error) {

	// track call via prometheus
	gh.prometheusOutboundAPICallTotals.With(prometheus.Labels{"target": "github"}).Inc()

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
	client := pester.New()
	client.MaxRetries = 3
	client.Backoff = pester.ExponentialJitterBackoff
	client.KeepLog = true
	client.Timeout = time.Second * 10
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

	return
}
