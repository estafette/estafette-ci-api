package bigquery

import (
	"context"
	"time"

	"github.com/estafette/estafette-ci-api/pkg/api"
	"github.com/go-kit/kit/metrics"
)

// NewMetricsClient returns a new instance of a metrics Client.
func NewMetricsClient(c Client, requestCount metrics.Counter, requestLatency metrics.Histogram) Client {
	return &metricsClient{c, requestCount, requestLatency}
}

type metricsClient struct {
	Client         Client
	requestCount   metrics.Counter
	requestLatency metrics.Histogram
}

func (c *metricsClient) Init(ctx context.Context) (err error) {
	defer func(begin time.Time) { api.UpdateMetrics(c.requestCount, c.requestLatency, "Init", begin) }(time.Now())

	return c.Client.Init(ctx)
}

func (c *metricsClient) CheckIfDatasetExists(ctx context.Context) (exists bool) {
	defer func(begin time.Time) {
		api.UpdateMetrics(c.requestCount, c.requestLatency, "CheckIfDatasetExists", begin)
	}(time.Now())

	return c.Client.CheckIfDatasetExists(ctx)
}

func (c *metricsClient) CheckIfTableExists(ctx context.Context, table string) (exists bool) {
	defer func(begin time.Time) {
		api.UpdateMetrics(c.requestCount, c.requestLatency, "CheckIfTableExists", begin)
	}(time.Now())

	return c.Client.CheckIfTableExists(ctx, table)
}

func (c *metricsClient) CreateTable(ctx context.Context, table string, typeForSchema interface{}, partitionField string, waitReady bool) (err error) {
	defer func(begin time.Time) { api.UpdateMetrics(c.requestCount, c.requestLatency, "CreateTable", begin) }(time.Now())

	return c.Client.CreateTable(ctx, table, typeForSchema, partitionField, waitReady)
}

func (c *metricsClient) UpdateTableSchema(ctx context.Context, table string, typeForSchema interface{}) (err error) {
	defer func(begin time.Time) {
		api.UpdateMetrics(c.requestCount, c.requestLatency, "UpdateTableSchema", begin)
	}(time.Now())

	return c.Client.UpdateTableSchema(ctx, table, typeForSchema)
}

func (c *metricsClient) InsertBuildEvent(ctx context.Context, event PipelineBuildEvent) (err error) {
	defer func(begin time.Time) {
		api.UpdateMetrics(c.requestCount, c.requestLatency, "InsertBuildEvent", begin)
	}(time.Now())

	return c.Client.InsertBuildEvent(ctx, event)
}

func (c *metricsClient) InsertReleaseEvent(ctx context.Context, event PipelineReleaseEvent) (err error) {
	defer func(begin time.Time) {
		api.UpdateMetrics(c.requestCount, c.requestLatency, "InsertReleaseEvent", begin)
	}(time.Now())

	return c.Client.InsertReleaseEvent(ctx, event)
}
