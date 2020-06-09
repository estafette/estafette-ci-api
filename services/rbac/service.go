package rbac

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/estafette/estafette-ci-api/clients/cockroachdb"
	"github.com/estafette/estafette-ci-api/config"
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
type Service interface {
	GetProviders(ctx context.Context) (providers []*config.OAuthProvider, err error)
	GetProviderByName(ctx context.Context, name string) (provider *config.OAuthProvider, err error)

	CreateUser(ctx context.Context, identity contracts.UserIdentity) (user *contracts.User, err error)
	UpdateUser(ctx context.Context, user contracts.User) (err error)

	CreateGroup(ctx context.Context, group contracts.Group) (insertedGroup *contracts.Group, err error)
	UpdateGroup(ctx context.Context, group contracts.Group) (err error)

	CreateOrganization(ctx context.Context, organization contracts.Organization) (insertedOrganization *contracts.Organization, err error)
	UpdateOrganization(ctx context.Context, organization contracts.Organization) (err error)

	CreateClient(ctx context.Context, client contracts.Client) (insertedClient *contracts.Client, err error)
	UpdateClient(ctx context.Context, client contracts.Client) (err error)
}

// NewService returns a github.Service to handle incoming webhook events
func NewService(config *config.APIConfig, cockroachdbClient cockroachdb.Client) Service {
	return &service{
		config:            config,
		cockroachdbClient: cockroachdbClient,
	}
}

type service struct {
	config            *config.APIConfig
	cockroachdbClient cockroachdb.Client
}

func (s *service) GetProviders(ctx context.Context) (providers []*config.OAuthProvider, err error) {
	return s.config.Auth.OAuthProviders, nil
}

func (s *service) GetProviderByName(ctx context.Context, name string) (provider *config.OAuthProvider, err error) {
	providers, err := s.GetProviders(ctx)
	if err != nil {
		return
	}

	for _, p := range providers {
		if p.Name == name {
			return p, nil
		}
	}

	return nil, fmt.Errorf("Provider %v is not configured", name)
}

func (s *service) CreateUser(ctx context.Context, identity contracts.UserIdentity) (user *contracts.User, err error) {

	log.Info().Msgf("Creating record for user %v from provider %v", identity.Email, identity.Provider)

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

	return s.cockroachdbClient.InsertUser(ctx, *user)
}

func (s *service) UpdateUser(ctx context.Context, user contracts.User) (err error) {
	return s.cockroachdbClient.UpdateUser(ctx, user)
}

func (s *service) CreateGroup(ctx context.Context, group contracts.Group) (insertedGroup *contracts.Group, err error) {

	log.Info().Msgf("Creating record for group %v", group.Name)

	insertedGroup = &contracts.Group{
		Name: group.Name,
	}

	return s.cockroachdbClient.InsertGroup(ctx, *insertedGroup)
}

func (s *service) UpdateGroup(ctx context.Context, group contracts.Group) (err error) {
	return s.cockroachdbClient.UpdateGroup(ctx, group)
}

func (s *service) CreateOrganization(ctx context.Context, organization contracts.Organization) (insertedOrganization *contracts.Organization, err error) {

	log.Info().Msgf("Creating record for organization %v", organization.Name)

	insertedOrganization = &contracts.Organization{
		Name: organization.Name,
	}

	return s.cockroachdbClient.InsertOrganization(ctx, *insertedOrganization)
}

func (s *service) UpdateOrganization(ctx context.Context, organization contracts.Organization) (err error) {
	return s.cockroachdbClient.UpdateOrganization(ctx, organization)
}

func (s *service) CreateClient(ctx context.Context, client contracts.Client) (insertedClient *contracts.Client, err error) {

	log.Info().Msgf("Creating record for client %v", client.Name)

	insertedClient = &contracts.Client{
		Name:     client.Name,
		ClientID: uuid.New().String(),
	}

	// generate random client id and client secret
	clientSecret, err := password.Generate(64, 10, 10, false, false)
	if err != nil {
		return nil, err
	}

	insertedClient.ClientSecret = clientSecret

	return s.cockroachdbClient.InsertClient(ctx, *insertedClient)
}

func (s *service) UpdateClient(ctx context.Context, client contracts.Client) (err error) {
	return s.cockroachdbClient.UpdateClient(ctx, client)
}
