package bitbucket

import (
	"context"
	"time"

	"github.com/estafette/estafette-ci-api/pkg/api"
	"github.com/estafette/estafette-ci-api/pkg/clients/bitbucketapi"
	contracts "github.com/estafette/estafette-ci-contracts"
	manifest "github.com/estafette/estafette-ci-manifest"
	"github.com/go-kit/kit/metrics"
)

// NewMetricsService returns a new instance of a metrics Service.
func NewMetricsService(s Service, requestCount metrics.Counter, requestLatency metrics.Histogram) Service {
	return &metricsService{s, requestCount, requestLatency}
}

type metricsService struct {
	Service        Service
	requestCount   metrics.Counter
	requestLatency metrics.Histogram
}

func (s *metricsService) CreateJobForBitbucketPush(ctx context.Context, installation bitbucketapi.BitbucketAppInstallation, event bitbucketapi.RepositoryPushEvent) (err error) {
	defer func(begin time.Time) {
		api.UpdateMetrics(s.requestCount, s.requestLatency, "CreateJobForBitbucketPush", begin)
	}(time.Now())

	return s.Service.CreateJobForBitbucketPush(ctx, installation, event)
}

func (s *metricsService) PublishBitbucketEvent(ctx context.Context, event manifest.EstafetteBitbucketEvent) (err error) {
	defer func(begin time.Time) {
		api.UpdateMetrics(s.requestCount, s.requestLatency, "PublishBitbucketEvent", begin)
	}(time.Now())

	return s.Service.PublishBitbucketEvent(ctx, event)
}

func (s *metricsService) Rename(ctx context.Context, fromRepoSource, fromRepoOwner, fromRepoName, toRepoSource, toRepoOwner, toRepoName string) (err error) {
	defer func(begin time.Time) { api.UpdateMetrics(s.requestCount, s.requestLatency, "Rename", begin) }(time.Now())

	return s.Service.Rename(ctx, fromRepoSource, fromRepoOwner, fromRepoName, toRepoSource, toRepoOwner, toRepoName)
}

func (s *metricsService) Archive(ctx context.Context, repoSource, repoOwner, repoName string) (err error) {
	defer func(begin time.Time) { api.UpdateMetrics(s.requestCount, s.requestLatency, "Archive", begin) }(time.Now())

	return s.Service.Archive(ctx, repoSource, repoOwner, repoName)
}

func (s *metricsService) Unarchive(ctx context.Context, repoSource, repoOwner, repoName string) (err error) {
	defer func(begin time.Time) { api.UpdateMetrics(s.requestCount, s.requestLatency, "Unarchive", begin) }(time.Now())

	return s.Service.Unarchive(ctx, repoSource, repoOwner, repoName)
}

func (s *metricsService) IsAllowedOwner(ctx context.Context, repository *bitbucketapi.Repository) (isAllowed bool, organizations []*contracts.Organization) {
	defer func(begin time.Time) { api.UpdateMetrics(s.requestCount, s.requestLatency, "IsAllowedOwner", begin) }(time.Now())

	return s.Service.IsAllowedOwner(ctx, repository)
}
