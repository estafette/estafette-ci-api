package database

import "time"

// JobResources represents the used cpu and memory resources for a job and the measured maximum once it's done
type JobResources struct {
	CPURequest     float64
	CPULimit       float64
	CPUMaxUsage    float64
	MemoryRequest  float64
	MemoryLimit    float64
	MemoryMaxUsage float64
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
