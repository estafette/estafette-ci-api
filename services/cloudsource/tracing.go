package cloudsource

import (
	"context"

	"github.com/estafette/estafette-ci-api/clients/cloudsourceapi"
	"github.com/estafette/estafette-ci-api/helpers"
	"github.com/opentracing/opentracing-go"
)

// NewTracingService returns a new instance of a tracing Service.
func NewTracingService(s Service) Service {
	return &tracingService{s, "cloudsource"}
}

type tracingService struct {
	Service
	prefix string
}

func (s *tracingService) CreateJobForCloudsourcePush(ctx context.Context, notification cloudsourceapi.PubSubNotification) (err error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, helpers.GetSpanName(s.prefix, "CreateJobForCloudSourcePush"))
	defer func() { helpers.FinishSpanWithError(span, err) }()

	return s.Service.CreateJobForCloudSourcePush(ctx, notification)
}
