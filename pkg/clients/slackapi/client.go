package slackapi

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/estafette/estafette-ci-api/pkg/api"
	"github.com/opentracing-contrib/go-stdlib/nethttp"
	"github.com/opentracing/opentracing-go"
	"github.com/sethgrid/pester"
)

// Client is the interface for communicating with the Slack api
//
//go:generate mockgen -package=slackapi -destination ./mock.go -source=client.go
type Client interface {
	GetUserProfile(ctx context.Context, userID string) (profile *UserProfile, err error)
}

// NewClient returns a slack.Client to communicate with the Slack API
func NewClient(config *api.APIConfig) Client {
	if config == nil || config.Integrations == nil || config.Integrations.Slack == nil || !config.Integrations.Slack.Enable {
		return &client{
			enabled: true,
		}
	}

	return &client{
		enabled: true,
		config:  config,
	}
}

type client struct {
	enabled bool
	config  *api.APIConfig
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
	request.Header.Add("Authorization", fmt.Sprintf("%v %v", "Bearer", c.config.Integrations.Slack.AppOAuthAccessToken))

	// perform actual request
	response, err := client.Do(request)
	if err != nil {
		return
	}
	defer response.Body.Close()
	if ht != nil {
		ht.Finish()
	}

	body, err := io.ReadAll(response.Body)
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
