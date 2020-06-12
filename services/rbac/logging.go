package rbac

import (
	"context"

	"github.com/estafette/estafette-ci-api/config"
	"github.com/estafette/estafette-ci-api/helpers"
	contracts "github.com/estafette/estafette-ci-contracts"
)

// NewLoggingService returns a new instance of a logging Service.
func NewLoggingService(s Service) Service {
	return &loggingService{s, "oauth"}
}

type loggingService struct {
	Service
	prefix string
}

func (s *loggingService) GetRoles(ctx context.Context) (roles []string, err error) {
	defer func() { helpers.HandleLogError(s.prefix, "GetRoles", err) }()

	return s.Service.GetRoles(ctx)
}

func (s *loggingService) GetProviders(ctx context.Context) (providers []*config.OAuthProvider, err error) {
	defer func() { helpers.HandleLogError(s.prefix, "GetProviders", err) }()

	return s.Service.GetProviders(ctx)
}

func (s *loggingService) GetProviderByName(ctx context.Context, name string) (provider *config.OAuthProvider, err error) {
	defer func() { helpers.HandleLogError(s.prefix, "GetProviderByName", err) }()

	return s.Service.GetProviderByName(ctx, name)
}

func (s *loggingService) CreateUserFromIdentity(ctx context.Context, identity contracts.UserIdentity) (user *contracts.User, err error) {
	defer func() { helpers.HandleLogError(s.prefix, "CreateUserFromIdentity", err) }()

	return s.Service.CreateUserFromIdentity(ctx, identity)
}

func (s *loggingService) CreateUser(ctx context.Context, user contracts.User) (insertedUser *contracts.User, err error) {
	defer func() { helpers.HandleLogError(s.prefix, "CreateUser", err) }()

	return s.Service.CreateUser(ctx, user)
}

func (s *loggingService) UpdateUser(ctx context.Context, user contracts.User) (err error) {
	defer func() { helpers.HandleLogError(s.prefix, "UpdateUser", err) }()

	return s.Service.UpdateUser(ctx, user)
}

func (s *loggingService) CreateGroup(ctx context.Context, group contracts.Group) (insertedGroup *contracts.Group, err error) {
	defer func() { helpers.HandleLogError(s.prefix, "CreateGroup", err) }()

	return s.Service.CreateGroup(ctx, group)
}

func (s *loggingService) UpdateGroup(ctx context.Context, group contracts.Group) (err error) {
	defer func() { helpers.HandleLogError(s.prefix, "UpdateGroup", err) }()

	return s.Service.UpdateGroup(ctx, group)
}

func (s *loggingService) CreateOrganization(ctx context.Context, organization contracts.Organization) (insertedOrganization *contracts.Organization, err error) {
	defer func() { helpers.HandleLogError(s.prefix, "CreateOrganization", err) }()

	return s.Service.CreateOrganization(ctx, organization)
}

func (s *loggingService) UpdateOrganization(ctx context.Context, organization contracts.Organization) (err error) {
	defer func() { helpers.HandleLogError(s.prefix, "UpdateOrganization", err) }()

	return s.Service.UpdateOrganization(ctx, organization)
}

func (s *loggingService) CreateClient(ctx context.Context, client contracts.Client) (insertedClient *contracts.Client, err error) {
	defer func() { helpers.HandleLogError(s.prefix, "CreateClient", err) }()

	return s.Service.CreateClient(ctx, client)
}

func (s *loggingService) UpdateClient(ctx context.Context, client contracts.Client) (err error) {
	defer func() { helpers.HandleLogError(s.prefix, "UpdateClient", err) }()

	return s.Service.UpdateClient(ctx, client)
}
