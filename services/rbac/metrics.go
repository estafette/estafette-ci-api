package rbac

import (
	"context"
	"time"

	"github.com/estafette/estafette-ci-api/auth"
	"github.com/estafette/estafette-ci-api/config"
	"github.com/estafette/estafette-ci-api/helpers"
	contracts "github.com/estafette/estafette-ci-contracts"
	"github.com/go-kit/kit/metrics"
)

// NewMetricsService returns a new instance of a metrics Service.
func NewMetricsService(s Service, requestCount metrics.Counter, requestLatency metrics.Histogram) Service {
	return &metricsService{s, requestCount, requestLatency}
}

type metricsService struct {
	Service
	requestCount   metrics.Counter
	requestLatency metrics.Histogram
}

func (s *metricsService) GetProviders(ctx context.Context) (providers []*config.OAuthProvider, err error) {
	defer func(begin time.Time) { helpers.UpdateMetrics(s.requestCount, s.requestLatency, "GetProviders", begin) }(time.Now())

	return s.Service.GetProviders(ctx)
}

func (s *metricsService) GetUser(ctx context.Context, authUser auth.User) (user *contracts.User, err error) {
	defer func(begin time.Time) {
		helpers.UpdateMetrics(s.requestCount, s.requestLatency, "GetUser", begin)
	}(time.Now())

	return s.Service.GetUser(ctx, authUser)
}

func (s *metricsService) CreateUser(ctx context.Context, authUser auth.User) (user *contracts.User, err error) {
	defer func(begin time.Time) {
		helpers.UpdateMetrics(s.requestCount, s.requestLatency, "CreateUser", begin)
	}(time.Now())

	return s.Service.CreateUser(ctx, authUser)
}

func (s *metricsService) UpdateUser(ctx context.Context, authUser auth.User) (err error) {
	defer func(begin time.Time) {
		helpers.UpdateMetrics(s.requestCount, s.requestLatency, "UpdateUser", begin)
	}(time.Now())

	return s.Service.UpdateUser(ctx, authUser)
}
