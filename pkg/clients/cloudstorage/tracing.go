package cloudstorage

import (
	"context"
	"net/http"

	"github.com/estafette/estafette-ci-api/pkg/api"
	contracts "github.com/estafette/estafette-ci-contracts"
	"github.com/opentracing/opentracing-go"
)

// NewTracingClient returns a new instance of a tracing Client.
func NewTracingClient(c Client) Client {
	return &tracingClient{c, "cloudstorage"}
}

type tracingClient struct {
	Client Client
	prefix string
}

func (c *tracingClient) DeleteLogs(ctx context.Context, repoSource, repoOwner, repoName string) (err error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, api.GetSpanName(c.prefix, "DeleteLogs"))
	defer func() { api.FinishSpanWithError(span, err) }()

	return c.Client.DeleteLogs(ctx, repoSource, repoOwner, repoName)
}

func (c *tracingClient) InsertBuildLog(ctx context.Context, buildLog contracts.BuildLog) (err error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, api.GetSpanName(c.prefix, "InsertBuildLog"))
	defer func() { api.FinishSpanWithError(span, err) }()

	return c.Client.InsertBuildLog(ctx, buildLog)
}

func (c *tracingClient) InsertReleaseLog(ctx context.Context, releaseLog contracts.ReleaseLog) (err error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, api.GetSpanName(c.prefix, "InsertReleaseLog"))
	defer func() { api.FinishSpanWithError(span, err) }()

	return c.Client.InsertReleaseLog(ctx, releaseLog)
}

func (c *tracingClient) InsertBotLog(ctx context.Context, botLog contracts.BotLog) (err error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, api.GetSpanName(c.prefix, "InsertBotLog"))
	defer func() { api.FinishSpanWithError(span, err) }()

	return c.Client.InsertBotLog(ctx, botLog)
}

func (c *tracingClient) GetPipelineBuildLogs(ctx context.Context, buildLog contracts.BuildLog, acceptGzipEncoding bool, responseWriter http.ResponseWriter) (err error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, api.GetSpanName(c.prefix, "GetPipelineBuildLogs"))
	defer func() { api.FinishSpanWithError(span, err) }()

	return c.Client.GetPipelineBuildLogs(ctx, buildLog, acceptGzipEncoding, responseWriter)
}

func (c *tracingClient) GetPipelineReleaseLogs(ctx context.Context, releaseLog contracts.ReleaseLog, acceptGzipEncoding bool, responseWriter http.ResponseWriter) (err error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, api.GetSpanName(c.prefix, "GetPipelineReleaseLogs"))
	defer func() { api.FinishSpanWithError(span, err) }()

	return c.Client.GetPipelineReleaseLogs(ctx, releaseLog, acceptGzipEncoding, responseWriter)
}

func (c *tracingClient) GetPipelineBotLogs(ctx context.Context, botLog contracts.BotLog, acceptGzipEncoding bool, responseWriter http.ResponseWriter) (err error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, api.GetSpanName(c.prefix, "GetPipelineBotLogs"))
	defer func() { api.FinishSpanWithError(span, err) }()

	return c.Client.GetPipelineBotLogs(ctx, botLog, acceptGzipEncoding, responseWriter)
}

func (c *tracingClient) Rename(ctx context.Context, fromRepoSource, fromRepoOwner, fromRepoName, toRepoSource, toRepoOwner, toRepoName string) (err error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, api.GetSpanName(c.prefix, "Rename"))
	defer func() { api.FinishSpanWithError(span, err) }()

	return c.Client.Rename(ctx, fromRepoSource, fromRepoOwner, fromRepoName, toRepoSource, toRepoOwner, toRepoName)
}
