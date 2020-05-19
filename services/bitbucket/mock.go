package bitbucket

import (
	"context"

	"github.com/estafette/estafette-ci-api/clients/bitbucketapi"
	"github.com/estafette/estafette-ci-api/config"
)

type MockService struct {
	CreateJobForBitbucketPushFunc func(ctx context.Context, event bitbucketapi.RepositoryPushEvent) (err error)
	RenameFunc                    func(ctx context.Context, fromRepoSource, fromRepoOwner, fromRepoName, toRepoSource, toRepoOwner, toRepoName string) (err error)
	IsWhitelistedOwnerFunc        func(repository bitbucketapi.Repository) (isWhiteListed bool)
}

func (s MockService) CreateJobForBitbucketPush(ctx context.Context, event bitbucketapi.RepositoryPushEvent) (err error) {
	if s.CreateJobForBitbucketPushFunc == nil {
		return
	}
	return s.CreateJobForBitbucketPushFunc(ctx, event)
}

func (s MockService) Rename(ctx context.Context, fromRepoSource, fromRepoOwner, fromRepoName, toRepoSource, toRepoOwner, toRepoName string) (err error) {
	if s.RenameFunc == nil {
		return
	}
	return s.RenameFunc(ctx, fromRepoSource, fromRepoOwner, fromRepoName, toRepoSource, toRepoOwner, toRepoName)
}

func (s MockService) IsWhitelistedOwner(repository bitbucketapi.Repository) (isWhiteListed bool) {
	if s.IsWhitelistedOwnerFunc == nil {
		return
	}
	return s.IsWhitelistedOwnerFunc(repository)
}

func (s MockService) RefreshConfig(config *config.APIConfig) {
}
