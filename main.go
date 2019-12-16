package main

import (
	"context"
	"io"
	"io/ioutil"
	"net/http"
	"runtime"
	"sync"
	"time"

	stdbigquery "cloud.google.com/go/bigquery"
	stdpubsub "cloud.google.com/go/pubsub"
	stdstorage "cloud.google.com/go/storage"
	"github.com/alecthomas/kingpin"
	"github.com/ericchiang/k8s"
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
	"github.com/estafette/estafette-ci-api/transport"

	"github.com/estafette/estafette-ci-api/clients/bigquery"
	"github.com/estafette/estafette-ci-api/clients/bitbucketapi"
	"github.com/estafette/estafette-ci-api/clients/builderapi"
	"github.com/estafette/estafette-ci-api/clients/cloudstorage"
	"github.com/estafette/estafette-ci-api/clients/cockroachdb"
	"github.com/estafette/estafette-ci-api/clients/dockerhubapi"
	"github.com/estafette/estafette-ci-api/clients/githubapi"
	"github.com/estafette/estafette-ci-api/clients/prometheus"
	"github.com/estafette/estafette-ci-api/clients/pubsubapi"
	"github.com/estafette/estafette-ci-api/clients/slackapi"

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

	ctx := context.Background()

	secretDecryptionKeyBytes, err := ioutil.ReadFile(*secretDecryptionKeyPath)
	if err != nil {
		log.Fatal().Err(err).Msgf("Failed reading secret decryption key from path %v", *secretDecryptionKeyPath)
	}

	log.Debug().Msg("Creating helpers...")

	warningHelper := helpers.NewWarningHelper()
	secretHelper := crypt.NewSecretHelper(string(secretDecryptionKeyBytes), false)

	log.Debug().Msg("Creating config reader...")

	configReader := config.NewConfigReader(secretHelper)

	config, err := configReader.ReadConfigFromFile(*configFilePath, true)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed reading configuration")
	}

	encryptedConfig, err := configReader.ReadConfigFromFile(*configFilePath, false)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed reading configuration without decrypting")
	}

	log.Debug().Msg("Creating clients...")

	bqClient, err := stdbigquery.NewClient(ctx, config.Integrations.BigQuery.ProjectID)
	if err != nil {
		log.Fatal().Err(err).Msg("Creating new BigQueryClient has failed")
	}
	bigqueryClient := bigquery.NewClient(config.Integrations.BigQuery, bqClient)
	bigqueryClient = bigquery.NewTracingClient(bigqueryClient)
	err = bigqueryClient.Init(ctx)
	if err != nil {
		log.Error().Err(err).Msg("Initializing BigQuery tables has failed")
	}

	bitbucketAPIClient := bitbucketapi.NewClient(*config.Integrations.Bitbucket, prometheusOutboundAPICallTotals)
	bitbucketAPIClient = bitbucketapi.NewTracingClient(bitbucketAPIClient)

	githubAPIClient := githubapi.NewClient(*config.Integrations.Github, prometheusOutboundAPICallTotals)
	githubAPIClient = githubapi.NewTracingClient(githubAPIClient)

	slackAPIClient := slackapi.NewClient(*config.Integrations.Slack, prometheusOutboundAPICallTotals)
	slackAPIClient = slackapi.NewTracingClient(slackAPIClient)

	pubsubClient, err := stdpubsub.NewClient(ctx, config.Integrations.Pubsub.DefaultProject)
	if err != nil {
		log.Fatal().Err(err).Msg("Creating new PubSubAPIClient has failed")
	}
	pubSubAPIClient := pubsubapi.NewClient(*config.Integrations.Pubsub, pubsubClient)
	pubSubAPIClient = pubsubapi.NewTracingClient(pubSubAPIClient)

	cockroachDBClient := cockroachdb.NewClient(*config.Database, prometheusOutboundAPICallTotals)
	cockroachDBClient = cockroachdb.NewTracingClient(cockroachDBClient)

	dockerHubClient := dockerhubapi.NewClient()
	dockerHubClient = dockerhubapi.NewTracingClient(dockerHubClient)

	kubeClient, err := k8s.NewInClusterClient()
	if err != nil {
		log.Fatal().Err(err).Msg("Creating k8s client failed")
	}
	ciBuilderClient := builderapi.NewClient(*config, *encryptedConfig, secretHelper, kubeClient, dockerHubClient, prometheusOutboundAPICallTotals)
	ciBuilderClient = builderapi.NewTracingClient(ciBuilderClient)

	gcsClient, err := stdstorage.NewClient(ctx)
	if err != nil {
		log.Fatal().Err(err).Msg("Creating new CloudStorageClient has failed")
	}
	cloudStorageClient := cloudstorage.NewClient(config.Integrations.CloudStorage, gcsClient)
	cloudStorageClient = cloudstorage.NewTracingClient(cloudStorageClient)

	prometheusClient := prometheus.NewClient(*config.Integrations.Prometheus)
	prometheusClient = prometheus.NewTracingClient(prometheusClient)

	// set up database
	err = cockroachDBClient.Connect(ctx)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed connecting to CockroachDB")
	}

	log.Debug().Msg("Creating services...")

	estafetteService := estafette.NewService(*config.Jobs, *config.APIServer, cockroachDBClient, prometheusClient, cloudStorageClient, ciBuilderClient, githubAPIClient.JobVarsFunc(ctx), bitbucketAPIClient.JobVarsFunc(ctx))
	estafetteService = estafette.NewTracingService(estafetteService)

	githubService := github.NewService(githubAPIClient, pubSubAPIClient, estafetteService, *config.Integrations.Github, prometheusInboundEventTotals)
	githubService = github.NewTracingService(githubService)

	bitbucketService := bitbucket.NewService(*config.Integrations.Bitbucket, bitbucketAPIClient, pubSubAPIClient, estafetteService, prometheusInboundEventTotals)
	bitbucketService = bitbucket.NewTracingService(bitbucketService)

	// transport
	bitbucketHandler := bitbucket.NewHandler(bitbucketService)
	githubHandler := github.NewHandler(githubService)
	estafetteHandler := estafette.NewHandler(*configFilePath, *config.APIServer, *config.Auth, *encryptedConfig, cockroachDBClient, cloudStorageClient, ciBuilderClient, estafetteService, warningHelper, secretHelper, githubAPIClient.JobVarsFunc(ctx), bitbucketAPIClient.JobVarsFunc(ctx))
	pubsubHandler := pubsub.NewHandler(pubSubAPIClient, estafetteService)
	slackHandler := slack.NewHandler(secretHelper, *config.Integrations.Slack, slackAPIClient, cockroachDBClient, *config.APIServer, estafetteService, githubAPIClient.JobVarsFunc(ctx), bitbucketAPIClient.JobVarsFunc(ctx), prometheusInboundEventTotals)

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
	router.Use(transport.OpenTracingMiddleware())

	// Gzip and logging middleware
	log.Debug().Msg("Adding gzip middleware...")

	alreadyZippedRoutes := router.Group("/")
	routes := router.Group("/", gzip.Gzip(gzip.BestSpeed))

	// middleware to handle auth for different endpoints
	log.Debug().Msg("Adding auth middleware...")
	authMiddleware := auth.NewAuthMiddleware(*config.Auth)

	log.Debug().Msg("Setting up routes...")
	routes.POST("/api/integrations/github/events", githubHandler.Handle)
	routes.GET("/api/integrations/github/status", func(c *gin.Context) { c.String(200, "Github, I'm cool!") })

	routes.POST("/api/integrations/bitbucket/events", bitbucketHandler.Handle)
	routes.GET("/api/integrations/bitbucket/status", func(c *gin.Context) { c.String(200, "Bitbucket, I'm cool!") })

	routes.POST("/api/integrations/slack/slash", slackHandler.Handle)
	routes.GET("/api/integrations/slack/status", func(c *gin.Context) { c.String(200, "Slack, I'm cool!") })

	// google jwt auth protected endpoints
	googleAuthorizedRoutes := routes.Group("/", authMiddleware.GoogleJWTMiddlewareFunc())
	{
		googleAuthorizedRoutes.POST("/api/integrations/pubsub/events", pubsubHandler.PostPubsubEvent)
	}
	routes.GET("/api/integrations/pubsub/status", func(c *gin.Context) { c.String(200, "Pub/Sub, I'm cool!") })

	routes.GET("/api/pipelines", estafetteHandler.GetPipelines)
	routes.GET("/api/pipelines/:source/:owner/:repo", estafetteHandler.GetPipeline)
	routes.GET("/api/pipelines/:source/:owner/:repo/builds", estafetteHandler.GetPipelineBuilds)
	routes.GET("/api/pipelines/:source/:owner/:repo/builds/:revisionOrId", estafetteHandler.GetPipelineBuild)
	routes.GET("/api/pipelines/:source/:owner/:repo/builds/:revisionOrId/warnings", estafetteHandler.GetPipelineBuildWarnings)
	alreadyZippedRoutes.GET("/api/pipelines/:source/:owner/:repo/builds/:revisionOrId/logs", estafetteHandler.GetPipelineBuildLogs)
	alreadyZippedRoutes.GET("/api/pipelines/:source/:owner/:repo/builds/:revisionOrId/logs/tail", estafetteHandler.TailPipelineBuildLogs)
	alreadyZippedRoutes.GET("/api/pipelines/:source/:owner/:repo/builds/:revisionOrId/logs.stream", estafetteHandler.TailPipelineBuildLogs)
	routes.GET("/api/pipelines/:source/:owner/:repo/releases", estafetteHandler.GetPipelineReleases)
	routes.GET("/api/pipelines/:source/:owner/:repo/releases/:id", estafetteHandler.GetPipelineRelease)
	alreadyZippedRoutes.GET("/api/pipelines/:source/:owner/:repo/releases/:id/logs", estafetteHandler.GetPipelineReleaseLogs)
	alreadyZippedRoutes.GET("/api/pipelines/:source/:owner/:repo/releases/:id/logs/tail", estafetteHandler.TailPipelineReleaseLogs)
	alreadyZippedRoutes.GET("/api/pipelines/:source/:owner/:repo/releases/:id/logs.stream", estafetteHandler.TailPipelineReleaseLogs)
	routes.GET("/api/pipelines/:source/:owner/:repo/stats/buildsdurations", estafetteHandler.GetPipelineStatsBuildsDurations)
	routes.GET("/api/pipelines/:source/:owner/:repo/stats/releasesdurations", estafetteHandler.GetPipelineStatsReleasesDurations)
	routes.GET("/api/pipelines/:source/:owner/:repo/stats/buildscpu", estafetteHandler.GetPipelineStatsBuildsCPUUsageMeasurements)
	routes.GET("/api/pipelines/:source/:owner/:repo/stats/releasescpu", estafetteHandler.GetPipelineStatsReleasesCPUUsageMeasurements)
	routes.GET("/api/pipelines/:source/:owner/:repo/stats/buildsmemory", estafetteHandler.GetPipelineStatsBuildsMemoryUsageMeasurements)
	routes.GET("/api/pipelines/:source/:owner/:repo/stats/releasesmemory", estafetteHandler.GetPipelineStatsReleasesMemoryUsageMeasurements)
	routes.GET("/api/pipelines/:source/:owner/:repo/warnings", estafetteHandler.GetPipelineWarnings)
	routes.GET("/api/stats/pipelinescount", estafetteHandler.GetStatsPipelinesCount)
	routes.GET("/api/stats/buildscount", estafetteHandler.GetStatsBuildsCount)
	routes.GET("/api/stats/releasescount", estafetteHandler.GetStatsReleasesCount)
	routes.GET("/api/stats/buildsduration", estafetteHandler.GetStatsBuildsDuration)
	routes.GET("/api/stats/buildsadoption", estafetteHandler.GetStatsBuildsAdoption)
	routes.GET("/api/stats/releasesadoption", estafetteHandler.GetStatsReleasesAdoption)
	routes.GET("/api/stats/mostbuilds", estafetteHandler.GetStatsMostBuilds)
	routes.GET("/api/stats/mostreleases", estafetteHandler.GetStatsMostReleases)
	routes.GET("/api/manifest/templates", estafetteHandler.GetManifestTemplates)
	routes.POST("/api/manifest/generate", estafetteHandler.GenerateManifest)
	routes.POST("/api/manifest/validate", estafetteHandler.ValidateManifest)
	routes.POST("/api/manifest/encrypt", estafetteHandler.EncryptSecret)
	routes.GET("/api/labels/frequent", estafetteHandler.GetFrequentLabels)

	// api key protected endpoints
	apiKeyAuthorizedRoutes := routes.Group("/", authMiddleware.APIKeyMiddlewareFunc())
	{
		apiKeyAuthorizedRoutes.POST("/api/commands", estafetteHandler.Commands)
		apiKeyAuthorizedRoutes.POST("/api/pipelines/:source/:owner/:repo/builds/:revisionOrId/logs", estafetteHandler.PostPipelineBuildLogs)
		apiKeyAuthorizedRoutes.POST("/api/pipelines/:source/:owner/:repo/releases/:id/logs", estafetteHandler.PostPipelineReleaseLogs)
		apiKeyAuthorizedRoutes.POST("/api/integrations/cron/events", estafetteHandler.PostCronEvent)
		apiKeyAuthorizedRoutes.GET("/api/copylogstocloudstorage/:source/:owner/:repo", estafetteHandler.CopyLogsToCloudStorage)
	}

	// iap protected endpoints
	iapAuthorizedRoutes := routes.Group("/", authMiddleware.IAPJWTMiddlewareFunc())
	{
		iapAuthorizedRoutes.POST("/api/pipelines/:source/:owner/:repo/builds", estafetteHandler.CreatePipelineBuild)
		iapAuthorizedRoutes.POST("/api/pipelines/:source/:owner/:repo/releases", estafetteHandler.CreatePipelineRelease)
		iapAuthorizedRoutes.DELETE("/api/pipelines/:source/:owner/:repo/builds/:revisionOrId", estafetteHandler.CancelPipelineBuild)
		iapAuthorizedRoutes.DELETE("/api/pipelines/:source/:owner/:repo/releases/:id", estafetteHandler.CancelPipelineRelease)
		iapAuthorizedRoutes.GET("/api/users/me", estafetteHandler.GetLoggedInUser)
		iapAuthorizedRoutes.GET("/api/config", estafetteHandler.GetConfig)
		iapAuthorizedRoutes.GET("/api/config/credentials", estafetteHandler.GetConfigCredentials)
		iapAuthorizedRoutes.GET("/api/config/trustedimages", estafetteHandler.GetConfigTrustedImages)
		iapAuthorizedRoutes.GET("/api/update-computed-tables", estafetteHandler.UpdateComputedTables)
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
