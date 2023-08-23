package cloudstorage

import (
	"context"
	"net/http"
	"time"

	"github.com/estafette/estafette-ci-api/pkg/api"
	contracts "github.com/estafette/estafette-ci-contracts"
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

func (c *metricsClient) DeleteLogs(ctx context.Context, repoSource, repoOwner, repoName string) (err error) {
	defer func(begin time.Time) {
		api.UpdateMetrics(c.requestCount, c.requestLatency, "DeleteLogs", begin)
	}(time.Now())

	return c.Client.DeleteLogs(ctx, repoSource, repoOwner, repoName)
}

func (c *metricsClient) InsertBuildLog(ctx context.Context, buildLog contracts.BuildLog) (err error) {
	defer func(begin time.Time) {
		api.UpdateMetrics(c.requestCount, c.requestLatency, "InsertBuildLog", begin)
	}(time.Now())

	return c.Client.InsertBuildLog(ctx, buildLog)
}

func (c *metricsClient) InsertReleaseLog(ctx context.Context, releaseLog contracts.ReleaseLog) (err error) {
	defer func(begin time.Time) {
		api.UpdateMetrics(c.requestCount, c.requestLatency, "InsertReleaseLog", begin)
	}(time.Now())

	return c.Client.InsertReleaseLog(ctx, releaseLog)
}

func (c *metricsClient) InsertBotLog(ctx context.Context, botLog contracts.BotLog) (err error) {
	defer func(begin time.Time) {
		api.UpdateMetrics(c.requestCount, c.requestLatency, "InsertBotLog", begin)
	}(time.Now())

	return c.Client.InsertBotLog(ctx, botLog)
}

func (c *metricsClient) GetPipelineBuildLogs(ctx context.Context, buildLog contracts.BuildLog, acceptGzipEncoding bool, responseWriter http.ResponseWriter) (err error) {
	defer func(begin time.Time) {
		api.UpdateMetrics(c.requestCount, c.requestLatency, "GetPipelineBuildLogs", begin)
	}(time.Now())

	return c.Client.GetPipelineBuildLogs(ctx, buildLog, acceptGzipEncoding, responseWriter)
}

func (c *metricsClient) GetPipelineReleaseLogs(ctx context.Context, releaseLog contracts.ReleaseLog, acceptGzipEncoding bool, responseWriter http.ResponseWriter) (err error) {
	defer func(begin time.Time) {
		api.UpdateMetrics(c.requestCount, c.requestLatency, "GetPipelineReleaseLogs", begin)
	}(time.Now())

	return c.Client.GetPipelineReleaseLogs(ctx, releaseLog, acceptGzipEncoding, responseWriter)
}

func (c *metricsClient) GetPipelineBotLogs(ctx context.Context, botLog contracts.BotLog, acceptGzipEncoding bool, responseWriter http.ResponseWriter) (err error) {
	defer func(begin time.Time) {
		api.UpdateMetrics(c.requestCount, c.requestLatency, "GetPipelineBotLogs", begin)
	}(time.Now())

	return c.Client.GetPipelineBotLogs(ctx, botLog, acceptGzipEncoding, responseWriter)
}

func (c *metricsClient) Rename(ctx context.Context, fromRepoSource, fromRepoOwner, fromRepoName, toRepoSource, toRepoOwner, toRepoName string) (err error) {
	defer func(begin time.Time) {
		api.UpdateMetrics(c.requestCount, c.requestLatency, "Rename", begin)
	}(time.Now())

	return c.Client.Rename(ctx, fromRepoSource, fromRepoOwner, fromRepoName, toRepoSource, toRepoOwner, toRepoName)
}
