package slackapi

import (
	"context"

	"github.com/estafette/estafette-ci-api/pkg/api"
)

// NewLoggingClient returns a new instance of a logging Client.
func NewLoggingClient(c Client) Client {
	return &loggingClient{c, "slackapi"}
}

type loggingClient struct {
	Client Client
	prefix string
}

func (c *loggingClient) GetUserProfile(ctx context.Context, userID string) (profile *UserProfile, err error) {
	defer func() { api.HandleLogError(c.prefix, "Client", "GetUserProfile", err) }()

	return c.Client.GetUserProfile(ctx, userID)
}
