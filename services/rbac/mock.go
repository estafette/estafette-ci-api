package rbac

import (
	"context"

	"github.com/dgrijalva/jwt-go"
	"github.com/estafette/estafette-ci-api/auth"
	"github.com/estafette/estafette-ci-api/config"
	contracts "github.com/estafette/estafette-ci-contracts"
)

type MockService struct {
	GetProvidersFunc      func(ctx context.Context) (providers []*config.OAuthProvider, err error)
	GetProviderByNameFunc func(ctx context.Context, name string) (provider *config.OAuthProvider, err error)
	GetUserByIdentityFunc func(ctx context.Context, identity contracts.UserIdentity) (user *contracts.User, err error)
	GetUserByIDFunc       func(ctx context.Context, id string) (user *contracts.User, err error)
	CreateUserFunc        func(ctx context.Context, authUser auth.User) (user *contracts.User, err error)
	UpdateUserFunc        func(ctx context.Context, authUser auth.User) (err error)
	GenerateJWTFunc       func(ctx context.Context, optionalClaims jwt.MapClaims) (tokenString string, err error)
	ValidateJWTFunc       func(ctx context.Context, tokenString string) (token *jwt.Token, err error)
	GetClaimsFromJWTFunc  func(ctx context.Context, tokenString string) (claims jwt.MapClaims, err error)
}

func (s MockService) GetProviders(ctx context.Context) (providers []*config.OAuthProvider, err error) {
	if s.GetProvidersFunc == nil {
		return
	}
	return s.GetProvidersFunc(ctx)
}

func (s MockService) GetProviderByName(ctx context.Context, name string) (provider *config.OAuthProvider, err error) {
	if s.GetProviderByNameFunc == nil {
		return
	}
	return s.GetProviderByNameFunc(ctx, name)
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

func (s MockService) GenerateJWT(ctx context.Context, optionalClaims jwt.MapClaims) (tokenString string, err error) {
	if s.GenerateJWTFunc == nil {
		return
	}
	return s.GenerateJWTFunc(ctx, optionalClaims)
}

func (s MockService) ValidateJWT(ctx context.Context, tokenString string) (token *jwt.Token, err error) {
	if s.ValidateJWTFunc == nil {
		return
	}
	return s.ValidateJWTFunc(ctx, tokenString)
}

func (s MockService) GetClaimsFromJWT(ctx context.Context, tokenString string) (claims jwt.MapClaims, err error) {
	if s.GetClaimsFromJWTFunc == nil {
		return
	}
	return s.GetClaimsFromJWTFunc(ctx, tokenString)
}
