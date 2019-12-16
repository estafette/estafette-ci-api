package bitbucket

import (
	"context"

	"github.com/estafette/estafette-ci-api/clients/bitbucketapi"
	"github.com/opentracing/opentracing-go"
	"github.com/opentracing/opentracing-go/ext"
	"github.com/opentracing/opentracing-go/log"
)

type tracingService struct {
	Service
}

// NewTracingService returns a new instance of a tracing Service.
func NewTracingService(s Service) Service {
	return &tracingService{s}
}

func (s *tracingService) CreateJobForBitbucketPush(ctx context.Context, event bitbucketapi.RepositoryPushEvent) {

	span, ctx := opentracing.StartSpanFromContext(ctx, s.getSpanName("CreateJobForBitbucketPush"))
	defer span.Finish()

	s.Service.CreateJobForBitbucketPush(ctx, event)
}

func (s *tracingService) Rename(ctx context.Context, fromRepoSource, fromRepoOwner, fromRepoName, toRepoSource, toRepoOwner, toRepoName string) error {

	span, ctx := opentracing.StartSpanFromContext(ctx, s.getSpanName("Rename"))
	defer span.Finish()

	return s.handleError(span, s.Service.Rename(ctx, fromRepoSource, fromRepoOwner, fromRepoName, toRepoSource, toRepoOwner, toRepoName))
}

func (s *tracingService) getSpanName(funcName string) string {
	return "bitbucket:" + funcName
}

func (s *tracingService) handleError(span opentracing.Span, err error) error {
	if err != nil {
		ext.Error.Set(span, true)
		span.LogFields(log.Error(err))
	}
	return err
}
