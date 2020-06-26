package topics

import (
	"context"
	"sync"

	"github.com/estafette/estafette-ci-api/helpers"
	manifest "github.com/estafette/estafette-ci-manifest"
	"github.com/opentracing/opentracing-go"
	opentracinglog "github.com/opentracing/opentracing-go/log"
	"github.com/rs/zerolog/log"
)

// based on https://eli.thegreenplace.net/2020/pubsub-using-channels-in-go/

type GitEventTopicMessage struct {
	Ctx   context.Context
	Event manifest.EstafetteGitEvent
}

type GitEventTopic struct {
	name        string
	mu          sync.RWMutex
	subscribers map[string]chan GitEventTopicMessage
	closed      bool
}

func NewGitEventTopic(name string) *GitEventTopic {
	return &GitEventTopic{
		name:        name,
		subscribers: make(map[string]chan GitEventTopicMessage, 0),
	}
}

func (t *GitEventTopic) Subscribe(name string) <-chan GitEventTopicMessage {
	t.mu.Lock()
	defer t.mu.Unlock()

	log.Info().Msgf("Subscribing %v to GitEventTopic %v", name, t.name)

	subscriber := make(chan GitEventTopicMessage, 1)

	t.subscribers[name] = subscriber

	return subscriber
}

func (t *GitEventTopic) Publish(publisher string, message GitEventTopicMessage) {

	span, ctx := opentracing.StartSpanFromContext(message.Ctx, helpers.GetSpanName("topics.GitEventTopic", "Publish"))
	message.Ctx = ctx
	defer func() { helpers.FinishSpan(span) }()

	t.mu.RLock()
	defer t.mu.RUnlock()

	span.LogFields(opentracinglog.String("event", "LockAcquired"))

	if t.closed {
		return
	}

	log.Debug().Msgf("Publishing message from %v to %v subscribers in GitEventTopic %v", publisher, len(t.subscribers), t.name)

	for subscriber, ch := range t.subscribers {
		log.Debug().Msgf("Publishing message from %v to subscriber %v in GitEventTopic %v", publisher, subscriber, t.name)
		go func(ch chan GitEventTopicMessage) {
			ch <- message
		}(ch)
	}
}

func (t *GitEventTopic) Close() {
	t.mu.Lock()
	defer t.mu.Unlock()

	log.Info().Msgf("Closing channels for %v subscribers in GitEventTopic %v", len(t.subscribers), t.name)

	if !t.closed {
		t.closed = true
		for subscriber, ch := range t.subscribers {
			log.Debug().Msgf("Closing channel for subscriber %v in GitEventTopic %v", subscriber, t.name)
			close(ch)
		}
	}
}
