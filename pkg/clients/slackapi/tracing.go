package slackapi

import (
	"context"

	"github.com/estafette/estafette-ci-api/pkg/api"
	"github.com/opentracing/opentracing-go"
)

// NewTracingClient returns a new instance of a tracing Client.
func NewTracingClient(c Client) Client {
	return &tracingClient{c, "slackapi"}
}

type tracingClient struct {
	Client Client
	prefix string
}

func (c *tracingClient) GetUserProfile(ctx context.Context, userID string) (profile *UserProfile, err error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, api.GetSpanName(c.prefix, "GetUserProfile"))
	defer func() { api.FinishSpanWithError(span, err) }()

	return c.Client.GetUserProfile(ctx, userID)
}
