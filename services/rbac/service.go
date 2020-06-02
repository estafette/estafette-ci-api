package rbac

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/estafette/estafette-ci-api/auth"
	"github.com/estafette/estafette-ci-api/clients/cockroachdb"
	"github.com/estafette/estafette-ci-api/config"
	contracts "github.com/estafette/estafette-ci-contracts"
	"github.com/rs/zerolog/log"
)

var (
	ErrUserNotFound = errors.New("The user can't be found")
)

// Service handles http requests for role-based-access-control
type Service interface {
	GetProviders(ctx context.Context) (providers []*config.OAuthProvider, err error)
	GetUser(ctx context.Context, authUser auth.User) (user *contracts.User, err error)
	CreateUser(ctx context.Context, authUser auth.User) (user *contracts.User, err error)
	UpdateUser(ctx context.Context, authUser auth.User) (err error)
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

func (s *service) GetUser(ctx context.Context, authUser auth.User) (user *contracts.User, err error) {
	if !authUser.Authenticated {
		return nil, fmt.Errorf("User %v is not authenticated, won't fetch user record from database", authUser.Email)
	}

	user, err = s.cockroachdbClient.GetUserByEmail(ctx, authUser.Email)

	if err != nil {
		if errors.Is(err, cockroachdb.ErrUserNotFound) {
			return nil, ErrUserNotFound
		}

		return nil, err
	}

	return user, nil
}

func (s *service) CreateUser(ctx context.Context, authUser auth.User) (user *contracts.User, err error) {
	if !authUser.Authenticated {
		return nil, fmt.Errorf("User %v is not authenticated, won't create user record in database", authUser.Email)
	}

	log.Info().Msgf("Creating user record for user %v from provider %v", authUser.Email, authUser.Provider)

	firstVisit := time.Now().UTC()

	username := ""
	if authUser.User != nil {
		username = authUser.User.Name
	}

	user = &contracts.User{
		Active: true,
		Name:   username,
		Identities: []*contracts.UserIdentity{
			{
				Provider: authUser.Provider,
				Name:     username,
				Email:    authUser.Email,
			},
		},
		FirstVisit: &firstVisit,
		LastVisit:  &firstVisit,
	}

	user, err = s.cockroachdbClient.InsertUser(ctx, *user)
	if err != nil {
		return nil, err
	}

	return user, nil
}

func (s *service) UpdateUser(ctx context.Context, authUser auth.User) (err error) {
	if !authUser.Authenticated || authUser.User == nil {
		return fmt.Errorf("User %v is not authenticated, won't update user record in database", authUser.Email)
	}

	err = s.cockroachdbClient.UpdateUser(ctx, *authUser.User)
	if err != nil {
		return
	}

	return nil
}
