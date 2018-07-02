package slack

import (
	"github.com/estafette/estafette-ci-api/config"
	"github.com/prometheus/client_golang/prometheus"
)

// APIClient is the interface for communicating with Slack's api
type APIClient interface {
}

type apiClientImpl struct {
	config                          config.SlackConfig
	prometheusOutboundAPICallTotals *prometheus.CounterVec
}

// NewSlackAPIClient returns a new slack.APIClient
func NewSlackAPIClient(config config.SlackConfig, prometheusOutboundAPICallTotals *prometheus.CounterVec) APIClient {
	return &apiClientImpl{
		config: config,
		prometheusOutboundAPICallTotals: prometheusOutboundAPICallTotals,
	}
}
