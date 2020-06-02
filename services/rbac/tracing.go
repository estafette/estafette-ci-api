package rbac

import (
	"context"

	"github.com/estafette/estafette-ci-api/auth"
	"github.com/estafette/estafette-ci-api/config"
	"github.com/estafette/estafette-ci-api/helpers"
	contracts "github.com/estafette/estafette-ci-contracts"
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

func (s *tracingService) GetUser(ctx context.Context, authUser auth.User) (user *contracts.User, err error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, helpers.GetSpanName(s.prefix, "GetUser"))
	defer func() { helpers.FinishSpanWithError(span, err) }()

	return s.Service.GetUser(ctx, authUser)
}

func (s *tracingService) CreateUser(ctx context.Context, authUser auth.User) (user *contracts.User, err error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, helpers.GetSpanName(s.prefix, "CreateUser"))
	defer func() { helpers.FinishSpanWithError(span, err) }()

	return s.Service.CreateUser(ctx, authUser)
}

func (s *tracingService) UpdateUser(ctx context.Context, authUser auth.User) (err error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, helpers.GetSpanName(s.prefix, "UpdateUser"))
	defer func() { helpers.FinishSpanWithError(span, err) }()

	return s.Service.UpdateUser(ctx, authUser)
}
