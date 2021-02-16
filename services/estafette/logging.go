package estafette

import (
	"context"

	"github.com/estafette/estafette-ci-api/api"
	"github.com/estafette/estafette-ci-api/clients/builderapi"
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
	defer func() { api.HandleLogError(s.prefix, "CreateBuild", err) }()

	return s.Service.CreateBuild(ctx, build, waitForJobToStart)
}

func (s *loggingService) FinishBuild(ctx context.Context, repoSource, repoOwner, repoName string, buildID int, buildStatus contracts.Status) (err error) {
	defer func() { api.HandleLogError(s.prefix, "FinishBuild", err) }()

	return s.Service.FinishBuild(ctx, repoSource, repoOwner, repoName, buildID, buildStatus)
}

func (s *loggingService) CreateRelease(ctx context.Context, release contracts.Release, mft manifest.EstafetteManifest, repoBranch, repoRevision string, waitForJobToStart bool) (r *contracts.Release, err error) {
	defer func() { api.HandleLogError(s.prefix, "CreateRelease", err) }()

	return s.Service.CreateRelease(ctx, release, mft, repoBranch, repoRevision, waitForJobToStart)
}

func (s *loggingService) FinishRelease(ctx context.Context, repoSource, repoOwner, repoName string, releaseID int, releaseStatus contracts.Status) (err error) {
	defer func() { api.HandleLogError(s.prefix, "FinishRelease", err) }()

	return s.Service.FinishRelease(ctx, repoSource, repoOwner, repoName, releaseID, releaseStatus)
}

func (s *loggingService) FireGitTriggers(ctx context.Context, gitEvent manifest.EstafetteGitEvent) (err error) {
	defer func() { api.HandleLogError(s.prefix, "FireGitTriggers", err) }()

	return s.Service.FireGitTriggers(ctx, gitEvent)
}

func (s *loggingService) FirePipelineTriggers(ctx context.Context, build contracts.Build, event string) (err error) {
	defer func() { api.HandleLogError(s.prefix, "FirePipelineTriggers", err) }()

	return s.Service.FirePipelineTriggers(ctx, build, event)
}

func (s *loggingService) FireReleaseTriggers(ctx context.Context, release contracts.Release, event string) (err error) {
	defer func() { api.HandleLogError(s.prefix, "FireReleaseTriggers", err) }()

	return s.Service.FireReleaseTriggers(ctx, release, event)
}

func (s *loggingService) FirePubSubTriggers(ctx context.Context, pubsubEvent manifest.EstafettePubSubEvent) (err error) {
	defer func() { api.HandleLogError(s.prefix, "FirePubSubTriggers", err) }()

	return s.Service.FirePubSubTriggers(ctx, pubsubEvent)
}

func (s *loggingService) FireCronTriggers(ctx context.Context) (err error) {
	defer func() { api.HandleLogError(s.prefix, "FireCronTriggers", err) }()

	return s.Service.FireCronTriggers(ctx)
}

func (s *loggingService) Rename(ctx context.Context, fromRepoSource, fromRepoOwner, fromRepoName, toRepoSource, toRepoOwner, toRepoName string) (err error) {
	defer func() { api.HandleLogError(s.prefix, "Rename", err) }()

	return s.Service.Rename(ctx, fromRepoSource, fromRepoOwner, fromRepoName, toRepoSource, toRepoOwner, toRepoName)
}

func (s *loggingService) Archive(ctx context.Context, repoSource, repoOwner, repoName string) (err error) {
	defer func() { api.HandleLogError(s.prefix, "Archive", err) }()

	return s.Service.Archive(ctx, repoSource, repoOwner, repoName)
}

func (s *loggingService) Unarchive(ctx context.Context, repoSource, repoOwner, repoName string) (err error) {
	defer func() { api.HandleLogError(s.prefix, "Unarchive", err) }()

	return s.Service.Unarchive(ctx, repoSource, repoOwner, repoName)
}

func (s *loggingService) UpdateBuildStatus(ctx context.Context, event builderapi.CiBuilderEvent) (err error) {
	defer func() { api.HandleLogError(s.prefix, "UpdateBuildStatus", err) }()

	return s.Service.UpdateBuildStatus(ctx, event)
}

func (s *loggingService) UpdateJobResources(ctx context.Context, event builderapi.CiBuilderEvent) (err error) {
	defer func() { api.HandleLogError(s.prefix, "UpdateJobResources", err) }()

	return s.Service.UpdateJobResources(ctx, event)
}

func (s *loggingService) SubscribeToGitEventsTopic(ctx context.Context, gitEventTopic *api.GitEventTopic) {
	s.Service.SubscribeToGitEventsTopic(ctx, gitEventTopic)
}

func (s *loggingService) GetEventsForJobEnvvars(ctx context.Context, triggers []manifest.EstafetteTrigger, events []manifest.EstafetteEvent) (triggersAsEvents []manifest.EstafetteEvent, err error) {
	defer func() { api.HandleLogError(s.prefix, "GetEventsForJobEnvvars", err) }()

	return s.Service.GetEventsForJobEnvvars(ctx, triggers, events)
}
