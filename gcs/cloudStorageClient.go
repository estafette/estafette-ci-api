package gcs

import (
	"compress/gzip"
	"context"
	"encoding/json"
	"fmt"
	"path"

	"cloud.google.com/go/storage"
	"github.com/estafette/estafette-ci-api/config"
	contracts "github.com/estafette/estafette-ci-contracts"
	"github.com/opentracing/opentracing-go"
)

// CloudStorageClient is the interface for connecting to google cloud storage
type CloudStorageClient interface {
	InsertBuildLog(ctx context.Context, buildLog contracts.BuildLog) (err error)
	InsertReleaseLog(ctx context.Context, releaseLog contracts.ReleaseLog) (err error)
	GetPipelineBuildLogs(ctx context.Context, buildLog contracts.BuildLog) (updatedBuildLog contracts.BuildLog, err error)
	GetPipelineReleaseLogs(ctx context.Context, releaseLog contracts.ReleaseLog) (updatedReleaseLog contracts.ReleaseLog, err error)
}

type cloudStorageClientImpl struct {
	client *storage.Client
	config *config.CloudStorageConfig
}

// NewCloudStorageClient returns new CloudStorageClient
func NewCloudStorageClient(config *config.CloudStorageConfig) (CloudStorageClient, error) {

	if config == nil {
		return &cloudStorageClientImpl{
			config: config,
		}, nil
	}

	ctx := context.Background()

	storageClient, err := storage.NewClient(ctx)
	if err != nil {
		return nil, err
	}

	return &cloudStorageClientImpl{
		client: storageClient,
		config: config,
	}, nil
}

func (impl *cloudStorageClientImpl) InsertBuildLog(ctx context.Context, buildLog contracts.BuildLog) (err error) {

	span, ctx := opentracing.StartSpanFromContext(ctx, "CloudStorageClient::InsertBuildLog")
	defer span.Finish()

	logPath := impl.getBuildLogPath(buildLog)

	return impl.insertLog(ctx, logPath, buildLog.Steps)
}

func (impl *cloudStorageClientImpl) InsertReleaseLog(ctx context.Context, releaseLog contracts.ReleaseLog) (err error) {

	span, ctx := opentracing.StartSpanFromContext(ctx, "CloudStorageClient::InsertReleaseLog")
	defer span.Finish()

	logPath := impl.getReleaseLogPath(releaseLog)

	return impl.insertLog(ctx, logPath, releaseLog.Steps)
}

func (impl *cloudStorageClientImpl) GetPipelineBuildLogs(ctx context.Context, buildLog contracts.BuildLog) (updatedBuildLog contracts.BuildLog, err error) {

	span, ctx := opentracing.StartSpanFromContext(ctx, "CloudStorageClient::GetPipelineBuildLogs")
	defer span.Finish()

	logPath := impl.getBuildLogPath(buildLog)

	steps, err := impl.getLog(ctx, logPath)
	if err != nil {
		return buildLog, err
	}

	updatedBuildLog = buildLog
	updatedBuildLog.Steps = steps

	return updatedBuildLog, nil
}

func (impl *cloudStorageClientImpl) GetPipelineReleaseLogs(ctx context.Context, releaseLog contracts.ReleaseLog) (updatedReleaseLog contracts.ReleaseLog, err error) {

	span, ctx := opentracing.StartSpanFromContext(ctx, "CloudStorageClient::GetPipelineReleaseLogs")
	defer span.Finish()

	logPath := impl.getReleaseLogPath(releaseLog)

	steps, err := impl.getLog(ctx, logPath)
	if err != nil {
		return releaseLog, err
	}

	updatedReleaseLog = releaseLog
	updatedReleaseLog.Steps = steps

	return updatedReleaseLog, nil

}

func (impl *cloudStorageClientImpl) insertLog(ctx context.Context, path string, steps []*contracts.BuildLogStep) (err error) {

	bucket := impl.client.Bucket(impl.config.Bucket)

	// marshal json
	jsonBytes, err := json.Marshal(steps)
	if err != nil {
		return err
	}

	// create writer for cloud storage object
	logObject := bucket.Object(path)
	writer := logObject.NewWriter(ctx)
	if writer == nil {
		return fmt.Errorf("Writer for logobject %v is nil", path)
	}
	defer writer.Close()

	// write compressed bytes
	gz, err := gzip.NewWriterLevel(writer, gzip.BestSpeed)
	if err != nil {
		return err
	}
	_, err = gz.Write(jsonBytes)
	if err != nil {
		_ = writer.Close()
		return err
	}
	err = gz.Close()
	if err != nil {
		_ = writer.Close()
		return err
	}
	err = writer.Close()
	if err != nil {
		return err
	}

	return nil
}

func (impl *cloudStorageClientImpl) getLog(ctx context.Context, path string) (steps []*contracts.BuildLogStep, err error) {

	bucket := impl.client.Bucket(impl.config.Bucket)

	// create reader for cloud storage object
	logObject := bucket.Object(path)
	reader, err := logObject.NewReader(ctx)
	if err != nil {
		return nil, err
	}
	if reader == nil {
		return nil, fmt.Errorf("Writer for logobject %v is nil", path)
	}

	// read compressed bytes
	var jsonBytes []byte
	gz, err := gzip.NewReader(reader)
	_, err = gz.Read(jsonBytes)
	if err != nil {
		_ = reader.Close()
		return nil, err
	}
	err = gz.Close()
	if err != nil {
		_ = reader.Close()
		return nil, err
	}
	err = reader.Close()
	if err != nil {
		return nil, err
	}

	// unmarshal json
	err = json.Unmarshal(jsonBytes, &steps)
	if err != nil {
		return nil, err
	}

	return
}

func (impl *cloudStorageClientImpl) getBuildLogPath(buildLog contracts.BuildLog) (logPath string) {

	logPath = path.Join(impl.config.LogsDirectory, buildLog.RepoSource, buildLog.RepoOwner, buildLog.RepoName, "builds", fmt.Sprintf("%v.log", buildLog.ID))

	return logPath
}

func (impl *cloudStorageClientImpl) getReleaseLogPath(releaseLog contracts.ReleaseLog) (logPath string) {

	logPath = path.Join(impl.config.LogsDirectory, releaseLog.RepoSource, releaseLog.RepoOwner, releaseLog.RepoName, "releases", fmt.Sprintf("%v.log", releaseLog.ID))

	return logPath
}
