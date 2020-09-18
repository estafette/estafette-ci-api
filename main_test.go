package main

import (
	"context"
	"testing"

	"github.com/estafette/estafette-ci-api/api"

	"github.com/estafette/estafette-ci-api/clients/bitbucketapi"
	"github.com/estafette/estafette-ci-api/clients/builderapi"
	"github.com/estafette/estafette-ci-api/clients/cloudsourceapi"
	"github.com/estafette/estafette-ci-api/clients/cloudstorage"
	"github.com/estafette/estafette-ci-api/clients/cockroachdb"
	"github.com/estafette/estafette-ci-api/clients/githubapi"
	"github.com/estafette/estafette-ci-api/clients/pubsubapi"
	"github.com/estafette/estafette-ci-api/clients/slackapi"

	"github.com/estafette/estafette-ci-api/services/bitbucket"
	"github.com/estafette/estafette-ci-api/services/catalog"
	"github.com/estafette/estafette-ci-api/services/cloudsource"
	"github.com/estafette/estafette-ci-api/services/estafette"
	"github.com/estafette/estafette-ci-api/services/github"
	"github.com/estafette/estafette-ci-api/services/pubsub"
	"github.com/estafette/estafette-ci-api/services/rbac"
	"github.com/estafette/estafette-ci-api/services/slack"

	crypt "github.com/estafette/estafette-ci-crypt"
)

func TestConfigureGinGonic(t *testing.T) {
	t.Run("DoesNotPanic", func(t *testing.T) {

		config := &api.APIConfig{
			Auth: &api.AuthConfig{
				JWT: &api.JWTConfig{
					Domain: "mydomain",
					Key:    "abc",
				},
			},
		}

		ctx := context.Background()

		cockroachdbClient := cockroachdb.MockClient{}
		cloudstorageClient := cloudstorage.MockClient{}
		builderapiClient := builderapi.MockClient{}
		estafetteService := estafette.MockService{}
		secretHelper := crypt.NewSecretHelper("abc", false)
		warningHelper := api.NewWarningHelper(secretHelper)
		githubapiClient := githubapi.MockClient{}
		bitbucketapiClient := bitbucketapi.MockClient{}
		cloudsourceapiClient := cloudsourceapi.MockClient{}
		pubsubapiclient := pubsubapi.MockClient{}
		slackapiClient := slackapi.MockClient{}

		bitbucketHandler := bitbucket.NewHandler(bitbucket.MockService{})
		githubHandler := github.NewHandler(github.MockService{})
		estafetteHandler := estafette.NewHandler("", "", config, config, cockroachdbClient, cloudstorageClient, builderapiClient, estafetteService, warningHelper, secretHelper, githubapiClient.JobVarsFunc(ctx), bitbucketapiClient.JobVarsFunc(ctx), cloudsourceapiClient.JobVarsFunc(ctx))

		rbacHandler := rbac.NewHandler(config, rbac.MockService{}, cockroachdbClient)
		pubsubHandler := pubsub.NewHandler(pubsubapiclient, estafetteService)
		slackHandler := slack.NewHandler(secretHelper, config, slackapiClient, cockroachdbClient, estafetteService, githubapiClient.JobVarsFunc(ctx), bitbucketapiClient.JobVarsFunc(ctx))
		cloudsourceHandler := cloudsource.NewHandler(pubsubapiclient, cloudsource.MockService{})
		catalogHandler := catalog.NewHandler(config, catalog.MockService{}, cockroachdbClient)

		// act
		_ = configureGinGonic(config, bitbucketHandler, githubHandler, estafetteHandler, rbacHandler, pubsubHandler, slackHandler, cloudsourceHandler, catalogHandler)
	})
}
