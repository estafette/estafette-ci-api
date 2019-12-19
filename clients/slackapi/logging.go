package slackapi

import (
	"context"

	"github.com/estafette/estafette-ci-api/helpers"
)

// NewLoggingClient returns a new instance of a logging Client.
func NewLoggingClient(c Client) Client {
	return &tracingClient{c, "slackapi"}
}

type loggingClient struct {
	Client
	prefix string
}

func (c *loggingClient) GetUserProfile(ctx context.Context, userID string) (profile *UserProfile, err error) {
	defer func() { helpers.HandleLogError(c.prefix, "GetUserProfile", err) }()

	return c.Client.GetUserProfile(ctx, userID)
}
