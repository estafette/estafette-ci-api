package cockroach

import (
	"time"

	manifest "github.com/estafette/estafette-ci-manifest"
)

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

// EstafetteTriggerDb represents a trigger defined in a manifest
type EstafetteTriggerDb struct {
	ID         int
	RepoSource string
	RepoOwner  string
	RepoName   string
	Trigger    *manifest.EstafetteTrigger
	InsertedAt time.Time
	UpdatedAt  time.Time
}
