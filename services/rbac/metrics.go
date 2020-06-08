package rbac

import (
	"context"
	"time"

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

func (s *metricsService) GetProviderByName(ctx context.Context, name string) (provider *config.OAuthProvider, err error) {
	defer func(begin time.Time) {
		helpers.UpdateMetrics(s.requestCount, s.requestLatency, "GetProviderByName", begin)
	}(time.Now())

	return s.Service.GetProviderByName(ctx, name)
}

func (s *metricsService) GetUserByIdentity(ctx context.Context, identity contracts.UserIdentity) (user *contracts.User, err error) {
	defer func(begin time.Time) {
		helpers.UpdateMetrics(s.requestCount, s.requestLatency, "GetUserByIdentity", begin)
	}(time.Now())

	return s.Service.GetUserByIdentity(ctx, identity)
}

func (s *metricsService) GetUserByID(ctx context.Context, id string) (user *contracts.User, err error) {
	defer func(begin time.Time) {
		helpers.UpdateMetrics(s.requestCount, s.requestLatency, "GetUserByID", begin)
	}(time.Now())

	return s.Service.GetUserByID(ctx, id)
}

func (s *metricsService) CreateUser(ctx context.Context, identity contracts.UserIdentity) (user *contracts.User, err error) {
	defer func(begin time.Time) {
		helpers.UpdateMetrics(s.requestCount, s.requestLatency, "CreateUser", begin)
	}(time.Now())

	return s.Service.CreateUser(ctx, identity)
}

func (s *metricsService) UpdateUser(ctx context.Context, user contracts.User) (err error) {
	defer func(begin time.Time) {
		helpers.UpdateMetrics(s.requestCount, s.requestLatency, "UpdateUser", begin)
	}(time.Now())

	return s.Service.UpdateUser(ctx, user)
}

func (s *metricsService) GetUsers(ctx context.Context) (users []*contracts.User, err error) {
	defer func(begin time.Time) {
		helpers.UpdateMetrics(s.requestCount, s.requestLatency, "GetUsers", begin)
	}(time.Now())

	return s.Service.GetUsers(ctx)
}
