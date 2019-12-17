package github

import (
	"context"

	"github.com/estafette/estafette-ci-api/clients/githubapi"
	"github.com/estafette/estafette-ci-api/helpers"
	"github.com/opentracing/opentracing-go"
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
	defer func() { helpers.FinishSpan(span) }()

	s.Service.CreateJobForGithubPush(ctx, event)
}

func (s *tracingService) HasValidSignature(ctx context.Context, body []byte, signatureHeader string) (valid bool, err error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, s.getSpanName("HasValidSignature"))
	defer func() { helpers.FinishSpanWithError(span, err) }()

	return s.Service.HasValidSignature(ctx, body, signatureHeader)
}

func (s *tracingService) Rename(ctx context.Context, fromRepoSource, fromRepoOwner, fromRepoName, toRepoSource, toRepoOwner, toRepoName string) (err error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, s.getSpanName("Rename"))
	defer func() { helpers.FinishSpanWithError(span, err) }()

	return s.Service.Rename(ctx, fromRepoSource, fromRepoOwner, fromRepoName, toRepoSource, toRepoOwner, toRepoName)
}

func (s *tracingService) IsWhitelistedInstallation(ctx context.Context, installation githubapi.Installation) bool {
	span, ctx := opentracing.StartSpanFromContext(ctx, s.getSpanName("Rename"))
	defer func() { helpers.FinishSpan(span) }()

	return s.Service.IsWhitelistedInstallation(ctx, installation)
}

func (s *tracingService) getSpanName(funcName string) string {
	return "github:" + funcName
}
