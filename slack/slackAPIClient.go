package slack

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"time"

	"github.com/estafette/estafette-ci-api/config"
	slcontracts "github.com/estafette/estafette-ci-api/slack/contracts"
	"github.com/sethgrid/pester"

	"github.com/prometheus/client_golang/prometheus"
)

// APIClient is the interface for communicating with the Slack api
type APIClient interface {
	GetUserProfile(string) (*slcontracts.UserProfile, error)
}

type apiClientImpl struct {
	config                          config.SlackConfig
	prometheusOutboundAPICallTotals *prometheus.CounterVec
}

// NewSlackAPIClient creates an slack.APIClient to communicate with the Slack api
func NewSlackAPIClient(config config.SlackConfig, prometheusOutboundAPICallTotals *prometheus.CounterVec) APIClient {
	return &apiClientImpl{
		config:                          config,
		prometheusOutboundAPICallTotals: prometheusOutboundAPICallTotals,
	}
}

// GetUserProfile returns a Slack user profile
func (sl *apiClientImpl) GetUserProfile(userID string) (profile *slcontracts.UserProfile, err error) {

	// track call via prometheus
	sl.prometheusOutboundAPICallTotals.With(prometheus.Labels{"target": "slack"}).Inc()

	url := fmt.Sprintf("https://slack.com/api/users.profile.get?user=%v", userID)

	// create client, in order to add headers
	client := pester.New()
	client.MaxRetries = 3
	client.Backoff = pester.ExponentialJitterBackoff
	client.KeepLog = true
	client.Timeout = time.Second * 10
	request, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return
	}

	// add headers
	request.Header.Add("Authorization", fmt.Sprintf("%v %v", "Bearer", sl.config.AppOAuthAccessToken))

	// perform actual request
	response, err := client.Do(request)
	if err != nil {
		return
	}
	defer response.Body.Close()

	body, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return
	}

	var profileResponse slcontracts.GetUserProfileResponse

	// unmarshal json body
	err = json.Unmarshal(body, &profileResponse)
	if err != nil {
		return
	}

	return profileResponse.Profile, nil
}
