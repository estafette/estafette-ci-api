package rbac

import (
	"context"

	"github.com/estafette/estafette-ci-api/auth"
	"github.com/estafette/estafette-ci-api/config"
	contracts "github.com/estafette/estafette-ci-contracts"
)

type MockService struct {
	GetProvidersFunc func(ctx context.Context) (providers []*config.OAuthProvider, err error)
	GetUserFunc      func(ctx context.Context, authUser auth.User) (user *contracts.User, err error)
	CreateUserFunc   func(ctx context.Context, authUser auth.User) (user *contracts.User, err error)
	UpdateUserFunc   func(ctx context.Context, authUser auth.User) (err error)
}

func (s MockService) GetProviders(ctx context.Context) (providers []*config.OAuthProvider, err error) {
	if s.GetProvidersFunc == nil {
		return
	}
	return s.GetProvidersFunc(ctx)
}

func (s MockService) GetUser(ctx context.Context, authUser auth.User) (user *contracts.User, err error) {
	if s.GetUserFunc == nil {
		return
	}
	return s.GetUserFunc(ctx, authUser)
}

func (s MockService) CreateUser(ctx context.Context, authUser auth.User) (user *contracts.User, err error) {
	if s.CreateUserFunc == nil {
		return
	}
	return s.CreateUserFunc(ctx, authUser)
}

func (s MockService) UpdateUser(ctx context.Context, authUser auth.User) (err error) {
	if s.UpdateUserFunc == nil {
		return
	}
	return s.UpdateUserFunc(ctx, authUser)
}
