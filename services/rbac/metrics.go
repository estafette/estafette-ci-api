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

func (s *metricsService) GetRoles(ctx context.Context) (roles []string, err error) {
	defer func(begin time.Time) { helpers.UpdateMetrics(s.requestCount, s.requestLatency, "GetRoles", begin) }(time.Now())

	return s.Service.GetRoles(ctx)
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

func (s *metricsService) CreateUserFromIdentity(ctx context.Context, identity contracts.UserIdentity) (user *contracts.User, err error) {
	defer func(begin time.Time) {
		helpers.UpdateMetrics(s.requestCount, s.requestLatency, "CreateUserFromIdentity", begin)
	}(time.Now())

	return s.Service.CreateUserFromIdentity(ctx, identity)
}

func (s *metricsService) CreateUser(ctx context.Context, user contracts.User) (insertedUser *contracts.User, err error) {
	defer func(begin time.Time) {
		helpers.UpdateMetrics(s.requestCount, s.requestLatency, "CreateUser", begin)
	}(time.Now())

	return s.Service.CreateUser(ctx, user)
}

func (s *metricsService) UpdateUser(ctx context.Context, user contracts.User) (err error) {
	defer func(begin time.Time) {
		helpers.UpdateMetrics(s.requestCount, s.requestLatency, "UpdateUser", begin)
	}(time.Now())

	return s.Service.UpdateUser(ctx, user)
}

func (s *metricsService) CreateGroup(ctx context.Context, group contracts.Group) (insertedGroup *contracts.Group, err error) {
	defer func(begin time.Time) {
		helpers.UpdateMetrics(s.requestCount, s.requestLatency, "CreateGroup", begin)
	}(time.Now())

	return s.Service.CreateGroup(ctx, group)
}

func (s *metricsService) UpdateGroup(ctx context.Context, group contracts.Group) (err error) {
	defer func(begin time.Time) {
		helpers.UpdateMetrics(s.requestCount, s.requestLatency, "UpdateGroup", begin)
	}(time.Now())

	return s.Service.UpdateGroup(ctx, group)
}

func (s *metricsService) CreateOrganization(ctx context.Context, organization contracts.Organization) (insertedOrganization *contracts.Organization, err error) {
	defer func(begin time.Time) {
		helpers.UpdateMetrics(s.requestCount, s.requestLatency, "CreateOrganization", begin)
	}(time.Now())

	return s.Service.CreateOrganization(ctx, organization)
}

func (s *metricsService) UpdateOrganization(ctx context.Context, organization contracts.Organization) (err error) {
	defer func(begin time.Time) {
		helpers.UpdateMetrics(s.requestCount, s.requestLatency, "UpdateOrganization", begin)
	}(time.Now())

	return s.Service.UpdateOrganization(ctx, organization)
}

func (s *metricsService) CreateClient(ctx context.Context, client contracts.Client) (insertedClient *contracts.Client, err error) {
	defer func(begin time.Time) {
		helpers.UpdateMetrics(s.requestCount, s.requestLatency, "CreateClient", begin)
	}(time.Now())

	return s.Service.CreateClient(ctx, client)
}

func (s *metricsService) UpdateClient(ctx context.Context, client contracts.Client) (err error) {
	defer func(begin time.Time) {
		helpers.UpdateMetrics(s.requestCount, s.requestLatency, "UpdateClient", begin)
	}(time.Now())

	return s.Service.UpdateClient(ctx, client)
}

func (s *metricsService) GetInheritedRolesForUser(ctx context.Context, user contracts.User) (roles []*string, err error) {
	defer func(begin time.Time) {
		helpers.UpdateMetrics(s.requestCount, s.requestLatency, "GetInheritedRolesForUser", begin)
	}(time.Now())

	return s.Service.GetInheritedRolesForUser(ctx, user)
}
