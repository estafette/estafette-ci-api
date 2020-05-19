package slackapi

import (
	"context"

	"github.com/estafette/estafette-ci-api/config"
	"github.com/estafette/estafette-ci-api/helpers"
	"github.com/opentracing/opentracing-go"
)

// NewTracingClient returns a new instance of a tracing Client.
func NewTracingClient(c Client) Client {
	return &tracingClient{c, "slackapi"}
}

type tracingClient struct {
	Client
	prefix string
}

func (c *tracingClient) GetUserProfile(ctx context.Context, userID string) (profile *UserProfile, err error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, helpers.GetSpanName(c.prefix, "GetUserProfile"))
	defer func() { helpers.FinishSpanWithError(span, err) }()

	return c.Client.GetUserProfile(ctx, userID)
}

func (c *tracingClient) RefreshConfig(config *config.APIConfig) {
	c.Client.RefreshConfig(config)
}
