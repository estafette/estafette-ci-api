package catalog

import (
	"context"

	"github.com/estafette/estafette-ci-api/pkg/api"
	contracts "github.com/estafette/estafette-ci-contracts"
)

// NewLoggingService returns a new instance of a logging Service.
func NewLoggingService(s Service) Service {
	return &loggingService{s, "oauth"}
}

type loggingService struct {
	Service Service
	prefix  string
}

func (s *loggingService) CreateCatalogEntity(ctx context.Context, catalogEntity contracts.CatalogEntity) (insertedCatalogEntity *contracts.CatalogEntity, err error) {
	defer func() { api.HandleLogError(s.prefix, "Service", "CreateCatalogEntity", err) }()

	return s.Service.CreateCatalogEntity(ctx, catalogEntity)
}

func (s *loggingService) UpdateCatalogEntity(ctx context.Context, catalogEntity contracts.CatalogEntity) (err error) {
	defer func() { api.HandleLogError(s.prefix, "Service", "UpdateCatalogEntity", err) }()

	return s.Service.UpdateCatalogEntity(ctx, catalogEntity)
}

func (s *loggingService) DeleteCatalogEntity(ctx context.Context, id string) (err error) {
	defer func() { api.HandleLogError(s.prefix, "Service", "DeleteCatalogEntity", err) }()

	return s.Service.DeleteCatalogEntity(ctx, id)
}
