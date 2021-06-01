package api

import (
	"context"
	"sync"

	contracts "github.com/estafette/estafette-ci-contracts"
	"github.com/opentracing/opentracing-go"
	opentracinglog "github.com/opentracing/opentracing-go/log"
	"github.com/rs/zerolog/log"
)

// based on https://eli.thegreenplace.net/2020/pubsub-using-channels-in-go/

type BuildTopicMessage struct {
	Ctx   context.Context
	Event contracts.Build
}

type BuildTopic struct {
	name        string
	mu          sync.RWMutex
	subscribers map[string]chan BuildTopicMessage
	closed      bool
}

func NewBuildTopic(name string) *BuildTopic {
	return &BuildTopic{
		name:        name,
		subscribers: make(map[string]chan BuildTopicMessage, 0),
	}
}

func (t *BuildTopic) Subscribe(name string) <-chan BuildTopicMessage {
	t.mu.Lock()
	defer t.mu.Unlock()

	log.Info().Msgf("Subscribing %v to BuildTopic %v", name, t.name)

	subscriber := make(chan BuildTopicMessage, 1)

	t.subscribers[name] = subscriber

	return subscriber
}

func (t *BuildTopic) Publish(publisher string, message BuildTopicMessage) {

	span, ctx := opentracing.StartSpanFromContext(message.Ctx, GetSpanName("topics.EventTopic", "Publish"))
	message.Ctx = ctx
	defer func() { FinishSpan(span) }()

	t.mu.RLock()
	defer t.mu.RUnlock()

	span.LogFields(opentracinglog.String("event", "LockAcquired"))

	if t.closed {
		return
	}

	log.Debug().Msgf("Publishing message from %v to %v subscribers in BuildTopic %v", publisher, len(t.subscribers), t.name)

	for subscriber, ch := range t.subscribers {
		log.Debug().Msgf("Publishing message from %v  to subscriber %v in BuildTopic %v", publisher, subscriber, t.name)
		go func(ch chan BuildTopicMessage) {
			ch <- message
		}(ch)
	}
}

func (t *BuildTopic) Close() {
	t.mu.Lock()
	defer t.mu.Unlock()

	log.Info().Msgf("Closing channels for %v subscribers in BuildTopic %v", len(t.subscribers), t.name)

	if !t.closed {
		t.closed = true
		for subscriber, ch := range t.subscribers {
			log.Debug().Msgf("Closing channel for subscriber %v in BuildTopic %v", subscriber, t.name)
			close(ch)
		}
	}
}
