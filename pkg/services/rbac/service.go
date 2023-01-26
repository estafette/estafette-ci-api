package rbac

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/estafette/estafette-ci-api/pkg/api"
	"github.com/estafette/estafette-ci-api/pkg/clients/database"
	contracts "github.com/estafette/estafette-ci-contracts"
	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
	"github.com/sethvargo/go-password/password"
)

var (
	// ErrUserNotFound indicates that a user cannot be found in the database
	ErrUserNotFound = errors.New("The user can't be found")
)

// Service handles http requests for role-based-access-control
//
//go:generate mockgen -package=rbac -destination ./mock.go -source=service.go
type Service interface {
	GetRoles(ctx context.Context) (roles []string, err error)

	GetProviders(ctx context.Context) (providers []*api.OAuthProvider, err error)
	GetProviderByName(ctx context.Context, organization, name string) (provider *api.OAuthProvider, err error)

	GetUserByIdentity(ctx context.Context, identity contracts.UserIdentity) (user *contracts.User, err error)
	CreateUserFromIdentity(ctx context.Context, identity contracts.UserIdentity) (user *contracts.User, err error)
	CreateUser(ctx context.Context, user contracts.User) (insertedUser *contracts.User, err error)
	UpdateUser(ctx context.Context, user contracts.User) (err error)
	DeleteUser(ctx context.Context, id string) (err error)

	CreateGroup(ctx context.Context, group contracts.Group) (insertedGroup *contracts.Group, err error)
	UpdateGroup(ctx context.Context, group contracts.Group) (err error)
	DeleteGroup(ctx context.Context, id string) (err error)

	CreateOrganization(ctx context.Context, organization contracts.Organization) (insertedOrganization *contracts.Organization, err error)
	UpdateOrganization(ctx context.Context, organization contracts.Organization) (err error)
	DeleteOrganization(ctx context.Context, id string) (err error)

	CreateClient(ctx context.Context, client contracts.Client) (insertedClient *contracts.Client, err error)
	UpdateClient(ctx context.Context, client contracts.Client) (err error)
	DeleteClient(ctx context.Context, id string) (err error)

	UpdatePipeline(ctx context.Context, pipeline contracts.Pipeline) (err error)

	GetInheritedRolesForUser(ctx context.Context, user contracts.User) (roles []*string, err error)
	GetInheritedOrganizationsForUser(ctx context.Context, user contracts.User) (organizations []*contracts.Organization, err error)
}

// NewService returns a github.Service to handle incoming webhook events
func NewService(config *api.APIConfig, databaseClient database.Client) Service {
	return &service{
		config:         config,
		databaseClient: databaseClient,
	}
}

type service struct {
	config         *api.APIConfig
	databaseClient database.Client
}

func (s *service) GetRoles(ctx context.Context) (roles []string, err error) {
	return api.Roles(), nil
}

func (s *service) GetProviders(ctx context.Context) (providers []*api.OAuthProvider, err error) {

	providers = make([]*api.OAuthProvider, 0)

	for _, c := range s.config.Auth.Organizations {
		for _, p := range c.OAuthProviders {
			if p != nil && p.ClientID != "" {
				providers = append(providers, p)
			}
		}
	}

	if s.config.Auth.Google != nil && s.config.Auth.Google.ClientID != "" {
		providers = append(providers, s.config.Auth.Google)
	}

	if s.config.Auth.Github != nil && s.config.Auth.Github.ClientID != "" {
		providers = append(providers, s.config.Auth.Github)
	}

	return providers, nil
}

func (s *service) GetProviderByName(ctx context.Context, organization, name string) (provider *api.OAuthProvider, err error) {
	providers, err := s.GetProviders(ctx)
	if err != nil {
		return
	}

	for _, provider := range providers {
		if organization == "" || provider.Organization == organization {
			if provider.Name == name {
				return provider, nil
			}
		}
	}

	return nil, fmt.Errorf("Provider %v is not configured", name)
}

func (s *service) GetUserByIdentity(ctx context.Context, identity contracts.UserIdentity) (user *contracts.User, err error) {

	user, err = s.databaseClient.GetUserByIdentity(ctx, identity)
	if err != nil {
		return nil, err
	}

	s.setAdminRoleForUserIfConfigured(user)

	return user, nil
}

func (s *service) CreateUserFromIdentity(ctx context.Context, identity contracts.UserIdentity) (user *contracts.User, err error) {

	log.Debug().Msgf("Creating record for user %v from provider %v", identity.Email, identity.Provider)

	firstVisit := time.Now().UTC()

	user = &contracts.User{
		Active: true,
		Name:   identity.Name,
		Email:  identity.Email,
		Identities: []*contracts.UserIdentity{
			&identity,
		},
		FirstVisit:      &firstVisit,
		LastVisit:       &firstVisit,
		CurrentProvider: identity.Provider,
	}

	s.setAdminRoleForUserIfConfigured(user)

	return s.databaseClient.InsertUser(ctx, *user)
}

func (s *service) CreateUser(ctx context.Context, user contracts.User) (insertedUser *contracts.User, err error) {

	log.Debug().Msgf("Creating record for user %v", user.Email)

	firstVisit := time.Now().UTC()

	insertedUser = &contracts.User{
		Active:        true,
		Name:          user.Name,
		Email:         user.Email,
		Identities:    user.Identities,
		Groups:        user.Groups,
		Organizations: user.Organizations,
		Roles:         user.Roles,
		Preferences:   user.Preferences,
		FirstVisit:    &firstVisit,
	}

	s.setAdminRoleForUserIfConfigured(insertedUser)

	return s.databaseClient.InsertUser(ctx, *insertedUser)
}

func (s *service) UpdateUser(ctx context.Context, user contracts.User) (err error) {

	// get user from db
	currentUser, err := s.databaseClient.GetUserByID(ctx, user.ID, map[api.FilterType][]string{})
	if err != nil {
		return
	}
	if currentUser == nil {
		return fmt.Errorf("User is nil")
	}

	// copy updateable fields
	currentUser.Name = user.Name
	currentUser.Email = user.Email
	currentUser.Active = user.Active
	currentUser.Identities = user.Identities
	currentUser.Groups = user.Groups
	currentUser.Organizations = user.Organizations
	currentUser.Roles = user.Roles
	currentUser.Preferences = user.Preferences
	currentUser.LastVisit = user.LastVisit
	currentUser.CurrentProvider = user.CurrentProvider
	currentUser.CurrentOrganization = user.CurrentOrganization

	s.setAdminRoleForUserIfConfigured(currentUser)

	return s.databaseClient.UpdateUser(ctx, *currentUser)
}

func (s *service) DeleteUser(ctx context.Context, id string) (err error) {

	// get user from db
	currentUser, err := s.databaseClient.GetUserByID(ctx, id, map[api.FilterType][]string{})
	if err != nil {
		return
	}
	if currentUser == nil {
		return fmt.Errorf("User is nil")
	}

	return s.databaseClient.DeleteUser(ctx, *currentUser)
}
func (s *service) CreateGroup(ctx context.Context, group contracts.Group) (insertedGroup *contracts.Group, err error) {

	log.Debug().Msgf("Creating record for group %v", group.Name)

	insertedGroup = &contracts.Group{
		Name:          group.Name,
		Description:   group.Description,
		Identities:    group.Identities,
		Organizations: group.Organizations,
		Roles:         group.Roles,
	}

	return s.databaseClient.InsertGroup(ctx, *insertedGroup)
}

func (s *service) UpdateGroup(ctx context.Context, group contracts.Group) (err error) {

	// get group from db
	currentGroup, err := s.databaseClient.GetGroupByID(ctx, group.ID, map[api.FilterType][]string{})
	if err != nil {
		return
	}
	if currentGroup == nil {
		return fmt.Errorf("Group is nil")
	}

	// copy updateable fields
	currentGroup.Name = group.Name
	currentGroup.Description = group.Description
	currentGroup.Identities = group.Identities
	currentGroup.Organizations = group.Organizations
	currentGroup.Roles = group.Roles

	return s.databaseClient.UpdateGroup(ctx, *currentGroup)
}

func (s *service) DeleteGroup(ctx context.Context, id string) (err error) {

	// get group from db
	currentGroup, err := s.databaseClient.GetGroupByID(ctx, id, map[api.FilterType][]string{})
	if err != nil {
		return
	}
	if currentGroup == nil {
		return fmt.Errorf("Group is nil")
	}

	return s.databaseClient.DeleteGroup(ctx, *currentGroup)
}

func (s *service) CreateOrganization(ctx context.Context, organization contracts.Organization) (insertedOrganization *contracts.Organization, err error) {

	log.Debug().Msgf("Creating record for organization %v", organization.Name)

	insertedOrganization = &contracts.Organization{
		Name:       organization.Name,
		Identities: organization.Identities,
		Roles:      organization.Roles,
	}

	return s.databaseClient.InsertOrganization(ctx, *insertedOrganization)
}

func (s *service) UpdateOrganization(ctx context.Context, organization contracts.Organization) (err error) {

	// get organization from db
	currentOrganization, err := s.databaseClient.GetOrganizationByID(ctx, organization.ID)
	if err != nil {
		return
	}
	if currentOrganization == nil {
		return fmt.Errorf("Organization is nil")
	}

	// copy updateable fields
	currentOrganization.Name = organization.Name
	currentOrganization.Identities = organization.Identities
	currentOrganization.Roles = organization.Roles

	return s.databaseClient.UpdateOrganization(ctx, *currentOrganization)
}

func (s *service) DeleteOrganization(ctx context.Context, id string) (err error) {

	// get organization from db
	currentOrganization, err := s.databaseClient.GetOrganizationByID(ctx, id)
	if err != nil {
		return
	}
	if currentOrganization == nil {
		return fmt.Errorf("Organization is nil")
	}

	return s.databaseClient.DeleteOrganization(ctx, *currentOrganization)
}

func (s *service) CreateClient(ctx context.Context, client contracts.Client) (insertedClient *contracts.Client, err error) {

	log.Debug().Msgf("Creating record for client %v", client.Name)

	insertedClient = &contracts.Client{}

	// set updateable fields
	insertedClient.Name = client.Name
	insertedClient.Roles = client.Roles
	insertedClient.Organizations = client.Organizations
	insertedClient.Active = true

	// generate random client id and client secret
	clientSecret, err := password.Generate(64, 10, 0, false, true)
	if err != nil {
		return nil, err
	}

	// set immutable fields
	insertedClient.ClientSecret = clientSecret
	insertedClient.ClientID = uuid.New().String()

	return s.databaseClient.InsertClient(ctx, *insertedClient)
}

func (s *service) UpdateClient(ctx context.Context, client contracts.Client) (err error) {

	// get client from db
	currentClient, err := s.databaseClient.GetClientByID(ctx, client.ID)
	if err != nil {
		return
	}
	if currentClient == nil {
		return fmt.Errorf("Client is nil")
	}

	// copy updateable fields
	currentClient.Name = client.Name
	currentClient.Roles = client.Roles
	currentClient.Organizations = client.Organizations

	return s.databaseClient.UpdateClient(ctx, *currentClient)
}

func (s *service) DeleteClient(ctx context.Context, id string) (err error) {

	// get client from db
	currentClient, err := s.databaseClient.GetClientByID(ctx, id)
	if err != nil {
		return
	}
	if currentClient == nil {
		return fmt.Errorf("Client is nil")
	}

	return s.databaseClient.DeleteClient(ctx, *currentClient)
}

func (s *service) UpdatePipeline(ctx context.Context, pipeline contracts.Pipeline) (err error) {
	// get pipeline from db
	currentPipeline, err := s.databaseClient.GetPipeline(ctx, pipeline.RepoSource, pipeline.RepoOwner, pipeline.RepoName, map[api.FilterType][]string{}, true)
	if err != nil {
		return
	}
	if currentPipeline == nil {
		return fmt.Errorf("Pipeline is nil")
	}

	// copy updateable fields
	currentPipeline.Groups = pipeline.Groups
	currentPipeline.Organizations = pipeline.Organizations
	currentPipeline.Archived = pipeline.Archived

	return s.databaseClient.UpdateComputedPipelinePermissions(ctx, *currentPipeline)
}

func (s *service) GetInheritedRolesForUser(ctx context.Context, user contracts.User) (roles []*string, err error) {

	retrievedRoles := make([]*string, 0)

	// get direct roles from user
	retrievedRoles = append(retrievedRoles, user.Roles...)

	// get roles from groups linked to user
	for _, g := range user.Groups {
		group, err := s.databaseClient.GetGroupByID(ctx, g.ID, map[api.FilterType][]string{})
		if err != nil {
			return nil, err
		}
		retrievedRoles = append(retrievedRoles, group.Roles...)

		// get roles from organizations linked to groups
		for _, o := range group.Organizations {
			organization, err := s.databaseClient.GetOrganizationByID(ctx, o.ID)
			if err != nil {
				return nil, err
			}
			retrievedRoles = append(retrievedRoles, organization.Roles...)
		}
	}

	// get roles from organizations linked to user
	for _, o := range user.Organizations {
		organization, err := s.databaseClient.GetOrganizationByID(ctx, o.ID)
		if err != nil {
			return nil, err
		}
		retrievedRoles = append(retrievedRoles, organization.Roles...)
	}

	return s.dedupeRoles(retrievedRoles), nil
}

func (s *service) dedupeRoles(retrievedRoles []*string) (roles []*string) {
	roles = make([]*string, 0)
	for _, r := range retrievedRoles {
		isInRoles := false
		for _, rr := range roles {
			if r != nil && rr != nil && *r == *rr {
				isInRoles = true
				break
			}
		}
		if !isInRoles {
			roles = append(roles, r)
		}
	}

	return roles
}

func (s *service) GetInheritedOrganizationsForUser(ctx context.Context, user contracts.User) (organizations []*contracts.Organization, err error) {
	retrievedOrganizations := make([]*contracts.Organization, 0)

	// get direct roles from user
	retrievedOrganizations = append(retrievedOrganizations, user.Organizations...)

	// get organizations from groups linked to user
	for _, g := range user.Groups {
		// get roles from organizations linked to groups
		retrievedOrganizations = append(retrievedOrganizations, g.Organizations...)
	}

	return s.dedupeOrganizations(retrievedOrganizations), nil
}

func (s *service) dedupeOrganizations(retrievedOrganizations []*contracts.Organization) (organizations []*contracts.Organization) {
	organizations = make([]*contracts.Organization, 0)
	for _, o := range retrievedOrganizations {
		isInOrganizations := false
		for _, oo := range organizations {
			if o != nil && oo != nil && o.ID == oo.ID {
				isInOrganizations = true
				break
			}
		}
		if !isInOrganizations {
			organizations = append(organizations, o)
		}
	}

	return organizations
}

func (s *service) setAdminRoleForUserIfConfigured(user *contracts.User) {
	// check if email matches configured administrators and add/remove administrator role correspondingly
	if s.config.Auth.IsConfiguredAsAdministrator(user.Email) {
		// ensure user has administrator role
		user.AddRole(api.RoleAdministrator.String())
	} else {
		// ensure user does not have administrator role
		user.RemoveRole(api.RoleAdministrator.String())
	}
}
