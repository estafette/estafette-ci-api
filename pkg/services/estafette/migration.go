package estafette

import (
	"context"
	"encoding/json"
	"fmt"
	"sync/atomic"
	"time"

	"github.com/rs/zerolog/log"
	"google.golang.org/grpc/metadata"

	"github.com/estafette/estafette-ci-api/pkg/services/estafette/migrationpb"
	"github.com/estafette/migration"
)

var (
	concurrentMigrations int64 = 10
	inProgressMigrations       = &atomic.Int64{}
	migrationTimeout           = 10 * time.Minute
)

func (h *Handler) pollMigrationTasks() {
POLL:
	for {
		time.Sleep(5 * time.Second)
		if inProgressMigrations.Load() >= concurrentMigrations {
			goto POLL
		}
		inProgressMigrations.Add(1)
		// worker function
		go func() {
			defer inProgressMigrations.Add(-1)
			ctx, cancel := context.WithTimeout(context.Background(), migrationTimeout)
			defer cancel()
			// this picks the next migration from the queue and puts in_progress atomically
			task, err := h.databaseClient.PickMigration(ctx)
			if err != nil {
				log.Error().Err(err).Msg("Error picking migration from database")
				return
			}
			if task != nil {
				log.Info().Msgf("Starting migration from %v/%v/%v to %v/%v/%v", task.FromSource, task.FromOwner, task.FromName, task.ToSource, task.ToOwner, task.ToName)
				h.performMigration(ctx, task)
			}
		}()
	}
}

func (h *Handler) performMigration(ctx context.Context, task *migration.Task) {
	// sequence of stages for migration of repository
	stages := []migration.Stage{
		migration.Releases(h.databaseClient.MigrateReleases),
		migration.ReleaseLogs(h.databaseClient.MigrateReleaseLogs),
		migration.ReleaseLogObjects(h.migrateReleaseLogObjects),
		migration.Builds(h.databaseClient.MigrateBuilds),
		migration.BuildLogs(h.databaseClient.MigrateBuildLogs),
		migration.BuildLogObjects(h.migrateBuildLogObjects),
		migration.BuildVersions(h.databaseClient.MigrateBuildVersions),
	}
	if task.CallbackURL != nil {
		// if a callback url is provided, add a callback stage
		stages = append(stages, migration.Callback())
	}
	// always add a completed stage to the end of the sequence
	stages = append(stages, migration.Completed())
	var err error
	if task.LastStep != migration.StepWaiting {
		// if migration was restarted, skip stages that were already completed
		for i, stage := range stages {
			// release
			if task.LastStep < stage.Success() {
				stages = stages[i:]
				break
			}
		}
	}
	var failed bool
	for _, stage := range stages {
		_, failed = stage.Execute(ctx, task)
		err = h.databaseClient.UpdateMigration(ctx, task)
		if err != nil {
			log.Error().Err(err).Msg("Error updating migration status in database")
			return
		}
		if failed {
			return
		}
	}
}

func (h *Handler) migrateReleaseLogObjects(ctx context.Context, task *migration.Task) ([]migration.Change, error) {
	return h.migrateLogObjects(ctx, migration.BuildLog, task)
}

func (h *Handler) migrateBuildLogObjects(ctx context.Context, task *migration.Task) ([]migration.Change, error) {
	return h.migrateLogObjects(ctx, migration.ReleaseLog, task)
}

func (h *Handler) migrateLogObjects(ctx context.Context, logType migration.LogType, task *migration.Task) ([]migration.Change, error) {
	md := metadata.New(map[string]string{"messageuniqueid": task.ID})
	ctx = metadata.NewOutgoingContext(ctx, md)
	stream, err := h.gcsMigratorClient.Migrate(ctx)
	if err != nil {
		log.Fatal().Msgf("gcsMigratorClient.Migrate failed: %v", err)
	}
	// fetch changes from database since in case of migration restart
	var changes []migration.Change
	if logType == migration.BuildLog {
		changes, err = h.databaseClient.GetMigrationBuildLogsChanges(ctx, task)
	} else if logType == migration.ReleaseLog {
		changes, err = h.databaseClient.GetMigrationReleaseLogsChanges(ctx, task)
	} else {
		return nil, fmt.Errorf("invalid logType: %s", logType)
	}
	if err != nil {
		return nil, err
	}
	requests := make(map[int64]migration.Change, 0)
	errs := ""
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
			errs += fmt.Sprintf("\n\t%d - %d: %s", change.FromID, change.ToID, err.Error())
			log.Error().Msgf("gcsMigratorClient.Migrate: stream.Send(%v) failed: %v", req, err)
		}
		requests[int64(index)] = change
	}
	var response *migrationpb.Response
	response, err = stream.CloseAndRecv()
	if err != nil {
		return nil, fmt.Errorf("gcsMigratorClient.Migrate: stream.CloseAndRecv() failed: %w", err)
	}
	for _, m := range response.Migrations {
		req := requests[m.Id]
		if m.Error != nil {
			data, err := json.Marshal(m.Error)
			if err != nil {
				return nil, fmt.Errorf("error marshalling Error field from response: %w", err)
			}
			errs += fmt.Sprintf("\n\t%d - %d: %s", req.FromID, req.ToID, data)
		}
	}
	if errs != "" {
		return nil, fmt.Errorf("log_object migration errors: %s", errs)
	}
	return nil, nil
}
