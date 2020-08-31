package cloudsource

import (
	"context"

	"github.com/estafette/estafette-ci-api/clients/cloudsourceapi"
	contracts "github.com/estafette/estafette-ci-contracts"
)

type MockService struct {
	CreateJobForCloudSourcePushFunc func(ctx context.Context, notification cloudsourceapi.PubSubNotification) (err error)
	IsAllowedProjectFunc            func(notification cloudsourceapi.PubSubNotification) (isAllowed bool, organizations []*contracts.Organization)
}

func (s MockService) CreateJobForCloudSourcePush(ctx context.Context, notification cloudsourceapi.PubSubNotification) (err error) {
	if s.CreateJobForCloudSourcePushFunc == nil {
		return
	}
	return s.CreateJobForCloudSourcePushFunc(ctx, notification)
}

func (s MockService) IsAllowedProject(notification cloudsourceapi.PubSubNotification) (isAllowed bool, organizations []*contracts.Organization) {
	if s.IsAllowedProjectFunc == nil {
		return
	}
	return s.IsAllowedProjectFunc(notification)
}
