package builderapi

import (
	"context"
	"time"

	"github.com/estafette/estafette-ci-api/pkg/api"
	contracts "github.com/estafette/estafette-ci-contracts"
	"github.com/go-kit/kit/metrics"
	batchv1 "k8s.io/api/batch/v1"
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

func (c *metricsClient) CreateCiBuilderJob(ctx context.Context, params CiBuilderParams) (job *batchv1.Job, err error) {
	defer func(begin time.Time) {
		api.UpdateMetrics(c.requestCount, c.requestLatency, "CreateCiBuilderJob", begin)
	}(time.Now())

	return c.Client.CreateCiBuilderJob(ctx, params)
}

func (c *metricsClient) RemoveCiBuilderJob(ctx context.Context, jobName string) (err error) {
	defer func(begin time.Time) {
		api.UpdateMetrics(c.requestCount, c.requestLatency, "RemoveCiBuilderJob", begin)
	}(time.Now())

	return c.Client.RemoveCiBuilderJob(ctx, jobName)
}

func (c *metricsClient) CancelCiBuilderJob(ctx context.Context, jobName string) (err error) {
	defer func(begin time.Time) {
		api.UpdateMetrics(c.requestCount, c.requestLatency, "CancelCiBuilderJob", begin)
	}(time.Now())

	return c.Client.CancelCiBuilderJob(ctx, jobName)
}

func (c *metricsClient) RemoveCiBuilderConfigMap(ctx context.Context, configmapName string) (err error) {
	defer func(begin time.Time) {
		api.UpdateMetrics(c.requestCount, c.requestLatency, "RemoveCiBuilderConfigMap", begin)
	}(time.Now())

	return c.Client.RemoveCiBuilderConfigMap(ctx, configmapName)
}

func (c *metricsClient) RemoveCiBuilderSecret(ctx context.Context, secretName string) (err error) {
	defer func(begin time.Time) {
		api.UpdateMetrics(c.requestCount, c.requestLatency, "RemoveCiBuilderSecret", begin)
	}(time.Now())

	return c.Client.RemoveCiBuilderSecret(ctx, secretName)
}

func (c *metricsClient) RemoveCiBuilderImagePullSecret(ctx context.Context, secretName string) (err error) {
	defer func(begin time.Time) {
		api.UpdateMetrics(c.requestCount, c.requestLatency, "RemoveCiBuilderImagePullSecret", begin)
	}(time.Now())

	return c.Client.RemoveCiBuilderImagePullSecret(ctx, secretName)
}

func (c *metricsClient) TailCiBuilderJobLogs(ctx context.Context, jobName string, logChannel chan contracts.TailLogLine) (err error) {
	defer func(begin time.Time) {
		api.UpdateMetrics(c.requestCount, c.requestLatency, "TailCiBuilderJobLogs", begin)
	}(time.Now())

	return c.Client.TailCiBuilderJobLogs(ctx, jobName, logChannel)
}

func (c *metricsClient) GetJobName(ctx context.Context, jobType contracts.JobType, repoOwner, repoName, id string) string {
	defer func(begin time.Time) { api.UpdateMetrics(c.requestCount, c.requestLatency, "GetJobName", begin) }(time.Now())

	return c.Client.GetJobName(ctx, jobType, repoOwner, repoName, id)
}
