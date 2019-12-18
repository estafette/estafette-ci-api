package bigquery

import "context"

type MockClient struct {
	InitFunc                 func(ctx context.Context) (err error)
	CheckIfDatasetExistsFunc func(ctx context.Context) (exists bool)
	CheckIfTableExistsFunc   func(ctx context.Context, table string) (exists bool)
	CreateTableFunc          func(ctx context.Context, table string, typeForSchema interface{}, partitionField string, waitReady bool) (err error)
	UpdateTableSchemaFunc    func(ctx context.Context, table string, typeForSchema interface{}) (err error)
	InsertBuildEventFunc     func(ctx context.Context, event PipelineBuildEvent) (err error)
	InsertReleaseEventFunc   func(ctx context.Context, event PipelineReleaseEvent) (err error)
}

func (c MockClient) Init(ctx context.Context) (err error) {
	if c.InitFunc == nil {
		return
	}
	return c.InitFunc(ctx)
}

func (c MockClient) CheckIfDatasetExists(ctx context.Context) (exists bool) {
	if c.CheckIfDatasetExistsFunc == nil {
		return
	}
	return c.CheckIfDatasetExistsFunc(ctx)
}

func (c MockClient) CheckIfTableExists(ctx context.Context, table string) (exists bool) {
	if c.CheckIfTableExistsFunc == nil {
		return
	}
	return c.CheckIfTableExistsFunc(ctx, table)
}

func (c MockClient) CreateTable(ctx context.Context, table string, typeForSchema interface{}, partitionField string, waitReady bool) (err error) {
	if c.CreateTableFunc == nil {
		return
	}
	return c.CreateTableFunc(ctx, table, typeForSchema, partitionField, waitReady)
}

func (c MockClient) UpdateTableSchema(ctx context.Context, table string, typeForSchema interface{}) (err error) {
	if c.UpdateTableSchemaFunc == nil {
		return
	}
	return c.UpdateTableSchemaFunc(ctx, table, typeForSchema)
}

func (c MockClient) InsertBuildEvent(ctx context.Context, event PipelineBuildEvent) (err error) {
	if c.InsertBuildEventFunc == nil {
		return
	}
	return c.InsertBuildEventFunc(ctx, event)
}

func (c MockClient) InsertReleaseEvent(ctx context.Context, event PipelineReleaseEvent) (err error) {
	if c.InsertReleaseEventFunc == nil {
		return
	}
	return c.InsertReleaseEventFunc(ctx, event)
}
