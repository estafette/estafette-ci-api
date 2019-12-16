package builderapi

import (
	"context"

	batchv1 "github.com/ericchiang/k8s/apis/batch/v1"
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

func (c *tracingClient) CreateCiBuilderJob(ctx context.Context, params CiBuilderParams) (job *batchv1.Job, err error) {

	span, ctx := opentracing.StartSpanFromContext(ctx, c.getSpanName("CreateCiBuilderJob"))
	defer span.Finish()
	defer func(span opentracing.Span) {
		c.handleError(span, err)
	}(span)

	return c.Client.CreateCiBuilderJob(ctx, params)
}

func (c *tracingClient) RemoveCiBuilderJob(ctx context.Context, jobName string) (err error) {

	span, ctx := opentracing.StartSpanFromContext(ctx, c.getSpanName("RemoveCiBuilderJob"))
	defer span.Finish()
	defer func(span opentracing.Span) {
		c.handleError(span, err)
	}(span)

	return c.Client.RemoveCiBuilderJob(ctx, jobName)
}

func (c *tracingClient) CancelCiBuilderJob(ctx context.Context, jobName string) (err error) {

	span, ctx := opentracing.StartSpanFromContext(ctx, c.getSpanName("CancelCiBuilderJob"))
	defer span.Finish()
	defer func(span opentracing.Span) {
		c.handleError(span, err)
	}(span)

	return c.Client.CancelCiBuilderJob(ctx, jobName)
}

func (c *tracingClient) RemoveCiBuilderConfigMap(ctx context.Context, configmapName string) (err error) {

	span, ctx := opentracing.StartSpanFromContext(ctx, c.getSpanName("RemoveCiBuilderConfigMap"))
	defer span.Finish()
	defer func(span opentracing.Span) {
		c.handleError(span, err)
	}(span)

	return c.Client.RemoveCiBuilderConfigMap(ctx, configmapName)
}

func (c *tracingClient) RemoveCiBuilderSecret(ctx context.Context, secretName string) (err error) {

	span, ctx := opentracing.StartSpanFromContext(ctx, c.getSpanName("RemoveCiBuilderSecret"))
	defer span.Finish()
	defer func(span opentracing.Span) {
		c.handleError(span, err)
	}(span)

	return c.Client.RemoveCiBuilderSecret(ctx, secretName)
}

func (c *tracingClient) TailCiBuilderJobLogs(ctx context.Context, jobName string, logChannel chan contracts.TailLogLine) (err error) {

	span, ctx := opentracing.StartSpanFromContext(ctx, c.getSpanName("TailCiBuilderJobLogs"))
	defer span.Finish()
	defer func(span opentracing.Span) {
		c.handleError(span, err)
	}(span)

	return c.Client.TailCiBuilderJobLogs(ctx, jobName, logChannel)
}

func (c *tracingClient) GetJobName(ctx context.Context, jobType, repoOwner, repoName, id string) string {

	span, ctx := opentracing.StartSpanFromContext(ctx, c.getSpanName("GetJobName"))
	defer span.Finish()

	return c.Client.GetJobName(ctx, jobType, repoOwner, repoOwner, id)
}

func (c *tracingClient) GetBuilderConfig(ctx context.Context, params CiBuilderParams, jobName string) contracts.BuilderConfig {

	span, ctx := opentracing.StartSpanFromContext(ctx, c.getSpanName("GetBuilderConfig"))
	defer span.Finish()

	return c.Client.GetBuilderConfig(ctx, params, jobName)
}

func (c *tracingClient) getSpanName(funcName string) string {
	return "builderapi:" + funcName
}

func (c *tracingClient) handleError(span opentracing.Span, err error) {
	if err != nil {
		ext.Error.Set(span, true)
		span.LogFields(log.Error(err))
	}
}
