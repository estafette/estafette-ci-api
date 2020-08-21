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
	"github.com/estafette/estafette-ci-api/api"
	"github.com/opentracing-contrib/go-stdlib/nethttp"
	"github.com/opentracing/opentracing-go"
	"github.com/rs/zerolog/log"
	"github.com/sethgrid/pester"
)

// Client is the interface for communicating with the github api
type Client interface {
	GetGithubAppToken(ctx context.Context) (token string, err error)
	GetInstallationID(ctx context.Context, repoOwner string) (installationID int, err error)
	GetInstallationToken(ctx context.Context, installationID int) (token AccessToken, err error)
	GetAuthenticatedRepositoryURL(ctx context.Context, accesstoken AccessToken, htmlURL string) (url string, err error)
	GetEstafetteManifest(ctx context.Context, accesstoken AccessToken, event PushEvent) (valid bool, manifest string, err error)
	JobVarsFunc(ctx context.Context) func(ctx context.Context, repoSource, repoOwner, repoName string) (token string, url string, err error)
}

// NewClient creates an githubapi.Client to communicate with the Github api
func NewClient(config *api.APIConfig) Client {
	if config == nil || config.Integrations == nil || config.Integrations.Github == nil || !config.Integrations.Github.Enable {
		return &client{
			enabled: false,
		}
	}

	return &client{
		enabled: true,
		config:  config,
	}
}

type client struct {
	enabled bool
	config  *api.APIConfig
}

// GetGithubAppToken returns a Github app token with which to retrieve an installation token
func (c *client) GetGithubAppToken(ctx context.Context) (githubAppToken string, err error) {

	// https://developer.github.com/apps/building-integrations/setting-up-and-registering-github-apps/about-authentication-options-for-github-apps/

	// load private key from pem file
	pemFileByteArray, err := ioutil.ReadFile(c.config.Integrations.Github.PrivateKeyPath)
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
		"iss": c.config.Integrations.Github.AppID,
	})

	// sign and get the complete encoded token as a string using the private key
	githubAppToken, err = token.SignedString(privateKey)
	if err != nil {
		return
	}

	return
}

// GetInstallationID returns the id for an installation of a Github app
func (c *client) GetInstallationID(ctx context.Context, repoOwner string) (installationID int, err error) {

	githubAppToken, err := c.GetGithubAppToken(ctx)
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

	_, body, err := c.callGithubAPI(ctx, "GET", "https://api.github.com/app/installations", nil, "Bearer", githubAppToken)

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

	return installationID, fmt.Errorf("Github installation of app %v with account login %v can't be found", c.config.Integrations.Github.AppID, repoOwner)
}

// GetInstallationToken returns an access token for an installation of a Github app
func (c *client) GetInstallationToken(ctx context.Context, installationID int) (accessToken AccessToken, err error) {

	githubAppToken, err := c.GetGithubAppToken(ctx)
	if err != nil {
		return
	}

	_, body, err := c.callGithubAPI(ctx, "POST", fmt.Sprintf("https://api.github.com/app/installations/%v/access_tokens", installationID), nil, "Bearer", githubAppToken)

	// unmarshal json body
	err = json.Unmarshal(body, &accessToken)
	if err != nil {
		return
	}

	return
}

// GetAuthenticatedRepositoryURL returns a repository url with a time-limited access token embedded
func (c *client) GetAuthenticatedRepositoryURL(ctx context.Context, accessToken AccessToken, htmlURL string) (url string, err error) {

	url = strings.Replace(htmlURL, "https://github.com", fmt.Sprintf("https://x-access-token:%v@github.com", accessToken.Token), -1)

	return
}

func (c *client) GetEstafetteManifest(ctx context.Context, accessToken AccessToken, pushEvent PushEvent) (exists bool, manifest string, err error) {

	// https://developer.github.com/v3/repos/contents/

	statusCode, body, err := c.callGithubAPI(ctx, "GET", fmt.Sprintf("https://api.github.com/repos/%v/contents/.estafette.yaml?ref=%v", pushEvent.Repository.FullName, pushEvent.After), nil, "token", accessToken.Token)
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

// JobVarsFunc returns a function that can get an access token and authenticated url for a repository
func (c *client) JobVarsFunc(ctx context.Context) func(context.Context, string, string, string) (string, string, error) {
	return func(ctx context.Context, repoSource, repoOwner, repoName string) (token string, url string, err error) {
		// get installation id with just the repo owner
		installationID, err := c.GetInstallationID(ctx, repoOwner)
		if err != nil {
			return "", "", err
		}

		// get access token
		accessToken, err := c.GetInstallationToken(ctx, installationID)
		if err != nil {
			return "", "", err
		}

		// get authenticated url for the repository
		url, err = c.GetAuthenticatedRepositoryURL(ctx, accessToken, fmt.Sprintf("https://%v/%v/%v", repoSource, repoOwner, repoName))
		if err != nil {
			return "", "", err
		}

		return accessToken.Token, url, nil
	}
}

func (c *client) callGithubAPI(ctx context.Context, method, url string, params interface{}, authorizationType, token string) (statusCode int, body []byte, err error) {

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

	span := opentracing.SpanFromContext(ctx)
	var ht *nethttp.Tracer
	if span != nil {
		// add tracing context
		request = request.WithContext(opentracing.ContextWithSpan(request.Context(), span))

		// collect additional information on setting up connections
		request, ht = nethttp.TraceRequest(span.Tracer(), request)
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
	if ht != nil {
		ht.Finish()
	}

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
