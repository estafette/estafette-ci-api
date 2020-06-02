package oauth

import (
	"context"
	"time"

	"github.com/estafette/estafette-ci-api/config"
	"github.com/estafette/estafette-ci-api/helpers"
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
