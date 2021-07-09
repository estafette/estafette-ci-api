package builderapi

import (
	"context"
	"testing"

	"github.com/estafette/estafette-ci-api/pkg/api"
	contracts "github.com/estafette/estafette-ci-contracts"
	manifest "github.com/estafette/estafette-ci-manifest"
	"github.com/stretchr/testify/assert"
	v1 "k8s.io/api/core/v1"
)

func TestGetJobName(t *testing.T) {

	t.Run("ReturnsJobNameForBuild", func(t *testing.T) {

		ciBuilderClient := &client{}

		// act
		jobName := ciBuilderClient.GetJobName(context.Background(), contracts.JobTypeBuild, "estafette", "estafette-ci-api", "390605593734184965")

		assert.Equal(t, "build-estafette-estafette-ci-api-390605593734184965", jobName)
		assert.Equal(t, 51, len(jobName))
	})

	t.Run("ReturnsJobNameForBuildWithMaxLengthOf63Characters", func(t *testing.T) {

		ciBuilderClient := &client{}

		// act
		jobName := ciBuilderClient.GetJobName(context.Background(), contracts.JobTypeBuild, "estafette", "estafette-extension-slack-build-status", "390605593734184965")

		assert.Equal(t, "build-estafette-estafette-extension-slack-bu-390605593734184965", jobName)
		assert.Equal(t, 63, len(jobName))
	})

	t.Run("ReturnsJobNameForRelease", func(t *testing.T) {

		ciBuilderClient := &client{}

		// act
		jobName := ciBuilderClient.GetJobName(context.Background(), contracts.JobTypeRelease, "estafette", "estafette-ci-api", "390605593734184965")

		assert.Equal(t, "release-estafette-estafette-ci-api-390605593734184965", jobName)
		assert.Equal(t, 53, len(jobName))
	})

	t.Run("ReturnsJobNameForReleaseWithMaxLengthOf63Characters", func(t *testing.T) {

		ciBuilderClient := &client{}

		// act
		jobName := ciBuilderClient.GetJobName(context.Background(), contracts.JobTypeRelease, "estafette", "estafette-extension-slack-build-status", "390605593734184965")

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
			BuilderConfig: contracts.BuilderConfig{
				JobType: contracts.JobTypeBuild,
				Build: &contracts.Build{
					ID: "390605593734184965",
				},
			},
			OperatingSystem: manifest.OperatingSystemWindows,
		}

		builderConfig := contracts.BuilderConfig{}

		// act
		affinity := ciBuilderClient.getCiBuilderJobAffinity(context.Background(), ciBuilderParams, builderConfig)

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
			BuilderConfig: contracts.BuilderConfig{
				JobType: contracts.JobTypeBuild,
				Build: &contracts.Build{
					ID: "390605593734184965",
				},
			},
			OperatingSystem: manifest.OperatingSystemWindows,
		}

		builderConfig := contracts.BuilderConfig{}

		// act
		ciBuilderClient.getCiBuilderJobAffinity(context.Background(), ciBuilderParams, builderConfig)

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
			BuilderConfig: contracts.BuilderConfig{
				JobType: contracts.JobTypeRelease,
				Build: &contracts.Build{
					ID: "390605593734184965",
				},
			},
			OperatingSystem: manifest.OperatingSystemWindows,
		}

		builderConfig := contracts.BuilderConfig{}

		// act
		affinity := ciBuilderClient.getCiBuilderJobAffinity(context.Background(), ciBuilderParams, builderConfig)

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
			BuilderConfig: contracts.BuilderConfig{
				JobType: contracts.JobTypeRelease,
				Build: &contracts.Build{
					ID: "390605593734184965",
				},
			},
			OperatingSystem: manifest.OperatingSystemWindows,
		}

		builderConfig := contracts.BuilderConfig{}

		// act
		ciBuilderClient.getCiBuilderJobAffinity(context.Background(), ciBuilderParams, builderConfig)

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
			BuilderConfig: contracts.BuilderConfig{
				JobType: contracts.JobTypeBuild,
				Build: &contracts.Build{
					ID: "390605593734184965",
				},
			},
			OperatingSystem: manifest.OperatingSystemWindows,
		}

		builderConfig := contracts.BuilderConfig{}

		// act
		tolerations := ciBuilderClient.getCiBuilderJobTolerations(context.Background(), ciBuilderParams, builderConfig)

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
			BuilderConfig: contracts.BuilderConfig{
				JobType: contracts.JobTypeRelease,
				Release: &contracts.Release{
					ID: "390605593734184965",
				},
			},
			OperatingSystem: manifest.OperatingSystemWindows,
		}

		builderConfig := contracts.BuilderConfig{}

		// act
		tolerations := ciBuilderClient.getCiBuilderJobTolerations(context.Background(), ciBuilderParams, builderConfig)

		assert.Equal(t, 2, len(tolerations))
		assert.Equal(t, "node.kubernetes.io/os", tolerations[1].Key)
		assert.Equal(t, v1.TaintEffectNoSchedule, tolerations[1].Effect)
		assert.Equal(t, "windows", tolerations[1].Value)
		assert.Equal(t, v1.TolerationOpEqual, tolerations[1].Operator)
	})
}

func TestGetBuilderConfig(t *testing.T) {

	t.Run("ReturnsEventsForTriggeredEvents", func(t *testing.T) {

		ciBuilderClient := &client{
			encryptedConfig: &api.APIConfig{
				TrustedImages: []*contracts.TrustedImageConfig{},
				Credentials:   []*contracts.CredentialConfig{},
			},
			config: &api.APIConfig{
				Auth: &api.AuthConfig{
					JWT: &api.JWTConfig{
						Key: "abcd",
					},
				},
				APIServer: &api.APIServerConfig{
					ServiceURL: "https://ci.estafette.api",
				},
			},
		}
		ciBuilderParams := CiBuilderParams{
			BuilderConfig: contracts.BuilderConfig{
				JobType: contracts.JobTypeBuild,
				Build: &contracts.Build{
					ID: "390605593734184965",
				},
				Git:      &contracts.GitConfig{},
				Version:  &contracts.VersionConfig{},
				Manifest: &manifest.EstafetteManifest{},
				Events: []manifest.EstafetteEvent{
					{
						Name:  "trigger1",
						Fired: false,
						Pipeline: &manifest.EstafettePipelineEvent{
							RepoSource:   "github.com",
							RepoOwner:    "estafette",
							RepoName:     "repo1",
							Branch:       "main",
							Event:        "finished",
							Status:       "succeeded",
							BuildVersion: "1.0.5",
						},
					},
					{
						Name:  "trigger2",
						Fired: false,
						Pipeline: &manifest.EstafettePipelineEvent{
							RepoSource:   "github.com",
							RepoOwner:    "estafette",
							RepoName:     "repo2",
							Branch:       "main",
							Event:        "finished",
							Status:       "succeeded",
							BuildVersion: "6.4.3",
						},
					},
				},
			},
		}
		jobName := "build-estafette-estafette-ci-api-390605593734184965"

		// act
		builderConfig, err := ciBuilderClient.getBuilderConfig(context.Background(), ciBuilderParams, jobName)

		assert.Nil(t, err)
		assert.Equal(t, 2, len(builderConfig.Events))
		assert.Equal(t, "trigger1", builderConfig.Events[0].Name)
		assert.Equal(t, "repo1", builderConfig.Events[0].Pipeline.RepoName)
		assert.Equal(t, "1.0.5", builderConfig.Events[0].Pipeline.BuildVersion)
		assert.Equal(t, "trigger2", builderConfig.Events[1].Name)
		assert.Equal(t, "repo2", builderConfig.Events[1].Pipeline.RepoName)
		assert.Equal(t, "6.4.3", builderConfig.Events[1].Pipeline.BuildVersion)
	})

	t.Run("ReturnsLegacyFields", func(t *testing.T) {

		ciBuilderClient := &client{
			encryptedConfig: &api.APIConfig{
				TrustedImages: []*contracts.TrustedImageConfig{},
				Credentials:   []*contracts.CredentialConfig{},
			},
			config: &api.APIConfig{
				Auth: &api.AuthConfig{
					JWT: &api.JWTConfig{
						Key: "abcd",
					},
				},
				APIServer: &api.APIServerConfig{
					ServiceURL: "https://ci.estafette.api",
				},
			},
		}
		ciBuilderParams := CiBuilderParams{
			BuilderConfig: contracts.BuilderConfig{
				JobType: contracts.JobTypeBuild,
				Build: &contracts.Build{
					ID: "390605593734184965",
				},
				Git: &contracts.GitConfig{},
				Version: &contracts.VersionConfig{
					Version:                 "1.0.7456",
					CurrentCounter:          7456,
					MaxCounter:              7456,
					MaxCounterCurrentBranch: 7456,
				},
				Manifest: &manifest.EstafetteManifest{},
			},
		}
		jobName := "build-estafette-estafette-ci-api-390605593734184965"

		// act
		builderConfig, err := ciBuilderClient.getBuilderConfig(context.Background(), ciBuilderParams, jobName)

		assert.Nil(t, err)
		assert.Equal(t, "1.0.7456", builderConfig.Version.Version)
		assert.Equal(t, "390605593734184965", builderConfig.Build.ID)
		assert.Equal(t, contracts.JobTypeBuild, builderConfig.JobType)
	})
}
