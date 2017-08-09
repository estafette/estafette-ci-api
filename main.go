package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"math/rand"
	"net/http"
	"os"
	"runtime"
	"time"

	"github.com/rs/zerolog"
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
	webhookTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "estafette_ci_api_webhook_totals",
			Help: "Total of received webhooks.",
		},
		[]string{"event", "source"},
	)
)

func init() {
	// Metrics have to be registered to be exposed:
	prometheus.MustRegister(webhookTotal)
}

func main() {

	// set some default fields added to all logs
	log := zerolog.New(os.Stdout).With().
		Timestamp().
		Str("app", "estafette-ci-api").
		Str("version", version).
		Logger()

	// log startup message
	log.Info().
		Str("branch", branch).
		Str("revision", revision).
		Str("buildDate", buildDate).
		Str("goVersion", goVersion).
		Msg("Starting estafette-ci-api...")

	// start prometheus
	go func() {
		log.Info().
			Str("port", *prometheusAddress).
			Str("path", *prometheusMetricsPath).
			Msg("Serving Prometheus metrics...")

		http.Handle(*prometheusMetricsPath, promhttp.Handler())

		if err := http.ListenAndServe(*prometheusAddress, nil); err != nil {
			log.Fatal().Err(err).Msg("Starting Prometheus listener failed")
		}
	}()

	log.Info().
		Str("port", *apiAddress).
		Msg("Serving api calls...")

	http.HandleFunc("/webhook/github", githubWebhookHandler)
	http.HandleFunc("/webhook/bitbucket", bitbucketWebhookHandler)
	http.HandleFunc("/liveness", livenessHandler)
	http.HandleFunc("/readiness", readinessHandler)

	if err := http.ListenAndServe(*apiAddress, nil); err != nil {
		log.Fatal().Err(err).Msg("Starting api listener failed")
	}
}

func githubWebhookHandler(w http.ResponseWriter, r *http.Request) {

	// https://developer.github.com/webhooks/
	eventType := r.Header.Get("X-GitHub-Event")
	webhookTotal.With(prometheus.Labels{"event": eventType, "source": "github"}).Inc()

	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		log.Error().Err(err).Msg("Reading body from Github webhook failed")
		http.Error(w, "Reading body from Github webhook failed", 500)
		return
	}

	// unmarshal json body
	var b interface{}
	err = json.Unmarshal(body, &b)
	if err != nil {
		log.Error().Err(err).Str("body", string(body)).Msg("Deserializing body from Github webhook failed")
		http.Error(w, "Deserializing body from Github webhook failed", 500)
		return
	}

	log.Info().
		Str("method", r.Method).
		Str("url", r.URL.String()).
		Interface("headers", r.Header).
		Interface("body", b).
		Msgf("Received webhook event '%v' from GitHub...", eventType)

	switch eventType {
	case "push":
		handleGithubPush(body)
	default:
		log.Warn().Str("event", eventType).Msgf("Unsupported Github webhook event '%v'", eventType)
	}

	fmt.Fprintf(w, "Aye aye!")
}

func bitbucketWebhookHandler(w http.ResponseWriter, r *http.Request) {

	// https://confluence.atlassian.com/bitbucket/manage-webhooks-735643732.html

	eventType := r.Header.Get("X-Event-Key")
	webhookTotal.With(prometheus.Labels{"event": eventType, "source": "bitbucket"}).Inc()

	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		log.Error().Err(err).Msg("Reading body from Bitbucket webhook failed")
		http.Error(w, "Reading body from Bitbucket webhook failed", 500)
		return
	}

	// unmarshal json body
	var b interface{}
	err = json.Unmarshal(body, &b)
	if err != nil {
		log.Error().Err(err).Str("body", string(body)).Msg("Deserializing body from Bitbucket webhook failed")
		http.Error(w, "Deserializing body from Bitbucket webhook failed", 500)
		return
	}

	log.Info().
		Str("method", r.Method).
		Str("url", r.URL.String()).
		Interface("headers", r.Header).
		Interface("body", string(body)).
		Msgf("Received webhook event '%v' from Bitbucket", eventType)

	switch eventType {
	case "repo:push":
		handleBitbucketPush(body)
	default:
		log.Warn().Str("event", eventType).Msgf("Unsupported Bitbucket webhook event '%v'", eventType)
	}

	fmt.Fprintf(w, "Aye aye!")
}

// GithubPushEvent represents a Github webhook push event
type GithubPushEvent struct {
	After      string           `json:"after"`
	Commits    []GithubCommit   `json:"commits"`
	HeadCommit GithubCommit     `json:"head_commit"`
	Pusher     GithubPusher     `json:"pusher"`
	Repository GithubRepository `json:"repository"`
}

// GithubCommit represents a Github commit
type GithubCommit struct {
	Author  GithubAuthor `json:"author"`
	Message string       `json:"message"`
	ID      string       `json:"id"`
}

// GithubAuthor represents a Github author
type GithubAuthor struct {
	Email    string `json:"email"`
	Name     string `json:"name"`
	UserName string `json:"username"`
}

// GithubPusher represents a Github pusher
type GithubPusher struct {
	Email string `json:"email"`
	Name  string `json:"name"`
}

// GithubRepository represents a Github repository
type GithubRepository struct {
	GitURL   string `json:"git_url"`
	HTMLURL  string `json:"html_url"`
	Name     string `json:"name"`
	FullName string `json:"full_name"`
}

func handleGithubPush(body []byte) {

	// unmarshal json body
	var pushEvent GithubPushEvent
	err := json.Unmarshal(body, &pushEvent)
	if err != nil {
		log.Error().Err(err).Str("body", string(body)).Msg("Deserializing body to GithubPushEvent failed")
		return
	}

	log.Info().Interface("pushEvent", pushEvent).Msg("Deserialized GitHub push event")
}

func handleBitbucketPush(body []byte) {

}

func livenessHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "I'm alive!")
}

func readinessHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "I'm ready!")
}
