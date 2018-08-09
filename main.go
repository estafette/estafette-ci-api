package main

import (
	"context"
	stdlog "log"
	"net/http"
	"os"
	"os/signal"
	"runtime"
	"sync"
	"syscall"
	"time"

	"github.com/estafette/estafette-ci-crypt"

	"github.com/alecthomas/kingpin"
	"github.com/estafette/estafette-ci-api/bitbucket"
	"github.com/estafette/estafette-ci-api/cockroach"
	"github.com/estafette/estafette-ci-api/config"
	"github.com/estafette/estafette-ci-api/estafette"
	"github.com/estafette/estafette-ci-api/github"
	"github.com/estafette/estafette-ci-api/slack"
	"github.com/gin-contrib/gzip"
	"github.com/gin-gonic/gin"
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
	apiAddress               = kingpin.Flag("api-listen-address", "The address to listen on for api HTTP requests.").Default(":5000").String()
	configFilePath           = kingpin.Flag("config-file-path", "The path to yaml config file configuring this application.").Default("/config/config.yaml").String()
	secretDecryptionKey      = kingpin.Flag("secret-decryption-key", "The AES-256 key used to decrypt secrets that have been encrypted with it.").Envar("SECRET_DECRYPTION_KEY").String()

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

	// configure json logging
	initLogging()

	// define channels and waitgroup to gracefully shutdown the application
	sigs := make(chan os.Signal, 1)                                    // Create channel to receive OS signals
	stop := make(chan struct{})                                        // Create channel to receive stop signal
	signal.Notify(sigs, os.Interrupt, syscall.SIGTERM, syscall.SIGINT) // Register the sigs channel to receieve SIGTERM
	wg := &sync.WaitGroup{}                                            // Goroutines can add themselves to this to be waited on so that they finish

	// start prometheus
	go startPrometheus()

	// handle api requests
	srv := handleRequests(stop, wg)

	// wait for graceful shutdown to finish
	<-sigs // Wait for signals (this hangs until a signal arrives)
	log.Debug().Msg("Shutting down...")

	// shut down gracefully
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		log.Fatal().Err(err).Msg("Graceful server shutdown failed")
	}

	log.Debug().Msg("Stopping goroutines...")
	close(stop) // Tell goroutines to stop themselves

	log.Debug().Msg("Awaiting waitgroup...")
	wg.Wait() // Wait for all to be stopped

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

func initLogging() {

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
}

func createRouter() *gin.Engine {

	// run gin in release mode and other defaults
	gin.SetMode(gin.ReleaseMode)
	gin.DefaultWriter = log.Logger
	gin.DisableConsoleColor()

	// Creates a router without any middleware by default
	router := gin.New()

	// Logging middleware
	router.Use(ZeroLogMiddleware())

	// Recovery middleware recovers from any panics and writes a 500 if there was one.
	router.Use(gin.Recovery())

	// Gzip middleware
	router.Use(gzip.Gzip(gzip.DefaultCompression))

	// liveness and readiness
	router.GET("/liveness", func(c *gin.Context) {
		c.String(200, "I'm alive!")
	})
	router.GET("/readiness", func(c *gin.Context) {
		c.String(200, "I'm ready!")
	})

	return router
}

func handleRequests(stopChannel <-chan struct{}, waitGroup *sync.WaitGroup) *http.Server {

	secretHelper := crypt.NewSecretHelper(*secretDecryptionKey)
	configReader := config.NewConfigReader(secretHelper)

	config, err := configReader.ReadConfigFromFile(*configFilePath)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed reading configuration")
	}

	githubAPIClient := github.NewGithubAPIClient(*config.Integrations.Github, prometheusOutboundAPICallTotals)
	bitbucketAPIClient := bitbucket.NewBitbucketAPIClient(*config.Integrations.Bitbucket, prometheusOutboundAPICallTotals)
	cockroachDBClient := cockroach.NewCockroachDBClient(*config.Database, prometheusOutboundAPICallTotals)
	ciBuilderClient, err := estafette.NewCiBuilderClient(*config.APIServer, *secretDecryptionKey, prometheusOutboundAPICallTotals)
	if err != nil {
		log.Fatal().Err(err).Msg("Creating new CiBuilderClient has failed")
	}

	// set up database
	err = cockroachDBClient.Connect()
	if err != nil {
		log.Fatal().Err(err).Msg("Failed connecting to CockroachDB")
	}

	// listen to channels for push events
	githubPushEvents := make(chan github.PushEvent, config.Integrations.Github.EventChannelBufferSize)
	githubDispatcher := github.NewGithubDispatcher(stopChannel, waitGroup, config.Integrations.Github.MaxWorkers, githubAPIClient, ciBuilderClient, cockroachDBClient, githubPushEvents)
	githubDispatcher.Run()

	bitbucketPushEvents := make(chan bitbucket.RepositoryPushEvent, config.Integrations.Bitbucket.EventChannelBufferSize)
	bitbucketDispatcher := bitbucket.NewBitbucketDispatcher(stopChannel, waitGroup, config.Integrations.Bitbucket.MaxWorkers, bitbucketAPIClient, ciBuilderClient, cockroachDBClient, bitbucketPushEvents)
	bitbucketDispatcher.Run()

	estafetteCiBuilderEvents := make(chan estafette.CiBuilderEvent, config.APIServer.MaxWorkers)
	estafetteDispatcher := estafette.NewEstafetteDispatcher(stopChannel, waitGroup, config.APIServer.MaxWorkers, ciBuilderClient, cockroachDBClient, estafetteCiBuilderEvents)
	estafetteDispatcher.Run()

	// listen to http calls
	log.Debug().
		Str("port", *apiAddress).
		Msg("Serving api calls...")

	// create and init router
	router := createRouter()

	githubEventHandler := github.NewGithubEventHandler(githubPushEvents, *config.Integrations.Github, prometheusInboundEventTotals)
	router.POST("/api/integrations/github/events", githubEventHandler.Handle)

	bitbucketEventHandler := bitbucket.NewBitbucketEventHandler(bitbucketPushEvents, prometheusInboundEventTotals)
	router.POST("/api/integrations/bitbucket/events", bitbucketEventHandler.Handle)

	slackEventHandler := slack.NewSlackEventHandler(secretHelper, *config.Integrations.Slack, cockroachDBClient, *config.APIServer, prometheusInboundEventTotals)
	router.POST("/api/integrations/slack/slash", slackEventHandler.Handle)

	estafetteEventHandler := estafette.NewEstafetteEventHandler(*config.APIServer, estafetteCiBuilderEvents, prometheusInboundEventTotals)
	router.POST("/api/commands", estafetteEventHandler.Handle)

	estafetteAPIHandler := estafette.NewAPIHandler(*config.APIServer, cockroachDBClient)
	router.GET("/api/pipelines", estafetteAPIHandler.GetPipelines)
	router.GET("/api/pipelines/:source/:owner/:repo", estafetteAPIHandler.GetPipeline)
	router.GET("/api/pipelines/:source/:owner/:repo/builds", estafetteAPIHandler.GetPipelineBuilds)
	router.GET("/api/pipelines/:source/:owner/:repo/builds/:revision", estafetteAPIHandler.GetPipelineBuild)
	router.GET("/api/pipelines/:source/:owner/:repo/builds/:revision/logs", estafetteAPIHandler.GetPipelineBuildLogs)
	router.GET("/api/pipelines/:source/:owner/:repo/releases", estafetteAPIHandler.GetPipelineReleases)
	router.GET("/api/pipelines/:source/:owner/:repo/releases/:id", estafetteAPIHandler.GetPipelineRelease)
	router.POST("/api/pipelines/:source/:owner/:repo/builds/:revision/logs", estafetteAPIHandler.PostPipelineBuildLogs)
	router.GET("/api/stats/pipelinescount", estafetteAPIHandler.GetStatsPipelinesCount)
	router.GET("/api/stats/buildscount", estafetteAPIHandler.GetStatsBuildsCount)
	router.GET("/api/stats/buildsduration", estafetteAPIHandler.GetStatsBuildsDuration)

	// instantiate servers instead of using router.Run in order to handle graceful shutdown
	srv := &http.Server{
		Addr:           *apiAddress,
		Handler:        router,
		ReadTimeout:    30 * time.Second,
		WriteTimeout:   30 * time.Second,
		MaxHeaderBytes: 1 << 20,
	}

	go func() {
		if err := srv.ListenAndServe(); err != nil {
			log.Fatal().Err(err).Msg("Starting gin router failed")
		}
	}()

	return srv
}
