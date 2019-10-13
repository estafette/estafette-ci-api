package estafette

import (
	contracts "github.com/estafette/estafette-ci-contracts"
	manifest "github.com/estafette/estafette-ci-manifest"
)

// CiBuilderEvent represents a finished estafette build
type CiBuilderEvent struct {
	JobName      string `json:"job_name"`
	PodName      string `json:"job_name,omitempty"`
	RepoSource   string `json:"repo_source,omitempty"`
	RepoOwner    string `json:"repo_owner,omitempty"`
	RepoName     string `json:"repo_name,omitempty"`
	RepoBranch   string `json:"repo_branch,omitempty"`
	RepoRevision string `json:"repo_revision,omitempty"`
	ReleaseID    string `json:"release_id,omitempty"`
	BuildID      string `json:"build_id,omitempty"`
	BuildStatus  string `json:"build_status,omitempty"`
}

// CiBuilderLogLine represents a line logged by the ci builder
type CiBuilderLogLine struct {
	Time     string `json:"time"`
	Severity string `json:"severity"`
	Message  string `json:"message"`
}

// CiBuilderParams contains the parameters required to create a ci builder job
type CiBuilderParams struct {
	JobType              string
	RepoSource           string
	RepoOwner            string
	RepoName             string
	RepoURL              string
	RepoBranch           string
	RepoRevision         string
	EnvironmentVariables map[string]string
	Track                string
	OperatingSystem      string
	AutoIncrement        int
	VersionNumber        string
	//HasValidManifest     bool
	Manifest           manifest.EstafetteManifest
	ReleaseName        string
	ReleaseAction      string
	ReleaseID          int
	ReleaseTriggeredBy string
	BuildID            int
	TriggeredByEvents  []manifest.EstafetteEvent
}

type zeroLogLine struct {
	TailLogLine *contracts.TailLogLine `json:"tailLogLine"`
}
