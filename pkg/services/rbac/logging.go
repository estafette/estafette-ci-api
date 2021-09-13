package rbac

import (
	"context"

	"github.com/estafette/estafette-ci-api/pkg/api"
	contracts "github.com/estafette/estafette-ci-contracts"
)

// NewLoggingService returns a new instance of a logging Service.
func NewLoggingService(s Service) Service {
	return &loggingService{s, "oauth"}
}

type loggingService struct {
	Service Service
	prefix  string
}

func (s *loggingService) GetRoles(ctx context.Context) (roles []string, err error) {
	defer func() { api.HandleLogError(s.prefix, "Service", "GetRoles", err) }()

	return s.Service.GetRoles(ctx)
}

func (s *loggingService) GetProviders(ctx context.Context) (providers []*api.OAuthProvider, err error) {
	defer func() { api.HandleLogError(s.prefix, "Service", "GetProviders", err) }()

	return s.Service.GetProviders(ctx)
}

func (s *loggingService) GetProviderByName(ctx context.Context, organization, name string) (provider *api.OAuthProvider, err error) {
	defer func() { api.HandleLogError(s.prefix, "Service", "GetProviderByName", err) }()

	return s.Service.GetProviderByName(ctx, organization, name)
}

func (s *loggingService) GetUserByIdentity(ctx context.Context, identity contracts.UserIdentity) (user *contracts.User, err error) {
	defer func() { api.HandleLogError(s.prefix, "Service", "GetUserByIdentity", err) }()

	return s.Service.GetUserByIdentity(ctx, identity)
}

func (s *loggingService) CreateUserFromIdentity(ctx context.Context, identity contracts.UserIdentity) (user *contracts.User, err error) {
	defer func() { api.HandleLogError(s.prefix, "Service", "CreateUserFromIdentity", err) }()

	return s.Service.CreateUserFromIdentity(ctx, identity)
}

func (s *loggingService) CreateUser(ctx context.Context, user contracts.User) (insertedUser *contracts.User, err error) {
	defer func() { api.HandleLogError(s.prefix, "Service", "CreateUser", err) }()

	return s.Service.CreateUser(ctx, user)
}

func (s *loggingService) UpdateUser(ctx context.Context, user contracts.User) (err error) {
	defer func() { api.HandleLogError(s.prefix, "Service", "UpdateUser", err) }()

	return s.Service.UpdateUser(ctx, user)
}

func (s *loggingService) DeleteUser(ctx context.Context, id string) (err error) {
	defer func() { api.HandleLogError(s.prefix, "Service", "DeleteUser", err) }()

	return s.Service.DeleteUser(ctx, id)
}

func (s *loggingService) CreateGroup(ctx context.Context, group contracts.Group) (insertedGroup *contracts.Group, err error) {
	defer func() { api.HandleLogError(s.prefix, "Service", "CreateGroup", err) }()

	return s.Service.CreateGroup(ctx, group)
}

func (s *loggingService) UpdateGroup(ctx context.Context, group contracts.Group) (err error) {
	defer func() { api.HandleLogError(s.prefix, "Service", "UpdateGroup", err) }()

	return s.Service.UpdateGroup(ctx, group)
}

func (s *loggingService) DeleteGroup(ctx context.Context, id string) (err error) {
	defer func() { api.HandleLogError(s.prefix, "Service", "DeleteGroup", err) }()

	return s.Service.DeleteGroup(ctx, id)
}

func (s *loggingService) CreateOrganization(ctx context.Context, organization contracts.Organization) (insertedOrganization *contracts.Organization, err error) {
	defer func() { api.HandleLogError(s.prefix, "Service", "CreateOrganization", err) }()

	return s.Service.CreateOrganization(ctx, organization)
}

func (s *loggingService) UpdateOrganization(ctx context.Context, organization contracts.Organization) (err error) {
	defer func() { api.HandleLogError(s.prefix, "Service", "UpdateOrganization", err) }()

	return s.Service.UpdateOrganization(ctx, organization)
}

func (s *loggingService) DeleteOrganization(ctx context.Context, id string) (err error) {
	defer func() { api.HandleLogError(s.prefix, "Service", "DeleteOrganization", err) }()

	return s.Service.DeleteOrganization(ctx, id)
}

func (s *loggingService) CreateClient(ctx context.Context, client contracts.Client) (insertedClient *contracts.Client, err error) {
	defer func() { api.HandleLogError(s.prefix, "Service", "CreateClient", err) }()

	return s.Service.CreateClient(ctx, client)
}

func (s *loggingService) UpdateClient(ctx context.Context, client contracts.Client) (err error) {
	defer func() { api.HandleLogError(s.prefix, "Service", "UpdateClient", err) }()

	return s.Service.UpdateClient(ctx, client)
}

func (s *loggingService) DeleteClient(ctx context.Context, id string) (err error) {
	defer func() { api.HandleLogError(s.prefix, "Service", "DeleteClient", err) }()

	return s.Service.DeleteClient(ctx, id)
}

func (s *loggingService) UpdatePipeline(ctx context.Context, pipeline contracts.Pipeline) (err error) {
	defer func() { api.HandleLogError(s.prefix, "Service", "UpdatePipeline", err) }()

	return s.Service.UpdatePipeline(ctx, pipeline)
}

func (s *loggingService) GetInheritedRolesForUser(ctx context.Context, user contracts.User) (roles []*string, err error) {
	defer func() { api.HandleLogError(s.prefix, "Service", "GetInheritedRolesForUser", err) }()

	return s.Service.GetInheritedRolesForUser(ctx, user)
}

func (s *loggingService) GetInheritedOrganizationsForUser(ctx context.Context, user contracts.User) (organizations []*contracts.Organization, err error) {
	defer func() { api.HandleLogError(s.prefix, "Service", "GetInheritedOrganizationsForUser", err) }()

	return s.Service.GetInheritedOrganizationsForUser(ctx, user)
}
