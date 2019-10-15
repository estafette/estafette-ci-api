package bigquery

import (
	"context"
	"fmt"
	"time"

	"cloud.google.com/go/bigquery"
	bqcontracts "github.com/estafette/estafette-ci-api/bigquery/contracts"
	"github.com/estafette/estafette-ci-api/config"
	"github.com/rs/zerolog/log"
)

// BigQueryClient is the interface for connecting to bigquery
type BigQueryClient interface {
	Init() error
	CheckIfDatasetExists() bool
	CheckIfTableExists(table string) bool
	CreateTable(table string, typeForSchema interface{}, partitionField string, waitReady bool) error
	UpdateTableSchema(table string, typeForSchema interface{}) error
	InsertBuildEvent(event bqcontracts.PipelineBuildEvent) error
	InsertReleaseEvent(event bqcontracts.PipelineReleaseEvent) error
}

type bigQueryClientImpl struct {
	client                 *bigquery.Client
	config                 config.BigQueryConfig
	buildEventsTableName   string
	releaseEventsTableName string
}

// NewBigQueryClient returns new BigQueryClient
func NewBigQueryClient(config config.BigQueryConfig) (BigQueryClient, error) {

	ctx := context.Background()

	bigqueryClient, err := bigquery.NewClient(ctx, config.ProjectID)
	if err != nil {
		return nil, err
	}

	return &bigQueryClientImpl{
		client:                 bigqueryClient,
		config:                 config,
		buildEventsTableName:   "estafette_ci_build_events",
		releaseEventsTableName: "estafette_ci_release_events",
	}, nil
}

func (bqc *bigQueryClientImpl) Init() (err error) {

	if !bqc.config.Enable {
		return
	}

	datasetExists := bqc.CheckIfDatasetExists()
	if !datasetExists {
		return fmt.Errorf("Dataset %v does not exist, create it first; make sure to set the region you want your data to reside in", bqc.config.Dataset)
	}

	buildEventsTableExists := bqc.CheckIfTableExists(bqc.buildEventsTableName)
	if buildEventsTableExists {
		err = bqc.UpdateTableSchema(bqc.buildEventsTableName, bqcontracts.PipelineBuildEvent{})
	} else {
		err = bqc.CreateTable(bqc.buildEventsTableName, bqcontracts.PipelineBuildEvent{}, "inserted_at", true)
	}
	if err != nil {
		return
	}

	releaseEventsTableExists := bqc.CheckIfTableExists(bqc.releaseEventsTableName)
	if releaseEventsTableExists {
		err = bqc.UpdateTableSchema(bqc.releaseEventsTableName, bqcontracts.PipelineReleaseEvent{})
	} else {
		err = bqc.CreateTable(bqc.releaseEventsTableName, bqcontracts.PipelineReleaseEvent{}, "inserted_at", true)
	}
	if err != nil {
		return
	}

	return nil
}

func (bqc *bigQueryClientImpl) CheckIfDatasetExists() bool {

	ds := bqc.client.Dataset(bqc.config.Dataset)

	md, err := ds.Metadata(context.Background())

	log.Error().Err(err).Msgf("Error retrieving metadata for dataset %v", bqc.config.Dataset)

	return md != nil
}

func (bqc *bigQueryClientImpl) CheckIfTableExists(table string) bool {

	tbl := bqc.client.Dataset(bqc.config.Dataset).Table(table)

	md, _ := tbl.Metadata(context.Background())

	// log.Error().Err(err).Msgf("Error retrieving metadata for table %v", table)

	return md != nil
}

func (bqc *bigQueryClientImpl) CreateTable(table string, typeForSchema interface{}, partitionField string, waitReady bool) error {
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

	return nil
}

func (bqc *bigQueryClientImpl) UpdateTableSchema(table string, typeForSchema interface{}) error {
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

func (bqc *bigQueryClientImpl) InsertBuildEvent(event bqcontracts.PipelineBuildEvent) error {

	if !bqc.config.Enable {
		return nil
	}

	tbl := bqc.client.Dataset(bqc.config.Dataset).Table(bqc.buildEventsTableName)

	u := tbl.Uploader()

	if err := u.Put(context.Background(), event); err != nil {
		return err
	}

	return nil
}

func (bqc *bigQueryClientImpl) InsertReleaseEvent(event bqcontracts.PipelineReleaseEvent) error {

	if !bqc.config.Enable {
		return nil
	}

	tbl := bqc.client.Dataset(bqc.config.Dataset).Table(bqc.releaseEventsTableName)

	u := tbl.Uploader()

	if err := u.Put(context.Background(), event); err != nil {
		return err
	}

	return nil
}
