package main

import (
	"fmt"
	"net/http"
	"os"
	"runtime"

	"github.com/alecthomas/kingpin"

	"github.com/rs/zerolog"

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
	// flags
	prometheusMetricsAddress   = kingpin.Flag("metrics-listen-address", "The address to listen on for Prometheus metrics requests.").Default(":9001").String()
	prometheusMetricsPath      = kingpin.Flag("metrics-path", "The path to listen for Prometheus metrics requests.").Default("/metrics").String()
	apiAddress                 = kingpin.Flag("api-listen-address", "The address to listen on for api HTTP requests.").Default(":5000").String()
	githubAppPrivateKeyPath    = kingpin.Flag("github-app-privatey-key-path", "The path to the pem file for the private key of the Github App.").Default("/github-app-key/private-key.pem").String()
	githubAppOAuthClientID     = kingpin.Flag("github-app-oauth-client-id", "The OAuth client id for the Github App.").Envar("GITHUB_APP_OAUTH_CLIENT_ID").String()
	githubAppOAuthClientSecret = kingpin.Flag("github-app-oauth-client-secret", "The OAuth client secret for the Github App.").Envar("GITHUB_APP_OAUTH_CLIENT_SECRET").String()

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

	// parse command line parameters
	kingpin.Parse()

	// log as severity for stackdriver logging to recognize the level
	zerolog.LevelFieldName = "severity"

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
		log.Debug().
			Str("port", *prometheusMetricsAddress).
			Str("path", *prometheusMetricsPath).
			Msg("Serving Prometheus metrics...")

		http.Handle(*prometheusMetricsPath, promhttp.Handler())

		if err := http.ListenAndServe(*prometheusMetricsAddress, nil); err != nil {
			log.Fatal().Err(err).Msg("Starting Prometheus listener failed")
		}
	}()

	log.Debug().
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

func livenessHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "I'm alive!")
}

func readinessHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "I'm ready!")
}
