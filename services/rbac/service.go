package rbac

import (
	"context"
	"errors"
	"time"

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
	GetUser(ctx context.Context, identity contracts.UserIdentity) (user *contracts.User, err error)
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

func (s *service) GetUser(ctx context.Context, identity contracts.UserIdentity) (user *contracts.User, err error) {

	user, err = s.cockroachdbClient.GetUserByIdentity(ctx, identity)

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
		Identities: []*contracts.UserIdentity{
			&identity,
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

func (s *service) UpdateUser(ctx context.Context, user contracts.User) (err error) {
	err = s.cockroachdbClient.UpdateUser(ctx, user)
	if err != nil {
		return
	}

	return nil
}