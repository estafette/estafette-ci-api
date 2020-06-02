package oauth

import (
	"context"

	"github.com/estafette/estafette-ci-api/config"
)

// Service handles http events for Github integration
type Service interface {
	GetProviders(ctx context.Context) (providers []*config.OAuthProvider, err error)
}

// NewService returns a github.Service to handle incoming webhook events
func NewService(config *config.APIConfig) Service {
	return &service{
		config: config,
	}
}

type service struct {
	config *config.APIConfig
}

func (s *service) GetProviders(ctx context.Context) (providers []*config.OAuthProvider, err error) {
	return s.config.Auth.OAuthProviders, nil
}
