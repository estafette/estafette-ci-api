package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/rs/zerolog/log"
)

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
	case "integration_installation_repositories": // ?
		log.Debug().Str("event", eventType).Msgf("Not implemented Github webhook event of type '%v'", eventType)

	default:
		log.Warn().Str("event", eventType).Msgf("Unsupported Github webhook event of type '%v'", eventType)
	}

	fmt.Fprintf(w, "Aye aye!")
}

func handleGithubPush(body []byte) {

	// unmarshal json body
	var pushEvent GithubPushEvent
	err := json.Unmarshal(body, &pushEvent)
	if err != nil {
		log.Error().Err(err).Str("body", string(body)).Msg("Deserializing body to GithubPushEvent failed")
		return
	}

	log.Debug().Interface("pushEvent", pushEvent).Msgf("Deserialized GitHub push event for repository %v", pushEvent.Repository.FullName)

	// test making api calls for github app
	ghClient := CreateGithubAPIClient(*githubAppPrivateKeyPath, *githubAppID, *githubAppOAuthClientID, *githubAppOAuthClientSecret)
	//ghClient.getGithubAppDetails()
	//ghClient.getInstallationRepositories(pushEvent.Installation.ID)
	authenticatedRepositoryURL, err := ghClient.getAuthenticatedRepositoryURL(pushEvent.Installation.ID, pushEvent.Repository.HTMLURL)
	if err != nil {
		log.Error().Err(err).
			Msg("Retrieving authenticated repository failed")
		return
	}

	log.Debug().Str("url", authenticatedRepositoryURL).Msgf("Authenticated url for Github repository %v", pushEvent.Repository.FullName)

	// create kubernetes client
	kubernetes, err := NewKubernetesClient()
	if err != nil {
		log.Error().Err(err).Msg("Initializing Kubernetes client failed")
		return
	}

	// create job cloning git repository
	_, err = kubernetes.CreateJob(pushEvent, authenticatedRepositoryURL)
	if err != nil {
		log.Error().Err(err).Msg("Creating Kubernetes job failed")
		return
	}
}
