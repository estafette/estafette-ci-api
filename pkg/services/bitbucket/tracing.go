package bitbucket

import (
	"context"

	"github.com/estafette/estafette-ci-api/pkg/api"
	"github.com/estafette/estafette-ci-api/pkg/clients/bitbucketapi"
	contracts "github.com/estafette/estafette-ci-contracts"
	manifest "github.com/estafette/estafette-ci-manifest"
	"github.com/opentracing/opentracing-go"
)

// NewTracingService returns a new instance of a tracing Service.
func NewTracingService(s Service) Service {
	return &tracingService{s, "bitbucket"}
}

type tracingService struct {
	Service Service
	prefix  string
}

func (s *tracingService) CreateJobForBitbucketPush(ctx context.Context, installation bitbucketapi.BitbucketAppInstallation, event bitbucketapi.RepositoryPushEvent) (err error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, api.GetSpanName(s.prefix, "CreateJobForBitbucketPush"))
	defer func() { api.FinishSpanWithError(span, err) }()

	return s.Service.CreateJobForBitbucketPush(ctx, installation, event)
}

func (s *tracingService) PublishBitbucketEvent(ctx context.Context, event manifest.EstafetteBitbucketEvent) (err error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, api.GetSpanName(s.prefix, "PublishBitbucketEvent"))
	defer func() { api.FinishSpanWithError(span, err) }()

	return s.Service.PublishBitbucketEvent(ctx, event)
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

func (s *tracingService) IsAllowedOwner(ctx context.Context, repository *bitbucketapi.Repository) (isAllowed bool, organizations []*contracts.Organization) {
	_, ctx = opentracing.StartSpanFromContext(ctx, api.GetSpanName(s.prefix, "IsAllowedOwner"))

	return s.Service.IsAllowedOwner(ctx, repository)
}
