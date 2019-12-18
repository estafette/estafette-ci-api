package cloudstorage

import (
	"context"
	"net/http"

	contracts "github.com/estafette/estafette-ci-contracts"
)

type MockClient struct {
	InsertBuildLogFunc         func(ctx context.Context, buildLog contracts.BuildLog) (err error)
	InsertReleaseLogFunc       func(ctx context.Context, releaseLog contracts.ReleaseLog) (err error)
	GetPipelineBuildLogsFunc   func(ctx context.Context, buildLog contracts.BuildLog, acceptGzipEncoding bool, responseWriter http.ResponseWriter) (err error)
	GetPipelineReleaseLogsFunc func(ctx context.Context, releaseLog contracts.ReleaseLog, acceptGzipEncoding bool, responseWriter http.ResponseWriter) (err error)
	RenameFunc                 func(ctx context.Context, fromRepoSource, fromRepoOwner, fromRepoName, toRepoSource, toRepoOwner, toRepoName string) (err error)
}

func (c MockClient) InsertBuildLog(ctx context.Context, buildLog contracts.BuildLog) (err error) {
	if c.InsertBuildLogFunc == nil {
		return
	}
	return c.InsertBuildLogFunc(ctx, buildLog)
}

func (c MockClient) InsertReleaseLog(ctx context.Context, releaseLog contracts.ReleaseLog) (err error) {
	if c.InsertReleaseLogFunc == nil {
		return
	}
	return c.InsertReleaseLogFunc(ctx, releaseLog)
}

func (c MockClient) GetPipelineBuildLogs(ctx context.Context, buildLog contracts.BuildLog, acceptGzipEncoding bool, responseWriter http.ResponseWriter) (err error) {
	if c.GetPipelineBuildLogsFunc == nil {
		return
	}
	return c.GetPipelineBuildLogsFunc(ctx, buildLog, acceptGzipEncoding, responseWriter)
}

func (c MockClient) GetPipelineReleaseLogs(ctx context.Context, releaseLog contracts.ReleaseLog, acceptGzipEncoding bool, responseWriter http.ResponseWriter) (err error) {
	if c.GetPipelineReleaseLogsFunc == nil {
		return
	}
	return c.GetPipelineReleaseLogsFunc(ctx, releaseLog, acceptGzipEncoding, responseWriter)
}

func (c MockClient) Rename(ctx context.Context, fromRepoSource, fromRepoOwner, fromRepoName, toRepoSource, toRepoOwner, toRepoName string) (err error) {
	if c.RenameFunc == nil {
		return
	}
	return c.RenameFunc(ctx, fromRepoSource, fromRepoOwner, fromRepoName, toRepoSource, toRepoOwner, toRepoName)
}
