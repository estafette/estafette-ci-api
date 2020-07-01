package githubapi

import (
	"context"

	"github.com/estafette/estafette-ci-api/api"
)

// NewLoggingClient returns a new instance of a logging Client.
func NewLoggingClient(c Client) Client {
	return &loggingClient{c, "githubapi"}
}

type loggingClient struct {
	Client
	prefix string
}

func (c *loggingClient) GetGithubAppToken(ctx context.Context) (token string, err error) {
	defer func() { api.HandleLogError(c.prefix, "GetGithubAppToken", err) }()

	return c.Client.GetGithubAppToken(ctx)
}

func (c *loggingClient) GetInstallationID(ctx context.Context, repoOwner string) (installationID int, err error) {
	defer func() { api.HandleLogError(c.prefix, "GetInstallationID", err) }()

	return c.Client.GetInstallationID(ctx, repoOwner)
}

func (c *loggingClient) GetInstallationToken(ctx context.Context, installationID int) (token AccessToken, err error) {
	defer func() { api.HandleLogError(c.prefix, "GetInstallationToken", err) }()

	return c.Client.GetInstallationToken(ctx, installationID)
}

func (c *loggingClient) GetAuthenticatedRepositoryURL(ctx context.Context, accesstoken AccessToken, htmlURL string) (url string, err error) {
	defer func() { api.HandleLogError(c.prefix, "GetAuthenticatedRepositoryURL", err) }()

	return c.Client.GetAuthenticatedRepositoryURL(ctx, accesstoken, htmlURL)
}

func (c *loggingClient) GetEstafetteManifest(ctx context.Context, accesstoken AccessToken, event PushEvent) (valid bool, manifest string, err error) {
	defer func() { api.HandleLogError(c.prefix, "GetEstafetteManifest", err) }()

	return c.Client.GetEstafetteManifest(ctx, accesstoken, event)
}

func (c *loggingClient) JobVarsFunc(ctx context.Context) func(context.Context, string, string, string) (string, string, error) {
	return c.Client.JobVarsFunc(ctx)
}
