package cloudsource

import (
	"context"

	"github.com/estafette/estafette-ci-api/api"
	"github.com/estafette/estafette-ci-api/clients/cloudsourceapi"
)

// NewLoggingService returns a new instance of a logging Service.
func NewLoggingService(s Service) Service {
	return &loggingService{s, "cloudsource"}
}

type loggingService struct {
	Service
	prefix string
}

func (s *loggingService) CreateJobForCloudSourcePush(ctx context.Context, notification cloudsourceapi.PubSubNotification) (err error) {
	defer func() {
		api.HandleLogError(s.prefix, "CreateJobForCloudSourcePush", err, ErrNonCloneableEvent, ErrNoManifest)
	}()

	return s.Service.CreateJobForCloudSourcePush(ctx, notification)
}
