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
	ID           string    `jsonapi:"primary,pipeline"`
	RepoSource   string    `jsonapi:"attr,repoSource"`
	RepoOwner    string    `jsonapi:"attr,repoOwner"`
	RepoName     string    `jsonapi:"attr,repoName"`
	RepoBranch   string    `jsonapi:"attr,repoBranch"`
	RepoRevision string    `jsonapi:"attr,repoRevision"`
	BuildVersion string    `jsonapi:"attr,buildVersion"`
	BuildStatus  string    `jsonapi:"attr,buildStatus"`
	Labels       string    `jsonapi:"attr,labels"`
	Manifest     string    `jsonapi:"attr,manifest"`
	InsertedAt   time.Time `jsonapi:"attr,insertedAt"`
	UpdatedAt    time.Time `jsonapi:"attr,updatedAt"`
}
