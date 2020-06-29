package rbac

import (
	"context"

	"github.com/estafette/estafette-ci-api/config"
	contracts "github.com/estafette/estafette-ci-contracts"
)

type MockService struct {
	GetRolesFunc                 func(ctx context.Context) (roles []string, err error)
	GetProvidersFunc             func(ctx context.Context) (providers map[string][]*config.OAuthProvider, err error)
	GetProviderByNameFunc        func(ctx context.Context, organization, name string) (provider *config.OAuthProvider, err error)
	GetUserByIdentityFunc        func(ctx context.Context, identity contracts.UserIdentity) (user *contracts.User, err error)
	CreateUserFromIdentityFunc   func(ctx context.Context, identity contracts.UserIdentity) (user *contracts.User, err error)
	CreateUserFunc               func(ctx context.Context, user contracts.User) (insertedUser *contracts.User, err error)
	UpdateUserFunc               func(ctx context.Context, user contracts.User) (err error)
	DeleteUserFunc               func(ctx context.Context, id string) (err error)
	CreateGroupFunc              func(ctx context.Context, group contracts.Group) (insertedGroup *contracts.Group, err error)
	UpdateGroupFunc              func(ctx context.Context, group contracts.Group) (err error)
	DeleteGroupFunc              func(ctx context.Context, id string) (err error)
	CreateOrganizationFunc       func(ctx context.Context, organization contracts.Organization) (insertedOrganization *contracts.Organization, err error)
	UpdateOrganizationFunc       func(ctx context.Context, organization contracts.Organization) (err error)
	DeleteOrganizationFunc       func(ctx context.Context, id string) (err error)
	CreateClientFunc             func(ctx context.Context, client contracts.Client) (insertedClient *contracts.Client, err error)
	UpdateClientFunc             func(ctx context.Context, client contracts.Client) (err error)
	DeleteClientFunc             func(ctx context.Context, id string) (err error)
	UpdatePipelineFunc           func(ctx context.Context, pipeline contracts.Pipeline) (err error)
	ArchivePipelineFunc          func(ctx context.Context, repoSource, repoOwner, repoName string) (err error)
	GetInheritedRolesForUserFunc func(ctx context.Context, user contracts.User) (roles []*string, err error)
}

func (s MockService) GetRoles(ctx context.Context) (roles []string, err error) {
	if s.GetRolesFunc == nil {
		return
	}
	return s.GetRolesFunc(ctx)
}

func (s MockService) GetProviders(ctx context.Context) (providers map[string][]*config.OAuthProvider, err error) {
	if s.GetProvidersFunc == nil {
		return
	}
	return s.GetProvidersFunc(ctx)
}

func (s MockService) GetProviderByName(ctx context.Context, organization, name string) (provider *config.OAuthProvider, err error) {
	if s.GetProviderByNameFunc == nil {
		return
	}
	return s.GetProviderByNameFunc(ctx, organization, name)
}

func (s MockService) GetUserByIdentity(ctx context.Context, identity contracts.UserIdentity) (user *contracts.User, err error) {
	if s.GetUserByIdentityFunc == nil {
		return
	}
	return s.GetUserByIdentityFunc(ctx, identity)
}

func (s MockService) CreateUserFromIdentity(ctx context.Context, identity contracts.UserIdentity) (user *contracts.User, err error) {
	if s.CreateUserFromIdentityFunc == nil {
		return
	}
	return s.CreateUserFromIdentityFunc(ctx, identity)
}

func (s MockService) CreateUser(ctx context.Context, user contracts.User) (insertedUser *contracts.User, err error) {
	if s.CreateUserFunc == nil {
		return
	}
	return s.CreateUserFunc(ctx, user)
}

func (s MockService) UpdateUser(ctx context.Context, user contracts.User) (err error) {
	if s.UpdateUserFunc == nil {
		return
	}
	return s.UpdateUserFunc(ctx, user)
}

func (s MockService) DeleteUser(ctx context.Context, id string) (err error) {
	if s.DeleteUserFunc == nil {
		return
	}
	return s.DeleteUserFunc(ctx, id)
}

func (s MockService) CreateGroup(ctx context.Context, group contracts.Group) (insertedGroup *contracts.Group, err error) {
	if s.CreateGroupFunc == nil {
		return
	}
	return s.CreateGroupFunc(ctx, group)
}

func (s MockService) UpdateGroup(ctx context.Context, group contracts.Group) (err error) {
	if s.UpdateGroupFunc == nil {
		return
	}
	return s.UpdateGroupFunc(ctx, group)
}

func (s MockService) DeleteGroup(ctx context.Context, id string) (err error) {
	if s.DeleteGroupFunc == nil {
		return
	}
	return s.DeleteGroupFunc(ctx, id)
}

func (s MockService) CreateOrganization(ctx context.Context, organization contracts.Organization) (insertedOrganization *contracts.Organization, err error) {
	if s.CreateOrganizationFunc == nil {
		return
	}
	return s.CreateOrganizationFunc(ctx, organization)
}

func (s MockService) UpdateOrganization(ctx context.Context, organization contracts.Organization) (err error) {
	if s.UpdateOrganizationFunc == nil {
		return
	}
	return s.UpdateOrganizationFunc(ctx, organization)
}

func (s MockService) DeleteOrganization(ctx context.Context, id string) (err error) {
	if s.DeleteOrganizationFunc == nil {
		return
	}
	return s.DeleteOrganizationFunc(ctx, id)
}

func (s MockService) CreateClient(ctx context.Context, client contracts.Client) (insertedClient *contracts.Client, err error) {
	if s.CreateClientFunc == nil {
		return
	}
	return s.CreateClientFunc(ctx, client)
}

func (s MockService) UpdateClient(ctx context.Context, client contracts.Client) (err error) {
	if s.UpdateClientFunc == nil {
		return
	}
	return s.UpdateClientFunc(ctx, client)
}

func (s MockService) DeleteClient(ctx context.Context, id string) (err error) {
	if s.DeleteClientFunc == nil {
		return
	}
	return s.DeleteClientFunc(ctx, id)
}

func (s MockService) UpdatePipeline(ctx context.Context, pipeline contracts.Pipeline) (err error) {
	if s.UpdatePipelineFunc == nil {
		return
	}
	return s.UpdatePipelineFunc(ctx, pipeline)
}

func (s MockService) ArchivePipeline(ctx context.Context, repoSource, repoOwner, repoName string) (err error) {
	if s.ArchivePipelineFunc == nil {
		return
	}
	return s.ArchivePipelineFunc(ctx, repoSource, repoOwner, repoName)
}

func (s MockService) GetInheritedRolesForUser(ctx context.Context, user contracts.User) (roles []*string, err error) {
	if s.GetInheritedRolesForUserFunc == nil {
		return
	}
	return s.GetInheritedRolesForUserFunc(ctx, user)
}
