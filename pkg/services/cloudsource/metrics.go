package cloudsource

import (
	"context"
	"time"

	"github.com/estafette/estafette-ci-api/pkg/api"
	"github.com/estafette/estafette-ci-api/pkg/clients/cloudsourceapi"
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

func (s *metricsService) CreateJobForCloudSourcePush(ctx context.Context, notification cloudsourceapi.PubSubNotification) (err error) {
	defer func(begin time.Time) {
		api.UpdateMetrics(s.requestCount, s.requestLatency, "CreateJobForCloudSourcePush", begin)
	}(time.Now())

	return s.Service.CreateJobForCloudSourcePush(ctx, notification)
}

func (s *metricsService) IsAllowedProject(ctx context.Context, notification cloudsourceapi.PubSubNotification) (isAllowed bool, organizations []*contracts.Organization) {
	defer func(begin time.Time) {
		api.UpdateMetrics(s.requestCount, s.requestLatency, "IsAllowedProject", begin)
	}(time.Now())

	return s.Service.IsAllowedProject(ctx, notification)
}
