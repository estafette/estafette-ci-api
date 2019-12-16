package dockerhubapi

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

func (s *tracingClient) GetToken(ctx context.Context, repository string) (DockerHubToken, error) {

	span, ctx := opentracing.StartSpanFromContext(ctx, s.getSpanName("GetToken"))
	defer span.Finish()

	return s.Client.GetToken(ctx, repository)
}

func (s *tracingClient) GetDigest(ctx context.Context, token DockerHubToken, repository string, tag string) (DockerImageDigest, error) {

	span, ctx := opentracing.StartSpanFromContext(ctx, s.getSpanName("GetDigest"))
	defer span.Finish()

	return s.Client.GetDigest(ctx, token, repository, tag)
}

func (s *tracingClient) GetDigestCached(ctx context.Context, repository string, tag string) (DockerImageDigest, error) {

	span, ctx := opentracing.StartSpanFromContext(ctx, s.getSpanName("GetDigestCached"))
	defer span.Finish()

	return s.Client.GetDigestCached(ctx, repository, tag)
}

func (s *tracingClient) getSpanName(funcName string) string {
	return "dockerhubapi:" + funcName
}
