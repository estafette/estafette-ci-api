package cloudsource

import (
	"context"

	"github.com/estafette/estafette-ci-api/pkg/api"
	"github.com/estafette/estafette-ci-api/pkg/clients/cloudsourceapi"
	contracts "github.com/estafette/estafette-ci-contracts"
	"github.com/opentracing/opentracing-go"
)

// NewTracingService returns a new instance of a tracing Service.
func NewTracingService(s Service) Service {
	return &tracingService{s, "cloudsource"}
}

type tracingService struct {
	Service Service
	prefix  string
}

func (s *tracingService) CreateJobForCloudSourcePush(ctx context.Context, notification cloudsourceapi.PubSubNotification) (err error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, api.GetSpanName(s.prefix, "CreateJobForCloudSourcePush"))
	defer func() { api.FinishSpanWithError(span, err) }()

	return s.Service.CreateJobForCloudSourcePush(ctx, notification)
}

func (s *tracingService) IsAllowedProject(ctx context.Context, notification cloudsourceapi.PubSubNotification) (isAllowed bool, organizations []*contracts.Organization) {
	_, ctx = opentracing.StartSpanFromContext(ctx, api.GetSpanName(s.prefix, "IsAllowedProject"))

	return s.Service.IsAllowedProject(ctx, notification)
}
