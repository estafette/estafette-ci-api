package bigquery

import (
	"context"

	"github.com/estafette/estafette-ci-api/pkg/api"
)

func NewLoggingClient(c Client) Client {
	return &loggingClient{c, "bigquery"}
}

type loggingClient struct {
	Client Client
	prefix string
}

func (c *loggingClient) Init(ctx context.Context) (err error) {
	defer func() { api.HandleLogError(c.prefix, "Client", "Init", err) }()

	return c.Client.Init(ctx)
}

func (c *loggingClient) CheckIfDatasetExists(ctx context.Context) (exists bool) {
	return c.Client.CheckIfDatasetExists(ctx)
}

func (c *loggingClient) CheckIfTableExists(ctx context.Context, table string) (exists bool) {
	return c.Client.CheckIfTableExists(ctx, table)
}

func (c *loggingClient) CreateTable(ctx context.Context, table string, typeForSchema interface{}, partitionField string, waitReady bool) (err error) {
	defer func() { api.HandleLogError(c.prefix, "Client", "CreateTable", err) }()

	return c.Client.CreateTable(ctx, table, typeForSchema, partitionField, waitReady)
}

func (c *loggingClient) UpdateTableSchema(ctx context.Context, table string, typeForSchema interface{}) (err error) {
	defer func() { api.HandleLogError(c.prefix, "Client", "UpdateTableSchema", err) }()

	return c.Client.UpdateTableSchema(ctx, table, typeForSchema)
}

func (c *loggingClient) InsertBuildEvent(ctx context.Context, event PipelineBuildEvent) (err error) {
	defer func() { api.HandleLogError(c.prefix, "Client", "InsertBuildEvent", err) }()

	return c.Client.InsertBuildEvent(ctx, event)
}

func (c *loggingClient) InsertReleaseEvent(ctx context.Context, event PipelineReleaseEvent) (err error) {
	defer func() { api.HandleLogError(c.prefix, "Client", "InsertReleaseEvent", err) }()

	return c.Client.InsertReleaseEvent(ctx, event)
}
