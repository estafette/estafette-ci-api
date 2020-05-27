package estafette

import (
	"context"

	"github.com/estafette/estafette-ci-api/auth"
	"github.com/estafette/estafette-ci-api/clients/builderapi"
	"github.com/estafette/estafette-ci-api/helpers"
	contracts "github.com/estafette/estafette-ci-contracts"
	manifest "github.com/estafette/estafette-ci-manifest"
)

// NewLoggingService returns a new instance of a logging Service.
func NewLoggingService(s Service) Service {
	return &loggingService{s, "estafette"}
}

type loggingService struct {
	Service
	prefix string
}

func (s *loggingService) CreateBuild(ctx context.Context, build contracts.Build, waitForJobToStart bool) (b *contracts.Build, err error) {
	defer func() { helpers.HandleLogError(s.prefix, "CreateBuild", err) }()

	return s.Service.CreateBuild(ctx, build, waitForJobToStart)
}

func (s *loggingService) FinishBuild(ctx context.Context, repoSource, repoOwner, repoName string, buildID int, buildStatus string) (err error) {
	defer func() { helpers.HandleLogError(s.prefix, "FinishBuild", err) }()

	return s.Service.FinishBuild(ctx, repoSource, repoOwner, repoName, buildID, buildStatus)
}

func (s *loggingService) CreateRelease(ctx context.Context, release contracts.Release, mft manifest.EstafetteManifest, repoBranch, repoRevision string, waitForJobToStart bool) (r *contracts.Release, err error) {
	defer func() { helpers.HandleLogError(s.prefix, "CreateRelease", err) }()

	return s.Service.CreateRelease(ctx, release, mft, repoBranch, repoRevision, waitForJobToStart)
}

func (s *loggingService) FinishRelease(ctx context.Context, repoSource, repoOwner, repoName string, releaseID int, releaseStatus string) (err error) {
	defer func() { helpers.HandleLogError(s.prefix, "FinishRelease", err) }()

	return s.Service.FinishRelease(ctx, repoSource, repoOwner, repoName, releaseID, releaseStatus)
}

func (s *loggingService) FireGitTriggers(ctx context.Context, gitEvent manifest.EstafetteGitEvent) (err error) {
	defer func() { helpers.HandleLogError(s.prefix, "FireGitTriggers", err) }()

	return s.Service.FireGitTriggers(ctx, gitEvent)
}

func (s *loggingService) FirePipelineTriggers(ctx context.Context, build contracts.Build, event string) (err error) {
	defer func() { helpers.HandleLogError(s.prefix, "FirePipelineTriggers", err) }()

	return s.Service.FirePipelineTriggers(ctx, build, event)
}

func (s *loggingService) FireReleaseTriggers(ctx context.Context, release contracts.Release, event string) (err error) {
	defer func() { helpers.HandleLogError(s.prefix, "FireReleaseTriggers", err) }()

	return s.Service.FireReleaseTriggers(ctx, release, event)
}

func (s *loggingService) FirePubSubTriggers(ctx context.Context, pubsubEvent manifest.EstafettePubSubEvent) (err error) {
	defer func() { helpers.HandleLogError(s.prefix, "FirePubSubTriggers", err) }()

	return s.Service.FirePubSubTriggers(ctx, pubsubEvent)
}

func (s *loggingService) FireCronTriggers(ctx context.Context) (err error) {
	defer func() { helpers.HandleLogError(s.prefix, "FireCronTriggers", err) }()

	return s.Service.FireCronTriggers(ctx)
}

func (s *loggingService) Rename(ctx context.Context, fromRepoSource, fromRepoOwner, fromRepoName, toRepoSource, toRepoOwner, toRepoName string) (err error) {
	defer func() { helpers.HandleLogError(s.prefix, "Rename", err) }()

	return s.Service.Rename(ctx, fromRepoSource, fromRepoOwner, fromRepoName, toRepoSource, toRepoOwner, toRepoName)
}

func (s *loggingService) Archive(ctx context.Context, repoSource, repoOwner, repoName string) (err error) {
	defer func() { helpers.HandleLogError(s.prefix, "Archive", err) }()

	return s.Service.Archive(ctx, repoSource, repoOwner, repoName)
}

func (s *loggingService) Unarchive(ctx context.Context, repoSource, repoOwner, repoName string) (err error) {
	defer func() { helpers.HandleLogError(s.prefix, "Unarchive", err) }()

	return s.Service.Unarchive(ctx, repoSource, repoOwner, repoName)
}

func (s *loggingService) UpdateBuildStatus(ctx context.Context, event builderapi.CiBuilderEvent) (err error) {
	defer func() { helpers.HandleLogError(s.prefix, "UpdateBuildStatus", err) }()

	return s.Service.UpdateBuildStatus(ctx, event)
}

func (s *loggingService) UpdateJobResources(ctx context.Context, event builderapi.CiBuilderEvent) (err error) {
	defer func() { helpers.HandleLogError(s.prefix, "UpdateJobResources", err) }()

	return s.Service.UpdateJobResources(ctx, event)
}

func (s *loggingService) GetUser(ctx context.Context, authUser auth.User) (user *contracts.User, err error) {
	defer func() { helpers.HandleLogError(s.prefix, "GetUser", err) }()

	return s.Service.GetUser(ctx, authUser)
}

func (s *loggingService) CreateUser(ctx context.Context, authUser auth.User) (user *contracts.User, err error) {
	defer func() { helpers.HandleLogError(s.prefix, "CreateUser", err) }()

	return s.Service.CreateUser(ctx, authUser)
}
