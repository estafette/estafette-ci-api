package bitbucketapi

import (
	"context"

	"github.com/estafette/estafette-ci-api/pkg/api"
)

// NewLoggingClient returns a new instance of a logging Client.
func NewLoggingClient(c Client) Client {
	return &loggingClient{c, "bitbucketapi"}
}

type loggingClient struct {
	Client Client
	prefix string
}

func (c *loggingClient) GetAccessToken(ctx context.Context, installation BitbucketAppInstallation) (accesstoken AccessToken, err error) {
	defer func() { api.HandleLogError(c.prefix, "Client", "GetAccessToken", err) }()

	return c.Client.GetAccessToken(ctx, installation)
}

func (c *loggingClient) GetEstafetteManifest(ctx context.Context, accesstoken AccessToken, event RepositoryPushEvent) (valid bool, manifest string, err error) {
	defer func() { api.HandleLogError(c.prefix, "Client", "GetEstafetteManifest", err) }()

	return c.Client.GetEstafetteManifest(ctx, accesstoken, event)
}

func (c *loggingClient) JobVarsFunc(ctx context.Context) func(ctx context.Context, repoSource, repoOwner, repoName string) (token string, err error) {
	return c.Client.JobVarsFunc(ctx)
}

func (c *loggingClient) ValidateInstallationJWT(ctx context.Context, authorizationHeader string) (installation *BitbucketAppInstallation, err error) {
	return c.Client.ValidateInstallationJWT(ctx, authorizationHeader)
}

func (c *loggingClient) GenerateJWT(ctx context.Context, installation BitbucketAppInstallation) (tokenString string, err error) {
	return c.Client.GenerateJWT(ctx, installation)
}

func (c *loggingClient) GetInstallations(ctx context.Context) (installations []*BitbucketAppInstallation, err error) {
	return c.Client.GetInstallations(ctx)
}

func (c *loggingClient) AddInstallation(ctx context.Context, installation BitbucketAppInstallation) (err error) {
	return c.Client.AddInstallation(ctx, installation)
}

func (c *loggingClient) RemoveInstallation(ctx context.Context, installation BitbucketAppInstallation) (err error) {
	return c.Client.RemoveInstallation(ctx, installation)
}
