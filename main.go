package main

import (
	"context"
	"encoding/json"
	"fmt"
	stdlog "log"
	"math"
	"net/http"
	"os"
	"os/signal"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/estafette/estafette-ci-contracts"
	"github.com/estafette/estafette-ci-crypt"

	"github.com/alecthomas/kingpin"
	"github.com/estafette/estafette-ci-api/bitbucket"
	"github.com/estafette/estafette-ci-api/cockroach"
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

	apiAddress = kingpin.Flag("api-listen-address", "The address to listen on for api HTTP requests.").Default(":5000").String()

	githubAppPrivateKeyPath      = kingpin.Flag("github-app-privatey-key-path", "The path to the pem file for the private key of the Github App.").Default("/github-app-key/private-key.pem").String()
	githubAppID                  = kingpin.Flag("github-app-id", "The Github App id.").Envar("GITHUB_APP_ID").String()
	githubAppOAuthClientID       = kingpin.Flag("github-app-oauth-client-id", "The OAuth client id for the Github App.").Envar("GITHUB_APP_OAUTH_CLIENT_ID").String()
	githubAppOAuthClientSecret   = kingpin.Flag("github-app-oauth-client-secret", "The OAuth client secret for the Github App.").Envar("GITHUB_APP_OAUTH_CLIENT_SECRET").String()
	githubWebhookSecret          = kingpin.Flag("github-webhook-secret", "The secret to verify webhook authenticity.").Envar("GITHUB_WEBHOOK_SECRET").String()
	githubEventChannelBufferSize = kingpin.Flag("github-event-channel-buffer-size", "The buffer size of the github event channel.").Envar("GITHUB_EVENT_CHANNEL_BUFFER_SIZE").Default("100").Int()
	githubMaxWorkers             = kingpin.Flag("github-max-workers", "The maximum number of workers to handle github events from the event channel.").Envar("GITHUB_MAX_WORKERS").Default("5").Int()

	bitbucketAPIKey                 = kingpin.Flag("bitbucket-api-key", "The api key for Bitbucket.").Envar("BITBUCKET_API_KEY").String()
	bitbucketAppOAuthKey            = kingpin.Flag("bitbucket-app-oauth-key", "The OAuth key for the Bitbucket App.").Envar("BITBUCKET_APP_OAUTH_KEY").String()
	bitbucketAppOAuthSecret         = kingpin.Flag("bitbucket-app-oauth-secret", "The OAuth secret for the Bitbucket App.").Envar("BITBUCKET_APP_OAUTH_SECRET").String()
	bitbucketEventChannelBufferSize = kingpin.Flag("bitbucket-event-channel-buffer-size", "The buffer size of the bitbucket event channel.").Envar("BITBUCKET_EVENT_CHANNEL_BUFFER_SIZE").Default("100").Int()
	bitbucketMaxWorkers             = kingpin.Flag("bitbucket-max-workers", "The maximum number of workers to handle bitbucket events from the event channel.").Envar("BITBUCKET_MAX_WORKERS").Default("5").Int()

	estafetteCiServerBaseURL        = kingpin.Flag("estafette-ci-server-base-url", "The base url of this api server.").Envar("ESTAFETTE_CI_SERVER_BASE_URL").String()
	estafetteCiServerServiceURL     = kingpin.Flag("estafette-ci-server-service-url", "The kubernetes service url of this api server.").Envar("ESTAFETTE_CI_SERVER_SERVICE_URL").String()
	estafetteCiAPIKey               = kingpin.Flag("estafette-ci-api-key", "An api key for estafette itself to use until real oauth is supported.").Envar("ESTAFETTE_CI_API_KEY").String()
	estafetteEventChannelBufferSize = kingpin.Flag("estafette-event-channel-buffer-size", "The buffer size of the estafette event channel.").Envar("ESTAFETTE_EVENT_CHANNEL_BUFFER_SIZE").Default("100").Int()
	estafetteMaxWorkers             = kingpin.Flag("estafette-max-workers", "The maximum number of workers to handle estafette events from the event channel.").Envar("ESTAFETTE_MAX_WORKERS").Default("5").Int()

	slackAppClientID            = kingpin.Flag("slack-app-client-id", "The Slack App id for accessing Slack API.").Envar("SLACK_APP_CLIENT_ID").String()
	slackAppClientSecret        = kingpin.Flag("slack-app-client-secret", "The Slack App secret for accessing Slack API.").Envar("SLACK_APP_CLIENT_ID").String()
	slackAppVerificationToken   = kingpin.Flag("slack-app-verification-token", "The token used to verify incoming Slack webhook events.").Envar("SLACK_APP_VERIFICATION_TOKEN").String()
	slackAppOAuthAccessToken    = kingpin.Flag("slack-app-oauth-access-token", "The OAuth access token for the Slack App.").Envar("SLACK_APP_OAUTH_ACCESS_TOKEN").String()
	slackEventChannelBufferSize = kingpin.Flag("slack-event-channel-buffer-size", "The buffer size of the slack event channel.").Envar("SLACK_EVENT_CHANNEL_BUFFER_SIZE").Default("100").Int()
	slackMaxWorkers             = kingpin.Flag("slack-max-workers", "The maximum number of workers to handle slack events from the event channel.").Envar("SLACK_MAX_WORKERS").Default("5").Int()

	secretDecryptionKey = kingpin.Flag("secret-decryption-key", "The AES-256 key used to decrypt secrets that have been encrypted with it.").Envar("SECRET_DECRYPTION_KEY").String()

	cockroachDatabase       = kingpin.Flag("cockroach-database", "CockroachDB database.").Envar("COCKROACH_DATABASE").String()
	cockroachHost           = kingpin.Flag("cockroach-host", "CockroachDB host.").Envar("COCKROACH_HOST").String()
	cockroachInsecure       = kingpin.Flag("cockroach-insecure", "CockroachDB insecure connection.").Envar("COCKROACH_INSECURE").Bool()
	cockroachCertificateDir = kingpin.Flag("cockroach-certs-dir", "CockroachDB certificate directory.").Envar("COCKROACH_CERTS_DIR").String()
	cockroachPort           = kingpin.Flag("cockroach-port", "CockroachDB port.").Envar("COCKROACH_PORT").Int()
	cockroachUser           = kingpin.Flag("cockroach-user", "CockroachDB user.").Envar("COCKROACH_USER").String()
	cockroachPassword       = kingpin.Flag("cockroach-password", "CockroachDB password.").Envar("COCKROACH_PASSWORD").String()

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

	githubAPIClient := github.NewGithubAPIClient(*githubAppPrivateKeyPath, *githubAppID, *githubAppOAuthClientID, *githubAppOAuthClientSecret, prometheusOutboundAPICallTotals)
	bitbucketAPIClient := bitbucket.NewBitbucketAPIClient(*bitbucketAPIKey, *bitbucketAppOAuthKey, *bitbucketAppOAuthSecret, prometheusOutboundAPICallTotals)
	slackAPIClient := slack.NewSlackAPIClient(*slackAppClientID, *slackAppClientSecret, *slackAppOAuthAccessToken, prometheusOutboundAPICallTotals)
	secretHelper := crypt.NewSecretHelper(*secretDecryptionKey)
	cockroachDBClient := cockroach.NewCockroachDBClient(*cockroachDatabase, *cockroachHost, *cockroachInsecure, *cockroachCertificateDir, *cockroachPort, *cockroachUser, *cockroachPassword, prometheusOutboundAPICallTotals)
	ciBuilderClient, err := estafette.NewCiBuilderClient(*estafetteCiServerBaseURL, *estafetteCiServerServiceURL, *estafetteCiAPIKey, *secretDecryptionKey, prometheusOutboundAPICallTotals)
	if err != nil {
		log.Fatal().Err(err).Msg("Creating new CiBuilderClient has failed")
	}

	// set up database
	err = cockroachDBClient.Connect()
	if err != nil {
		log.Warn().Err(err).Msg("Failed connecting to CockroachDB")
	}
	err = cockroachDBClient.MigrateSchema()
	if err != nil {
		log.Warn().Err(err).Msg("Failed migrating schema of CockroachDB")
	}

	// listen to channels for push events
	githubPushEvents := make(chan github.PushEvent, *githubEventChannelBufferSize)
	githubDispatcher := github.NewGithubDispatcher(stopChannel, waitGroup, *githubMaxWorkers, githubAPIClient, ciBuilderClient, cockroachDBClient, githubPushEvents)
	githubDispatcher.Run()

	bitbucketPushEvents := make(chan bitbucket.RepositoryPushEvent, *bitbucketEventChannelBufferSize)
	bitbucketDispatcher := bitbucket.NewBitbucketDispatcher(stopChannel, waitGroup, *bitbucketMaxWorkers, bitbucketAPIClient, ciBuilderClient, cockroachDBClient, bitbucketPushEvents)
	bitbucketDispatcher.Run()

	slackEvents := make(chan slack.SlashCommand, *slackEventChannelBufferSize)
	slackDispatcher := slack.NewSlackDispatcher(stopChannel, waitGroup, *slackMaxWorkers, slackAPIClient, slackEvents)
	slackDispatcher.Run()

	estafetteCiBuilderEvents := make(chan estafette.CiBuilderEvent, *estafetteEventChannelBufferSize)
	estafetteBuildJobLogs := make(chan cockroach.BuildJobLogs, *estafetteEventChannelBufferSize)
	estafetteDispatcher := estafette.NewEstafetteDispatcher(stopChannel, waitGroup, *estafetteMaxWorkers, ciBuilderClient, cockroachDBClient, estafetteCiBuilderEvents, estafetteBuildJobLogs)
	estafetteDispatcher.Run()

	// listen to http calls
	log.Debug().
		Str("port", *apiAddress).
		Msg("Serving api calls...")

	// create and init router
	router := createRouter()

	githubEventHandler := github.NewGithubEventHandler(githubPushEvents, *githubWebhookSecret, prometheusInboundEventTotals)
	router.POST("/api/integrations/github/events", githubEventHandler.Handle)

	bitbucketEventHandler := bitbucket.NewBitbucketEventHandler(bitbucketPushEvents, prometheusInboundEventTotals)
	router.POST("/api/integrations/bitbucket/events", bitbucketEventHandler.Handle)

	slackEventHandler := slack.NewSlackEventHandler(secretHelper, *slackAppVerificationToken, slackEvents, prometheusInboundEventTotals)
	router.POST("/api/integrations/slack/slash", slackEventHandler.Handle)

	estafetteEventHandler := estafette.NewEstafetteEventHandler(*estafetteCiAPIKey, estafetteCiBuilderEvents, estafetteBuildJobLogs, prometheusInboundEventTotals)
	router.POST("/api/commands", estafetteEventHandler.Handle)

	router.GET("/logs/:source/:owner/:repo/:branch/:revision", func(c *gin.Context) {
		source := c.Param("source")
		owner := c.Param("owner")
		repo := c.Param("repo")
		branch := c.Param("branch")
		revision := c.Param("revision")

		buildJobLogsParams := cockroach.BuildJobLogs{
			RepoSource:   source,
			RepoFullName: fmt.Sprintf("%v/%v", owner, repo),
			RepoBranch:   branch,
			RepoRevision: revision,
		}

		// retrieve logs from database
		logs, err := cockroachDBClient.GetBuildLogs(buildJobLogsParams)
		if err != nil {
			c.String(http.StatusInternalServerError, err.Error())
			return
		}

		if len(logs) == 0 {
			c.String(http.StatusOK, "These logs are no longer available")
			return
		}

		// get text from logs
		logTexts := make([]string, 0)
		for _, logItem := range logs {

			// split text on newline
			logLines := strings.Split(logItem.LogText, "\n")
			for _, logLine := range logLines {

				if logLine == "" {
					logTexts = append(logTexts, logLine)
					continue
				}

				// deserialize json log
				var ciBuilderLogLine estafette.CiBuilderLogLine
				err = json.Unmarshal([]byte(logLine), &ciBuilderLogLine)
				if err != nil {
					log.Warn().Err(err).Str("logLine", logLine).Msg("Failed unmarshalling log line")
					logTexts = append(logTexts, logLine)
					continue
				}

				logTexts = append(logTexts, fmt.Sprintf("%v | %-5s | %v", ciBuilderLogLine.Time, strings.ToUpper(ciBuilderLogLine.Severity), ciBuilderLogLine.Message))
			}
		}

		c.String(http.StatusOK, strings.Join(logTexts, "\n"))
	})

	router.GET("/api/pipelines", func(c *gin.Context) {

		// get page number query string value or default to 1
		pageNumberValue, pageNumberExists := c.GetQuery("page[number]")
		pageNumber, err := strconv.Atoi(pageNumberValue)
		if !pageNumberExists || err != nil {
			pageNumber = 1
		}

		// get page number query string value or default to 20 (maximize at 100)
		pageSizeValue, pageSizeExists := c.GetQuery("page[size]")
		pageSize, err := strconv.Atoi(pageSizeValue)
		if !pageSizeExists || err != nil {
			pageSize = 20
		}
		if pageSize > 100 {
			pageSize = 100
		}

		// get filters (?filter[post]=1,2&filter[author]=12)
		filters := map[string][]string{}
		filterStatusValues, filterStatusExist := c.GetQueryArray("filter[status]")
		if filterStatusExist && len(filterStatusValues) > 0 && filterStatusValues[0] != "" {
			filters["status"] = filterStatusValues
		}
		filterSinceValues, filterSinceExist := c.GetQueryArray("filter[since]")
		if filterSinceExist {
			filters["since"] = filterSinceValues
		} else {
			filters["since"] = []string{"eternity"}
		}

		pipelines, err := cockroachDBClient.GetPipelines(pageNumber, pageSize, filters)
		if err != nil {
			log.Error().Err(err).
				Msg("Failed retrieving pipelines from db")
		}
		log.Info().Msgf("Retrieved %v pipelines", len(pipelines))

		pipelinesCount, err := cockroachDBClient.GetPipelinesCount(filters)
		if err != nil {
			log.Error().Err(err).
				Msg("Failed retrieving pipelines count from db")
		}
		log.Info().Msgf("Retrieved pipelines count %v", pipelinesCount)

		response := contracts.ListResponse{
			Pagination: contracts.Pagination{
				Page:       pageNumber,
				Size:       pageSize,
				TotalItems: pipelinesCount,
				TotalPages: int(math.Ceil(float64(pipelinesCount) / float64(pageSize))),
			},
		}

		response.Items = make([]interface{}, len(pipelines))
		for i := range pipelines {
			response.Items[i] = pipelines[i]
		}

		c.JSON(http.StatusOK, response)
	})

	router.GET("/api/pipelines/:source/:owner/:repo", func(c *gin.Context) {

		source := c.Param("source")
		owner := c.Param("owner")
		repo := c.Param("repo")

		pipeline, err := cockroachDBClient.GetPipeline(source, owner, repo)
		if err != nil {
			log.Error().Err(err).
				Msgf("Failed retrieving pipeline for %v/%v/%v from db", source, owner, repo)
		}
		if pipeline == nil {
			c.JSON(http.StatusNotFound, gin.H{"code": "PAGE_NOT_FOUND", "message": "Pipeline not found"})
			return
		}

		log.Info().Msgf("Retrieved pipeline for %v/%v/%v", source, owner, repo)

		c.JSON(http.StatusOK, pipeline)
	})

	router.GET("/api/pipelines/:source/:owner/:repo/builds", func(c *gin.Context) {

		source := c.Param("source")
		owner := c.Param("owner")
		repo := c.Param("repo")

		// get page number query string value or default to 1
		pageNumberValue, pageNumberExists := c.GetQuery("page[number]")
		pageNumber, err := strconv.Atoi(pageNumberValue)
		if !pageNumberExists || err != nil {
			pageNumber = 1
		}

		// get page number query string value or default to 20 (maximize at 100)
		pageSizeValue, pageSizeExists := c.GetQuery("page[size]")
		pageSize, err := strconv.Atoi(pageSizeValue)
		if !pageSizeExists || err != nil {
			pageSize = 20
		}
		if pageSize > 100 {
			pageSize = 100
		}

		builds, err := cockroachDBClient.GetPipelineBuilds(source, owner, repo, pageNumber, pageSize)
		if err != nil {
			log.Error().Err(err).
				Msgf("Failed retrieving builds for %v/%v/%v from db", source, owner, repo)
		}
		log.Info().Msgf("Retrieved %v builds for %v/%v/%v", len(builds), source, owner, repo)

		buildsCount, err := cockroachDBClient.GetPipelineBuildsCount(source, owner, repo)
		if err != nil {
			log.Error().Err(err).
				Msgf("Failed retrieving builds count for %v/%v/%v from db", source, owner, repo)
		}
		log.Info().Msgf("Retrieved builds count %v for %v/%v/%v", buildsCount, source, owner, repo)

		response := contracts.ListResponse{
			Pagination: contracts.Pagination{
				Page:       pageNumber,
				Size:       pageSize,
				TotalItems: buildsCount,
				TotalPages: int(math.Ceil(float64(buildsCount) / float64(pageSize))),
			},
		}

		response.Items = make([]interface{}, len(builds))
		for i := range builds {
			response.Items[i] = builds[i]
		}

		c.JSON(http.StatusOK, response)
	})

	router.GET("/api/pipelines/:source/:owner/:repo/builds/:revision", func(c *gin.Context) {

		source := c.Param("source")
		owner := c.Param("owner")
		repo := c.Param("repo")
		revision := c.Param("revision")

		build, err := cockroachDBClient.GetPipelineBuild(source, owner, repo, revision)
		if err != nil {
			log.Error().Err(err).
				Msgf("Failed retrieving build for %v/%v/%v/%v from db", source, owner, repo, revision)
		}
		if build == nil {
			c.JSON(http.StatusNotFound, gin.H{"code": "PAGE_NOT_FOUND", "message": "Pipeline build not found"})
			return
		}
		log.Info().Msgf("Retrieved builds for %v/%v/%v/%v", source, owner, repo, revision)

		c.JSON(http.StatusOK, build)
	})

	router.GET("/api/pipelines/:source/:owner/:repo/builds/:revision/logs", func(c *gin.Context) {

		source := c.Param("source")
		owner := c.Param("owner")
		repo := c.Param("repo")
		revision := c.Param("revision")

		buildLog, err := cockroachDBClient.GetPipelineBuildLogs(source, owner, repo, revision)
		if err != nil {
			log.Error().Err(err).
				Msgf("Failed retrieving build logs for %v/%v/%v/%v from db", source, owner, repo, revision)
		}
		if buildLog == nil {
			c.JSON(http.StatusNotFound, gin.H{"code": "PAGE_NOT_FOUND", "message": "Pipeline build log not found"})
			return
		}
		log.Info().Msgf("Retrieved build logs for %v/%v/%v/%v", source, owner, repo, revision)

		c.JSON(http.StatusOK, buildLog)
	})

	router.POST("/api/pipelines/:source/:owner/:repo/builds/:revision/logs", func(c *gin.Context) {

		authorizationHeader := c.GetHeader("Authorization")
		if authorizationHeader != fmt.Sprintf("Bearer %v", *estafetteCiAPIKey) {
			log.Error().
				Str("authorizationHeader", authorizationHeader).
				Msg("Authorization header for Estafette v2 logs is incorrect")
			c.String(http.StatusUnauthorized, "Authorization failed")
			return
		}

		source := c.Param("source")
		owner := c.Param("owner")
		repo := c.Param("repo")
		revision := c.Param("revision")

		var buildLog contracts.BuildLog
		err := c.Bind(&buildLog)
		if err != nil {
			log.Error().Err(err).
				Msgf("Failed binding v2 logs for %v/%v/%v/%v", source, owner, repo, revision)
		}

		log.Info().Interface("buildLog", buildLog).Msgf("Binded v2 logs for for %v/%v/%v/%v", source, owner, repo, revision)

		err = cockroachDBClient.InsertBuildLog(buildLog)
		if err != nil {
			log.Error().Err(err).
				Msgf("Failed inserting v2 logs for %v/%v/%v/%v", source, owner, repo, revision)
		}
		log.Info().Msgf("Inserted v2 logs for %v/%v/%v/%v", source, owner, repo, revision)

		c.String(http.StatusOK, "Aye aye!")
	})

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
