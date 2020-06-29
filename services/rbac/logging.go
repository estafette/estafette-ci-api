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

func (s *loggingService) GetProviders(ctx context.Context) (providers map[string][]*config.OAuthProvider, err error) {
	defer func() { helpers.HandleLogError(s.prefix, "GetProviders", err) }()

	return s.Service.GetProviders(ctx)
}

func (s *loggingService) GetProviderByName(ctx context.Context, organization, name string) (provider *config.OAuthProvider, err error) {
	defer func() { helpers.HandleLogError(s.prefix, "GetProviderByName", err) }()

	return s.Service.GetProviderByName(ctx, organization, name)
}

func (s *loggingService) GetUserByIdentity(ctx context.Context, identity contracts.UserIdentity) (user *contracts.User, err error) {
	defer func() { helpers.HandleLogError(s.prefix, "GetUserByIdentity", err) }()

	return s.Service.GetUserByIdentity(ctx, identity)
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

func (s *loggingService) DeleteUser(ctx context.Context, id string) (err error) {
	defer func() { helpers.HandleLogError(s.prefix, "DeleteUser", err) }()

	return s.Service.DeleteUser(ctx, id)
}

func (s *loggingService) CreateGroup(ctx context.Context, group contracts.Group) (insertedGroup *contracts.Group, err error) {
	defer func() { helpers.HandleLogError(s.prefix, "CreateGroup", err) }()

	return s.Service.CreateGroup(ctx, group)
}

func (s *loggingService) UpdateGroup(ctx context.Context, group contracts.Group) (err error) {
	defer func() { helpers.HandleLogError(s.prefix, "UpdateGroup", err) }()

	return s.Service.UpdateGroup(ctx, group)
}

func (s *loggingService) DeleteGroup(ctx context.Context, id string) (err error) {
	defer func() { helpers.HandleLogError(s.prefix, "DeleteGroup", err) }()

	return s.Service.DeleteGroup(ctx, id)
}

func (s *loggingService) CreateOrganization(ctx context.Context, organization contracts.Organization) (insertedOrganization *contracts.Organization, err error) {
	defer func() { helpers.HandleLogError(s.prefix, "CreateOrganization", err) }()

	return s.Service.CreateOrganization(ctx, organization)
}

func (s *loggingService) UpdateOrganization(ctx context.Context, organization contracts.Organization) (err error) {
	defer func() { helpers.HandleLogError(s.prefix, "UpdateOrganization", err) }()

	return s.Service.UpdateOrganization(ctx, organization)
}

func (s *loggingService) DeleteOrganization(ctx context.Context, id string) (err error) {
	defer func() { helpers.HandleLogError(s.prefix, "DeleteOrganization", err) }()

	return s.Service.DeleteOrganization(ctx, id)
}

func (s *loggingService) CreateClient(ctx context.Context, client contracts.Client) (insertedClient *contracts.Client, err error) {
	defer func() { helpers.HandleLogError(s.prefix, "CreateClient", err) }()

	return s.Service.CreateClient(ctx, client)
}

func (s *loggingService) UpdateClient(ctx context.Context, client contracts.Client) (err error) {
	defer func() { helpers.HandleLogError(s.prefix, "UpdateClient", err) }()

	return s.Service.UpdateClient(ctx, client)
}

func (s *loggingService) DeleteClient(ctx context.Context, id string) (err error) {
	defer func() { helpers.HandleLogError(s.prefix, "DeleteClient", err) }()

	return s.Service.DeleteClient(ctx, id)
}

func (s *loggingService) UpdatePipeline(ctx context.Context, pipeline contracts.Pipeline) (err error) {
	defer func() { helpers.HandleLogError(s.prefix, "UpdatePipeline", err) }()

	return s.Service.UpdatePipeline(ctx, pipeline)
}

func (s *loggingService) TogglePipelineArchival(ctx context.Context, repoSource, repoOwner, repoName string) (err error) {
	defer func() { helpers.HandleLogError(s.prefix, "TogglePipelineArchival", err) }()

	return s.Service.TogglePipelineArchival(ctx, repoSource, repoOwner, repoName)
}

func (s *loggingService) GetInheritedRolesForUser(ctx context.Context, user contracts.User) (roles []*string, err error) {
	defer func() { helpers.HandleLogError(s.prefix, "GetInheritedRolesForUser", err) }()

	return s.Service.GetInheritedRolesForUser(ctx, user)
}
