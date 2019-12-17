package bigquery

import (
	"context"

	"github.com/estafette/estafette-ci-api/helpers"
	"github.com/opentracing/opentracing-go"
)

// NewTracingClient returns a new instance of a tracing Client.
func NewTracingClient(c Client) Client {
	return &tracingClient{c}
}

type tracingClient struct {
	Client
}

func (c *tracingClient) Init(ctx context.Context) (err error) {

	span, ctx := opentracing.StartSpanFromContext(ctx, c.getSpanName("Init"))
	defer func() { helpers.FinishSpanWithError(span, err) }()

	return c.Client.Init(ctx)
}

func (c *tracingClient) CheckIfDatasetExists(ctx context.Context) bool {

	span, ctx := opentracing.StartSpanFromContext(ctx, c.getSpanName("CheckIfDatasetExists"))
	defer func() { helpers.FinishSpan(span) }()

	return c.Client.CheckIfDatasetExists(ctx)
}

func (c *tracingClient) CheckIfTableExists(ctx context.Context, table string) bool {

	span, ctx := opentracing.StartSpanFromContext(ctx, c.getSpanName("CheckIfTableExists"))
	defer func() { helpers.FinishSpan(span) }()

	return c.Client.CheckIfTableExists(ctx, table)
}

func (c *tracingClient) CreateTable(ctx context.Context, table string, typeForSchema interface{}, partitionField string, waitReady bool) (err error) {

	span, ctx := opentracing.StartSpanFromContext(ctx, c.getSpanName("CreateTable"))
	defer func() { helpers.FinishSpanWithError(span, err) }()

	return c.Client.CreateTable(ctx, table, typeForSchema, partitionField, waitReady)
}

func (c *tracingClient) UpdateTableSchema(ctx context.Context, table string, typeForSchema interface{}) (err error) {

	span, ctx := opentracing.StartSpanFromContext(ctx, c.getSpanName("UpdateTableSchema"))
	defer func() { helpers.FinishSpanWithError(span, err) }()

	return c.Client.UpdateTableSchema(ctx, table, typeForSchema)
}

func (c *tracingClient) InsertBuildEvent(ctx context.Context, event PipelineBuildEvent) (err error) {

	span, ctx := opentracing.StartSpanFromContext(ctx, c.getSpanName("InsertBuildEvent"))
	defer func() { helpers.FinishSpanWithError(span, err) }()

	return c.Client.InsertBuildEvent(ctx, event)
}

func (c *tracingClient) InsertReleaseEvent(ctx context.Context, event PipelineReleaseEvent) (err error) {

	span, ctx := opentracing.StartSpanFromContext(ctx, c.getSpanName("InsertReleaseEvent"))
	defer func() { helpers.FinishSpanWithError(span, err) }()

	return c.Client.InsertReleaseEvent(ctx, event)
}

func (c *tracingClient) getSpanName(funcName string) string {
	return "bigquery:" + funcName
}
