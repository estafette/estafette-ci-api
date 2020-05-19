package bigquery

import (
	"context"

	"github.com/estafette/estafette-ci-api/config"
	"github.com/estafette/estafette-ci-api/helpers"
)

func NewLoggingClient(c Client) Client {
	return &loggingClient{c, "bigquery"}
}

type loggingClient struct {
	Client
	prefix string
}

func (c *loggingClient) Init(ctx context.Context) (err error) {
	defer func() { helpers.HandleLogError(c.prefix, "Init", err) }()

	return c.Client.Init(ctx)
}

func (c *loggingClient) CheckIfDatasetExists(ctx context.Context) (exists bool) {
	return c.Client.CheckIfDatasetExists(ctx)
}

func (c *loggingClient) CheckIfTableExists(ctx context.Context, table string) (exists bool) {
	return c.Client.CheckIfTableExists(ctx, table)
}

func (c *loggingClient) CreateTable(ctx context.Context, table string, typeForSchema interface{}, partitionField string, waitReady bool) (err error) {
	defer func() { helpers.HandleLogError(c.prefix, "CreateTable", err) }()

	return c.Client.CreateTable(ctx, table, typeForSchema, partitionField, waitReady)
}

func (c *loggingClient) UpdateTableSchema(ctx context.Context, table string, typeForSchema interface{}) (err error) {
	defer func() { helpers.HandleLogError(c.prefix, "UpdateTableSchema", err) }()

	return c.Client.UpdateTableSchema(ctx, table, typeForSchema)
}

func (c *loggingClient) InsertBuildEvent(ctx context.Context, event PipelineBuildEvent) (err error) {
	defer func() { helpers.HandleLogError(c.prefix, "InsertBuildEvent", err) }()

	return c.Client.InsertBuildEvent(ctx, event)
}

func (c *loggingClient) InsertReleaseEvent(ctx context.Context, event PipelineReleaseEvent) (err error) {
	defer func() { helpers.HandleLogError(c.prefix, "InsertReleaseEvent", err) }()

	return c.Client.InsertReleaseEvent(ctx, event)
}

func (c *loggingClient) RefreshConfig(config *config.APIConfig) {
	c.Client.RefreshConfig(config)
}
