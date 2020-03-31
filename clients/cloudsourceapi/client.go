package cloudsourceapi

import (
	"context"
	"fmt"
	"strings"

	"github.com/estafette/estafette-ci-api/config"
	"golang.org/x/oauth2"
	sourcerepo "google.golang.org/api/sourcerepo/v1"
)

// Client is the interface for communicating with the Google Cloud Source Repository api
type Client interface {
	GetAccessToken(ctx context.Context) (accesstoken AccessToken, err error)
	GetAuthenticatedRepositoryURL(ctx context.Context, accesstoken AccessToken, htmlURL string) (url string, err error)
	GetEstafetteManifest(ctx context.Context, accesstoken AccessToken, notification PubSubNotification) (valid bool, manifest string, err error)
	JobVarsFunc(ctx context.Context) func(ctx context.Context, repoSource, repoOwner, repoName string) (token string, url string, err error)
}

// NewClient creates an githubapi.Client to communicate with the Google Cloud Source Repository api
func NewClient(config config.CloudSourceConfig, sourcerepoService *sourcerepo.Service, tokenSource oauth2.TokenSource) Client {

	return &client{
		service:     sourcerepoService,
		config:      config,
		tokenSource: tokenSource,
	}
}

type client struct {
	service     *sourcerepo.Service
	config      config.CloudSourceConfig
	tokenSource oauth2.TokenSource
}

// GetAccessToken returns an access token to access the Bitbucket api
func (c *client) GetAccessToken(ctx context.Context) (accesstoken AccessToken, err error) {

	token, err := c.tokenSource.Token()
	if err != nil {
		return
	}

	accesstoken = AccessToken{
		AccessToken:  token.AccessToken,
		Expiry:       token.Expiry,
		RefreshToken: token.RefreshToken,
		TokenType:    token.TokenType,
	}

	return
}

func (c *client) GetAuthenticatedRepositoryURL(ctx context.Context, accesstoken AccessToken, htmlURL string) (url string, err error) {

	url = strings.Replace(htmlURL, "https://source.developers.google.com", fmt.Sprintf("https://x-token-auth:%v@source.developers.google.com", accesstoken.AccessToken), -1)

	return
}

func (c *client) GetEstafetteManifest(ctx context.Context, accesstoken AccessToken, pubsubNotification PubSubNotification) (exists bool, manifest string, err error) {
	// TODO(varins): Find a way to fetch only the estafette manifest. Otherwise will have to clone the whole repo.
	return
}

// JobVarsFunc returns a function that can get an access token and authenticated url for a repository
func (c *client) JobVarsFunc(ctx context.Context) func(context.Context, string, string, string) (string, string, error) {
	return func(ctx context.Context, repoSource, repoOwner, repoName string) (token string, url string, err error) {
		// get access token
		accesstoken, err := c.GetAccessToken(ctx)
		if err != nil {
			return
		}
		token = accesstoken.AccessToken

		// get authenticated url for the repository
		url, err = c.GetAuthenticatedRepositoryURL(ctx, accesstoken, fmt.Sprintf("https://%v/p/%v/r/%v", repoSource, repoOwner, repoName))
		if err != nil {
			return
		}

		return
	}
}
