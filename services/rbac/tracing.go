package rbac

import (
	"context"

	"github.com/dgrijalva/jwt-go"
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

func (s *tracingService) GetProviderByName(ctx context.Context, name string) (provider *config.OAuthProvider, err error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, helpers.GetSpanName(s.prefix, "GetProviderByName"))
	defer func() { helpers.FinishSpanWithError(span, err) }()

	return s.Service.GetProviderByName(ctx, name)
}

func (s *tracingService) GetUserByIdentity(ctx context.Context, identity contracts.UserIdentity) (user *contracts.User, err error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, helpers.GetSpanName(s.prefix, "GetUserByIdentity"))
	defer func() { helpers.FinishSpanWithError(span, err) }()

	return s.Service.GetUserByIdentity(ctx, identity)
}

func (s *tracingService) GetUserByID(ctx context.Context, id string) (user *contracts.User, err error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, helpers.GetSpanName(s.prefix, "GetUserByID"))
	defer func() { helpers.FinishSpanWithError(span, err) }()

	return s.Service.GetUserByID(ctx, id)
}

func (s *tracingService) CreateUser(ctx context.Context, identity contracts.UserIdentity) (user *contracts.User, err error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, helpers.GetSpanName(s.prefix, "CreateUser"))
	defer func() { helpers.FinishSpanWithError(span, err) }()

	return s.Service.CreateUser(ctx, identity)
}

func (s *tracingService) UpdateUser(ctx context.Context, user contracts.User) (err error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, helpers.GetSpanName(s.prefix, "UpdateUser"))
	defer func() { helpers.FinishSpanWithError(span, err) }()

	return s.Service.UpdateUser(ctx, user)
}

func (s *tracingService) GenerateJWT(ctx context.Context, optionalClaims jwt.MapClaims) (tokenString string, err error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, helpers.GetSpanName(s.prefix, "GenerateJWT"))
	defer func() { helpers.FinishSpanWithError(span, err) }()

	return s.Service.GenerateJWT(ctx, optionalClaims)
}
func (s *tracingService) ValidateJWT(ctx context.Context, tokenString string) (token *jwt.Token, err error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, helpers.GetSpanName(s.prefix, "ValidateJWT"))
	defer func() { helpers.FinishSpanWithError(span, err) }()

	return s.Service.ValidateJWT(ctx, tokenString)
}

func (s *tracingService) GetClaimsFromJWT(ctx context.Context, tokenString string) (claims jwt.MapClaims, err error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, helpers.GetSpanName(s.prefix, "GetClaimsFromJWT"))
	defer func() { helpers.FinishSpanWithError(span, err) }()

	return s.Service.GetClaimsFromJWT(ctx, tokenString)
}
