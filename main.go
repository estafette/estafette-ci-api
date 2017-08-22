package main

import (
	"context"
	"fmt"
	stdlog "log"
	"net/http"
	"os"
	"os/signal"
	"runtime"
	"sync"
	"syscall"
	"time"

	"github.com/alecthomas/kingpin"
	"github.com/estafette/estafette-ci-api/bitbucket"
	"github.com/estafette/estafette-ci-api/estafette"
	"github.com/estafette/estafette-ci-api/github"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
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
	prometheusMetricsAddress = kingpin.Flag("metrics-listen-address", "The address to listen on for Prometheus metrics requests.").Default(":9001").String()
	prometheusMetricsPath    = kingpin.Flag("metrics-path", "The path to listen for Prometheus metrics requests.").Default("/metrics").String()

	apiAddress = kingpin.Flag("api-listen-address", "The address to listen on for api HTTP requests.").Default(":5000").String()

	githubAppPrivateKeyPath    = kingpin.Flag("github-app-privatey-key-path", "The path to the pem file for the private key of the Github App.").Default("/github-app-key/private-key.pem").String()
	githubAppID                = kingpin.Flag("github-app-id", "The Github App id.").Envar("GITHUB_APP_ID").String()
	githubAppOAuthClientID     = kingpin.Flag("github-app-oauth-client-id", "The OAuth client id for the Github App.").Envar("GITHUB_APP_OAUTH_CLIENT_ID").String()
	githubAppOAuthClientSecret = kingpin.Flag("github-app-oauth-client-secret", "The OAuth client secret for the Github App.").Envar("GITHUB_APP_OAUTH_CLIENT_SECRET").String()

	bitbucketAPIKey         = kingpin.Flag("bitbucket-api-key", "The api key for Bitbucket.").Envar("BITBUCKET_API_KEY").String()
	bitbucketAppOAuthKey    = kingpin.Flag("bitbucket-app-oauth-key", "The OAuth key for the Bitbucket App.").Envar("BITBUCKET_APP_OAUTH_KEY").String()
	bitbucketAppOAuthSecret = kingpin.Flag("bitbucket-app-oauth-secret", "The OAuth secret for the Bitbucket App.").Envar("BITBUCKET_APP_OAUTH_SECRET").String()

	estafetteCiServerBaseURL = kingpin.Flag("estafette-ci-server-base-url", "The base url of this api server.").Envar("ESTAFETTE_CI_SERVER_BASE_URL").String()
	estafetteCiAPIKey        = kingpin.Flag("estafette-ci-api-key", "An api key for estafette itself to use until real oauth is supported.").Envar("ESTAFETTE_CI_API_KEY").String()

	// prometheusInboundEventTotals is the prometheus timeline serie that keeps track of inbound events
	prometheusInboundEventTotals = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "estafette_ci_api_inbound_event_totals",
			Help: "Total of inbound events.",
		},
		[]string{"event", "source"},
	)

	// prometheusOutboundAPICallTotals is the prometheus timeline serie that keeps track of outbound api calls
	prometheusOutboundAPICallTotals = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "estafette_ci_api_outbound_api_call_totals",
			Help: "Total of outgoing api calls.",
		},
		[]string{"target"},
	)
)

func init() {
	// Metrics have to be registered to be exposed:
	prometheus.MustRegister(prometheusInboundEventTotals)
	prometheus.MustRegister(prometheusOutboundAPICallTotals)
}

func main() {

	// parse command line parameters
	kingpin.Parse()

	// log as severity for stackdriver logging to recognize the level
	zerolog.LevelFieldName = "severity"

	// set some default fields added to all logs
	log.Logger = zerolog.New(os.Stdout).With().
		Timestamp().
		Str("app", "estafette-ci-api").
		Str("version", version).
		Logger()

	// use zerolog for any logs sent via standard log library
	stdlog.SetFlags(0)
	stdlog.SetOutput(log.Logger)

	// log startup message
	log.Info().
		Str("branch", branch).
		Str("revision", revision).
		Str("buildDate", buildDate).
		Str("goVersion", goVersion).
		Msg("Starting estafette-ci-api...")

	// define channel and wait group to gracefully shutdown the application
	stopChan := make(chan os.Signal)
	signal.Notify(stopChan, syscall.SIGTERM, syscall.SIGINT)
	waitGroup := &sync.WaitGroup{}

	// start prometheus
	go startPrometheus()

	githubAPIClient := github.NewGithubAPIClient(*githubAppPrivateKeyPath, *githubAppID, *githubAppOAuthClientID, *githubAppOAuthClientSecret, prometheusOutboundAPICallTotals)
	bitbucketAPIClient := bitbucket.NewBitbucketAPIClient(*bitbucketAPIKey, *bitbucketAppOAuthKey, *bitbucketAppOAuthSecret, prometheusOutboundAPICallTotals)
	ciBuilderClient, err := estafette.NewCiBuilderClient(*estafetteCiServerBaseURL, *estafetteCiAPIKey, prometheusOutboundAPICallTotals)
	if err != nil {
		log.Fatal().Err(err).Msg("Creating new CiBuilderClient has failed")
	}

	// channel for passing push events to handler that creates ci-builder job
	githubPushEvents := make(chan github.PushEvent, 100)
	// channel for passing push events to handler that creates ci-builder job
	bitbucketPushEvents := make(chan bitbucket.RepositoryPushEvent, 100)
	// channel for passing push events to worker that cleans up finished jobs
	estafetteCiBuilderEvents := make(chan estafette.CiBuilderEvent, 100)

	// listen to channels for push events
	githubEventWorker := github.NewGithubEventWorker(waitGroup, githubAPIClient, ciBuilderClient, githubPushEvents)
	githubEventWorker.ListenToEventChannels()

	bitbucketEventWorker := bitbucket.NewBitbucketEventWorker(waitGroup, bitbucketAPIClient, ciBuilderClient, bitbucketPushEvents)
	bitbucketEventWorker.ListenToEventChannels()

	estafetteEventWorker := estafette.NewEstafetteEventWorker(waitGroup, ciBuilderClient, estafetteCiBuilderEvents)
	estafetteEventWorker.ListenToEventChannels()

	// listen to http calls
	log.Debug().
		Str("port", *apiAddress).
		Msg("Serving api calls...")

	srv := &http.Server{Addr: *apiAddress}

	githubEventHandler := github.NewGithubEventHandler(githubPushEvents, prometheusInboundEventTotals)
	http.HandleFunc("/events/github", githubEventHandler.Handle)

	bitbucketEventHandler := bitbucket.NewBitbucketEventHandler(bitbucketPushEvents, prometheusInboundEventTotals)
	http.HandleFunc("/events/bitbucket", bitbucketEventHandler.Handle)

	estafetteEventHandler := estafette.NewEstafetteEventHandler(*estafetteCiAPIKey, estafetteCiBuilderEvents, prometheusInboundEventTotals)
	http.HandleFunc("/events/estafette/ci-builder", estafetteEventHandler.Handle)

	http.HandleFunc("/liveness", livenessHandler)
	http.HandleFunc("/readiness", readinessHandler)

	go func() {
		if err := srv.ListenAndServe(); err != nil {
			log.Fatal().Err(err).Msg("Starting api listener failed")
		}
	}()

	// wait for graceful shutdown to finish
	<-stopChan // wait for SIGINT
	log.Debug().Msg("Shutting down server...")

	// shut down gracefully
	ctx, _ := context.WithTimeout(context.Background(), 30*time.Second)
	srv.Shutdown(ctx)

	githubEventWorker.Stop()
	bitbucketEventWorker.Stop()
	estafetteEventWorker.Stop()

	log.Debug().Msg("Awaiting waitgroup...")
	waitGroup.Wait()

	log.Info().Msg("Server gracefully stopped")
}

func startPrometheus() {
	log.Debug().
		Str("port", *prometheusMetricsAddress).
		Str("path", *prometheusMetricsPath).
		Msg("Serving Prometheus metrics...")

	http.Handle(*prometheusMetricsPath, promhttp.Handler())

	if err := http.ListenAndServe(*prometheusMetricsAddress, nil); err != nil {
		log.Fatal().Err(err).Msg("Starting Prometheus listener failed")
	}
}

func livenessHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "I'm alive!")
}

func readinessHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "I'm ready!")
}
