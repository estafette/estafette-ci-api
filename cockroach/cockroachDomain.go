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

// Pipeline represents a pipeline with the latest build info, including version number, repo, branch, revision, labels and manifest
type Pipeline struct {
	ID           string    `jsonapi:"primary,pipelines"`
	RepoSource   string    `jsonapi:"attr,repo-source"`
	RepoOwner    string    `jsonapi:"attr,repo-owner"`
	RepoName     string    `jsonapi:"attr,repo-name"`
	RepoBranch   string    `jsonapi:"attr,repo-branch"`
	RepoRevision string    `jsonapi:"attr,repo-revision"`
	BuildVersion string    `jsonapi:"attr,build-version"`
	BuildStatus  string    `jsonapi:"attr,build-status"`
	Labels       string    `jsonapi:"attr,labels"`
	Manifest     string    `jsonapi:"attr,manifest"`
	InsertedAt   time.Time `jsonapi:"attr,inserted-at"`
	UpdatedAt    time.Time `jsonapi:"attr,updated-at"`
}

// Build represents a specific build, including version number, repo, branch, revision, labels and manifest
type Build struct {
	ID           string    `jsonapi:"primary,builds"`
	RepoSource   string    `jsonapi:"attr,repo-source"`
	RepoOwner    string    `jsonapi:"attr,repo-owner"`
	RepoName     string    `jsonapi:"attr,repo-name"`
	RepoBranch   string    `jsonapi:"attr,repo-branch"`
	RepoRevision string    `jsonapi:"attr,repo-revision"`
	BuildVersion string    `jsonapi:"attr,build-version"`
	BuildStatus  string    `jsonapi:"attr,build-status"`
	Labels       string    `jsonapi:"attr,labels"`
	Manifest     string    `jsonapi:"attr,manifest"`
	InsertedAt   time.Time `jsonapi:"attr,inserted-at"`
	UpdatedAt    time.Time `jsonapi:"attr,updated-at"`
}
