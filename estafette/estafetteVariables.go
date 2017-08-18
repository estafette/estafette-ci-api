package estafette

import "github.com/prometheus/client_golang/prometheus"

var (
	// OutgoingAPIRequestTotal is the prometheus timeline serie that keeps track of outbound api calls
	OutgoingAPIRequestTotal *prometheus.CounterVec
	// WebhookTotal is the prometheus timeline serie that keeps track of inbound events
	WebhookTotal *prometheus.CounterVec
	// channel for passing push events to worker that cleans up finished jobs
	estafetteCiBuilderEvents = make(chan CiBuilderEvent, 100)
)
