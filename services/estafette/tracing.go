package estafette

import (
	"context"

	"github.com/estafette/estafette-ci-api/clients/builderapi"
	contracts "github.com/estafette/estafette-ci-contracts"
	manifest "github.com/estafette/estafette-ci-manifest"
	"github.com/opentracing/opentracing-go"
	"github.com/opentracing/opentracing-go/ext"
	"github.com/opentracing/opentracing-go/log"
)

// NewTracingService returns a new instance of a tracing Service.
func NewTracingService(s Service) Service {
	return &tracingService{s}
}

type tracingService struct {
	Service
}

func (s *tracingService) CreateBuild(ctx context.Context, build contracts.Build, waitForJobToStart bool) (b *contracts.Build, err error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, s.getSpanName("CreateBuild"))
	defer span.Finish()
	defer func(span opentracing.Span) {
		s.handleError(span, err)
	}(span)

	return s.Service.CreateBuild(ctx, build, waitForJobToStart)
}

func (s *tracingService) FinishBuild(ctx context.Context, repoSource, repoOwner, repoName string, buildID int, buildStatus string) (err error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, s.getSpanName("FinishBuild"))
	defer span.Finish()
	defer func(span opentracing.Span) {
		s.handleError(span, err)
	}(span)

	return s.Service.FinishBuild(ctx, repoSource, repoOwner, repoName, buildID, buildStatus)
}

func (s *tracingService) CreateRelease(ctx context.Context, release contracts.Release, mft manifest.EstafetteManifest, repoBranch, repoRevision string, waitForJobToStart bool) (r *contracts.Release, err error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, s.getSpanName("CreateRelease"))
	defer span.Finish()
	defer func(span opentracing.Span) {
		s.handleError(span, err)
	}(span)

	return s.Service.CreateRelease(ctx, release, mft, repoBranch, repoRevision, waitForJobToStart)
}

func (s *tracingService) FinishRelease(ctx context.Context, repoSource, repoOwner, repoName string, releaseID int, releaseStatus string) (err error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, s.getSpanName("FinishRelease"))
	defer span.Finish()
	defer func(span opentracing.Span) {
		s.handleError(span, err)
	}(span)

	return s.Service.FinishRelease(ctx, repoSource, repoOwner, repoName, releaseID, releaseStatus)
}

func (s *tracingService) FireGitTriggers(ctx context.Context, gitEvent manifest.EstafetteGitEvent) (err error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, s.getSpanName("FireGitTriggers"))
	defer span.Finish()
	defer func(span opentracing.Span) {
		s.handleError(span, err)
	}(span)

	return s.Service.FireGitTriggers(ctx, gitEvent)
}

func (s *tracingService) FirePipelineTriggers(ctx context.Context, build contracts.Build, event string) (err error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, s.getSpanName("FirePipelineTriggers"))
	defer span.Finish()
	defer func(span opentracing.Span) {
		s.handleError(span, err)
	}(span)

	return s.Service.FirePipelineTriggers(ctx, build, event)
}

func (s *tracingService) FireReleaseTriggers(ctx context.Context, release contracts.Release, event string) (err error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, s.getSpanName("FireReleaseTriggers"))
	defer span.Finish()
	defer func(span opentracing.Span) {
		s.handleError(span, err)
	}(span)

	return s.Service.FireReleaseTriggers(ctx, release, event)
}

func (s *tracingService) FirePubSubTriggers(ctx context.Context, pubsubEvent manifest.EstafettePubSubEvent) (err error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, s.getSpanName("FirePubSubTriggers"))
	defer span.Finish()
	defer func(span opentracing.Span) {
		s.handleError(span, err)
	}(span)

	return s.Service.FirePubSubTriggers(ctx, pubsubEvent)
}

func (s *tracingService) FireCronTriggers(ctx context.Context) (err error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, s.getSpanName("FireCronTriggers"))
	defer span.Finish()
	defer func(span opentracing.Span) {
		s.handleError(span, err)
	}(span)

	return s.Service.FireCronTriggers(ctx)
}

func (s *tracingService) Rename(ctx context.Context, fromRepoSource, fromRepoOwner, fromRepoName, toRepoSource, toRepoOwner, toRepoName string) (err error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, s.getSpanName("Rename"))
	defer span.Finish()
	defer func(span opentracing.Span) {
		s.handleError(span, err)
	}(span)

	return s.Service.Rename(ctx, fromRepoSource, fromRepoOwner, fromRepoName, toRepoSource, toRepoOwner, toRepoName)
}

func (s *tracingService) UpdateBuildStatus(ctx context.Context, event builderapi.CiBuilderEvent) (err error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, s.getSpanName("UpdateBuildStatus"))
	defer span.Finish()
	defer func(span opentracing.Span) {
		s.handleError(span, err)
	}(span)

	return s.Service.UpdateBuildStatus(ctx, event)
}

func (s *tracingService) UpdateJobResources(ctx context.Context, event builderapi.CiBuilderEvent) (err error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, s.getSpanName("UpdateJobResources"))
	defer span.Finish()
	defer func(span opentracing.Span) {
		s.handleError(span, err)
	}(span)

	return s.Service.UpdateJobResources(ctx, event)
}

func (s *tracingService) getSpanName(funcName string) string {
	return "estafette:" + funcName
}

func (s *tracingService) handleError(span opentracing.Span, err error) {
	if err != nil {
		ext.Error.Set(span, true)
		span.LogFields(log.Error(err))
	}
}
