package github

import (
	"context"

	"github.com/estafette/estafette-ci-api/clients/githubapi"
	"github.com/estafette/estafette-ci-api/config"
)

type MockService struct {
	CreateJobForGithubPushFunc    func(ctx context.Context, event githubapi.PushEvent)
	HasValidSignatureFunc         func(ctx context.Context, body []byte, signatureHeader string) (validSignature bool, err error)
	RenameFunc                    func(ctx context.Context, fromRepoSource, fromRepoOwner, fromRepoName, toRepoSource, toRepoOwner, toRepoName string) (err error)
	IsWhitelistedInstallationFunc func(ctx context.Context, installation githubapi.Installation) (isWhiteListed bool)
}

func (s MockService) CreateJobForGithubPush(ctx context.Context, event githubapi.PushEvent) {
	if s.CreateJobForGithubPushFunc == nil {
		return
	}
	s.CreateJobForGithubPushFunc(ctx, event)
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

func (s MockService) IsWhitelistedInstallation(ctx context.Context, installation githubapi.Installation) (isWhiteListed bool) {
	if s.IsWhitelistedInstallationFunc == nil {
		return
	}
	return s.IsWhitelistedInstallationFunc(ctx, installation)
}

func (s MockService) RefreshConfig(config *config.APIConfig) {
}
