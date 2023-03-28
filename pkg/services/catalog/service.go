package catalog

import (
	"context"

	"github.com/estafette/estafette-ci-api/pkg/api"
	"github.com/estafette/estafette-ci-api/pkg/clients/database"
	contracts "github.com/estafette/estafette-ci-contracts"
)

// Service handles http requests for role-based-access-control
//
//go:generate mockgen -package=catalog -destination ./mock.go -source=service.go
type Service interface {
	CreateCatalogEntity(ctx context.Context, catalogEntity contracts.CatalogEntity) (insertedCatalogEntity *contracts.CatalogEntity, err error)
	UpdateCatalogEntity(ctx context.Context, catalogEntity contracts.CatalogEntity) (err error)
	DeleteCatalogEntity(ctx context.Context, id string) (err error)
}

// NewService returns a github.Service to handle incoming webhook events
func NewService(config *api.APIConfig, databaseClient database.Client) Service {
	return &service{
		config:         config,
		databaseClient: databaseClient,
	}
}

type service struct {
	config         *api.APIConfig
	databaseClient database.Client
}

func (s *service) CreateCatalogEntity(ctx context.Context, catalogEntity contracts.CatalogEntity) (insertedCatalogEntity *contracts.CatalogEntity, err error) {
	return s.databaseClient.InsertCatalogEntity(ctx, catalogEntity)
}

func (s *service) UpdateCatalogEntity(ctx context.Context, catalogEntity contracts.CatalogEntity) (err error) {
	return s.databaseClient.UpdateCatalogEntity(ctx, catalogEntity)
}

func (s *service) DeleteCatalogEntity(ctx context.Context, id string) (err error) {
	return s.databaseClient.DeleteCatalogEntity(ctx, id)
}
