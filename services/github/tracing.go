package github

import (
	"context"

	"github.com/estafette/estafette-ci-api/clients/githubapi"
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

func (s *tracingService) CreateJobForGithubPush(ctx context.Context, event githubapi.PushEvent) {
	span, ctx := opentracing.StartSpanFromContext(ctx, s.getSpanName("CreateJobForGithubPush"))
	defer span.Finish()

	s.Service.CreateJobForGithubPush(ctx, event)
}

func (s *tracingService) HasValidSignature(ctx context.Context, body []byte, signatureHeader string) (bool, error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, s.getSpanName("HasValidSignature"))
	defer span.Finish()

	valid, err := s.Service.HasValidSignature(ctx, body, signatureHeader)
	s.handleError(span, err)

	return valid, err
}

func (s *tracingService) Rename(ctx context.Context, fromRepoSource, fromRepoOwner, fromRepoName, toRepoSource, toRepoOwner, toRepoName string) error {
	span, ctx := opentracing.StartSpanFromContext(ctx, s.getSpanName("Rename"))
	defer span.Finish()

	return s.handleError(span, s.Service.Rename(ctx, fromRepoSource, fromRepoOwner, fromRepoName, toRepoSource, toRepoOwner, toRepoName))
}

func (s *tracingService) IsWhitelistedInstallation(ctx context.Context, installation githubapi.Installation) bool {
	span, ctx := opentracing.StartSpanFromContext(ctx, s.getSpanName("Rename"))
	defer span.Finish()

	return s.Service.IsWhitelistedInstallation(ctx, installation)
}

func (s *tracingService) getSpanName(funcName string) string {
	return "github:" + funcName
}

func (s *tracingService) handleError(span opentracing.Span, err error) error {
	if err != nil {
		ext.Error.Set(span, true)
		span.LogFields(log.Error(err))
	}
	return err
}
