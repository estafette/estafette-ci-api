package rbac

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/dgrijalva/jwt-go"
	"github.com/estafette/estafette-ci-api/clients/cockroachdb"
	"github.com/estafette/estafette-ci-api/config"
	contracts "github.com/estafette/estafette-ci-contracts"
	"github.com/rs/zerolog/log"
)

var (
	// ErrUserNotFound indicates that a user cannot be found in the database
	ErrUserNotFound = errors.New("The user can't be found")

	// ErrInvalidSigningAlgorithm indicates signing algorithm is invalid, needs to be HS256, HS384, HS512, RS256, RS384 or RS512
	ErrInvalidSigningAlgorithm = errors.New("invalid signing algorithm")
)

// Service handles http requests for role-based-access-control
type Service interface {
	GetProviders(ctx context.Context) (providers []*config.OAuthProvider, err error)
	GetProviderByName(ctx context.Context, name string) (provider *config.OAuthProvider, err error)
	GetUserByIdentity(ctx context.Context, identity contracts.UserIdentity) (user *contracts.User, err error)
	GetUserByID(ctx context.Context, id string) (user *contracts.User, err error)
	CreateUser(ctx context.Context, identity contracts.UserIdentity) (user *contracts.User, err error)
	UpdateUser(ctx context.Context, user contracts.User) (err error)
	GenerateJWT(ctx context.Context, optionalClaims jwt.MapClaims) (tokenString string, err error)
	ValidateJWT(ctx context.Context, tokenString string) (token *jwt.Token, err error)
	GetClaimsFromJWT(ctx context.Context, tokenString string) (claims jwt.MapClaims, err error)
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

func (s *service) GenerateJWT(ctx context.Context, optionalClaims jwt.MapClaims) (tokenString string, err error) {

	// Create the token
	token := jwt.New(jwt.GetSigningMethod("HS256"))
	claims := token.Claims.(jwt.MapClaims)

	// set required claims
	now := time.Now().UTC()
	expire := now.Add(time.Hour)
	claims["exp"] = expire.Unix()
	claims["orig_iat"] = now.Unix()

	if optionalClaims != nil {
		for key, value := range optionalClaims {
			claims[key] = value
		}
	}

	// sign the token
	return token.SignedString(s.config.Auth.JWT.Key)
}

func (s *service) ValidateJWT(ctx context.Context, tokenString string) (token *jwt.Token, err error) {
	return jwt.Parse(tokenString, func(t *jwt.Token) (interface{}, error) {
		if jwt.GetSigningMethod("HS256") != t.Method {
			return nil, ErrInvalidSigningAlgorithm
		}
		return s.config.Auth.JWT.Key, nil
	})
}

func (s *service) GetClaimsFromJWT(ctx context.Context, tokenString string) (claims jwt.MapClaims, err error) {
	token, err := s.ValidateJWT(ctx, tokenString)
	if err != nil {
		return nil, err
	}

	claims = jwt.MapClaims{}
	for key, value := range token.Claims.(jwt.MapClaims) {
		claims[key] = value
	}

	return claims, nil
}
