package bitbucket

import (
	"context"

	"github.com/estafette/estafette-ci-api/api"
	"github.com/estafette/estafette-ci-api/clients/bitbucketapi"
	"github.com/opentracing/opentracing-go"
)

// NewTracingService returns a new instance of a tracing Service.
func NewTracingService(s Service) Service {
	return &tracingService{s, "bitbucket"}
}

type tracingService struct {
	Service
	prefix string
}

func (s *tracingService) CreateJobForBitbucketPush(ctx context.Context, event bitbucketapi.RepositoryPushEvent) (err error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, api.GetSpanName(s.prefix, "CreateJobForBitbucketPush"))
	defer func() { api.FinishSpanWithError(span, err) }()

	return s.Service.CreateJobForBitbucketPush(ctx, event)
}

func (s *tracingService) Rename(ctx context.Context, fromRepoSource, fromRepoOwner, fromRepoName, toRepoSource, toRepoOwner, toRepoName string) (err error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, api.GetSpanName(s.prefix, "Rename"))
	defer func() { api.FinishSpanWithError(span, err) }()

	return s.Service.Rename(ctx, fromRepoSource, fromRepoOwner, fromRepoName, toRepoSource, toRepoOwner, toRepoName)
}
