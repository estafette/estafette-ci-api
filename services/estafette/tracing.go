package estafette

import (
	"context"

	"github.com/estafette/estafette-ci-api/auth"
	"github.com/estafette/estafette-ci-api/clients/builderapi"
	"github.com/estafette/estafette-ci-api/helpers"
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

func (s *tracingService) CreateBuild(ctx context.Context, build contracts.Build, waitForJobToStart bool) (b *contracts.Build, err error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, helpers.GetSpanName(s.prefix, "CreateBuild"))
	defer func() { helpers.FinishSpanWithError(span, err) }()

	return s.Service.CreateBuild(ctx, build, waitForJobToStart)
}

func (s *tracingService) FinishBuild(ctx context.Context, repoSource, repoOwner, repoName string, buildID int, buildStatus string) (err error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, helpers.GetSpanName(s.prefix, "FinishBuild"))
	defer func() { helpers.FinishSpanWithError(span, err) }()

	return s.Service.FinishBuild(ctx, repoSource, repoOwner, repoName, buildID, buildStatus)
}

func (s *tracingService) CreateRelease(ctx context.Context, release contracts.Release, mft manifest.EstafetteManifest, repoBranch, repoRevision string, waitForJobToStart bool) (r *contracts.Release, err error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, helpers.GetSpanName(s.prefix, "CreateRelease"))
	defer func() { helpers.FinishSpanWithError(span, err) }()

	return s.Service.CreateRelease(ctx, release, mft, repoBranch, repoRevision, waitForJobToStart)
}

func (s *tracingService) FinishRelease(ctx context.Context, repoSource, repoOwner, repoName string, releaseID int, releaseStatus string) (err error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, helpers.GetSpanName(s.prefix, "FinishRelease"))
	defer func() { helpers.FinishSpanWithError(span, err) }()

	return s.Service.FinishRelease(ctx, repoSource, repoOwner, repoName, releaseID, releaseStatus)
}

func (s *tracingService) FireGitTriggers(ctx context.Context, gitEvent manifest.EstafetteGitEvent) (err error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, helpers.GetSpanName(s.prefix, "FireGitTriggers"))
	defer func() { helpers.FinishSpanWithError(span, err) }()

	return s.Service.FireGitTriggers(ctx, gitEvent)
}

func (s *tracingService) FirePipelineTriggers(ctx context.Context, build contracts.Build, event string) (err error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, helpers.GetSpanName(s.prefix, "FirePipelineTriggers"))
	defer func() { helpers.FinishSpanWithError(span, err) }()

	return s.Service.FirePipelineTriggers(ctx, build, event)
}

func (s *tracingService) FireReleaseTriggers(ctx context.Context, release contracts.Release, event string) (err error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, helpers.GetSpanName(s.prefix, "FireReleaseTriggers"))
	defer func() { helpers.FinishSpanWithError(span, err) }()

	return s.Service.FireReleaseTriggers(ctx, release, event)
}

func (s *tracingService) FirePubSubTriggers(ctx context.Context, pubsubEvent manifest.EstafettePubSubEvent) (err error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, helpers.GetSpanName(s.prefix, "FirePubSubTriggers"))
	defer func() { helpers.FinishSpanWithError(span, err) }()

	return s.Service.FirePubSubTriggers(ctx, pubsubEvent)
}

func (s *tracingService) FireCronTriggers(ctx context.Context) (err error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, helpers.GetSpanName(s.prefix, "FireCronTriggers"))
	defer func() { helpers.FinishSpanWithError(span, err) }()

	return s.Service.FireCronTriggers(ctx)
}

func (s *tracingService) Rename(ctx context.Context, fromRepoSource, fromRepoOwner, fromRepoName, toRepoSource, toRepoOwner, toRepoName string) (err error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, helpers.GetSpanName(s.prefix, "Rename"))
	defer func() { helpers.FinishSpanWithError(span, err) }()

	return s.Service.Rename(ctx, fromRepoSource, fromRepoOwner, fromRepoName, toRepoSource, toRepoOwner, toRepoName)
}

func (s *tracingService) Archive(ctx context.Context, repoSource, repoOwner, repoName string) (err error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, helpers.GetSpanName(s.prefix, "Rename"))
	defer func() { helpers.FinishSpanWithError(span, err) }()

	return s.Service.Archive(ctx, repoSource, repoOwner, repoName)
}

func (s *tracingService) Unarchive(ctx context.Context, repoSource, repoOwner, repoName string) (err error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, helpers.GetSpanName(s.prefix, "Rename"))
	defer func() { helpers.FinishSpanWithError(span, err) }()

	return s.Service.Unarchive(ctx, repoSource, repoOwner, repoName)
}

func (s *tracingService) UpdateBuildStatus(ctx context.Context, event builderapi.CiBuilderEvent) (err error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, helpers.GetSpanName(s.prefix, "UpdateBuildStatus"))
	defer func() { helpers.FinishSpanWithError(span, err) }()

	return s.Service.UpdateBuildStatus(ctx, event)
}

func (s *tracingService) UpdateJobResources(ctx context.Context, event builderapi.CiBuilderEvent) (err error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, helpers.GetSpanName(s.prefix, "UpdateJobResources"))
	defer func() { helpers.FinishSpanWithError(span, err) }()

	return s.Service.UpdateJobResources(ctx, event)
}

func (s *tracingService) GetUser(ctx context.Context, authUser auth.User) (user *contracts.User, err error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, helpers.GetSpanName(s.prefix, "GetUser"))
	defer func() { helpers.FinishSpanWithError(span, err) }()

	return s.Service.GetUser(ctx, authUser)
}

func (s *tracingService) CreateUser(ctx context.Context, authUser auth.User) (user *contracts.User, err error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, helpers.GetSpanName(s.prefix, "CreateUser"))
	defer func() { helpers.FinishSpanWithError(span, err) }()

	return s.Service.CreateUser(ctx, authUser)
}
