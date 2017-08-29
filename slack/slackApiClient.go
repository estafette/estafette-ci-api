package slack

import (
	"github.com/prometheus/client_golang/prometheus"
)

// APIClient is the interface for communicating with Slack's api
type APIClient interface {
}

type apiClientImpl struct {
	slackAppOAuthAccessToken        string
	prometheusOutboundAPICallTotals *prometheus.CounterVec
}

// NewSlackAPIClient returns a new slack.APIClient
func NewSlackAPIClient(slackAppOAuthAccessToken string, prometheusOutboundAPICallTotals *prometheus.CounterVec) APIClient {
	return &apiClientImpl{
		slackAppOAuthAccessToken:        slackAppOAuthAccessToken,
		prometheusOutboundAPICallTotals: prometheusOutboundAPICallTotals,
	}
}
