package bigquery

import (
	"context"

	"github.com/estafette/estafette-ci-api/pkg/api"
	"github.com/opentracing/opentracing-go"
)

// NewTracingClient returns a new instance of a tracing Client.
func NewTracingClient(c Client) Client {
	return &tracingClient{c, "bigquery"}
}

type tracingClient struct {
	Client Client
	prefix string
}

func (c *tracingClient) Init(ctx context.Context) (err error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, api.GetSpanName(c.prefix, "Init"))
	defer func() { api.FinishSpanWithError(span, err) }()

	return c.Client.Init(ctx)
}

func (c *tracingClient) CheckIfDatasetExists(ctx context.Context) (exists bool) {
	span, ctx := opentracing.StartSpanFromContext(ctx, api.GetSpanName(c.prefix, "CheckIfDatasetExists"))
	defer func() { api.FinishSpan(span) }()

	return c.Client.CheckIfDatasetExists(ctx)
}

func (c *tracingClient) CheckIfTableExists(ctx context.Context, table string) (exists bool) {
	span, ctx := opentracing.StartSpanFromContext(ctx, api.GetSpanName(c.prefix, "CheckIfTableExists"))
	defer func() { api.FinishSpan(span) }()

	return c.Client.CheckIfTableExists(ctx, table)
}

func (c *tracingClient) CreateTable(ctx context.Context, table string, typeForSchema interface{}, partitionField string, waitReady bool) (err error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, api.GetSpanName(c.prefix, "CreateTable"))
	defer func() { api.FinishSpanWithError(span, err) }()

	return c.Client.CreateTable(ctx, table, typeForSchema, partitionField, waitReady)
}

func (c *tracingClient) UpdateTableSchema(ctx context.Context, table string, typeForSchema interface{}) (err error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, api.GetSpanName(c.prefix, "UpdateTableSchema"))
	defer func() { api.FinishSpanWithError(span, err) }()

	return c.Client.UpdateTableSchema(ctx, table, typeForSchema)
}

func (c *tracingClient) InsertBuildEvent(ctx context.Context, event PipelineBuildEvent) (err error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, api.GetSpanName(c.prefix, "InsertBuildEvent"))
	defer func() { api.FinishSpanWithError(span, err) }()

	return c.Client.InsertBuildEvent(ctx, event)
}

func (c *tracingClient) InsertReleaseEvent(ctx context.Context, event PipelineReleaseEvent) (err error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, api.GetSpanName(c.prefix, "InsertReleaseEvent"))
	defer func() { api.FinishSpanWithError(span, err) }()

	return c.Client.InsertReleaseEvent(ctx, event)
}
