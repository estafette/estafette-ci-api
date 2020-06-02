package oauth

import (
	"context"

	"github.com/estafette/estafette-ci-api/config"
	"github.com/estafette/estafette-ci-api/helpers"
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
