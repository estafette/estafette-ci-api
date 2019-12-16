package slackapi

import (
	"context"

	"github.com/opentracing/opentracing-go"
	"github.com/opentracing/opentracing-go/ext"
	"github.com/opentracing/opentracing-go/log"
)

// NewTracingClient returns a new instance of a tracing Client.
func NewTracingClient(c Client) Client {
	return &tracingClient{c}
}

type tracingClient struct {
	Client
}

func (c *tracingClient) GetUserProfile(ctx context.Context, userID string) (*UserProfile, error) {

	span, ctx := opentracing.StartSpanFromContext(ctx, c.getSpanName("GetUserProfile"))
	defer span.Finish()

	profile, err := c.Client.GetUserProfile(ctx, userID)
	c.handleError(span, err)

	return profile, err
}

func (c *tracingClient) getSpanName(funcName string) string {
	return "slackapi:" + funcName
}

func (c *tracingClient) handleError(span opentracing.Span, err error) error {
	if err != nil {
		ext.Error.Set(span, true)
		span.LogFields(log.Error(err))
	}
	return err
}
