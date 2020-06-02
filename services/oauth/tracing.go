package oauth

import (
	"context"

	"github.com/estafette/estafette-ci-api/config"
	"github.com/estafette/estafette-ci-api/helpers"
	"github.com/opentracing/opentracing-go"
)

// NewTracingService returns a new instance of a tracing Service.
func NewTracingService(s Service) Service {
	return &tracingService{s, "estafette"}
}

type tracingService struct {
	Service
	prefix string
}

func (s *tracingService) GetProviders(ctx context.Context) (providers []*config.OAuthProvider, err error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, helpers.GetSpanName(s.prefix, "GetProviders"))
	defer func() { helpers.FinishSpanWithError(span, err) }()

	return s.Service.GetProviders(ctx)
}
