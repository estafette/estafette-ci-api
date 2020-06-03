package rbac

import (
	"context"

	"github.com/estafette/estafette-ci-api/auth"
	"github.com/estafette/estafette-ci-api/config"
	contracts "github.com/estafette/estafette-ci-contracts"
)

type MockService struct {
	GetProvidersFunc      func(ctx context.Context) (providers []*config.OAuthProvider, err error)
	GetUserByIdentityFunc func(ctx context.Context, identity contracts.UserIdentity) (user *contracts.User, err error)
	GetUserByIDFunc       func(ctx context.Context, id string) (user *contracts.User, err error)
	CreateUserFunc        func(ctx context.Context, authUser auth.User) (user *contracts.User, err error)
	UpdateUserFunc        func(ctx context.Context, authUser auth.User) (err error)
}

func (s MockService) GetProviders(ctx context.Context) (providers []*config.OAuthProvider, err error) {
	if s.GetProvidersFunc == nil {
		return
	}
	return s.GetProvidersFunc(ctx)
}

func (s MockService) GetUserByIdentity(ctx context.Context, identity contracts.UserIdentity) (user *contracts.User, err error) {
	if s.GetUserByIdentityFunc == nil {
		return
	}
	return s.GetUserByIdentityFunc(ctx, identity)
}

func (s MockService) GetUserByID(ctx context.Context, id string) (user *contracts.User, err error) {
	if s.GetUserByIDFunc == nil {
		return
	}
	return s.GetUserByIDFunc(ctx, id)
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
