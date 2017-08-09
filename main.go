package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"math/rand"
	"net/http"
	"runtime"
	"time"

	"github.com/rs/zerolog/log"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	version   string
	branch    string
	revision  string
	buildDate string
	goVersion = runtime.Version()
)

var (
	prometheusAddress     = flag.String("metrics-listen-address", ":9001", "The address to listen on for Prometheus metrics requests.")
	prometheusMetricsPath = flag.String("metrics-path", "/metrics", "The path to listen for Prometheus metrics requests.")
	apiAddress            = flag.String("api-listen-address", ":5000", "The address to listen on for api HTTP requests.")

	// seed random number
	r = rand.New(rand.NewSource(time.Now().UnixNano()))

	// define prometheus counter
	httpRequestTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "http_requests_total",
			Help: "Number of handled http requests.",
		},
		[]string{"code", "handler"},
	)
)

func init() {
	// Metrics have to be registered to be exposed:
	prometheus.MustRegister(httpRequestTotal)
}

func main() {

	log.Info().
		Str("version", version).
		Str("branch", branch).
		Str("revision", revision).
		Str("buildDate", buildDate).
		Str("goVersion", goVersion).
		Msg("Starting estafette-ci-api")

	// start prometheus
	go func() {
		log.Info().
			Str("port", *prometheusAddress).
			Str("path", *prometheusMetricsPath).
			Msg("Serving Prometheus metrics")

		http.Handle(*prometheusMetricsPath, promhttp.Handler())

		if err := http.ListenAndServe(*prometheusAddress, nil); err != nil {
			log.Fatal().Err(err).Msg("Prometheus listener failed")
		}
	}()

	log.Info().
		Str("port", *apiAddress).
		Msg("Serving api calls")

	http.HandleFunc("/webhook/github", githubWebhookHandler)
	http.HandleFunc("/webhook/bitbucket", bitbucketWebhookHandler)
	http.HandleFunc("/liveness", livenessHandler)
	http.HandleFunc("/readiness", readinessHandler)

	if err := http.ListenAndServe(*apiAddress, nil); err != nil {
		log.Fatal().Err(err).Msg("Api listener failed")
	}
}

func githubWebhookHandler(w http.ResponseWriter, r *http.Request) {

	// https://developer.github.com/webhooks/

	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		log.Error().Err(err).Msg("Reading body from Github webhook failed")
	}

	log.Info().
		Interface("url", r.URL).
		Interface("headers", r.Header).
		Interface("body", body).
		Msg("Received webhook from GitHub...")

	fmt.Fprintf(w, "Aye aye!")
}

func bitbucketWebhookHandler(w http.ResponseWriter, r *http.Request) {

	// https://confluence.atlassian.com/bitbucket/manage-webhooks-735643732.html

	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		log.Error().Err(err).Msg("Reading body from Bitbucket webhook failed")
	}

	log.Info().
		Interface("url", r.URL).
		Interface("headers", r.Header).
		Interface("body", body).
		Msg("Received webhook from Bitbucket")

	fmt.Fprintf(w, "Aye aye!")
}

func livenessHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "I'm alive!")
}

func readinessHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "I'm ready!")
}
