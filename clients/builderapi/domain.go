package builderapi

import (
	"fmt"

	"github.com/estafette/estafette-ci-api/clients/cockroachdb"
	contracts "github.com/estafette/estafette-ci-contracts"
	manifest "github.com/estafette/estafette-ci-manifest"
)

// CiBuilderParams contains the parameters required to create a ci builder job
type CiBuilderParams struct {
	BuilderConfig        contracts.BuilderConfig
	EnvironmentVariables map[string]string
	OperatingSystem      manifest.OperatingSystem
	JobResources         cockroachdb.JobResources
}

// GetFullRepoPath returns the full path of the pipeline / build / release repository with source, owner and name
func (cbp *CiBuilderParams) GetFullRepoPath() string {
	if cbp.BuilderConfig.Git == nil {
		return ""
	}
	return fmt.Sprintf("%v/%v/%v", cbp.BuilderConfig.Git.RepoSource, cbp.BuilderConfig.Git.RepoOwner, cbp.BuilderConfig.Git.RepoName)
}

type ZeroLogLine struct {
	TailLogLine *contracts.TailLogLine `json:"tailLogLine"`
}

// CiBuilderEvent represents a finished estafette build
type CiBuilderEvent struct {
	JobName      string           `json:"job_name"`
	PodName      string           `json:"pod_name,omitempty"`
	RepoSource   string           `json:"repo_source,omitempty"`
	RepoOwner    string           `json:"repo_owner,omitempty"`
	RepoName     string           `json:"repo_name,omitempty"`
	RepoBranch   string           `json:"repo_branch,omitempty"`
	RepoRevision string           `json:"repo_revision,omitempty"`
	ReleaseID    string           `json:"release_id,omitempty"`
	BuildID      string           `json:"build_id,omitempty"`
	BuildStatus  contracts.Status `json:"build_status,omitempty"`
}
