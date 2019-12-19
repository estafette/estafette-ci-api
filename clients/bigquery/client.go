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
	Init(ctx context.Context) (err error)
	CheckIfDatasetExists(ctx context.Context) (exists bool)
	CheckIfTableExists(ctx context.Context, table string) (exists bool)
	CreateTable(ctx context.Context, table string, typeForSchema interface{}, partitionField string, waitReady bool) (err error)
	UpdateTableSchema(ctx context.Context, table string, typeForSchema interface{}) (err error)
	InsertBuildEvent(ctx context.Context, event PipelineBuildEvent) (err error)
	InsertReleaseEvent(ctx context.Context, event PipelineReleaseEvent) (err error)
}

// NewClient returns new bigquery.Client
func NewClient(config *config.BigQueryConfig, bigqueryClient *bigquery.Client) Client {

	if config == nil || !config.Enable {
		return &client{
			config: config,
		}
	}

	return &client{
		client:                 bigqueryClient,
		config:                 config,
		buildEventsTableName:   "estafette_ci_build_events",
		releaseEventsTableName: "estafette_ci_release_events",
	}
}

type client struct {
	client                 *bigquery.Client
	config                 *config.BigQueryConfig
	buildEventsTableName   string
	releaseEventsTableName string
}

func (c *client) Init(ctx context.Context) (err error) {

	if c.config == nil || !c.config.Enable {
		return
	}

	log.Info().Msgf("Initializing BigQuery tables %v and %v...", c.buildEventsTableName, c.releaseEventsTableName)

	datasetExists := c.CheckIfDatasetExists(ctx)
	if !datasetExists {
		return fmt.Errorf("Dataset %v does not exist, create it first; make sure to set the region you want your data to reside in", c.config.Dataset)
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

	log.Info().Msgf("Checking if BigQuery dataset %v exists...", c.config.Dataset)

	ds := c.client.Dataset(c.config.Dataset)

	md, err := ds.Metadata(context.Background())
	if err != nil {
		log.Warn().Err(err).Msgf("Error retrieving metadata for dataset %v", c.config.Dataset)
	}

	return md != nil
}

func (c *client) CheckIfTableExists(ctx context.Context, table string) bool {

	log.Info().Msgf("Checking if BigQuery table %v exists...", table)

	tbl := c.client.Dataset(c.config.Dataset).Table(table)

	md, err := tbl.Metadata(context.Background())
	if err != nil {
		log.Warn().Err(err).Msgf("Error retrieving metadata for table %v in dataset %v", table, c.config.Dataset)
	}

	return md != nil
}

func (c *client) CreateTable(ctx context.Context, table string, typeForSchema interface{}, partitionField string, waitReady bool) error {

	log.Info().Msgf("Creating BigQuery table %v in dataset %v...", table, c.config.Dataset)

	tbl := c.client.Dataset(c.config.Dataset).Table(table)

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
			if c.CheckIfTableExists(ctx, table) {
				break
			}
			time.Sleep(time.Second)
		}
	}

	log.Info().Msgf("Finished creating BigQuery table %v in dataset %v", table, c.config.Dataset)

	return nil
}

func (c *client) UpdateTableSchema(ctx context.Context, table string, typeForSchema interface{}) error {

	log.Info().Msgf("Updating BigQuery table %v schema in dataset %v...", table, c.config.Dataset)

	tbl := c.client.Dataset(c.config.Dataset).Table(table)

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

func (c *client) InsertBuildEvent(ctx context.Context, event PipelineBuildEvent) error {

	if c.config == nil || !c.config.Enable {
		return nil
	}

	tbl := c.client.Dataset(c.config.Dataset).Table(c.buildEventsTableName)

	u := tbl.Uploader()

	if err := u.Put(context.Background(), event); err != nil {
		return err
	}

	return nil
}

func (c *client) InsertReleaseEvent(ctx context.Context, event PipelineReleaseEvent) error {

	if c.config == nil || !c.config.Enable {
		return nil
	}

	tbl := c.client.Dataset(c.config.Dataset).Table(c.releaseEventsTableName)

	u := tbl.Uploader()

	if err := u.Put(context.Background(), event); err != nil {
		return err
	}

	return nil
}
