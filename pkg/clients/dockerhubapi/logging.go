package dockerhubapi

import (
	"context"

	"github.com/estafette/estafette-ci-api/pkg/api"
)

// NewLoggingClient returns a new instance of a logging Client.
func NewLoggingClient(c Client) Client {
	return &loggingClient{c, "dockerhubapi"}
}

type loggingClient struct {
	Client Client
	prefix string
}

func (c *loggingClient) GetToken(ctx context.Context, repository string) (token DockerHubToken, err error) {
	defer func() { api.HandleLogError(c.prefix, "Client", "GetToken", err) }()

	return c.Client.GetToken(ctx, repository)
}

func (c *loggingClient) GetDigest(ctx context.Context, token DockerHubToken, repository string, tag string) (digest DockerImageDigest, err error) {
	defer func() { api.HandleLogError(c.prefix, "Client", "GetDigest", err) }()

	return c.Client.GetDigest(ctx, token, repository, tag)
}

func (c *loggingClient) GetDigestCached(ctx context.Context, repository string, tag string) (digest DockerImageDigest, err error) {
	defer func() { api.HandleLogError(c.prefix, "Client", "GetDigestCached", err) }()

	return c.Client.GetDigestCached(ctx, repository, tag)
}
