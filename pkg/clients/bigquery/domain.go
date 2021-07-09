package bigquery

import (
	"time"

	"cloud.google.com/go/bigquery"
)

// PipelineBuildEvent tracks a build once it's finished
type PipelineBuildEvent struct {
	BuildID      int    `bigquery:"build_id"`
	RepoSource   string `bigquery:"repo_source"`
	RepoOwner    string `bigquery:"repo_owner"`
	RepoName     string `bigquery:"repo_name"`
	RepoBranch   string `bigquery:"repo_branch"`
	RepoRevision string `bigquery:"repo_revision"`
	BuildVersion string `bigquery:"build_version"`
	BuildStatus  string `bigquery:"build_status"`

	Labels []struct {
		Key   string `bigquery:"key"`
		Value string `bigquery:"value"`
	} `bigquery:"labels"`

	InsertedAt time.Time `bigquery:"inserted_at"`
	UpdatedAt  time.Time `bigquery:"updated_at"`

	Commits []struct {
		Message string `bigquery:"message"`
		Author  struct {
			Email string `bigquery:"email"`
		} `bigquery:"author"`
	} `bigquery:"commits"`

	CPURequest     bigquery.NullFloat64 `bigquery:"cpu_request"`
	CPULimit       bigquery.NullFloat64 `bigquery:"cpu_limit"`
	CPUMaxUsage    bigquery.NullFloat64 `bigquery:"cpu_max_usage"`
	MemoryRequest  bigquery.NullFloat64 `bigquery:"memory_request"`
	MemoryLimit    bigquery.NullFloat64 `bigquery:"memory_limit"`
	MemoryMaxUsage bigquery.NullFloat64 `bigquery:"memory_max_usage"`

	TotalDuration time.Duration `bigquery:"duration"`
	TimeToRunning time.Duration `bigquery:"time_to_running"`

	Manifest string `bigquery:"manifest"`

	Jobs []Job `bigquery:"logs"`
}

// PipelineReleaseEvent tracks a release once it's finished
type PipelineReleaseEvent struct {
	ReleaseID      int    `bigquery:"release_id"`
	RepoSource     string `bigquery:"repo_source"`
	RepoOwner      string `bigquery:"repo_owner"`
	RepoName       string `bigquery:"repo_name"`
	ReleaseTarget  string `bigquery:"release_target"`
	ReleaseVersion string `bigquery:"release_version"`
	ReleaseStatus  string `bigquery:"release_status"`

	Labels []struct {
		Key   string `bigquery:"key"`
		Value string `bigquery:"value"`
	} `bigquery:"labels"`

	InsertedAt time.Time `bigquery:"inserted_at"`
	UpdatedAt  time.Time `bigquery:"updated_at"`

	CPURequest     bigquery.NullFloat64 `bigquery:"cpu_request"`
	CPULimit       bigquery.NullFloat64 `bigquery:"cpu_limit"`
	CPUMaxUsage    bigquery.NullFloat64 `bigquery:"cpu_max_usage"`
	MemoryRequest  bigquery.NullFloat64 `bigquery:"memory_request"`
	MemoryLimit    bigquery.NullFloat64 `bigquery:"memory_limit"`
	MemoryMaxUsage bigquery.NullFloat64 `bigquery:"memory_max_usage"`

	TotalDuration time.Duration `bigquery:"duration"`
	TimeToRunning time.Duration `bigquery:"time_to_running"`

	Jobs []Job `bigquery:"logs"`
}

// Job represent and actual job execution; a build / release can have multiple runs of a job if Kubernetes reschedules it
type Job struct {
	JobID  int `bigquery:"job_id"`
	Stages []struct {
		Name           string `bigquery:"name"`
		ContainerImage struct {
			Name         string        `bigquery:"name"`
			Tag          string        `bigquery:"tag"`
			IsPulled     bool          `bigquery:"is_pulled"`
			ImageSize    int           `bigquery:"image_size"`
			PullDuration time.Duration `bigquery:"pull_duration"`
			IsTrusted    bool          `bigquery:"is_trusted"`
		} `bigquery:"container_image"`

		RunDuration time.Duration `bigquery:"run_duration"`
		LogLines    []struct {
			Timestamp  time.Time `bigquery:"timestamp"`
			StreamType string    `bigquery:"stream_type"`
			Text       string    `bigquery:"text"`
		} `bigquery:"log_lines"`
	} `bigquery:"stages"`
	InsertedAt time.Time `bigquery:"inserted_at"`
}
