package githubapi

import (
	"context"

	"github.com/opentracing/opentracing-go"
)

type tracingClient struct {
	Client
}

// NewTracingClient returns a new instance of a tracing Client.
func NewTracingClient(c Client) Client {
	return &tracingClient{c}
}

func (s *tracingClient) GetGithubAppToken(ctx context.Context) (string, error) {

	span, ctx := opentracing.StartSpanFromContext(ctx, s.getSpanName("GetGithubAppToken"))
	defer span.Finish()

	return s.Client.GetGithubAppToken(ctx)
}

func (s *tracingClient) GetInstallationID(ctx context.Context, repoOwner string) (int, error) {

	span, ctx := opentracing.StartSpanFromContext(ctx, s.getSpanName("GetInstallationID"))
	defer span.Finish()

	return s.Client.GetInstallationID(ctx, repoOwner)
}

func (s *tracingClient) GetInstallationToken(ctx context.Context, installationID int) (AccessToken, error) {

	span, ctx := opentracing.StartSpanFromContext(ctx, s.getSpanName("GetInstallationToken"))
	defer span.Finish()

	return s.Client.GetInstallationToken(ctx, installationID)
}

func (s *tracingClient) GetAuthenticatedRepositoryURL(ctx context.Context, accesstoken AccessToken, htmlURL string) (string, error) {

	span, ctx := opentracing.StartSpanFromContext(ctx, s.getSpanName("GetAuthenticatedRepositoryURL"))
	defer span.Finish()

	return s.Client.GetAuthenticatedRepositoryURL(ctx, accesstoken, htmlURL)
}

func (s *tracingClient) GetEstafetteManifest(ctx context.Context, accesstoken AccessToken, event PushEvent) (bool, string, error) {

	span, ctx := opentracing.StartSpanFromContext(ctx, s.getSpanName("GetEstafetteManifest"))
	defer span.Finish()

	return s.Client.GetEstafetteManifest(ctx, accesstoken, event)
}

func (s *tracingClient) JobVarsFunc(ctx context.Context) func(context.Context, string, string, string) (string, string, error) {

	span, ctx := opentracing.StartSpanFromContext(ctx, s.getSpanName("JobVarsFunc"))
	defer span.Finish()

	return s.Client.JobVarsFunc(ctx)
}

func (s *tracingClient) getSpanName(funcName string) string {
	return "githubapi:" + funcName
}
