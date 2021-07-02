package main

import (
	"context"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"sync"
	"time"

	stdbigquery "cloud.google.com/go/bigquery"
	stdpubsub "cloud.google.com/go/pubsub"
	stdstorage "cloud.google.com/go/storage"
	"github.com/alecthomas/kingpin"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"

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
	"google.golang.org/api/option"
	"google.golang.org/api/sourcerepo/v1"
	stdsourcerepo "google.golang.org/api/sourcerepo/v1"

	"github.com/estafette/estafette-ci-api/api"

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
	"github.com/estafette/estafette-ci-api/services/catalog"
	"github.com/estafette/estafette-ci-api/services/cloudsource"
	"github.com/estafette/estafette-ci-api/services/estafette"
	"github.com/estafette/estafette-ci-api/services/github"
	"github.com/estafette/estafette-ci-api/services/pubsub"
	"github.com/estafette/estafette-ci-api/services/queue"
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
)

var (
	// flags
	apiAddress                   = kingpin.Flag("api-listen-address", "The address to listen on for api HTTP requests.").Default(":5000").String()
	configFilePath               = kingpin.Flag("config-file-path", "The path to yaml config file configuring this application.").Default("/configs/config.yaml").OverrideDefaultFromEnvar("CONFIG_FILE_PATH").String()
	templatesPath                = kingpin.Flag("templates-path", "The path to the manifest templates being used by the 'Create' functionality.").Default("/templates").OverrideDefaultFromEnvar("TEMPLATES_PATH").String()
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

	ctx := foundation.InitCancellationContext(context.Background())

	// start prometheus
	foundation.InitMetricsWithPort(9001)

	// handle api requests
	srv, queueService := initRequestHandlers(ctx, stop, wg)
	defer queueService.CloseConnection(ctx)

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

func initRequestHandlers(ctx context.Context, stopChannel <-chan struct{}, waitGroup *sync.WaitGroup) (*http.Server, queue.Service) {

	config, encryptedConfig, secretHelper := getConfig(ctx)
	gitEventTopic := getTopics(ctx, stopChannel)
	bqClient, pubsubClient, gcsClient, sourcerepoTokenSource, sourcerepoService := getGoogleCloudClients(ctx, config)
	bigqueryClient, bitbucketapiClient, githubapiClient, slackapiClient, pubsubapiClient, cockroachdbClient, dockerhubapiClient, builderapiClient, cloudstorageClient, prometheusClient, cloudsourceClient := getClients(ctx, config, encryptedConfig, secretHelper, bqClient, pubsubClient, gcsClient, sourcerepoTokenSource, sourcerepoService)
	estafetteService, queueService, rbacService, githubService, bitbucketService, cloudsourceService, catalogService := getServices(ctx, config, encryptedConfig, secretHelper, bigqueryClient, bitbucketapiClient, githubapiClient, slackapiClient, pubsubapiClient, cockroachdbClient, dockerhubapiClient, builderapiClient, cloudstorageClient, prometheusClient, cloudsourceClient, gitEventTopic)
	bitbucketHandler, githubHandler, estafetteHandler, rbacHandler, pubsubHandler, slackHandler, cloudsourceHandler, catalogHandler := getHandlers(ctx, config, encryptedConfig, secretHelper, bigqueryClient, bitbucketapiClient, githubapiClient, slackapiClient, pubsubapiClient, cockroachdbClient, dockerhubapiClient, builderapiClient, cloudstorageClient, prometheusClient, cloudsourceClient, estafetteService, rbacService, githubService, bitbucketService, cloudsourceService, catalogService)

	err := queueService.CreateConnection(ctx)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed creating connection to queue")
	}
	err = queueService.InitSubscriptions(ctx)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed initializing queue subscriptions")
	}

	subscribeToTopics(ctx, gitEventTopic, estafetteService)

	srv := configureGinGonic(config, bitbucketHandler, githubHandler, estafetteHandler, rbacHandler, pubsubHandler, slackHandler, cloudsourceHandler, catalogHandler)

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

	return srv, queueService
}

func getTopics(ctx context.Context, stopChannel <-chan struct{}) (gitEventTopic *api.EventTopic) {
	gitEventTopic = api.NewEventTopic("push events")

	// close channels when stopChannel is signaled
	go func(stopChannel <-chan struct{}) {
		<-stopChannel
		gitEventTopic.Close()
	}(stopChannel)

	return
}

func subscribeToTopics(ctx context.Context, gitEventTopic *api.EventTopic, estafetteService estafette.Service) {
	go estafetteService.SubscribeToEventTopic(ctx, gitEventTopic)
}

func getConfig(ctx context.Context) (*api.APIConfig, *api.APIConfig, crypt.SecretHelper) {

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

	configReader := api.NewConfigReader(secretHelper)

	// await for config file to be present, due to git-sync sidecar startup it can take some time
	for !foundation.FileExists(*configFilePath) {
		log.Debug().Msg("Sleeping for 5 seconds while config file is synced...")
		time.Sleep(5 * time.Second)
	}

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

func getGoogleCloudClients(ctx context.Context, config *api.APIConfig) (bqClient *stdbigquery.Client, pubsubClient *stdpubsub.Client, gcsClient *stdstorage.Client, tokenSource oauth2.TokenSource, sourcerepoService *stdsourcerepo.Service) {

	log.Debug().Msg("Creating Google Cloud clients...")

	var err error

	if config != nil && config.Integrations != nil && config.Integrations.BigQuery != nil && config.Integrations.BigQuery.Enable {
		bqClient, err = stdbigquery.NewClient(ctx, config.Integrations.BigQuery.ProjectID)
		if err != nil {
			log.Fatal().Err(err).Msg("Creating new BigQueryClient has failed")
		}
	}

	if config != nil && config.Integrations != nil && config.Integrations.Pubsub != nil && config.Integrations.Pubsub.Enable {
		pubsubClient, err = stdpubsub.NewClient(ctx, config.Integrations.Pubsub.DefaultProject)
		if err != nil {
			log.Fatal().Err(err).Msg("Creating google pubsub client has failed")
		}
	}

	if config != nil && config.Integrations != nil && config.Integrations.CloudStorage != nil && config.Integrations.CloudStorage.Enable {
		gcsClient, err = stdstorage.NewClient(ctx)
		if err != nil {
			log.Fatal().Err(err).Msg("Creating google cloud storage client has failed")
		}
	}

	if config != nil && config.Integrations != nil && config.Integrations.CloudSource != nil && config.Integrations.CloudSource.Enable {
		tokenSource, err = google.DefaultTokenSource(ctx, sourcerepo.CloudPlatformScope)
		if err != nil {
			log.Fatal().Err(err).Msg("Creating google cloud token source has failed")
		}
		sourcerepoService, err = stdsourcerepo.NewService(ctx, option.WithHTTPClient(oauth2.NewClient(ctx, tokenSource)))
		if err != nil {
			log.Fatal().Err(err).Msg("Creating google cloud source repo service has failed")
		}
	}

	return bqClient, pubsubClient, gcsClient, tokenSource, sourcerepoService
}

func getClients(ctx context.Context, config *api.APIConfig, encryptedConfig *api.APIConfig, secretHelper crypt.SecretHelper, bqClient *stdbigquery.Client, pubsubClient *stdpubsub.Client, gcsClient *stdstorage.Client, sourcerepoTokenSource oauth2.TokenSource, sourcerepoService *stdsourcerepo.Service) (bigqueryClient bigquery.Client, bitbucketapiClient bitbucketapi.Client, githubapiClient githubapi.Client, slackapiClient slackapi.Client, pubsubapiClient pubsubapi.Client, cockroachdbClient cockroachdb.Client, dockerhubapiClient dockerhubapi.Client, builderapiClient builderapi.Client, cloudstorageClient cloudstorage.Client, prometheusClient prometheus.Client, cloudsourceClient cloudsourceapi.Client) {

	log.Debug().Msg("Creating clients...")

	// bigquery client
	bigqueryClient = bigquery.NewClient(config, bqClient)
	bigqueryClient = bigquery.NewTracingClient(bigqueryClient)
	bigqueryClient = bigquery.NewLoggingClient(bigqueryClient)
	bigqueryClient = bigquery.NewMetricsClient(bigqueryClient,
		api.NewRequestCounter("bigquery_client"),
		api.NewRequestHistogram("bigquery_client"),
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
		api.NewRequestCounter("bitbucketapi_client"),
		api.NewRequestHistogram("bitbucketapi_client"),
	)

	// githubapi client
	githubapiClient = githubapi.NewClient(config)
	githubapiClient = githubapi.NewTracingClient(githubapiClient)
	githubapiClient = githubapi.NewLoggingClient(githubapiClient)
	githubapiClient = githubapi.NewMetricsClient(githubapiClient,
		api.NewRequestCounter("githubapi_client"),
		api.NewRequestHistogram("githubapi_client"),
	)

	// slackapi client
	slackapiClient = slackapi.NewClient(config)
	slackapiClient = slackapi.NewTracingClient(slackapiClient)
	slackapiClient = slackapi.NewLoggingClient(slackapiClient)
	slackapiClient = slackapi.NewMetricsClient(slackapiClient,
		api.NewRequestCounter("slackapi_client"),
		api.NewRequestHistogram("slackapi_client"),
	)

	// pubsubapi client
	pubsubapiClient = pubsubapi.NewClient(config, pubsubClient)
	pubsubapiClient = pubsubapi.NewTracingClient(pubsubapiClient)
	pubsubapiClient = pubsubapi.NewLoggingClient(pubsubapiClient)
	pubsubapiClient = pubsubapi.NewMetricsClient(pubsubapiClient,
		api.NewRequestCounter("pubsubapi_client"),
		api.NewRequestHistogram("pubsubapi_client"),
	)

	// cockroachdb client
	cockroachdbClient = cockroachdb.NewClient(config)
	cockroachdbClient = cockroachdb.NewTracingClient(cockroachdbClient)
	cockroachdbClient = cockroachdb.NewLoggingClient(cockroachdbClient)
	cockroachdbClient = cockroachdb.NewMetricsClient(cockroachdbClient,
		api.NewRequestCounter("cockroachdb_client"),
		api.NewRequestHistogram("cockroachdb_client"),
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
		api.NewRequestCounter("dockerhubapi_client"),
		api.NewRequestHistogram("dockerhubapi_client"),
	)

	// builderapi client
	// creates the in-cluster config
	kubeClientConfig, err := rest.InClusterConfig()
	if err != nil {
		log.Fatal().Err(err).Msg("Failed getting in-cluster kubernetes config")
	}
	// creates the clientset
	kubeClientset, err := kubernetes.NewForConfig(kubeClientConfig)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed creating kubernetes clientset")
	}

	// kubeClient, err := k8s.NewInClusterClient()
	if err != nil {
		log.Fatal().Err(err).Msg("Creating kubernetes client failed")
	}
	builderapiClient = builderapi.NewClient(config, encryptedConfig, secretHelper, kubeClientset, dockerhubapiClient)
	builderapiClient = builderapi.NewTracingClient(builderapiClient)
	builderapiClient = builderapi.NewLoggingClient(builderapiClient)
	builderapiClient = builderapi.NewMetricsClient(builderapiClient,
		api.NewRequestCounter("builderapi_client"),
		api.NewRequestHistogram("builderapi_client"),
	)

	// cloudstorage client
	cloudstorageClient = cloudstorage.NewClient(config, gcsClient)
	cloudstorageClient = cloudstorage.NewTracingClient(cloudstorageClient)
	cloudstorageClient = cloudstorage.NewLoggingClient(cloudstorageClient)
	cloudstorageClient = cloudstorage.NewMetricsClient(cloudstorageClient,
		api.NewRequestCounter("cloudstorage_client"),
		api.NewRequestHistogram("cloudstorage_client"),
	)

	// prometheus client
	prometheusClient = prometheus.NewClient(config)
	prometheusClient = prometheus.NewTracingClient(prometheusClient)
	prometheusClient = prometheus.NewLoggingClient(prometheusClient)
	prometheusClient = prometheus.NewMetricsClient(prometheusClient,
		api.NewRequestCounter("prometheus_client"),
		api.NewRequestHistogram("prometheus_client"),
	)

	// cloudsourceapi client
	cloudsourceClient = cloudsourceapi.NewClient(config, sourcerepoTokenSource, sourcerepoService)
	cloudsourceClient = cloudsourceapi.NewTracingClient(cloudsourceClient)
	cloudsourceClient = cloudsourceapi.NewLoggingClient(cloudsourceClient)
	cloudsourceClient = cloudsourceapi.NewMetricsClient(cloudsourceClient,
		api.NewRequestCounter("cloudsource_client"),
		api.NewRequestHistogram("cloudsource_client"),
	)

	return
}

func getServices(ctx context.Context, config *api.APIConfig, encryptedConfig *api.APIConfig, secretHelper crypt.SecretHelper, bigqueryClient bigquery.Client, bitbucketapiClient bitbucketapi.Client, githubapiClient githubapi.Client, slackapiClient slackapi.Client, pubsubapiClient pubsubapi.Client, cockroachdbClient cockroachdb.Client, dockerhubapiClient dockerhubapi.Client, builderapiClient builderapi.Client, cloudstorageClient cloudstorage.Client, prometheusClient prometheus.Client, cloudsourceClient cloudsourceapi.Client, gitEventTopic *api.EventTopic) (estafetteService estafette.Service, queueService queue.Service, rbacService rbac.Service, githubService github.Service, bitbucketService bitbucket.Service, cloudsourceService cloudsource.Service, catalogService catalog.Service) {

	log.Debug().Msg("Creating services...")

	// estafette service
	estafetteService = estafette.NewService(config, cockroachdbClient, secretHelper, prometheusClient, cloudstorageClient, builderapiClient, githubapiClient.JobVarsFunc(ctx), bitbucketapiClient.JobVarsFunc(ctx), cloudsourceClient.JobVarsFunc(ctx))
	estafetteService = estafette.NewTracingService(estafetteService)
	estafetteService = estafette.NewLoggingService(estafetteService)
	estafetteService = estafette.NewMetricsService(estafetteService,
		api.NewRequestCounter("estafette_service"),
		api.NewRequestHistogram("estafette_service"),
	)

	// queue service
	queueService = queue.NewService(config, estafetteService)

	// rbac service
	rbacService = rbac.NewService(config, cockroachdbClient)
	rbacService = rbac.NewTracingService(rbacService)
	rbacService = rbac.NewLoggingService(rbacService)
	rbacService = rbac.NewMetricsService(rbacService,
		api.NewRequestCounter("rbac_service"),
		api.NewRequestHistogram("rbac_service"),
	)

	// github service
	githubService = github.NewService(config, githubapiClient, pubsubapiClient, estafetteService, queueService)
	githubService = github.NewTracingService(githubService)
	githubService = github.NewLoggingService(githubService)
	githubService = github.NewMetricsService(githubService,
		api.NewRequestCounter("github_service"),
		api.NewRequestHistogram("github_service"),
	)

	// bitbucket service
	bitbucketService = bitbucket.NewService(config, bitbucketapiClient, pubsubapiClient, estafetteService, queueService)
	bitbucketService = bitbucket.NewTracingService(bitbucketService)
	bitbucketService = bitbucket.NewLoggingService(bitbucketService)
	bitbucketService = bitbucket.NewMetricsService(bitbucketService,
		api.NewRequestCounter("bitbucket_service"),
		api.NewRequestHistogram("bitbucket_service"),
	)

	// cloudsource service
	cloudsourceService = cloudsource.NewService(config, cloudsourceClient, pubsubapiClient, estafetteService, queueService)
	cloudsourceService = cloudsource.NewTracingService(cloudsourceService)
	cloudsourceService = cloudsource.NewLoggingService(cloudsourceService)
	cloudsourceService = cloudsource.NewMetricsService(cloudsourceService,
		api.NewRequestCounter("cloudsource_service"),
		api.NewRequestHistogram("cloudsource_service"),
	)

	// catalog service
	catalogService = catalog.NewService(config, cockroachdbClient)
	catalogService = catalog.NewTracingService(catalogService)
	catalogService = catalog.NewLoggingService(catalogService)
	catalogService = catalog.NewMetricsService(catalogService,
		api.NewRequestCounter("catalog_service"),
		api.NewRequestHistogram("catalog_service"),
	)

	return
}

func getHandlers(ctx context.Context, config *api.APIConfig, encryptedConfig *api.APIConfig, secretHelper crypt.SecretHelper, bigqueryClient bigquery.Client, bitbucketapiClient bitbucketapi.Client, githubapiClient githubapi.Client, slackapiClient slackapi.Client, pubsubapiClient pubsubapi.Client, cockroachdbClient cockroachdb.Client, dockerhubapiClient dockerhubapi.Client, builderapiClient builderapi.Client, cloudstorageClient cloudstorage.Client, prometheusClient prometheus.Client, cloudsourceClient cloudsourceapi.Client, estafetteService estafette.Service, rbacService rbac.Service, githubService github.Service, bitbucketService bitbucket.Service, cloudsourceService cloudsource.Service, catalogService catalog.Service) (bitbucketHandler bitbucket.Handler, githubHandler github.Handler, estafetteHandler estafette.Handler, rbacHandler rbac.Handler, pubsubHandler pubsub.Handler, slackHandler slack.Handler, cloudsourceHandler cloudsource.Handler, catalogHandler catalog.Handler) {

	log.Debug().Msg("Creating http handlers...")

	warningHelper := api.NewWarningHelper(secretHelper)

	// transport
	bitbucketHandler = bitbucket.NewHandler(bitbucketService)
	githubHandler = github.NewHandler(githubService)
	estafetteHandler = estafette.NewHandler(*configFilePath, *templatesPath, config, encryptedConfig, cockroachdbClient, cloudstorageClient, builderapiClient, estafetteService, warningHelper, secretHelper, githubapiClient.JobVarsFunc(ctx), bitbucketapiClient.JobVarsFunc(ctx), cloudsourceClient.JobVarsFunc(ctx))
	rbacHandler = rbac.NewHandler(config, rbacService, cockroachdbClient)
	pubsubHandler = pubsub.NewHandler(pubsubapiClient, estafetteService)
	slackHandler = slack.NewHandler(secretHelper, config, slackapiClient, cockroachdbClient, estafetteService, githubapiClient.JobVarsFunc(ctx), bitbucketapiClient.JobVarsFunc(ctx))
	cloudsourceHandler = cloudsource.NewHandler(pubsubapiClient, cloudsourceService)
	catalogHandler = catalog.NewHandler(config, catalogService, cockroachdbClient)

	return
}

func configureGinGonic(config *api.APIConfig, bitbucketHandler bitbucket.Handler, githubHandler github.Handler, estafetteHandler estafette.Handler, rbacHandler rbac.Handler, pubsubHandler pubsub.Handler, slackHandler slack.Handler, cloudsourceHandler cloudsource.Handler, catalogHandler catalog.Handler) *http.Server {

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
	router.Use(api.OpenTracingMiddleware())

	// middleware to handle auth for different endpoints
	log.Debug().Msg("Adding auth middleware...")
	authMiddleware := api.NewAuthMiddleware(config)
	jwtMiddleware, err := authMiddleware.GinJWTMiddleware(rbacHandler.HandleOAuthLoginProviderAuthenticator())
	if err != nil {
		log.Fatal().Err(err).Msg("Failed creating JWT middleware")
	}
	clientLoginJWTMiddleware, err := authMiddleware.GinJWTMiddlewareForClientLogin(rbacHandler.HandleClientLoginProviderAuthenticator())
	if err != nil {
		log.Fatal().Err(err).Msg("Failed creating JWT middleware for client login")
	}
	impersonateJWTMiddleware, err := authMiddleware.GinJWTMiddleware(rbacHandler.HandleImpersonateAuthenticator())
	if err != nil {
		log.Fatal().Err(err).Msg("Failed creating JWT middleware for user impersonation")
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
	routes.POST("/api/auth/client/login", clientLoginJWTMiddleware.LoginHandler)
	routes.POST("/api/auth/client/logout", clientLoginJWTMiddleware.LogoutHandler)

	// routes that require to be logged in and have a valid jwt
	jwtMiddlewareRoutes := routes.Group("/", jwtMiddleware.MiddlewareFunc())
	{
		// logged in user endpoints
		jwtMiddlewareRoutes.GET("/api/me", rbacHandler.GetLoggedInUser)

		// actions
		jwtMiddlewareRoutes.POST("/api/pipelines/:source/:owner/:repo/builds", estafetteHandler.CreatePipelineBuild)
		jwtMiddlewareRoutes.POST("/api/pipelines/:source/:owner/:repo/releases", estafetteHandler.CreatePipelineRelease)
		jwtMiddlewareRoutes.POST("/api/pipelines/:source/:owner/:repo/bots", estafetteHandler.CreatePipelineBot)
		jwtMiddlewareRoutes.POST("/api/notifications", estafetteHandler.CreateNotification)
		jwtMiddlewareRoutes.DELETE("/api/pipelines/:source/:owner/:repo/builds/:revisionOrId", estafetteHandler.CancelPipelineBuild)
		jwtMiddlewareRoutes.DELETE("/api/pipelines/:source/:owner/:repo/releases/:id", estafetteHandler.CancelPipelineRelease)
		jwtMiddlewareRoutes.DELETE("/api/pipelines/:source/:owner/:repo/bots/:id", estafetteHandler.CancelPipelineBot)

		// to be removed after changing web frontend to use the /api/admin routes
		jwtMiddlewareRoutes.GET("/api/roles", rbacHandler.GetRoles)

		jwtMiddlewareRoutes.GET("/api/users", rbacHandler.GetUsers)
		jwtMiddlewareRoutes.GET("/api/users/:id", rbacHandler.GetUser)
		jwtMiddlewareRoutes.POST("/api/users", rbacHandler.CreateUser)
		jwtMiddlewareRoutes.PUT("/api/users/:id", rbacHandler.UpdateUser)
		jwtMiddlewareRoutes.DELETE("/api/users/:id", rbacHandler.DeleteUser)

		jwtMiddlewareRoutes.GET("/api/groups", rbacHandler.GetGroups)
		jwtMiddlewareRoutes.GET("/api/groups/:id", rbacHandler.GetGroup)
		jwtMiddlewareRoutes.POST("/api/groups", rbacHandler.CreateGroup)
		jwtMiddlewareRoutes.PUT("/api/groups/:id", rbacHandler.UpdateGroup)
		jwtMiddlewareRoutes.DELETE("/api/groups/:id", rbacHandler.DeleteGroup)

		jwtMiddlewareRoutes.GET("/api/organizations", rbacHandler.GetOrganizations)
		jwtMiddlewareRoutes.GET("/api/organizations/:id", rbacHandler.GetOrganization)
		jwtMiddlewareRoutes.POST("/api/organizations", rbacHandler.CreateOrganization)
		jwtMiddlewareRoutes.PUT("/api/organizations/:id", rbacHandler.UpdateOrganization)
		jwtMiddlewareRoutes.DELETE("/api/organizations/:id", rbacHandler.DeleteOrganization)

		jwtMiddlewareRoutes.GET("/api/clients", rbacHandler.GetClients)
		jwtMiddlewareRoutes.GET("/api/clients/:id", rbacHandler.GetClient)
		jwtMiddlewareRoutes.POST("/api/clients", rbacHandler.CreateClient)
		jwtMiddlewareRoutes.PUT("/api/clients/:id", rbacHandler.UpdateClient)
		jwtMiddlewareRoutes.DELETE("/api/clients/:id", rbacHandler.DeleteClient)

		// admin section
		jwtMiddlewareRoutes.GET("/api/auth/impersonate/:id", impersonateJWTMiddleware.LoginHandler)
		jwtMiddlewareRoutes.GET("/api/admin/roles", rbacHandler.GetRoles)

		jwtMiddlewareRoutes.GET("/api/admin/users", rbacHandler.GetUsers)
		jwtMiddlewareRoutes.GET("/api/admin/users/:id", rbacHandler.GetUser)
		jwtMiddlewareRoutes.POST("/api/admin/users", rbacHandler.CreateUser)
		jwtMiddlewareRoutes.PUT("/api/admin/users/:id", rbacHandler.UpdateUser)
		jwtMiddlewareRoutes.DELETE("/api/admin/users/:id", rbacHandler.DeleteUser)

		jwtMiddlewareRoutes.GET("/api/admin/pipelines", rbacHandler.GetPipelines)
		jwtMiddlewareRoutes.GET("/api/admin/pipelines/:source/:owner/:repo", rbacHandler.GetPipeline)
		jwtMiddlewareRoutes.PUT("/api/admin/pipelines/:source/:owner/:repo", rbacHandler.UpdatePipeline)

		jwtMiddlewareRoutes.POST("/api/admin/batch/users", rbacHandler.BatchUpdateUsers)
		jwtMiddlewareRoutes.POST("/api/admin/batch/groups", rbacHandler.BatchUpdateGroups)
		jwtMiddlewareRoutes.POST("/api/admin/batch/organizations", rbacHandler.BatchUpdateOrganizations)
		jwtMiddlewareRoutes.POST("/api/admin/batch/clients", rbacHandler.BatchUpdateClients)
		jwtMiddlewareRoutes.POST("/api/admin/batch/pipelines", rbacHandler.BatchUpdatePipelines)

		jwtMiddlewareRoutes.GET("/api/admin/groups", rbacHandler.GetGroups)
		jwtMiddlewareRoutes.GET("/api/admin/groups/:id", rbacHandler.GetGroup)
		jwtMiddlewareRoutes.POST("/api/admin/groups", rbacHandler.CreateGroup)
		jwtMiddlewareRoutes.PUT("/api/admin/groups/:id", rbacHandler.UpdateGroup)
		jwtMiddlewareRoutes.DELETE("/api/admin/groups/:id", rbacHandler.DeleteGroup)

		jwtMiddlewareRoutes.GET("/api/admin/organizations", rbacHandler.GetOrganizations)
		jwtMiddlewareRoutes.GET("/api/admin/organizations/:id", rbacHandler.GetOrganization)
		jwtMiddlewareRoutes.POST("/api/admin/organizations", rbacHandler.CreateOrganization)
		jwtMiddlewareRoutes.PUT("/api/admin/organizations/:id", rbacHandler.UpdateOrganization)
		jwtMiddlewareRoutes.DELETE("/api/admin/organizations/:id", rbacHandler.DeleteOrganization)

		jwtMiddlewareRoutes.GET("/api/admin/clients", rbacHandler.GetClients)
		jwtMiddlewareRoutes.GET("/api/admin/clients/:id", rbacHandler.GetClient)
		jwtMiddlewareRoutes.POST("/api/admin/clients", rbacHandler.CreateClient)
		jwtMiddlewareRoutes.PUT("/api/admin/clients/:id", rbacHandler.UpdateClient)
		jwtMiddlewareRoutes.DELETE("/api/admin/clients/:id", rbacHandler.DeleteClient)

		// catalog routes
		jwtMiddlewareRoutes.GET("/api/catalog/entity-labels", catalogHandler.GetCatalogEntityLabels)
		jwtMiddlewareRoutes.GET("/api/catalog/entity-parent-keys", catalogHandler.GetCatalogEntityParentKeys)
		jwtMiddlewareRoutes.GET("/api/catalog/entity-parent-values", catalogHandler.GetCatalogEntityParentValues)
		jwtMiddlewareRoutes.GET("/api/catalog/entity-keys", catalogHandler.GetCatalogEntityKeys)
		jwtMiddlewareRoutes.GET("/api/catalog/entity-values", catalogHandler.GetCatalogEntityValues)
		jwtMiddlewareRoutes.GET("/api/catalog/entities", catalogHandler.GetCatalogEntities)
		jwtMiddlewareRoutes.GET("/api/catalog/entities/:id", catalogHandler.GetCatalogEntity)
		jwtMiddlewareRoutes.POST("/api/catalog/entities", catalogHandler.CreateCatalogEntity)
		jwtMiddlewareRoutes.PUT("/api/catalog/entities/:id", catalogHandler.UpdateCatalogEntity)
		jwtMiddlewareRoutes.DELETE("/api/catalog/entities/:id", catalogHandler.DeleteCatalogEntity)
		jwtMiddlewareRoutes.GET("/api/catalog/users", catalogHandler.GetCatalogUsers)
		jwtMiddlewareRoutes.GET("/api/catalog/users/:id", catalogHandler.GetCatalogUser)
		jwtMiddlewareRoutes.GET("/api/catalog/groups", catalogHandler.GetCatalogGroups)
		jwtMiddlewareRoutes.GET("/api/catalog/groups/:id", catalogHandler.GetCatalogGroup)

		// do not require claims
		jwtMiddlewareRoutes.GET("/api/config", estafetteHandler.GetConfig)
		jwtMiddlewareRoutes.GET("/api/config/credentials", estafetteHandler.GetConfigCredentials)
		jwtMiddlewareRoutes.GET("/api/config/trustedimages", estafetteHandler.GetConfigTrustedImages)
		jwtMiddlewareRoutes.GET("/api/pipelines", estafetteHandler.GetPipelines)
		jwtMiddlewareRoutes.GET("/api/pipelines/:source/:owner/:repo", estafetteHandler.GetPipeline)
		jwtMiddlewareRoutes.GET("/api/pipelines/:source/:owner/:repo/recentbuilds", estafetteHandler.GetPipelineRecentBuilds)
		jwtMiddlewareRoutes.GET("/api/pipelines/:source/:owner/:repo/buildbranches", estafetteHandler.GetPipelineBuildBranches)
		jwtMiddlewareRoutes.GET("/api/pipelines/:source/:owner/:repo/builds", estafetteHandler.GetPipelineBuilds)
		jwtMiddlewareRoutes.GET("/api/pipelines/:source/:owner/:repo/builds/:revisionOrId", estafetteHandler.GetPipelineBuild)
		jwtMiddlewareRoutes.GET("/api/pipelines/:source/:owner/:repo/builds/:revisionOrId/warnings", estafetteHandler.GetPipelineBuildWarnings)
		jwtMiddlewareRoutes.GET("/api/pipelines/:source/:owner/:repo/builds/:revisionOrId/alllogs", estafetteHandler.GetPipelineBuildLogsPerPage)
		jwtMiddlewareRoutes.GET("/api/pipelines/:source/:owner/:repo/releases", estafetteHandler.GetPipelineReleases)
		jwtMiddlewareRoutes.GET("/api/pipelines/:source/:owner/:repo/releases/:releaseId", estafetteHandler.GetPipelineRelease)
		jwtMiddlewareRoutes.GET("/api/pipelines/:source/:owner/:repo/releases/:releaseId/alllogs", estafetteHandler.GetPipelineReleaseLogsPerPage)
		jwtMiddlewareRoutes.GET("/api/pipelines/:source/:owner/:repo/botnames", estafetteHandler.GetPipelineBotNames)
		jwtMiddlewareRoutes.GET("/api/pipelines/:source/:owner/:repo/bots", estafetteHandler.GetPipelineBots)
		jwtMiddlewareRoutes.GET("/api/pipelines/:source/:owner/:repo/bots/:botId", estafetteHandler.GetPipelineBot)
		jwtMiddlewareRoutes.GET("/api/pipelines/:source/:owner/:repo/bots/:botId/alllogs", estafetteHandler.GetPipelineBotLogsPerPage)
		jwtMiddlewareRoutes.GET("/api/pipelines/:source/:owner/:repo/stats/buildsdurations", estafetteHandler.GetPipelineStatsBuildsDurations)
		jwtMiddlewareRoutes.GET("/api/pipelines/:source/:owner/:repo/stats/releasesdurations", estafetteHandler.GetPipelineStatsReleasesDurations)
		jwtMiddlewareRoutes.GET("/api/pipelines/:source/:owner/:repo/stats/botsdurations", estafetteHandler.GetPipelineStatsBotsDurations)
		jwtMiddlewareRoutes.GET("/api/pipelines/:source/:owner/:repo/stats/buildscpu", estafetteHandler.GetPipelineStatsBuildsCPUUsageMeasurements)
		jwtMiddlewareRoutes.GET("/api/pipelines/:source/:owner/:repo/stats/releasescpu", estafetteHandler.GetPipelineStatsReleasesCPUUsageMeasurements)
		jwtMiddlewareRoutes.GET("/api/pipelines/:source/:owner/:repo/stats/botscpu", estafetteHandler.GetPipelineStatsBotsCPUUsageMeasurements)
		jwtMiddlewareRoutes.GET("/api/pipelines/:source/:owner/:repo/stats/buildsmemory", estafetteHandler.GetPipelineStatsBuildsMemoryUsageMeasurements)
		jwtMiddlewareRoutes.GET("/api/pipelines/:source/:owner/:repo/stats/releasesmemory", estafetteHandler.GetPipelineStatsReleasesMemoryUsageMeasurements)
		jwtMiddlewareRoutes.GET("/api/pipelines/:source/:owner/:repo/stats/botsmemory", estafetteHandler.GetPipelineStatsBotsMemoryUsageMeasurements)
		jwtMiddlewareRoutes.GET("/api/pipelines/:source/:owner/:repo/warnings", estafetteHandler.GetPipelineWarnings)
		jwtMiddlewareRoutes.GET("/api/builds", estafetteHandler.GetAllPipelineBuilds)
		jwtMiddlewareRoutes.GET("/api/releases", estafetteHandler.GetAllPipelineReleases)
		jwtMiddlewareRoutes.GET("/api/bots", estafetteHandler.GetAllPipelineBots)
		jwtMiddlewareRoutes.GET("/api/notifications", estafetteHandler.GetAllNotifications)
		jwtMiddlewareRoutes.GET("/api/releasetargets/pipelines", estafetteHandler.GetAllPipelinesReleaseTargets)
		jwtMiddlewareRoutes.GET("/api/releasetargets/releases", estafetteHandler.GetAllReleasesReleaseTargets)
		jwtMiddlewareRoutes.GET("/api/catalog/filters", estafetteHandler.GetCatalogFilters)
		jwtMiddlewareRoutes.GET("/api/catalog/filtervalues", estafetteHandler.GetCatalogFilterValues)
		jwtMiddlewareRoutes.GET("/api/stats/pipelinescount", estafetteHandler.GetStatsPipelinesCount)
		jwtMiddlewareRoutes.GET("/api/stats/buildscount", estafetteHandler.GetStatsBuildsCount)
		jwtMiddlewareRoutes.GET("/api/stats/releasescount", estafetteHandler.GetStatsReleasesCount)
		jwtMiddlewareRoutes.GET("/api/stats/botscount", estafetteHandler.GetStatsBotsCount)
		jwtMiddlewareRoutes.GET("/api/stats/buildsduration", estafetteHandler.GetStatsBuildsDuration)
		jwtMiddlewareRoutes.GET("/api/stats/buildsadoption", estafetteHandler.GetStatsBuildsAdoption)
		jwtMiddlewareRoutes.GET("/api/stats/releasesadoption", estafetteHandler.GetStatsReleasesAdoption)
		jwtMiddlewareRoutes.GET("/api/stats/botsadoption", estafetteHandler.GetStatsBotsAdoption)
		jwtMiddlewareRoutes.GET("/api/stats/mostbuilds", estafetteHandler.GetStatsMostBuilds)
		jwtMiddlewareRoutes.GET("/api/stats/mostreleases", estafetteHandler.GetStatsMostReleases)
		jwtMiddlewareRoutes.GET("/api/stats/mostbots", estafetteHandler.GetStatsMostBots)
		jwtMiddlewareRoutes.GET("/api/manifest/templates", estafetteHandler.GetManifestTemplates)
		jwtMiddlewareRoutes.POST("/api/manifest/generate", estafetteHandler.GenerateManifest)
		jwtMiddlewareRoutes.POST("/api/manifest/validate", estafetteHandler.ValidateManifest)
		jwtMiddlewareRoutes.POST("/api/manifest/encrypt", estafetteHandler.EncryptSecret)
		jwtMiddlewareRoutes.GET("/api/labels/frequent", estafetteHandler.GetFrequentLabels)

		// communication from build/release jobs back to api
		jwtMiddlewareRoutes.POST("/api/commands", estafetteHandler.Commands)
		jwtMiddlewareRoutes.POST("/api/pipelines/:source/:owner/:repo/builds/:revisionOrId/logs", estafetteHandler.PostPipelineBuildLogs)
		jwtMiddlewareRoutes.POST("/api/pipelines/:source/:owner/:repo/releases/:releaseId/logs", estafetteHandler.PostPipelineReleaseLogs)
		jwtMiddlewareRoutes.POST("/api/pipelines/:source/:owner/:repo/bots/:botId/logs", estafetteHandler.PostPipelineBotLogs)

		// communication from cron-event-sender to api
		jwtMiddlewareRoutes.POST("/api/integrations/cron/events", estafetteHandler.PostCronEvent)

		// do not require claims and avoid re-zipping
		preZippedJWTMiddlewareRoutes.GET("/api/pipelines/:source/:owner/:repo/builds/:revisionOrId/logs", estafetteHandler.GetPipelineBuildLogs)
		preZippedJWTMiddlewareRoutes.GET("/api/pipelines/:source/:owner/:repo/builds/:revisionOrId/logs/tail", estafetteHandler.TailPipelineBuildLogs)
		preZippedJWTMiddlewareRoutes.GET("/api/pipelines/:source/:owner/:repo/builds/:revisionOrId/logsbyid/:id", estafetteHandler.GetPipelineBuildLogsByID)
		preZippedJWTMiddlewareRoutes.GET("/api/pipelines/:source/:owner/:repo/builds/:revisionOrId/logs.stream", estafetteHandler.TailPipelineBuildLogs)
		preZippedJWTMiddlewareRoutes.GET("/api/pipelines/:source/:owner/:repo/releases/:releaseId/logs", estafetteHandler.GetPipelineReleaseLogs)
		preZippedJWTMiddlewareRoutes.GET("/api/pipelines/:source/:owner/:repo/releases/:releaseId/logs/tail", estafetteHandler.TailPipelineReleaseLogs)
		preZippedJWTMiddlewareRoutes.GET("/api/pipelines/:source/:owner/:repo/releases/:releaseId/logsbyid/:id", estafetteHandler.GetPipelineReleaseLogsByID)
		preZippedJWTMiddlewareRoutes.GET("/api/pipelines/:source/:owner/:repo/releases/:releaseId/logs.stream", estafetteHandler.TailPipelineReleaseLogs)
		preZippedJWTMiddlewareRoutes.GET("/api/pipelines/:source/:owner/:repo/bots/:botId/logs", estafetteHandler.GetPipelineBotLogs)
		preZippedJWTMiddlewareRoutes.GET("/api/pipelines/:source/:owner/:repo/bots/:botId/logs/tail", estafetteHandler.TailPipelineBotLogs)
		preZippedJWTMiddlewareRoutes.GET("/api/pipelines/:source/:owner/:repo/bots/:botId/logsbyid/:id", estafetteHandler.GetPipelineBotLogsByID)
		preZippedJWTMiddlewareRoutes.GET("/api/pipelines/:source/:owner/:repo/bots/:botId/logs.stream", estafetteHandler.TailPipelineBotLogs)
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

	// disable jaeger if service name is empty
	if cfg.ServiceName == "" {
		cfg.Disabled = true
	}

	closer, err := cfg.InitGlobalTracer(cfg.ServiceName, jaegercfg.Logger(jaeger.StdLogger), jaegercfg.Metrics(jprom.New()))

	if err != nil {
		log.Fatal().Err(err).Msg("Generating Jaeger tracer failed")
	}

	return closer
}
