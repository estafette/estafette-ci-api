package slackapi

import (
	"context"
)

type MockClient struct {
	GetUserProfileFunc func(ctx context.Context, userID string) (profile *UserProfile, err error)
}

func (c MockClient) GetUserProfile(ctx context.Context, userID string) (profile *UserProfile, err error) {
	if c.GetUserProfileFunc == nil {
		return
	}
	return c.GetUserProfileFunc(ctx, userID)
}
