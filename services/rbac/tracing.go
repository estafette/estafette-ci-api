package rbac

import (
	"context"

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

func (s *tracingService) GetRoles(ctx context.Context) (roles []string, err error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, helpers.GetSpanName(s.prefix, "GetRoles"))
	defer func() { helpers.FinishSpanWithError(span, err) }()

	return s.Service.GetRoles(ctx)
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

func (s *tracingService) CreateUserFromIdentity(ctx context.Context, identity contracts.UserIdentity) (user *contracts.User, err error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, helpers.GetSpanName(s.prefix, "CreateUserFromIdentity"))
	defer func() { helpers.FinishSpanWithError(span, err) }()

	return s.Service.CreateUserFromIdentity(ctx, identity)
}

func (s *tracingService) CreateUser(ctx context.Context, user contracts.User) (insertedUser *contracts.User, err error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, helpers.GetSpanName(s.prefix, "CreateUser"))
	defer func() { helpers.FinishSpanWithError(span, err) }()

	return s.Service.CreateUser(ctx, user)
}

func (s *tracingService) UpdateUser(ctx context.Context, user contracts.User) (err error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, helpers.GetSpanName(s.prefix, "UpdateUser"))
	defer func() { helpers.FinishSpanWithError(span, err) }()

	return s.Service.UpdateUser(ctx, user)
}

func (s *tracingService) CreateGroup(ctx context.Context, group contracts.Group) (insertedGroup *contracts.Group, err error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, helpers.GetSpanName(s.prefix, "CreateGroup"))
	defer func() { helpers.FinishSpanWithError(span, err) }()

	return s.Service.CreateGroup(ctx, group)
}

func (s *tracingService) UpdateGroup(ctx context.Context, group contracts.Group) (err error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, helpers.GetSpanName(s.prefix, "UpdateGroup"))
	defer func() { helpers.FinishSpanWithError(span, err) }()

	return s.Service.UpdateGroup(ctx, group)
}

func (s *tracingService) CreateOrganization(ctx context.Context, organization contracts.Organization) (insertedOrganization *contracts.Organization, err error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, helpers.GetSpanName(s.prefix, "CreateOrganization"))
	defer func() { helpers.FinishSpanWithError(span, err) }()

	return s.Service.CreateOrganization(ctx, organization)
}

func (s *tracingService) UpdateOrganization(ctx context.Context, organization contracts.Organization) (err error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, helpers.GetSpanName(s.prefix, "UpdateOrganization"))
	defer func() { helpers.FinishSpanWithError(span, err) }()

	return s.Service.UpdateOrganization(ctx, organization)
}

func (s *tracingService) CreateClient(ctx context.Context, client contracts.Client) (insertedClient *contracts.Client, err error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, helpers.GetSpanName(s.prefix, "CreateClient"))
	defer func() { helpers.FinishSpanWithError(span, err) }()

	return s.Service.CreateClient(ctx, client)
}

func (s *tracingService) UpdateClient(ctx context.Context, client contracts.Client) (err error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, helpers.GetSpanName(s.prefix, "UpdateClient"))
	defer func() { helpers.FinishSpanWithError(span, err) }()

	return s.Service.UpdateClient(ctx, client)
}
