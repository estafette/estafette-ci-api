package slackapi

import (
	"context"

	"github.com/opentracing/opentracing-go"
)

type tracingClient struct {
	Client
}

// NewTracingClient returns a new instance of a tracing Client.
func NewTracingClient(c Client) Client {
	return &tracingClient{c}
}

func (s *tracingClient) GetUserProfile(ctx context.Context, userID string) (*UserProfile, error) {

	span, ctx := opentracing.StartSpanFromContext(ctx, s.getSpanName("GetUserProfile"))
	defer span.Finish()

	return s.Client.GetUserProfile(ctx, userID)
}

func (s *tracingClient) getSpanName(funcName string) string {
	return "slackapi:" + funcName
}
