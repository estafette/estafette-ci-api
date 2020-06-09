package github

import (
	"context"

	"github.com/estafette/estafette-ci-api/clients/githubapi"
)

type MockService struct {
	CreateJobForGithubPushFunc    func(ctx context.Context, event githubapi.PushEvent) (err error)
	HasValidSignatureFunc         func(ctx context.Context, body []byte, signatureHeader string) (validSignature bool, err error)
	RenameFunc                    func(ctx context.Context, fromRepoSource, fromRepoOwner, fromRepoName, toRepoSource, toRepoOwner, toRepoName string) (err error)
	ArchiveFunc                   func(ctx context.Context, repoSource, repoOwner, repoName string) (err error)
	UnarchiveFunc                 func(ctx context.Context, repoSource, repoOwner, repoName string) (err error)
	IsWhitelistedInstallationFunc func(ctx context.Context, installation githubapi.Installation) (isWhiteListed bool)
}

func (s MockService) CreateJobForGithubPush(ctx context.Context, event githubapi.PushEvent) (err error) {
	if s.CreateJobForGithubPushFunc == nil {
		return
	}
	return s.CreateJobForGithubPushFunc(ctx, event)
}

func (s MockService) HasValidSignature(ctx context.Context, body []byte, signatureHeader string) (validSignature bool, err error) {
	if s.HasValidSignatureFunc == nil {
		return
	}
	return s.HasValidSignatureFunc(ctx, body, signatureHeader)
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

func (s MockService) IsWhitelistedInstallation(ctx context.Context, installation githubapi.Installation) (isWhiteListed bool) {
	if s.IsWhitelistedInstallationFunc == nil {
		return
	}
	return s.IsWhitelistedInstallationFunc(ctx, installation)
}
