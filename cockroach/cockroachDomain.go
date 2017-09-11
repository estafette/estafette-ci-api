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
	Id           int
	RepoFullName string
	RepoBranch   string
	RepoRevision string
	RepoSource   string
	LogText      string
	InsertedAt   time.Time
}
