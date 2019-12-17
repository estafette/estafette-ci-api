package builderapi

import (
	"context"

	batchv1 "github.com/ericchiang/k8s/apis/batch/v1"
	contracts "github.com/estafette/estafette-ci-contracts"
)

type MockClient struct {
	CreateCiBuilderJobFunc       func(ctx context.Context, params CiBuilderParams) (job *batchv1.Job, err error)
	RemoveCiBuilderJobFunc       func(ctx context.Context, jobName string) (err error)
	CancelCiBuilderJobFunc       func(ctx context.Context, jobName string) (err error)
	RemoveCiBuilderConfigMapFunc func(ctx context.Context, configmapName string) (err error)
	RemoveCiBuilderSecretFunc    func(ctx context.Context, secretName string) (err error)
	TailCiBuilderJobLogsFunc     func(ctx context.Context, jobName string, logChannel chan contracts.TailLogLine) (err error)
	GetJobNameFunc               func(ctx context.Context, jobType, repoOwner, repoName, id string) (jobname string)
	GetBuilderConfigFunc         func(ctx context.Context, params CiBuilderParams, jobName string) (config contracts.BuilderConfig)
}

func (c *MockClient) CreateCiBuilderJob(ctx context.Context, params CiBuilderParams) (job *batchv1.Job, err error) {
	if c.CreateCiBuilderJobFunc == nil {
		return
	}
	return c.CreateCiBuilderJobFunc(ctx, params)
}

func (c *MockClient) RemoveCiBuilderJob(ctx context.Context, jobName string) (err error) {
	if c.RemoveCiBuilderJobFunc == nil {
		return
	}
	return c.RemoveCiBuilderJobFunc(ctx, jobName)
}

func (c *MockClient) CancelCiBuilderJob(ctx context.Context, jobName string) (err error) {
	if c.CancelCiBuilderJobFunc == nil {
		return
	}
	return c.CancelCiBuilderJobFunc(ctx, jobName)
}

func (c *MockClient) RemoveCiBuilderConfigMap(ctx context.Context, configmapName string) (err error) {
	if c.RemoveCiBuilderConfigMapFunc == nil {
		return
	}
	return c.RemoveCiBuilderConfigMapFunc(ctx, configmapName)
}

func (c *MockClient) RemoveCiBuilderSecret(ctx context.Context, secretName string) (err error) {
	if c.RemoveCiBuilderSecretFunc == nil {
		return
	}
	return c.RemoveCiBuilderSecretFunc(ctx, secretName)
}

func (c *MockClient) TailCiBuilderJobLogs(ctx context.Context, jobName string, logChannel chan contracts.TailLogLine) (err error) {
	if c.TailCiBuilderJobLogsFunc == nil {
		return
	}
	return c.TailCiBuilderJobLogsFunc(ctx, jobName, logChannel)
}

func (c *MockClient) GetJobName(ctx context.Context, jobType, repoOwner, repoName, id string) (jobname string) {
	if c.GetJobNameFunc == nil {
		return
	}
	return c.GetJobNameFunc(ctx, jobType, repoOwner, repoName, id)
}

func (c *MockClient) GetBuilderConfig(ctx context.Context, params CiBuilderParams, jobName string) (config contracts.BuilderConfig) {
	if c.GetBuilderConfigFunc == nil {
		return contracts.BuilderConfig{}
	}
	return c.GetBuilderConfigFunc(ctx, params, jobName)
}
