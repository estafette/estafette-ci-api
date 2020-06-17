package catalog

import (
	"context"

	"github.com/estafette/estafette-ci-api/helpers"
	contracts "github.com/estafette/estafette-ci-contracts"
)

// NewLoggingService returns a new instance of a logging Service.
func NewLoggingService(s Service) Service {
	return &loggingService{s, "oauth"}
}

type loggingService struct {
	Service
	prefix string
}

func (s *loggingService) CreateCatalogEntity(ctx context.Context, catalogEntity contracts.CatalogEntity) (insertedCatalogEntity *contracts.CatalogEntity, err error) {
	defer func() { helpers.HandleLogError(s.prefix, "CreateCatalogEntity", err) }()

	return s.Service.CreateCatalogEntity(ctx, catalogEntity)
}

func (s *loggingService) UpdateCatalogEntity(ctx context.Context, catalogEntity contracts.CatalogEntity) (err error) {
	defer func() { helpers.HandleLogError(s.prefix, "UpdateCatalogEntity", err) }()

	return s.Service.UpdateCatalogEntity(ctx, catalogEntity)
}

func (s *loggingService) DeleteCatalogEntity(ctx context.Context, id string) (err error) {
	defer func() { helpers.HandleLogError(s.prefix, "DeleteCatalogEntity", err) }()

	return s.Service.DeleteCatalogEntity(ctx, id)
}
