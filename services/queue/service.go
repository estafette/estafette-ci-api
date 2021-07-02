package queue

import (
	"context"
	"strings"

	"github.com/estafette/estafette-ci-api/api"
	"github.com/estafette/estafette-ci-api/services/estafette"
	manifest "github.com/estafette/estafette-ci-manifest"
	"github.com/nats-io/nats.go"
	"github.com/opentracing/opentracing-go"
	"github.com/rs/zerolog/log"
)

type Service interface {
	CreateConnection(ctx context.Context) (err error)
	CloseConnection(ctx context.Context)
	InitSubscriptions(ctx context.Context) (err error)
	InitCronSubscription(ctx context.Context) (err error)
	HandleCronEvent(cronEvent *manifest.EstafetteCronEvent)
}

// NewService returns a new estafette.Service
func NewService(config *api.APIConfig, estafetteService estafette.Service) Service {

	return &service{
		config:           config,
		estafetteService: estafetteService,
	}
}

type service struct {
	config                *api.APIConfig
	estafetteService      estafette.Service
	natsConnection        *nats.Conn
	natsEncodedConnection *nats.EncodedConn
}

func (s *service) CreateConnection(ctx context.Context) (err error) {
	s.natsConnection, err = nats.Connect(strings.Join(s.config.Queue.Hosts, ","))
	if err != nil {
		return
	}

	s.natsEncodedConnection, err = nats.NewEncodedConn(s.natsConnection, nats.JSON_ENCODER)
	if err != nil {
		return
	}

	return nil
}

func (s *service) CloseConnection(ctx context.Context) {
	if s.natsEncodedConnection != nil {
		s.natsEncodedConnection.Close()
	}
	if s.natsConnection != nil {
		s.natsConnection.Close()
	}
}

func (s *service) InitSubscriptions(ctx context.Context) (err error) {

	err = s.InitCronSubscription(ctx)
	if err != nil {
		return
	}

	return nil
}

func (s *service) InitCronSubscription(ctx context.Context) (err error) {
	_, err = s.natsEncodedConnection.QueueSubscribe("cron", "estafette-ci-api", s.HandleCronEvent)
	if err != nil {
		return
	}

	return nil
}

func (s *service) HandleCronEvent(cronEvent *manifest.EstafetteCronEvent) {

	var err error
	ctx := context.Background()
	span, ctx := opentracing.StartSpanFromContext(ctx, api.GetSpanName("queue", "HandleCronEvent"))
	defer func() { api.FinishSpanWithError(span, err) }()

	err = s.estafetteService.FireCronTriggers(ctx, *cronEvent)
	if err != nil {
		log.Error().Err(err).Msgf("Failed handling cron event from queu")
	}
}
