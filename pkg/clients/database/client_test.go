package database

import (
	"context"
	"errors"
	"os"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/estafette/estafette-ci-api/pkg/api"
	contracts "github.com/estafette/estafette-ci-contracts"
	manifest "github.com/estafette/estafette-ci-manifest"
	"github.com/stretchr/testify/assert"
)

func TestIntegrationGetAutoIncrement(t *testing.T) {
	t.Run("ReturnsAnIncrementingCountForUniqueRepo", func(t *testing.T) {

		if testing.Short() {
			t.Skip("skipping test in short mode.")
		}

		ctx := context.Background()
		databaseClient := getDatabaseClient(ctx, t)

		// act
		autoincrement, err := databaseClient.GetAutoIncrement(ctx, "github", "estafette", "estafette-ci-api")

		assert.Nil(t, err)
		assert.True(t, autoincrement > 0)
	})

	t.Run("ReturnsLargerCountForSubsequentRequests", func(t *testing.T) {

		if testing.Short() {
			t.Skip("skipping test in short mode.")
		}

		ctx := context.Background()
		databaseClient := getDatabaseClient(ctx, t)

		// act
		autoincrement1, err1 := databaseClient.GetAutoIncrement(ctx, "github", "estafette", "estafette-ci-api")
		autoincrement2, err2 := databaseClient.GetAutoIncrement(ctx, "github", "estafette", "estafette-ci-api")

		assert.Nil(t, err1)
		assert.Nil(t, err2)
		assert.True(t, autoincrement1 > 0)
		assert.True(t, autoincrement2 > 0)
		assert.True(t, autoincrement2 > autoincrement1)
	})
}

func TestIntegrationInsertBuild(t *testing.T) {
	t.Run("ReturnsInsertedBuildWithID", func(t *testing.T) {

		if testing.Short() {
			t.Skip("skipping test in short mode.")
		}

		ctx := context.Background()
		databaseClient := getDatabaseClient(ctx, t)
		build := getBuild()
		jobResources := getJobResources()

		// act
		insertedBuild, err := databaseClient.InsertBuild(ctx, build, jobResources)

		assert.Nil(t, err)
		assert.NotNil(t, insertedBuild)
		assert.True(t, insertedBuild.ID != "")
	})
}

func TestIntegrationUpdateBuildStatus(t *testing.T) {
	t.Run("UpdatesStatusForInsertedBuild", func(t *testing.T) {

		if testing.Short() {
			t.Skip("skipping test in short mode.")
		}

		ctx := context.Background()
		databaseClient := getDatabaseClient(ctx, t)
		build := getBuild()
		jobResources := getJobResources()
		insertedBuild, err := databaseClient.InsertBuild(ctx, build, jobResources)
		assert.Nil(t, err)

		// act
		err = databaseClient.UpdateBuildStatus(ctx, insertedBuild.RepoSource, insertedBuild.RepoOwner, insertedBuild.RepoName, insertedBuild.ID, contracts.StatusSucceeded)

		assert.Nil(t, err)
	})

	t.Run("UpdatesStatusForNonExistingBuild", func(t *testing.T) {

		if testing.Short() {
			t.Skip("skipping test in short mode.")
		}

		ctx := context.Background()
		databaseClient := getDatabaseClient(ctx, t)
		build := getBuild()
		buildID := "15"

		// act
		err := databaseClient.UpdateBuildStatus(ctx, build.RepoSource, build.RepoOwner, build.RepoName, buildID, contracts.StatusSucceeded)

		assert.Nil(t, err)
	})
}

func TestIntegrationGetBuildsDuration(t *testing.T) {
	t.Run("RetrievesBuildsDuration", func(t *testing.T) {

		if testing.Short() {
			t.Skip("skipping test in short mode.")
		}

		ctx := context.Background()
		databaseClient := getDatabaseClient(ctx, t)
		build := getBuild()
		jobResources := getJobResources()
		insertedBuild, err := databaseClient.InsertBuild(ctx, build, jobResources)
		assert.Nil(t, err)
		err = databaseClient.UpdateBuildStatus(ctx, insertedBuild.RepoSource, insertedBuild.RepoOwner, insertedBuild.RepoName, insertedBuild.ID, contracts.StatusSucceeded)
		assert.Nil(t, err)

		// act
		duration, err := databaseClient.GetBuildsDuration(ctx, make(map[api.FilterType][]string))

		assert.Nil(t, err)
		assert.True(t, duration.Milliseconds() >= 0)
	})
}

func TestIntegrationUpdateBuildResourceUtilization(t *testing.T) {
	t.Run("UpdatesJobResourceForInsertedBuild", func(t *testing.T) {

		if testing.Short() {
			t.Skip("skipping test in short mode.")
		}

		ctx := context.Background()
		databaseClient := getDatabaseClient(ctx, t)
		build := getBuild()
		jobResources := getJobResources()
		insertedBuild, err := databaseClient.InsertBuild(ctx, build, jobResources)
		assert.Nil(t, err)

		newJobResources := JobResources{
			CPURequest:    float64(0.3),
			CPULimit:      float64(4.0),
			MemoryRequest: float64(67108864),
			MemoryLimit:   float64(21474836480),
		}

		// act
		err = databaseClient.UpdateBuildResourceUtilization(ctx, insertedBuild.RepoSource, insertedBuild.RepoOwner, insertedBuild.RepoName, insertedBuild.ID, newJobResources)

		assert.Nil(t, err)
	})

	t.Run("UpdatesJobResourcesForNonExistingBuild", func(t *testing.T) {

		if testing.Short() {
			t.Skip("skipping test in short mode.")
		}

		ctx := context.Background()
		databaseClient := getDatabaseClient(ctx, t)
		build := getBuild()
		buildID := "15"

		newJobResources := JobResources{
			CPURequest:    float64(0.3),
			CPULimit:      float64(4.0),
			MemoryRequest: float64(67108864),
			MemoryLimit:   float64(21474836480),
		}

		// act
		err := databaseClient.UpdateBuildResourceUtilization(ctx, build.RepoSource, build.RepoOwner, build.RepoName, buildID, newJobResources)

		assert.Nil(t, err)
	})
}

func TestIntegrationInsertRelease(t *testing.T) {
	t.Run("ReturnsInsertedReleaseWithID", func(t *testing.T) {

		if testing.Short() {
			t.Skip("skipping test in short mode.")
		}

		ctx := context.Background()
		databaseClient := getDatabaseClient(ctx, t)
		release := getRelease()
		jobResources := getJobResources()

		// act
		insertedRelease, err := databaseClient.InsertRelease(ctx, release, jobResources)

		assert.Nil(t, err)
		assert.NotNil(t, insertedRelease)
		assert.True(t, insertedRelease.ID != "")
	})
}

func TestIntegrationUpdateReleaseStatus(t *testing.T) {
	t.Run("UpdatesStatusForInsertedRelease", func(t *testing.T) {

		if testing.Short() {
			t.Skip("skipping test in short mode.")
		}

		ctx := context.Background()
		databaseClient := getDatabaseClient(ctx, t)
		release := getRelease()
		release.ReleaseStatus = contracts.StatusPending
		jobResources := getJobResources()
		insertedRelease, err := databaseClient.InsertRelease(ctx, release, jobResources)
		assert.Nil(t, err)

		// act
		err = databaseClient.UpdateReleaseStatus(ctx, insertedRelease.RepoSource, insertedRelease.RepoOwner, insertedRelease.RepoName, insertedRelease.ID, contracts.StatusRunning)

		assert.Nil(t, err)
	})

	t.Run("UpdatesStatusForNonExistingRelease", func(t *testing.T) {

		if testing.Short() {
			t.Skip("skipping test in short mode.")
		}

		ctx := context.Background()
		databaseClient := getDatabaseClient(ctx, t)
		release := getRelease()
		releaseID := "15"

		// act
		err := databaseClient.UpdateReleaseStatus(ctx, release.RepoSource, release.RepoOwner, release.RepoName, releaseID, contracts.StatusRunning)

		assert.Nil(t, err)
	})
}

func TestIntegrationUpdateReleaseResourceUtilization(t *testing.T) {
	t.Run("UpdatesJobResourceForInsertedBuild", func(t *testing.T) {

		if testing.Short() {
			t.Skip("skipping test in short mode.")
		}

		ctx := context.Background()
		databaseClient := getDatabaseClient(ctx, t)
		release := getRelease()
		jobResources := getJobResources()
		insertedRelease, err := databaseClient.InsertRelease(ctx, release, jobResources)
		assert.Nil(t, err)

		newJobResources := JobResources{
			CPURequest:    float64(0.3),
			CPULimit:      float64(4.0),
			MemoryRequest: float64(67108864),
			MemoryLimit:   float64(21474836480),
		}

		// act
		err = databaseClient.UpdateReleaseResourceUtilization(ctx, release.RepoSource, release.RepoOwner, release.RepoName, insertedRelease.ID, newJobResources)

		assert.Nil(t, err)
	})

	t.Run("UpdatesJobResourcesForNonExistingBuild", func(t *testing.T) {

		if testing.Short() {
			t.Skip("skipping test in short mode.")
		}

		ctx := context.Background()
		databaseClient := getDatabaseClient(ctx, t)
		release := getRelease()
		releaseID := "15"

		newJobResources := JobResources{
			CPURequest:    float64(0.3),
			CPULimit:      float64(4.0),
			MemoryRequest: float64(67108864),
			MemoryLimit:   float64(21474836480),
		}

		// act
		err := databaseClient.UpdateReleaseResourceUtilization(ctx, release.RepoSource, release.RepoOwner, release.RepoName, releaseID, newJobResources)

		assert.Nil(t, err)
	})
}

func TestIntegrationInsertBot(t *testing.T) {
	t.Run("ReturnsInsertedBotWithID", func(t *testing.T) {

		if testing.Short() {
			t.Skip("skipping test in short mode.")
		}

		ctx := context.Background()
		databaseClient := getDatabaseClient(ctx, t)
		bot := getBot()
		jobResources := getJobResources()

		// act
		insertedBot, err := databaseClient.InsertBot(ctx, bot, jobResources)

		assert.Nil(t, err)
		assert.NotNil(t, insertedBot)
		assert.True(t, insertedBot.ID != "")
	})
}

func TestIntegrationUpdateBotStatus(t *testing.T) {
	t.Run("UpdatesStatusForInsertedBot", func(t *testing.T) {

		if testing.Short() {
			t.Skip("skipping test in short mode.")
		}

		ctx := context.Background()
		databaseClient := getDatabaseClient(ctx, t)
		bot := getBot()
		jobResources := getJobResources()
		insertedBot, err := databaseClient.InsertBot(ctx, bot, jobResources)
		assert.Nil(t, err)

		// act
		err = databaseClient.UpdateBotStatus(ctx, insertedBot.RepoSource, insertedBot.RepoOwner, insertedBot.RepoName, insertedBot.ID, contracts.StatusSucceeded)

		assert.Nil(t, err)
	})

	t.Run("UpdatesStatusForNonExistingBot", func(t *testing.T) {

		if testing.Short() {
			t.Skip("skipping test in short mode.")
		}

		ctx := context.Background()
		databaseClient := getDatabaseClient(ctx, t)
		bot := getBot()
		botID := "15"

		// act
		err := databaseClient.UpdateBotStatus(ctx, bot.RepoSource, bot.RepoOwner, bot.RepoName, botID, contracts.StatusSucceeded)

		assert.Nil(t, err)
	})
}

func TestIntegrationUpdateBotResourceUtilization(t *testing.T) {
	t.Run("UpdatesJobResourceForInsertedBot", func(t *testing.T) {

		if testing.Short() {
			t.Skip("skipping test in short mode.")
		}

		ctx := context.Background()
		databaseClient := getDatabaseClient(ctx, t)
		bot := getBot()
		jobResources := getJobResources()
		insertedBot, err := databaseClient.InsertBot(ctx, bot, jobResources)
		assert.Nil(t, err)

		newJobResources := JobResources{
			CPURequest:    float64(0.3),
			CPULimit:      float64(4.0),
			MemoryRequest: float64(67108864),
			MemoryLimit:   float64(21474836480),
		}

		// act
		err = databaseClient.UpdateBotResourceUtilization(ctx, insertedBot.RepoSource, insertedBot.RepoOwner, insertedBot.RepoName, insertedBot.ID, newJobResources)

		assert.Nil(t, err)
	})

	t.Run("UpdatesJobResourcesForNonExistingBot", func(t *testing.T) {

		if testing.Short() {
			t.Skip("skipping test in short mode.")
		}

		ctx := context.Background()
		databaseClient := getDatabaseClient(ctx, t)
		bot := getBot()
		botID := "15"

		newJobResources := JobResources{
			CPURequest:    float64(0.3),
			CPULimit:      float64(4.0),
			MemoryRequest: float64(67108864),
			MemoryLimit:   float64(21474836480),
		}

		// act
		err := databaseClient.UpdateBotResourceUtilization(ctx, bot.RepoSource, bot.RepoOwner, bot.RepoName, botID, newJobResources)

		assert.Nil(t, err)
	})
}

func TestIntegrationInsertBuildLog(t *testing.T) {

	t.Run("ReturnsInsertedBuildLogWithIDWhenWriteLogToDatabaseIsFalse", func(t *testing.T) {

		if testing.Short() {
			t.Skip("skipping test in short mode.")
		}

		ctx := context.Background()
		databaseClient := getDatabaseClient(ctx, t)
		buildLog := getBuildLog()

		// act
		insertedBuildLog, err := databaseClient.InsertBuildLog(ctx, buildLog)

		assert.Nil(t, err)
		assert.NotNil(t, insertedBuildLog)
		assert.True(t, insertedBuildLog.ID != "")
	})
}

func TestIntegrationInsertReleaseLog(t *testing.T) {

	t.Run("ReturnsInsertedReleaseLogWithIDWhenWriteLogToDatabaseIsFalse", func(t *testing.T) {

		if testing.Short() {
			t.Skip("skipping test in short mode.")
		}

		ctx := context.Background()
		databaseClient := getDatabaseClient(ctx, t)
		releaseLog := getReleaseLog()

		// act
		insertedReleaseLog, err := databaseClient.InsertReleaseLog(ctx, releaseLog)

		assert.Nil(t, err)
		assert.NotNil(t, insertedReleaseLog)
		assert.True(t, insertedReleaseLog.ID != "")
	})
}

func TestIntegrationInsertBotLog(t *testing.T) {

	t.Run("ReturnsInsertedBotLogWithIDWhenWriteLogToDatabaseIsFalse", func(t *testing.T) {

		if testing.Short() {
			t.Skip("skipping test in short mode.")
		}

		ctx := context.Background()
		databaseClient := getDatabaseClient(ctx, t)
		botLog := getBotLog()

		// act
		insertedBotLog, err := databaseClient.InsertBotLog(ctx, botLog)

		assert.Nil(t, err)
		assert.NotNil(t, insertedBotLog)
		assert.True(t, insertedBotLog.ID != "")
	})
}

func TestIngrationGetPipelines(t *testing.T) {
	t.Run("ReturnsNoError", func(t *testing.T) {

		if testing.Short() {
			t.Skip("skipping test in short mode.")
		}

		ctx := context.Background()
		databaseClient := getDatabaseClient(ctx, t)
		build := getBuild()
		jobResources := getJobResources()
		_, err := databaseClient.InsertBuild(ctx, build, jobResources)
		assert.Nil(t, err)
		err = databaseClient.UpdateComputedTables(ctx, build.RepoSource, build.RepoOwner, build.RepoName)
		assert.Nil(t, err)

		// act
		pipelines, err := databaseClient.GetPipelines(ctx, 1, 10, map[api.FilterType][]string{}, []api.OrderField{}, false)

		assert.Nil(t, err)
		assert.True(t, len(pipelines) > 0)
	})

	t.Run("ReturnsPipelinesForSortings", func(t *testing.T) {

		if testing.Short() {
			t.Skip("skipping test in short mode.")
		}

		ctx := context.Background()
		databaseClient := getDatabaseClient(ctx, t)
		build := getBuild()
		jobResources := getJobResources()
		_, err := databaseClient.InsertBuild(ctx, build, jobResources)
		assert.Nil(t, err)
		err = databaseClient.UpdateComputedTables(ctx, build.RepoSource, build.RepoOwner, build.RepoName)
		assert.Nil(t, err)

		sortings := []api.OrderField{
			api.OrderField{
				FieldName: "inserted_at",
				Direction: "DESC",
			},
		}

		// act
		pipelines, err := databaseClient.GetPipelines(ctx, 1, 10, map[api.FilterType][]string{}, sortings, false)

		assert.Nil(t, err)
		assert.True(t, len(pipelines) > 0)
	})

	t.Run("ReturnsPipelinesForSinceFilter", func(t *testing.T) {

		if testing.Short() {
			t.Skip("skipping test in short mode.")
		}

		ctx := context.Background()
		databaseClient := getDatabaseClient(ctx, t)
		build := getBuild()
		jobResources := getJobResources()
		_, err := databaseClient.InsertBuild(ctx, build, jobResources)
		assert.Nil(t, err)
		err = databaseClient.UpdateComputedTables(ctx, build.RepoSource, build.RepoOwner, build.RepoName)
		assert.Nil(t, err)

		filters := map[api.FilterType][]string{
			api.FilterSince: {"1h"},
		}

		// act
		pipelines, err := databaseClient.GetPipelines(ctx, 1, 10, filters, []api.OrderField{}, false)

		assert.Nil(t, err)
		assert.True(t, len(pipelines) > 0)
	})

	t.Run("ReturnsPipelinesForStatusFilter", func(t *testing.T) {

		if testing.Short() {
			t.Skip("skipping test in short mode.")
		}

		ctx := context.Background()
		databaseClient := getDatabaseClient(ctx, t)
		build := getBuild()
		jobResources := getJobResources()
		_, err := databaseClient.InsertBuild(ctx, build, jobResources)
		assert.Nil(t, err)
		err = databaseClient.UpdateComputedTables(ctx, build.RepoSource, build.RepoOwner, build.RepoName)
		assert.Nil(t, err)

		filters := map[api.FilterType][]string{
			api.FilterStatus: {string(contracts.StatusSucceeded)},
		}

		// act
		pipelines, err := databaseClient.GetPipelines(ctx, 1, 10, filters, []api.OrderField{}, false)

		assert.Nil(t, err)
		assert.True(t, len(pipelines) > 0)
	})

	t.Run("ReturnsPipelinesForLabelsFilter", func(t *testing.T) {

		if testing.Short() {
			t.Skip("skipping test in short mode.")
		}

		ctx := context.Background()
		databaseClient := getDatabaseClient(ctx, t)
		build := getBuild()
		jobResources := getJobResources()
		_, err := databaseClient.InsertBuild(ctx, build, jobResources)
		assert.Nil(t, err)
		err = databaseClient.UpdateComputedTables(ctx, build.RepoSource, build.RepoOwner, build.RepoName)
		assert.Nil(t, err)

		filters := map[api.FilterType][]string{
			api.FilterLabels: {"app-group=estafette-ci"},
		}

		// act
		pipelines, err := databaseClient.GetPipelines(ctx, 1, 10, filters, []api.OrderField{}, false)

		assert.Nil(t, err)
		assert.True(t, len(pipelines) > 0)
	})

	t.Run("ReturnsPipelinesForSearchFilter", func(t *testing.T) {

		if testing.Short() {
			t.Skip("skipping test in short mode.")
		}

		ctx := context.Background()
		databaseClient := getDatabaseClient(ctx, t)
		build := getBuild()
		jobResources := getJobResources()
		_, err := databaseClient.InsertBuild(ctx, build, jobResources)
		assert.Nil(t, err)
		err = databaseClient.UpdateComputedTables(ctx, build.RepoSource, build.RepoOwner, build.RepoName)
		assert.Nil(t, err)

		filters := map[api.FilterType][]string{
			api.FilterSearch: {"ci-api"},
		}

		// act
		pipelines, err := databaseClient.GetPipelines(ctx, 1, 10, filters, []api.OrderField{}, false)

		assert.Nil(t, err)
		assert.True(t, len(pipelines) > 0)
	})

	t.Run("ReturnsPipelinesForRecentCommitterFilter", func(t *testing.T) {

		if testing.Short() {
			t.Skip("skipping test in short mode.")
		}

		ctx := context.Background()
		databaseClient := getDatabaseClient(ctx, t)
		build := getBuild()
		jobResources := getJobResources()
		_, err := databaseClient.InsertBuild(ctx, build, jobResources)
		assert.Nil(t, err)
		err = databaseClient.UpdateComputedTables(ctx, build.RepoSource, build.RepoOwner, build.RepoName)
		assert.Nil(t, err)

		filters := map[api.FilterType][]string{
			api.FilterRecentCommitter: {"me@estafette.io"},
		}

		// act
		pipelines, err := databaseClient.GetPipelines(ctx, 1, 10, filters, []api.OrderField{}, false)

		assert.Nil(t, err)
		assert.True(t, len(pipelines) > 0)
	})

	t.Run("ReturnsPipelinesForRecentReleaserFilter", func(t *testing.T) {

		if testing.Short() {
			t.Skip("skipping test in short mode.")
		}

		ctx := context.Background()
		databaseClient := getDatabaseClient(ctx, t)
		build := getBuild()
		jobResources := getJobResources()
		_, err := databaseClient.InsertBuild(ctx, build, jobResources)
		assert.Nil(t, err)

		release := getRelease()
		release.Events = []manifest.EstafetteEvent{
			{
				Fired: true,
				Manual: &manifest.EstafetteManualEvent{
					UserID: "me@estafette.io",
				},
			},
		}
		_, err = databaseClient.InsertRelease(ctx, release, jobResources)
		assert.Nil(t, err)
		err = databaseClient.UpsertComputedRelease(ctx, release.RepoSource, release.RepoOwner, release.RepoName, release.Name, release.Action)
		assert.Nil(t, err)
		err = databaseClient.UpdateComputedTables(ctx, build.RepoSource, build.RepoOwner, build.RepoName)
		assert.Nil(t, err)

		filters := map[api.FilterType][]string{
			api.FilterRecentReleaser: {"me@estafette.io"},
		}

		// act
		pipelines, err := databaseClient.GetPipelines(ctx, 1, 10, filters, []api.OrderField{}, false)

		assert.Nil(t, err)
		assert.True(t, len(pipelines) > 0)
	})

	t.Run("ReturnsPipelinesForGroupsFilter", func(t *testing.T) {

		if testing.Short() {
			t.Skip("skipping test in short mode.")
		}

		ctx := context.Background()
		databaseClient := getDatabaseClient(ctx, t)
		build := getBuild()
		jobResources := getJobResources()
		_, err := databaseClient.InsertBuild(ctx, build, jobResources)
		assert.Nil(t, err)
		err = databaseClient.UpdateComputedTables(ctx, build.RepoSource, build.RepoOwner, build.RepoName)
		assert.Nil(t, err)
		pipeline := contracts.Pipeline{
			RepoSource: build.RepoSource,
			RepoOwner:  build.RepoOwner,
			RepoName:   build.RepoName,
			Groups: []*contracts.Group{
				{
					ID:   "123123",
					Name: "my group",
				},
			},
		}
		err = databaseClient.UpdateComputedPipelinePermissions(ctx, pipeline)
		assert.Nil(t, err)

		filters := map[api.FilterType][]string{
			api.FilterGroups: {"my group", "my other group"},
		}

		// act
		pipelines, err := databaseClient.GetPipelines(ctx, 1, 10, filters, []api.OrderField{}, false)

		assert.Nil(t, err)
		assert.True(t, len(pipelines) > 0)
		for _, p := range pipelines {
			for _, g := range p.Groups {
				assert.Equal(t, "my group", g.Name)
			}
		}
	})

	t.Run("ReturnsPipelinesForOrganizationsFilter", func(t *testing.T) {

		if testing.Short() {
			t.Skip("skipping test in short mode.")
		}

		ctx := context.Background()
		databaseClient := getDatabaseClient(ctx, t)
		build := getBuild()
		jobResources := getJobResources()
		_, err := databaseClient.InsertBuild(ctx, build, jobResources)
		assert.Nil(t, err)
		err = databaseClient.UpdateComputedTables(ctx, build.RepoSource, build.RepoOwner, build.RepoName)
		assert.Nil(t, err)
		pipeline := contracts.Pipeline{
			RepoSource: build.RepoSource,
			RepoOwner:  build.RepoOwner,
			RepoName:   build.RepoName,
			Organizations: []*contracts.Organization{
				{
					ID:   "234423435",
					Name: "my org",
				},
			},
		}
		err = databaseClient.UpdateComputedPipelinePermissions(ctx, pipeline)
		assert.Nil(t, err)

		filters := map[api.FilterType][]string{
			api.FilterOrganizations: {"my org", "my other org"},
		}

		// act
		pipelines, err := databaseClient.GetPipelines(ctx, 1, 10, filters, []api.OrderField{}, false)

		assert.Nil(t, err)
		assert.True(t, len(pipelines) > 0)
		for _, p := range pipelines {
			for _, o := range p.Organizations {
				assert.Equal(t, "my org", o.Name)
			}
		}
	})
}

func TestIngrationUpsertComputedPipeline(t *testing.T) {
	t.Run("ReturnsNoError", func(t *testing.T) {

		if testing.Short() {
			t.Skip("skipping test in short mode.")
		}

		ctx := context.Background()
		databaseClient := getDatabaseClient(ctx, t)
		build := getBuild()
		jobResources := getJobResources()
		_, err := databaseClient.InsertBuild(ctx, build, jobResources)
		assert.Nil(t, err)

		// act
		err = databaseClient.UpdateComputedTables(ctx, build.RepoSource, build.RepoOwner, build.RepoName)

		assert.Nil(t, err)
	})
}

func TestIngrationUpdateComputedPipelinePermissions(t *testing.T) {
	t.Run("ReturnsNoError", func(t *testing.T) {

		if testing.Short() {
			t.Skip("skipping test in short mode.")
		}

		ctx := context.Background()
		databaseClient := getDatabaseClient(ctx, t)
		build := getBuild()
		jobResources := getJobResources()
		_, err := databaseClient.InsertBuild(ctx, build, jobResources)
		assert.Nil(t, err)

		pipeline := contracts.Pipeline{
			RepoSource: build.RepoSource,
			RepoOwner:  build.RepoOwner,
			RepoName:   build.RepoName,
			Groups: []*contracts.Group{
				{
					ID:   "123123",
					Name: "my group",
				},
			},
			Organizations: []*contracts.Organization{
				{
					ID:   "234423435",
					Name: "my org",
				},
			},
		}

		// act
		err = databaseClient.UpdateComputedPipelinePermissions(ctx, pipeline)

		assert.Nil(t, err)
	})
}

func TestIngrationUpsertComputedRelease(t *testing.T) {
	t.Run("ReturnsNoError", func(t *testing.T) {

		if testing.Short() {
			t.Skip("skipping test in short mode.")
		}

		ctx := context.Background()
		databaseClient := getDatabaseClient(ctx, t)
		release := getRelease()
		jobResources := getJobResources()
		_, err := databaseClient.InsertRelease(ctx, release, jobResources)
		assert.Nil(t, err)

		assert.Nil(t, err)

		// act
		err = databaseClient.UpsertComputedRelease(ctx, release.RepoSource, release.RepoOwner, release.RepoName, release.Name, release.Action)

		assert.Nil(t, err)
	})
}

func TestIngrationArchiveComputedPipeline(t *testing.T) {
	t.Run("ReturnsNoErrorIfArchivalSucceeds", func(t *testing.T) {

		if testing.Short() {
			t.Skip("skipping test in short mode.")
		}

		ctx := context.Background()
		databaseClient := getDatabaseClient(ctx, t)
		build := getBuild()
		build.RepoName = "archival-test"
		build.Labels = []contracts.Label{{Key: "test", Value: "archival"}}
		jobResources := getJobResources()
		_, err := databaseClient.InsertBuild(ctx, build, jobResources)
		assert.Nil(t, err)
		err = databaseClient.UpdateComputedTables(ctx, build.RepoSource, build.RepoOwner, build.RepoName)
		assert.Nil(t, err)

		// act
		err = databaseClient.ArchiveComputedPipeline(ctx, build.RepoSource, build.RepoOwner, build.RepoName)

		assert.Nil(t, err)
	})

	t.Run("ReturnsNoPipelineAfterArchivalSucceeds", func(t *testing.T) {

		if testing.Short() {
			t.Skip("skipping test in short mode.")
		}

		ctx := context.Background()
		databaseClient := getDatabaseClient(ctx, t)
		build := getBuild()
		build.RepoName = "archival-test-2"
		build.Labels = []contracts.Label{{Key: "test", Value: "archival"}}
		jobResources := getJobResources()
		_, err := databaseClient.InsertBuild(ctx, build, jobResources)
		assert.Nil(t, err)
		err = databaseClient.UpdateComputedTables(ctx, build.RepoSource, build.RepoOwner, build.RepoName)
		assert.Nil(t, err)
		err = databaseClient.ArchiveComputedPipeline(ctx, build.RepoSource, build.RepoOwner, build.RepoName)
		assert.Nil(t, err)

		// act
		pipelines, err := databaseClient.GetPipelinesByRepoName(ctx, build.RepoName, false)

		assert.Nil(t, err)
		assert.Equal(t, 0, len(pipelines))
	})
}

func TestIngrationUnarchiveComputedPipeline(t *testing.T) {
	t.Run("ReturnsNoErrorIfUnarchivalSucceeds", func(t *testing.T) {

		if testing.Short() {
			t.Skip("skipping test in short mode.")
		}

		ctx := context.Background()
		databaseClient := getDatabaseClient(ctx, t)
		build := getBuild()
		build.RepoName = "unarchival-test"
		build.Labels = []contracts.Label{{Key: "test", Value: "unarchival"}}
		jobResources := getJobResources()
		_, err := databaseClient.InsertBuild(ctx, build, jobResources)
		assert.Nil(t, err)
		err = databaseClient.UpdateComputedTables(ctx, build.RepoSource, build.RepoOwner, build.RepoName)
		assert.Nil(t, err)
		err = databaseClient.ArchiveComputedPipeline(ctx, build.RepoSource, build.RepoOwner, build.RepoName)
		assert.Nil(t, err)

		// act
		err = databaseClient.UnarchiveComputedPipeline(ctx, build.RepoSource, build.RepoOwner, build.RepoName)

		assert.Nil(t, err)
	})

	t.Run("ReturnsNoPipelineAfterArchivalSucceeds", func(t *testing.T) {

		if testing.Short() {
			t.Skip("skipping test in short mode.")
		}

		ctx := context.Background()
		databaseClient := getDatabaseClient(ctx, t)
		build := getBuild()
		build.RepoName = "unarchival-test-2"
		build.Labels = []contracts.Label{{Key: "test", Value: "unarchival"}}
		jobResources := getJobResources()
		_, err := databaseClient.InsertBuild(ctx, build, jobResources)
		assert.Nil(t, err)
		err = databaseClient.UpdateComputedTables(ctx, build.RepoSource, build.RepoOwner, build.RepoName)
		assert.Nil(t, err)
		err = databaseClient.ArchiveComputedPipeline(ctx, build.RepoSource, build.RepoOwner, build.RepoName)
		assert.Nil(t, err)
		err = databaseClient.UnarchiveComputedPipeline(ctx, build.RepoSource, build.RepoOwner, build.RepoName)
		assert.Nil(t, err)

		// act
		pipelines, err := databaseClient.GetPipelinesByRepoName(ctx, build.RepoName, false)

		assert.Nil(t, err)
		assert.Equal(t, 1, len(pipelines))
	})
}
func TestIntegrationGetLabelValues(t *testing.T) {

	t.Run("ReturnsLabelValuesForMatchingLabelKey", func(t *testing.T) {

		if testing.Short() {
			t.Skip("skipping test in short mode.")
		}

		ctx := context.Background()
		databaseClient := getDatabaseClient(ctx, t)
		jobResources := getJobResources()
		build := getBuild()
		build.RepoName = "labels-test-1"
		build.Labels = []contracts.Label{{Key: "type", Value: "api"}}
		_, err := databaseClient.InsertBuild(ctx, build, jobResources)
		assert.Nil(t, err, "failed inserting first build record")

		otherBuild := getBuild()
		otherBuild.RepoName = "labels-test-2"
		otherBuild.Labels = []contracts.Label{{Key: "type", Value: "web"}}
		_, err = databaseClient.InsertBuild(ctx, otherBuild, jobResources)
		assert.Nil(t, err, "failed inserting other build record")

		// ensure computed_pipelines are updated in time (they run as a goroutine, so unpredictable when they're finished)
		err = databaseClient.UpdateComputedTables(ctx, build.RepoSource, build.RepoOwner, build.RepoName)
		assert.Nil(t, err, "failed upserting computed pipeline")
		err = databaseClient.UpdateComputedTables(ctx, otherBuild.RepoSource, otherBuild.RepoOwner, otherBuild.RepoName)
		assert.Nil(t, err, "failed upserting computed pipeline for other build")

		// act
		labels, err := databaseClient.GetLabelValues(ctx, "type")

		assert.Nil(t, err, "failed getting label values")
		if !assert.Equal(t, 2, len(labels)) {
			assert.Equal(t, "", labels)
		}
	})
}

func TestIntegrationGetFrequentLabels(t *testing.T) {
	t.Run("ReturnsFrequentLabelsForMatchingLabels", func(t *testing.T) {

		if testing.Short() {
			t.Skip("skipping test in short mode.")
		}

		ctx := context.Background()
		databaseClient := getDatabaseClient(ctx, t)
		jobResources := getJobResources()
		build := getBuild()
		build.RepoName = "frequent-label-test-1"
		build.Labels = []contracts.Label{{Key: "label-test", Value: "GetFrequentLabels"}}
		_, err := databaseClient.InsertBuild(ctx, build, jobResources)
		assert.Nil(t, err, "failed inserting first build record")

		otherBuild := getBuild()
		otherBuild.RepoName = "frequent-label-test-2"
		otherBuild.Labels = []contracts.Label{{Key: "label-test", Value: "GetFrequentLabels"}}
		_, err = databaseClient.InsertBuild(ctx, otherBuild, jobResources)
		assert.Nil(t, err, "failed inserting other build record")

		filters := map[api.FilterType][]string{
			api.FilterLabels: {
				"label-test=GetFrequentLabels",
			},
			api.FilterSince: {
				"1d",
			},
		}

		// ensure computed_pipelines are updated in time (they run as a goroutine, so unpredictable when they're finished)
		err = databaseClient.UpdateComputedTables(ctx, build.RepoSource, build.RepoOwner, build.RepoName)
		assert.Nil(t, err, "failed upserting computed pipeline")
		err = databaseClient.UpdateComputedTables(ctx, otherBuild.RepoSource, otherBuild.RepoOwner, otherBuild.RepoName)
		assert.Nil(t, err, "failed upserting computed pipeline for other build")

		// act
		labels, err := databaseClient.GetFrequentLabels(ctx, 1, 10, filters)

		assert.Nil(t, err, "failed getting frequent label")
		if !assert.Equal(t, 1, len(labels)) {
			assert.Equal(t, "", labels)
		}
	})
}

func TestIntegrationGetFrequentLabelsCount(t *testing.T) {
	t.Run("ReturnsFrequentLabelCountForMatchingLabels", func(t *testing.T) {

		if testing.Short() {
			t.Skip("skipping test in short mode.")
		}

		ctx := context.Background()
		databaseClient := getDatabaseClient(ctx, t)
		jobResources := getJobResources()
		build := getBuild()
		build.RepoName = "frequent-label-count-test-1"
		build.Labels = []contracts.Label{{Key: "label-count-test", Value: "GetFrequentLabelsCount"}}
		_, err := databaseClient.InsertBuild(ctx, build, jobResources)
		assert.Nil(t, err, "failed inserting build record")

		otherBuild := getBuild()
		otherBuild.RepoName = "frequent-label-count-test-2"
		otherBuild.Labels = []contracts.Label{{Key: "label-count-test", Value: "GetFrequentLabelsCount"}}
		_, err = databaseClient.InsertBuild(ctx, otherBuild, jobResources)
		assert.Nil(t, err, "failed inserting other build record")

		filters := map[api.FilterType][]string{
			api.FilterLabels: {
				"label-count-test=GetFrequentLabelsCount",
			},
			api.FilterSince: {
				"1d",
			},
		}

		// ensure computed_pipelines are updated in time (they run as a goroutine, so unpredictable when they're finished)
		err = databaseClient.UpdateComputedTables(ctx, build.RepoSource, build.RepoOwner, build.RepoName)
		assert.Nil(t, err, "failed upserting computed pipeline")
		err = databaseClient.UpdateComputedTables(ctx, otherBuild.RepoSource, otherBuild.RepoOwner, otherBuild.RepoName)
		assert.Nil(t, err, "failed upserting computed pipeline for other build")

		// act
		count, err := databaseClient.GetFrequentLabelsCount(ctx, filters)

		assert.Nil(t, err, "failed getting frequent label count")
		assert.Equal(t, 1, count)
	})
}

func TestIntegrationGetAllPipelinesReleaseTargets(t *testing.T) {
	t.Run("ReturnsReleaseTargetsForMatchingTargets", func(t *testing.T) {

		if testing.Short() {
			t.Skip("skipping test in short mode.")
		}

		ctx := context.Background()
		databaseClient := getDatabaseClient(ctx, t)
		jobResources := getJobResources()
		build := getBuild()
		build.RepoName = "release-targets-test-1"
		build.Labels = []contracts.Label{{Key: "release-targets-test", Value: "GetAllPipelinesReleaseTargets"}}
		build.ReleaseTargets = []contracts.ReleaseTarget{{
			Name: "GetAllPipelinesReleaseTargets",
		}}
		_, err := databaseClient.InsertBuild(ctx, build, jobResources)
		assert.Nil(t, err, "failed inserting first build record")

		otherBuild := getBuild()
		otherBuild.RepoName = "release-targets-test-2"
		otherBuild.Labels = []contracts.Label{{Key: "release-targets-test", Value: "GetAllPipelinesReleaseTargets"}}
		otherBuild.ReleaseTargets = []contracts.ReleaseTarget{{
			Name: "GetAllPipelinesReleaseTargets",
		}}
		_, err = databaseClient.InsertBuild(ctx, otherBuild, jobResources)
		assert.Nil(t, err, "failed inserting other build record")

		filters := map[api.FilterType][]string{
			api.FilterReleaseTarget: {
				"GetAllPipelinesReleaseTargets",
			},
			api.FilterLabels: {
				"release-targets-test=GetAllPipelinesReleaseTargets",
			},
			api.FilterSince: {
				"1d",
			},
		}

		// ensure computed_pipelines are updated in time (they run as a goroutine, so unpredictable when they're finished)
		err = databaseClient.UpdateComputedTables(ctx, build.RepoSource, build.RepoOwner, build.RepoName)
		assert.Nil(t, err, "failed upserting computed pipeline")
		err = databaseClient.UpdateComputedTables(ctx, otherBuild.RepoSource, otherBuild.RepoOwner, otherBuild.RepoName)
		assert.Nil(t, err, "failed upserting computed pipeline for other build")

		// act
		releaseTargets, err := databaseClient.GetAllPipelinesReleaseTargets(ctx, 1, 10, filters)

		assert.Nil(t, err, "failed getting release targets")
		if !assert.Equal(t, 1, len(releaseTargets)) {
			assert.Equal(t, "", releaseTargets)
		}
		assert.Equal(t, "GetAllPipelinesReleaseTargets", releaseTargets[0]["name"])
		assert.Equal(t, int64(2), releaseTargets[0]["count"])
	})
}

func TestIntegrationGetAllPipelinesReleaseTargetsCount(t *testing.T) {
	t.Run("ReturnsReleaseTargetsCountForMatchingTargets", func(t *testing.T) {

		if testing.Short() {
			t.Skip("skipping test in short mode.")
		}

		ctx := context.Background()
		databaseClient := getDatabaseClient(ctx, t)
		jobResources := getJobResources()
		build := getBuild()
		build.RepoName = "release-targets-count-test-1"
		build.Labels = []contracts.Label{{Key: "release-targets-count-test", Value: "GetAllPipelinesReleaseTargetsCount"}}
		build.ReleaseTargets = []contracts.ReleaseTarget{{
			Name: "GetAllPipelinesReleaseTargetsCount",
		}}
		_, err := databaseClient.InsertBuild(ctx, build, jobResources)
		assert.Nil(t, err, "failed inserting build record")

		otherBuild := getBuild()
		otherBuild.RepoName = "release-targets-count-test-2"
		otherBuild.Labels = []contracts.Label{{Key: "release-targets-count-test", Value: "GetAllPipelinesReleaseTargetsCount"}}
		otherBuild.ReleaseTargets = []contracts.ReleaseTarget{{
			Name: "GetAllPipelinesReleaseTargetsCount",
		}}
		_, err = databaseClient.InsertBuild(ctx, otherBuild, jobResources)
		assert.Nil(t, err, "failed inserting other build record")

		filters := map[api.FilterType][]string{
			api.FilterReleaseTarget: {
				"GetAllPipelinesReleaseTargetsCount",
			},
			api.FilterLabels: {
				"release-targets-count-test=GetAllPipelinesReleaseTargetsCount",
			},
			api.FilterSince: {
				"1d",
			},
		}

		// ensure computed_pipelines are updated in time (they run as a goroutine, so unpredictable when they're finished)
		err = databaseClient.UpdateComputedTables(ctx, build.RepoSource, build.RepoOwner, build.RepoName)
		assert.Nil(t, err, "failed upserting computed pipeline")
		err = databaseClient.UpdateComputedTables(ctx, otherBuild.RepoSource, otherBuild.RepoOwner, otherBuild.RepoName)
		assert.Nil(t, err, "failed upserting computed pipeline for other build")

		// act
		count, err := databaseClient.GetAllPipelinesReleaseTargetsCount(ctx, filters)

		assert.Nil(t, err, "failed getting release targets count")
		assert.Equal(t, 1, count)
	})
}

func TestIntegrationGetAllReleasesReleaseTargets(t *testing.T) {
	t.Run("ReturnsReleaseTargetsForMatchingTargets", func(t *testing.T) {

		if testing.Short() {
			t.Skip("skipping test in short mode.")
		}

		ctx := context.Background()
		databaseClient := getDatabaseClient(ctx, t)

		release := getRelease()
		release.Name = "GetAllReleasesReleaseTargets"
		jobResources := getJobResources()
		_, err := databaseClient.InsertRelease(ctx, release, jobResources)
		assert.Nil(t, err)

		otherRelease := getRelease()
		otherRelease.Name = "GetAllReleasesReleaseTargets"
		_, err = databaseClient.InsertRelease(ctx, otherRelease, jobResources)
		assert.Nil(t, err)

		filters := map[api.FilterType][]string{
			api.FilterSince: {
				"1d",
			},
		}

		// act
		releaseTargets, err := databaseClient.GetAllReleasesReleaseTargets(ctx, 1, 10, filters)

		assert.Nil(t, err, "failed getting release targets")
		assert.GreaterOrEqual(t, len(releaseTargets), 1)
	})
}

func TestIntegrationGetAllReleasesReleaseTargetsCount(t *testing.T) {
	t.Run("ReturnsReleaseTargetsCountForMatchingTargets", func(t *testing.T) {

		if testing.Short() {
			t.Skip("skipping test in short mode.")
		}

		ctx := context.Background()
		databaseClient := getDatabaseClient(ctx, t)

		release := getRelease()
		release.Name = "GetAllReleasesReleaseTargetsCount"
		jobResources := getJobResources()
		_, err := databaseClient.InsertRelease(ctx, release, jobResources)
		assert.Nil(t, err)

		otherRelease := getRelease()
		otherRelease.Name = "GetAllReleasesReleaseTargetsCount"
		_, err = databaseClient.InsertRelease(ctx, otherRelease, jobResources)
		assert.Nil(t, err)

		filters := map[api.FilterType][]string{
			api.FilterSince: {
				"1d",
			},
		}

		// act
		count, err := databaseClient.GetAllReleasesReleaseTargetsCount(ctx, filters)

		assert.Nil(t, err, "failed getting release targets count")
		assert.GreaterOrEqual(t, count, 1)
	})
}

func TestIntegrationGetPipelineBuildBranches(t *testing.T) {
	t.Run("ReturnsBuildBranchesForMatchingPipeline", func(t *testing.T) {

		if testing.Short() {
			t.Skip("skipping test in short mode.")
		}

		ctx := context.Background()
		databaseClient := getDatabaseClient(ctx, t)
		jobResources := getJobResources()
		build := getBuild()
		build.RepoBranch = "GetPipelineBuildBranches"
		build.RepoName = "build-branches-test-1"
		build.Labels = []contracts.Label{{Key: "build-branches-test", Value: "GetPipelineBuildBranches"}}
		_, err := databaseClient.InsertBuild(ctx, build, jobResources)
		assert.Nil(t, err, "failed inserting first build record")

		otherBuild := getBuild()
		otherBuild.RepoBranch = "GetPipelineBuildBranches"
		otherBuild.RepoName = "build-branches-test-1"
		otherBuild.Labels = []contracts.Label{{Key: "build-branches-test", Value: "GetPipelineBuildBranches"}}
		_, err = databaseClient.InsertBuild(ctx, otherBuild, jobResources)
		assert.Nil(t, err, "failed inserting other build record")

		filters := map[api.FilterType][]string{
			api.FilterLabels: {
				"build-branches-test=GetPipelineBuildBranches",
			},
			api.FilterSince: {
				"1d",
			},
		}

		// act
		buildBranches, err := databaseClient.GetPipelineBuildBranches(ctx, build.RepoSource, build.RepoOwner, build.RepoName, 1, 10, filters)

		assert.Nil(t, err, "failed getting build branches")
		if !assert.Equal(t, 1, len(buildBranches)) {
			assert.Equal(t, "", buildBranches)
		}
		assert.Equal(t, "GetPipelineBuildBranches", buildBranches[0]["name"])
		assert.Equal(t, int64(2), buildBranches[0]["count"])
	})
}

func TestIntegrationGetPipelineBuildBranchesCount(t *testing.T) {
	t.Run("ReturnsBuildBranchesCountForMatchingPipeline", func(t *testing.T) {

		if testing.Short() {
			t.Skip("skipping test in short mode.")
		}

		ctx := context.Background()
		databaseClient := getDatabaseClient(ctx, t)
		jobResources := getJobResources()
		build := getBuild()
		build.RepoBranch = "GetPipelineBuildBranchesCount"
		build.RepoName = "build-branches-count-test-1"
		build.Labels = []contracts.Label{{Key: "build-branches-count-test", Value: "GetPipelineBuildBranchesCount"}}
		_, err := databaseClient.InsertBuild(ctx, build, jobResources)
		assert.Nil(t, err, "failed inserting build record")

		otherBuild := getBuild()
		otherBuild.RepoBranch = "GetPipelineBuildBranchesCount"
		otherBuild.RepoName = "build-branches-count-test-1"
		otherBuild.Labels = []contracts.Label{{Key: "build-branches-count-test", Value: "GetPipelineBuildBranchesCount"}}
		_, err = databaseClient.InsertBuild(ctx, otherBuild, jobResources)
		assert.Nil(t, err, "failed inserting other build record")

		filters := map[api.FilterType][]string{
			api.FilterLabels: {
				"build-branches-count-test=GetPipelineBuildBranchesCount",
			},
			api.FilterSince: {
				"1d",
			},
		}

		// act
		count, err := databaseClient.GetPipelineBuildBranchesCount(ctx, build.RepoSource, build.RepoOwner, build.RepoName, filters)

		assert.Nil(t, err, "failed getting build branches count")
		assert.Equal(t, 1, count)
	})
}

func TestIntegrationGetPipelineBotNames(t *testing.T) {
	t.Run("ReturnsBotNamesForMatchingPipeline", func(t *testing.T) {

		if testing.Short() {
			t.Skip("skipping test in short mode.")
		}

		ctx := context.Background()
		databaseClient := getDatabaseClient(ctx, t)
		jobResources := getJobResources()
		bot := getBot()
		bot.Name = "GetPipelineBotNames"
		bot.RepoName = "bot-names-test-1"
		_, err := databaseClient.InsertBot(ctx, bot, jobResources)
		assert.Nil(t, err, "failed inserting first bot record")

		otherBot := getBot()
		otherBot.Name = "GetPipelineBotNames"
		otherBot.RepoName = "bot-names-test-1"
		_, err = databaseClient.InsertBot(ctx, otherBot, jobResources)
		assert.Nil(t, err, "failed inserting other bot record")

		filters := map[api.FilterType][]string{
			api.FilterSince: {
				"1d",
			},
		}

		// act
		buildBranches, err := databaseClient.GetPipelineBotNames(ctx, bot.RepoSource, bot.RepoOwner, bot.RepoName, 1, 10, filters)

		assert.Nil(t, err, "failed getting bot names")
		if !assert.Equal(t, 1, len(buildBranches)) {
			assert.Equal(t, "", buildBranches)
		}
		assert.Equal(t, "GetPipelineBotNames", buildBranches[0]["name"])
		assert.Equal(t, int64(2), buildBranches[0]["count"])
	})
}

func TestIntegrationGetPipelineBotNamesCount(t *testing.T) {
	t.Run("ReturnsBotNamesCountForMatchingPipeline", func(t *testing.T) {

		if testing.Short() {
			t.Skip("skipping test in short mode.")
		}

		ctx := context.Background()
		databaseClient := getDatabaseClient(ctx, t)
		jobResources := getJobResources()
		bot := getBot()
		bot.Name = "GetPipelineBotNamesCount"
		bot.RepoName = "bot-names-count-test-1"
		_, err := databaseClient.InsertBot(ctx, bot, jobResources)
		assert.Nil(t, err, "failed inserting bot record")

		otherBot := getBot()
		otherBot.Name = "GetPipelineBotNamesCount"
		otherBot.RepoName = "bot-names-count-test-1"
		_, err = databaseClient.InsertBot(ctx, otherBot, jobResources)
		assert.Nil(t, err, "failed inserting other bot record")

		filters := map[api.FilterType][]string{
			api.FilterSince: {
				"1d",
			},
		}

		// act
		count, err := databaseClient.GetPipelineBotNamesCount(ctx, bot.RepoSource, bot.RepoOwner, bot.RepoName, filters)

		assert.Nil(t, err, "failed getting bot names count")
		assert.Equal(t, 1, count)
	})
}

func TestIntegrationInsertUser(t *testing.T) {
	t.Run("ReturnsInsertedUserWithID", func(t *testing.T) {

		if testing.Short() {
			t.Skip("skipping test in short mode.")
		}

		ctx := context.Background()
		databaseClient := getDatabaseClient(ctx, t)
		user := getUser()

		// act
		insertedUser, err := databaseClient.InsertUser(ctx, user)

		assert.Nil(t, err)
		assert.NotNil(t, insertedUser)
		assert.True(t, insertedUser.ID != "")
	})
}

func TestIntegrationUpdateUser(t *testing.T) {
	t.Run("UpdatesUser", func(t *testing.T) {

		if testing.Short() {
			t.Skip("skipping test in short mode.")
		}

		ctx := context.Background()
		databaseClient := getDatabaseClient(ctx, t)
		user := getUser()
		insertedUser, err := databaseClient.InsertUser(ctx, user)
		assert.Nil(t, err)

		// act
		err = databaseClient.UpdateUser(ctx, *insertedUser)

		assert.Nil(t, err)
	})

	t.Run("UpdatesUserForNonExistingUser", func(t *testing.T) {

		if testing.Short() {
			t.Skip("skipping test in short mode.")
		}

		ctx := context.Background()
		databaseClient := getDatabaseClient(ctx, t)
		user := getUser()
		insertedUser, err := databaseClient.InsertUser(ctx, user)
		assert.Nil(t, err)
		insertedUser.ID = "15"

		// act
		err = databaseClient.UpdateUser(ctx, *insertedUser)

		assert.Nil(t, err)
	})
}

func TestIntegrationDeleteUser(t *testing.T) {
	t.Run("DeletesUser", func(t *testing.T) {

		if testing.Short() {
			t.Skip("skipping test in short mode.")
		}

		ctx := context.Background()
		databaseClient := getDatabaseClient(ctx, t)
		user := getUser()
		insertedUser, err := databaseClient.InsertUser(ctx, user)
		assert.Nil(t, err)

		// act
		err = databaseClient.DeleteUser(ctx, *insertedUser)

		assert.Nil(t, err)
	})

	t.Run("DeletesUserForNonExistingUser", func(t *testing.T) {

		if testing.Short() {
			t.Skip("skipping test in short mode.")
		}

		ctx := context.Background()
		databaseClient := getDatabaseClient(ctx, t)
		user := getUser()
		insertedUser, err := databaseClient.InsertUser(ctx, user)
		assert.Nil(t, err)
		insertedUser.ID = "15"

		// act
		err = databaseClient.DeleteUser(ctx, *insertedUser)

		assert.Nil(t, err)
	})
}
func TestIntegrationGetUserByIdentity(t *testing.T) {
	t.Run("ReturnsInsertedUserWithID", func(t *testing.T) {

		if testing.Short() {
			t.Skip("skipping test in short mode.")
		}

		ctx := context.Background()
		databaseClient := getDatabaseClient(ctx, t)
		user := getUser()
		user.Identities = []*contracts.UserIdentity{
			{
				Provider: "google",
				ID:       "wilson",
				Email:    "wilson-test@homeimprovement.com",
			},
		}
		insertedUser, err := databaseClient.InsertUser(ctx, user)
		assert.Nil(t, err)

		identity := contracts.UserIdentity{
			Provider: "google",
			Email:    "wilson-test@homeimprovement.com",
		}

		// act
		retrievedUser, err := databaseClient.GetUserByIdentity(ctx, identity)

		assert.Nil(t, err)
		assert.NotNil(t, retrievedUser)
		assert.Equal(t, retrievedUser.ID, insertedUser.ID)
	})

	t.Run("DoesNotReturnUserIfProviderDoesNotMatch", func(t *testing.T) {

		if testing.Short() {
			t.Skip("skipping test in short mode.")
		}

		ctx := context.Background()
		databaseClient := getDatabaseClient(ctx, t)
		user := getUser()
		user.Identities = []*contracts.UserIdentity{
			{
				Provider: "google",
				ID:       "wilson",
				Email:    "wilson-test@homeimprovement.com",
			},
		}
		_, err := databaseClient.InsertUser(ctx, user)
		assert.Nil(t, err)

		identity := contracts.UserIdentity{
			Provider: "github",
			Email:    "wilson-test@homeimprovement.com",
		}

		// act
		retrievedUser, err := databaseClient.GetUserByIdentity(ctx, identity)

		assert.NotNil(t, err)
		assert.True(t, errors.Is(err, ErrUserNotFound))
		assert.Nil(t, retrievedUser)
	})
}

func TestIntegrationGetUserByID(t *testing.T) {
	t.Run("ReturnsInsertedUserWithID", func(t *testing.T) {

		if testing.Short() {
			t.Skip("skipping test in short mode.")
		}

		ctx := context.Background()
		databaseClient := getDatabaseClient(ctx, t)
		user := getUser()
		insertedUser, err := databaseClient.InsertUser(ctx, user)
		assert.Nil(t, err)

		// act
		retrievedUser, err := databaseClient.GetUserByID(ctx, insertedUser.ID, map[api.FilterType][]string{})

		assert.Nil(t, err)
		assert.NotNil(t, retrievedUser)
		assert.Equal(t, retrievedUser.ID, insertedUser.ID)
	})

	t.Run("ReturnsInsertedUserWithIDFilteredOnOrganization", func(t *testing.T) {

		if testing.Short() {
			t.Skip("skipping test in short mode.")
		}

		ctx := context.Background()
		databaseClient := getDatabaseClient(ctx, t)
		user := getUser()
		insertedUser, err := databaseClient.InsertUser(ctx, user)
		assert.Nil(t, err)
		filters := map[api.FilterType][]string{
			api.FilterOrganizations: []string{"Estafette"},
		}

		// act
		retrievedUser, err := databaseClient.GetUserByID(ctx, insertedUser.ID, filters)

		assert.Nil(t, err)
		assert.NotNil(t, retrievedUser)
		assert.Equal(t, retrievedUser.ID, insertedUser.ID)
	})
}

func TestIntegrationGetUsers(t *testing.T) {
	t.Run("ReturnsInsertedUsers", func(t *testing.T) {

		if testing.Short() {
			t.Skip("skipping test in short mode.")
		}

		ctx := context.Background()
		databaseClient := getDatabaseClient(ctx, t)
		user := getUser()
		_, err := databaseClient.InsertUser(ctx, user)
		assert.Nil(t, err)

		// act
		users, err := databaseClient.GetUsers(ctx, 1, 100, map[api.FilterType][]string{}, []api.OrderField{})

		assert.Nil(t, err)
		assert.NotNil(t, users)
		assert.True(t, len(users) > 0)
	})

	t.Run("ReturnsInsertedUsersFilteredByGroupID", func(t *testing.T) {

		if testing.Short() {
			t.Skip("skipping test in short mode.")
		}

		ctx := context.Background()
		databaseClient := getDatabaseClient(ctx, t)
		user := getUser()
		user.Groups = []*contracts.Group{
			{
				ID:   "36",
				Name: "Team A",
			},
		}
		_, err := databaseClient.InsertUser(ctx, user)
		assert.Nil(t, err)

		filters := map[api.FilterType][]string{
			api.FilterGroupID: {"36"},
		}

		// act
		users, err := databaseClient.GetUsers(ctx, 1, 100, filters, []api.OrderField{})

		assert.Nil(t, err)
		assert.NotNil(t, users)
		assert.True(t, len(users) > 0)
	})

	t.Run("ReturnsInsertedUsersFilteredByOrganizationID", func(t *testing.T) {

		if testing.Short() {
			t.Skip("skipping test in short mode.")
		}

		ctx := context.Background()
		databaseClient := getDatabaseClient(ctx, t)
		user := getUser()
		user.Organizations = []*contracts.Organization{
			{
				ID:   "638",
				Name: "Estafette",
			},
		}
		_, err := databaseClient.InsertUser(ctx, user)
		assert.Nil(t, err)

		filters := map[api.FilterType][]string{
			api.FilterOrganizationID: {"638"},
		}

		// act
		users, err := databaseClient.GetUsers(ctx, 1, 100, filters, []api.OrderField{})

		assert.Nil(t, err)
		assert.NotNil(t, users)
		assert.True(t, len(users) > 0)
	})
}

func TestIntegrationGetUsersCount(t *testing.T) {
	t.Run("ReturnsInsertedUsersCount", func(t *testing.T) {

		if testing.Short() {
			t.Skip("skipping test in short mode.")
		}

		ctx := context.Background()
		databaseClient := getDatabaseClient(ctx, t)
		user := getUser()
		_, err := databaseClient.InsertUser(ctx, user)
		assert.Nil(t, err)

		// act
		count, err := databaseClient.GetUsersCount(ctx, map[api.FilterType][]string{})

		assert.Nil(t, err)
		assert.True(t, count > 0)
	})
}

func TestIntegrationInsertGroup(t *testing.T) {
	t.Run("ReturnsInsertedGroupWithID", func(t *testing.T) {

		if testing.Short() {
			t.Skip("skipping test in short mode.")
		}

		ctx := context.Background()
		databaseClient := getDatabaseClient(ctx, t)
		group := getGroup()

		// act
		insertedGroup, err := databaseClient.InsertGroup(ctx, group)

		assert.Nil(t, err)
		assert.NotNil(t, insertedGroup)
		assert.True(t, insertedGroup.ID != "")
	})
}

func TestIntegrationUpdateGroup(t *testing.T) {
	t.Run("UpdatesGroup", func(t *testing.T) {

		if testing.Short() {
			t.Skip("skipping test in short mode.")
		}

		ctx := context.Background()
		databaseClient := getDatabaseClient(ctx, t)
		group := getGroup()
		insertedGroup, err := databaseClient.InsertGroup(ctx, group)
		assert.Nil(t, err)

		// act
		err = databaseClient.UpdateGroup(ctx, *insertedGroup)

		assert.Nil(t, err)
	})

	t.Run("UpdatesGroupForNonExistingGroup", func(t *testing.T) {

		if testing.Short() {
			t.Skip("skipping test in short mode.")
		}

		ctx := context.Background()
		databaseClient := getDatabaseClient(ctx, t)
		group := getGroup()
		insertedGroup, err := databaseClient.InsertGroup(ctx, group)
		assert.Nil(t, err)
		insertedGroup.ID = "15"

		// act
		err = databaseClient.UpdateGroup(ctx, *insertedGroup)

		assert.Nil(t, err)
	})
}

func TestIntegrationDeleteGroup(t *testing.T) {
	t.Run("DeletesGroup", func(t *testing.T) {

		if testing.Short() {
			t.Skip("skipping test in short mode.")
		}

		ctx := context.Background()
		databaseClient := getDatabaseClient(ctx, t)
		group := getGroup()
		insertedGroup, err := databaseClient.InsertGroup(ctx, group)
		assert.Nil(t, err)

		// act
		err = databaseClient.DeleteGroup(ctx, *insertedGroup)

		assert.Nil(t, err)
	})

	t.Run("DeletesGroupForNonExistingGroup", func(t *testing.T) {

		if testing.Short() {
			t.Skip("skipping test in short mode.")
		}

		ctx := context.Background()
		databaseClient := getDatabaseClient(ctx, t)
		group := getGroup()
		insertedGroup, err := databaseClient.InsertGroup(ctx, group)
		assert.Nil(t, err)
		insertedGroup.ID = "15"

		// act
		err = databaseClient.DeleteGroup(ctx, *insertedGroup)

		assert.Nil(t, err)
	})
}

func TestIntegrationGetGroupByIdentity(t *testing.T) {
	t.Run("ReturnsInsertedGroupWithID", func(t *testing.T) {

		if testing.Short() {
			t.Skip("skipping test in short mode.")
		}

		ctx := context.Background()
		databaseClient := getDatabaseClient(ctx, t)
		group := getGroup()
		group.Identities = []*contracts.GroupIdentity{
			{
				Provider: "google",
				ID:       "team-z",
				Name:     "Team Z",
			},
		}
		insertedGroup, err := databaseClient.InsertGroup(ctx, group)
		assert.Nil(t, err)

		identity := contracts.GroupIdentity{
			Provider: "google",
			Name:     "Team Z",
		}

		// act
		retrievedGroup, err := databaseClient.GetGroupByIdentity(ctx, identity)

		assert.Nil(t, err)
		assert.NotNil(t, retrievedGroup)
		assert.Equal(t, retrievedGroup.ID, insertedGroup.ID)
	})

	t.Run("DoesNotReturnGroupIfProviderDoesNotMatch", func(t *testing.T) {

		if testing.Short() {
			t.Skip("skipping test in short mode.")
		}

		ctx := context.Background()
		databaseClient := getDatabaseClient(ctx, t)
		group := getGroup()
		group.Identities = []*contracts.GroupIdentity{
			{
				Provider: "google",
				ID:       "team-z",
				Name:     "Team Z",
			},
		}
		_, err := databaseClient.InsertGroup(ctx, group)
		assert.Nil(t, err)

		identity := contracts.GroupIdentity{
			Provider: "google",
			Name:     "Team Y",
		}

		// act
		retrievedGroup, err := databaseClient.GetGroupByIdentity(ctx, identity)

		assert.NotNil(t, err)
		assert.True(t, errors.Is(err, ErrGroupNotFound))
		assert.Nil(t, retrievedGroup)
	})
}

func TestIntegrationGetGroupByID(t *testing.T) {
	t.Run("ReturnsInsertedGroupWithID", func(t *testing.T) {

		if testing.Short() {
			t.Skip("skipping test in short mode.")
		}

		ctx := context.Background()
		databaseClient := getDatabaseClient(ctx, t)
		group := getGroup()
		insertedGroup, err := databaseClient.InsertGroup(ctx, group)
		assert.Nil(t, err)

		// act
		retrievedGroup, err := databaseClient.GetGroupByID(ctx, insertedGroup.ID, map[api.FilterType][]string{})

		assert.Nil(t, err)
		assert.NotNil(t, retrievedGroup)
		assert.Equal(t, retrievedGroup.ID, insertedGroup.ID)
	})

	t.Run("ReturnsInsertedGroupWithIDIfFilteredOnOrganization", func(t *testing.T) {

		if testing.Short() {
			t.Skip("skipping test in short mode.")
		}

		ctx := context.Background()
		databaseClient := getDatabaseClient(ctx, t)
		group := getGroup()
		insertedGroup, err := databaseClient.InsertGroup(ctx, group)
		assert.Nil(t, err)
		filters := map[api.FilterType][]string{
			api.FilterOrganizations: []string{"Org A"},
		}

		// act
		retrievedGroup, err := databaseClient.GetGroupByID(ctx, insertedGroup.ID, filters)

		assert.Nil(t, err)
		assert.NotNil(t, retrievedGroup)
		assert.Equal(t, retrievedGroup.ID, insertedGroup.ID)
	})
}

func TestIntegrationGetGroups(t *testing.T) {
	t.Run("ReturnsInsertedGroups", func(t *testing.T) {

		if testing.Short() {
			t.Skip("skipping test in short mode.")
		}

		ctx := context.Background()
		databaseClient := getDatabaseClient(ctx, t)
		group := getGroup()
		_, err := databaseClient.InsertGroup(ctx, group)
		assert.Nil(t, err)

		// act
		groups, err := databaseClient.GetGroups(ctx, 1, 100, map[api.FilterType][]string{}, []api.OrderField{})

		assert.Nil(t, err)
		assert.NotNil(t, groups)
		assert.True(t, len(groups) > 0)
	})
}

func TestIntegrationGetGroupsCount(t *testing.T) {
	t.Run("ReturnsInsertedGroupsCount", func(t *testing.T) {

		if testing.Short() {
			t.Skip("skipping test in short mode.")
		}

		ctx := context.Background()
		databaseClient := getDatabaseClient(ctx, t)
		group := getGroup()
		_, err := databaseClient.InsertGroup(ctx, group)
		assert.Nil(t, err)

		// act
		count, err := databaseClient.GetGroupsCount(ctx, map[api.FilterType][]string{})

		assert.Nil(t, err)
		assert.True(t, count > 0)
	})
}

func TestIntegrationInsertOrganization(t *testing.T) {
	t.Run("ReturnsInsertedOrganizationWithID", func(t *testing.T) {

		if testing.Short() {
			t.Skip("skipping test in short mode.")
		}

		ctx := context.Background()
		databaseClient := getDatabaseClient(ctx, t)
		organization := getOrganization()

		// act
		insertedOrganization, err := databaseClient.InsertOrganization(ctx, organization)

		assert.Nil(t, err)
		assert.NotNil(t, insertedOrganization)
		assert.True(t, insertedOrganization.ID != "")
	})
}

func TestIntegrationUpdateOrganization(t *testing.T) {
	t.Run("UpdatesOrganization", func(t *testing.T) {

		if testing.Short() {
			t.Skip("skipping test in short mode.")
		}

		ctx := context.Background()
		databaseClient := getDatabaseClient(ctx, t)
		organization := getOrganization()
		insertedOrganization, err := databaseClient.InsertOrganization(ctx, organization)
		assert.Nil(t, err)

		// act
		err = databaseClient.UpdateOrganization(ctx, *insertedOrganization)

		assert.Nil(t, err)
	})

	t.Run("UpdatesOrganizationForNonExistingOrganization", func(t *testing.T) {

		if testing.Short() {
			t.Skip("skipping test in short mode.")
		}

		ctx := context.Background()
		databaseClient := getDatabaseClient(ctx, t)
		organization := getOrganization()
		insertedOrganization, err := databaseClient.InsertOrganization(ctx, organization)
		assert.Nil(t, err)
		insertedOrganization.ID = "15"

		// act
		err = databaseClient.UpdateOrganization(ctx, *insertedOrganization)

		assert.Nil(t, err)
	})
}

func TestIntegrationDeleteOrganization(t *testing.T) {
	t.Run("DeletesOrganization", func(t *testing.T) {

		if testing.Short() {
			t.Skip("skipping test in short mode.")
		}

		ctx := context.Background()
		databaseClient := getDatabaseClient(ctx, t)
		organization := getOrganization()
		insertedOrganization, err := databaseClient.InsertOrganization(ctx, organization)
		assert.Nil(t, err)

		// act
		err = databaseClient.DeleteOrganization(ctx, *insertedOrganization)

		assert.Nil(t, err)
	})

	t.Run("DeletesOrganizationForNonExistingOrganization", func(t *testing.T) {

		if testing.Short() {
			t.Skip("skipping test in short mode.")
		}

		ctx := context.Background()
		databaseClient := getDatabaseClient(ctx, t)
		organization := getOrganization()
		insertedOrganization, err := databaseClient.InsertOrganization(ctx, organization)
		assert.Nil(t, err)
		insertedOrganization.ID = "15"

		// act
		err = databaseClient.DeleteOrganization(ctx, *insertedOrganization)

		assert.Nil(t, err)
	})
}

func TestIntegrationGetOrganizationByIdentity(t *testing.T) {
	t.Run("ReturnsInsertedOrganizationWithID", func(t *testing.T) {

		if testing.Short() {
			t.Skip("skipping test in short mode.")
		}

		ctx := context.Background()
		databaseClient := getDatabaseClient(ctx, t)
		organization := getOrganization()
		organization.Identities = []*contracts.OrganizationIdentity{
			{
				Provider: "google",
				ID:       "org-z",
				Name:     "Org Z",
			},
		}
		insertedOrganization, err := databaseClient.InsertOrganization(ctx, organization)
		assert.Nil(t, err)

		identity := contracts.OrganizationIdentity{
			Provider: "google",
			Name:     "Org Z",
		}

		// act
		retrievedOrganization, err := databaseClient.GetOrganizationByIdentity(ctx, identity)

		assert.Nil(t, err)
		assert.NotNil(t, retrievedOrganization)
		assert.Equal(t, retrievedOrganization.ID, insertedOrganization.ID)
	})

	t.Run("DoesNotReturnOrganizationIfProviderDoesNotMatch", func(t *testing.T) {

		if testing.Short() {
			t.Skip("skipping test in short mode.")
		}

		ctx := context.Background()
		databaseClient := getDatabaseClient(ctx, t)
		organization := getOrganization()
		organization.Identities = []*contracts.OrganizationIdentity{
			{
				Provider: "google",
				ID:       "org-z",
				Name:     "Org Z",
			},
		}
		_, err := databaseClient.InsertOrganization(ctx, organization)
		assert.Nil(t, err)

		identity := contracts.OrganizationIdentity{
			Provider: "google",
			Name:     "Org y",
		}

		// act
		retrievedOrganization, err := databaseClient.GetOrganizationByIdentity(ctx, identity)

		assert.NotNil(t, err)
		assert.True(t, errors.Is(err, ErrOrganizationNotFound))
		assert.Nil(t, retrievedOrganization)
	})
}

func TestIntegrationGetOrganizationByID(t *testing.T) {
	t.Run("ReturnsInsertedOrganizationWithID", func(t *testing.T) {

		if testing.Short() {
			t.Skip("skipping test in short mode.")
		}

		ctx := context.Background()
		databaseClient := getDatabaseClient(ctx, t)
		organization := getOrganization()
		insertedOrganization, err := databaseClient.InsertOrganization(ctx, organization)
		assert.Nil(t, err)

		// act
		retrievedOrganization, err := databaseClient.GetOrganizationByID(ctx, insertedOrganization.ID)

		assert.Nil(t, err)
		assert.NotNil(t, retrievedOrganization)
		assert.Equal(t, retrievedOrganization.ID, insertedOrganization.ID)
	})
}

func TestIntegrationGetOrganizationByName(t *testing.T) {
	t.Run("ReturnsInsertedOrganizationWithName", func(t *testing.T) {

		if testing.Short() {
			t.Skip("skipping test in short mode.")
		}

		ctx := context.Background()
		databaseClient := getDatabaseClient(ctx, t)
		organization := getOrganization()
		organization.Name = "organization-name-test"
		insertedOrganization, err := databaseClient.InsertOrganization(ctx, organization)
		assert.Nil(t, err)

		// act
		retrievedOrganization, err := databaseClient.GetOrganizationByName(ctx, "organization-name-test")

		assert.Nil(t, err)
		assert.NotNil(t, retrievedOrganization)
		assert.Equal(t, retrievedOrganization.ID, insertedOrganization.ID)
		assert.Equal(t, retrievedOrganization.Name, insertedOrganization.Name)
	})
}

func TestIntegrationGetOrganizations(t *testing.T) {
	t.Run("ReturnsInsertedOrganizations", func(t *testing.T) {

		if testing.Short() {
			t.Skip("skipping test in short mode.")
		}

		ctx := context.Background()
		databaseClient := getDatabaseClient(ctx, t)
		organization := getOrganization()
		_, err := databaseClient.InsertOrganization(ctx, organization)
		assert.Nil(t, err)

		// act
		organizations, err := databaseClient.GetOrganizations(ctx, 1, 100, map[api.FilterType][]string{}, []api.OrderField{})

		assert.Nil(t, err)
		assert.NotNil(t, organizations)
		assert.True(t, len(organizations) > 0)
	})
}

func TestIntegrationGetOrganizationsCount(t *testing.T) {
	t.Run("ReturnsInsertedOrganizationsCount", func(t *testing.T) {

		if testing.Short() {
			t.Skip("skipping test in short mode.")
		}

		ctx := context.Background()
		databaseClient := getDatabaseClient(ctx, t)
		organization := getOrganization()
		_, err := databaseClient.InsertOrganization(ctx, organization)
		assert.Nil(t, err)

		// act
		count, err := databaseClient.GetOrganizationsCount(ctx, map[api.FilterType][]string{})

		assert.Nil(t, err)
		assert.True(t, count > 0)
	})
}

func TestIntegrationInsertClient(t *testing.T) {
	t.Run("ReturnsInsertedClientWithID", func(t *testing.T) {

		if testing.Short() {
			t.Skip("skipping test in short mode.")
		}

		ctx := context.Background()
		databaseClient := getDatabaseClient(ctx, t)
		client := getClient()

		// act
		insertedClient, err := databaseClient.InsertClient(ctx, client)

		assert.Nil(t, err)
		assert.NotNil(t, insertedClient)
		assert.True(t, insertedClient.ID != "")
	})
}

func TestIntegrationUpdateClient(t *testing.T) {
	t.Run("UpdatesClient", func(t *testing.T) {

		if testing.Short() {
			t.Skip("skipping test in short mode.")
		}

		ctx := context.Background()
		databaseClient := getDatabaseClient(ctx, t)
		client := getClient()
		insertedClient, err := databaseClient.InsertClient(ctx, client)
		assert.Nil(t, err)

		// act
		err = databaseClient.UpdateClient(ctx, *insertedClient)

		assert.Nil(t, err)
	})

	t.Run("UpdatesClientForNonExistingClient", func(t *testing.T) {

		if testing.Short() {
			t.Skip("skipping test in short mode.")
		}

		ctx := context.Background()
		databaseClient := getDatabaseClient(ctx, t)
		client := getClient()
		insertedClient, err := databaseClient.InsertClient(ctx, client)
		assert.Nil(t, err)
		insertedClient.ID = "15"

		// act
		err = databaseClient.UpdateClient(ctx, *insertedClient)

		assert.Nil(t, err)
	})
}

func TestIntegrationDeleteClient(t *testing.T) {
	t.Run("DeletesClient", func(t *testing.T) {

		if testing.Short() {
			t.Skip("skipping test in short mode.")
		}

		ctx := context.Background()
		databaseClient := getDatabaseClient(ctx, t)
		client := getClient()
		insertedClient, err := databaseClient.InsertClient(ctx, client)
		assert.Nil(t, err)

		// act
		err = databaseClient.DeleteClient(ctx, *insertedClient)

		assert.Nil(t, err)
	})

	t.Run("DeletesClientForNonExistingClient", func(t *testing.T) {

		if testing.Short() {
			t.Skip("skipping test in short mode.")
		}

		ctx := context.Background()
		databaseClient := getDatabaseClient(ctx, t)
		client := getClient()
		insertedClient, err := databaseClient.InsertClient(ctx, client)
		assert.Nil(t, err)
		insertedClient.ID = "15"

		// act
		err = databaseClient.DeleteClient(ctx, *insertedClient)

		assert.Nil(t, err)
	})
}

func TestIntegrationGetClientByClientID(t *testing.T) {
	t.Run("ReturnsInsertedClientWithID", func(t *testing.T) {

		if testing.Short() {
			t.Skip("skipping test in short mode.")
		}

		ctx := context.Background()
		databaseClient := getDatabaseClient(ctx, t)
		client := getClient()
		client.ClientID = "zyx"
		insertedClient, err := databaseClient.InsertClient(ctx, client)
		assert.Nil(t, err)

		clientID := "zyx"

		// act
		retrievedClient, err := databaseClient.GetClientByClientID(ctx, clientID)

		assert.Nil(t, err)
		assert.NotNil(t, retrievedClient)
		assert.Equal(t, retrievedClient.ID, insertedClient.ID)
	})

	t.Run("DoesNotReturnClientIfClientIDDoesNotMatch", func(t *testing.T) {

		if testing.Short() {
			t.Skip("skipping test in short mode.")
		}

		ctx := context.Background()
		databaseClient := getDatabaseClient(ctx, t)
		client := getClient()
		client.ClientID = "zyx"
		_, err := databaseClient.InsertClient(ctx, client)
		assert.Nil(t, err)

		clientID := "xyz"

		// act
		retrievedClient, err := databaseClient.GetClientByClientID(ctx, clientID)

		assert.NotNil(t, err)
		assert.True(t, errors.Is(err, ErrClientNotFound))
		assert.Nil(t, retrievedClient)
	})
}

func TestIntegrationGetClientByID(t *testing.T) {
	t.Run("ReturnsInsertedClientWithID", func(t *testing.T) {

		if testing.Short() {
			t.Skip("skipping test in short mode.")
		}

		ctx := context.Background()
		databaseClient := getDatabaseClient(ctx, t)
		client := getClient()
		insertedClient, err := databaseClient.InsertClient(ctx, client)
		assert.Nil(t, err)

		// act
		retrievedClient, err := databaseClient.GetClientByID(ctx, insertedClient.ID)

		assert.Nil(t, err)
		assert.NotNil(t, retrievedClient)
		assert.Equal(t, retrievedClient.ID, insertedClient.ID)
	})
}

func TestIntegrationGetClients(t *testing.T) {
	t.Run("ReturnsInsertedClients", func(t *testing.T) {

		if testing.Short() {
			t.Skip("skipping test in short mode.")
		}

		ctx := context.Background()
		databaseClient := getDatabaseClient(ctx, t)
		client := getClient()
		_, err := databaseClient.InsertClient(ctx, client)
		assert.Nil(t, err)

		// act
		clients, err := databaseClient.GetClients(ctx, 1, 100, map[api.FilterType][]string{}, []api.OrderField{})

		assert.Nil(t, err)
		assert.NotNil(t, clients)
		assert.True(t, len(clients) > 0)
	})
}

func TestIntegrationGetClientsCount(t *testing.T) {
	t.Run("ReturnsInsertedClientsCount", func(t *testing.T) {

		if testing.Short() {
			t.Skip("skipping test in short mode.")
		}

		ctx := context.Background()
		databaseClient := getDatabaseClient(ctx, t)
		client := getClient()
		_, err := databaseClient.InsertClient(ctx, client)
		assert.Nil(t, err)

		// act
		count, err := databaseClient.GetClientsCount(ctx, map[api.FilterType][]string{})

		assert.Nil(t, err)
		assert.True(t, count > 0)
	})
}

func TestIntegrationInsertCatalogEntity(t *testing.T) {
	t.Run("ReturnsInsertedCatalogEntityWithID", func(t *testing.T) {

		if testing.Short() {
			t.Skip("skipping test in short mode.")
		}

		ctx := context.Background()
		databaseClient := getDatabaseClient(ctx, t)
		catalogEntity := getCatalogEntity()

		// act
		insertedCatalogEntity, err := databaseClient.InsertCatalogEntity(ctx, catalogEntity)

		assert.Nil(t, err)
		assert.NotNil(t, insertedCatalogEntity)
		assert.True(t, insertedCatalogEntity.ID != "")
	})
}

func TestIntegrationUpdateCatalogEntity(t *testing.T) {
	t.Run("UpdatesCatalogEntity", func(t *testing.T) {

		if testing.Short() {
			t.Skip("skipping test in short mode.")
		}

		ctx := context.Background()
		databaseClient := getDatabaseClient(ctx, t)
		catalogEntity := getCatalogEntity()
		insertedCatalogEntity, err := databaseClient.InsertCatalogEntity(ctx, catalogEntity)
		assert.Nil(t, err)

		// act
		err = databaseClient.UpdateCatalogEntity(ctx, *insertedCatalogEntity)

		assert.Nil(t, err)
	})

	t.Run("UpdatesCatalogEntityForNonExistingCatalogEntity", func(t *testing.T) {

		if testing.Short() {
			t.Skip("skipping test in short mode.")
		}

		ctx := context.Background()
		databaseClient := getDatabaseClient(ctx, t)
		catalogEntity := getCatalogEntity()
		insertedCatalogEntity, err := databaseClient.InsertCatalogEntity(ctx, catalogEntity)
		assert.Nil(t, err)
		insertedCatalogEntity.ID = "15"

		// act
		err = databaseClient.UpdateCatalogEntity(ctx, *insertedCatalogEntity)

		assert.Nil(t, err)
	})
}

func TestIntegrationDeleteCatalogEntity(t *testing.T) {
	t.Run("DeletesInsertedCatalogEntityWithID", func(t *testing.T) {

		if testing.Short() {
			t.Skip("skipping test in short mode.")
		}

		ctx := context.Background()
		databaseClient := getDatabaseClient(ctx, t)
		catalogEntity := getCatalogEntity()
		insertedCatalogEntity, err := databaseClient.InsertCatalogEntity(ctx, catalogEntity)
		assert.Nil(t, err)

		// act
		err = databaseClient.DeleteCatalogEntity(ctx, insertedCatalogEntity.ID)

		assert.Nil(t, err)

		retrievedCatalogEntity, err := databaseClient.GetCatalogEntityByID(ctx, insertedCatalogEntity.ID)

		assert.NotNil(t, err)
		assert.True(t, errors.Is(err, ErrCatalogEntityNotFound))
		assert.Nil(t, retrievedCatalogEntity)
	})
}

func TestIntegrationGetCatalogEntityByID(t *testing.T) {
	t.Run("ReturnsInsertedCatalogEntityByID", func(t *testing.T) {

		if testing.Short() {
			t.Skip("skipping test in short mode.")
		}

		ctx := context.Background()
		databaseClient := getDatabaseClient(ctx, t)
		catalogEntity := getCatalogEntity()
		insertedCatalogEntity, err := databaseClient.InsertCatalogEntity(ctx, catalogEntity)
		assert.Nil(t, err)

		// act
		retrievedCatalogEntity, err := databaseClient.GetCatalogEntityByID(ctx, insertedCatalogEntity.ID)

		assert.Nil(t, err)
		assert.NotNil(t, retrievedCatalogEntity)
		assert.Equal(t, retrievedCatalogEntity.ID, insertedCatalogEntity.ID)
	})

	t.Run("ReturnsCatalogEntityNotFoundErrorWhenItDoesNotExist", func(t *testing.T) {

		if testing.Short() {
			t.Skip("skipping test in short mode.")
		}

		ctx := context.Background()
		databaseClient := getDatabaseClient(ctx, t)
		catalogEntity := getCatalogEntity()
		_, err := databaseClient.InsertCatalogEntity(ctx, catalogEntity)
		assert.Nil(t, err)

		// act
		retrievedCatalogEntity, err := databaseClient.GetCatalogEntityByID(ctx, "14")

		assert.NotNil(t, err)
		assert.True(t, errors.Is(err, ErrCatalogEntityNotFound))
		assert.Nil(t, retrievedCatalogEntity)
	})
}

func TestIntegrationGetCatalogEntities(t *testing.T) {
	t.Run("ReturnsInsertedCatalogEntities", func(t *testing.T) {

		if testing.Short() {
			t.Skip("skipping test in short mode.")
		}

		ctx := context.Background()
		databaseClient := getDatabaseClient(ctx, t)
		catalogEntity := getCatalogEntity()
		_, err := databaseClient.InsertCatalogEntity(ctx, catalogEntity)
		assert.Nil(t, err)

		// act
		catalogEntitys, err := databaseClient.GetCatalogEntities(ctx, 1, 100, map[api.FilterType][]string{}, []api.OrderField{})

		assert.Nil(t, err)
		assert.NotNil(t, catalogEntitys)
		assert.True(t, len(catalogEntitys) > 0)
	})

	t.Run("ReturnsInsertedCatalogEntitiesByParentKey", func(t *testing.T) {

		if testing.Short() {
			t.Skip("skipping test in short mode.")
		}

		ctx := context.Background()
		databaseClient := getDatabaseClient(ctx, t)
		catalogEntity := getCatalogEntity()
		catalogEntity.ParentKey = "parent-key-retrieval-test"
		_, err := databaseClient.InsertCatalogEntity(ctx, catalogEntity)
		assert.Nil(t, err)

		filters := map[api.FilterType][]string{
			api.FilterParent: {
				"parent-key-retrieval-test",
			},
		}

		// act
		catalogEntitys, err := databaseClient.GetCatalogEntities(ctx, 1, 100, filters, []api.OrderField{})

		assert.Nil(t, err)
		assert.NotNil(t, catalogEntitys)
		assert.True(t, len(catalogEntitys) > 0)
	})

	t.Run("ReturnsInsertedCatalogEntitiesByParentKeyAndValue", func(t *testing.T) {

		if testing.Short() {
			t.Skip("skipping test in short mode.")
		}

		ctx := context.Background()
		databaseClient := getDatabaseClient(ctx, t)
		catalogEntity := getCatalogEntity()
		catalogEntity.ParentKey = "parent-key-value-retrieval-test"
		catalogEntity.ParentValue = "some-value"
		_, err := databaseClient.InsertCatalogEntity(ctx, catalogEntity)
		assert.Nil(t, err)

		filters := map[api.FilterType][]string{
			api.FilterParent: {
				"parent-key-value-retrieval-test=some-value",
			},
		}

		// act
		catalogEntities, err := databaseClient.GetCatalogEntities(ctx, 1, 100, filters, []api.OrderField{})

		assert.Nil(t, err)
		assert.NotNil(t, catalogEntities)
		assert.True(t, len(catalogEntities) > 0)
	})

	t.Run("ReturnsInsertedCatalogEntitiesByEntityKey", func(t *testing.T) {

		if testing.Short() {
			t.Skip("skipping test in short mode.")
		}

		ctx := context.Background()
		databaseClient := getDatabaseClient(ctx, t)
		catalogEntity := getCatalogEntity()
		catalogEntity.Key = "entity-key-retrieval-test"
		_, err := databaseClient.InsertCatalogEntity(ctx, catalogEntity)
		assert.Nil(t, err)

		filters := map[api.FilterType][]string{
			api.FilterEntity: {
				"entity-key-retrieval-test",
			},
		}

		// act
		catalogEntitys, err := databaseClient.GetCatalogEntities(ctx, 1, 100, filters, []api.OrderField{})

		assert.Nil(t, err)
		assert.NotNil(t, catalogEntitys)
		assert.True(t, len(catalogEntitys) > 0)
	})

	t.Run("ReturnsInsertedCatalogEntitiesByEntityKeyAndValue", func(t *testing.T) {

		if testing.Short() {
			t.Skip("skipping test in short mode.")
		}

		ctx := context.Background()
		databaseClient := getDatabaseClient(ctx, t)
		catalogEntity := getCatalogEntity()
		catalogEntity.Key = "entity-key-value-retrieval-test"
		catalogEntity.Value = "some-value"
		_, err := databaseClient.InsertCatalogEntity(ctx, catalogEntity)
		assert.Nil(t, err)

		filters := map[api.FilterType][]string{
			api.FilterEntity: {
				"entity-key-value-retrieval-test=some-value",
			},
		}

		// act
		catalogEntitys, err := databaseClient.GetCatalogEntities(ctx, 1, 100, filters, []api.OrderField{})

		assert.Nil(t, err)
		assert.NotNil(t, catalogEntitys)
		assert.True(t, len(catalogEntitys) > 0)
	})

	t.Run("ReturnsInsertedCatalogEntitiesByLinkedPipeline", func(t *testing.T) {

		if testing.Short() {
			t.Skip("skipping test in short mode.")
		}

		ctx := context.Background()
		databaseClient := getDatabaseClient(ctx, t)
		catalogEntity := getCatalogEntity()
		catalogEntity.LinkedPipeline = "github.com/estafette/estafette-ci-api"
		_, err := databaseClient.InsertCatalogEntity(ctx, catalogEntity)
		assert.Nil(t, err)

		filters := map[api.FilterType][]string{
			api.FilterPipeline: {
				"github.com/estafette/estafette-ci-api",
			},
		}

		// act
		catalogEntitys, err := databaseClient.GetCatalogEntities(ctx, 1, 100, filters, []api.OrderField{})

		assert.Nil(t, err)
		assert.NotNil(t, catalogEntitys)
		assert.True(t, len(catalogEntitys) > 0)
	})

	t.Run("ReturnsInsertedCatalogEntitiesByLabels", func(t *testing.T) {

		if testing.Short() {
			t.Skip("skipping test in short mode.")
		}

		ctx := context.Background()
		databaseClient := getDatabaseClient(ctx, t)
		catalogEntity := getCatalogEntity()
		catalogEntity.Labels = []contracts.Label{
			{
				Key:   "environment",
				Value: "production",
			},
		}
		_, err := databaseClient.InsertCatalogEntity(ctx, catalogEntity)
		assert.Nil(t, err)

		filters := map[api.FilterType][]string{
			api.FilterLabels: {
				"environment=production",
			},
		}

		// act
		catalogEntitys, err := databaseClient.GetCatalogEntities(ctx, 1, 100, filters, []api.OrderField{})

		assert.Nil(t, err)
		assert.NotNil(t, catalogEntitys)
		assert.True(t, len(catalogEntitys) > 0)
	})
}

func TestIntegrationGetCatalogEntitiesCount(t *testing.T) {
	t.Run("ReturnsInsertedCatalogEntitiesCount", func(t *testing.T) {

		if testing.Short() {
			t.Skip("skipping test in short mode.")
		}

		ctx := context.Background()
		databaseClient := getDatabaseClient(ctx, t)
		catalogEntity := getCatalogEntity()
		_, err := databaseClient.InsertCatalogEntity(ctx, catalogEntity)
		assert.Nil(t, err)

		// act
		count, err := databaseClient.GetCatalogEntitiesCount(ctx, map[api.FilterType][]string{})

		assert.Nil(t, err)
		assert.True(t, count > 0)
	})
}

func TestIntegrationGetCatalogEntityParentKeys(t *testing.T) {
	t.Run("ReturnsGetCatalogEntityParentKeys", func(t *testing.T) {

		if testing.Short() {
			t.Skip("skipping test in short mode.")
		}

		ctx := context.Background()
		databaseClient := getDatabaseClient(ctx, t)
		catalogEntity := getCatalogEntity()
		_, err := databaseClient.InsertCatalogEntity(ctx, catalogEntity)
		assert.Nil(t, err)

		// act
		keys, err := databaseClient.GetCatalogEntityParentKeys(ctx, 1, 100, map[api.FilterType][]string{}, []api.OrderField{})

		assert.Nil(t, err)
		assert.True(t, len(keys) > 0)
	})
}

func TestIntegrationGetCatalogEntityParentKeysCount(t *testing.T) {
	t.Run("ReturnsGetCatalogEntityParentKeysCount", func(t *testing.T) {

		if testing.Short() {
			t.Skip("skipping test in short mode.")
		}

		ctx := context.Background()
		databaseClient := getDatabaseClient(ctx, t)
		catalogEntity := getCatalogEntity()
		_, err := databaseClient.InsertCatalogEntity(ctx, catalogEntity)
		assert.Nil(t, err)

		// act
		count, err := databaseClient.GetCatalogEntityParentKeysCount(ctx, map[api.FilterType][]string{})

		assert.Nil(t, err)
		assert.True(t, count > 0)
	})
}

func TestIntegrationGetCatalogEntityParentValues(t *testing.T) {
	t.Run("ReturnsGetCatalogEntityParentValues", func(t *testing.T) {

		if testing.Short() {
			t.Skip("skipping test in short mode.")
		}

		ctx := context.Background()
		databaseClient := getDatabaseClient(ctx, t)
		catalogEntity := getCatalogEntity()
		_, err := databaseClient.InsertCatalogEntity(ctx, catalogEntity)
		assert.Nil(t, err)

		// act
		values, err := databaseClient.GetCatalogEntityParentValues(ctx, 1, 100, map[api.FilterType][]string{}, []api.OrderField{})

		assert.Nil(t, err)
		assert.True(t, len(values) > 0)
	})
}

func TestIntegrationGetCatalogEntityParentValuesCount(t *testing.T) {
	t.Run("ReturnsGetCatalogEntityParentValuesCount", func(t *testing.T) {

		if testing.Short() {
			t.Skip("skipping test in short mode.")
		}

		ctx := context.Background()
		databaseClient := getDatabaseClient(ctx, t)
		catalogEntity := getCatalogEntity()
		_, err := databaseClient.InsertCatalogEntity(ctx, catalogEntity)
		assert.Nil(t, err)

		// act
		count, err := databaseClient.GetCatalogEntityParentValuesCount(ctx, map[api.FilterType][]string{})

		assert.Nil(t, err)
		assert.True(t, count > 0)
	})
}

func TestIntegrationGetCatalogEntityKeys(t *testing.T) {
	t.Run("ReturnsGetCatalogEntityKeys", func(t *testing.T) {

		if testing.Short() {
			t.Skip("skipping test in short mode.")
		}

		ctx := context.Background()
		databaseClient := getDatabaseClient(ctx, t)
		catalogEntity := getCatalogEntity()
		_, err := databaseClient.InsertCatalogEntity(ctx, catalogEntity)
		assert.Nil(t, err)

		// act
		keys, err := databaseClient.GetCatalogEntityKeys(ctx, 1, 100, map[api.FilterType][]string{}, []api.OrderField{})

		assert.Nil(t, err)
		assert.True(t, len(keys) > 0)
	})
}

func TestIntegrationGetCatalogEntityKeysCount(t *testing.T) {
	t.Run("ReturnsGetCatalogEntityKeysCount", func(t *testing.T) {

		if testing.Short() {
			t.Skip("skipping test in short mode.")
		}

		ctx := context.Background()
		databaseClient := getDatabaseClient(ctx, t)
		catalogEntity := getCatalogEntity()
		_, err := databaseClient.InsertCatalogEntity(ctx, catalogEntity)
		assert.Nil(t, err)

		// act
		count, err := databaseClient.GetCatalogEntityKeysCount(ctx, map[api.FilterType][]string{})

		assert.Nil(t, err)
		assert.True(t, count > 0)
	})
}

func TestIntegrationGetCatalogEntityValues(t *testing.T) {
	t.Run("ReturnsGetCatalogEntityValues", func(t *testing.T) {

		if testing.Short() {
			t.Skip("skipping test in short mode.")
		}

		ctx := context.Background()
		databaseClient := getDatabaseClient(ctx, t)
		catalogEntity := getCatalogEntity()
		_, err := databaseClient.InsertCatalogEntity(ctx, catalogEntity)
		assert.Nil(t, err)

		// act
		values, err := databaseClient.GetCatalogEntityValues(ctx, 1, 100, map[api.FilterType][]string{}, []api.OrderField{})

		assert.Nil(t, err)
		assert.True(t, len(values) > 0)
	})
}

func TestIntegrationGetCatalogEntityValuesCount(t *testing.T) {
	t.Run("ReturnsGetCatalogEntityValuesCount", func(t *testing.T) {

		if testing.Short() {
			t.Skip("skipping test in short mode.")
		}

		ctx := context.Background()
		databaseClient := getDatabaseClient(ctx, t)
		catalogEntity := getCatalogEntity()
		_, err := databaseClient.InsertCatalogEntity(ctx, catalogEntity)
		assert.Nil(t, err)

		// act
		count, err := databaseClient.GetCatalogEntityValuesCount(ctx, map[api.FilterType][]string{})

		assert.Nil(t, err)
		assert.True(t, count > 0)
	})
}

func TestIntegrationGetCatalogEntityLabels(t *testing.T) {
	t.Run("ReturnsGetCatalogEntityLabels", func(t *testing.T) {

		if testing.Short() {
			t.Skip("skipping test in short mode.")
		}

		ctx := context.Background()
		databaseClient := getDatabaseClient(ctx, t)
		catalogEntity := getCatalogEntity()
		catalogEntity.Value = "entity-1"
		_, err := databaseClient.InsertCatalogEntity(ctx, catalogEntity)
		assert.Nil(t, err)
		catalogEntity2 := getCatalogEntity()
		catalogEntity2.Value = "entity-2"
		_, err = databaseClient.InsertCatalogEntity(ctx, catalogEntity2)
		assert.Nil(t, err)

		// act
		keys, err := databaseClient.GetCatalogEntityLabels(ctx, 1, 100, map[api.FilterType][]string{})

		assert.Nil(t, err)
		assert.True(t, len(keys) > 0)
	})
}

func TestIntegrationGetCatalogEntityLabelsCount(t *testing.T) {
	t.Run("ReturnsGetCatalogEntityLabelsCount", func(t *testing.T) {

		if testing.Short() {
			t.Skip("skipping test in short mode.")
		}

		ctx := context.Background()
		databaseClient := getDatabaseClient(ctx, t)
		catalogEntity := getCatalogEntity()
		catalogEntity.Value = "entity-1"
		_, err := databaseClient.InsertCatalogEntity(ctx, catalogEntity)
		assert.Nil(t, err)
		catalogEntity2 := getCatalogEntity()
		catalogEntity2.Value = "entity-2"
		_, err = databaseClient.InsertCatalogEntity(ctx, catalogEntity2)
		assert.Nil(t, err)

		// act
		count, err := databaseClient.GetCatalogEntityLabelsCount(ctx, map[api.FilterType][]string{})

		assert.Nil(t, err)
		assert.True(t, count > 0)
	})
}

func TestIntegrationGetPipelineBuildsDurations(t *testing.T) {
	t.Run("ReturnsDurations", func(t *testing.T) {
		if testing.Short() {
			t.Skip("skipping test in short mode.")
		}

		ctx := context.Background()
		databaseClient := getDatabaseClient(ctx, t)
		build := getBuild()
		jobResources := getJobResources()
		_, err := databaseClient.InsertBuild(ctx, build, jobResources)
		assert.Nil(t, err)
		err = databaseClient.UpdateComputedTables(ctx, build.RepoSource, build.RepoOwner, build.RepoName)
		assert.Nil(t, err)

		// act
		durations, err := databaseClient.GetPipelineBuildsDurations(ctx, build.RepoSource, build.RepoOwner, build.RepoName, map[api.FilterType][]string{})

		assert.Nil(t, err)
		assert.True(t, len(durations) > 0)
	})
}

func TestIntegrationGetPipelineReleasesDurations(t *testing.T) {
	t.Run("ReturnsDurations", func(t *testing.T) {
		if testing.Short() {
			t.Skip("skipping test in short mode.")
		}

		ctx := context.Background()
		databaseClient := getDatabaseClient(ctx, t)
		release := getRelease()
		jobResources := getJobResources()
		_, err := databaseClient.InsertRelease(ctx, release, jobResources)
		assert.Nil(t, err)

		// act
		durations, err := databaseClient.GetPipelineReleasesDurations(ctx, release.RepoSource, release.RepoOwner, release.RepoName, map[api.FilterType][]string{})

		assert.Nil(t, err)
		assert.True(t, len(durations) > 0)
	})
}

func TestIntegrationGetPipelineBotsDurations(t *testing.T) {
	t.Run("ReturnsDurations", func(t *testing.T) {
		if testing.Short() {
			t.Skip("skipping test in short mode.")
		}

		ctx := context.Background()
		databaseClient := getDatabaseClient(ctx, t)
		bot := getBot()
		jobResources := getJobResources()
		_, err := databaseClient.InsertBot(ctx, bot, jobResources)
		assert.Nil(t, err)

		// act
		durations, err := databaseClient.GetPipelineBotsDurations(ctx, bot.RepoSource, bot.RepoOwner, bot.RepoName, map[api.FilterType][]string{})

		assert.Nil(t, err)
		assert.True(t, len(durations) > 0)
	})
}

func TestIntegrationGetPipelineBuilds(t *testing.T) {
	t.Run("ReturnsBuilds", func(t *testing.T) {
		if testing.Short() {
			t.Skip("skipping test in short mode.")
		}

		ctx := context.Background()
		databaseClient := getDatabaseClient(ctx, t)
		build := getBuild()
		jobResources := getJobResources()
		_, err := databaseClient.InsertBuild(ctx, build, jobResources)
		assert.Nil(t, err)
		err = databaseClient.UpdateComputedTables(ctx, build.RepoSource, build.RepoOwner, build.RepoName)
		assert.Nil(t, err)

		// act
		builds, err := databaseClient.GetPipelineBuilds(ctx, build.RepoSource, build.RepoOwner, build.RepoName, 1, 10, map[api.FilterType][]string{}, []api.OrderField{}, false)

		assert.Nil(t, err)
		assert.True(t, len(builds) > 0)
	})
}

func TestIntegrationGetPipelineBuildsCount(t *testing.T) {
	t.Run("ReturnsBuilds", func(t *testing.T) {
		if testing.Short() {
			t.Skip("skipping test in short mode.")
		}

		ctx := context.Background()
		databaseClient := getDatabaseClient(ctx, t)
		build := getBuild()
		jobResources := getJobResources()
		_, err := databaseClient.InsertBuild(ctx, build, jobResources)
		assert.Nil(t, err)
		err = databaseClient.UpdateComputedTables(ctx, build.RepoSource, build.RepoOwner, build.RepoName)
		assert.Nil(t, err)

		// act
		buildCount, err := databaseClient.GetPipelineBuildsCount(ctx, build.RepoSource, build.RepoOwner, build.RepoName, map[api.FilterType][]string{})

		assert.Nil(t, err)
		assert.True(t, buildCount > 0)
	})
}

func TestIntegrationGetPipelineReleases(t *testing.T) {
	t.Run("ReturnsReleases", func(t *testing.T) {
		if testing.Short() {
			t.Skip("skipping test in short mode.")
		}

		ctx := context.Background()
		databaseClient := getDatabaseClient(ctx, t)
		release := getRelease()
		jobResources := getJobResources()
		_, err := databaseClient.InsertRelease(ctx, release, jobResources)
		assert.Nil(t, err)

		// act
		releases, err := databaseClient.GetPipelineReleases(ctx, release.RepoSource, release.RepoOwner, release.RepoName, 1, 10, map[api.FilterType][]string{}, []api.OrderField{})

		assert.Nil(t, err)
		assert.True(t, len(releases) > 0)
	})
}

func TestIntegrationGetPipelineReleasesCount(t *testing.T) {
	t.Run("ReturnsReleases", func(t *testing.T) {
		if testing.Short() {
			t.Skip("skipping test in short mode.")
		}

		ctx := context.Background()
		databaseClient := getDatabaseClient(ctx, t)
		release := getRelease()
		jobResources := getJobResources()
		_, err := databaseClient.InsertRelease(ctx, release, jobResources)
		assert.Nil(t, err)

		// act
		releasesCount, err := databaseClient.GetPipelineReleasesCount(ctx, release.RepoSource, release.RepoOwner, release.RepoName, map[api.FilterType][]string{})

		assert.Nil(t, err)
		assert.True(t, releasesCount > 0)
	})
}

func TestIntegrationGetPipelineBots(t *testing.T) {
	t.Run("ReturnsBots", func(t *testing.T) {
		if testing.Short() {
			t.Skip("skipping test in short mode.")
		}

		ctx := context.Background()
		databaseClient := getDatabaseClient(ctx, t)
		bot := getBot()
		jobResources := getJobResources()
		_, err := databaseClient.InsertBot(ctx, bot, jobResources)
		assert.Nil(t, err)

		// act
		bots, err := databaseClient.GetPipelineBots(ctx, bot.RepoSource, bot.RepoOwner, bot.RepoName, 1, 10, map[api.FilterType][]string{}, []api.OrderField{})

		assert.Nil(t, err)
		assert.True(t, len(bots) > 0)
	})
}

func TestIntegrationGetPipelineBotsCount(t *testing.T) {
	t.Run("ReturnsBotsCount", func(t *testing.T) {
		if testing.Short() {
			t.Skip("skipping test in short mode.")
		}

		ctx := context.Background()
		databaseClient := getDatabaseClient(ctx, t)
		bot := getBot()
		jobResources := getJobResources()
		_, err := databaseClient.InsertBot(ctx, bot, jobResources)
		assert.Nil(t, err)

		// act
		botsCount, err := databaseClient.GetPipelineBotsCount(ctx, bot.RepoSource, bot.RepoOwner, bot.RepoName, map[api.FilterType][]string{})

		assert.Nil(t, err)
		assert.True(t, botsCount > 0)
	})
}

func TestIntegrationGetGitTriggers(t *testing.T) {
	t.Run("ReturnsPipelineIfTriggerEventAndRepositoryMatch", func(t *testing.T) {
		if testing.Short() {
			t.Skip("skipping test in short mode.")
		}

		ctx := context.Background()
		databaseClient := getDatabaseClient(ctx, t)
		jobResources := getJobResources()
		build := getBuild()
		build.RepoName = "github-trigger-test-1"
		build.Labels = []contracts.Label{{Key: "github-trigger-test", Value: "ReturnsPipelineIfOneTriggerEventMatches"}}

		build.Triggers = []manifest.EstafetteTrigger{
			{
				Git: &manifest.EstafetteGitTrigger{
					Event:      "push",
					Repository: "github.com/estafette/estafette-ci-api",
					Branch:     "main",
				},
			},
		}

		_, err := databaseClient.InsertBuild(ctx, build, jobResources)
		assert.Nil(t, err, "failed inserting first build record")

		// ensure computed_pipelines are updated in time (they run as a goroutine, so unpredictable when they're finished)
		err = databaseClient.UpdateComputedTables(ctx, build.RepoSource, build.RepoOwner, build.RepoName)
		assert.Nil(t, err, "failed upserting computed pipeline")

		gitEvent := manifest.EstafetteGitEvent{
			Event:      "push",
			Repository: "github.com/estafette/estafette-ci-api",
			Branch:     "main",
		}

		// act
		pipelines, err := databaseClient.GetGitTriggers(ctx, gitEvent)

		assert.Nil(t, err)
		assert.Equal(t, 1, len(pipelines))
	})

	t.Run("ReturnsNoPipelineIfTriggerEventDoesNotMatch", func(t *testing.T) {
		if testing.Short() {
			t.Skip("skipping test in short mode.")
		}

		ctx := context.Background()
		databaseClient := getDatabaseClient(ctx, t)
		jobResources := getJobResources()
		build := getBuild()
		build.RepoName = "git-trigger-test-2"
		build.Labels = []contracts.Label{{Key: "git-trigger-test", Value: "ReturnsNoPipelineIfNoTriggerEventMatches"}}

		build.Triggers = []manifest.EstafetteTrigger{
			{
				Git: &manifest.EstafetteGitTrigger{
					Event:      "push",
					Repository: "github.com/estafette/estafette-ci-api",
					Branch:     "main",
				},
			},
		}

		_, err := databaseClient.InsertBuild(ctx, build, jobResources)
		assert.Nil(t, err, "failed inserting first build record")

		// ensure computed_pipelines are updated in time (they run as a goroutine, so unpredictable when they're finished)
		err = databaseClient.UpdateComputedTables(ctx, build.RepoSource, build.RepoOwner, build.RepoName)
		assert.Nil(t, err, "failed upserting computed pipeline")

		gitEvent := manifest.EstafetteGitEvent{
			Event:      "release",
			Repository: "github.com/estafette/estafette-ci-api",
			Branch:     "main",
		}

		// act
		pipelines, err := databaseClient.GetGitTriggers(ctx, gitEvent)

		assert.Nil(t, err)
		assert.Equal(t, 0, len(pipelines))
	})
}

func TestIntegrationGetGithubTriggers(t *testing.T) {
	t.Run("ReturnsPipelineIfOneTriggerEventMatches", func(t *testing.T) {
		if testing.Short() {
			t.Skip("skipping test in short mode.")
		}

		ctx := context.Background()
		databaseClient := getDatabaseClient(ctx, t)
		jobResources := getJobResources()
		build := getBuild()
		build.RepoName = "github-trigger-test-1"
		build.Labels = []contracts.Label{{Key: "github-trigger-test", Value: "ReturnsPipelineIfOneTriggerEventMatches"}}

		build.Triggers = []manifest.EstafetteTrigger{
			{
				Github: &manifest.EstafetteGithubTrigger{
					Events: []string{
						"commit_comment",
						"create",
						"delete",
					},
					Repository: "github.com/estafette/estafette-ci-api",
				},
			},
		}

		_, err := databaseClient.InsertBuild(ctx, build, jobResources)
		assert.Nil(t, err, "failed inserting first build record")

		// ensure computed_pipelines are updated in time (they run as a goroutine, so unpredictable when they're finished)
		err = databaseClient.UpdateComputedTables(ctx, build.RepoSource, build.RepoOwner, build.RepoName)
		assert.Nil(t, err, "failed upserting computed pipeline")

		githubEvent := manifest.EstafetteGithubEvent{
			Event:      "commit_comment",
			Repository: "github.com/estafette/estafette-ci-api",
			Payload:    "{...}",
		}

		// act
		pipelines, err := databaseClient.GetGithubTriggers(ctx, githubEvent)

		assert.Nil(t, err)
		assert.Equal(t, 1, len(pipelines))
	})

	t.Run("ReturnsNoPipelineIfNoTriggerEventMatches", func(t *testing.T) {
		if testing.Short() {
			t.Skip("skipping test in short mode.")
		}

		ctx := context.Background()
		databaseClient := getDatabaseClient(ctx, t)
		jobResources := getJobResources()
		build := getBuild()
		build.RepoName = "github-trigger-test-2"
		build.Labels = []contracts.Label{{Key: "github-trigger-test", Value: "ReturnsNoPipelineIfNoTriggerEventMatches"}}

		build.Triggers = []manifest.EstafetteTrigger{
			{
				Github: &manifest.EstafetteGithubTrigger{
					Events: []string{
						"marketplace_purchase",
						"member",
						"membership",
					},
					Repository: "github.com/estafette/estafette-ci-api",
				},
			},
		}

		_, err := databaseClient.InsertBuild(ctx, build, jobResources)
		assert.Nil(t, err, "failed inserting first build record")

		// ensure computed_pipelines are updated in time (they run as a goroutine, so unpredictable when they're finished)
		err = databaseClient.UpdateComputedTables(ctx, build.RepoSource, build.RepoOwner, build.RepoName)
		assert.Nil(t, err, "failed upserting computed pipeline")

		githubEvent := manifest.EstafetteGithubEvent{
			Event:      "deployment_status",
			Repository: "github.com/estafette/estafette-ci-api",
			Payload:    "{...}",
		}

		// act
		pipelines, err := databaseClient.GetGithubTriggers(ctx, githubEvent)

		assert.Nil(t, err)
		assert.Equal(t, 0, len(pipelines))
	})
}

func TestIntegrationGetBitbucketTriggers(t *testing.T) {
	t.Run("ReturnsPipelineIfOneTriggerEventMatches", func(t *testing.T) {
		if testing.Short() {
			t.Skip("skipping test in short mode.")
		}

		ctx := context.Background()
		databaseClient := getDatabaseClient(ctx, t)
		jobResources := getJobResources()
		build := getBuild()
		build.RepoName = "github-trigger-test-1"
		build.Labels = []contracts.Label{{Key: "github-trigger-test", Value: "ReturnsPipelineIfOneTriggerEventMatches"}}

		build.Triggers = []manifest.EstafetteTrigger{
			{
				Bitbucket: &manifest.EstafetteBitbucketTrigger{
					Events: []string{
						"pullrequest:created",
						"pullrequest:updated",
						"pullrequest:approved",
					},
					Repository: "bitbucket.org/estafette/estafette-ci-api",
				},
			},
		}

		_, err := databaseClient.InsertBuild(ctx, build, jobResources)
		assert.Nil(t, err, "failed inserting first build record")

		// ensure computed_pipelines are updated in time (they run as a goroutine, so unpredictable when they're finished)
		err = databaseClient.UpdateComputedTables(ctx, build.RepoSource, build.RepoOwner, build.RepoName)
		assert.Nil(t, err, "failed upserting computed pipeline")

		bitbucketEvent := manifest.EstafetteBitbucketEvent{
			Event:      "pullrequest:updated",
			Repository: "bitbucket.org/estafette/estafette-ci-api",
			Payload:    "{...}",
		}

		// act
		pipelines, err := databaseClient.GetBitbucketTriggers(ctx, bitbucketEvent)

		assert.Nil(t, err)
		assert.Equal(t, 1, len(pipelines))
	})

	t.Run("ReturnsNoPipelineIfNoTriggerEventMatches", func(t *testing.T) {
		if testing.Short() {
			t.Skip("skipping test in short mode.")
		}

		ctx := context.Background()
		databaseClient := getDatabaseClient(ctx, t)
		jobResources := getJobResources()
		build := getBuild()
		build.RepoName = "bitbucket-trigger-test-2"
		build.Labels = []contracts.Label{{Key: "bitbucket-trigger-test", Value: "ReturnsNoPipelineIfNoTriggerEventMatches"}}
		build.Triggers = []manifest.EstafetteTrigger{
			{
				Bitbucket: &manifest.EstafetteBitbucketTrigger{
					Events: []string{
						"repo:commit_comment_created",
						"repo:commit_status_created",
						"repo:commit_status_updated",
					},
					Repository: "bitbucket.org/estafette/estafette-ci-api",
				},
			},
		}

		_, err := databaseClient.InsertBuild(ctx, build, jobResources)
		assert.Nil(t, err, "failed inserting first build record")

		// ensure computed_pipelines are updated in time (they run as a goroutine, so unpredictable when they're finished)
		err = databaseClient.UpdateComputedTables(ctx, build.RepoSource, build.RepoOwner, build.RepoName)
		assert.Nil(t, err, "failed upserting computed pipeline")

		bitbucketEvent := manifest.EstafetteBitbucketEvent{
			Event:      "issue:created",
			Repository: "bitbucket.org/estafette/estafette-ci-api",
			Payload:    "{...}",
		}

		// act
		pipelines, err := databaseClient.GetBitbucketTriggers(ctx, bitbucketEvent)

		assert.Nil(t, err)
		assert.Equal(t, 0, len(pipelines))
	})
}

func TestIntegrationInsertNotification(t *testing.T) {
	t.Run("ReturnsInsertedNotificationWithID", func(t *testing.T) {

		if testing.Short() {
			t.Skip("skipping test in short mode.")
		}

		ctx := context.Background()
		databaseClient := getDatabaseClient(ctx, t)
		notification := getNotification()

		// act
		insertedNotification, err := databaseClient.InsertNotification(ctx, notification)

		assert.Nil(t, err)
		assert.NotNil(t, insertedNotification)
		assert.True(t, insertedNotification.ID != "")
	})
}

func TestIntegrationGetAllNotifications(t *testing.T) {
	t.Run("ReturnsFirstPageOfNotifications", func(t *testing.T) {

		if testing.Short() {
			t.Skip("skipping test in short mode.")
		}

		ctx := context.Background()
		databaseClient := getDatabaseClient(ctx, t)
		notification := getNotification()
		_, err := databaseClient.InsertNotification(ctx, notification)
		assert.Nil(t, err)

		// act
		notifications, err := databaseClient.GetAllNotifications(ctx, 1, 20, map[api.FilterType][]string{}, []api.OrderField{})

		assert.Nil(t, err)
		assert.True(t, len(notifications) > 0)
	})
}

func TestIntegrationGetAllNotificationsCount(t *testing.T) {
	t.Run("ReturnsCountForAllNotifications", func(t *testing.T) {

		if testing.Short() {
			t.Skip("skipping test in short mode.")
		}

		ctx := context.Background()
		databaseClient := getDatabaseClient(ctx, t)
		notification := getNotification()
		_, err := databaseClient.InsertNotification(ctx, notification)
		assert.Nil(t, err)

		// act
		count, err := databaseClient.GetAllNotificationsCount(ctx, map[api.FilterType][]string{})

		assert.Nil(t, err)
		assert.True(t, count > 0)
	})
}

var dbTestClient Client
var dbTestClientMutex = &sync.Mutex{}

func getDatabaseClient(ctx context.Context, t *testing.T) Client {

	dbTestClientMutex.Lock()
	defer dbTestClientMutex.Unlock()

	if dbTestClient != nil {
		return dbTestClient
	}

	databaseName := "defaultdb"
	if os.Getenv("DB_DATABASE") != "" {
		databaseName = os.Getenv("DB_DATABASE")
	}
	host := "estafette-ci-db-public"
	if os.Getenv("DB_HOST") != "" {
		host = os.Getenv("DB_HOST")
	}
	insecure := true
	if os.Getenv("DB_INSECURE") != "" {
		dbInsecure, err := strconv.ParseBool(os.Getenv("DB_INSECURE"))
		if err == nil {
			insecure = dbInsecure
		}
	}
	port := 26257
	if os.Getenv("DB_PORT") != "" {
		dbPort, err := strconv.Atoi(os.Getenv("DB_PORT"))
		if err == nil {
			port = dbPort
		}
	}
	user := "root"
	if os.Getenv("DB_USER") != "" {
		user = os.Getenv("DB_USER")
	}
	password := ""
	if os.Getenv("DB_PASSWORD") != "" {
		password = os.Getenv("DB_PASSWORD")
	}

	maxOpenConnections := 0
	if os.Getenv("DB_MAX_OPEN_CONNECTIONS") != "" {
		dbMaxOpenConnections, err := strconv.Atoi(os.Getenv("DB_MAX_OPEN_CONNECTIONS"))
		if err == nil {
			maxOpenConnections = dbMaxOpenConnections
		}
	}

	maxIdleConnections := 2
	if os.Getenv("DB_MAX_IDLE_CONNECTIONS") != "" {
		dbMaxIdleConnections, err := strconv.Atoi(os.Getenv("DB_MAX_IDLE_CONNECTIONS"))
		if err == nil {
			maxIdleConnections = dbMaxIdleConnections
		}
	}

	apiConfig := &api.APIConfig{
		Database: &api.DatabaseConfig{
			DatabaseName: databaseName,
			Host:         host,
			Insecure:     insecure,
			Port:         port,
			User:         user,
			Password:     password,
			MaxOpenConns: maxOpenConnections,
			MaxIdleConns: maxIdleConnections,
		},
	}

	apiConfig.SetDefaults()

	dbTestClient = NewClient(apiConfig)
	err := dbTestClient.Connect(ctx)

	assert.Nil(t, err)

	err = dbTestClient.AwaitDatabaseReadiness(ctx)

	assert.Nil(t, err)

	return dbTestClient
}

func getBuild() contracts.Build {
	return contracts.Build{
		RepoSource:     "github.com",
		RepoOwner:      "estafette",
		RepoName:       "estafette-ci-api",
		RepoBranch:     "master",
		RepoRevision:   "08e9480b75154b5584995053344beb4d4aef65f4",
		BuildVersion:   "0.0.99",
		BuildStatus:    contracts.StatusPending,
		Labels:         []contracts.Label{{Key: "app-group", Value: "estafette-ci"}, {Key: "language", Value: "golang"}},
		ReleaseTargets: []contracts.ReleaseTarget{},
		Manifest:       "stages:\n  test:\n    image: golang:1.15.5-alpine3.12\n    commands:\n    - go test -short ./...",
		Commits: []contracts.GitCommit{
			{
				Message: "test commit",
				Author: contracts.GitAuthor{
					Email: "me@estafette.io",
				},
			},
		},
		Triggers: []manifest.EstafetteTrigger{},
		Events:   []manifest.EstafetteEvent{},
	}
}

func getRelease() contracts.Release {
	return contracts.Release{
		Name:           "production",
		Action:         "",
		RepoSource:     "github.com",
		RepoOwner:      "estafette",
		RepoName:       "estafette-ci-api",
		ReleaseVersion: "0.0.99",
		ReleaseStatus:  contracts.StatusPending,
		Events:         []manifest.EstafetteEvent{},
	}
}

func getBot() contracts.Bot {
	return contracts.Bot{
		RepoSource: "github.com",
		RepoOwner:  "estafette",
		RepoName:   "estafette-ci-api",
		BotStatus:  contracts.StatusPending,
		Events:     []manifest.EstafetteEvent{},
	}
}

func getNotification() contracts.NotificationRecord {
	return contracts.NotificationRecord{
		LinkType: contracts.NotificationLinkTypePipeline,
		LinkID:   "github.com/estafette/estafette-ci-api",
		PipelineDetail: &contracts.PipelineLinkDetail{
			Branch:   "main",
			Revision: "a4d05af37cfe169633793f68faea88746b240325",
			Version:  "1.0.0-main-1759",
			Status:   contracts.StatusSucceeded,
		},
		Source: "extensions/docker",
		Notifications: []contracts.Notification{
			{
				Type:    contracts.NotificationTypeVulnerability,
				Level:   contracts.NotificationLevelCritical,
				Message: "CVE-xxx",
			},
		},
	}
}

func getJobResources() JobResources {
	return JobResources{
		CPURequest:    float64(0.1),
		CPULimit:      float64(7.0),
		MemoryRequest: float64(67108864),
		MemoryLimit:   float64(21474836480),
	}
}

func getBuildLog() contracts.BuildLog {
	return contracts.BuildLog{
		RepoSource:   "github.com",
		RepoOwner:    "estafette",
		RepoName:     "estafette-ci-api",
		RepoBranch:   "master",
		RepoRevision: "08e9480b75154b5584995053344beb4d4aef65f4",
		BuildID:      "15",
		Steps: []*contracts.BuildLogStep{
			{
				Step: "stage-1",
				Image: &contracts.BuildLogStepDockerImage{
					Name: "golang",
					Tag:  "1.14.2-alpine3.11",
				},
				Duration: time.Duration(1234567),
				Status:   contracts.LogStatusSucceeded,
				LogLines: []contracts.BuildLogLine{
					{
						LineNumber: 1,
						Timestamp:  time.Now().UTC(),
						StreamType: "stdout",
						Text:       "ok",
					},
				},
			},
		},
	}
}

func getReleaseLog() contracts.ReleaseLog {
	return contracts.ReleaseLog{
		RepoSource: "github.com",
		RepoOwner:  "estafette",
		RepoName:   "estafette-ci-api",
		ReleaseID:  "15",
		Steps: []*contracts.BuildLogStep{
			{
				Step: "stage-1",
				Image: &contracts.BuildLogStepDockerImage{
					Name: "golang",
					Tag:  "1.14.2-alpine3.11",
				},
				Duration: time.Duration(1234567),
				Status:   contracts.LogStatusSucceeded,
				LogLines: []contracts.BuildLogLine{
					{
						LineNumber: 1,
						Timestamp:  time.Now().UTC(),
						StreamType: "stdout",
						Text:       "ok",
					},
				},
			},
		},
	}
}

func getBotLog() contracts.BotLog {
	return contracts.BotLog{
		RepoSource: "github.com",
		RepoOwner:  "estafette",
		RepoName:   "estafette-ci-api",
		BotID:      "15",
		Steps: []*contracts.BuildLogStep{
			{
				Step: "stage-1",
				Image: &contracts.BuildLogStepDockerImage{
					Name: "golang",
					Tag:  "1.14.2-alpine3.11",
				},
				Duration: time.Duration(1234567),
				Status:   contracts.LogStatusSucceeded,
				LogLines: []contracts.BuildLogLine{
					{
						LineNumber: 1,
						Timestamp:  time.Now().UTC(),
						StreamType: "stdout",
						Text:       "ok",
					},
				},
			},
		},
	}
}

func getUser() contracts.User {
	return contracts.User{
		Name: "Wilson Wilson",
		Identities: []*contracts.UserIdentity{
			{
				Provider: "google",
				ID:       "wilson",
				Email:    "wilson@homeimprovement.com",
			},
		},
		Groups: []*contracts.Group{
			{
				ID:   "33",
				Name: "Team A",
			},
		},
		Organizations: []*contracts.Organization{
			{
				ID:   "512",
				Name: "Estafette",
			},
		},
		Preferences: map[string]interface{}{
			"pipelines-page-size": 25,
			"builds-page-size":    20,
			"releases-page-size":  50,
			"user-filter":         "recent-committer",
		},
	}
}

func getGroup() contracts.Group {
	return contracts.Group{
		Name: "Team A",
		Identities: []*contracts.GroupIdentity{
			{
				Provider: "google",
				Name:     "Team A",
				ID:       "team-a",
			},
		},
		Organizations: []*contracts.Organization{
			{
				ID:   "12332443",
				Name: "Org A",
			},
		},
	}
}

func getOrganization() contracts.Organization {
	return contracts.Organization{
		Name: "Org A",
		Identities: []*contracts.OrganizationIdentity{
			{
				Provider: "google",
				Name:     "Org A",
				ID:       "org-a",
			},
		},
	}
}

func getClient() contracts.Client {
	now := time.Now().UTC()
	return contracts.Client{
		Name:         "estafette-cron-event-sender",
		ClientID:     "abc",
		ClientSecret: "my secret token",
		Roles:        []*string{},
		CreatedAt:    &now,
		Active:       true,
	}
}

func getCatalogEntity() contracts.CatalogEntity {
	now := time.Now().UTC()
	return contracts.CatalogEntity{
		ParentKey:      "organization",
		ParentValue:    "Estafette",
		Key:            "cloud",
		Value:          "Google Cloud",
		LinkedPipeline: "",
		Labels: []contracts.Label{
			{
				Key:   "organization",
				Value: "Estafette",
			},
		},
		Metadata: map[string]interface{}{
			"href": "/organizations/estafette/",
		},
		InsertedAt: &now,
		UpdatedAt:  &now,
	}
}
