package cloudstorage

import (
	"context"
	"net/http"

	contracts "github.com/estafette/estafette-ci-contracts"
	"github.com/opentracing/opentracing-go"
)

type tracingClient struct {
	Client
}

// NewTracingClient returns a new instance of a tracing Client.
func NewTracingClient(c Client) Client {
	return &tracingClient{c}
}

func (s *tracingClient) InsertBuildLog(ctx context.Context, buildLog contracts.BuildLog) error {

	span, ctx := opentracing.StartSpanFromContext(ctx, s.getSpanName("InsertBuildLog"))
	defer span.Finish()

	return s.Client.InsertBuildLog(ctx, buildLog)
}

func (s *tracingClient) InsertReleaseLog(ctx context.Context, releaseLog contracts.ReleaseLog) (err error) {

	span, ctx := opentracing.StartSpanFromContext(ctx, s.getSpanName("InsertReleaseLog"))
	defer span.Finish()

	return s.Client.InsertReleaseLog(ctx, releaseLog)
}

func (s *tracingClient) GetPipelineBuildLogs(ctx context.Context, buildLog contracts.BuildLog, acceptGzipEncoding bool, responseWriter http.ResponseWriter) (err error) {

	span, ctx := opentracing.StartSpanFromContext(ctx, s.getSpanName("GetPipelineBuildLogs"))
	defer span.Finish()

	return s.Client.GetPipelineBuildLogs(ctx, buildLog, acceptGzipEncoding, responseWriter)
}

func (s *tracingClient) GetPipelineReleaseLogs(ctx context.Context, releaseLog contracts.ReleaseLog, acceptGzipEncoding bool, responseWriter http.ResponseWriter) (err error) {

	span, ctx := opentracing.StartSpanFromContext(ctx, s.getSpanName("GetPipelineReleaseLogs"))
	defer span.Finish()

	return s.Client.GetPipelineReleaseLogs(ctx, releaseLog, acceptGzipEncoding, responseWriter)
}

func (s *tracingClient) getSpanName(funcName string) string {
	return "cloudstorage:" + funcName
}
