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

func (c *tracingClient) GetUserProfile(ctx context.Context, userID string) (profile *UserProfile, err error) {

	span, ctx := opentracing.StartSpanFromContext(ctx, c.getSpanName("GetUserProfile"))
	defer span.Finish()
	defer func(span opentracing.Span) {
		c.handleError(span, err)
	}(span)

	return c.Client.GetUserProfile(ctx, userID)
}

func (c *tracingClient) getSpanName(funcName string) string {
	return "slackapi:" + funcName
}

func (c *tracingClient) handleError(span opentracing.Span, err error) {
	if err != nil {
		ext.Error.Set(span, true)
		span.LogFields(log.Error(err))
	}
}
