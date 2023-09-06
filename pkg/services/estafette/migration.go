package estafette

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"google.golang.org/grpc"
	"k8s.io/utils/ptr"
	"sync/atomic"
	"time"

	"github.com/rs/zerolog/log"
	"google.golang.org/grpc/metadata"

	"github.com/estafette/estafette-ci-api/pkg/migrationpb"
	"github.com/estafette/migration"
)

const (
	_100MiB = 100 * 1024 * 1024
)

var (
	// maximumParallelMigrations for single pod
	maximumParallelMigrations int64 = 2
	stopWorkers                     = &atomic.Bool{}
	inProgressMigrations            = &atomic.Int64{}
	migrationTimeout                = 15 * time.Minute
	pollingInterval                 = 10 * time.Second
)

func (h *Handler) PollMigrationTasks(stop <-chan struct{}, done func()) {
	ctx := context.Background()
	go func() {
		<-stop
		stopWorkers.Store(true)
	}()
POLL:
	for {
		if stopWorkers.Load() {
			log.Info().Msg("stopping migration workers")
			done()
			break
		}
		time.Sleep(pollingInterval)
		maxTasks := maximumParallelMigrations - inProgressMigrations.Load()
		if maxTasks < 1 {
			goto POLL
		}
		// this picks the next migration tasks from queue
		tasks, err := h.databaseClient.PickMigration(ctx, maxTasks)
		if err != nil {
			log.Fatal().Err(err).Msg("error picking migration from database")
			return
		}
		for _, task := range tasks {
			inProgressMigrations.Add(1)
			go h.migration(task)
		}
	}
}

// migration worker
func (h *Handler) migration(task *migration.Task) {
	log.Info().Str("taskID", task.ID).Str("fromFQN", task.FromFQN()).Str("toFQN", task.ToFQN()).Msgf("Starting migration")
	defer inProgressMigrations.Add(-1)
	ctx, cancel := context.WithTimeout(context.Background(), migrationTimeout)
	defer cancel()
	stages := migration.
		NewStages(h.databaseClient.UpdateMigration, task).
		Set(migration.ReleasesStage, h.databaseClient.MigrateReleases).
		Set(migration.ReleaseLogsStage, h.databaseClient.MigrateReleaseLogs).
		Set(migration.ReleaseLogObjectsStage, h.migrateReleaseLogObjects).
		Set(migration.BuildsStage, h.databaseClient.MigrateBuilds).
		Set(migration.BuildLogsStage, h.databaseClient.MigrateBuildLogs).
		Set(migration.BuildLogObjectsStage, h.migrateBuildLogObjects).
		Set(migration.BuildVersionsStage, h.databaseClient.MigrateBuildVersions).
		Set(migration.ComputedTablesStage, h.databaseClient.MigrateComputedTables).
		Set(migration.ArchiveStage, h.archiveRepository).
		Set(migration.CompletedStage, migration.CompletedExecutor)
	var result bool
	for stages.HasNext() {
		if result = stages.ExecuteNext(ctx); !result {
			return
		}
		if stopWorkers.Load() {
			// put the task back to queue if this pod is shutting down
			task.Status = migration.StatusQueued
			log.Warn().Str("taskID", task.ID).Str("fromFQN", task.FromFQN()).Str("toFQN", task.ToFQN()).Msgf("Migration worker is shutting down, putting task back to queue")
			return
		}
	}
}

func (h *Handler) archiveRepository(ctx context.Context, task *migration.Task) error {
	return h.databaseClient.ArchiveComputedPipeline(ctx, task.ToSource, task.ToOwner, task.ToName)
}

func (h *Handler) migrateReleaseLogObjects(ctx context.Context, task *migration.Task) error {
	changes, err := h.databaseClient.GetMigratedReleaseLogs(ctx, task)
	if err != nil {
		return err
	}
	return h.migrateLogObjects(ctx, migration.ReleaseLog, changes, task)
}

func (h *Handler) migrateBuildLogObjects(ctx context.Context, task *migration.Task) error {
	changes, err := h.databaseClient.GetMigratedBuildLogs(ctx, task)
	if err != nil {
		return err
	}
	return h.migrateLogObjects(ctx, migration.BuildLog, changes, task)
}

func (h *Handler) migrateLogObjects(ctx context.Context, logType migration.LogType, changes []migration.Change, task *migration.Task) error {
	md := metadata.New(map[string]string{"messageuniqueid": task.ID})
	ctx = metadata.NewOutgoingContext(ctx, md)
	stream, err := h.gcsMigratorClient.Migrate(ctx, grpc.MaxCallRecvMsgSize(_100MiB))
	if err != nil {
		log.Fatal().Msgf("gcsMigratorClient.Migrate failed: %v", err)
	}
	requests := make(map[int64]migration.Change)
	for index, change := range changes {
		req := &migrationpb.Request{
			Id: int64(index),
			FromLogObject: &migrationpb.LogObject{
				Bucket: h.config.Integrations.CloudStorage.Bucket,
				// logs/bitbucket.org/xivart/repo1/builds/A.log
				Path: fmt.Sprintf("logs/%s/%s/%d.log", task.FromFQN(), logType, change.FromID),
			},
			ToLogObject: &migrationpb.LogObject{
				Bucket: h.config.Integrations.CloudStorage.Bucket,
				// logs/github.com/xivart/repo1/builds/B.log
				Path: fmt.Sprintf("logs/%s/%s/%d.log", task.ToFQN(), logType, change.ToID),
			},
		}
		err = stream.Send(req)
		if err != nil {
			return fmt.Errorf("gcsMigratorClient.Migrate: stream.Send(%v) failed: %v", req, err)
		}
		requests[int64(index)] = change
	}
	var response *migrationpb.Response
	response, err = stream.CloseAndRecv()
	if err != nil {
		return fmt.Errorf("gcsMigratorClient.Migrate: stream.CloseAndRecv() failed: %w", err)
	}
	notFoundErrors := fmt.Sprintf("not found %s logs:", logType)
	unknownErrors := fmt.Sprintf("unknown gcs-migrator %s logs errors:", logType)
	totalErrors := 0
MIGRATIONS:
	for _, m := range response.Migrations {
		req := requests[m.Id]
		if m.Error != nil {
			for _, _err := range m.Error.Errors {
				if _err.Reason == "retentionPolicyNotMe" {
					continue MIGRATIONS
				}
			}
			totalErrors++
			if m.Error.Code == 404 {
				notFoundErrors += fmt.Sprintf(" %d.log,", req.FromID)
				continue
			}
			data, err := json.Marshal(m.Error)
			if err != nil {
				return fmt.Errorf("error marshalling Error field from response: %w", err)
			}
			paddedData := bytes.ReplaceAll(data, []byte("\n"), []byte("\n\t"))
			unknownErrors += fmt.Sprintf("\n\tfrom: %d, to: %d err: %s", req.FromID, req.ToID, paddedData)
		}
	}
	if totalErrors > 0 {
		task.ErrorDetails = ptr.To(fmt.Sprintf("%s\n%s", notFoundErrors, unknownErrors))
		if totalErrors == len(changes) {
			return fmt.Errorf("errors while migrating %s log objects", logType)
		}
	}
	return nil
}

func (h *Handler) rollbackLogObjects(ctx context.Context, task *migration.Task) error {
	return h.cloudStorageClient.DeleteLogs(ctx, task.ToSource, task.ToOwner, task.ToName)
}
