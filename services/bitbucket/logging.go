package bitbucket

import (
	"context"

	"github.com/estafette/estafette-ci-api/clients/bitbucketapi"
	"github.com/estafette/estafette-ci-api/config"
	"github.com/estafette/estafette-ci-api/helpers"
)

// NewLoggingService returns a new instance of a logging Service.
func NewLoggingService(s Service) Service {
	return &loggingService{s, "bitbucket"}
}

type loggingService struct {
	Service
	prefix string
}

func (s *loggingService) CreateJobForBitbucketPush(ctx context.Context, event bitbucketapi.RepositoryPushEvent) (err error) {
	defer func() {
		helpers.HandleLogErrorWithIgnoredErrors(s.prefix, "CreateJobForBitbucketPush", err, ErrNonCloneableEvent, ErrNoManifest)
	}()

	return s.Service.CreateJobForBitbucketPush(ctx, event)
}

func (s *loggingService) Rename(ctx context.Context, fromRepoSource, fromRepoOwner, fromRepoName, toRepoSource, toRepoOwner, toRepoName string) (err error) {
	defer func() { helpers.HandleLogError(s.prefix, "Rename", err) }()

	return s.Service.Rename(ctx, fromRepoSource, fromRepoOwner, fromRepoName, toRepoSource, toRepoOwner, toRepoName)
}

func (s *loggingService) RefreshConfig(config *config.APIConfig) {
	s.Service.RefreshConfig(config)
}
