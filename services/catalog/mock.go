package catalog

import (
	"context"

	contracts "github.com/estafette/estafette-ci-contracts"
)

type MockService struct {
	CreateCatalogEntityFunc func(ctx context.Context, catalogEntity contracts.CatalogEntity) (insertedCatalogEntity *contracts.CatalogEntity, err error)
	UpdateCatalogEntityFunc func(ctx context.Context, catalogEntity contracts.CatalogEntity) (err error)
	DeleteCatalogEntityFunc func(ctx context.Context, id string) (err error)
}

func (s MockService) CreateCatalogEntity(ctx context.Context, catalogEntity contracts.CatalogEntity) (insertedCatalogEntity *contracts.CatalogEntity, err error) {
	if s.CreateCatalogEntityFunc == nil {
		return
	}
	return s.CreateCatalogEntityFunc(ctx, catalogEntity)
}

func (s MockService) UpdateCatalogEntity(ctx context.Context, catalogEntity contracts.CatalogEntity) (err error) {
	if s.UpdateCatalogEntityFunc == nil {
		return
	}
	return s.UpdateCatalogEntityFunc(ctx, catalogEntity)
}

func (s MockService) DeleteCatalogEntity(ctx context.Context, id string) (err error) {
	if s.DeleteCatalogEntityFunc == nil {
		return
	}
	return s.DeleteCatalogEntityFunc(ctx, id)
}
