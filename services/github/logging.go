package github

import (
	"context"

	"github.com/estafette/estafette-ci-api/api"
	"github.com/estafette/estafette-ci-api/clients/githubapi"
)

// NewLoggingService returns a new instance of a logging Service.
func NewLoggingService(s Service) Service {
	return &loggingService{s, "github"}
}

type loggingService struct {
	Service
	prefix string
}

func (s *loggingService) CreateJobForGithubPush(ctx context.Context, event githubapi.PushEvent) (err error) {
	defer func() {
		api.HandleLogError(s.prefix, "CreateJobForGithubPush", err, ErrNonCloneableEvent, ErrNoManifest)
	}()

	return s.Service.CreateJobForGithubPush(ctx, event)
}

func (s *loggingService) HasValidSignature(ctx context.Context, body []byte, signatureHeader string) (valid bool, err error) {
	defer func() { api.HandleLogError(s.prefix, "HasValidSignature", err) }()

	return s.Service.HasValidSignature(ctx, body, signatureHeader)
}

func (s *loggingService) Rename(ctx context.Context, fromRepoSource, fromRepoOwner, fromRepoName, toRepoSource, toRepoOwner, toRepoName string) (err error) {
	defer func() { api.HandleLogError(s.prefix, "Rename", err) }()

	return s.Service.Rename(ctx, fromRepoSource, fromRepoOwner, fromRepoName, toRepoSource, toRepoOwner, toRepoName)
}

func (s *loggingService) IsWhitelistedInstallation(ctx context.Context, installation githubapi.Installation) bool {
	return s.Service.IsWhitelistedInstallation(ctx, installation)
}
