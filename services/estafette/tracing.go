package estafette

import (
	"context"

	"github.com/estafette/estafette-ci-api/api"
	contracts "github.com/estafette/estafette-ci-contracts"
	manifest "github.com/estafette/estafette-ci-manifest"
	"github.com/opentracing/opentracing-go"
)

// NewTracingService returns a new instance of a tracing Service.
func NewTracingService(s Service) Service {
	return &tracingService{s, "estafette"}
}

type tracingService struct {
	Service
	prefix string
}

func (s *tracingService) CreateBuild(ctx context.Context, build contracts.Build) (b *contracts.Build, err error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, api.GetSpanName(s.prefix, "CreateBuild"))
	defer func() { api.FinishSpanWithError(span, err) }()

	return s.Service.CreateBuild(ctx, build)
}

func (s *tracingService) FinishBuild(ctx context.Context, repoSource, repoOwner, repoName string, buildID string, buildStatus contracts.Status) (err error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, api.GetSpanName(s.prefix, "FinishBuild"))
	defer func() { api.FinishSpanWithError(span, err) }()

	return s.Service.FinishBuild(ctx, repoSource, repoOwner, repoName, buildID, buildStatus)
}

func (s *tracingService) CreateRelease(ctx context.Context, release contracts.Release, mft manifest.EstafetteManifest, repoBranch, repoRevision string) (r *contracts.Release, err error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, api.GetSpanName(s.prefix, "CreateRelease"))
	defer func() { api.FinishSpanWithError(span, err) }()

	return s.Service.CreateRelease(ctx, release, mft, repoBranch, repoRevision)
}

func (s *tracingService) FinishRelease(ctx context.Context, repoSource, repoOwner, repoName string, releaseID string, releaseStatus contracts.Status) (err error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, api.GetSpanName(s.prefix, "FinishRelease"))
	defer func() { api.FinishSpanWithError(span, err) }()

	return s.Service.FinishRelease(ctx, repoSource, repoOwner, repoName, releaseID, releaseStatus)
}

func (s *tracingService) FireGitTriggers(ctx context.Context, gitEvent manifest.EstafetteGitEvent) (err error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, api.GetSpanName(s.prefix, "FireGitTriggers"))
	defer func() { api.FinishSpanWithError(span, err) }()

	return s.Service.FireGitTriggers(ctx, gitEvent)
}

func (s *tracingService) FirePipelineTriggers(ctx context.Context, build contracts.Build, event string) (err error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, api.GetSpanName(s.prefix, "FirePipelineTriggers"))
	defer func() { api.FinishSpanWithError(span, err) }()

	return s.Service.FirePipelineTriggers(ctx, build, event)
}

func (s *tracingService) FireReleaseTriggers(ctx context.Context, release contracts.Release, event string) (err error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, api.GetSpanName(s.prefix, "FireReleaseTriggers"))
	defer func() { api.FinishSpanWithError(span, err) }()

	return s.Service.FireReleaseTriggers(ctx, release, event)
}

func (s *tracingService) FirePubSubTriggers(ctx context.Context, pubsubEvent manifest.EstafettePubSubEvent) (err error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, api.GetSpanName(s.prefix, "FirePubSubTriggers"))
	defer func() { api.FinishSpanWithError(span, err) }()

	return s.Service.FirePubSubTriggers(ctx, pubsubEvent)
}

func (s *tracingService) FireCronTriggers(ctx context.Context) (err error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, api.GetSpanName(s.prefix, "FireCronTriggers"))
	defer func() { api.FinishSpanWithError(span, err) }()

	return s.Service.FireCronTriggers(ctx)
}

func (s *tracingService) Rename(ctx context.Context, fromRepoSource, fromRepoOwner, fromRepoName, toRepoSource, toRepoOwner, toRepoName string) (err error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, api.GetSpanName(s.prefix, "Rename"))
	defer func() { api.FinishSpanWithError(span, err) }()

	return s.Service.Rename(ctx, fromRepoSource, fromRepoOwner, fromRepoName, toRepoSource, toRepoOwner, toRepoName)
}

func (s *tracingService) Archive(ctx context.Context, repoSource, repoOwner, repoName string) (err error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, api.GetSpanName(s.prefix, "Rename"))
	defer func() { api.FinishSpanWithError(span, err) }()

	return s.Service.Archive(ctx, repoSource, repoOwner, repoName)
}

func (s *tracingService) Unarchive(ctx context.Context, repoSource, repoOwner, repoName string) (err error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, api.GetSpanName(s.prefix, "Rename"))
	defer func() { api.FinishSpanWithError(span, err) }()

	return s.Service.Unarchive(ctx, repoSource, repoOwner, repoName)
}

func (s *tracingService) UpdateBuildStatus(ctx context.Context, event contracts.EstafetteCiBuilderEvent) (err error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, api.GetSpanName(s.prefix, "UpdateBuildStatus"))
	defer func() { api.FinishSpanWithError(span, err) }()

	return s.Service.UpdateBuildStatus(ctx, event)
}

func (s *tracingService) UpdateJobResources(ctx context.Context, event contracts.EstafetteCiBuilderEvent) (err error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, api.GetSpanName(s.prefix, "UpdateJobResources"))
	defer func() { api.FinishSpanWithError(span, err) }()

	return s.Service.UpdateJobResources(ctx, event)
}

func (s *tracingService) SubscribeToEventTopic(ctx context.Context, gitEventTopic *api.EventTopic) {
	span, ctx := opentracing.StartSpanFromContext(ctx, api.GetSpanName(s.prefix, "SubscribeToEventTopic"))
	defer func() { api.FinishSpan(span) }()

	s.Service.SubscribeToEventTopic(ctx, gitEventTopic)
}

func (s *tracingService) GetEventsForJobEnvvars(ctx context.Context, triggers []manifest.EstafetteTrigger, events []manifest.EstafetteEvent) (triggersAsEvents []manifest.EstafetteEvent, err error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, api.GetSpanName(s.prefix, "GetEventsForJobEnvvars"))
	defer func() { api.FinishSpan(span) }()

	return s.Service.GetEventsForJobEnvvars(ctx, triggers, events)
}
