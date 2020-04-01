package cloudsourceapi

import (
	"context"

	"github.com/estafette/estafette-ci-api/helpers"
)

// NewLoggingClient returns a new instance of a logging Client.
func NewLoggingClient(c Client) Client {
	return &loggingClient{c, "cloudsourceapi"}
}

type loggingClient struct {
	Client
	prefix string
}

func (c *loggingClient) GetAccessToken(ctx context.Context) (accesstoken AccessToken, err error) {
	defer func() { helpers.HandleLogError(c.prefix, "GetAccessToken", err) }()

	return c.Client.GetAccessToken(ctx)
}

func (c *loggingClient) GetAuthenticatedRepositoryURL(ctx context.Context, accesstoken AccessToken, htmlURL string) (url string, err error) {
	defer func() { helpers.HandleLogError(c.prefix, "GetAuthenticatedRepositoryURL", err) }()

	return c.Client.GetAuthenticatedRepositoryURL(ctx, accesstoken, htmlURL)
}

func (c *loggingClient) GetEstafetteManifest(ctx context.Context, accesstoken AccessToken, notification PubSubNotification) (valid bool, manifest string, err error) {
	defer func() { helpers.HandleLogError(c.prefix, "GetEstafetteManifest", err) }()

	return c.Client.GetEstafetteManifest(ctx, accesstoken, notification)
}

func (c *loggingClient) JobVarsFunc(ctx context.Context) func(ctx context.Context, repoSource, repoOwner, repoName string) (token string, url string, err error) {
	return c.Client.JobVarsFunc(ctx)
}
