package main

import (
	"context"
	"io"
	"io/ioutil"
	"net/http"
	"runtime"
	"sync"
	"time"

	"github.com/alecthomas/kingpin"
	"github.com/estafette/estafette-ci-api/auth"
	"github.com/estafette/estafette-ci-api/bigquery"
	"github.com/estafette/estafette-ci-api/bitbucket"
	"github.com/estafette/estafette-ci-api/cockroach"
	"github.com/estafette/estafette-ci-api/config"
	"github.com/estafette/estafette-ci-api/estafette"
	"github.com/estafette/estafette-ci-api/gcs"
	"github.com/estafette/estafette-ci-api/github"
	prom "github.com/estafette/estafette-ci-api/prometheus"
	"github.com/estafette/estafette-ci-api/pubsub"
	"github.com/estafette/estafette-ci-api/slack"
	crypt "github.com/estafette/estafette-ci-crypt"
	foundation "github.com/estafette/estafette-foundation"
	"github.com/gin-contrib/gzip"
	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/rs/zerolog/log"
	"github.com/uber/jaeger-client-go"
	jaegercfg "github.com/uber/jaeger-client-go/config"
	jprom "github.com/uber/jaeger-lib/metrics/prometheus"
)

var (
	appgroup  string
	app       string
	version   string
	branch    string
	revision  string
	buildDate string
	goVersion = runtime.Version()
)

var (
	// flags
	apiAddress                   = kingpin.Flag("api-listen-address", "The address to listen on for api HTTP requests.").Default(":5000").String()
	configFilePath               = kingpin.Flag("config-file-path", "The path to yaml config file configuring this application.").Default("/configs/config.yaml").String()
	secretDecryptionKeyPath      = kingpin.Flag("secret-decryption-key-path", "The path to the AES-256 key used to decrypt secrets that have been encrypted with it.").Default("/secrets/secretDecryptionKey").OverrideDefaultFromEnvar("SECRET_DECRYPTION_KEY_PATH").String()
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

	// init log format from envvar ESTAFETTE_LOG_FORMAT
	foundation.InitLoggingFromEnv(appgroup, app, version, branch, revision, buildDate)

	closer := initJaeger()
	defer closer.Close()

	sigs, wg := foundation.InitGracefulShutdownHandling()
	stop := make(chan struct{}) // channel to signal goroutines to stop

	// start prometheus
	foundation.InitMetricsWithPort(9001)

	// handle api requests
	srv := initRequestHandlers(stop, wg)

	log.Debug().Msg("Handling requests...")

	foundation.HandleGracefulShutdown(sigs, wg, func() {

		time.Sleep(time.Duration(*gracefulShutdownDelaySeconds) * 1000 * time.Millisecond)

		// shut down gracefully
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		if err := srv.Shutdown(ctx); err != nil {
			log.Fatal().Err(err).Msg("Graceful server shutdown failed")
		}

		log.Debug().Msg("Stopping goroutines...")
		close(stop) // tell goroutines to stop themselves
	})
}

func initRequestHandlers(stopChannel <-chan struct{}, waitGroup *sync.WaitGroup) *http.Server {

	// read decryption key from secretDecryptionKeyPath
	if !foundation.FileExists(*secretDecryptionKeyPath) {
		log.Fatal().Msgf("Cannot find secret decryption key at path %v", *secretDecryptionKeyPath)
	}

	secretDecryptionKeyBytes, err := ioutil.ReadFile(*secretDecryptionKeyPath)
	if err != nil {
		log.Fatal().Err(err).Msgf("Failed reading secret decryption key from path %v", *secretDecryptionKeyPath)
	}

	secretHelper := crypt.NewSecretHelper(string(secretDecryptionKeyBytes), false)
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
	pubSubAPIClient, err := pubsub.NewPubSubAPIClient(*config.Integrations.Pubsub)
	if err != nil {
		log.Fatal().Err(err).Msg("Creating new PubSubAPIClient has failed")
	}
	cockroachDBClient := cockroach.NewCockroachDBClient(*config.Database, prometheusOutboundAPICallTotals)
	ciBuilderClient, err := estafette.NewCiBuilderClient(*config, *encryptedConfig, secretHelper, prometheusOutboundAPICallTotals)
	if err != nil {
		log.Fatal().Err(err).Msg("Creating new CiBuilderClient has failed")
	}

	bigqueryClient, err := bigquery.NewBigQueryClient(config.Integrations.BigQuery)
	if err != nil {
		log.Fatal().Err(err).Msg("Creating new BigQueryClient has failed")
	}
	err = bigqueryClient.Init()
	if err != nil {
		log.Error().Err(err).Msg("Initializing BigQuery tables has failed")
	}

	cloudStorageClient, err := gcs.NewCloudStorageClient(config.Integrations.CloudStorage)
	if err != nil {
		log.Fatal().Err(err).Msg("Creating new CloudStorageClient has failed")
	}

	// set up database
	err = cockroachDBClient.Connect()
	if err != nil {
		log.Fatal().Err(err).Msg("Failed connecting to CockroachDB")
	}

	log.Debug().Msg("Creating services, handlers and helpers...")
	prometheusClient := prom.NewPrometheusClient(*config.Integrations.Prometheus)
	estafetteBuildService := estafette.NewBuildService(*config.Jobs, *config.APIServer, cockroachDBClient, cloudStorageClient, ciBuilderClient, githubAPIClient.JobVarsFunc(), bitbucketAPIClient.JobVarsFunc())
	githubEventHandler := github.NewGithubEventHandler(githubAPIClient, pubSubAPIClient, estafetteBuildService, *config.Integrations.Github, prometheusInboundEventTotals)
	bitbucketEventHandler := bitbucket.NewBitbucketEventHandler(*config.Integrations.Bitbucket, bitbucketAPIClient, pubSubAPIClient, estafetteBuildService, prometheusInboundEventTotals)
	slackEventHandler := slack.NewSlackEventHandler(secretHelper, *config.Integrations.Slack, slackAPIClient, cockroachDBClient, *config.APIServer, estafetteBuildService, githubAPIClient.JobVarsFunc(), bitbucketAPIClient.JobVarsFunc(), prometheusInboundEventTotals)
	pubsubEventHandler := pubsub.NewPubSubEventHandler(pubSubAPIClient, estafetteBuildService)
	estafetteEventHandler := estafette.NewEstafetteEventHandler(*config.APIServer, ciBuilderClient, prometheusClient, estafetteBuildService, cockroachDBClient, prometheusInboundEventTotals)
	warningHelper := estafette.NewWarningHelper()
	estafetteAPIHandler := estafette.NewAPIHandler(*configFilePath, *config.APIServer, *config.Auth, *encryptedConfig, cockroachDBClient, cloudStorageClient, ciBuilderClient, estafetteBuildService, warningHelper, secretHelper, githubAPIClient.JobVarsFunc(), bitbucketAPIClient.JobVarsFunc())

	// run gin in release mode and other defaults
	gin.SetMode(gin.ReleaseMode)
	gin.DefaultWriter = log.Logger
	gin.DisableConsoleColor()

	// creates a router without any middleware
	log.Debug().Msg("Creating gin router...")
	router := gin.New()

	// recovery middleware recovers from any panics and writes a 500 if there was one.
	log.Debug().Msg("Adding recovery middleware...")
	router.Use(gin.Recovery())

	// opentracing middleware
	log.Debug().Msg("Adding opentracing middleware...")
	router.Use(OpenTracingMiddleware())

	// Gzip and logging middleware
	log.Debug().Msg("Adding gzip middleware...")
	router.Use(gzip.Gzip(gzip.DefaultCompression, gzip.WithExcludedExtensions([]string{".stream"})))

	// middleware to handle auth for different endpoints
	log.Debug().Msg("Adding auth middleware...")
	authMiddleware := auth.NewAuthMiddleware(*config.Auth)

	log.Debug().Msg("Setting up routes...")
	router.POST("/api/integrations/github/events", githubEventHandler.Handle)
	router.GET("/api/integrations/github/status", func(c *gin.Context) { c.String(200, "Github, I'm cool!") })

	router.POST("/api/integrations/bitbucket/events", bitbucketEventHandler.Handle)
	router.GET("/api/integrations/bitbucket/status", func(c *gin.Context) { c.String(200, "Bitbucket, I'm cool!") })

	router.POST("/api/integrations/slack/slash", slackEventHandler.Handle)
	router.GET("/api/integrations/slack/status", func(c *gin.Context) { c.String(200, "Slack, I'm cool!") })

	// google jwt auth protected endpoints
	googleAuthorizedRoutes := router.Group("/", authMiddleware.GoogleJWTMiddlewareFunc())
	{
		googleAuthorizedRoutes.POST("/api/integrations/pubsub/events", pubsubEventHandler.PostPubsubEvent)
	}
	router.GET("/api/integrations/pubsub/status", func(c *gin.Context) { c.String(200, "Pub/Sub, I'm cool!") })

	router.GET("/api/pipelines", estafetteAPIHandler.GetPipelines)
	router.GET("/api/pipelines/:source/:owner/:repo", estafetteAPIHandler.GetPipeline)
	router.GET("/api/pipelines/:source/:owner/:repo/builds", estafetteAPIHandler.GetPipelineBuilds)
	router.GET("/api/pipelines/:source/:owner/:repo/builds/:revisionOrId", estafetteAPIHandler.GetPipelineBuild)
	router.GET("/api/pipelines/:source/:owner/:repo/builds/:revisionOrId/logs", estafetteAPIHandler.GetPipelineBuildLogs)
	router.GET("/api/pipelines/:source/:owner/:repo/builds/:revisionOrId/warnings", estafetteAPIHandler.GetPipelineBuildWarnings)
	router.GET("/api/pipelines/:source/:owner/:repo/builds/:revisionOrId/logs/tail", estafetteAPIHandler.TailPipelineBuildLogs)
	router.GET("/api/pipelines/:source/:owner/:repo/builds/:revisionOrId/logs.stream", estafetteAPIHandler.TailPipelineBuildLogs)
	router.GET("/api/pipelines/:source/:owner/:repo/releases", estafetteAPIHandler.GetPipelineReleases)
	router.GET("/api/pipelines/:source/:owner/:repo/releases/:id", estafetteAPIHandler.GetPipelineRelease)
	router.GET("/api/pipelines/:source/:owner/:repo/releases/:id/logs", estafetteAPIHandler.GetPipelineReleaseLogs)
	router.GET("/api/pipelines/:source/:owner/:repo/releases/:id/logs/tail", estafetteAPIHandler.TailPipelineReleaseLogs)
	router.GET("/api/pipelines/:source/:owner/:repo/releases/:id/logs.stream", estafetteAPIHandler.TailPipelineReleaseLogs)
	router.GET("/api/pipelines/:source/:owner/:repo/stats/buildsdurations", estafetteAPIHandler.GetPipelineStatsBuildsDurations)
	router.GET("/api/pipelines/:source/:owner/:repo/stats/releasesdurations", estafetteAPIHandler.GetPipelineStatsReleasesDurations)
	router.GET("/api/pipelines/:source/:owner/:repo/stats/buildscpu", estafetteAPIHandler.GetPipelineStatsBuildsCPUUsageMeasurements)
	router.GET("/api/pipelines/:source/:owner/:repo/stats/releasescpu", estafetteAPIHandler.GetPipelineStatsReleasesCPUUsageMeasurements)
	router.GET("/api/pipelines/:source/:owner/:repo/stats/buildsmemory", estafetteAPIHandler.GetPipelineStatsBuildsMemoryUsageMeasurements)
	router.GET("/api/pipelines/:source/:owner/:repo/stats/releasesmemory", estafetteAPIHandler.GetPipelineStatsReleasesMemoryUsageMeasurements)
	router.GET("/api/pipelines/:source/:owner/:repo/warnings", estafetteAPIHandler.GetPipelineWarnings)
	router.GET("/api/stats/pipelinescount", estafetteAPIHandler.GetStatsPipelinesCount)
	router.GET("/api/stats/buildscount", estafetteAPIHandler.GetStatsBuildsCount)
	router.GET("/api/stats/releasescount", estafetteAPIHandler.GetStatsReleasesCount)
	router.GET("/api/stats/buildsduration", estafetteAPIHandler.GetStatsBuildsDuration)
	router.GET("/api/stats/buildsadoption", estafetteAPIHandler.GetStatsBuildsAdoption)
	router.GET("/api/stats/releasesadoption", estafetteAPIHandler.GetStatsReleasesAdoption)
	router.GET("/api/stats/mostbuilds", estafetteAPIHandler.GetStatsMostBuilds)
	router.GET("/api/stats/mostreleases", estafetteAPIHandler.GetStatsMostReleases)
	router.GET("/api/manifest/templates", estafetteAPIHandler.GetManifestTemplates)
	router.POST("/api/manifest/generate", estafetteAPIHandler.GenerateManifest)
	router.POST("/api/manifest/validate", estafetteAPIHandler.ValidateManifest)
	router.POST("/api/manifest/encrypt", estafetteAPIHandler.EncryptSecret)
	router.GET("/api/labels/frequent", estafetteAPIHandler.GetFrequentLabels)

	router.GET("/api/copylogstocloudstorage/:source/:owner/:repo", estafetteAPIHandler.CopyLogsToCloudStorage)

	// api key protected endpoints
	apiKeyAuthorizedRoutes := router.Group("/", authMiddleware.APIKeyMiddlewareFunc())
	{
		apiKeyAuthorizedRoutes.POST("/api/commands", estafetteEventHandler.Handle)
		apiKeyAuthorizedRoutes.POST("/api/pipelines/:source/:owner/:repo/builds/:revisionOrId/logs", estafetteAPIHandler.PostPipelineBuildLogs)
		apiKeyAuthorizedRoutes.POST("/api/pipelines/:source/:owner/:repo/releases/:id/logs", estafetteAPIHandler.PostPipelineReleaseLogs)
		apiKeyAuthorizedRoutes.POST("/api/integrations/cron/events", estafetteAPIHandler.PostCronEvent)
	}

	// iap protected endpoints
	iapAuthorizedRoutes := router.Group("/", authMiddleware.IAPJWTMiddlewareFunc())
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

	// default routes
	router.GET("/liveness", func(c *gin.Context) {
		c.String(200, "I'm alive!")
	})
	router.GET("/readiness", func(c *gin.Context) {
		c.String(200, "I'm ready!")
	})
	router.NoRoute(func(c *gin.Context) {
		c.JSON(http.StatusNotFound, gin.H{"code": http.StatusText(http.StatusNotFound), "message": "Page not found"})
	})

	// instantiate servers instead of using router.Run in order to handle graceful shutdown
	log.Debug().Msg("Starting server...")
	srv := &http.Server{
		Addr:        *apiAddress,
		Handler:     router,
		ReadTimeout: 30 * time.Second,
		//WriteTimeout:   30 * time.Second,
		MaxHeaderBytes: 1 << 20,
	}

	go func() {
		log.Debug().Msg("Listening for incoming requests...")
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal().Err(err).Msg("Starting gin router failed")
		}
	}()

	return srv
}

// initJaeger returns an instance of Jaeger Tracer that can be configured with environment variables
// https://github.com/jaegertracing/jaeger-client-go#environment-variables
func initJaeger() io.Closer {

	cfg, err := jaegercfg.FromEnv()
	if err != nil {
		log.Fatal().Err(err).Msg("Generating Jaeger config from environment variables failed")
	}

	closer, err := cfg.InitGlobalTracer(cfg.ServiceName, jaegercfg.Logger(jaeger.StdLogger), jaegercfg.Metrics(jprom.New()))

	if err != nil {
		log.Fatal().Err(err).Msg("Generating Jaeger tracer failed")
	}

	return closer
}
