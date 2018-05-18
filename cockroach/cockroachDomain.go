package cockroach

import "time"

// BuildJobLogs represents the logs for a build job
type BuildJobLogs struct {
	RepoFullName string
	RepoBranch   string
	RepoRevision string
	RepoSource   string
	LogText      string
}

// BuildJobLogRow represents the logs for a build job as stored in the database
type BuildJobLogRow struct {
	ID           int
	RepoFullName string
	RepoBranch   string
	RepoRevision string
	RepoSource   string
	LogText      string
	InsertedAt   time.Time
}

// BuildVersionDetail represents a specific build, including version number, repo, branch, revision and manifest
type BuildVersionDetail struct {
	ID           int
	BuildVersion string
	RepoSource   string
	RepoFullName string
	RepoBranch   string
	RepoRevision string
	Manifest     string
	InsertedAt   time.Time
}

// Build represents a specific build, including version number, repo, branch, revision, labels and manifest
type Build struct {
	ID           int       `json:"id"`
	RepoSource   string    `json:"repoSource"`
	RepoOwner    string    `json:"repoOwner"`
	RepoName     string    `json:"repoName"`
	RepoBranch   string    `json:"repoBranch"`
	RepoRevision string    `json:"repoRevision"`
	BuildVersion string    `json:"buildVersion"`
	BuildStatus  string    `json:"buildStatus"`
	Labels       string    `json:"labels"`
	Manifest     string    `json:"manifest"`
	InsertedAt   time.Time `json:"insertedAt"`
	UpdatedAt    time.Time `json:"updatedAt"`
}
