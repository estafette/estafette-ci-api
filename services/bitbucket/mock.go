package bitbucket

import (
	"context"

	"github.com/estafette/estafette-ci-api/clients/bitbucketapi"
	contracts "github.com/estafette/estafette-ci-contracts"
)

type MockService struct {
	CreateJobForBitbucketPushFunc func(ctx context.Context, event bitbucketapi.RepositoryPushEvent) (err error)
	RenameFunc                    func(ctx context.Context, fromRepoSource, fromRepoOwner, fromRepoName, toRepoSource, toRepoOwner, toRepoName string) (err error)
	ArchiveFunc                   func(ctx context.Context, repoSource, repoOwner, repoName string) (err error)
	UnarchiveFunc                 func(ctx context.Context, repoSource, repoOwner, repoName string) (err error)
	IsWhitelistedOwnerFunc        func(repository bitbucketapi.Repository) (isWhiteListed bool, organizations []*contracts.Organization)
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

func (s MockService) Archive(ctx context.Context, repoSource, repoOwner, repoName string) (err error) {
	if s.ArchiveFunc == nil {
		return
	}
	return s.ArchiveFunc(ctx, repoSource, repoOwner, repoName)
}

func (s MockService) Unarchive(ctx context.Context, repoSource, repoOwner, repoName string) (err error) {
	if s.UnarchiveFunc == nil {
		return
	}
	return s.UnarchiveFunc(ctx, repoSource, repoOwner, repoName)
}

func (s MockService) IsWhitelistedOwner(repository bitbucketapi.Repository) (isWhiteListed bool, organizations []*contracts.Organization) {
	if s.IsWhitelistedOwnerFunc == nil {
		return
	}
	return s.IsWhitelistedOwnerFunc(repository)
}
