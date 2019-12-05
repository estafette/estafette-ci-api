package gcs

import (
	"context"

	"cloud.google.com/go/storage"
	"github.com/estafette/estafette-ci-api/config"
)

// CloudStorageClient is the interface for connecting to google cloud storage
type CloudStorageClient interface {
}

type cloudStorageClientImpl struct {
	client *storage.Client
	config *config.CloudStorageConfig
}

// NewCloudStorageClient returns new CloudStorageClient
func NewCloudStorageClient(config *config.CloudStorageConfig) (CloudStorageClient, error) {

	if config == nil {
		return &cloudStorageClientImpl{
			config: config,
		}, nil
	}

	ctx := context.Background()

	storageClient, err := storage.NewClient(ctx)
	if err != nil {
		return nil, err
	}

	return &cloudStorageClientImpl{
		client: storageClient,
		config: config,
	}, nil
}
