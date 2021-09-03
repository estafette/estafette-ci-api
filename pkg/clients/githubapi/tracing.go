package githubapi

import (
	"context"

	"github.com/estafette/estafette-ci-api/pkg/api"
	"github.com/opentracing/opentracing-go"
)

// NewTracingClient returns a new instance of a tracing Client.
func NewTracingClient(c Client) Client {
	return &tracingClient{c, "githubapi"}
}

type tracingClient struct {
	Client Client
	prefix string
}

func (c *tracingClient) GetGithubAppToken(ctx context.Context, app GithubApp) (token string, err error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, api.GetSpanName(c.prefix, "GetGithubAppToken"))
	defer func() { api.FinishSpanWithError(span, err) }()

	return c.Client.GetGithubAppToken(ctx, app)
}

func (c *tracingClient) GetAppAndInstallationByOwner(ctx context.Context, repoOwner string) (app *GithubApp, installation *GithubInstallation, err error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, api.GetSpanName(c.prefix, "GetAppAndInstallationByOwner"))
	defer func() { api.FinishSpanWithError(span, err) }()

	return c.Client.GetAppAndInstallationByOwner(ctx, repoOwner)
}

func (c *tracingClient) GetAppAndInstallationByID(ctx context.Context, installationID int) (app *GithubApp, installation *GithubInstallation, err error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, api.GetSpanName(c.prefix, "GetAppAndInstallationByID"))
	defer func() { api.FinishSpanWithError(span, err) }()

	return c.Client.GetAppAndInstallationByID(ctx, installationID)
}

func (c *tracingClient) GetInstallationToken(ctx context.Context, app GithubApp, installation GithubInstallation) (accessToken AccessToken, err error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, api.GetSpanName(c.prefix, "GetInstallationToken"))
	defer func() { api.FinishSpanWithError(span, err) }()

	return c.Client.GetInstallationToken(ctx, app, installation)
}

func (c *tracingClient) GetEstafetteManifest(ctx context.Context, accesstoken AccessToken, event PushEvent) (valid bool, manifest string, err error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, api.GetSpanName(c.prefix, "GetEstafetteManifest"))
	defer func() { api.FinishSpanWithError(span, err) }()

	return c.Client.GetEstafetteManifest(ctx, accesstoken, event)
}

func (c *tracingClient) JobVarsFunc(ctx context.Context) func(context.Context, string, string, string) (string, error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, api.GetSpanName(c.prefix, "JobVarsFunc"))
	defer func() { api.FinishSpan(span) }()

	return c.Client.JobVarsFunc(ctx)
}

func (c *tracingClient) ConvertAppManifestCode(ctx context.Context, code string) (err error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, api.GetSpanName(c.prefix, "ConvertAppManifestCode"))
	defer func() { api.FinishSpanWithError(span, err) }()

	return c.Client.ConvertAppManifestCode(ctx, code)
}

func (c *tracingClient) GetApps(ctx context.Context) (apps []*GithubApp, err error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, api.GetSpanName(c.prefix, "GetApps"))
	defer func() { api.FinishSpanWithError(span, err) }()

	return c.Client.GetApps(ctx)
}

func (c *tracingClient) GetAppByID(ctx context.Context, id int) (app *GithubApp, err error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, api.GetSpanName(c.prefix, "GetAppByID"))
	defer func() { api.FinishSpanWithError(span, err) }()

	return c.Client.GetAppByID(ctx, id)
}

func (c *tracingClient) AddApp(ctx context.Context, app GithubApp) (err error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, api.GetSpanName(c.prefix, "AddApp"))
	defer func() { api.FinishSpanWithError(span, err) }()

	return c.Client.AddApp(ctx, app)
}

func (c *tracingClient) RemoveApp(ctx context.Context, app GithubApp) (err error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, api.GetSpanName(c.prefix, "RemoveApp"))
	defer func() { api.FinishSpanWithError(span, err) }()

	return c.Client.RemoveApp(ctx, app)
}

func (c *tracingClient) AddInstallation(ctx context.Context, installation GithubInstallation) (err error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, api.GetSpanName(c.prefix, "AddInstallation"))
	defer func() { api.FinishSpanWithError(span, err) }()

	return c.Client.AddInstallation(ctx, installation)
}

func (c *tracingClient) RemoveInstallation(ctx context.Context, installation GithubInstallation) (err error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, api.GetSpanName(c.prefix, "RemoveInstallation"))
	defer func() { api.FinishSpanWithError(span, err) }()

	return c.Client.RemoveInstallation(ctx, installation)
}
