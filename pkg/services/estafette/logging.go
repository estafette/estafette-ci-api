package estafette

import (
	"context"

	"github.com/estafette/estafette-ci-api/pkg/api"
	contracts "github.com/estafette/estafette-ci-contracts"
	manifest "github.com/estafette/estafette-ci-manifest"
)

// NewLoggingService returns a new instance of a logging Service.
func NewLoggingService(s Service) Service {
	return &loggingService{s, "estafette"}
}

type loggingService struct {
	Service Service
	prefix  string
}

func (s *loggingService) CreateBuild(ctx context.Context, build contracts.Build) (b *contracts.Build, err error) {
	defer func() { api.HandleLogError(s.prefix, "Service", "CreateBuild", err) }()

	return s.Service.CreateBuild(ctx, build)
}

func (s *loggingService) FinishBuild(ctx context.Context, repoSource, repoOwner, repoName string, buildID string, buildStatus contracts.Status) (err error) {
	defer func() { api.HandleLogError(s.prefix, "Service", "FinishBuild", err) }()

	return s.Service.FinishBuild(ctx, repoSource, repoOwner, repoName, buildID, buildStatus)
}

func (s *loggingService) CreateRelease(ctx context.Context, release contracts.Release, mft manifest.EstafetteManifest, repoBranch, repoRevision string) (r *contracts.Release, err error) {
	defer func() { api.HandleLogError(s.prefix, "Service", "CreateRelease", err) }()

	return s.Service.CreateRelease(ctx, release, mft, repoBranch, repoRevision)
}

func (s *loggingService) FinishRelease(ctx context.Context, repoSource, repoOwner, repoName string, releaseID string, releaseStatus contracts.Status) (err error) {
	defer func() { api.HandleLogError(s.prefix, "Service", "FinishRelease", err) }()

	return s.Service.FinishRelease(ctx, repoSource, repoOwner, repoName, releaseID, releaseStatus)
}

func (s *loggingService) CreateBot(ctx context.Context, bot contracts.Bot, mft manifest.EstafetteManifest, repoBranch string) (b *contracts.Bot, err error) {
	defer func() { api.HandleLogError(s.prefix, "Service", "CreateBot", err) }()

	return s.Service.CreateBot(ctx, bot, mft, repoBranch)
}

func (s *loggingService) FinishBot(ctx context.Context, repoSource, repoOwner, repoName string, botID string, botStatus contracts.Status) (err error) {
	defer func() { api.HandleLogError(s.prefix, "Service", "FinishBot", err) }()

	return s.Service.FinishBot(ctx, repoSource, repoOwner, repoName, botID, botStatus)
}

func (s *loggingService) FireGitTriggers(ctx context.Context, gitEvent manifest.EstafetteGitEvent) (err error) {
	defer func() { api.HandleLogError(s.prefix, "Service", "FireGitTriggers", err) }()

	return s.Service.FireGitTriggers(ctx, gitEvent)
}

func (s *loggingService) FirePipelineTriggers(ctx context.Context, build contracts.Build, event string) (err error) {
	defer func() { api.HandleLogError(s.prefix, "Service", "FirePipelineTriggers", err) }()

	return s.Service.FirePipelineTriggers(ctx, build, event)
}

func (s *loggingService) FireReleaseTriggers(ctx context.Context, release contracts.Release, event string) (err error) {
	defer func() { api.HandleLogError(s.prefix, "Service", "FireReleaseTriggers", err) }()

	return s.Service.FireReleaseTriggers(ctx, release, event)
}

func (s *loggingService) FirePubSubTriggers(ctx context.Context, pubsubEvent manifest.EstafettePubSubEvent) (err error) {
	defer func() { api.HandleLogError(s.prefix, "Service", "FirePubSubTriggers", err) }()

	return s.Service.FirePubSubTriggers(ctx, pubsubEvent)
}

func (s *loggingService) FireCronTriggers(ctx context.Context, cronEvent manifest.EstafetteCronEvent) (err error) {
	defer func() { api.HandleLogError(s.prefix, "Service", "FireCronTriggers", err) }()

	return s.Service.FireCronTriggers(ctx, cronEvent)
}

func (s *loggingService) FireGithubTriggers(ctx context.Context, githubEvent manifest.EstafetteGithubEvent) (err error) {
	defer func() { api.HandleLogError(s.prefix, "Service", "FireGithubTriggers", err) }()

	return s.Service.FireGithubTriggers(ctx, githubEvent)
}

func (s *loggingService) FireBitbucketTriggers(ctx context.Context, bitbucketEvent manifest.EstafetteBitbucketEvent) (err error) {
	defer func() { api.HandleLogError(s.prefix, "Service", "FireBitbucketTriggers", err) }()

	return s.Service.FireBitbucketTriggers(ctx, bitbucketEvent)
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

func (s *loggingService) UpdateBuildStatus(ctx context.Context, event contracts.EstafetteCiBuilderEvent) (err error) {
	defer func() { api.HandleLogError(s.prefix, "Service", "UpdateBuildStatus", err) }()

	return s.Service.UpdateBuildStatus(ctx, event)
}

func (s *loggingService) UpdateJobResources(ctx context.Context, event contracts.EstafetteCiBuilderEvent) (err error) {
	defer func() { api.HandleLogError(s.prefix, "Service", "UpdateJobResources", err) }()

	return s.Service.UpdateJobResources(ctx, event)
}

func (s *loggingService) GetEventsForJobEnvvars(ctx context.Context, triggers []manifest.EstafetteTrigger, events []manifest.EstafetteEvent) (triggersAsEvents []manifest.EstafetteEvent, err error) {
	defer func() { api.HandleLogError(s.prefix, "Service", "GetEventsForJobEnvvars", err) }()

	return s.Service.GetEventsForJobEnvvars(ctx, triggers, events)
}
