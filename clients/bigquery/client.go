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
	Init(ctx context.Context) error
	CheckIfDatasetExists(ctx context.Context) bool
	CheckIfTableExists(ctx context.Context, table string) bool
	CreateTable(ctx context.Context, table string, typeForSchema interface{}, partitionField string, waitReady bool) error
	UpdateTableSchema(ctx context.Context, table string, typeForSchema interface{}) error
	InsertBuildEvent(ctx context.Context, event PipelineBuildEvent) error
	InsertReleaseEvent(ctx context.Context, event PipelineReleaseEvent) error
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

func (bqc *client) Init(ctx context.Context) (err error) {

	if bqc.config == nil || !bqc.config.Enable {
		return
	}

	log.Info().Msgf("Initializing BigQuery tables %v and %v...", bqc.buildEventsTableName, bqc.releaseEventsTableName)

	datasetExists := bqc.CheckIfDatasetExists(ctx)
	if !datasetExists {
		return fmt.Errorf("Dataset %v does not exist, create it first; make sure to set the region you want your data to reside in", bqc.config.Dataset)
	}

	buildEventsTableExists := bqc.CheckIfTableExists(ctx, bqc.buildEventsTableName)
	if buildEventsTableExists {
		err = bqc.UpdateTableSchema(ctx, bqc.buildEventsTableName, PipelineBuildEvent{})
	} else {
		err = bqc.CreateTable(ctx, bqc.buildEventsTableName, PipelineBuildEvent{}, "inserted_at", true)
	}
	if err != nil {
		return
	}

	releaseEventsTableExists := bqc.CheckIfTableExists(ctx, bqc.releaseEventsTableName)
	if releaseEventsTableExists {
		err = bqc.UpdateTableSchema(ctx, bqc.releaseEventsTableName, PipelineReleaseEvent{})
	} else {
		err = bqc.CreateTable(ctx, bqc.releaseEventsTableName, PipelineReleaseEvent{}, "inserted_at", true)
	}
	if err != nil {
		return
	}

	return nil
}

func (bqc *client) CheckIfDatasetExists(ctx context.Context) bool {

	log.Info().Msgf("Checking if BigQuery dataset %v exists...", bqc.config.Dataset)

	ds := bqc.client.Dataset(bqc.config.Dataset)

	md, err := ds.Metadata(context.Background())
	if err != nil {
		log.Warn().Err(err).Msgf("Error retrieving metadata for dataset %v", bqc.config.Dataset)
	}

	return md != nil
}

func (bqc *client) CheckIfTableExists(ctx context.Context, table string) bool {

	log.Info().Msgf("Checking if BigQuery table %v exists...", table)

	tbl := bqc.client.Dataset(bqc.config.Dataset).Table(table)

	md, err := tbl.Metadata(context.Background())
	if err != nil {
		log.Warn().Err(err).Msgf("Error retrieving metadata for table %v in dataset %v", table, bqc.config.Dataset)
	}

	return md != nil
}

func (bqc *client) CreateTable(ctx context.Context, table string, typeForSchema interface{}, partitionField string, waitReady bool) error {

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
			if bqc.CheckIfTableExists(ctx, table) {
				break
			}
			time.Sleep(time.Second)
		}
	}

	log.Info().Msgf("Finished creating BigQuery table %v in dataset %v", table, bqc.config.Dataset)

	return nil
}

func (bqc *client) UpdateTableSchema(ctx context.Context, table string, typeForSchema interface{}) error {

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

func (bqc *client) InsertBuildEvent(ctx context.Context, event PipelineBuildEvent) error {

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

func (bqc *client) InsertReleaseEvent(ctx context.Context, event PipelineReleaseEvent) error {

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
