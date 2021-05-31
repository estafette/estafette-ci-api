package api

import (
	"context"
	"sync"

	manifest "github.com/estafette/estafette-ci-manifest"
	"github.com/opentracing/opentracing-go"
	opentracinglog "github.com/opentracing/opentracing-go/log"
	"github.com/rs/zerolog/log"
)

// based on https://eli.thegreenplace.net/2020/pubsub-using-channels-in-go/

type EventTopicMessage struct {
	Ctx   context.Context
	Event manifest.EstafetteEvent
}

type EventTopic struct {
	name        string
	mu          sync.RWMutex
	subscribers map[string]chan EventTopicMessage
	closed      bool
}

func NewEventTopic(name string) *EventTopic {
	return &EventTopic{
		name:        name,
		subscribers: make(map[string]chan EventTopicMessage),
	}
}

func (t *EventTopic) Subscribe(name string) <-chan EventTopicMessage {
	t.mu.Lock()
	defer t.mu.Unlock()

	log.Info().Msgf("Subscribing %v to EventTopic %v", name, t.name)

	subscriber := make(chan EventTopicMessage, 1)

	t.subscribers[name] = subscriber

	return subscriber
}

func (t *EventTopic) Publish(publisher string, message EventTopicMessage) {

	span, ctx := opentracing.StartSpanFromContext(message.Ctx, GetSpanName("topics.EventTopic", "Publish"))
	message.Ctx = ctx
	defer func() { FinishSpan(span) }()

	t.mu.RLock()
	defer t.mu.RUnlock()

	span.LogFields(opentracinglog.String("event", "LockAcquired"))

	if t.closed {
		return
	}

	log.Debug().Msgf("Publishing message from %v to %v subscribers in EventTopic %v", publisher, len(t.subscribers), t.name)

	for subscriber, ch := range t.subscribers {
		log.Debug().Msgf("Publishing message from %v to subscriber %v in EventTopic %v", publisher, subscriber, t.name)
		go func(ch chan EventTopicMessage) {
			ch <- message
		}(ch)
	}
}

func (t *EventTopic) Close() {
	t.mu.Lock()
	defer t.mu.Unlock()

	log.Info().Msgf("Closing channels for %v subscribers in EventTopic %v", len(t.subscribers), t.name)

	if !t.closed {
		t.closed = true
		for subscriber, ch := range t.subscribers {
			log.Debug().Msgf("Closing channel for subscriber %v in EventTopic %v", subscriber, t.name)
			close(ch)
		}
	}
}
