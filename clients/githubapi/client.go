package githubapi

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
	githubdom "github.com/estafette/estafette-ci-api/domain/github"
	"github.com/opentracing-contrib/go-stdlib/nethttp"
	"github.com/opentracing/opentracing-go"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/rs/zerolog/log"
	"github.com/sethgrid/pester"
)

// Client is the interface for communicating with the github api
type Client interface {
	GetGithubAppToken(ctx context.Context) (string, error)
	GetInstallationID(context.Context, string) (int, error)
	GetInstallationToken(context.Context, int) (githubdom.AccessToken, error)
	GetAuthenticatedRepositoryURL(githubdom.AccessToken, string) (string, error)
	GetEstafetteManifest(context.Context, githubdom.AccessToken, githubdom.PushEvent) (bool, string, error)
	callGithubAPI(opentracing.Span, string, string, interface{}, string, string) (int, []byte, error)

	JobVarsFunc() func(context.Context, string, string, string) (string, string, error)
}

type client struct {
	config                          config.GithubConfig
	prometheusOutboundAPICallTotals *prometheus.CounterVec
	tracer                          opentracing.Tracer
}

// NewClient creates an github.Client to communicate with the Github api
func NewClient(config config.GithubConfig, prometheusOutboundAPICallTotals *prometheus.CounterVec) Client {
	return &client{
		config:                          config,
		prometheusOutboundAPICallTotals: prometheusOutboundAPICallTotals,
	}
}

// GetGithubAppToken returns a Github app token with which to retrieve an installation token
func (gh *client) GetGithubAppToken(ctx context.Context) (githubAppToken string, err error) {

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
func (gh *client) GetInstallationID(ctx context.Context, repoOwner string) (installationID int, err error) {

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

	_, body, err := gh.callGithubAPI(span, "GET", "https://api.github.com/app/installations", nil, "Bearer", githubAppToken)

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
func (gh *client) GetInstallationToken(ctx context.Context, installationID int) (accessToken githubdom.AccessToken, err error) {

	span, ctx := opentracing.StartSpanFromContext(ctx, "GithubApi::GetInstallationToken")
	defer span.Finish()

	githubAppToken, err := gh.GetGithubAppToken(ctx)
	if err != nil {
		return
	}

	_, body, err := gh.callGithubAPI(span, "POST", fmt.Sprintf("https://api.github.com/installations/%v/access_tokens", installationID), nil, "Bearer", githubAppToken)

	// unmarshal json body
	err = json.Unmarshal(body, &accessToken)
	if err != nil {
		return
	}

	return
}

// GetAuthenticatedRepositoryURL returns a repository url with a time-limited access token embedded
func (gh *client) GetAuthenticatedRepositoryURL(accessToken githubdom.AccessToken, htmlURL string) (url string, err error) {

	url = strings.Replace(htmlURL, "https://github.com", fmt.Sprintf("https://x-access-token:%v@github.com", accessToken.Token), -1)

	return
}

func (gh *client) GetEstafetteManifest(ctx context.Context, accessToken githubdom.AccessToken, pushEvent githubdom.PushEvent) (exists bool, manifest string, err error) {

	span, ctx := opentracing.StartSpanFromContext(ctx, "GithubApi::GetEstafetteManifest")
	defer span.Finish()

	// https://developer.github.com/v3/repos/contents/

	statusCode, body, err := gh.callGithubAPI(span, "GET", fmt.Sprintf("https://api.github.com/repos/%v/contents/.estafette.yaml?ref=%v", pushEvent.Repository.FullName, pushEvent.After), nil, "token", accessToken.Token)
	if err != nil {
		return
	}

	if statusCode == http.StatusNotFound {
		return
	}

	var content githubdom.RepositoryContent

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
func (gh *client) JobVarsFunc() func(context.Context, string, string, string) (string, string, error) {
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

func (gh *client) callGithubAPI(span opentracing.Span, method, url string, params interface{}, authorizationType, token string) (statusCode int, body []byte, err error) {

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
	client := pester.NewExtendedClient(&http.Client{Transport: &nethttp.Transport{}})
	client.MaxRetries = 3
	client.Backoff = pester.ExponentialJitterBackoff
	client.KeepLog = true
	client.Timeout = time.Second * 10
	request, err := http.NewRequest(method, url, requestBody)
	if err != nil {
		return
	}

	// add tracing context
	request = request.WithContext(opentracing.ContextWithSpan(request.Context(), span))

	// collect additional information on setting up connections
	request, ht := nethttp.TraceRequest(span.Tracer(), request)

	// add headers
	request.Header.Add("Authorization", fmt.Sprintf("%v %v", authorizationType, token))
	request.Header.Add("Accept", "application/vnd.github.machine-man-preview+json")

	// perform actual request
	response, err := client.Do(request)
	if err != nil {
		return
	}

	defer response.Body.Close()
	ht.Finish()

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
