package catalog

import (
	"context"

	"github.com/estafette/estafette-ci-api/clients/cockroachdb"
	"github.com/estafette/estafette-ci-api/config"
	contracts "github.com/estafette/estafette-ci-contracts"
)

// Service handles http requests for role-based-access-control
type Service interface {
	CreateCatalogEntity(ctx context.Context, catalogEntity contracts.CatalogEntity) (insertedCatalogEntity *contracts.CatalogEntity, err error)
	UpdateCatalogEntity(ctx context.Context, catalogEntity contracts.CatalogEntity) (err error)
	DeleteCatalogEntity(ctx context.Context, id string) (err error)
}

// NewService returns a github.Service to handle incoming webhook events
func NewService(config *config.APIConfig, cockroachdbClient cockroachdb.Client) Service {
	return &service{
		config:            config,
		cockroachdbClient: cockroachdbClient,
	}
}

type service struct {
	config            *config.APIConfig
	cockroachdbClient cockroachdb.Client
}

func (s *service) CreateCatalogEntity(ctx context.Context, catalogEntity contracts.CatalogEntity) (insertedCatalogEntity *contracts.CatalogEntity, err error) {
	return s.cockroachdbClient.InsertCatalogEntity(ctx, catalogEntity)
}

func (s *service) UpdateCatalogEntity(ctx context.Context, catalogEntity contracts.CatalogEntity) (err error) {
	return s.cockroachdbClient.UpdateCatalogEntity(ctx, catalogEntity)
}

func (s *service) DeleteCatalogEntity(ctx context.Context, id string) (err error) {
	return s.cockroachdbClient.DeleteCatalogEntity(ctx, id)
}
