package main

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/signal"
	"runtime"
	"sync"
	"syscall"
	"time"

	"github.com/alecthomas/kingpin"
	"github.com/estafette/estafette-ci-api/auth"
	"github.com/estafette/estafette-ci-api/bitbucket"
	"github.com/estafette/estafette-ci-api/cockroach"
	"github.com/estafette/estafette-ci-api/config"
	"github.com/estafette/estafette-ci-api/estafette"
	"github.com/estafette/estafette-ci-api/github"
	"github.com/estafette/estafette-ci-api/slack"
	crypt "github.com/estafette/estafette-ci-crypt"
	foundation "github.com/estafette/estafette-foundation"
	"github.com/gin-contrib/gzip"
	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/rs/zerolog/log"
	"github.com/uber/jaeger-client-go"
	jaegercfg "github.com/uber/jaeger-client-go/config"
	jprom "github.com/uber/jaeger-lib/metrics/prometheus"
)

var (
	app       string
	version   string
	branch    string
	revision  string
	buildDate string
	goVersion = runtime.Version()
)

var (
	// flags
	prometheusMetricsAddress     = kingpin.Flag("metrics-listen-address", "The address to listen on for Prometheus metrics requests.").Default(":9001").String()
	prometheusMetricsPath        = kingpin.Flag("metrics-path", "The path to listen for Prometheus metrics requests.").Default("/metrics").String()
	apiAddress                   = kingpin.Flag("api-listen-address", "The address to listen on for api HTTP requests.").Default(":5000").String()
	configFilePath               = kingpin.Flag("config-file-path", "The path to yaml config file configuring this application.").Default("/configs/config.yaml").String()
	secretDecryptionKeyBase64    = kingpin.Flag("secret-decryption-key-base64", "The base64 encoded AES-256 key used to decrypt secrets that have been encrypted with it.").Envar("SECRET_DECRYPTION_KEY_BASE64").String()
	gracefulShutdownDelaySeconds = kingpin.Flag("graceful-shutdown-delay-seconds", "The number of seconds to wait with graceful shutdown in order to let endpoints update propagation finish.").Default("15").OverrideDefaultFromEnvar("GRACEFUL_SHUTDOWN_DELAY_SECONDS").Int()

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
	foundation.InitLogging(app, version, branch, revision, buildDate)

	closer := initJaeger(app)
	defer closer.Close()

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
	log.Info().Msgf("Shutting down in %v seconds...", *gracefulShutdownDelaySeconds)
	time.Sleep(time.Duration(*gracefulShutdownDelaySeconds) * 1000 * time.Millisecond)

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

func createRouter() *gin.Engine {

	// run gin in release mode and other defaults
	gin.SetMode(gin.ReleaseMode)
	gin.DefaultWriter = log.Logger
	gin.DisableConsoleColor()

	// Creates a router without any middleware by default
	router := gin.New()

	// Recovery middleware recovers from any panics and writes a 500 if there was one.
	router.Use(gin.Recovery())

	// access logs with zerolog
	// router.Use(ZeroLogMiddleware())

	// opentracing middleware
	router.Use(OpenTracingMiddleware())

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

	secretHelper := crypt.NewSecretHelper(*secretDecryptionKeyBase64, true)
	configReader := config.NewConfigReader(secretHelper)

	config, err := configReader.ReadConfigFromFile(*configFilePath, true)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed reading configuration")
	}

	encryptedConfig, err := configReader.ReadConfigFromFile(*configFilePath, false)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed reading configuration without decrypting")
	}

	githubAPIClient := github.NewGithubAPIClient(*config.Integrations.Github, prometheusOutboundAPICallTotals)
	bitbucketAPIClient := bitbucket.NewBitbucketAPIClient(*config.Integrations.Bitbucket, prometheusOutboundAPICallTotals)
	slackAPIClient := slack.NewSlackAPIClient(*config.Integrations.Slack, prometheusOutboundAPICallTotals)
	cockroachDBClient := cockroach.NewCockroachDBClient(*config.Database, prometheusOutboundAPICallTotals)
	ciBuilderClient, err := estafette.NewCiBuilderClient(*config, *encryptedConfig, secretHelper, prometheusOutboundAPICallTotals)
	if err != nil {
		log.Fatal().Err(err).Msg("Creating new CiBuilderClient has failed")
	}

	// set up database
	err = cockroachDBClient.Connect()
	if err != nil {
		log.Fatal().Err(err).Msg("Failed connecting to CockroachDB")
	}

	// create and init router
	router := createRouter()

	// Gzip and logging middleware
	gzippedRoutes := router.Group("/", gzip.Gzip(gzip.DefaultCompression))

	// middleware to handle auth for different endpoints
	authMiddleware := auth.NewAuthMiddleware(*config.Auth)

	estafetteBuildService := estafette.NewBuildService(cockroachDBClient, ciBuilderClient, githubAPIClient.JobVarsFunc(), bitbucketAPIClient.JobVarsFunc())

	githubEventHandler := github.NewGithubEventHandler(githubAPIClient, estafetteBuildService, *config.Integrations.Github, prometheusInboundEventTotals)
	gzippedRoutes.POST("/api/integrations/github/events", githubEventHandler.Handle)
	gzippedRoutes.GET("/api/integrations/github/status", func(c *gin.Context) { c.String(200, "Github, I'm cool!") })

	bitbucketEventHandler := bitbucket.NewBitbucketEventHandler(bitbucketAPIClient, estafetteBuildService, prometheusInboundEventTotals)
	gzippedRoutes.POST("/api/integrations/bitbucket/events", bitbucketEventHandler.Handle)
	gzippedRoutes.GET("/api/integrations/bitbucket/status", func(c *gin.Context) { c.String(200, "Bitbucket, I'm cool!") })

	slackEventHandler := slack.NewSlackEventHandler(secretHelper, *config.Integrations.Slack, slackAPIClient, cockroachDBClient, *config.APIServer, estafetteBuildService, githubAPIClient.JobVarsFunc(), bitbucketAPIClient.JobVarsFunc(), prometheusInboundEventTotals)
	gzippedRoutes.POST("/api/integrations/slack/slash", slackEventHandler.Handle)
	gzippedRoutes.GET("/api/integrations/slack/status", func(c *gin.Context) { c.String(200, "Slack, I'm cool!") })

	estafetteEventHandler := estafette.NewEstafetteEventHandler(*config.APIServer, ciBuilderClient, estafetteBuildService, prometheusInboundEventTotals)
	warningHelper := estafette.NewWarningHelper()

	estafetteAPIHandler := estafette.NewAPIHandler(*configFilePath, *config.APIServer, *config.Auth, *encryptedConfig, cockroachDBClient, ciBuilderClient, estafetteBuildService, warningHelper, secretHelper, githubAPIClient.JobVarsFunc(), bitbucketAPIClient.JobVarsFunc())
	gzippedRoutes.GET("/api/pipelines", estafetteAPIHandler.GetPipelines)
	gzippedRoutes.GET("/api/pipelines/:source/:owner/:repo", estafetteAPIHandler.GetPipeline)
	gzippedRoutes.GET("/api/pipelines/:source/:owner/:repo/builds", estafetteAPIHandler.GetPipelineBuilds)
	gzippedRoutes.GET("/api/pipelines/:source/:owner/:repo/builds/:revisionOrId", estafetteAPIHandler.GetPipelineBuild)
	gzippedRoutes.GET("/api/pipelines/:source/:owner/:repo/builds/:revisionOrId/logs", estafetteAPIHandler.GetPipelineBuildLogs)
	gzippedRoutes.GET("/api/pipelines/:source/:owner/:repo/builds/:revisionOrId/warnings", estafetteAPIHandler.GetPipelineBuildWarnings)
	router.GET("/api/pipelines/:source/:owner/:repo/builds/:revisionOrId/logs/tail", estafetteAPIHandler.TailPipelineBuildLogs)
	gzippedRoutes.GET("/api/pipelines/:source/:owner/:repo/releases", estafetteAPIHandler.GetPipelineReleases)
	gzippedRoutes.GET("/api/pipelines/:source/:owner/:repo/releases/:id", estafetteAPIHandler.GetPipelineRelease)
	gzippedRoutes.GET("/api/pipelines/:source/:owner/:repo/releases/:id/logs", estafetteAPIHandler.GetPipelineReleaseLogs)
	router.GET("/api/pipelines/:source/:owner/:repo/releases/:id/logs/tail", estafetteAPIHandler.TailPipelineReleaseLogs)
	gzippedRoutes.GET("/api/pipelines/:source/:owner/:repo/stats/buildsdurations", estafetteAPIHandler.GetPipelineStatsBuildsDurations)
	gzippedRoutes.GET("/api/pipelines/:source/:owner/:repo/stats/releasesdurations", estafetteAPIHandler.GetPipelineStatsReleasesDurations)
	gzippedRoutes.GET("/api/pipelines/:source/:owner/:repo/warnings", estafetteAPIHandler.GetPipelineWarnings)
	gzippedRoutes.GET("/api/stats/pipelinescount", estafetteAPIHandler.GetStatsPipelinesCount)
	gzippedRoutes.GET("/api/stats/buildscount", estafetteAPIHandler.GetStatsBuildsCount)
	gzippedRoutes.GET("/api/stats/releasescount", estafetteAPIHandler.GetStatsReleasesCount)
	gzippedRoutes.GET("/api/stats/buildsduration", estafetteAPIHandler.GetStatsBuildsDuration)
	gzippedRoutes.GET("/api/stats/buildsadoption", estafetteAPIHandler.GetStatsBuildsAdoption)
	gzippedRoutes.GET("/api/stats/releasesadoption", estafetteAPIHandler.GetStatsReleasesAdoption)
	gzippedRoutes.GET("/api/stats/mostbuilds", estafetteAPIHandler.GetStatsMostBuilds)
	gzippedRoutes.GET("/api/stats/mostreleases", estafetteAPIHandler.GetStatsMostReleases)
	gzippedRoutes.GET("/api/manifest/templates", estafetteAPIHandler.GetManifestTemplates)
	gzippedRoutes.POST("/api/manifest/generate", estafetteAPIHandler.GenerateManifest)
	gzippedRoutes.POST("/api/manifest/validate", estafetteAPIHandler.ValidateManifest)
	gzippedRoutes.POST("/api/manifest/encrypt", estafetteAPIHandler.EncryptSecret)
	gzippedRoutes.GET("/api/labels/frequent", estafetteAPIHandler.GetFrequentLabels)

	// api key protected endpoints
	apiKeyAuthorizedRoutes := gzippedRoutes.Group("/", authMiddleware.APIKeyMiddlewareFunc())
	{
		apiKeyAuthorizedRoutes.POST("/api/commands", estafetteEventHandler.Handle)
		apiKeyAuthorizedRoutes.POST("/api/pipelines/:source/:owner/:repo/builds/:revisionOrId/logs", estafetteAPIHandler.PostPipelineBuildLogs)
		apiKeyAuthorizedRoutes.POST("/api/pipelines/:source/:owner/:repo/releases/:id/logs", estafetteAPIHandler.PostPipelineReleaseLogs)
		apiKeyAuthorizedRoutes.POST("/api/integrations/cron/events", estafetteAPIHandler.PostCronEvent)
	}

	// iap protected endpoints
	iapAuthorizedRoutes := gzippedRoutes.Group("/", authMiddleware.MiddlewareFunc())
	{
		iapAuthorizedRoutes.POST("/api/pipelines/:source/:owner/:repo/builds", estafetteAPIHandler.CreatePipelineBuild)
		iapAuthorizedRoutes.POST("/api/pipelines/:source/:owner/:repo/releases", estafetteAPIHandler.CreatePipelineRelease)
		iapAuthorizedRoutes.DELETE("/api/pipelines/:source/:owner/:repo/builds/:revisionOrId", estafetteAPIHandler.CancelPipelineBuild)
		iapAuthorizedRoutes.DELETE("/api/pipelines/:source/:owner/:repo/releases/:id", estafetteAPIHandler.CancelPipelineRelease)
		iapAuthorizedRoutes.GET("/api/users/me", estafetteAPIHandler.GetLoggedInUser)
		iapAuthorizedRoutes.GET("/api/config", estafetteAPIHandler.GetConfig)
		iapAuthorizedRoutes.GET("/api/config/credentials", estafetteAPIHandler.GetConfigCredentials)
		iapAuthorizedRoutes.GET("/api/config/trustedimages", estafetteAPIHandler.GetConfigTrustedImages)
		iapAuthorizedRoutes.GET("/api/update-computed-tables", estafetteAPIHandler.UpdateComputedTables)
	}

	router.NoRoute(func(c *gin.Context) {
		c.JSON(http.StatusNotFound, gin.H{"code": http.StatusText(http.StatusNotFound), "message": "Page not found"})
	})

	// instantiate servers instead of using router.Run in order to handle graceful shutdown
	srv := &http.Server{
		Addr:        *apiAddress,
		Handler:     router,
		ReadTimeout: 30 * time.Second,
		//WriteTimeout:   30 * time.Second,
		MaxHeaderBytes: 1 << 20,
	}

	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal().Err(err).Msg("Starting gin router failed")
		}
	}()

	return srv
}

// initJaeger returns an instance of Jaeger Tracer that can be configured with environment variables
// https://github.com/jaegertracing/jaeger-client-go#environment-variables
func initJaeger(service string) io.Closer {

	cfg, err := jaegercfg.FromEnv()
	if err != nil {
		log.Fatal().Err(err).Msg("Generating Jaeger config from environment variables failed")
	}

	if os.Getenv("JAEGER_AGENT_HOST") != "" {
		// get remote config from jaeger-agent running as daemonset
		if cfg != nil && cfg.Sampler != nil && cfg.Sampler.SamplingServerURL == "" {
			cfg.Sampler.SamplingServerURL = fmt.Sprintf("http://%v:5778/sampling", os.Getenv("JAEGER_AGENT_HOST"))
		}

		// get remote config for baggage restrictions from jaeger-agent running as deamonset
		if cfg != nil && cfg.BaggageRestrictions != nil && cfg.BaggageRestrictions.HostPort == "" {
			cfg.BaggageRestrictions.HostPort = fmt.Sprintf("%v:5778", os.Getenv("JAEGER_AGENT_HOST"))
		}
	}

	closer, err := cfg.InitGlobalTracer(service, jaegercfg.Logger(jaeger.StdLogger), jaegercfg.Metrics(jprom.New()))

	if err != nil {
		log.Fatal().Err(err).Msg("Generating Jaeger tracer failed")
	}

	return closer
}

func startPrometheus() {
	http.Handle(*prometheusMetricsPath, promhttp.Handler())

	if err := http.ListenAndServe(*prometheusMetricsAddress, nil); err != nil {
		log.Fatal().Err(err).Msg("Starting Prometheus listener failed")
	}
}
