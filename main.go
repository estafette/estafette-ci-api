package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"math/rand"
	"net/http"
	"os"
	"runtime"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	version   string
	branch    string
	revision  string
	buildDate string
	goVersion = runtime.Version()
)

var (
	prometheusAddress     = flag.String("metrics-listen-address", ":9001", "The address to listen on for Prometheus metrics requests.")
	prometheusMetricsPath = flag.String("metrics-path", "/metrics", "The path to listen for Prometheus metrics requests.")
	apiAddress            = flag.String("api-listen-address", ":5000", "The address to listen on for api HTTP requests.")

	// seed random number
	r = rand.New(rand.NewSource(time.Now().UnixNano()))

	// define prometheus counter
	webhookTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "estafette_ci_api_webhook_totals",
			Help: "Total of received webhooks.",
		},
		[]string{"event", "source"},
	)
)

func init() {
	// Metrics have to be registered to be exposed:
	prometheus.MustRegister(webhookTotal)
}

func main() {

	// parse command line parameters
	flag.Parse()

	// set some default fields added to all logs
	log := zerolog.New(os.Stdout).With().
		Timestamp().
		Str("app", "estafette-ci-api").
		Str("version", version).
		Logger()

	// log startup message
	log.Info().
		Str("branch", branch).
		Str("revision", revision).
		Str("buildDate", buildDate).
		Str("goVersion", goVersion).
		Msg("Starting estafette-ci-api...")

	// start prometheus
	go func() {
		log.Info().
			Str("port", *prometheusAddress).
			Str("path", *prometheusMetricsPath).
			Msg("Serving Prometheus metrics...")

		http.Handle(*prometheusMetricsPath, promhttp.Handler())

		if err := http.ListenAndServe(*prometheusAddress, nil); err != nil {
			log.Fatal().Err(err).Msg("Starting Prometheus listener failed")
		}
	}()

	log.Info().
		Str("port", *apiAddress).
		Msg("Serving api calls...")

	http.HandleFunc("/webhook/github", githubWebhookHandler)
	http.HandleFunc("/webhook/bitbucket", bitbucketWebhookHandler)
	http.HandleFunc("/liveness", livenessHandler)
	http.HandleFunc("/readiness", readinessHandler)

	if err := http.ListenAndServe(*apiAddress, nil); err != nil {
		log.Fatal().Err(err).Msg("Starting api listener failed")
	}
}

func githubWebhookHandler(w http.ResponseWriter, r *http.Request) {

	// https://developer.github.com/webhooks/
	eventType := r.Header.Get("X-GitHub-Event")
	webhookTotal.With(prometheus.Labels{"event": eventType, "source": "github"}).Inc()

	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		log.Error().Err(err).Msg("Reading body from Github webhook failed")
		http.Error(w, "Reading body from Github webhook failed", 500)
		return
	}

	// unmarshal json body
	var b interface{}
	err = json.Unmarshal(body, &b)
	if err != nil {
		log.Error().Err(err).Str("body", string(body)).Msg("Deserializing body from Github webhook failed")
		http.Error(w, "Deserializing body from Github webhook failed", 500)
		return
	}

	log.Info().
		Str("method", r.Method).
		Str("url", r.URL.String()).
		Interface("headers", r.Header).
		Interface("body", b).
		Msgf("Received webhook event of type '%v' from GitHub...", eventType)

	switch eventType {
	case "push": // Any Git push to a Repository, including editing tags or branches. Commits via API actions that update references are also counted. This is the default event.
		handleGithubPush(body)

	case "commit_comment": // Any time a Commit is commented on.
	case "create": // Any time a Branch or Tag is created.
	case "delete": // Any time a Branch or Tag is deleted.
	case "deployment": // Any time a Repository has a new deployment created from the API.
	case "deployment_status": // Any time a deployment for a Repository has a status update from the API.
	case "fork": // Any time a Repository is forked.
	case "gollum": // Any time a Wiki page is updated.
	case "installation": // Any time a GitHub App is installed or uninstalled.
	case "installation_repositories": // Any time a repository is added or removed from an installation.
	case "issue_comment": // Any time a comment on an issue is created, edited, or deleted.
	case "issues": // Any time an Issue is assigned, unassigned, labeled, unlabeled, opened, edited, milestoned, demilestoned, closed, or reopened.
	case "label": // Any time a Label is created, edited, or deleted.
	case "marketplace_purchase": // Any time a user purchases, cancels, or changes their GitHub Marketplace plan.
	case "member": // Any time a User is added or removed as a collaborator to a Repository, or has their permissions modified.
	case "membership": // Any time a User is added or removed from a team. Organization hooks only.
	case "milestone": // Any time a Milestone is created, closed, opened, edited, or deleted.
	case "organization": // Any time a user is added, removed, or invited to an Organization. Organization hooks only.
	case "org_block": // Any time an organization blocks or unblocks a user. Organization hooks only.
	case "page_build": // Any time a Pages site is built or results in a failed build.
	case "project_card": // Any time a Project Card is created, edited, moved, converted to an issue, or deleted.
	case "project_column": // Any time a Project Column is created, edited, moved, or deleted.
	case "project": // Any time a Project is created, edited, closed, reopened, or deleted.
	case "public": // Any time a Repository changes from private to public.
	case "pull_request_review_comment": // Any time a comment on a pull request's unified diff is created, edited, or deleted (in the Files Changed tab).
	case "pull_request_review": // Any time a pull request review is submitted, edited, or dismissed.
	case "pull_request": // Any time a pull request is assigned, unassigned, labeled, unlabeled, opened, edited, closed, reopened, or synchronized (updated due to a new push in the branch that the pull request is tracking). Also any time a pull request review is requested, or a review request is removed.
	case "repository": // Any time a Repository is created, deleted (organization hooks only), made public, or made private.
	case "release": // Any time a Release is published in a Repository.
	case "status": // Any time a Repository has a status update from the API
	case "team": // Any time a team is created, deleted, modified, or added to or removed from a repository. Organization hooks only
	case "team_add": // Any time a team is added or modified on a Repository.
	case "watch": // Any time a User stars a Repository.
		log.Debug().Str("event", eventType).Msgf("Not implemented Github webhook event of type '%v'", eventType)

	default:
		log.Warn().Str("event", eventType).Msgf("Unsupported Github webhook event of type '%v'", eventType)
	}

	fmt.Fprintf(w, "Aye aye!")
}

func bitbucketWebhookHandler(w http.ResponseWriter, r *http.Request) {

	// https://confluence.atlassian.com/bitbucket/manage-webhooks-735643732.html

	eventType := r.Header.Get("X-Event-Key")
	webhookTotal.With(prometheus.Labels{"event": eventType, "source": "bitbucket"}).Inc()

	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		log.Error().Err(err).Msg("Reading body from Bitbucket webhook failed")
		http.Error(w, "Reading body from Bitbucket webhook failed", 500)
		return
	}

	// unmarshal json body
	var b interface{}
	err = json.Unmarshal(body, &b)
	if err != nil {
		log.Error().Err(err).Str("body", string(body)).Msg("Deserializing body from Bitbucket webhook failed")
		http.Error(w, "Deserializing body from Bitbucket webhook failed", 500)
		return
	}

	log.Info().
		Str("method", r.Method).
		Str("url", r.URL.String()).
		Interface("headers", r.Header).
		Interface("body", b).
		Msgf("Received webhook event of type '%v' from Bitbucket...", eventType)

	switch eventType {
	case "repo:push":
		handleBitbucketPush(body)

	case "repo:fork":
	case "repo:updated":
	case "repo:transfer":
	case "repo:commit_comment_created":
	case "repo:commit_status_created":
	case "repo:commit_status_updated":
	case "issue:created":
	case "issue:updated":
	case "issue:comment_created":
	case "pullrequest:created":
	case "pullrequest:updated":
	case "pullrequest:approved":
	case "pullrequest:unapproved":
	case "pullrequest:fulfilled":
	case "pullrequest:rejected":
	case "pullrequest:comment_created":
	case "pullrequest:comment_updated":
	case "pullrequest:comment_deleted":
		log.Debug().Str("event", eventType).Msgf("Not implemented Bitbucket webhook event of type '%v'", eventType)

	default:
		log.Warn().Str("event", eventType).Msgf("Unsupported Bitbucket webhook event of type '%v'", eventType)
	}

	fmt.Fprintf(w, "Aye aye!")
}

// GithubPushEvent represents a Github webhook push event
type GithubPushEvent struct {
	After      string           `json:"after"`
	Commits    []GithubCommit   `json:"commits"`
	HeadCommit GithubCommit     `json:"head_commit"`
	Pusher     GithubPusher     `json:"pusher"`
	Repository GithubRepository `json:"repository"`
}

// GithubCommit represents a Github commit
type GithubCommit struct {
	Author  GithubAuthor `json:"author"`
	Message string       `json:"message"`
	ID      string       `json:"id"`
}

// GithubAuthor represents a Github author
type GithubAuthor struct {
	Email    string `json:"email"`
	Name     string `json:"name"`
	UserName string `json:"username"`
}

// GithubPusher represents a Github pusher
type GithubPusher struct {
	Email string `json:"email"`
	Name  string `json:"name"`
}

// GithubRepository represents a Github repository
type GithubRepository struct {
	GitURL   string `json:"git_url"`
	HTMLURL  string `json:"html_url"`
	Name     string `json:"name"`
	FullName string `json:"full_name"`
}

func handleGithubPush(body []byte) {

	// unmarshal json body
	var pushEvent GithubPushEvent
	err := json.Unmarshal(body, &pushEvent)
	if err != nil {
		log.Error().Err(err).Str("body", string(body)).Msg("Deserializing body to GithubPushEvent failed")
		return
	}

	log.Info().Interface("pushEvent", pushEvent).Msgf("Deserialized GitHub push event for repository %v", pushEvent.Repository.FullName)
}

// BitbucketRepositoryPushEvent represents a Bitbucket webhook push event
type BitbucketRepositoryPushEvent struct {
	Actor      BitbucketOwner      `json:"actor"`
	Repository BitbucketRepository `json:"repository"`
	Push       BitbucketPushEvent  `json:"push"`
}

// BitbucketPushEvent represents a Bitbucket push event push info
type BitbucketPushEvent struct {
	Changes []BitbucketPushEventChange `json:"changes"`
}

// BitbucketPushEventChange represents a Bitbucket push change
type BitbucketPushEventChange struct {
	New       BitbucketPushEventChangeObject `json:"new,omitempty"`
	Old       BitbucketPushEventChangeObject `json:"old,omitempty"`
	Created   bool                           `json:"created"`
	Closed    bool                           `json:"closed"`
	Forced    bool                           `json:"forced"`
	Truncated bool                           `json:"truncated"`
}

// BitbucketPushEventChangeObject represents the state of the reference after a push
type BitbucketPushEventChangeObject struct {
	Type string `json:"type"`
	Name string `json:"name,omitempty"`
}

// BitbucketOwner represents a Bitbucket owern
type BitbucketOwner struct {
	Type        string `json:"type"`
	UserName    string `json:"username"`
	DisplayName string `json:"display_name"`
}

// BitbucketRepository represents a Bitbucket repository
type BitbucketRepository struct {
	Name      string         `json:"name"`
	FullName  string         `json:"full_name"`
	Owner     BitbucketOwner `json:"owner"`
	IsPrivate bool           `json:"is_private"`
	Scm       string         `json:"scm"`
}

func handleBitbucketPush(body []byte) {

	// unmarshal json body
	var pushEvent BitbucketRepositoryPushEvent
	err := json.Unmarshal(body, &pushEvent)
	if err != nil {
		log.Error().Err(err).Str("body", string(body)).Msg("Deserializing body to BitbucketRepositoryPushEvent failed")
		return
	}

	log.Info().Interface("pushEvent", pushEvent).Msgf("Deserialized Bitbucket push event for repository %v", pushEvent.Repository.FullName)

}

func livenessHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "I'm alive!")
}

func readinessHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "I'm ready!")
}
