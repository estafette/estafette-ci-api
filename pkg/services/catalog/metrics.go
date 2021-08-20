package catalog

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

func (s *metricsService) CreateCatalogEntity(ctx context.Context, catalogEntity contracts.CatalogEntity) (insertedCatalogEntity *contracts.CatalogEntity, err error) {
	defer func(begin time.Time) {
		api.UpdateMetrics(s.requestCount, s.requestLatency, "CreateCatalogEntity", begin)
	}(time.Now())

	return s.Service.CreateCatalogEntity(ctx, catalogEntity)
}

func (s *metricsService) UpdateCatalogEntity(ctx context.Context, catalogEntity contracts.CatalogEntity) (err error) {
	defer func(begin time.Time) {
		api.UpdateMetrics(s.requestCount, s.requestLatency, "UpdateCatalogEntity", begin)
	}(time.Now())

	return s.Service.UpdateCatalogEntity(ctx, catalogEntity)
}

func (s *metricsService) DeleteCatalogEntity(ctx context.Context, id string) (err error) {
	defer func(begin time.Time) {
		api.UpdateMetrics(s.requestCount, s.requestLatency, "DeleteCatalogEntity", begin)
	}(time.Now())

	return s.Service.DeleteCatalogEntity(ctx, id)
}
