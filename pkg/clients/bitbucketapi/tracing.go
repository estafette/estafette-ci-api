package bitbucketapi

import (
	"context"

	"github.com/estafette/estafette-ci-api/pkg/api"
	"github.com/opentracing/opentracing-go"
)

// NewTracingClient returns a new instance of a tracing Client.
func NewTracingClient(c Client) Client {
	return &tracingClient{c, "bitbucketapi"}
}

type tracingClient struct {
	Client Client
	prefix string
}

func (c *tracingClient) GetAccessTokenByInstallation(ctx context.Context, installation BitbucketAppInstallation) (accesstoken AccessToken, err error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, api.GetSpanName(c.prefix, "GetAccessTokenByInstallation"))
	defer func() { api.FinishSpanWithError(span, err) }()

	return c.Client.GetAccessTokenByInstallation(ctx, installation)
}

func (c *tracingClient) GetAccessTokenBySlug(ctx context.Context, workspaceSlug string) (accesstoken AccessToken, err error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, api.GetSpanName(c.prefix, "GetAccessTokenBySlug"))
	defer func() { api.FinishSpanWithError(span, err) }()

	return c.Client.GetAccessTokenBySlug(ctx, workspaceSlug)
}

func (c *tracingClient) GetAccessTokenByUUID(ctx context.Context, workspaceUUID string) (accesstoken AccessToken, err error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, api.GetSpanName(c.prefix, "GetAccessTokenByUUID"))
	defer func() { api.FinishSpanWithError(span, err) }()

	return c.Client.GetAccessTokenByUUID(ctx, workspaceUUID)
}

func (c *tracingClient) GetAccessTokenByJWTToken(ctx context.Context, jwtToken string) (accesstoken AccessToken, err error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, api.GetSpanName(c.prefix, "GetAccessTokenByJWTToken"))
	defer func() { api.FinishSpanWithError(span, err) }()

	return c.Client.GetAccessTokenByJWTToken(ctx, jwtToken)
}

func (c *tracingClient) GetEstafetteManifest(ctx context.Context, accesstoken AccessToken, event RepositoryPushEvent) (valid bool, manifest string, err error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, api.GetSpanName(c.prefix, "GetEstafetteManifest"))
	defer func() { api.FinishSpanWithError(span, err) }()

	return c.Client.GetEstafetteManifest(ctx, accesstoken, event)
}

func (c *tracingClient) JobVarsFunc(ctx context.Context) func(ctx context.Context, repoSource, repoOwner, repoName string) (token string, err error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, api.GetSpanName(c.prefix, "JobVarsFunc"))
	defer func() { api.FinishSpan(span) }()

	return c.Client.JobVarsFunc(ctx)
}

func (c *tracingClient) ValidateInstallationJWT(ctx context.Context, authorizationHeader string) (installation *BitbucketAppInstallation, err error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, api.GetSpanName(c.prefix, "JobVarsFunc"))
	defer func() { api.FinishSpan(span) }()

	return c.Client.ValidateInstallationJWT(ctx, authorizationHeader)
}

func (c *tracingClient) GenerateJWTBySlug(ctx context.Context, workspaceSlug string) (tokenString string, err error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, api.GetSpanName(c.prefix, "GenerateJWTBySlug"))
	defer func() { api.FinishSpan(span) }()

	return c.Client.GenerateJWTBySlug(ctx, workspaceSlug)
}

func (c *tracingClient) GenerateJWTByUUID(ctx context.Context, workspaceUUID string) (tokenString string, err error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, api.GetSpanName(c.prefix, "GenerateJWTByUUID"))
	defer func() { api.FinishSpan(span) }()

	return c.Client.GenerateJWTByUUID(ctx, workspaceUUID)
}

func (c *tracingClient) GenerateJWTByInstallation(ctx context.Context, installation BitbucketAppInstallation) (tokenString string, err error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, api.GetSpanName(c.prefix, "GenerateJWTByInstallation"))
	defer func() { api.FinishSpan(span) }()

	return c.Client.GenerateJWTByInstallation(ctx, installation)
}

func (c *tracingClient) GetInstallationBySlug(ctx context.Context, workspaceSlug string) (installation *BitbucketAppInstallation, err error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, api.GetSpanName(c.prefix, "GetInstallationBySlug"))
	defer func() { api.FinishSpan(span) }()

	return c.Client.GetInstallationBySlug(ctx, workspaceSlug)
}

func (c *tracingClient) GetInstallationByUUID(ctx context.Context, workspaceUUID string) (installation *BitbucketAppInstallation, err error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, api.GetSpanName(c.prefix, "GetInstallationByUUID"))
	defer func() { api.FinishSpan(span) }()

	return c.Client.GetInstallationByUUID(ctx, workspaceUUID)
}

func (c *tracingClient) GetInstallationByClientKey(ctx context.Context, clientKey string) (installation *BitbucketAppInstallation, err error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, api.GetSpanName(c.prefix, "GetInstallationByClientKey"))
	defer func() { api.FinishSpan(span) }()

	return c.Client.GetInstallationByClientKey(ctx, clientKey)
}

func (c *tracingClient) AddApp(ctx context.Context, app BitbucketApp) (err error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, api.GetSpanName(c.prefix, "AddApp"))
	defer func() { api.FinishSpan(span) }()

	return c.Client.AddApp(ctx, app)
}

func (c *tracingClient) GetApps(ctx context.Context) (apps []*BitbucketApp, err error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, api.GetSpanName(c.prefix, "GetApps"))
	defer func() { api.FinishSpan(span) }()

	return c.Client.GetApps(ctx)
}

func (c *tracingClient) GetAppByKey(ctx context.Context, key string) (app *BitbucketApp, err error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, api.GetSpanName(c.prefix, "GetAppByKey"))
	defer func() { api.FinishSpan(span) }()

	return c.Client.GetAppByKey(ctx, key)
}

func (c *tracingClient) AddInstallation(ctx context.Context, installation BitbucketAppInstallation) (err error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, api.GetSpanName(c.prefix, "AddInstallation"))
	defer func() { api.FinishSpan(span) }()

	return c.Client.AddInstallation(ctx, installation)
}

func (c *tracingClient) RemoveInstallation(ctx context.Context, installation BitbucketAppInstallation) (err error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, api.GetSpanName(c.prefix, "RemoveInstallation"))
	defer func() { api.FinishSpan(span) }()

	return c.Client.RemoveInstallation(ctx, installation)
}

func (c *tracingClient) GetWorkspace(ctx context.Context, installation BitbucketAppInstallation) (workspace *Workspace, err error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, api.GetSpanName(c.prefix, "GetWorkspace"))
	defer func() { api.FinishSpan(span) }()

	return c.Client.GetWorkspace(ctx, installation)
}
