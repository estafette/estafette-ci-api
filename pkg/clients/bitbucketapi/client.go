package bitbucketapi

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"time"

	"github.com/dgrijalva/jwt-go"
	"github.com/estafette/estafette-ci-api/pkg/api"
	"github.com/opentracing-contrib/go-stdlib/nethttp"
	"github.com/opentracing/opentracing-go"
	"github.com/rs/zerolog/log"
	"github.com/sethgrid/pester"
)

// Client is the interface for communicating with the bitbucket api
//go:generate mockgen -package=bitbucketapi -destination ./mock.go -source=client.go
type Client interface {
	GetAccessToken(ctx context.Context) (accesstoken AccessToken, err error)
	GetEstafetteManifest(ctx context.Context, accesstoken AccessToken, event RepositoryPushEvent) (valid bool, manifest string, err error)
	JobVarsFunc(ctx context.Context) func(ctx context.Context, repoSource, repoOwner, repoName string) (token string, err error)
	GenerateJWT() (tokenString string, err error)
}

// NewClient returns a new bitbucket.Client
func NewClient(config *api.APIConfig) Client {
	if config == nil || config.Integrations == nil || config.Integrations.Bitbucket == nil || !config.Integrations.Bitbucket.Enable {
		return &client{
			enabled: false,
			config:  config,
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

// GetAccessToken returns an access token to access the Bitbucket api
func (c *client) GetAccessToken(ctx context.Context) (accesstoken AccessToken, err error) {

	jtwToken, err := c.GenerateJWT()
	if err != nil {
		return
	}

	// form values
	data := url.Values{}
	data.Set("grant_type", "urn:bitbucket:oauth2:jwt")

	// create client, in order to add headers
	client := pester.NewExtendedClient(&http.Client{Transport: &nethttp.Transport{}})
	client.MaxRetries = 3
	client.Backoff = pester.ExponentialJitterBackoff
	client.KeepLog = true
	client.Timeout = time.Second * 10
	request, err := http.NewRequest("POST", "https://bitbucket.org/site/oauth2/access_token", bytes.NewBufferString(data.Encode()))
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
	request.Header.Add("Authorization", fmt.Sprintf("%v %v", "JWT", jtwToken))
	request.Header.Add("Content-Type", "application/x-www-form-urlencoded")

	// perform actual request
	response, err := client.Do(request)
	if err != nil {
		return
	}

	defer response.Body.Close()
	if ht != nil {
		ht.Finish()
	}

	body, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return
	}

	// unmarshal json body
	err = json.Unmarshal(body, &accesstoken)
	if err != nil {
		log.Warn().Str("body", string(body)).Msg("Failed unmarshalling access token")
		return
	}

	return
}

func (c *client) GetEstafetteManifest(ctx context.Context, accesstoken AccessToken, pushEvent RepositoryPushEvent) (exists bool, manifest string, err error) {

	// create client, in order to add headers
	client := pester.NewExtendedClient(&http.Client{Transport: &nethttp.Transport{}})
	client.MaxRetries = 3
	client.Backoff = pester.ExponentialJitterBackoff
	client.KeepLog = true
	client.Timeout = time.Second * 10

	manifestSourceAPIUrl := fmt.Sprintf("https://api.bitbucket.org/2.0/repositories/%v/src/%v/.estafette.yaml", pushEvent.Repository.FullName, pushEvent.Push.Changes[0].New.Target.Hash)

	request, err := http.NewRequest("GET", manifestSourceAPIUrl, nil)

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
	request.Header.Add("Authorization", fmt.Sprintf("Bearer %v", accesstoken.AccessToken))

	// perform actual request
	response, err := client.Do(request)
	if err != nil {
		return
	}

	defer response.Body.Close()
	if ht != nil {
		ht.Finish()
	}

	if response.StatusCode == http.StatusNotFound {
		return
	}

	if response.StatusCode != http.StatusOK {
		err = fmt.Errorf("Retrieving estafette manifest from %v failed with status code %v", manifestSourceAPIUrl, response.StatusCode)
		return
	}

	body, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return
	}

	exists = true
	manifest = string(body)

	return
}

// JobVarsFunc returns a function that can get an access token and authenticated url for a repository
func (c *client) JobVarsFunc(ctx context.Context) func(ctx context.Context, repoSource, repoOwner, repoName string) (token string, err error) {
	return func(ctx context.Context, repoSource, repoOwner, repoName string) (token string, err error) {
		// get access token
		accessToken, err := c.GetAccessToken(ctx)
		if err != nil {
			return "", err
		}

		return accessToken.AccessToken, nil
	}
}

func (c *client) GenerateJWT() (tokenString string, err error) {

	// Create the token
	token := jwt.New(jwt.GetSigningMethod("HS256"))
	claims := token.Claims.(jwt.MapClaims)

	now := time.Now().UTC()
	expiry := now.Add(time.Duration(180) * time.Second)

	// set required claims
	claims["iss"] = c.config.Integrations.Bitbucket.Key
	claims["iat"] = now.Unix()
	claims["exp"] = expiry.Unix()
	claims["sub"] = c.config.Integrations.Bitbucket.ClientKey

	// sign the token
	return token.SignedString([]byte(c.config.Integrations.Bitbucket.SharedSecret))
}
