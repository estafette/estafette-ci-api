package bitbucket

import (
	"context"

	"github.com/estafette/estafette-ci-api/pkg/api"
	"github.com/estafette/estafette-ci-api/pkg/clients/bitbucketapi"
	contracts "github.com/estafette/estafette-ci-contracts"
	manifest "github.com/estafette/estafette-ci-manifest"
)

// NewLoggingService returns a new instance of a logging Service.
func NewLoggingService(s Service) Service {
	return &loggingService{s, "bitbucket"}
}

type loggingService struct {
	Service Service
	prefix  string
}

func (s *loggingService) CreateJobForBitbucketPush(ctx context.Context, installation bitbucketapi.BitbucketAppInstallation, event bitbucketapi.RepositoryPushEvent) (err error) {
	defer func() {
		api.HandleLogError(s.prefix, "Service", "CreateJobForBitbucketPush", err, ErrNonCloneableEvent, ErrNoManifest)
	}()

	return s.Service.CreateJobForBitbucketPush(ctx, installation, event)
}

func (s *loggingService) PublishBitbucketEvent(ctx context.Context, event manifest.EstafetteBitbucketEvent) (err error) {
	defer func() {
		api.HandleLogError(s.prefix, "Service", "PublishBitbucketEvent", err)
	}()

	return s.Service.PublishBitbucketEvent(ctx, event)
}

func (s *loggingService) Rename(ctx context.Context, fromRepoSource, fromRepoOwner, fromRepoName, toRepoSource, toRepoOwner, toRepoName string) (err error) {
	defer func() { api.HandleLogError(s.prefix, "Service", "Rename", err) }()

	return s.Service.Rename(ctx, fromRepoSource, fromRepoOwner, fromRepoName, toRepoSource, toRepoOwner, toRepoName)
}

func (s *loggingService) Archive(ctx context.Context, repoSource, repoOwner, repoName string) (err error) {
	defer func() {
		api.HandleLogError(s.prefix, "Service", "Archive", err)
	}()

	return s.Service.Archive(ctx, repoSource, repoOwner, repoName)
}

func (s *loggingService) Unarchive(ctx context.Context, repoSource, repoOwner, repoName string) (err error) {
	defer func() {
		api.HandleLogError(s.prefix, "Service", "Unarchive", err)
	}()

	return s.Service.Unarchive(ctx, repoSource, repoOwner, repoName)
}

func (s *loggingService) IsAllowedOwner(ctx context.Context, repository *bitbucketapi.Repository) (isAllowed bool, organizations []*contracts.Organization) {

	return s.Service.IsAllowedOwner(ctx, repository)
}
