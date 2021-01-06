package builderapi

import (
	"context"
	"testing"

	"github.com/estafette/estafette-ci-api/api"
	"github.com/stretchr/testify/assert"
	v1 "k8s.io/api/core/v1"
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

func TestGetCiBuilderJobAffinity(t *testing.T) {

	t.Run("AddsKubernetesOSNodeSelectorExpressionForWindowsBuild", func(t *testing.T) {

		apiConfig := &api.APIConfig{
			Jobs: &api.JobsConfig{
				BuildAffinityAndTolerations: &api.AffinityAndTolerationsConfig{
					Affinity: &v1.Affinity{
						NodeAffinity: &v1.NodeAffinity{
							RequiredDuringSchedulingIgnoredDuringExecution: &v1.NodeSelector{
								NodeSelectorTerms: []v1.NodeSelectorTerm{
									{
										MatchExpressions: []v1.NodeSelectorRequirement{
											{
												Key:      "role",
												Operator: v1.NodeSelectorOpIn,
												Values: []string{
													"privileged",
												},
											},
											{
												Key:      "cloud.google.com/gke-preemptible",
												Operator: v1.NodeSelectorOpDoesNotExist,
											},
										},
									},
								},
							},
						},
					},
				},
			},
		}

		ciBuilderClient := &client{
			config: apiConfig,
		}

		ciBuilderParams := CiBuilderParams{
			JobType:         "build",
			OperatingSystem: "windows",
		}

		// act
		affinity := ciBuilderClient.getCiBuilderJobAffinity(context.Background(), ciBuilderParams)

		assert.Equal(t, 3, len(affinity.NodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution.NodeSelectorTerms[0].MatchExpressions))
		assert.Equal(t, "kubernetes.io/os", affinity.NodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution.NodeSelectorTerms[0].MatchExpressions[2].Key)
		assert.Equal(t, "windows", affinity.NodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution.NodeSelectorTerms[0].MatchExpressions[2].Values[0])
	})

	t.Run("LeavesConfiguredAffinityUnmodifiedForWindowsBuild", func(t *testing.T) {

		apiConfig := &api.APIConfig{
			Jobs: &api.JobsConfig{
				BuildAffinityAndTolerations: &api.AffinityAndTolerationsConfig{
					Affinity: &v1.Affinity{
						NodeAffinity: &v1.NodeAffinity{
							RequiredDuringSchedulingIgnoredDuringExecution: &v1.NodeSelector{
								NodeSelectorTerms: []v1.NodeSelectorTerm{
									{
										MatchExpressions: []v1.NodeSelectorRequirement{
											{
												Key:      "role",
												Operator: v1.NodeSelectorOpIn,
												Values: []string{
													"privileged",
												},
											},
											{
												Key:      "cloud.google.com/gke-preemptible",
												Operator: v1.NodeSelectorOpDoesNotExist,
											},
										},
									},
								},
							},
						},
					},
				},
			},
		}

		ciBuilderClient := &client{
			config: apiConfig,
		}

		ciBuilderParams := CiBuilderParams{
			JobType:         "build",
			OperatingSystem: "windows",
		}

		// act
		ciBuilderClient.getCiBuilderJobAffinity(context.Background(), ciBuilderParams)

		assert.Equal(t, 2, len(apiConfig.Jobs.BuildAffinityAndTolerations.Affinity.NodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution.NodeSelectorTerms[0].MatchExpressions))
	})

	t.Run("AddsKubernetesOSNodeSelectorExpressionForWindowsRelease", func(t *testing.T) {

		apiConfig := &api.APIConfig{
			Jobs: &api.JobsConfig{
				ReleaseAffinityAndTolerations: &api.AffinityAndTolerationsConfig{
					Affinity: &v1.Affinity{
						NodeAffinity: &v1.NodeAffinity{
							RequiredDuringSchedulingIgnoredDuringExecution: &v1.NodeSelector{
								NodeSelectorTerms: []v1.NodeSelectorTerm{
									{
										MatchExpressions: []v1.NodeSelectorRequirement{
											{
												Key:      "role",
												Operator: v1.NodeSelectorOpIn,
												Values: []string{
													"privileged",
												},
											},
											{
												Key:      "cloud.google.com/gke-preemptible",
												Operator: v1.NodeSelectorOpDoesNotExist,
											},
										},
									},
								},
							},
						},
					},
				},
			},
		}

		ciBuilderClient := &client{
			config: apiConfig,
		}

		ciBuilderParams := CiBuilderParams{
			JobType:         "release",
			OperatingSystem: "windows",
		}

		// act
		affinity := ciBuilderClient.getCiBuilderJobAffinity(context.Background(), ciBuilderParams)

		assert.Equal(t, 3, len(affinity.NodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution.NodeSelectorTerms[0].MatchExpressions))
		assert.Equal(t, "kubernetes.io/os", affinity.NodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution.NodeSelectorTerms[0].MatchExpressions[2].Key)
		assert.Equal(t, "windows", affinity.NodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution.NodeSelectorTerms[0].MatchExpressions[2].Values[0])
	})

	t.Run("LeavesConfiguredAffinityUnmodifiedForWindowsRelease", func(t *testing.T) {

		apiConfig := &api.APIConfig{
			Jobs: &api.JobsConfig{
				ReleaseAffinityAndTolerations: &api.AffinityAndTolerationsConfig{
					Affinity: &v1.Affinity{
						NodeAffinity: &v1.NodeAffinity{
							RequiredDuringSchedulingIgnoredDuringExecution: &v1.NodeSelector{
								NodeSelectorTerms: []v1.NodeSelectorTerm{
									{
										MatchExpressions: []v1.NodeSelectorRequirement{
											{
												Key:      "role",
												Operator: v1.NodeSelectorOpIn,
												Values: []string{
													"privileged",
												},
											},
											{
												Key:      "cloud.google.com/gke-preemptible",
												Operator: v1.NodeSelectorOpDoesNotExist,
											},
										},
									},
								},
							},
						},
					},
				},
			},
		}

		ciBuilderClient := &client{
			config: apiConfig,
		}

		ciBuilderParams := CiBuilderParams{
			JobType:         "release",
			OperatingSystem: "windows",
		}

		// act
		ciBuilderClient.getCiBuilderJobAffinity(context.Background(), ciBuilderParams)

		assert.Equal(t, 2, len(apiConfig.Jobs.ReleaseAffinityAndTolerations.Affinity.NodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution.NodeSelectorTerms[0].MatchExpressions))
	})
}

func TestGetCiBuilderJobTolerations(t *testing.T) {

	t.Run("AddsKubernetesOSTolerationForWindowsBuild", func(t *testing.T) {

		apiConfig := &api.APIConfig{
			Jobs: &api.JobsConfig{
				BuildAffinityAndTolerations: &api.AffinityAndTolerationsConfig{
					Tolerations: []v1.Toleration{
						{
							Key:      "role",
							Operator: v1.TolerationOpEqual,
							Value:    "privileged",
							Effect:   v1.TaintEffectNoSchedule,
						},
					},
				},
			},
		}

		ciBuilderClient := &client{
			config: apiConfig,
		}

		ciBuilderParams := CiBuilderParams{
			JobType:         "build",
			OperatingSystem: "windows",
		}

		// act
		tolerations := ciBuilderClient.getCiBuilderJobTolerations(context.Background(), ciBuilderParams)

		assert.Equal(t, 2, len(tolerations))
		assert.Equal(t, "node.kubernetes.io/os", tolerations[1].Key)
		assert.Equal(t, v1.TaintEffectNoSchedule, tolerations[1].Effect)
		assert.Equal(t, "windows", tolerations[1].Value)
		assert.Equal(t, v1.TolerationOpEqual, tolerations[1].Operator)
	})

	t.Run("AddsKubernetesOSTolerationForWindowsRelease", func(t *testing.T) {

		apiConfig := &api.APIConfig{
			Jobs: &api.JobsConfig{
				ReleaseAffinityAndTolerations: &api.AffinityAndTolerationsConfig{
					Tolerations: []v1.Toleration{
						{
							Key:      "role",
							Operator: v1.TolerationOpEqual,
							Value:    "privileged",
							Effect:   v1.TaintEffectNoSchedule,
						},
					},
				},
			},
		}

		ciBuilderClient := &client{
			config: apiConfig,
		}

		ciBuilderParams := CiBuilderParams{
			JobType:         "release",
			OperatingSystem: "windows",
		}

		// act
		tolerations := ciBuilderClient.getCiBuilderJobTolerations(context.Background(), ciBuilderParams)

		assert.Equal(t, 2, len(tolerations))
		assert.Equal(t, "node.kubernetes.io/os", tolerations[1].Key)
		assert.Equal(t, v1.TaintEffectNoSchedule, tolerations[1].Effect)
		assert.Equal(t, "windows", tolerations[1].Value)
		assert.Equal(t, v1.TolerationOpEqual, tolerations[1].Operator)
	})
}
