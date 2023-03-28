package bigquery

import (
	"context"
	"fmt"
	"time"

	"cloud.google.com/go/bigquery"
	"github.com/estafette/estafette-ci-api/pkg/api"
	"github.com/rs/zerolog/log"
)

// Client is the interface for connecting to bigquery
//
//go:generate mockgen -package=bigquery -destination ./mock.go -source=client.go
type Client interface {
	Init(ctx context.Context) (err error)
	CheckIfDatasetExists(ctx context.Context) (exists bool)
	CheckIfTableExists(ctx context.Context, table string) (exists bool)
	CreateTable(ctx context.Context, table string, typeForSchema interface{}, partitionField string, waitReady bool) (err error)
	UpdateTableSchema(ctx context.Context, table string, typeForSchema interface{}) (err error)
	InsertBuildEvent(ctx context.Context, event PipelineBuildEvent) (err error)
	InsertReleaseEvent(ctx context.Context, event PipelineReleaseEvent) (err error)
}

// NewClient returns new bigquery.Client
func NewClient(config *api.APIConfig, bigqueryClient *bigquery.Client) Client {
	if config == nil || config.Integrations == nil || config.Integrations.BigQuery == nil || !config.Integrations.BigQuery.Enable {
		return &client{
			enabled: false,
		}
	}

	return &client{
		enabled:                true,
		client:                 bigqueryClient,
		config:                 config,
		buildEventsTableName:   "estafette_ci_build_events",
		releaseEventsTableName: "estafette_ci_release_events",
	}
}

type client struct {
	enabled                bool
	client                 *bigquery.Client
	config                 *api.APIConfig
	buildEventsTableName   string
	releaseEventsTableName string
}

func (c *client) Init(ctx context.Context) (err error) {

	if !c.enabled {
		return
	}

	log.Debug().Msgf("Initializing BigQuery tables %v and %v...", c.buildEventsTableName, c.releaseEventsTableName)

	datasetExists := c.CheckIfDatasetExists(ctx)
	if !datasetExists {
		return fmt.Errorf("Dataset %v does not exist, create it first; make sure to set the region you want your data to reside in", c.config.Integrations.BigQuery.Dataset)
	}

	buildEventsTableExists := c.CheckIfTableExists(ctx, c.buildEventsTableName)
	if buildEventsTableExists {
		err = c.UpdateTableSchema(ctx, c.buildEventsTableName, PipelineBuildEvent{})
	} else {
		err = c.CreateTable(ctx, c.buildEventsTableName, PipelineBuildEvent{}, "inserted_at", true)
	}
	if err != nil {
		return
	}

	releaseEventsTableExists := c.CheckIfTableExists(ctx, c.releaseEventsTableName)
	if releaseEventsTableExists {
		err = c.UpdateTableSchema(ctx, c.releaseEventsTableName, PipelineReleaseEvent{})
	} else {
		err = c.CreateTable(ctx, c.releaseEventsTableName, PipelineReleaseEvent{}, "inserted_at", true)
	}
	if err != nil {
		return
	}

	return nil
}

func (c *client) CheckIfDatasetExists(ctx context.Context) bool {

	if !c.enabled {
		return false
	}

	log.Debug().Msgf("Checking if BigQuery dataset %v exists...", c.config.Integrations.BigQuery.Dataset)

	ds := c.client.Dataset(c.config.Integrations.BigQuery.Dataset)

	md, err := ds.Metadata(ctx)
	if err != nil {
		log.Error().Err(err).Msgf("Error retrieving metadata for dataset %v", c.config.Integrations.BigQuery.Dataset)
	}

	return md != nil
}

func (c *client) CheckIfTableExists(ctx context.Context, table string) bool {

	if !c.enabled {
		return false
	}

	log.Debug().Msgf("Checking if BigQuery table %v exists...", table)

	tbl := c.client.Dataset(c.config.Integrations.BigQuery.Dataset).Table(table)

	md, err := tbl.Metadata(ctx)
	if err != nil {
		log.Error().Err(err).Msgf("Error retrieving metadata for table %v in dataset %v", table, c.config.Integrations.BigQuery.Dataset)
	}

	return md != nil
}

func (c *client) CreateTable(ctx context.Context, table string, typeForSchema interface{}, partitionField string, waitReady bool) error {

	if !c.enabled {
		return nil
	}

	log.Debug().Msgf("Creating BigQuery table %v in dataset %v...", table, c.config.Integrations.BigQuery.Dataset)

	tbl := c.client.Dataset(c.config.Integrations.BigQuery.Dataset).Table(table)

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
	err = tbl.Create(ctx, tableMetadata)
	if err != nil {
		return err
	}

	if waitReady {
		for {
			if c.CheckIfTableExists(ctx, table) {
				break
			}
			time.Sleep(time.Second)
		}
	}

	log.Debug().Msgf("Finished creating BigQuery table %v in dataset %v", table, c.config.Integrations.BigQuery.Dataset)

	return nil
}

func (c *client) UpdateTableSchema(ctx context.Context, table string, typeForSchema interface{}) error {

	if !c.enabled {
		return nil
	}

	log.Debug().Msgf("Updating BigQuery table %v schema in dataset %v...", table, c.config.Integrations.BigQuery.Dataset)

	tbl := c.client.Dataset(c.config.Integrations.BigQuery.Dataset).Table(table)

	// infer the schema of the type
	schema, err := bigquery.InferSchema(typeForSchema)
	if err != nil {
		return err
	}

	meta, err := tbl.Metadata(ctx)
	if err != nil {
		return err
	}

	update := bigquery.TableMetadataToUpdate{
		Schema: schema,
	}
	if _, err := tbl.Update(ctx, update, meta.ETag); err != nil {
		return err
	}

	return nil
}

func (c *client) InsertBuildEvent(ctx context.Context, event PipelineBuildEvent) error {

	if !c.enabled {
		return nil
	}

	tbl := c.client.Dataset(c.config.Integrations.BigQuery.Dataset).Table(c.buildEventsTableName)

	u := tbl.Uploader()

	if err := u.Put(ctx, event); err != nil {
		return err
	}

	return nil
}

func (c *client) InsertReleaseEvent(ctx context.Context, event PipelineReleaseEvent) error {

	if !c.enabled {
		return nil
	}

	tbl := c.client.Dataset(c.config.Integrations.BigQuery.Dataset).Table(c.releaseEventsTableName)

	u := tbl.Uploader()

	if err := u.Put(ctx, event); err != nil {
		return err
	}

	return nil
}
