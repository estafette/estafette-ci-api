package bigquery

import (
	"context"

	"github.com/rs/zerolog/log"
)

func NewLoggingClient(c Client) Client {
	return &loggingClient{c}
}

type loggingClient struct {
	Client
}

func (c *loggingClient) Init(ctx context.Context) (err error) {
	defer func() {
		c.handleError(err)
	}()

	return c.Client.Init(ctx)
}

func (c *loggingClient) CheckIfDatasetExists(ctx context.Context) bool {
	return c.Client.CheckIfDatasetExists(ctx)
}

func (c *loggingClient) CheckIfTableExists(ctx context.Context, table string) bool {
	return c.Client.CheckIfTableExists(ctx, table)
}

func (c *loggingClient) CreateTable(ctx context.Context, table string, typeForSchema interface{}, partitionField string, waitReady bool) (err error) {
	defer func() {
		c.handleError(err)
	}()

	return c.Client.CreateTable(ctx, table, typeForSchema, partitionField, waitReady)
}

func (c *loggingClient) UpdateTableSchema(ctx context.Context, table string, typeForSchema interface{}) (err error) {
	defer func() {
		c.handleError(err)
	}()

	return c.Client.UpdateTableSchema(ctx, table, typeForSchema)
}

func (c *loggingClient) InsertBuildEvent(ctx context.Context, event PipelineBuildEvent) (err error) {
	defer func() {
		c.handleError(err)
	}()

	return c.Client.InsertBuildEvent(ctx, event)
}

func (c *loggingClient) InsertReleaseEvent(ctx context.Context, event PipelineReleaseEvent) (err error) {
	defer func() {
		c.handleError(err)
	}()

	return c.Client.InsertReleaseEvent(ctx, event)
}

func (c *loggingClient) handleError(err error) {
	if err != nil {
		log.Error().Err(err).Msg("Failure")
	}
}
