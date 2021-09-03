package github

import (
	"context"

	"github.com/estafette/estafette-ci-api/pkg/api"
	"github.com/estafette/estafette-ci-api/pkg/clients/githubapi"
	contracts "github.com/estafette/estafette-ci-contracts"
	manifest "github.com/estafette/estafette-ci-manifest"
)

// NewLoggingService returns a new instance of a logging Service.
func NewLoggingService(s Service) Service {
	return &loggingService{s, "github"}
}

type loggingService struct {
	Service Service
	prefix  string
}

func (s *loggingService) CreateJobForGithubPush(ctx context.Context, event githubapi.PushEvent) (err error) {
	defer func() {
		api.HandleLogError(s.prefix, "Service", "CreateJobForGithubPush", err, ErrNonCloneableEvent, ErrNoManifest)
	}()

	return s.Service.CreateJobForGithubPush(ctx, event)
}

func (s *loggingService) PublishGithubEvent(ctx context.Context, event manifest.EstafetteGithubEvent) (err error) {
	defer func() {
		api.HandleLogError(s.prefix, "Service", "PublishGithubEvent", err)
	}()

	return s.Service.PublishGithubEvent(ctx, event)
}

func (s *loggingService) HasValidSignature(ctx context.Context, body []byte, appIDHeader, signatureHeader string) (valid bool, err error) {
	defer func() { api.HandleLogError(s.prefix, "Service", "HasValidSignature", err) }()

	return s.Service.HasValidSignature(ctx, body, appIDHeader, signatureHeader)
}

func (s *loggingService) Rename(ctx context.Context, fromRepoSource, fromRepoOwner, fromRepoName, toRepoSource, toRepoOwner, toRepoName string) (err error) {
	defer func() { api.HandleLogError(s.prefix, "Service", "Rename", err) }()

	return s.Service.Rename(ctx, fromRepoSource, fromRepoOwner, fromRepoName, toRepoSource, toRepoOwner, toRepoName)
}

func (s *loggingService) Archive(ctx context.Context, repoSource, repoOwner, repoName string) (err error) {
	defer func() { api.HandleLogError(s.prefix, "Service", "Archive", err) }()

	return s.Service.Archive(ctx, repoSource, repoOwner, repoName)
}

func (s *loggingService) Unarchive(ctx context.Context, repoSource, repoOwner, repoName string) (err error) {
	defer func() { api.HandleLogError(s.prefix, "Service", "Unarchive", err) }()

	return s.Service.Unarchive(ctx, repoSource, repoOwner, repoName)
}

func (s *loggingService) IsAllowedInstallation(ctx context.Context, installationID int) (isAllowed bool, organizations []*contracts.Organization) {
	return s.Service.IsAllowedInstallation(ctx, installationID)
}
