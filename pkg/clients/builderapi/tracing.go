package builderapi

import (
	"context"

	"github.com/estafette/estafette-ci-api/pkg/api"
	contracts "github.com/estafette/estafette-ci-contracts"
	"github.com/opentracing/opentracing-go"
	batchv1 "k8s.io/api/batch/v1"
)

// NewTracingClient returns a new instance of a tracing Client.
func NewTracingClient(c Client) Client {
	return &tracingClient{c, "builderapi"}
}

type tracingClient struct {
	Client Client
	prefix string
}

func (c *tracingClient) CreateCiBuilderJob(ctx context.Context, params CiBuilderParams) (job *batchv1.Job, err error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, api.GetSpanName(c.prefix, "CreateCiBuilderJob"))
	defer func() { api.FinishSpanWithError(span, err) }()

	return c.Client.CreateCiBuilderJob(ctx, params)
}

func (c *tracingClient) RemoveCiBuilderJob(ctx context.Context, jobName string) (err error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, api.GetSpanName(c.prefix, "RemoveCiBuilderJob"))
	defer func() { api.FinishSpanWithError(span, err) }()

	return c.Client.RemoveCiBuilderJob(ctx, jobName)
}

func (c *tracingClient) CancelCiBuilderJob(ctx context.Context, jobName string) (err error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, api.GetSpanName(c.prefix, "CancelCiBuilderJob"))
	defer func() { api.FinishSpanWithError(span, err) }()

	return c.Client.CancelCiBuilderJob(ctx, jobName)
}

func (c *tracingClient) RemoveCiBuilderConfigMap(ctx context.Context, configmapName string) (err error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, api.GetSpanName(c.prefix, "RemoveCiBuilderConfigMap"))
	defer func() { api.FinishSpanWithError(span, err) }()

	return c.Client.RemoveCiBuilderConfigMap(ctx, configmapName)
}

func (c *tracingClient) RemoveCiBuilderSecret(ctx context.Context, secretName string) (err error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, api.GetSpanName(c.prefix, "RemoveCiBuilderSecret"))
	defer func() { api.FinishSpanWithError(span, err) }()

	return c.Client.RemoveCiBuilderSecret(ctx, secretName)
}

func (c *tracingClient) RemoveCiBuilderImagePullSecret(ctx context.Context, secretName string) (err error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, api.GetSpanName(c.prefix, "RemoveCiBuilderImagePullSecret"))
	defer func() { api.FinishSpanWithError(span, err) }()

	return c.Client.RemoveCiBuilderImagePullSecret(ctx, secretName)
}

func (c *tracingClient) TailCiBuilderJobLogs(ctx context.Context, jobName string, logChannel chan contracts.TailLogLine) (err error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, api.GetSpanName(c.prefix, "TailCiBuilderJobLogs"))
	defer func() { api.FinishSpanWithError(span, err) }()

	return c.Client.TailCiBuilderJobLogs(ctx, jobName, logChannel)
}

func (c *tracingClient) GetJobName(ctx context.Context, jobType contracts.JobType, repoOwner, repoName, id string) string {
	span, ctx := opentracing.StartSpanFromContext(ctx, api.GetSpanName(c.prefix, "GetJobName"))
	defer func() { api.FinishSpan(span) }()

	return c.Client.GetJobName(ctx, jobType, repoOwner, repoName, id)
}
