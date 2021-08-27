package main

import (
	"testing"

	"github.com/estafette/estafette-ci-api/pkg/api"
	"github.com/golang/mock/gomock"

	"github.com/estafette/estafette-ci-api/pkg/clients/bitbucketapi"
	"github.com/estafette/estafette-ci-api/pkg/clients/builderapi"
	"github.com/estafette/estafette-ci-api/pkg/clients/cloudsourceapi"
	"github.com/estafette/estafette-ci-api/pkg/clients/cloudstorage"
	"github.com/estafette/estafette-ci-api/pkg/clients/cockroachdb"
	"github.com/estafette/estafette-ci-api/pkg/clients/githubapi"
	"github.com/estafette/estafette-ci-api/pkg/clients/pubsubapi"
	"github.com/estafette/estafette-ci-api/pkg/clients/slackapi"

	"github.com/estafette/estafette-ci-api/pkg/services/bitbucket"
	"github.com/estafette/estafette-ci-api/pkg/services/catalog"
	"github.com/estafette/estafette-ci-api/pkg/services/cloudsource"
	"github.com/estafette/estafette-ci-api/pkg/services/estafette"
	"github.com/estafette/estafette-ci-api/pkg/services/github"
	"github.com/estafette/estafette-ci-api/pkg/services/pubsub"
	"github.com/estafette/estafette-ci-api/pkg/services/rbac"
	"github.com/estafette/estafette-ci-api/pkg/services/slack"

	crypt "github.com/estafette/estafette-ci-crypt"
)

func TestConfigureGinGonic(t *testing.T) {
	t.Run("DoesNotPanic", func(t *testing.T) {

		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		config := &api.APIConfig{
			Auth: &api.AuthConfig{
				JWT: &api.JWTConfig{
					Domain: "mydomain",
					Key:    "abc",
				},
			},
		}

		cockroachdbClient := cockroachdb.NewMockClient(ctrl)
		cloudstorageClient := cloudstorage.NewMockClient(ctrl)
		builderapiClient := builderapi.NewMockClient(ctrl)
		estafetteService := estafette.NewMockService(ctrl)
		secretHelper := crypt.NewSecretHelper("abc", false)
		warningHelper := api.NewWarningHelper(secretHelper)
		githubapiClient := githubapi.NewMockClient(ctrl)
		bitbucketapiClient := bitbucketapi.NewMockClient(ctrl)
		cloudsourceapiClient := cloudsourceapi.NewMockClient(ctrl)
		pubsubapiclient := pubsubapi.NewMockClient(ctrl)
		slackapiClient := slackapi.NewMockClient(ctrl)

		githubapiClient.EXPECT().JobVarsFunc(gomock.Any()).AnyTimes()
		bitbucketapiClient.EXPECT().JobVarsFunc(gomock.Any()).AnyTimes()
		cloudsourceapiClient.EXPECT().JobVarsFunc(gomock.Any()).AnyTimes()

		bitbucketHandler := bitbucket.NewHandler(bitbucket.NewMockService(ctrl), config, bitbucketapiClient)
		githubHandler := github.NewHandler(github.NewMockService(ctrl))
		estafetteHandler := estafette.NewHandler("", "", config, config, cockroachdbClient, cloudstorageClient, builderapiClient, estafetteService, warningHelper, secretHelper)

		rbacHandler := rbac.NewHandler(config, rbac.NewMockService(ctrl), cockroachdbClient, bitbucketapiClient)
		pubsubHandler := pubsub.NewHandler(pubsubapiclient, estafetteService)
		slackHandler := slack.NewHandler(secretHelper, config, slackapiClient, cockroachdbClient, estafetteService)
		cloudsourceHandler := cloudsource.NewHandler(pubsubapiclient, cloudsource.NewMockService(ctrl))
		catalogHandler := catalog.NewHandler(config, catalog.NewMockService(ctrl), cockroachdbClient)

		// act
		_ = configureGinGonic(config, bitbucketHandler, githubHandler, estafetteHandler, rbacHandler, pubsubHandler, slackHandler, cloudsourceHandler, catalogHandler)
	})
}
