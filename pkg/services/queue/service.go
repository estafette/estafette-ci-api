package queue

import (
	"context"
	"strings"

	"github.com/estafette/estafette-ci-api/pkg/api"
	"github.com/estafette/estafette-ci-api/pkg/services/estafette"
	manifest "github.com/estafette/estafette-ci-manifest"
	"github.com/nats-io/nats.go"
	"github.com/opentracing/opentracing-go"
	"github.com/rs/zerolog/log"
)

//go:generate mockgen -package=queue -destination ./mock.go -source=service.go
type Service interface {
	CreateConnection(ctx context.Context) (err error)
	CloseConnection(ctx context.Context)
	InitSubscriptions(ctx context.Context) (err error)
	ReceiveCronEvent(cronEvent *manifest.EstafetteCronEvent)
	ReceiveGitEvent(gitEvent *manifest.EstafetteGitEvent)
	ReceiveGithubEvent(githubEvent *manifest.EstafetteGithubEvent)
	ReceiveBitbucketEvent(bitbucketEvent *manifest.EstafetteBitbucketEvent)
	PublishGitEvent(ctx context.Context, gitEvent manifest.EstafetteGitEvent) (err error)
	PublishGithubEvent(ctx context.Context, githubEvent manifest.EstafetteGithubEvent) (err error)
	PublishBitbucketEvent(ctx context.Context, bitbucketEvent manifest.EstafetteBitbucketEvent) (err error)
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
	_, err = s.natsEncodedConnection.QueueSubscribe(s.config.Queue.SubjectCron, "estafette-ci-api", s.ReceiveCronEvent)
	if err != nil {
		return
	}

	_, err = s.natsEncodedConnection.QueueSubscribe(s.config.Queue.SubjectGit, "estafette-ci-api", s.ReceiveGitEvent)
	if err != nil {
		return
	}

	_, err = s.natsEncodedConnection.QueueSubscribe(s.config.Queue.SubjectGithub, "estafette-ci-api", s.ReceiveGithubEvent)
	if err != nil {
		return
	}

	_, err = s.natsEncodedConnection.QueueSubscribe(s.config.Queue.SubjectBitbucket, "estafette-ci-api", s.ReceiveBitbucketEvent)
	if err != nil {
		return
	}

	return nil
}

func (s *service) ReceiveCronEvent(cronEvent *manifest.EstafetteCronEvent) {
	var err error
	ctx := context.Background()
	span, ctx := opentracing.StartSpanFromContext(ctx, api.GetSpanName("queue", "ReceiveCronEvent"))
	defer func() { api.FinishSpanWithError(span, err) }()

	err = s.estafetteService.FireCronTriggers(ctx, *cronEvent)
	if err != nil {
		log.Error().Err(err).Msgf("Failed handling cron event from queue")
	}
}

func (s *service) ReceiveGitEvent(gitEvent *manifest.EstafetteGitEvent) {
	var err error
	ctx := context.Background()
	span, ctx := opentracing.StartSpanFromContext(ctx, api.GetSpanName("queue", "ReceiveGitEvent"))
	defer func() { api.FinishSpanWithError(span, err) }()

	err = s.estafetteService.FireGitTriggers(ctx, *gitEvent)
	if err != nil {
		log.Error().Err(err).Msgf("Failed handling git event from queue")
	}
}

func (s *service) ReceiveGithubEvent(githubEvent *manifest.EstafetteGithubEvent) {
	var err error
	ctx := context.Background()
	span, ctx := opentracing.StartSpanFromContext(ctx, api.GetSpanName("queue", "ReceiveGithubEvent"))
	defer func() { api.FinishSpanWithError(span, err) }()

	err = s.estafetteService.FireGithubTriggers(ctx, *githubEvent)
	if err != nil {
		log.Error().Err(err).Msgf("Failed handling github event from queue")
	}
}

func (s *service) ReceiveBitbucketEvent(bitbucketEvent *manifest.EstafetteBitbucketEvent) {
	var err error
	ctx := context.Background()
	span, ctx := opentracing.StartSpanFromContext(ctx, api.GetSpanName("queue", "ReceiveBitbucketEvent"))
	defer func() { api.FinishSpanWithError(span, err) }()

	err = s.estafetteService.FireBitbucketTriggers(ctx, *bitbucketEvent)
	if err != nil {
		log.Error().Err(err).Msgf("Failed handling bitbucket event from queue")
	}
}

func (s *service) PublishGitEvent(ctx context.Context, gitEvent manifest.EstafetteGitEvent) (err error) {
	span, _ := opentracing.StartSpanFromContext(ctx, api.GetSpanName("queue", "PublishGitEvent"))
	defer func() { api.FinishSpanWithError(span, err) }()

	return s.natsEncodedConnection.Publish(s.config.Queue.SubjectGit, &gitEvent)
}

func (s *service) PublishGithubEvent(ctx context.Context, githubEvent manifest.EstafetteGithubEvent) (err error) {
	span, _ := opentracing.StartSpanFromContext(ctx, api.GetSpanName("queue", "PublishGithubEvent"))
	defer func() { api.FinishSpanWithError(span, err) }()

	return s.natsEncodedConnection.Publish(s.config.Queue.SubjectGithub, &githubEvent)
}

func (s *service) PublishBitbucketEvent(ctx context.Context, bitbucketEvent manifest.EstafetteBitbucketEvent) (err error) {
	span, _ := opentracing.StartSpanFromContext(ctx, api.GetSpanName("queue", "PublishBitbucketEvent"))
	defer func() { api.FinishSpanWithError(span, err) }()

	return s.natsEncodedConnection.Publish(s.config.Queue.SubjectBitbucket, &bitbucketEvent)
}
