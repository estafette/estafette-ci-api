package main

import (
	"context"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"runtime"
	"sync"
	"time"

	stdbigquery "cloud.google.com/go/bigquery"
	stdpubsub "cloud.google.com/go/pubsub"
	stdstorage "cloud.google.com/go/storage"
	"github.com/alecthomas/kingpin"
	"github.com/ericchiang/k8s"
	crypt "github.com/estafette/estafette-ci-crypt"
	manifest "github.com/estafette/estafette-ci-manifest"
	foundation "github.com/estafette/estafette-foundation"
	"github.com/fsnotify/fsnotify"
	"github.com/gin-contrib/gzip"
	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog/log"
	"github.com/uber/jaeger-client-go"
	jaegercfg "github.com/uber/jaeger-client-go/config"
	jprom "github.com/uber/jaeger-lib/metrics/prometheus"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/sourcerepo/v1"
	stdsourcerepo "google.golang.org/api/sourcerepo/v1"

	"github.com/estafette/estafette-ci-api/auth"
	"github.com/estafette/estafette-ci-api/config"
	"github.com/estafette/estafette-ci-api/helpers"
	"github.com/estafette/estafette-ci-api/middlewares"

	"github.com/estafette/estafette-ci-api/clients/bigquery"
	"github.com/estafette/estafette-ci-api/clients/bitbucketapi"
	"github.com/estafette/estafette-ci-api/clients/builderapi"
	"github.com/estafette/estafette-ci-api/clients/cloudsourceapi"
	"github.com/estafette/estafette-ci-api/clients/cloudstorage"
	"github.com/estafette/estafette-ci-api/clients/cockroachdb"
	"github.com/estafette/estafette-ci-api/clients/dockerhubapi"
	"github.com/estafette/estafette-ci-api/clients/githubapi"
	"github.com/estafette/estafette-ci-api/clients/prometheus"
	"github.com/estafette/estafette-ci-api/clients/pubsubapi"
	"github.com/estafette/estafette-ci-api/clients/slackapi"

	"github.com/estafette/estafette-ci-api/services/bitbucket"
	"github.com/estafette/estafette-ci-api/services/cloudsource"
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
)

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

	ctx := context.Background()

	config, encryptedConfig, manifestPreferences, secretHelper := getConfig(ctx)
	bqClient, pubsubClient, gcsClient, sourcerepoTokenSource, sourcerepoService := getGoogleCloudClients(ctx, config)
	bigqueryClient, bitbucketapiClient, githubapiClient, slackapiClient, pubsubapiClient, cockroachdbClient, dockerhubapiClient, builderapiClient, cloudstorageClient, prometheusClient, cloudsourceClient := getClients(ctx, config, encryptedConfig, manifestPreferences, secretHelper, bqClient, pubsubClient, gcsClient, sourcerepoTokenSource, sourcerepoService)
	estafetteService, githubService, bitbucketService, cloudsourceService := getServices(ctx, config, encryptedConfig, manifestPreferences, secretHelper, bigqueryClient, bitbucketapiClient, githubapiClient, slackapiClient, pubsubapiClient, cockroachdbClient, dockerhubapiClient, builderapiClient, cloudstorageClient, prometheusClient, cloudsourceClient)
	bitbucketHandler, githubHandler, estafetteHandler, pubsubHandler, slackHandler, cloudsourceHandler := getHandlers(ctx, config, encryptedConfig, manifestPreferences, secretHelper, bigqueryClient, bitbucketapiClient, githubapiClient, slackapiClient, pubsubapiClient, cockroachdbClient, dockerhubapiClient, builderapiClient, cloudstorageClient, prometheusClient, cloudsourceClient, estafetteService, githubService, bitbucketService, cloudsourceService)

	srv, authMiddleware := configureGinGonic(config, bitbucketHandler, githubHandler, estafetteHandler, pubsubHandler, slackHandler, cloudsourceHandler)

	// watch for configmap changes
	foundation.WatchForFileChanges(*configFilePath, func(event fsnotify.Event) {
		log.Info().Msgf("Configmap at %v was updated, refreshing instances...", *configFilePath)

		// refresh config
		config, encryptedConfig, manifestPreferences, secretHelper := getConfig(ctx)
		refreshConfig(ctx, config, encryptedConfig, manifestPreferences, secretHelper, bigqueryClient, bitbucketapiClient, githubapiClient, slackapiClient, pubsubapiClient, cockroachdbClient, dockerhubapiClient, builderapiClient, cloudstorageClient, prometheusClient, cloudsourceClient, estafetteService, githubService, bitbucketService, cloudsourceService, bitbucketHandler, githubHandler, estafetteHandler, pubsubHandler, slackHandler, cloudsourceHandler, authMiddleware)
	})

	// watch for service account key file changes
	foundation.WatchForFileChanges(os.Getenv("GOOGLE_APPLICATION_CREDENTIALS"), func(event fsnotify.Event) {
		log.Info().Msg("Service account key file was updated, refreshing instances...")

		// refresh google cloud clients
		bqClient, pubsubClient, gcsClient, sourcerepoTokenSource, sourcerepoService = getGoogleCloudClients(ctx, config)
	})

	return srv
}

func getConfig(ctx context.Context) (*config.APIConfig, *config.APIConfig, *manifest.EstafetteManifestPreferences, crypt.SecretHelper) {

	// read decryption key from secretDecryptionKeyPath
	if !foundation.FileExists(*secretDecryptionKeyPath) {
		log.Fatal().Msgf("Cannot find secret decryption key at path %v", *secretDecryptionKeyPath)
	}

	secretDecryptionKeyBytes, err := ioutil.ReadFile(*secretDecryptionKeyPath)
	if err != nil {
		log.Fatal().Err(err).Msgf("Failed reading secret decryption key from path %v", *secretDecryptionKeyPath)
	}

	log.Debug().Msg("Creating helpers...")

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

	manifestPreferences := manifest.GetDefaultManifestPreferences()
	if config.ManifestPreferences != nil {
		manifestPreferences = config.ManifestPreferences
	}

	return config, encryptedConfig, manifestPreferences, secretHelper
}

func getGoogleCloudClients(ctx context.Context, config *config.APIConfig) (*stdbigquery.Client, *stdpubsub.Client, *stdstorage.Client, oauth2.TokenSource, *stdsourcerepo.Service) {

	log.Debug().Msg("Creating Google Cloud clients...")

	bqClient, err := stdbigquery.NewClient(ctx, config.Integrations.BigQuery.ProjectID)
	if err != nil {
		log.Fatal().Err(err).Msg("Creating new BigQueryClient has failed")
	}

	pubsubClient, err := stdpubsub.NewClient(ctx, config.Integrations.Pubsub.DefaultProject)
	if err != nil {
		log.Fatal().Err(err).Msg("Creating google pubsub client has failed")
	}

	gcsClient, err := stdstorage.NewClient(ctx)
	if err != nil {
		log.Fatal().Err(err).Msg("Creating google cloud storage client has failed")
	}

	tokenSource, err := google.DefaultTokenSource(ctx, sourcerepo.CloudPlatformScope)
	if err != nil {
		log.Fatal().Err(err).Msg("Creating google cloud token source has failed")
	}
	sourcerepoService, err := stdsourcerepo.New(oauth2.NewClient(ctx, tokenSource))
	if err != nil {
		log.Fatal().Err(err).Msg("Creating google cloud source repo service has failed")
	}

	return bqClient, pubsubClient, gcsClient, tokenSource, sourcerepoService
}

func getClients(ctx context.Context, config *config.APIConfig, encryptedConfig *config.APIConfig, manifestPreferences *manifest.EstafetteManifestPreferences, secretHelper crypt.SecretHelper, bqClient *stdbigquery.Client, pubsubClient *stdpubsub.Client, gcsClient *stdstorage.Client, sourcerepoTokenSource oauth2.TokenSource, sourcerepoService *stdsourcerepo.Service) (bigqueryClient bigquery.Client, bitbucketapiClient bitbucketapi.Client, githubapiClient githubapi.Client, slackapiClient slackapi.Client, pubsubapiClient pubsubapi.Client, cockroachdbClient cockroachdb.Client, dockerhubapiClient dockerhubapi.Client, builderapiClient builderapi.Client, cloudstorageClient cloudstorage.Client, prometheusClient prometheus.Client, cloudsourceClient cloudsourceapi.Client) {

	log.Debug().Msg("Creating clients...")

	{
		bigqueryClient = bigquery.NewClient(config.Integrations.BigQuery, bqClient)
		bigqueryClient = bigquery.NewTracingClient(bigqueryClient)
		bigqueryClient = bigquery.NewLoggingClient(bigqueryClient)
		bigqueryClient = bigquery.NewMetricsClient(bigqueryClient,
			helpers.NewRequestCounter("bigquery_client"),
			helpers.NewRequestHistogram("bigquery_client"),
		)
	}
	err := bigqueryClient.Init(ctx)
	if err != nil {
		log.Error().Err(err).Msg("Initializing BigQuery tables has failed")
	}

	{
		bitbucketapiClient = bitbucketapi.NewClient(*config.Integrations.Bitbucket)
		bitbucketapiClient = bitbucketapi.NewTracingClient(bitbucketapiClient)
		bitbucketapiClient = bitbucketapi.NewLoggingClient(bitbucketapiClient)
		bitbucketapiClient = bitbucketapi.NewMetricsClient(bitbucketapiClient,
			helpers.NewRequestCounter("bitbucketapi_client"),
			helpers.NewRequestHistogram("bitbucketapi_client"),
		)
	}

	{
		githubapiClient = githubapi.NewClient(*config.Integrations.Github)
		githubapiClient = githubapi.NewTracingClient(githubapiClient)
		githubapiClient = githubapi.NewLoggingClient(githubapiClient)
		githubapiClient = githubapi.NewMetricsClient(githubapiClient,
			helpers.NewRequestCounter("githubapi_client"),
			helpers.NewRequestHistogram("githubapi_client"),
		)
	}

	{
		slackapiClient = slackapi.NewClient(*config.Integrations.Slack)
		slackapiClient = slackapi.NewTracingClient(slackapiClient)
		slackapiClient = slackapi.NewLoggingClient(slackapiClient)
		slackapiClient = slackapi.NewMetricsClient(slackapiClient,
			helpers.NewRequestCounter("slackapi_client"),
			helpers.NewRequestHistogram("slackapi_client"),
		)
	}

	{
		pubsubapiClient = pubsubapi.NewClient(*config.Integrations.Pubsub, *manifestPreferences, pubsubClient)
		pubsubapiClient = pubsubapi.NewTracingClient(pubsubapiClient)
		pubsubapiClient = pubsubapi.NewLoggingClient(pubsubapiClient)
		pubsubapiClient = pubsubapi.NewMetricsClient(pubsubapiClient,
			helpers.NewRequestCounter("pubsubapi_client"),
			helpers.NewRequestHistogram("pubsubapi_client"),
		)
	}

	{
		cockroachdbClient = cockroachdb.NewClient(*config.Database, *manifestPreferences)
		cockroachdbClient = cockroachdb.NewTracingClient(cockroachdbClient)
		cockroachdbClient = cockroachdb.NewLoggingClient(cockroachdbClient)
		cockroachdbClient = cockroachdb.NewMetricsClient(cockroachdbClient,
			helpers.NewRequestCounter("cockroachdb_client"),
			helpers.NewRequestHistogram("cockroachdb_client"),
		)
	}
	err = cockroachdbClient.Connect(ctx)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed connecting to CockroachDB")
	}

	{
		dockerhubapiClient = dockerhubapi.NewClient()
		dockerhubapiClient = dockerhubapi.NewTracingClient(dockerhubapiClient)
		dockerhubapiClient = dockerhubapi.NewLoggingClient(dockerhubapiClient)
		dockerhubapiClient = dockerhubapi.NewMetricsClient(dockerhubapiClient,
			helpers.NewRequestCounter("dockerhubapi_client"),
			helpers.NewRequestHistogram("dockerhubapi_client"),
		)
	}

	{
		kubeClient, err := k8s.NewInClusterClient()
		if err != nil {
			log.Fatal().Err(err).Msg("Creating kubernetes client failed")
		}
		builderapiClient = builderapi.NewClient(*config, *encryptedConfig, secretHelper, kubeClient, dockerhubapiClient)
		builderapiClient = builderapi.NewTracingClient(builderapiClient)
		builderapiClient = builderapi.NewLoggingClient(builderapiClient)
		builderapiClient = builderapi.NewMetricsClient(builderapiClient,
			helpers.NewRequestCounter("builderapi_client"),
			helpers.NewRequestHistogram("builderapi_client"),
		)
	}

	{
		cloudstorageClient = cloudstorage.NewClient(config.Integrations.CloudStorage, gcsClient)
		cloudstorageClient = cloudstorage.NewTracingClient(cloudstorageClient)
		cloudstorageClient = cloudstorage.NewLoggingClient(cloudstorageClient)
		cloudstorageClient = cloudstorage.NewMetricsClient(cloudstorageClient,
			helpers.NewRequestCounter("cloudstorage_client"),
			helpers.NewRequestHistogram("cloudstorage_client"),
		)
	}

	{
		prometheusClient = prometheus.NewClient(*config.Integrations.Prometheus)
		prometheusClient = prometheus.NewTracingClient(prometheusClient)
		prometheusClient = prometheus.NewLoggingClient(prometheusClient)
		prometheusClient = prometheus.NewMetricsClient(prometheusClient,
			helpers.NewRequestCounter("prometheus_client"),
			helpers.NewRequestHistogram("prometheus_client"),
		)
	}

	{
		cloudsourceClient, err = cloudsourceapi.NewClient(*config.Integrations.CloudSource, sourcerepoTokenSource, sourcerepoService)
		if err != nil {
			log.Error().Err(err).Msg("Creating new client for Cloud Source has failed")
		}
		cloudsourceClient = cloudsourceapi.NewTracingClient(cloudsourceClient)
		cloudsourceClient = cloudsourceapi.NewLoggingClient(cloudsourceClient)
		cloudsourceClient = cloudsourceapi.NewMetricsClient(cloudsourceClient,
			helpers.NewRequestCounter("cloudsource_client"),
			helpers.NewRequestHistogram("cloudsource_client"),
		)
	}

	return
}

func getServices(ctx context.Context, config *config.APIConfig, encryptedConfig *config.APIConfig, manifestPreferences *manifest.EstafetteManifestPreferences, secretHelper crypt.SecretHelper, bigqueryClient bigquery.Client, bitbucketapiClient bitbucketapi.Client, githubapiClient githubapi.Client, slackapiClient slackapi.Client, pubsubapiClient pubsubapi.Client, cockroachdbClient cockroachdb.Client, dockerhubapiClient dockerhubapi.Client, builderapiClient builderapi.Client, cloudstorageClient cloudstorage.Client, prometheusClient prometheus.Client, cloudsourceClient cloudsourceapi.Client) (estafetteService estafette.Service, githubService github.Service, bitbucketService bitbucket.Service, cloudsourceService cloudsource.Service) {

	log.Debug().Msg("Creating services...")

	{
		estafetteService = estafette.NewService(*config.Jobs, *config.APIServer, *manifestPreferences, cockroachdbClient, prometheusClient, cloudstorageClient, builderapiClient, githubapiClient.JobVarsFunc(ctx), bitbucketapiClient.JobVarsFunc(ctx), cloudsourceClient.JobVarsFunc(ctx))
		estafetteService = estafette.NewTracingService(estafetteService)
		estafetteService = estafette.NewLoggingService(estafetteService)
		estafetteService = estafette.NewMetricsService(estafetteService,
			helpers.NewRequestCounter("estafette_service"),
			helpers.NewRequestHistogram("estafette_service"),
		)
	}

	{
		githubService = github.NewService(*config.Integrations.Github, githubapiClient, pubsubapiClient, estafetteService)
		githubService = github.NewTracingService(githubService)
		githubService = github.NewLoggingService(githubService)
		githubService = github.NewMetricsService(githubService,
			helpers.NewRequestCounter("github_service"),
			helpers.NewRequestHistogram("github_service"),
		)
	}

	{
		bitbucketService = bitbucket.NewService(*config.Integrations.Bitbucket, bitbucketapiClient, pubsubapiClient, estafetteService)
		bitbucketService = bitbucket.NewTracingService(bitbucketService)
		bitbucketService = bitbucket.NewLoggingService(bitbucketService)
		bitbucketService = bitbucket.NewMetricsService(bitbucketService,
			helpers.NewRequestCounter("bitbucket_service"),
			helpers.NewRequestHistogram("bitbucket_service"),
		)
	}

	{
		cloudsourceService = cloudsource.NewService(*config.Integrations.CloudSource, cloudsourceClient, pubsubapiClient, estafetteService)
		cloudsourceService = cloudsource.NewTracingService(cloudsourceService)
		cloudsourceService = cloudsource.NewLoggingService(cloudsourceService)
		cloudsourceService = cloudsource.NewMetricsService(cloudsourceService,
			helpers.NewRequestCounter("cloudsource_service"),
			helpers.NewRequestHistogram("cloudsource_service"),
		)
	}

	return
}

func getHandlers(ctx context.Context, config *config.APIConfig, encryptedConfig *config.APIConfig, manifestPreferences *manifest.EstafetteManifestPreferences, secretHelper crypt.SecretHelper, bigqueryClient bigquery.Client, bitbucketapiClient bitbucketapi.Client, githubapiClient githubapi.Client, slackapiClient slackapi.Client, pubsubapiClient pubsubapi.Client, cockroachdbClient cockroachdb.Client, dockerhubapiClient dockerhubapi.Client, builderapiClient builderapi.Client, cloudstorageClient cloudstorage.Client, prometheusClient prometheus.Client, cloudsourceClient cloudsourceapi.Client, estafetteService estafette.Service, githubService github.Service, bitbucketService bitbucket.Service, cloudsourceService cloudsource.Service) (bitbucketHandler bitbucket.Handler, githubHandler github.Handler, estafetteHandler estafette.Handler, pubsubHandler pubsub.Handler, slackHandler slack.Handler, cloudsourceHandler cloudsource.Handler) {

	log.Debug().Msg("Creating http handlers...")

	warningHelper := helpers.NewWarningHelper(secretHelper)

	// transport
	bitbucketHandler = bitbucket.NewHandler(bitbucketService)
	githubHandler = github.NewHandler(githubService)
	estafetteHandler = estafette.NewHandler(*configFilePath, *config.APIServer, *config.Auth, *encryptedConfig, config.Catalog, *manifestPreferences, cockroachdbClient, cloudstorageClient, builderapiClient, estafetteService, warningHelper, secretHelper, githubapiClient.JobVarsFunc(ctx), bitbucketapiClient.JobVarsFunc(ctx), cloudsourceClient.JobVarsFunc(ctx))
	pubsubHandler = pubsub.NewHandler(pubsubapiClient, estafetteService)
	slackHandler = slack.NewHandler(secretHelper, *config.Integrations.Slack, slackapiClient, cockroachdbClient, *config.APIServer, estafetteService, githubapiClient.JobVarsFunc(ctx), bitbucketapiClient.JobVarsFunc(ctx))
	cloudsourceHandler = cloudsource.NewHandler(pubsubapiClient, cloudsourceService)

	return
}

func refreshConfig(ctx context.Context, config *config.APIConfig, encryptedConfig *config.APIConfig, manifestPreferences *manifest.EstafetteManifestPreferences, secretHelper crypt.SecretHelper, bigqueryClient bigquery.Client, bitbucketapiClient bitbucketapi.Client, githubapiClient githubapi.Client, slackapiClient slackapi.Client, pubsubapiClient pubsubapi.Client, cockroachdbClient cockroachdb.Client, dockerhubapiClient dockerhubapi.Client, builderapiClient builderapi.Client, cloudstorageClient cloudstorage.Client, prometheusClient prometheus.Client, cloudsourceClient cloudsourceapi.Client, estafetteService estafette.Service, githubService github.Service, bitbucketService bitbucket.Service, cloudsourceService cloudsource.Service, bitbucketHandler bitbucket.Handler, githubHandler github.Handler, estafetteHandler estafette.Handler, pubsubHandler pubsub.Handler, slackHandler slack.Handler, cloudsourceHandler cloudsource.Handler, authMiddleware auth.Middleware) {

	bigqueryClient.RefreshConfig(config)
	bitbucketapiClient.RefreshConfig(config)
	githubapiClient.RefreshConfig(config)
	slackapiClient.RefreshConfig(config)
	pubsubapiClient.RefreshConfig(config, *manifestPreferences)
	cockroachdbClient.RefreshConfig(config, *manifestPreferences)
	// dockerhubapiClient.RefreshConfig(config)
	builderapiClient.RefreshConfig(config)
	cloudstorageClient.RefreshConfig(config)
	prometheusClient.RefreshConfig(config)
	cloudsourceClient.RefreshConfig(config)

	estafetteService.RefreshConfig(config, *manifestPreferences)
	githubService.RefreshConfig(config)
	bitbucketService.RefreshConfig(config)
	cloudsourceService.RefreshConfig(config)

	// bitbucketHandler.RefreshConfig(config)
	// githubHandler.RefreshConfig(config)
	estafetteHandler.RefreshConfig(config, encryptedConfig, *manifestPreferences)
	// pubsubHandler.RefreshConfig(config)
	// slackHandler.RefreshConfig(config)
	// cloudsourceHandler.RefreshConfig(config)

	authMiddleware.RefreshConfig(config)
}

func configureGinGonic(config *config.APIConfig, bitbucketHandler bitbucket.Handler, githubHandler github.Handler, estafetteHandler estafette.Handler, pubsubHandler pubsub.Handler, slackHandler slack.Handler, cloudsourceHandler cloudsource.Handler) (*http.Server, auth.Middleware) {

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
	router.Use(middlewares.OpenTracingMiddleware())

	// Gzip and logging middleware
	log.Debug().Msg("Adding gzip middleware...")

	alreadyZippedRoutes := router.Group("/")
	routes := router.Group("/", gzip.Gzip(gzip.BestSpeed))

	// middleware to handle auth for different endpoints
	log.Debug().Msg("Adding auth middleware...")
	authMiddleware := auth.NewAuthMiddleware(config.Auth)

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
		googleAuthorizedRoutes.POST("/api/integrations/cloudsource/events", cloudsourceHandler.PostPubsubEvent)
	}
	routes.GET("/api/integrations/pubsub/status", func(c *gin.Context) { c.String(200, "Pub/Sub, I'm cool!") })
	routes.GET("/api/integrations/cloudsource/status", func(c *gin.Context) { c.String(200, "Cloud Source, I'm cool!") })

	routes.GET("/api/pipelines", estafetteHandler.GetPipelines)
	routes.GET("/api/pipelines/:source/:owner/:repo", estafetteHandler.GetPipeline)
	routes.GET("/api/pipelines/:source/:owner/:repo/recentbuilds", estafetteHandler.GetPipelineRecentBuilds)
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
	routes.GET("/api/catalog/filters", estafetteHandler.GetCatalogFilters)
	routes.GET("/api/catalog/filtervalues", estafetteHandler.GetCatalogFilterValues)
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

	return srv, authMiddleware
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
