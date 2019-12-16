package bigquery

import (
	"context"

	"github.com/opentracing/opentracing-go"
)

type tracingClient struct {
	Client
}

// NewTracingClient returns a new instance of a tracing Client.
func NewTracingClient(c Client) Client {
	return &tracingClient{c}
}

func (s *tracingClient) Init(ctx context.Context) error {

	span, ctx := opentracing.StartSpanFromContext(ctx, s.getSpanName("Init"))
	defer span.Finish()

	return s.Client.Init(ctx)
}

func (s *tracingClient) CheckIfDatasetExists(ctx context.Context) bool {

	span, ctx := opentracing.StartSpanFromContext(ctx, s.getSpanName("CheckIfDatasetExists"))
	defer span.Finish()

	return s.Client.CheckIfDatasetExists(ctx)
}

func (s *tracingClient) CheckIfTableExists(ctx context.Context, table string) bool {

	span, ctx := opentracing.StartSpanFromContext(ctx, s.getSpanName("CheckIfTableExists"))
	defer span.Finish()

	return s.Client.CheckIfTableExists(ctx, table)
}

func (s *tracingClient) CreateTable(ctx context.Context, table string, typeForSchema interface{}, partitionField string, waitReady bool) error {

	span, ctx := opentracing.StartSpanFromContext(ctx, s.getSpanName("CreateTable"))
	defer span.Finish()

	return s.Client.CreateTable(ctx, table, typeForSchema, partitionField, waitReady)
}

func (s *tracingClient) UpdateTableSchema(ctx context.Context, table string, typeForSchema interface{}) error {

	span, ctx := opentracing.StartSpanFromContext(ctx, s.getSpanName("UpdateTableSchema"))
	defer span.Finish()

	return s.Client.UpdateTableSchema(ctx, table, typeForSchema)
}

func (s *tracingClient) InsertBuildEvent(ctx context.Context, event PipelineBuildEvent) error {

	span, ctx := opentracing.StartSpanFromContext(ctx, s.getSpanName("InsertBuildEvent"))
	defer span.Finish()

	return s.Client.InsertBuildEvent(ctx, event)
}

func (s *tracingClient) InsertReleaseEvent(ctx context.Context, event PipelineReleaseEvent) error {

	span, ctx := opentracing.StartSpanFromContext(ctx, s.getSpanName("InsertReleaseEvent"))
	defer span.Finish()

	return s.Client.InsertReleaseEvent(ctx, event)
}

func (s *tracingClient) getSpanName(funcName string) string {
	return "bigquery:" + funcName
}
