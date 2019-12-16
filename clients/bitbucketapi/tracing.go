package bitbucketapi

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

func (s *tracingClient) GetAccessToken(ctx context.Context) (AccessToken, error) {

	span, ctx := opentracing.StartSpanFromContext(ctx, s.getSpanName("GetAccessToken"))
	defer span.Finish()

	return s.Client.GetAccessToken(ctx)
}

func (s *tracingClient) GetAuthenticatedRepositoryURL(ctx context.Context, accessToken AccessToken, htmlURL string) (string, error) {

	span, ctx := opentracing.StartSpanFromContext(ctx, s.getSpanName("GetAuthenticatedRepositoryURL"))
	defer span.Finish()

	return s.Client.GetAuthenticatedRepositoryURL(ctx, accessToken, htmlURL)
}

func (s *tracingClient) GetEstafetteManifest(ctx context.Context, accesstoken AccessToken, event RepositoryPushEvent) (bool, string, error) {

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
	return "bitbucketapi:" + funcName
}
