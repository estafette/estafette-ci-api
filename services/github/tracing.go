package github

import (
	"context"

	"github.com/estafette/estafette-ci-api/api"
	"github.com/estafette/estafette-ci-api/clients/githubapi"
	"github.com/opentracing/opentracing-go"
)

// NewTracingService returns a new instance of a tracing Service.
func NewTracingService(s Service) Service {
	return &tracingService{s, "github"}
}

type tracingService struct {
	Service
	prefix string
}

func (s *tracingService) CreateJobForGithubPush(ctx context.Context, event githubapi.PushEvent) (err error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, api.GetSpanName(s.prefix, "CreateJobForGithubPush"))
	defer func() { api.FinishSpanWithError(span, err) }()

	return s.Service.CreateJobForGithubPush(ctx, event)
}

func (s *tracingService) HasValidSignature(ctx context.Context, body []byte, signatureHeader string) (valid bool, err error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, api.GetSpanName(s.prefix, "HasValidSignature"))
	defer func() { api.FinishSpanWithError(span, err) }()

	return s.Service.HasValidSignature(ctx, body, signatureHeader)
}

func (s *tracingService) Rename(ctx context.Context, fromRepoSource, fromRepoOwner, fromRepoName, toRepoSource, toRepoOwner, toRepoName string) (err error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, api.GetSpanName(s.prefix, "Rename"))
	defer func() { api.FinishSpanWithError(span, err) }()

	return s.Service.Rename(ctx, fromRepoSource, fromRepoOwner, fromRepoName, toRepoSource, toRepoOwner, toRepoName)
}

func (s *tracingService) IsWhitelistedInstallation(ctx context.Context, installation githubapi.Installation) bool {
	span, ctx := opentracing.StartSpanFromContext(ctx, api.GetSpanName(s.prefix, "Rename"))
	defer func() { api.FinishSpan(span) }()

	return s.Service.IsWhitelistedInstallation(ctx, installation)
}
