package bigquery

import (
	"context"
	"fmt"
	"time"

	"cloud.google.com/go/bigquery"
	"github.com/estafette/estafette-ci-api/config"
	"github.com/rs/zerolog/log"
)

// Client is the interface for connecting to bigquery
type Client interface {
	Init() error
	CheckIfDatasetExists() bool
	CheckIfTableExists(table string) bool
	CreateTable(table string, typeForSchema interface{}, partitionField string, waitReady bool) error
	UpdateTableSchema(table string, typeForSchema interface{}) error
	InsertBuildEvent(event PipelineBuildEvent) error
	InsertReleaseEvent(event PipelineReleaseEvent) error
}

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

// NewClient returns new bigquery.Client
func NewClient(config *config.BigQueryConfig) (Client, error) {

	if config == nil || !config.Enable {
		return &client{
			config: config,
		}, nil
	}

	ctx := context.Background()

	bigqueryClient, err := bigquery.NewClient(ctx, config.ProjectID)
	if err != nil {
		return nil, err
	}

	return &client{
		client:                 bigqueryClient,
		config:                 config,
		buildEventsTableName:   "estafette_ci_build_events",
		releaseEventsTableName: "estafette_ci_release_events",
	}, nil
}

type client struct {
	client                 *bigquery.Client
	config                 *config.BigQueryConfig
	buildEventsTableName   string
	releaseEventsTableName string
}

func (bqc *client) Init() (err error) {

	if bqc.config == nil || !bqc.config.Enable {
		return
	}

	log.Info().Msgf("Initializing BigQuery tables %v and %v...", bqc.buildEventsTableName, bqc.releaseEventsTableName)

	datasetExists := bqc.CheckIfDatasetExists()
	if !datasetExists {
		return fmt.Errorf("Dataset %v does not exist, create it first; make sure to set the region you want your data to reside in", bqc.config.Dataset)
	}

	buildEventsTableExists := bqc.CheckIfTableExists(bqc.buildEventsTableName)
	if buildEventsTableExists {
		err = bqc.UpdateTableSchema(bqc.buildEventsTableName, PipelineBuildEvent{})
	} else {
		err = bqc.CreateTable(bqc.buildEventsTableName, PipelineBuildEvent{}, "inserted_at", true)
	}
	if err != nil {
		return
	}

	releaseEventsTableExists := bqc.CheckIfTableExists(bqc.releaseEventsTableName)
	if releaseEventsTableExists {
		err = bqc.UpdateTableSchema(bqc.releaseEventsTableName, PipelineReleaseEvent{})
	} else {
		err = bqc.CreateTable(bqc.releaseEventsTableName, PipelineReleaseEvent{}, "inserted_at", true)
	}
	if err != nil {
		return
	}

	return nil
}

func (bqc *client) CheckIfDatasetExists() bool {

	log.Info().Msgf("Checking if BigQuery dataset %v exists...", bqc.config.Dataset)

	ds := bqc.client.Dataset(bqc.config.Dataset)

	md, err := ds.Metadata(context.Background())
	if err != nil {
		log.Warn().Err(err).Msgf("Error retrieving metadata for dataset %v", bqc.config.Dataset)
	}

	return md != nil
}

func (bqc *client) CheckIfTableExists(table string) bool {

	log.Info().Msgf("Checking if BigQuery table %v exists...", table)

	tbl := bqc.client.Dataset(bqc.config.Dataset).Table(table)

	md, err := tbl.Metadata(context.Background())
	if err != nil {
		log.Warn().Err(err).Msgf("Error retrieving metadata for table %v in dataset %v", table, bqc.config.Dataset)
	}

	return md != nil
}

func (bqc *client) CreateTable(table string, typeForSchema interface{}, partitionField string, waitReady bool) error {

	log.Info().Msgf("Creating BigQuery table %v in dataset %v...", table, bqc.config.Dataset)

	tbl := bqc.client.Dataset(bqc.config.Dataset).Table(table)

	// infer the schema of the type
	schema, err := bigquery.InferSchema(typeForSchema)
	if err != nil {
		return err
	}

	tableMetadata := &bigquery.TableMetadata{
		Schema: schema,
	}

	// if partitionField is set use it for time partitioning
	if partitionField != "" {
		tableMetadata.TimePartitioning = &bigquery.TimePartitioning{
			Field: partitionField,
		}
	}

	// create the table
	err = tbl.Create(context.Background(), tableMetadata)
	if err != nil {
		return err
	}

	if waitReady {
		for {
			if bqc.CheckIfTableExists(table) {
				break
			}
			time.Sleep(time.Second)
		}
	}

	log.Info().Msgf("Finished creating BigQuery table %v in dataset %v", table, bqc.config.Dataset)

	return nil
}

func (bqc *client) UpdateTableSchema(table string, typeForSchema interface{}) error {

	log.Info().Msgf("Updating BigQuery table %v schema in dataset %v...", table, bqc.config.Dataset)

	tbl := bqc.client.Dataset(bqc.config.Dataset).Table(table)

	// infer the schema of the type
	schema, err := bigquery.InferSchema(typeForSchema)
	if err != nil {
		return err
	}

	meta, err := tbl.Metadata(context.Background())
	if err != nil {
		return err
	}

	update := bigquery.TableMetadataToUpdate{
		Schema: schema,
	}
	if _, err := tbl.Update(context.Background(), update, meta.ETag); err != nil {
		return err
	}

	return nil
}

func (bqc *client) InsertBuildEvent(event PipelineBuildEvent) error {

	if bqc.config == nil || !bqc.config.Enable {
		return nil
	}

	tbl := bqc.client.Dataset(bqc.config.Dataset).Table(bqc.buildEventsTableName)

	u := tbl.Uploader()

	if err := u.Put(context.Background(), event); err != nil {
		return err
	}

	return nil
}

func (bqc *client) InsertReleaseEvent(event PipelineReleaseEvent) error {

	if bqc.config == nil || !bqc.config.Enable {
		return nil
	}

	tbl := bqc.client.Dataset(bqc.config.Dataset).Table(bqc.releaseEventsTableName)

	u := tbl.Uploader()

	if err := u.Put(context.Background(), event); err != nil {
		return err
	}

	return nil
}
