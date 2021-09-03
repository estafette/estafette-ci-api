package githubapi

import (
	"context"

	"github.com/estafette/estafette-ci-api/pkg/api"
)

// NewLoggingClient returns a new instance of a logging Client.
func NewLoggingClient(c Client) Client {
	return &loggingClient{c, "githubapi"}
}

type loggingClient struct {
	Client Client
	prefix string
}

func (c *loggingClient) GetGithubAppToken(ctx context.Context, app GithubApp) (token string, err error) {
	defer func() { api.HandleLogError(c.prefix, "Client", "GetGithubAppToken", err) }()

	return c.Client.GetGithubAppToken(ctx, app)
}

func (c *loggingClient) GetAppAndInstallationByOwner(ctx context.Context, repoOwner string) (app *GithubApp, installation *GithubInstallation, err error) {
	defer func() { api.HandleLogError(c.prefix, "Client", "GetAppAndInstallationByOwner", err) }()

	return c.Client.GetAppAndInstallationByOwner(ctx, repoOwner)
}

func (c *loggingClient) GetAppAndInstallationByID(ctx context.Context, installationID int) (app *GithubApp, installation *GithubInstallation, err error) {
	defer func() { api.HandleLogError(c.prefix, "Client", "GetAppAndInstallationByID", err) }()

	return c.Client.GetAppAndInstallationByID(ctx, installationID)
}

func (c *loggingClient) GetInstallationToken(ctx context.Context, app GithubApp, installation GithubInstallation) (accessToken AccessToken, err error) {
	defer func() { api.HandleLogError(c.prefix, "Client", "GetInstallationToken", err) }()

	return c.Client.GetInstallationToken(ctx, app, installation)
}

func (c *loggingClient) GetEstafetteManifest(ctx context.Context, accesstoken AccessToken, event PushEvent) (valid bool, manifest string, err error) {
	defer func() { api.HandleLogError(c.prefix, "Client", "GetEstafetteManifest", err) }()

	return c.Client.GetEstafetteManifest(ctx, accesstoken, event)
}

func (c *loggingClient) JobVarsFunc(ctx context.Context) func(context.Context, string, string, string) (string, error) {
	return c.Client.JobVarsFunc(ctx)
}

func (c *loggingClient) ConvertAppManifestCode(ctx context.Context, code string) (err error) {
	defer func() { api.HandleLogError(c.prefix, "Client", "ConvertAppManifestCode", err) }()

	return c.Client.ConvertAppManifestCode(ctx, code)
}

func (c *loggingClient) GetApps(ctx context.Context) (apps []*GithubApp, err error) {
	defer func() { api.HandleLogError(c.prefix, "Client", "GetApps", err) }()

	return c.Client.GetApps(ctx)
}

func (c *loggingClient) GetAppByID(ctx context.Context, id int) (app *GithubApp, err error) {
	defer func() { api.HandleLogError(c.prefix, "Client", "GetAppByID", err) }()

	return c.Client.GetAppByID(ctx, id)
}

func (c *loggingClient) AddApp(ctx context.Context, app GithubApp) (err error) {
	defer func() { api.HandleLogError(c.prefix, "Client", "AddApp", err) }()

	return c.Client.AddApp(ctx, app)
}

func (c *loggingClient) RemoveApp(ctx context.Context, app GithubApp) (err error) {
	defer func() { api.HandleLogError(c.prefix, "Client", "RemoveApp", err) }()

	return c.Client.RemoveApp(ctx, app)
}

func (c *loggingClient) AddInstallation(ctx context.Context, installation GithubInstallation) (err error) {
	defer func() { api.HandleLogError(c.prefix, "Client", "AddInstallation", err) }()

	return c.Client.AddInstallation(ctx, installation)
}

func (c *loggingClient) RemoveInstallation(ctx context.Context, installation GithubInstallation) (err error) {
	defer func() { api.HandleLogError(c.prefix, "Client", "RemoveInstallation", err) }()

	return c.Client.RemoveInstallation(ctx, installation)
}
