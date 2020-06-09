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
	"github.com/estafette/estafette-ci-api/services/rbac"
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

	config, encryptedConfig, secretHelper := getConfig(ctx)
	bqClient, pubsubClient, gcsClient, sourcerepoTokenSource, sourcerepoService := getGoogleCloudClients(ctx, config)
	bigqueryClient, bitbucketapiClient, githubapiClient, slackapiClient, pubsubapiClient, cockroachdbClient, dockerhubapiClient, builderapiClient, cloudstorageClient, prometheusClient, cloudsourceClient := getClients(ctx, config, encryptedConfig, secretHelper, bqClient, pubsubClient, gcsClient, sourcerepoTokenSource, sourcerepoService)
	estafetteService, rbacService, githubService, bitbucketService, cloudsourceService := getServices(ctx, config, encryptedConfig, secretHelper, bigqueryClient, bitbucketapiClient, githubapiClient, slackapiClient, pubsubapiClient, cockroachdbClient, dockerhubapiClient, builderapiClient, cloudstorageClient, prometheusClient, cloudsourceClient)
	bitbucketHandler, githubHandler, estafetteHandler, rbacHandler, pubsubHandler, slackHandler, cloudsourceHandler := getHandlers(ctx, config, encryptedConfig, secretHelper, bigqueryClient, bitbucketapiClient, githubapiClient, slackapiClient, pubsubapiClient, cockroachdbClient, dockerhubapiClient, builderapiClient, cloudstorageClient, prometheusClient, cloudsourceClient, estafetteService, rbacService, githubService, bitbucketService, cloudsourceService)

	srv := configureGinGonic(config, bitbucketHandler, githubHandler, estafetteHandler, rbacHandler, pubsubHandler, slackHandler, cloudsourceHandler)

	// watch for configmap changes
	foundation.WatchForFileChanges(*configFilePath, func(event fsnotify.Event) {
		log.Info().Msgf("Configmap at %v was updated, refreshing instances...", *configFilePath)

		// refresh config
		newConfig, newEncryptedConfig, _ := getConfig(ctx)

		*config = *newConfig
		*encryptedConfig = *newEncryptedConfig

		// refresh google cloud clients
		newBqClient, newPubsubClient, newGcsClient, newSourcerepoTokenSource, newSourcerepoService := getGoogleCloudClients(ctx, config)

		*bqClient = *newBqClient
		*pubsubClient = *newPubsubClient
		*gcsClient = *newGcsClient
		sourcerepoTokenSource = newSourcerepoTokenSource
		*sourcerepoService = *newSourcerepoService
	})

	// watch for service account key file changes
	foundation.WatchForFileChanges(os.Getenv("GOOGLE_APPLICATION_CREDENTIALS"), func(event fsnotify.Event) {
		log.Info().Msg("Service account key file was updated, refreshing instances...")

		// refresh google cloud clients
		newBqClient, newPubsubClient, newGcsClient, newSourcerepoTokenSource, newSourcerepoService := getGoogleCloudClients(ctx, config)

		*bqClient = *newBqClient
		*pubsubClient = *newPubsubClient
		*gcsClient = *newGcsClient
		sourcerepoTokenSource = newSourcerepoTokenSource
		*sourcerepoService = *newSourcerepoService
	})

	return srv
}

func getConfig(ctx context.Context) (*config.APIConfig, *config.APIConfig, crypt.SecretHelper) {

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

	return config, encryptedConfig, secretHelper
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

func getClients(ctx context.Context, config *config.APIConfig, encryptedConfig *config.APIConfig, secretHelper crypt.SecretHelper, bqClient *stdbigquery.Client, pubsubClient *stdpubsub.Client, gcsClient *stdstorage.Client, sourcerepoTokenSource oauth2.TokenSource, sourcerepoService *stdsourcerepo.Service) (bigqueryClient bigquery.Client, bitbucketapiClient bitbucketapi.Client, githubapiClient githubapi.Client, slackapiClient slackapi.Client, pubsubapiClient pubsubapi.Client, cockroachdbClient cockroachdb.Client, dockerhubapiClient dockerhubapi.Client, builderapiClient builderapi.Client, cloudstorageClient cloudstorage.Client, prometheusClient prometheus.Client, cloudsourceClient cloudsourceapi.Client) {

	log.Debug().Msg("Creating clients...")

	// bigquery client
	bigqueryClient = bigquery.NewClient(config, bqClient)
	bigqueryClient = bigquery.NewTracingClient(bigqueryClient)
	bigqueryClient = bigquery.NewLoggingClient(bigqueryClient)
	bigqueryClient = bigquery.NewMetricsClient(bigqueryClient,
		helpers.NewRequestCounter("bigquery_client"),
		helpers.NewRequestHistogram("bigquery_client"),
	)
	err := bigqueryClient.Init(ctx)
	if err != nil {
		log.Error().Err(err).Msg("Initializing BigQuery tables has failed")
	}

	// bitbucketapi client
	bitbucketapiClient = bitbucketapi.NewClient(config)
	bitbucketapiClient = bitbucketapi.NewTracingClient(bitbucketapiClient)
	bitbucketapiClient = bitbucketapi.NewLoggingClient(bitbucketapiClient)
	bitbucketapiClient = bitbucketapi.NewMetricsClient(bitbucketapiClient,
		helpers.NewRequestCounter("bitbucketapi_client"),
		helpers.NewRequestHistogram("bitbucketapi_client"),
	)

	// githubapi client
	githubapiClient = githubapi.NewClient(config)
	githubapiClient = githubapi.NewTracingClient(githubapiClient)
	githubapiClient = githubapi.NewLoggingClient(githubapiClient)
	githubapiClient = githubapi.NewMetricsClient(githubapiClient,
		helpers.NewRequestCounter("githubapi_client"),
		helpers.NewRequestHistogram("githubapi_client"),
	)

	// slackapi client
	slackapiClient = slackapi.NewClient(config)
	slackapiClient = slackapi.NewTracingClient(slackapiClient)
	slackapiClient = slackapi.NewLoggingClient(slackapiClient)
	slackapiClient = slackapi.NewMetricsClient(slackapiClient,
		helpers.NewRequestCounter("slackapi_client"),
		helpers.NewRequestHistogram("slackapi_client"),
	)

	// pubsubapi client
	pubsubapiClient = pubsubapi.NewClient(config, pubsubClient)
	pubsubapiClient = pubsubapi.NewTracingClient(pubsubapiClient)
	pubsubapiClient = pubsubapi.NewLoggingClient(pubsubapiClient)
	pubsubapiClient = pubsubapi.NewMetricsClient(pubsubapiClient,
		helpers.NewRequestCounter("pubsubapi_client"),
		helpers.NewRequestHistogram("pubsubapi_client"),
	)

	// cockroachdb client
	cockroachdbClient = cockroachdb.NewClient(config)
	cockroachdbClient = cockroachdb.NewTracingClient(cockroachdbClient)
	cockroachdbClient = cockroachdb.NewLoggingClient(cockroachdbClient)
	cockroachdbClient = cockroachdb.NewMetricsClient(cockroachdbClient,
		helpers.NewRequestCounter("cockroachdb_client"),
		helpers.NewRequestHistogram("cockroachdb_client"),
	)
	err = cockroachdbClient.Connect(ctx)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed connecting to CockroachDB")
	}

	// dockerhubapi client
	dockerhubapiClient = dockerhubapi.NewClient()
	dockerhubapiClient = dockerhubapi.NewTracingClient(dockerhubapiClient)
	dockerhubapiClient = dockerhubapi.NewLoggingClient(dockerhubapiClient)
	dockerhubapiClient = dockerhubapi.NewMetricsClient(dockerhubapiClient,
		helpers.NewRequestCounter("dockerhubapi_client"),
		helpers.NewRequestHistogram("dockerhubapi_client"),
	)

	// builderapi client
	kubeClient, err := k8s.NewInClusterClient()
	if err != nil {
		log.Fatal().Err(err).Msg("Creating kubernetes client failed")
	}
	builderapiClient = builderapi.NewClient(config, encryptedConfig, secretHelper, kubeClient, dockerhubapiClient)
	builderapiClient = builderapi.NewTracingClient(builderapiClient)
	builderapiClient = builderapi.NewLoggingClient(builderapiClient)
	builderapiClient = builderapi.NewMetricsClient(builderapiClient,
		helpers.NewRequestCounter("builderapi_client"),
		helpers.NewRequestHistogram("builderapi_client"),
	)

	// cloudstorage client
	cloudstorageClient = cloudstorage.NewClient(config, gcsClient)
	cloudstorageClient = cloudstorage.NewTracingClient(cloudstorageClient)
	cloudstorageClient = cloudstorage.NewLoggingClient(cloudstorageClient)
	cloudstorageClient = cloudstorage.NewMetricsClient(cloudstorageClient,
		helpers.NewRequestCounter("cloudstorage_client"),
		helpers.NewRequestHistogram("cloudstorage_client"),
	)

	// prometheus client
	prometheusClient = prometheus.NewClient(config)
	prometheusClient = prometheus.NewTracingClient(prometheusClient)
	prometheusClient = prometheus.NewLoggingClient(prometheusClient)
	prometheusClient = prometheus.NewMetricsClient(prometheusClient,
		helpers.NewRequestCounter("prometheus_client"),
		helpers.NewRequestHistogram("prometheus_client"),
	)

	// cloudsourceapi client
	cloudsourceClient, err = cloudsourceapi.NewClient(sourcerepoTokenSource, sourcerepoService)
	if err != nil {
		log.Error().Err(err).Msg("Creating new client for Cloud Source has failed")
	}
	cloudsourceClient = cloudsourceapi.NewTracingClient(cloudsourceClient)
	cloudsourceClient = cloudsourceapi.NewLoggingClient(cloudsourceClient)
	cloudsourceClient = cloudsourceapi.NewMetricsClient(cloudsourceClient,
		helpers.NewRequestCounter("cloudsource_client"),
		helpers.NewRequestHistogram("cloudsource_client"),
	)

	return
}

func getServices(ctx context.Context, config *config.APIConfig, encryptedConfig *config.APIConfig, secretHelper crypt.SecretHelper, bigqueryClient bigquery.Client, bitbucketapiClient bitbucketapi.Client, githubapiClient githubapi.Client, slackapiClient slackapi.Client, pubsubapiClient pubsubapi.Client, cockroachdbClient cockroachdb.Client, dockerhubapiClient dockerhubapi.Client, builderapiClient builderapi.Client, cloudstorageClient cloudstorage.Client, prometheusClient prometheus.Client, cloudsourceClient cloudsourceapi.Client) (estafetteService estafette.Service, rbacService rbac.Service, githubService github.Service, bitbucketService bitbucket.Service, cloudsourceService cloudsource.Service) {

	log.Debug().Msg("Creating services...")

	// estafette service
	estafetteService = estafette.NewService(config, cockroachdbClient, prometheusClient, cloudstorageClient, builderapiClient, githubapiClient.JobVarsFunc(ctx), bitbucketapiClient.JobVarsFunc(ctx), cloudsourceClient.JobVarsFunc(ctx))
	estafetteService = estafette.NewTracingService(estafetteService)
	estafetteService = estafette.NewLoggingService(estafetteService)
	estafetteService = estafette.NewMetricsService(estafetteService,
		helpers.NewRequestCounter("estafette_service"),
		helpers.NewRequestHistogram("estafette_service"),
	)

	// rbac service
	rbacService = rbac.NewService(config, cockroachdbClient)
	rbacService = rbac.NewTracingService(rbacService)
	rbacService = rbac.NewLoggingService(rbacService)
	rbacService = rbac.NewMetricsService(rbacService,
		helpers.NewRequestCounter("rbac_service"),
		helpers.NewRequestHistogram("rbac_service"),
	)

	// github service
	githubService = github.NewService(config, githubapiClient, pubsubapiClient, estafetteService)
	githubService = github.NewTracingService(githubService)
	githubService = github.NewLoggingService(githubService)
	githubService = github.NewMetricsService(githubService,
		helpers.NewRequestCounter("github_service"),
		helpers.NewRequestHistogram("github_service"),
	)

	// bitbucket service
	bitbucketService = bitbucket.NewService(config, bitbucketapiClient, pubsubapiClient, estafetteService)
	bitbucketService = bitbucket.NewTracingService(bitbucketService)
	bitbucketService = bitbucket.NewLoggingService(bitbucketService)
	bitbucketService = bitbucket.NewMetricsService(bitbucketService,
		helpers.NewRequestCounter("bitbucket_service"),
		helpers.NewRequestHistogram("bitbucket_service"),
	)

	// cloudsource service
	cloudsourceService = cloudsource.NewService(config, cloudsourceClient, pubsubapiClient, estafetteService)
	cloudsourceService = cloudsource.NewTracingService(cloudsourceService)
	cloudsourceService = cloudsource.NewLoggingService(cloudsourceService)
	cloudsourceService = cloudsource.NewMetricsService(cloudsourceService,
		helpers.NewRequestCounter("cloudsource_service"),
		helpers.NewRequestHistogram("cloudsource_service"),
	)

	return
}

func getHandlers(ctx context.Context, config *config.APIConfig, encryptedConfig *config.APIConfig, secretHelper crypt.SecretHelper, bigqueryClient bigquery.Client, bitbucketapiClient bitbucketapi.Client, githubapiClient githubapi.Client, slackapiClient slackapi.Client, pubsubapiClient pubsubapi.Client, cockroachdbClient cockroachdb.Client, dockerhubapiClient dockerhubapi.Client, builderapiClient builderapi.Client, cloudstorageClient cloudstorage.Client, prometheusClient prometheus.Client, cloudsourceClient cloudsourceapi.Client, estafetteService estafette.Service, rbacService rbac.Service, githubService github.Service, bitbucketService bitbucket.Service, cloudsourceService cloudsource.Service) (bitbucketHandler bitbucket.Handler, githubHandler github.Handler, estafetteHandler estafette.Handler, rbacHandler rbac.Handler, pubsubHandler pubsub.Handler, slackHandler slack.Handler, cloudsourceHandler cloudsource.Handler) {

	log.Debug().Msg("Creating http handlers...")

	warningHelper := helpers.NewWarningHelper(secretHelper)

	// transport
	bitbucketHandler = bitbucket.NewHandler(bitbucketService)
	githubHandler = github.NewHandler(githubService)
	estafetteHandler = estafette.NewHandler(*configFilePath, config, encryptedConfig, cockroachdbClient, cloudstorageClient, builderapiClient, estafetteService, warningHelper, secretHelper, githubapiClient.JobVarsFunc(ctx), bitbucketapiClient.JobVarsFunc(ctx), cloudsourceClient.JobVarsFunc(ctx))
	rbacHandler = rbac.NewHandler(config, rbacService, cockroachdbClient)
	pubsubHandler = pubsub.NewHandler(pubsubapiClient, estafetteService)
	slackHandler = slack.NewHandler(secretHelper, config, slackapiClient, cockroachdbClient, estafetteService, githubapiClient.JobVarsFunc(ctx), bitbucketapiClient.JobVarsFunc(ctx))
	cloudsourceHandler = cloudsource.NewHandler(pubsubapiClient, cloudsourceService)

	return
}

func configureGinGonic(config *config.APIConfig, bitbucketHandler bitbucket.Handler, githubHandler github.Handler, estafetteHandler estafette.Handler, rbacHandler rbac.Handler, pubsubHandler pubsub.Handler, slackHandler slack.Handler, cloudsourceHandler cloudsource.Handler) *http.Server {

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

	// middleware to handle auth for different endpoints
	log.Debug().Msg("Adding auth middleware...")
	authMiddleware := auth.NewAuthMiddleware(config)
	jwtMiddleware, err := authMiddleware.GinJWTMiddleware(rbacHandler.HandleLoginProviderAuthenticator())
	if err != nil {
		log.Fatal().Err(err).Msg("Failed creating JWT middleware")
	}
	preZippedJWTMiddlewareRoutes := router.Group("/", jwtMiddleware.MiddlewareFunc())

	// Gzip and logging middleware
	log.Debug().Msg("Adding gzip middleware...")
	routes := router.Group("/", gzip.Gzip(gzip.BestSpeed))

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

	// public routes for logging in
	routes.GET("/api/auth/providers", rbacHandler.GetProviders)
	routes.GET("/api/auth/login/:provider", rbacHandler.LoginProvider)
	routes.GET("/api/auth/logout", jwtMiddleware.LogoutHandler)
	routes.GET("/api/auth/handle/:provider", jwtMiddleware.LoginHandler)

	// routes that require to be logged in and have a valid jwt
	jwtMiddlewareRoutes := routes.Group("/", jwtMiddleware.MiddlewareFunc())
	{
		// require claims
		jwtMiddlewareRoutes.GET("/api/users/me", rbacHandler.GetLoggedInUser)
		jwtMiddlewareRoutes.GET("/api/update-computed-tables", estafetteHandler.UpdateComputedTables)
		jwtMiddlewareRoutes.POST("/api/pipelines/:source/:owner/:repo/builds", estafetteHandler.CreatePipelineBuild)
		jwtMiddlewareRoutes.POST("/api/pipelines/:source/:owner/:repo/releases", estafetteHandler.CreatePipelineRelease)
		jwtMiddlewareRoutes.DELETE("/api/pipelines/:source/:owner/:repo/builds/:revisionOrId", estafetteHandler.CancelPipelineBuild)
		jwtMiddlewareRoutes.DELETE("/api/pipelines/:source/:owner/:repo/releases/:id", estafetteHandler.CancelPipelineRelease)

		jwtMiddlewareRoutes.GET("/api/users", rbacHandler.GetUsers)
		jwtMiddlewareRoutes.GET("/api/groups", rbacHandler.GetGroups)
		jwtMiddlewareRoutes.GET("/api/organizations", rbacHandler.GetOrganizations)

		// do not require claims
		jwtMiddlewareRoutes.GET("/api/config", estafetteHandler.GetConfig)
		jwtMiddlewareRoutes.GET("/api/config/credentials", estafetteHandler.GetConfigCredentials)
		jwtMiddlewareRoutes.GET("/api/config/trustedimages", estafetteHandler.GetConfigTrustedImages)
		jwtMiddlewareRoutes.GET("/api/pipelines", estafetteHandler.GetPipelines)
		jwtMiddlewareRoutes.GET("/api/pipelines/:source/:owner/:repo", estafetteHandler.GetPipeline)
		jwtMiddlewareRoutes.GET("/api/pipelines/:source/:owner/:repo/recentbuilds", estafetteHandler.GetPipelineRecentBuilds)
		jwtMiddlewareRoutes.GET("/api/pipelines/:source/:owner/:repo/builds", estafetteHandler.GetPipelineBuilds)
		jwtMiddlewareRoutes.GET("/api/pipelines/:source/:owner/:repo/builds/:revisionOrId", estafetteHandler.GetPipelineBuild)
		jwtMiddlewareRoutes.GET("/api/pipelines/:source/:owner/:repo/builds/:revisionOrId/warnings", estafetteHandler.GetPipelineBuildWarnings)
		jwtMiddlewareRoutes.GET("/api/pipelines/:source/:owner/:repo/releases", estafetteHandler.GetPipelineReleases)
		jwtMiddlewareRoutes.GET("/api/pipelines/:source/:owner/:repo/releases/:id", estafetteHandler.GetPipelineRelease)
		jwtMiddlewareRoutes.GET("/api/pipelines/:source/:owner/:repo/stats/buildsdurations", estafetteHandler.GetPipelineStatsBuildsDurations)
		jwtMiddlewareRoutes.GET("/api/pipelines/:source/:owner/:repo/stats/releasesdurations", estafetteHandler.GetPipelineStatsReleasesDurations)
		jwtMiddlewareRoutes.GET("/api/pipelines/:source/:owner/:repo/stats/buildscpu", estafetteHandler.GetPipelineStatsBuildsCPUUsageMeasurements)
		jwtMiddlewareRoutes.GET("/api/pipelines/:source/:owner/:repo/stats/releasescpu", estafetteHandler.GetPipelineStatsReleasesCPUUsageMeasurements)
		jwtMiddlewareRoutes.GET("/api/pipelines/:source/:owner/:repo/stats/buildsmemory", estafetteHandler.GetPipelineStatsBuildsMemoryUsageMeasurements)
		jwtMiddlewareRoutes.GET("/api/pipelines/:source/:owner/:repo/stats/releasesmemory", estafetteHandler.GetPipelineStatsReleasesMemoryUsageMeasurements)
		jwtMiddlewareRoutes.GET("/api/pipelines/:source/:owner/:repo/warnings", estafetteHandler.GetPipelineWarnings)
		jwtMiddlewareRoutes.GET("/api/catalog/filters", estafetteHandler.GetCatalogFilters)
		jwtMiddlewareRoutes.GET("/api/catalog/filtervalues", estafetteHandler.GetCatalogFilterValues)
		jwtMiddlewareRoutes.GET("/api/stats/pipelinescount", estafetteHandler.GetStatsPipelinesCount)
		jwtMiddlewareRoutes.GET("/api/stats/buildscount", estafetteHandler.GetStatsBuildsCount)
		jwtMiddlewareRoutes.GET("/api/stats/releasescount", estafetteHandler.GetStatsReleasesCount)
		jwtMiddlewareRoutes.GET("/api/stats/buildsduration", estafetteHandler.GetStatsBuildsDuration)
		jwtMiddlewareRoutes.GET("/api/stats/buildsadoption", estafetteHandler.GetStatsBuildsAdoption)
		jwtMiddlewareRoutes.GET("/api/stats/releasesadoption", estafetteHandler.GetStatsReleasesAdoption)
		jwtMiddlewareRoutes.GET("/api/stats/mostbuilds", estafetteHandler.GetStatsMostBuilds)
		jwtMiddlewareRoutes.GET("/api/stats/mostreleases", estafetteHandler.GetStatsMostReleases)
		jwtMiddlewareRoutes.GET("/api/manifest/templates", estafetteHandler.GetManifestTemplates)
		jwtMiddlewareRoutes.POST("/api/manifest/generate", estafetteHandler.GenerateManifest)
		jwtMiddlewareRoutes.POST("/api/manifest/validate", estafetteHandler.ValidateManifest)
		jwtMiddlewareRoutes.POST("/api/manifest/encrypt", estafetteHandler.EncryptSecret)
		jwtMiddlewareRoutes.GET("/api/labels/frequent", estafetteHandler.GetFrequentLabels)

		// communication from build/release jobs back to api
		jwtMiddlewareRoutes.POST("/api/commands", estafetteHandler.Commands)
		jwtMiddlewareRoutes.POST("/api/pipelines/:source/:owner/:repo/builds/:revisionOrId/logs", estafetteHandler.PostPipelineBuildLogs)
		jwtMiddlewareRoutes.POST("/api/pipelines/:source/:owner/:repo/releases/:id/logs", estafetteHandler.PostPipelineReleaseLogs)

		// do not require claims and avoid re-zipping
		preZippedJWTMiddlewareRoutes.GET("/api/pipelines/:source/:owner/:repo/builds/:revisionOrId/logs", estafetteHandler.GetPipelineBuildLogs)
		preZippedJWTMiddlewareRoutes.GET("/api/pipelines/:source/:owner/:repo/builds/:revisionOrId/logs/tail", estafetteHandler.TailPipelineBuildLogs)
		preZippedJWTMiddlewareRoutes.GET("/api/pipelines/:source/:owner/:repo/builds/:revisionOrId/logs.stream", estafetteHandler.TailPipelineBuildLogs)
		preZippedJWTMiddlewareRoutes.GET("/api/pipelines/:source/:owner/:repo/releases/:id/logs", estafetteHandler.GetPipelineReleaseLogs)
		preZippedJWTMiddlewareRoutes.GET("/api/pipelines/:source/:owner/:repo/releases/:id/logs/tail", estafetteHandler.TailPipelineReleaseLogs)
		preZippedJWTMiddlewareRoutes.GET("/api/pipelines/:source/:owner/:repo/releases/:id/logs.stream", estafetteHandler.TailPipelineReleaseLogs)
	}

	// TODO move behind jwt middleware by letting build/release jobs and cron event sender use valid jwt's
	// api key protected endpoints
	apiKeyAuthorizedRoutes := routes.Group("/", authMiddleware.APIKeyMiddlewareFunc())
	{
		apiKeyAuthorizedRoutes.POST("/api/integrations/cron/events", estafetteHandler.PostCronEvent)
		apiKeyAuthorizedRoutes.GET("/api/copylogstocloudstorage/:source/:owner/:repo", estafetteHandler.CopyLogsToCloudStorage)
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
