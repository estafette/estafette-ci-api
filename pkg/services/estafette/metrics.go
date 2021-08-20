package estafette

import (
	"context"
	"time"

	"github.com/estafette/estafette-ci-api/pkg/api"
	contracts "github.com/estafette/estafette-ci-contracts"
	manifest "github.com/estafette/estafette-ci-manifest"
	"github.com/go-kit/kit/metrics"
)

// NewMetricsService returns a new instance of a metrics Service.
func NewMetricsService(s Service, requestCount metrics.Counter, requestLatency metrics.Histogram) Service {
	return &metricsService{s, requestCount, requestLatency}
}

type metricsService struct {
	Service        Service
	requestCount   metrics.Counter
	requestLatency metrics.Histogram
}

func (s *metricsService) CreateBuild(ctx context.Context, build contracts.Build) (b *contracts.Build, err error) {
	defer func(begin time.Time) { api.UpdateMetrics(s.requestCount, s.requestLatency, "CreateBuild", begin) }(time.Now())

	return s.Service.CreateBuild(ctx, build)
}

func (s *metricsService) FinishBuild(ctx context.Context, repoSource, repoOwner, repoName string, buildID string, buildStatus contracts.Status) (err error) {
	defer func(begin time.Time) { api.UpdateMetrics(s.requestCount, s.requestLatency, "FinishBuild", begin) }(time.Now())

	return s.Service.FinishBuild(ctx, repoSource, repoOwner, repoName, buildID, buildStatus)
}

func (s *metricsService) CreateRelease(ctx context.Context, release contracts.Release, mft manifest.EstafetteManifest, repoBranch, repoRevision string) (r *contracts.Release, err error) {
	defer func(begin time.Time) { api.UpdateMetrics(s.requestCount, s.requestLatency, "CreateRelease", begin) }(time.Now())

	return s.Service.CreateRelease(ctx, release, mft, repoBranch, repoRevision)
}

func (s *metricsService) FinishRelease(ctx context.Context, repoSource, repoOwner, repoName string, releaseID string, releaseStatus contracts.Status) (err error) {
	defer func(begin time.Time) { api.UpdateMetrics(s.requestCount, s.requestLatency, "FinishRelease", begin) }(time.Now())

	return s.Service.FinishRelease(ctx, repoSource, repoOwner, repoName, releaseID, releaseStatus)
}

func (s *metricsService) CreateBot(ctx context.Context, bot contracts.Bot, mft manifest.EstafetteManifest, repoBranch string) (b *contracts.Bot, err error) {
	defer func(begin time.Time) { api.UpdateMetrics(s.requestCount, s.requestLatency, "CreateBot", begin) }(time.Now())

	return s.Service.CreateBot(ctx, bot, mft, repoBranch)
}

func (s *metricsService) FinishBot(ctx context.Context, repoSource, repoOwner, repoName string, botID string, botStatus contracts.Status) (err error) {
	defer func(begin time.Time) { api.UpdateMetrics(s.requestCount, s.requestLatency, "FinishBot", begin) }(time.Now())

	return s.Service.FinishBot(ctx, repoSource, repoOwner, repoName, botID, botStatus)
}

func (s *metricsService) FireGitTriggers(ctx context.Context, gitEvent manifest.EstafetteGitEvent) (err error) {
	defer func(begin time.Time) {
		api.UpdateMetrics(s.requestCount, s.requestLatency, "FireGitTriggers", begin)
	}(time.Now())

	return s.Service.FireGitTriggers(ctx, gitEvent)
}

func (s *metricsService) FirePipelineTriggers(ctx context.Context, build contracts.Build, event string) (err error) {
	defer func(begin time.Time) {
		api.UpdateMetrics(s.requestCount, s.requestLatency, "FirePipelineTriggers", begin)
	}(time.Now())

	return s.Service.FirePipelineTriggers(ctx, build, event)
}

func (s *metricsService) FireReleaseTriggers(ctx context.Context, release contracts.Release, event string) (err error) {
	defer func(begin time.Time) {
		api.UpdateMetrics(s.requestCount, s.requestLatency, "FireReleaseTriggers", begin)
	}(time.Now())

	return s.Service.FireReleaseTriggers(ctx, release, event)
}

func (s *metricsService) FirePubSubTriggers(ctx context.Context, pubsubEvent manifest.EstafettePubSubEvent) (err error) {
	defer func(begin time.Time) {
		api.UpdateMetrics(s.requestCount, s.requestLatency, "FirePubSubTriggers", begin)
	}(time.Now())

	return s.Service.FirePubSubTriggers(ctx, pubsubEvent)
}

func (s *metricsService) FireCronTriggers(ctx context.Context, cronEvent manifest.EstafetteCronEvent) (err error) {
	defer func(begin time.Time) {
		api.UpdateMetrics(s.requestCount, s.requestLatency, "FireCronTriggers", begin)
	}(time.Now())

	return s.Service.FireCronTriggers(ctx, cronEvent)
}

func (s *metricsService) FireGithubTriggers(ctx context.Context, githubEvent manifest.EstafetteGithubEvent) (err error) {
	defer func(begin time.Time) {
		api.UpdateMetrics(s.requestCount, s.requestLatency, "FireGithubTriggers", begin)
	}(time.Now())

	return s.Service.FireGithubTriggers(ctx, githubEvent)
}

func (s *metricsService) FireBitbucketTriggers(ctx context.Context, bitbucketEvent manifest.EstafetteBitbucketEvent) (err error) {
	defer func(begin time.Time) {
		api.UpdateMetrics(s.requestCount, s.requestLatency, "FireBitbucketTriggers", begin)
	}(time.Now())

	return s.Service.FireBitbucketTriggers(ctx, bitbucketEvent)
}

func (s *metricsService) Rename(ctx context.Context, fromRepoSource, fromRepoOwner, fromRepoName, toRepoSource, toRepoOwner, toRepoName string) (err error) {
	defer func(begin time.Time) { api.UpdateMetrics(s.requestCount, s.requestLatency, "Rename", begin) }(time.Now())

	return s.Service.Rename(ctx, fromRepoSource, fromRepoOwner, fromRepoName, toRepoSource, toRepoOwner, toRepoName)
}

func (s *metricsService) Archive(ctx context.Context, repoSource, repoOwner, repoName string) (err error) {
	defer func(begin time.Time) { api.UpdateMetrics(s.requestCount, s.requestLatency, "Archive", begin) }(time.Now())

	return s.Service.Archive(ctx, repoSource, repoOwner, repoName)
}

func (s *metricsService) Unarchive(ctx context.Context, repoSource, repoOwner, repoName string) (err error) {
	defer func(begin time.Time) { api.UpdateMetrics(s.requestCount, s.requestLatency, "Unarchive", begin) }(time.Now())

	return s.Service.Unarchive(ctx, repoSource, repoOwner, repoName)
}

func (s *metricsService) UpdateBuildStatus(ctx context.Context, event contracts.EstafetteCiBuilderEvent) (err error) {
	defer func(begin time.Time) {
		api.UpdateMetrics(s.requestCount, s.requestLatency, "UpdateBuildStatus", begin)
	}(time.Now())

	return s.Service.UpdateBuildStatus(ctx, event)
}

func (s *metricsService) UpdateJobResources(ctx context.Context, event contracts.EstafetteCiBuilderEvent) (err error) {
	defer func(begin time.Time) {
		api.UpdateMetrics(s.requestCount, s.requestLatency, "UpdateJobResources", begin)
	}(time.Now())

	return s.Service.UpdateJobResources(ctx, event)
}

func (s *metricsService) GetEventsForJobEnvvars(ctx context.Context, triggers []manifest.EstafetteTrigger, events []manifest.EstafetteEvent) (triggersAsEvents []manifest.EstafetteEvent, err error) {
	defer func(begin time.Time) {
		api.UpdateMetrics(s.requestCount, s.requestLatency, "GetEventsForJobEnvvars", begin)
	}(time.Now())

	return s.Service.GetEventsForJobEnvvars(ctx, triggers, events)
}
