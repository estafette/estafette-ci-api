package catalog

import (
	"context"

	"github.com/estafette/estafette-ci-api/api"
	"github.com/estafette/estafette-ci-api/clients/cockroachdb"
	contracts "github.com/estafette/estafette-ci-contracts"
)

// Service handles http requests for role-based-access-control
//go:generate mockgen -package=catalog -destination ./mock.go -source=service.go
type Service interface {
	CreateCatalogEntity(ctx context.Context, catalogEntity contracts.CatalogEntity) (insertedCatalogEntity *contracts.CatalogEntity, err error)
	UpdateCatalogEntity(ctx context.Context, catalogEntity contracts.CatalogEntity) (err error)
	DeleteCatalogEntity(ctx context.Context, id string) (err error)
}

// NewService returns a github.Service to handle incoming webhook events
func NewService(config *api.APIConfig, cockroachdbClient cockroachdb.Client) Service {
	return &service{
		config:            config,
		cockroachdbClient: cockroachdbClient,
	}
}

type service struct {
	config            *api.APIConfig
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
