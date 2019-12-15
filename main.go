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
	crypt "github.com/estafette/estafette-ci-crypt"
	foundation "github.com/estafette/estafette-foundation"
	"github.com/gin-contrib/gzip"
	"github.com/gin-gonic/gin"
	stdprometheus "github.com/prometheus/client_golang/prometheus"
	"github.com/rs/zerolog/log"
	"github.com/uber/jaeger-client-go"
	jaegercfg "github.com/uber/jaeger-client-go/config"
	jprom "github.com/uber/jaeger-lib/metrics/prometheus"

	"github.com/estafette/estafette-ci-api/auth"
	"github.com/estafette/estafette-ci-api/config"
	"github.com/estafette/estafette-ci-api/helpers"

	"github.com/estafette/estafette-ci-api/clients/bigquery"
	bitbucketclt "github.com/estafette/estafette-ci-api/clients/bitbucket"
	"github.com/estafette/estafette-ci-api/clients/cloudstorage"
	"github.com/estafette/estafette-ci-api/clients/cockroach"
	estafetteclt "github.com/estafette/estafette-ci-api/clients/estafette"
	githubclt "github.com/estafette/estafette-ci-api/clients/github"
	prometheusclt "github.com/estafette/estafette-ci-api/clients/prometheus"
	pubsubclt "github.com/estafette/estafette-ci-api/clients/pubsub"
	slackclt "github.com/estafette/estafette-ci-api/clients/slack"

	"github.com/estafette/estafette-ci-api/services/bitbucket"
	"github.com/estafette/estafette-ci-api/services/estafette"
	"github.com/estafette/estafette-ci-api/services/github"
	"github.com/estafette/estafette-ci-api/services/pubsub"
	"github.com/estafette/estafette-ci-api/services/slack"
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
	prometheusInboundEventTotals = stdprometheus.NewCounterVec(
		stdprometheus.CounterOpts{
			Name: "estafette_ci_api_inbound_event_totals",
			Help: "Total of inbound events.",
		},
		[]string{"event", "source"},
	)

	// prometheusOutboundAPICallTotals is the prometheus timeline serie that keeps track of outbound api calls
	prometheusOutboundAPICallTotals = stdprometheus.NewCounterVec(
		stdprometheus.CounterOpts{
			Name: "estafette_ci_api_outbound_api_call_totals",
			Help: "Total of outgoing api calls.",
		},
		[]string{"target"},
	)
)

func init() {
	// Metrics have to be registered to be exposed:
	stdprometheus.MustRegister(prometheusInboundEventTotals)
	stdprometheus.MustRegister(prometheusOutboundAPICallTotals)
}

func main() {

	// parse command line parameters
	kingpin.Parse()

	// init log format from envvar ESTAFETTE_LOG_FORMAT
	foundation.InitLoggingFromEnv(foundation.NewApplicationInfo(appgroup, app, version, branch, revision, buildDate))

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

	githubAPIClient := githubclt.NewClient(*config.Integrations.Github, prometheusOutboundAPICallTotals)
	bitbucketAPIClient := bitbucketclt.NewClient(*config.Integrations.Bitbucket, prometheusOutboundAPICallTotals)
	slackAPIClient := slackclt.NewClient(*config.Integrations.Slack, prometheusOutboundAPICallTotals)
	pubSubAPIClient, err := pubsubclt.NewClient(*config.Integrations.Pubsub)
	if err != nil {
		log.Fatal().Err(err).Msg("Creating new PubSubAPIClient has failed")
	}
	cockroachDBClient := cockroach.NewClient(*config.Database, prometheusOutboundAPICallTotals)
	ciBuilderClient, err := estafetteclt.NewClient(*config, *encryptedConfig, secretHelper, prometheusOutboundAPICallTotals)
	if err != nil {
		log.Fatal().Err(err).Msg("Creating new CiBuilderClient has failed")
	}

	bigqueryClient, err := bigquery.NewClient(config.Integrations.BigQuery)
	if err != nil {
		log.Fatal().Err(err).Msg("Creating new BigQueryClient has failed")
	}
	err = bigqueryClient.Init()
	if err != nil {
		log.Error().Err(err).Msg("Initializing BigQuery tables has failed")
	}

	cloudStorageClient, err := cloudstorage.NewClient(config.Integrations.CloudStorage)
	if err != nil {
		log.Fatal().Err(err).Msg("Creating new CloudStorageClient has failed")
	}

	// set up database
	err = cockroachDBClient.Connect()
	if err != nil {
		log.Fatal().Err(err).Msg("Failed connecting to CockroachDB")
	}

	log.Debug().Msg("Creating services, handlers and helpers...")
	prometheusClient := prometheusclt.NewClient(*config.Integrations.Prometheus)
	estafetteBuildService := estafette.NewService(*config.Jobs, *config.APIServer, cockroachDBClient, prometheusClient, cloudStorageClient, ciBuilderClient, githubAPIClient.JobVarsFunc(), bitbucketAPIClient.JobVarsFunc())
	githubEventHandler := github.NewService(githubAPIClient, pubSubAPIClient, estafetteBuildService, *config.Integrations.Github, prometheusInboundEventTotals)
	bitbucketEventHandler := bitbucket.NewService(*config.Integrations.Bitbucket, bitbucketAPIClient, pubSubAPIClient, estafetteBuildService, prometheusInboundEventTotals)
	slackEventHandler := slack.NewService(secretHelper, *config.Integrations.Slack, slackAPIClient, cockroachDBClient, *config.APIServer, estafetteBuildService, githubAPIClient.JobVarsFunc(), bitbucketAPIClient.JobVarsFunc(), prometheusInboundEventTotals)
	pubsubEventHandler := pubsub.NewService(pubSubAPIClient, estafetteBuildService)
	warningHelper := helpers.NewWarningHelper()
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

	alreadyZippedRoutes := router.Group("/")
	routes := router.Group("/", gzip.Gzip(gzip.BestSpeed))

	// middleware to handle auth for different endpoints
	log.Debug().Msg("Adding auth middleware...")
	authMiddleware := auth.NewAuthMiddleware(*config.Auth)

	log.Debug().Msg("Setting up routes...")
	routes.POST("/api/integrations/github/events", githubEventHandler.Handle)
	routes.GET("/api/integrations/github/status", func(c *gin.Context) { c.String(200, "Github, I'm cool!") })

	routes.POST("/api/integrations/bitbucket/events", bitbucketEventHandler.Handle)
	routes.GET("/api/integrations/bitbucket/status", func(c *gin.Context) { c.String(200, "Bitbucket, I'm cool!") })

	routes.POST("/api/integrations/slack/slash", slackEventHandler.Handle)
	routes.GET("/api/integrations/slack/status", func(c *gin.Context) { c.String(200, "Slack, I'm cool!") })

	// google jwt auth protected endpoints
	googleAuthorizedRoutes := routes.Group("/", authMiddleware.GoogleJWTMiddlewareFunc())
	{
		googleAuthorizedRoutes.POST("/api/integrations/pubsub/events", pubsubEventHandler.PostPubsubEvent)
	}
	routes.GET("/api/integrations/pubsub/status", func(c *gin.Context) { c.String(200, "Pub/Sub, I'm cool!") })

	routes.GET("/api/pipelines", estafetteAPIHandler.GetPipelines)
	routes.GET("/api/pipelines/:source/:owner/:repo", estafetteAPIHandler.GetPipeline)
	routes.GET("/api/pipelines/:source/:owner/:repo/builds", estafetteAPIHandler.GetPipelineBuilds)
	routes.GET("/api/pipelines/:source/:owner/:repo/builds/:revisionOrId", estafetteAPIHandler.GetPipelineBuild)
	routes.GET("/api/pipelines/:source/:owner/:repo/builds/:revisionOrId/warnings", estafetteAPIHandler.GetPipelineBuildWarnings)
	alreadyZippedRoutes.GET("/api/pipelines/:source/:owner/:repo/builds/:revisionOrId/logs", estafetteAPIHandler.GetPipelineBuildLogs)
	alreadyZippedRoutes.GET("/api/pipelines/:source/:owner/:repo/builds/:revisionOrId/logs/tail", estafetteAPIHandler.TailPipelineBuildLogs)
	alreadyZippedRoutes.GET("/api/pipelines/:source/:owner/:repo/builds/:revisionOrId/logs.stream", estafetteAPIHandler.TailPipelineBuildLogs)
	routes.GET("/api/pipelines/:source/:owner/:repo/releases", estafetteAPIHandler.GetPipelineReleases)
	routes.GET("/api/pipelines/:source/:owner/:repo/releases/:id", estafetteAPIHandler.GetPipelineRelease)
	alreadyZippedRoutes.GET("/api/pipelines/:source/:owner/:repo/releases/:id/logs", estafetteAPIHandler.GetPipelineReleaseLogs)
	alreadyZippedRoutes.GET("/api/pipelines/:source/:owner/:repo/releases/:id/logs/tail", estafetteAPIHandler.TailPipelineReleaseLogs)
	alreadyZippedRoutes.GET("/api/pipelines/:source/:owner/:repo/releases/:id/logs.stream", estafetteAPIHandler.TailPipelineReleaseLogs)
	routes.GET("/api/pipelines/:source/:owner/:repo/stats/buildsdurations", estafetteAPIHandler.GetPipelineStatsBuildsDurations)
	routes.GET("/api/pipelines/:source/:owner/:repo/stats/releasesdurations", estafetteAPIHandler.GetPipelineStatsReleasesDurations)
	routes.GET("/api/pipelines/:source/:owner/:repo/stats/buildscpu", estafetteAPIHandler.GetPipelineStatsBuildsCPUUsageMeasurements)
	routes.GET("/api/pipelines/:source/:owner/:repo/stats/releasescpu", estafetteAPIHandler.GetPipelineStatsReleasesCPUUsageMeasurements)
	routes.GET("/api/pipelines/:source/:owner/:repo/stats/buildsmemory", estafetteAPIHandler.GetPipelineStatsBuildsMemoryUsageMeasurements)
	routes.GET("/api/pipelines/:source/:owner/:repo/stats/releasesmemory", estafetteAPIHandler.GetPipelineStatsReleasesMemoryUsageMeasurements)
	routes.GET("/api/pipelines/:source/:owner/:repo/warnings", estafetteAPIHandler.GetPipelineWarnings)
	routes.GET("/api/stats/pipelinescount", estafetteAPIHandler.GetStatsPipelinesCount)
	routes.GET("/api/stats/buildscount", estafetteAPIHandler.GetStatsBuildsCount)
	routes.GET("/api/stats/releasescount", estafetteAPIHandler.GetStatsReleasesCount)
	routes.GET("/api/stats/buildsduration", estafetteAPIHandler.GetStatsBuildsDuration)
	routes.GET("/api/stats/buildsadoption", estafetteAPIHandler.GetStatsBuildsAdoption)
	routes.GET("/api/stats/releasesadoption", estafetteAPIHandler.GetStatsReleasesAdoption)
	routes.GET("/api/stats/mostbuilds", estafetteAPIHandler.GetStatsMostBuilds)
	routes.GET("/api/stats/mostreleases", estafetteAPIHandler.GetStatsMostReleases)
	routes.GET("/api/manifest/templates", estafetteAPIHandler.GetManifestTemplates)
	routes.POST("/api/manifest/generate", estafetteAPIHandler.GenerateManifest)
	routes.POST("/api/manifest/validate", estafetteAPIHandler.ValidateManifest)
	routes.POST("/api/manifest/encrypt", estafetteAPIHandler.EncryptSecret)
	routes.GET("/api/labels/frequent", estafetteAPIHandler.GetFrequentLabels)

	// api key protected endpoints
	apiKeyAuthorizedRoutes := routes.Group("/", authMiddleware.APIKeyMiddlewareFunc())
	{
		apiKeyAuthorizedRoutes.POST("/api/commands", estafetteAPIHandler.Commands)
		apiKeyAuthorizedRoutes.POST("/api/pipelines/:source/:owner/:repo/builds/:revisionOrId/logs", estafetteAPIHandler.PostPipelineBuildLogs)
		apiKeyAuthorizedRoutes.POST("/api/pipelines/:source/:owner/:repo/releases/:id/logs", estafetteAPIHandler.PostPipelineReleaseLogs)
		apiKeyAuthorizedRoutes.POST("/api/integrations/cron/events", estafetteAPIHandler.PostCronEvent)
		apiKeyAuthorizedRoutes.GET("/api/copylogstocloudstorage/:source/:owner/:repo", estafetteAPIHandler.CopyLogsToCloudStorage)
	}

	// iap protected endpoints
	iapAuthorizedRoutes := routes.Group("/", authMiddleware.IAPJWTMiddlewareFunc())
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
	routes.GET("/liveness", func(c *gin.Context) {
		c.String(200, "I'm alive!")
	})
	routes.GET("/readiness", func(c *gin.Context) {
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
