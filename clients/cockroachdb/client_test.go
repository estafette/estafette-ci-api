package cockroachdb

import (
	"context"
	"regexp"
	"strconv"
	"testing"
	"time"

	"github.com/estafette/estafette-ci-api/config"
	contracts "github.com/estafette/estafette-ci-contracts"
	manifest "github.com/estafette/estafette-ci-manifest"
	"github.com/stretchr/testify/assert"
)

var (
	cdbClient = client{}
)

func TestIntegrationGetAutoIncrement(t *testing.T) {
	t.Run("ReturnsAnIncrementingCountForUniqueRepo", func(t *testing.T) {

		if testing.Short() {
			t.Skip("skipping test in short mode.")
		}

		ctx := context.Background()
		cockroachdbClient := getCockroachdbClient(ctx, t)

		// act
		autoincrement, err := cockroachdbClient.GetAutoIncrement(ctx, "github", "estafette", "estafette-ci-api")

		assert.Nil(t, err)
		assert.True(t, autoincrement > 0)
	})

	t.Run("ReturnsLargerCountForSubsequentRequests", func(t *testing.T) {

		if testing.Short() {
			t.Skip("skipping test in short mode.")
		}

		ctx := context.Background()
		cockroachdbClient := getCockroachdbClient(ctx, t)

		// act
		autoincrement1, err := cockroachdbClient.GetAutoIncrement(ctx, "github", "estafette", "estafette-ci-api")
		autoincrement2, err := cockroachdbClient.GetAutoIncrement(ctx, "github", "estafette", "estafette-ci-api")

		assert.Nil(t, err)
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
		cockroachdbClient := getCockroachdbClient(ctx, t)
		build := getBuild()
		jobResources := getJobResources()

		// act
		insertedBuild, err := cockroachdbClient.InsertBuild(ctx, build, jobResources)

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
		cockroachdbClient := getCockroachdbClient(ctx, t)
		build := getBuild()
		jobResources := getJobResources()
		insertedBuild, err := cockroachdbClient.InsertBuild(ctx, build, jobResources)
		assert.Nil(t, err)
		buildID, err := strconv.Atoi(insertedBuild.ID)
		assert.Nil(t, err)

		// act
		err = cockroachdbClient.UpdateBuildStatus(ctx, insertedBuild.RepoSource, insertedBuild.RepoOwner, insertedBuild.RepoName, buildID, "succeeded")

		assert.Nil(t, err)
	})

	t.Run("UpdatesStatusForNonExistingBuild", func(t *testing.T) {

		if testing.Short() {
			t.Skip("skipping test in short mode.")
		}

		ctx := context.Background()
		cockroachdbClient := getCockroachdbClient(ctx, t)
		build := getBuild()
		buildID := 15

		// act
		err := cockroachdbClient.UpdateBuildStatus(ctx, build.RepoSource, build.RepoOwner, build.RepoName, buildID, "succeeded")

		assert.Nil(t, err)
	})
}

func TestIntegrationUpdateBuildResourceUtilization(t *testing.T) {
	t.Run("UpdatesJobResourceForInsertedBuild", func(t *testing.T) {

		if testing.Short() {
			t.Skip("skipping test in short mode.")
		}

		ctx := context.Background()
		cockroachdbClient := getCockroachdbClient(ctx, t)
		build := getBuild()
		jobResources := getJobResources()
		insertedBuild, err := cockroachdbClient.InsertBuild(ctx, build, jobResources)
		assert.Nil(t, err)
		buildID, err := strconv.Atoi(insertedBuild.ID)
		assert.Nil(t, err)

		newJobResources := JobResources{
			CPURequest:    float64(0.3),
			CPULimit:      float64(4.0),
			MemoryRequest: float64(67108864),
			MemoryLimit:   float64(21474836480),
		}

		// act
		err = cockroachdbClient.UpdateBuildResourceUtilization(ctx, insertedBuild.RepoSource, insertedBuild.RepoOwner, insertedBuild.RepoName, buildID, newJobResources)

		assert.Nil(t, err)
	})

	t.Run("UpdatesJobResourcesForNonExistingBuild", func(t *testing.T) {

		if testing.Short() {
			t.Skip("skipping test in short mode.")
		}

		ctx := context.Background()
		cockroachdbClient := getCockroachdbClient(ctx, t)
		build := getBuild()
		buildID := 15

		newJobResources := JobResources{
			CPURequest:    float64(0.3),
			CPULimit:      float64(4.0),
			MemoryRequest: float64(67108864),
			MemoryLimit:   float64(21474836480),
		}

		// act
		err := cockroachdbClient.UpdateBuildResourceUtilization(ctx, build.RepoSource, build.RepoOwner, build.RepoName, buildID, newJobResources)

		assert.Nil(t, err)
	})
}

func TestIntegrationInsertRelease(t *testing.T) {
	t.Run("ReturnsInsertedReleaseWithID", func(t *testing.T) {

		if testing.Short() {
			t.Skip("skipping test in short mode.")
		}

		ctx := context.Background()
		cockroachdbClient := getCockroachdbClient(ctx, t)
		release := getRelease()
		jobResources := getJobResources()

		// act
		insertedRelease, err := cockroachdbClient.InsertRelease(ctx, release, jobResources)

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
		cockroachdbClient := getCockroachdbClient(ctx, t)
		release := getRelease()
		jobResources := getJobResources()
		insertedRelease, err := cockroachdbClient.InsertRelease(ctx, release, jobResources)
		assert.Nil(t, err)
		releaseID, err := strconv.Atoi(insertedRelease.ID)
		assert.Nil(t, err)

		// act
		err = cockroachdbClient.UpdateReleaseStatus(ctx, insertedRelease.RepoSource, insertedRelease.RepoOwner, insertedRelease.RepoName, releaseID, "succeeded")

		assert.Nil(t, err)
	})

	t.Run("UpdatesStatusForNonExistingRelease", func(t *testing.T) {

		if testing.Short() {
			t.Skip("skipping test in short mode.")
		}

		ctx := context.Background()
		cockroachdbClient := getCockroachdbClient(ctx, t)
		release := getRelease()
		releaseID := 15

		// act
		err := cockroachdbClient.UpdateReleaseStatus(ctx, release.RepoSource, release.RepoOwner, release.RepoName, releaseID, "succeeded")

		assert.Nil(t, err)
	})
}

func TestIntegrationUpdateReleaseResourceUtilization(t *testing.T) {
	t.Run("UpdatesJobResourceForInsertedBuild", func(t *testing.T) {

		if testing.Short() {
			t.Skip("skipping test in short mode.")
		}

		ctx := context.Background()
		cockroachdbClient := getCockroachdbClient(ctx, t)
		release := getRelease()
		jobResources := getJobResources()
		insertedRelease, err := cockroachdbClient.InsertRelease(ctx, release, jobResources)
		assert.Nil(t, err)
		releaseID, err := strconv.Atoi(insertedRelease.ID)
		assert.Nil(t, err)

		newJobResources := JobResources{
			CPURequest:    float64(0.3),
			CPULimit:      float64(4.0),
			MemoryRequest: float64(67108864),
			MemoryLimit:   float64(21474836480),
		}

		// act
		err = cockroachdbClient.UpdateReleaseResourceUtilization(ctx, release.RepoSource, release.RepoOwner, release.RepoName, releaseID, newJobResources)

		assert.Nil(t, err)
	})

	t.Run("UpdatesJobResourcesForNonExistingBuild", func(t *testing.T) {

		if testing.Short() {
			t.Skip("skipping test in short mode.")
		}

		ctx := context.Background()
		cockroachdbClient := getCockroachdbClient(ctx, t)
		release := getRelease()
		releaseID := 15

		newJobResources := JobResources{
			CPURequest:    float64(0.3),
			CPULimit:      float64(4.0),
			MemoryRequest: float64(67108864),
			MemoryLimit:   float64(21474836480),
		}

		// act
		err := cockroachdbClient.UpdateReleaseResourceUtilization(ctx, release.RepoSource, release.RepoOwner, release.RepoName, releaseID, newJobResources)

		assert.Nil(t, err)
	})
}

func TestIntegrationInsertBuildLog(t *testing.T) {
	t.Run("ReturnsInsertedBuildLogWithIDWhenWriteLogToDatabaseIsTrue", func(t *testing.T) {

		if testing.Short() {
			t.Skip("skipping test in short mode.")
		}

		ctx := context.Background()
		cockroachdbClient := getCockroachdbClient(ctx, t)
		buildLog := getBuildLog()

		// act
		insertedBuildLog, err := cockroachdbClient.InsertBuildLog(ctx, buildLog, true)

		assert.Nil(t, err)
		assert.NotNil(t, insertedBuildLog)
		assert.True(t, insertedBuildLog.ID != "")
	})

	t.Run("ReturnsInsertedBuildLogWithIDWhenWriteLogToDatabaseIsFalse", func(t *testing.T) {

		if testing.Short() {
			t.Skip("skipping test in short mode.")
		}

		ctx := context.Background()
		cockroachdbClient := getCockroachdbClient(ctx, t)
		buildLog := getBuildLog()

		// act
		insertedBuildLog, err := cockroachdbClient.InsertBuildLog(ctx, buildLog, false)

		assert.Nil(t, err)
		assert.NotNil(t, insertedBuildLog)
		assert.True(t, insertedBuildLog.ID != "")
	})
}

func TestIntegrationInsertReleaseLog(t *testing.T) {
	t.Run("ReturnsInsertedReleaseLogWithIDWhenWriteLogToDatabaseIsTrue", func(t *testing.T) {

		if testing.Short() {
			t.Skip("skipping test in short mode.")
		}

		ctx := context.Background()
		cockroachdbClient := getCockroachdbClient(ctx, t)
		releaseLog := getReleaseLog()

		// act
		insertedReleaseLog, err := cockroachdbClient.InsertReleaseLog(ctx, releaseLog, true)

		assert.Nil(t, err)
		assert.NotNil(t, insertedReleaseLog)
		assert.True(t, insertedReleaseLog.ID != "")
	})

	t.Run("ReturnsInsertedReleaseLogWithIDWhenWriteLogToDatabaseIsFalse", func(t *testing.T) {

		if testing.Short() {
			t.Skip("skipping test in short mode.")
		}

		ctx := context.Background()
		cockroachdbClient := getCockroachdbClient(ctx, t)
		releaseLog := getReleaseLog()

		// act
		insertedReleaseLog, err := cockroachdbClient.InsertReleaseLog(ctx, releaseLog, false)

		assert.Nil(t, err)
		assert.NotNil(t, insertedReleaseLog)
		assert.True(t, insertedReleaseLog.ID != "")
	})
}

func TestIngrationUpsertComputedPipeline(t *testing.T) {
	t.Run("ReturnsNoError", func(t *testing.T) {

		if testing.Short() {
			t.Skip("skipping test in short mode.")
		}

		ctx := context.Background()
		cockroachdbClient := getCockroachdbClient(ctx, t)
		build := getBuild()
		jobResources := getJobResources()
		_, err := cockroachdbClient.InsertBuild(ctx, build, jobResources)
		assert.Nil(t, err)

		// act
		err = cockroachdbClient.UpsertComputedPipeline(ctx, build.RepoSource, build.RepoOwner, build.RepoName)

		assert.Nil(t, err)
	})
}

func TestIngrationUpsertComputedRelease(t *testing.T) {
	t.Run("ReturnsNoError", func(t *testing.T) {

		if testing.Short() {
			t.Skip("skipping test in short mode.")
		}

		ctx := context.Background()
		cockroachdbClient := getCockroachdbClient(ctx, t)
		release := getRelease()
		jobResources := getJobResources()
		_, err := cockroachdbClient.InsertRelease(ctx, release, jobResources)
		assert.Nil(t, err)

		assert.Nil(t, err)

		// act
		err = cockroachdbClient.UpsertComputedRelease(ctx, release.RepoSource, release.RepoOwner, release.RepoName, release.Name, release.Action)

		assert.Nil(t, err)
	})
}

func TestIntegrationGetLabelValues(t *testing.T) {

	t.Run("ReturnsLabelValuesForMatchingLabelKey", func(t *testing.T) {

		if testing.Short() {
			t.Skip("skipping test in short mode.")
		}

		ctx := context.Background()
		cockroachdbClient := getCockroachdbClient(ctx, t)
		jobResources := getJobResources()
		build := getBuild()
		build.Labels = []contracts.Label{{Key: "type", Value: "api"}}
		_, err := cockroachdbClient.InsertBuild(ctx, build, jobResources)
		assert.Nil(t, err, "failed inserting first build record")

		otherBuild := getBuild()
		otherBuild.RepoName = "estafette-ci-web"
		build.Labels = []contracts.Label{{Key: "type", Value: "web"}}
		_, err = cockroachdbClient.InsertBuild(ctx, otherBuild, jobResources)
		assert.Nil(t, err, "failed inserting other build record")

		// act
		labels, err := cockroachdbClient.GetLabelValues(ctx, "type")

		assert.Nil(t, err, "failed getting label values")
		assert.Equal(t, 2, len(labels))
	})
}

func TestIntegrationGetFrequentLabels(t *testing.T) {
	t.Run("ReturnsFrequentLabelsForMatchingLabels", func(t *testing.T) {

		if testing.Short() {
			t.Skip("skipping test in short mode.")
		}

		ctx := context.Background()
		cockroachdbClient := getCockroachdbClient(ctx, t)
		jobResources := getJobResources()
		build := getBuild()
		_, err := cockroachdbClient.InsertBuild(ctx, build, jobResources)
		assert.Nil(t, err, "failed inserting first build record")

		otherBuild := getBuild()
		otherBuild.RepoName = "estafette-ci-db-migrator"
		_, err = cockroachdbClient.InsertBuild(ctx, otherBuild, jobResources)
		assert.Nil(t, err, "failed inserting other build record")

		filters := map[string][]string{
			"labels": {
				"app-group=estafette-ci",
			},
			"since": {
				"1d",
			},
		}

		// act
		labels, err := cockroachdbClient.GetFrequentLabels(ctx, 1, 10, filters)

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
		cockroachdbClient := getCockroachdbClient(ctx, t)
		jobResources := getJobResources()
		build := getBuild()
		_, err := cockroachdbClient.InsertBuild(ctx, build, jobResources)
		assert.Nil(t, err, "failed inserting build record")

		otherBuild := getBuild()
		otherBuild.RepoName = "estafette-ci-db-migrator"
		_, err = cockroachdbClient.InsertBuild(ctx, otherBuild, jobResources)
		assert.Nil(t, err, "failed inserting other build record")

		filters := map[string][]string{
			"labels": {
				"app-group=estafette-ci",
			},
			"since": {
				"1d",
			},
		}

		// act
		count, err := cockroachdbClient.GetFrequentLabelsCount(ctx, filters)

		assert.Nil(t, err, "failed getting frequent label count")
		assert.Equal(t, 1, count)
	})
}

func TestAutoincrement(t *testing.T) {

	t.Run("TestAutoincrementRegex", func(t *testing.T) {

		buildVersion := "0.0.126-MeD-1234123"
		re := regexp.MustCompile(`^[0-9]+\.[0-9]+\.([0-9]+)(-[0-9a-zA-Z-/]+)?$`)
		match := re.FindStringSubmatch(buildVersion)
		autoincrement := 0
		if len(match) > 1 {
			autoincrement, _ = strconv.Atoi(match[1])
		}

		assert.Equal(t, 126, autoincrement)
	})
}

func getCockroachdbClient(ctx context.Context, t *testing.T) Client {

	dbConfig := config.DatabaseConfig{
		DatabaseName:   "defaultdb",
		Host:           "cockroachdb",
		Insecure:       true,
		CertificateDir: "",
		Port:           26257,
		User:           "root",
		Password:       "",
	}

	cockroachdbClient := NewClient(dbConfig, *manifest.GetDefaultManifestPreferences())
	err := cockroachdbClient.Connect(ctx)

	assert.Nil(t, err)

	return cockroachdbClient
}

func getBuild() contracts.Build {
	return contracts.Build{
		RepoSource:     "github.com",
		RepoOwner:      "estafette",
		RepoName:       "estafette-ci-api",
		RepoBranch:     "master",
		RepoRevision:   "08e9480b75154b5584995053344beb4d4aef65f4",
		BuildVersion:   "0.0.99",
		BuildStatus:    "pending",
		Labels:         []contracts.Label{{Key: "app-group", Value: "estafette-ci"}, {Key: "language", Value: "golang"}, {Key: "type", Value: "api"}},
		ReleaseTargets: []contracts.ReleaseTarget{},
		Manifest:       "",
		Commits:        []contracts.GitCommit{},
		Triggers:       []manifest.EstafetteTrigger{},
		Events:         []manifest.EstafetteEvent{},
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
		ReleaseStatus:  "pending",
		Events:         []manifest.EstafetteEvent{},
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
				Status:   "SUCCEEDED",
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
				Status:   "SUCCEEDED",
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
