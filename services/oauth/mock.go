package oauth

import (
	"context"

	"github.com/estafette/estafette-ci-api/config"
)

type MockService struct {
	GetProvidersFunc func(ctx context.Context) (providers []*config.OAuthProvider, err error)
}

func (s MockService) GetProviders(ctx context.Context) (providers []*config.OAuthProvider, err error) {
	if s.GetProvidersFunc == nil {
		return
	}
	return s.GetProvidersFunc(ctx)
}
