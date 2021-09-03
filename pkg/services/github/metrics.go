package github

import (
	"context"
	"time"

	"github.com/estafette/estafette-ci-api/pkg/api"
	"github.com/estafette/estafette-ci-api/pkg/clients/githubapi"
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

func (s *metricsService) CreateJobForGithubPush(ctx context.Context, event githubapi.PushEvent) (err error) {
	defer func(begin time.Time) {
		api.UpdateMetrics(s.requestCount, s.requestLatency, "CreateJobForGithubPush", begin)
	}(time.Now())

	return s.Service.CreateJobForGithubPush(ctx, event)
}

func (s *metricsService) PublishGithubEvent(ctx context.Context, event manifest.EstafetteGithubEvent) (err error) {
	defer func(begin time.Time) {
		api.UpdateMetrics(s.requestCount, s.requestLatency, "PublishGithubEvent", begin)
	}(time.Now())

	return s.Service.PublishGithubEvent(ctx, event)
}

func (s *metricsService) HasValidSignature(ctx context.Context, body []byte, appIDHeader, signatureHeader string) (valid bool, err error) {
	defer func(begin time.Time) {
		api.UpdateMetrics(s.requestCount, s.requestLatency, "HasValidSignature", begin)
	}(time.Now())

	return s.Service.HasValidSignature(ctx, body, appIDHeader, signatureHeader)
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

func (s *metricsService) IsAllowedInstallation(ctx context.Context, installationID int) (isAllowed bool, organizations []*contracts.Organization) {
	defer func(begin time.Time) { api.UpdateMetrics(s.requestCount, s.requestLatency, "Rename", begin) }(time.Now())

	return s.Service.IsAllowedInstallation(ctx, installationID)
}
