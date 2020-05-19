package dockerhubapi

import (
	"context"
)

type MockClient struct {
	GetTokenFunc        func(ctx context.Context, repository string) (token DockerHubToken, err error)
	GetDigestFunc       func(ctx context.Context, token DockerHubToken, repository string, tag string) (digest DockerImageDigest, err error)
	GetDigestCachedFunc func(ctx context.Context, repository string, tag string) (digest DockerImageDigest, err error)
}

func (c MockClient) GetToken(ctx context.Context, repository string) (token DockerHubToken, err error) {
	if c.GetTokenFunc == nil {
		return
	}
	return c.GetTokenFunc(ctx, repository)
}

func (c MockClient) GetDigest(ctx context.Context, token DockerHubToken, repository string, tag string) (digest DockerImageDigest, err error) {
	if c.GetDigestFunc == nil {
		return
	}
	return c.GetDigestFunc(ctx, token, repository, tag)
}

func (c MockClient) GetDigestCached(ctx context.Context, repository string, tag string) (digest DockerImageDigest, err error) {
	if c.GetDigestCachedFunc == nil {
		return
	}
	return c.GetDigestCachedFunc(ctx, repository, tag)
}
