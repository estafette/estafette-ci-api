package dockerhubapi

import (
	"context"

	"github.com/estafette/estafette-ci-api/pkg/api"
	"github.com/opentracing/opentracing-go"
)

// NewTracingClient returns a new instance of a tracing Client.
func NewTracingClient(c Client) Client {
	return &tracingClient{c, "dockerhubapi"}
}

type tracingClient struct {
	Client Client
	prefix string
}

func (c *tracingClient) GetToken(ctx context.Context, repository string) (token DockerHubToken, err error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, api.GetSpanName(c.prefix, "GetToken"))
	defer func() { api.FinishSpanWithError(span, err) }()

	return c.Client.GetToken(ctx, repository)
}

func (c *tracingClient) GetDigest(ctx context.Context, token DockerHubToken, repository string, tag string) (digest DockerImageDigest, err error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, api.GetSpanName(c.prefix, "GetDigest"))
	defer func() { api.FinishSpanWithError(span, err) }()

	return c.Client.GetDigest(ctx, token, repository, tag)
}

func (c *tracingClient) GetDigestCached(ctx context.Context, repository string, tag string) (digest DockerImageDigest, err error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, api.GetSpanName(c.prefix, "GetDigestCached"))
	defer func() { api.FinishSpanWithError(span, err) }()

	return c.Client.GetDigestCached(ctx, repository, tag)
}
