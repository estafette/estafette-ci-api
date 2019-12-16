package cloudstorage

import (
	"context"
	"net/http"

	contracts "github.com/estafette/estafette-ci-contracts"
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

func (c *tracingClient) InsertBuildLog(ctx context.Context, buildLog contracts.BuildLog) (err error) {

	span, ctx := opentracing.StartSpanFromContext(ctx, c.getSpanName("InsertBuildLog"))
	defer span.Finish()
	defer func(span opentracing.Span) {
		c.handleError(span, err)
	}(span)

	return c.Client.InsertBuildLog(ctx, buildLog)
}

func (c *tracingClient) InsertReleaseLog(ctx context.Context, releaseLog contracts.ReleaseLog) (err error) {

	span, ctx := opentracing.StartSpanFromContext(ctx, c.getSpanName("InsertReleaseLog"))
	defer span.Finish()
	defer func(span opentracing.Span) {
		c.handleError(span, err)
	}(span)

	return c.Client.InsertReleaseLog(ctx, releaseLog)
}

func (c *tracingClient) GetPipelineBuildLogs(ctx context.Context, buildLog contracts.BuildLog, acceptGzipEncoding bool, responseWriter http.ResponseWriter) (err error) {

	span, ctx := opentracing.StartSpanFromContext(ctx, c.getSpanName("GetPipelineBuildLogs"))
	defer span.Finish()
	defer func(span opentracing.Span) {
		c.handleError(span, err)
	}(span)

	return c.Client.GetPipelineBuildLogs(ctx, buildLog, acceptGzipEncoding, responseWriter)
}

func (c *tracingClient) GetPipelineReleaseLogs(ctx context.Context, releaseLog contracts.ReleaseLog, acceptGzipEncoding bool, responseWriter http.ResponseWriter) (err error) {

	span, ctx := opentracing.StartSpanFromContext(ctx, c.getSpanName("GetPipelineReleaseLogs"))
	defer span.Finish()
	defer func(span opentracing.Span) {
		c.handleError(span, err)
	}(span)

	return c.Client.GetPipelineReleaseLogs(ctx, releaseLog, acceptGzipEncoding, responseWriter)
}

func (c *tracingClient) getSpanName(funcName string) string {
	return "cloudstorage:" + funcName
}

func (c *tracingClient) handleError(span opentracing.Span, err error) {
	if err != nil {
		ext.Error.Set(span, true)
		span.LogFields(log.Error(err))
	}
}
