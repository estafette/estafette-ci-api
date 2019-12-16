package bigquery

import (
	"context"

	"github.com/opentracing/opentracing-go"
	"github.com/opentracing/opentracing-go/ext"
	"github.com/opentracing/opentracing-go/log"
)

// NewTracingClient returns a new instance of a tracing Client.
func NewTracingClient(c Client) Client {
	return &tracingClient{c}
}

type tracingClient struct {
	Client
}

func (c *tracingClient) Init(ctx context.Context) error {

	span, ctx := opentracing.StartSpanFromContext(ctx, c.getSpanName("Init"))
	defer span.Finish()

	return c.handleError(span, c.Client.Init(ctx))
}

func (c *tracingClient) CheckIfDatasetExists(ctx context.Context) bool {

	span, ctx := opentracing.StartSpanFromContext(ctx, c.getSpanName("CheckIfDatasetExists"))
	defer span.Finish()

	return c.Client.CheckIfDatasetExists(ctx)
}

func (c *tracingClient) CheckIfTableExists(ctx context.Context, table string) bool {

	span, ctx := opentracing.StartSpanFromContext(ctx, c.getSpanName("CheckIfTableExists"))
	defer span.Finish()

	return c.Client.CheckIfTableExists(ctx, table)
}

func (c *tracingClient) CreateTable(ctx context.Context, table string, typeForSchema interface{}, partitionField string, waitReady bool) error {

	span, ctx := opentracing.StartSpanFromContext(ctx, c.getSpanName("CreateTable"))
	defer span.Finish()

	return c.handleError(span, c.Client.CreateTable(ctx, table, typeForSchema, partitionField, waitReady))
}

func (c *tracingClient) UpdateTableSchema(ctx context.Context, table string, typeForSchema interface{}) error {

	span, ctx := opentracing.StartSpanFromContext(ctx, c.getSpanName("UpdateTableSchema"))
	defer span.Finish()

	return c.handleError(span, c.Client.UpdateTableSchema(ctx, table, typeForSchema))
}

func (c *tracingClient) InsertBuildEvent(ctx context.Context, event PipelineBuildEvent) error {

	span, ctx := opentracing.StartSpanFromContext(ctx, c.getSpanName("InsertBuildEvent"))
	defer span.Finish()

	return c.handleError(span, c.Client.InsertBuildEvent(ctx, event))
}

func (c *tracingClient) InsertReleaseEvent(ctx context.Context, event PipelineReleaseEvent) error {

	span, ctx := opentracing.StartSpanFromContext(ctx, c.getSpanName("InsertReleaseEvent"))
	defer span.Finish()

	return c.handleError(span, c.Client.InsertReleaseEvent(ctx, event))
}

func (c *tracingClient) getSpanName(funcName string) string {
	return "bigquery:" + funcName
}

func (c *tracingClient) handleError(span opentracing.Span, err error) error {
	if err != nil {
		ext.Error.Set(span, true)
		span.LogFields(log.Error(err))
	}
	return err
}
