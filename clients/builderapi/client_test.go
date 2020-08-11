package builderapi

import (
	"testing"
	"context"

	"github.com/stretchr/testify/assert"
)

func TestGetJobName(t *testing.T) {

	t.Run("ReturnsJobNameForBuild", func(t *testing.T) {

		ciBuilderClient := &client{}

		// act
		jobName := ciBuilderClient.GetJobName(context.Background(), "build", "estafette", "estafette-ci-api", "390605593734184965")

		assert.Equal(t, "build-estafette-estafette-ci-api-390605593734184965", jobName)
		assert.Equal(t, 51, len(jobName))
	})

	t.Run("ReturnsJobNameForBuildWithMaxLengthOf63Characters", func(t *testing.T) {

		ciBuilderClient := &client{}

		// act
		jobName := ciBuilderClient.GetJobName(context.Background(), "build", "estafette", "estafette-extension-slack-build-status", "390605593734184965")

		assert.Equal(t, "build-estafette-estafette-extension-slack-bu-390605593734184965", jobName)
		assert.Equal(t, 63, len(jobName))
	})

	t.Run("ReturnsJobNameForRelease", func(t *testing.T) {

		ciBuilderClient := &client{}

		// act
		jobName := ciBuilderClient.GetJobName(context.Background(), "release", "estafette", "estafette-ci-api", "390605593734184965")

		assert.Equal(t, "release-estafette-estafette-ci-api-390605593734184965", jobName)
		assert.Equal(t, 53, len(jobName))
	})

	t.Run("ReturnsJobNameForReleaseWithMaxLengthOf63Characters", func(t *testing.T) {

		ciBuilderClient := &client{}

		// act
		jobName := ciBuilderClient.GetJobName(context.Background(), "release", "estafette", "estafette-extension-slack-build-status", "390605593734184965")

		assert.Equal(t, "release-estafette-estafette-extension-slack--390605593734184965", jobName)
		assert.Equal(t, 63, len(jobName))
	})
}
