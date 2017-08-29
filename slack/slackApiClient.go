package slack

import (
	"github.com/prometheus/client_golang/prometheus"
)

// APIClient is the interface for communicating with Slack's api
type APIClient interface {
}

type apiClientImpl struct {
	slackAppClientID                string
	slackAppClientSecret            string
	slackAppOAuthAccessToken        string
	prometheusOutboundAPICallTotals *prometheus.CounterVec
}

// NewSlackAPIClient returns a new slack.APIClient
func NewSlackAPIClient(slackAppClientID, slackAppClientSecret, slackAppOAuthAccessToken string, prometheusOutboundAPICallTotals *prometheus.CounterVec) APIClient {
	return &apiClientImpl{
		slackAppClientID:                slackAppClientID,
		slackAppClientSecret:            slackAppClientSecret,
		slackAppOAuthAccessToken:        slackAppOAuthAccessToken,
		prometheusOutboundAPICallTotals: prometheusOutboundAPICallTotals,
	}
}
