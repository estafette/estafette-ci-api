package builderapi

import (
	"context"

	batchv1 "github.com/ericchiang/k8s/apis/batch/v1"
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

func (s *tracingClient) CreateCiBuilderJob(ctx context.Context, params CiBuilderParams) (*batchv1.Job, error) {

	span, ctx := opentracing.StartSpanFromContext(ctx, s.getSpanName("CreateCiBuilderJob"))
	defer span.Finish()

	return s.Client.CreateCiBuilderJob(ctx, params)
}

func (s *tracingClient) RemoveCiBuilderJob(ctx context.Context, jobName string) error {

	span, ctx := opentracing.StartSpanFromContext(ctx, s.getSpanName("RemoveCiBuilderJob"))
	defer span.Finish()

	return s.Client.RemoveCiBuilderJob(ctx, jobName)
}

func (s *tracingClient) CancelCiBuilderJob(ctx context.Context, jobName string) error {

	span, ctx := opentracing.StartSpanFromContext(ctx, s.getSpanName("CancelCiBuilderJob"))
	defer span.Finish()

	return s.Client.CancelCiBuilderJob(ctx, jobName)
}

func (s *tracingClient) RemoveCiBuilderConfigMap(ctx context.Context, configmapName string) error {

	span, ctx := opentracing.StartSpanFromContext(ctx, s.getSpanName("RemoveCiBuilderConfigMap"))
	defer span.Finish()

	return s.Client.RemoveCiBuilderConfigMap(ctx, configmapName)
}

func (s *tracingClient) RemoveCiBuilderSecret(ctx context.Context, secretName string) error {

	span, ctx := opentracing.StartSpanFromContext(ctx, s.getSpanName("RemoveCiBuilderSecret"))
	defer span.Finish()

	return s.Client.RemoveCiBuilderSecret(ctx, secretName)
}

func (s *tracingClient) TailCiBuilderJobLogs(ctx context.Context, jobName string, logChannel chan contracts.TailLogLine) error {

	span, ctx := opentracing.StartSpanFromContext(ctx, s.getSpanName("TailCiBuilderJobLogs"))
	defer span.Finish()

	return s.Client.TailCiBuilderJobLogs(ctx, jobName, logChannel)
}

func (s *tracingClient) GetJobName(ctx context.Context, jobType, repoOwner, repoName, id string) string {

	span, ctx := opentracing.StartSpanFromContext(ctx, s.getSpanName("GetJobName"))
	defer span.Finish()

	return s.Client.GetJobName(ctx, jobType, repoOwner, repoOwner, id)
}

func (s *tracingClient) GetBuilderConfig(ctx context.Context, params CiBuilderParams, jobName string) contracts.BuilderConfig {

	span, ctx := opentracing.StartSpanFromContext(ctx, s.getSpanName("GetBuilderConfig"))
	defer span.Finish()

	return s.Client.GetBuilderConfig(ctx, params, jobName)
}

func (s *tracingClient) getSpanName(funcName string) string {
	return "builderapi:" + funcName
}
