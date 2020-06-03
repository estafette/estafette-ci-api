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

func (s *loggingService) GetProviders(ctx context.Context) (providers []*config.OAuthProvider, err error) {
	defer func() { helpers.HandleLogError(s.prefix, "GetProviders", err) }()

	return s.Service.GetProviders(ctx)
}

func (s *loggingService) GetUser(ctx context.Context, identity contracts.UserIdentity) (user *contracts.User, err error) {
	defer func() { helpers.HandleLogError(s.prefix, "GetUser", err, ErrUserNotFound) }()

	return s.Service.GetUser(ctx, identity)
}

func (s *loggingService) CreateUser(ctx context.Context, identity contracts.UserIdentity) (user *contracts.User, err error) {
	defer func() { helpers.HandleLogError(s.prefix, "CreateUser", err) }()

	return s.Service.CreateUser(ctx, identity)
}

func (s *loggingService) UpdateUser(ctx context.Context, user contracts.User) (err error) {
	defer func() { helpers.HandleLogError(s.prefix, "UpdateUser", err) }()

	return s.Service.UpdateUser(ctx, user)
}
