package github

import (
	"context"

	"github.com/estafette/estafette-ci-api/pkg/api"
	"github.com/estafette/estafette-ci-api/pkg/clients/githubapi"
	contracts "github.com/estafette/estafette-ci-contracts"
	manifest "github.com/estafette/estafette-ci-manifest"
	"github.com/opentracing/opentracing-go"
)

// NewTracingService returns a new instance of a tracing Service.
func NewTracingService(s Service) Service {
	return &tracingService{s, "github"}
}

type tracingService struct {
	Service Service
	prefix  string
}

func (s *tracingService) CreateJobForGithubPush(ctx context.Context, event githubapi.PushEvent) (err error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, api.GetSpanName(s.prefix, "CreateJobForGithubPush"))
	defer func() { api.FinishSpanWithError(span, err) }()

	return s.Service.CreateJobForGithubPush(ctx, event)
}

func (s *tracingService) PublishGithubEvent(ctx context.Context, event manifest.EstafetteGithubEvent) (err error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, api.GetSpanName(s.prefix, "PublishGithubEvent"))
	defer func() { api.FinishSpanWithError(span, err) }()

	return s.Service.PublishGithubEvent(ctx, event)
}

func (s *tracingService) HasValidSignature(ctx context.Context, body []byte, appIDHeader, signatureHeader string) (valid bool, err error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, api.GetSpanName(s.prefix, "HasValidSignature"))
	defer func() { api.FinishSpanWithError(span, err) }()

	return s.Service.HasValidSignature(ctx, body, appIDHeader, signatureHeader)
}

func (s *tracingService) Rename(ctx context.Context, fromRepoSource, fromRepoOwner, fromRepoName, toRepoSource, toRepoOwner, toRepoName string) (err error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, api.GetSpanName(s.prefix, "Rename"))
	defer func() { api.FinishSpanWithError(span, err) }()

	return s.Service.Rename(ctx, fromRepoSource, fromRepoOwner, fromRepoName, toRepoSource, toRepoOwner, toRepoName)
}

func (s *tracingService) Archive(ctx context.Context, repoSource, repoOwner, repoName string) (err error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, api.GetSpanName(s.prefix, "Archive"))
	defer func() { api.FinishSpanWithError(span, err) }()

	return s.Service.Archive(ctx, repoSource, repoOwner, repoName)
}

func (s *tracingService) Unarchive(ctx context.Context, repoSource, repoOwner, repoName string) (err error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, api.GetSpanName(s.prefix, "Unarchive"))
	defer func() { api.FinishSpanWithError(span, err) }()

	return s.Service.Unarchive(ctx, repoSource, repoOwner, repoName)
}

func (s *tracingService) IsAllowedInstallation(ctx context.Context, installationID int) (isAllowed bool, organizations []*contracts.Organization) {
	span, ctx := opentracing.StartSpanFromContext(ctx, api.GetSpanName(s.prefix, "Rename"))
	defer func() { api.FinishSpan(span) }()

	return s.Service.IsAllowedInstallation(ctx, installationID)
}
