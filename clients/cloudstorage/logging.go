package cloudstorage

import (
	"context"
	"net/http"

	"github.com/estafette/estafette-ci-api/api"
	contracts "github.com/estafette/estafette-ci-contracts"
)

// NewLoggingClient returns a new instance of a logging Client.
func NewLoggingClient(c Client) Client {
	return &loggingClient{c, "cloudstorage"}
}

type loggingClient struct {
	Client
	prefix string
}

func (c *loggingClient) InsertBuildLog(ctx context.Context, buildLog contracts.BuildLog) (err error) {
	defer func() { api.HandleLogError(c.prefix, "Client", "InsertBuildLog", err) }()

	return c.Client.InsertBuildLog(ctx, buildLog)
}

func (c *loggingClient) InsertReleaseLog(ctx context.Context, releaseLog contracts.ReleaseLog) (err error) {
	defer func() { api.HandleLogError(c.prefix, "Client", "InsertReleaseLog", err) }()

	return c.Client.InsertReleaseLog(ctx, releaseLog)
}

func (c *loggingClient) GetPipelineBuildLogs(ctx context.Context, buildLog contracts.BuildLog, acceptGzipEncoding bool, responseWriter http.ResponseWriter) (err error) {
	defer func() { api.HandleLogError(c.prefix, "Client", "GetPipelineBuildLogs", err) }()

	return c.Client.GetPipelineBuildLogs(ctx, buildLog, acceptGzipEncoding, responseWriter)
}

func (c *loggingClient) GetPipelineReleaseLogs(ctx context.Context, releaseLog contracts.ReleaseLog, acceptGzipEncoding bool, responseWriter http.ResponseWriter) (err error) {
	defer func() { api.HandleLogError(c.prefix, "Client", "GetPipelineReleaseLogs", err) }()

	return c.Client.GetPipelineReleaseLogs(ctx, releaseLog, acceptGzipEncoding, responseWriter)
}

func (c *loggingClient) Rename(ctx context.Context, fromRepoSource, fromRepoOwner, fromRepoName, toRepoSource, toRepoOwner, toRepoName string) (err error) {
	defer func() { api.HandleLogError(c.prefix, "Client", "Rename", err) }()

	return c.Client.Rename(ctx, fromRepoSource, fromRepoOwner, fromRepoName, toRepoSource, toRepoOwner, toRepoName)
}
