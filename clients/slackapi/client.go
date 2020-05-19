package slackapi

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"time"

	"github.com/estafette/estafette-ci-api/config"
	"github.com/opentracing-contrib/go-stdlib/nethttp"
	"github.com/opentracing/opentracing-go"
	"github.com/rs/zerolog/log"
	"github.com/sethgrid/pester"
)

// Client is the interface for communicating with the Slack api
type Client interface {
	GetUserProfile(ctx context.Context, userID string) (profile *UserProfile, err error)
	RefreshConfig(config *config.APIConfig)
}

// NewClient returns a slack.Client to communicate with the Slack API
func NewClient(config config.SlackConfig) Client {
	return &client{
		config: config,
	}
}

type client struct {
	config config.SlackConfig
}

// GetUserProfile returns a Slack user profile
func (c *client) GetUserProfile(ctx context.Context, userID string) (profile *UserProfile, err error) {

	url := fmt.Sprintf("https://slack.com/api/users.profile.get?user=%v", userID)

	// create client, in order to add headers
	client := pester.NewExtendedClient(&http.Client{Transport: &nethttp.Transport{}})
	client.MaxRetries = 3
	client.Backoff = pester.ExponentialJitterBackoff
	client.KeepLog = true
	client.Timeout = time.Second * 10
	request, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return
	}

	span := opentracing.SpanFromContext(ctx)
	var ht *nethttp.Tracer
	if span != nil {
		// add tracing context
		request = request.WithContext(opentracing.ContextWithSpan(request.Context(), span))

		// collect additional information on setting up connections
		request, ht = nethttp.TraceRequest(span.Tracer(), request)
	}

	// add headers
	request.Header.Add("Authorization", fmt.Sprintf("%v %v", "Bearer", c.config.AppOAuthAccessToken))

	// perform actual request
	response, err := client.Do(request)
	if err != nil {
		return
	}
	defer response.Body.Close()
	if ht != nil {
		ht.Finish()
	}

	body, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return
	}

	var profileResponse GetUserProfileResponse

	// unmarshal json body
	err = json.Unmarshal(body, &profileResponse)
	if err != nil {
		return
	}

	return profileResponse.Profile, nil
}

func (c *client) RefreshConfig(config *config.APIConfig) {
	log.Debug().Msg("Refreshing config in slackapi.Client")
	c.config = *config.Integrations.Slack
}
