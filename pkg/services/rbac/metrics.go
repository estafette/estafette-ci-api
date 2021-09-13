package rbac

import (
	"context"
	"time"

	"github.com/estafette/estafette-ci-api/pkg/api"
	contracts "github.com/estafette/estafette-ci-contracts"
	"github.com/go-kit/kit/metrics"
)

// NewMetricsService returns a new instance of a metrics Service.
func NewMetricsService(s Service, requestCount metrics.Counter, requestLatency metrics.Histogram) Service {
	return &metricsService{s, requestCount, requestLatency}
}

type metricsService struct {
	Service        Service
	requestCount   metrics.Counter
	requestLatency metrics.Histogram
}

func (s *metricsService) GetRoles(ctx context.Context) (roles []string, err error) {
	defer func(begin time.Time) { api.UpdateMetrics(s.requestCount, s.requestLatency, "GetRoles", begin) }(time.Now())

	return s.Service.GetRoles(ctx)
}

func (s *metricsService) GetProviders(ctx context.Context) (providers []*api.OAuthProvider, err error) {
	defer func(begin time.Time) { api.UpdateMetrics(s.requestCount, s.requestLatency, "GetProviders", begin) }(time.Now())

	return s.Service.GetProviders(ctx)
}

func (s *metricsService) GetProviderByName(ctx context.Context, organization, name string) (provider *api.OAuthProvider, err error) {
	defer func(begin time.Time) {
		api.UpdateMetrics(s.requestCount, s.requestLatency, "GetProviderByName", begin)
	}(time.Now())

	return s.Service.GetProviderByName(ctx, organization, name)
}

func (s *metricsService) GetUserByIdentity(ctx context.Context, identity contracts.UserIdentity) (user *contracts.User, err error) {
	defer func(begin time.Time) {
		api.UpdateMetrics(s.requestCount, s.requestLatency, "GetUserByIdentity", begin)
	}(time.Now())

	return s.Service.GetUserByIdentity(ctx, identity)
}

func (s *metricsService) CreateUserFromIdentity(ctx context.Context, identity contracts.UserIdentity) (user *contracts.User, err error) {
	defer func(begin time.Time) {
		api.UpdateMetrics(s.requestCount, s.requestLatency, "CreateUserFromIdentity", begin)
	}(time.Now())

	return s.Service.CreateUserFromIdentity(ctx, identity)
}

func (s *metricsService) CreateUser(ctx context.Context, user contracts.User) (insertedUser *contracts.User, err error) {
	defer func(begin time.Time) {
		api.UpdateMetrics(s.requestCount, s.requestLatency, "CreateUser", begin)
	}(time.Now())

	return s.Service.CreateUser(ctx, user)
}

func (s *metricsService) UpdateUser(ctx context.Context, user contracts.User) (err error) {
	defer func(begin time.Time) {
		api.UpdateMetrics(s.requestCount, s.requestLatency, "UpdateUser", begin)
	}(time.Now())

	return s.Service.UpdateUser(ctx, user)
}

func (s *metricsService) DeleteUser(ctx context.Context, id string) (err error) {
	defer func(begin time.Time) {
		api.UpdateMetrics(s.requestCount, s.requestLatency, "DeleteUser", begin)
	}(time.Now())

	return s.Service.DeleteUser(ctx, id)
}

func (s *metricsService) CreateGroup(ctx context.Context, group contracts.Group) (insertedGroup *contracts.Group, err error) {
	defer func(begin time.Time) {
		api.UpdateMetrics(s.requestCount, s.requestLatency, "CreateGroup", begin)
	}(time.Now())

	return s.Service.CreateGroup(ctx, group)
}

func (s *metricsService) UpdateGroup(ctx context.Context, group contracts.Group) (err error) {
	defer func(begin time.Time) {
		api.UpdateMetrics(s.requestCount, s.requestLatency, "UpdateGroup", begin)
	}(time.Now())

	return s.Service.UpdateGroup(ctx, group)
}

func (s *metricsService) DeleteGroup(ctx context.Context, id string) (err error) {
	defer func(begin time.Time) {
		api.UpdateMetrics(s.requestCount, s.requestLatency, "DeleteGroup", begin)
	}(time.Now())

	return s.Service.DeleteGroup(ctx, id)
}

func (s *metricsService) CreateOrganization(ctx context.Context, organization contracts.Organization) (insertedOrganization *contracts.Organization, err error) {
	defer func(begin time.Time) {
		api.UpdateMetrics(s.requestCount, s.requestLatency, "CreateOrganization", begin)
	}(time.Now())

	return s.Service.CreateOrganization(ctx, organization)
}

func (s *metricsService) UpdateOrganization(ctx context.Context, organization contracts.Organization) (err error) {
	defer func(begin time.Time) {
		api.UpdateMetrics(s.requestCount, s.requestLatency, "UpdateOrganization", begin)
	}(time.Now())

	return s.Service.UpdateOrganization(ctx, organization)
}

func (s *metricsService) DeleteOrganization(ctx context.Context, id string) (err error) {
	defer func(begin time.Time) {
		api.UpdateMetrics(s.requestCount, s.requestLatency, "DeleteOrganization", begin)
	}(time.Now())

	return s.Service.DeleteOrganization(ctx, id)
}

func (s *metricsService) CreateClient(ctx context.Context, client contracts.Client) (insertedClient *contracts.Client, err error) {
	defer func(begin time.Time) {
		api.UpdateMetrics(s.requestCount, s.requestLatency, "CreateClient", begin)
	}(time.Now())

	return s.Service.CreateClient(ctx, client)
}

func (s *metricsService) UpdateClient(ctx context.Context, client contracts.Client) (err error) {
	defer func(begin time.Time) {
		api.UpdateMetrics(s.requestCount, s.requestLatency, "UpdateClient", begin)
	}(time.Now())

	return s.Service.UpdateClient(ctx, client)
}

func (s *metricsService) DeleteClient(ctx context.Context, id string) (err error) {
	defer func(begin time.Time) {
		api.UpdateMetrics(s.requestCount, s.requestLatency, "DeleteClient", begin)
	}(time.Now())

	return s.Service.DeleteClient(ctx, id)
}

func (s *metricsService) UpdatePipeline(ctx context.Context, pipeline contracts.Pipeline) (err error) {
	defer func(begin time.Time) {
		api.UpdateMetrics(s.requestCount, s.requestLatency, "UpdatePipeline", begin)
	}(time.Now())

	return s.Service.UpdatePipeline(ctx, pipeline)
}

func (s *metricsService) GetInheritedRolesForUser(ctx context.Context, user contracts.User) (roles []*string, err error) {
	defer func(begin time.Time) {
		api.UpdateMetrics(s.requestCount, s.requestLatency, "GetInheritedRolesForUser", begin)
	}(time.Now())

	return s.Service.GetInheritedRolesForUser(ctx, user)
}

func (s *metricsService) GetInheritedOrganizationsForUser(ctx context.Context, user contracts.User) (organizations []*contracts.Organization, err error) {
	defer func(begin time.Time) {
		api.UpdateMetrics(s.requestCount, s.requestLatency, "GetInheritedOrganizationsForUser", begin)
	}(time.Now())

	return s.Service.GetInheritedOrganizationsForUser(ctx, user)
}
