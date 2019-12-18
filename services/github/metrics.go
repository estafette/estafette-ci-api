package github

import (
	"context"
	"time"

	"github.com/estafette/estafette-ci-api/clients/githubapi"
	"github.com/estafette/estafette-ci-api/helpers"
	"github.com/go-kit/kit/metrics"
)

// NewMetricsService returns a new instance of a metrics Service.
func NewMetricsService(s Service, requestCount metrics.Counter, requestLatency metrics.Histogram) Service {
	return &metricsService{s, requestCount, requestLatency}
}

type metricsService struct {
	Service
	requestCount   metrics.Counter
	requestLatency metrics.Histogram
}

func (s *metricsService) CreateJobForGithubPush(ctx context.Context, event githubapi.PushEvent) {
	defer func(begin time.Time) {
		helpers.UpdateMetrics(s.requestCount, s.requestLatency, "CreateJobForGithubPush", begin)
	}(time.Now())

	s.Service.CreateJobForGithubPush(ctx, event)
}

func (s *metricsService) HasValidSignature(ctx context.Context, body []byte, signatureHeader string) (valid bool, err error) {
	defer func(begin time.Time) {
		helpers.UpdateMetrics(s.requestCount, s.requestLatency, "HasValidSignature", begin)
	}(time.Now())

	return s.Service.HasValidSignature(ctx, body, signatureHeader)
}

func (s *metricsService) Rename(ctx context.Context, fromRepoSource, fromRepoOwner, fromRepoName, toRepoSource, toRepoOwner, toRepoName string) (err error) {
	defer func(begin time.Time) { helpers.UpdateMetrics(s.requestCount, s.requestLatency, "Rename", begin) }(time.Now())

	return s.Service.Rename(ctx, fromRepoSource, fromRepoOwner, fromRepoName, toRepoSource, toRepoOwner, toRepoName)
}

func (s *metricsService) IsWhitelistedInstallation(ctx context.Context, installation githubapi.Installation) bool {
	defer func(begin time.Time) { helpers.UpdateMetrics(s.requestCount, s.requestLatency, "Rename", begin) }(time.Now())

	return s.Service.IsWhitelistedInstallation(ctx, installation)
}
