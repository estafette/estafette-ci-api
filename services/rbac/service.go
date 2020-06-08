package rbac

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/estafette/estafette-ci-api/clients/cockroachdb"
	"github.com/estafette/estafette-ci-api/config"
	contracts "github.com/estafette/estafette-ci-contracts"
	"github.com/rs/zerolog/log"
)

var (
	// ErrUserNotFound indicates that a user cannot be found in the database
	ErrUserNotFound = errors.New("The user can't be found")
)

// Service handles http requests for role-based-access-control
type Service interface {
	GetProviders(ctx context.Context) (providers []*config.OAuthProvider, err error)
	GetProviderByName(ctx context.Context, name string) (provider *config.OAuthProvider, err error)
	GetUserByIdentity(ctx context.Context, identity contracts.UserIdentity) (user *contracts.User, err error)
	GetUserByID(ctx context.Context, id string) (user *contracts.User, err error)
	CreateUser(ctx context.Context, identity contracts.UserIdentity) (user *contracts.User, err error)
	UpdateUser(ctx context.Context, user contracts.User) (err error)
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

func (s *service) GetUserByIdentity(ctx context.Context, identity contracts.UserIdentity) (user *contracts.User, err error) {

	user, err = s.cockroachdbClient.GetUserByIdentity(ctx, identity)

	if err != nil {
		if errors.Is(err, cockroachdb.ErrUserNotFound) {
			return nil, ErrUserNotFound
		}

		return nil, err
	}

	return user, nil
}

func (s *service) GetUserByID(ctx context.Context, id string) (user *contracts.User, err error) {
	user, err = s.cockroachdbClient.GetUserByID(ctx, id)

	if err != nil {
		if errors.Is(err, cockroachdb.ErrUserNotFound) {
			return nil, ErrUserNotFound
		}

		return nil, err
	}

	return user, nil
}

func (s *service) CreateUser(ctx context.Context, identity contracts.UserIdentity) (user *contracts.User, err error) {

	log.Info().Msgf("Creating user record for user %v from provider %v", identity.Email, identity.Provider)

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
