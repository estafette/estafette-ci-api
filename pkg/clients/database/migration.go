package database

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/lib/pq"
	"github.com/opentracing/opentracing-go"
	"github.com/rs/zerolog/log"
	"k8s.io/utils/pointer"

	"github.com/estafette/estafette-ci-api/pkg/api"
	"github.com/estafette/estafette-ci-api/pkg/clients/database/queries"
	"github.com/estafette/migration"
)

var (
	isMigration         = struct{}{}
	ErrPipelineNotFound = errors.New("pipeline not found")
	ErrMigrationExists  = errors.New("migration already exists")
)

type MigrationDatabaseApi interface {
	QueueMigration(ctx context.Context, task *migration.Task) (*migration.Task, error)
	GetMigrationStatus(ctx context.Context, taskID string) (*migration.Task, error)
	PickMigration(ctx context.Context, maxTasks int64) ([]*migration.Task, error)
	UpdateMigration(ctx context.Context, task *migration.Task) error
	MigrateBuilds(ctx context.Context, task *migration.Task) error
	MigrateBuildLogs(ctx context.Context, task *migration.Task) error
	MigrateBuildVersions(ctx context.Context, task *migration.Task) error
	MigrateReleases(ctx context.Context, task *migration.Task) error
	MigrateReleaseLogs(ctx context.Context, task *migration.Task) error
	MigrateComputedTables(ctx context.Context, task *migration.Task) error
	GetMigratedBuildLogs(ctx context.Context, task *migration.Task) ([]migration.Change, error)
	GetMigratedReleaseLogs(ctx context.Context, task *migration.Task) ([]migration.Change, error)
	RollbackMigration(ctx context.Context, task *migration.Task) (*migration.Changes, error)
}

func _CloseRows(rows *sql.Rows) {
	err := rows.Close()
	if err != nil {
		log.Error().Err(err).Msg("failed to close rows")
	}
}

func (c *client) QueueMigration(ctx context.Context, taskToQueue *migration.Task) (task *migration.Task, err error) {
	query, args := queries.CheckExistingMigration(taskToQueue.SqlArgs()...)
	row := c.databaseConnection.QueryRowContext(ctx, query, args...)
	var pipelineExists bool
	var existingTaskID string
	if err = row.Scan(&pipelineExists, &existingTaskID); err != nil {
		return nil, fmt.Errorf("failed to scan existing migration for: %s: %w", taskToQueue.FromFQN(), err)
	}
	if !pipelineExists {
		return nil, fmt.Errorf("failed to queue migration for %s repository: %w", taskToQueue.FromFQN(), ErrPipelineNotFound)
	}
	if existingTaskID != "" {
		if taskToQueue.ID != existingTaskID {
			return nil, fmt.Errorf("%w for %s repository with ID: %s", ErrMigrationExists, taskToQueue.FromFQN(), existingTaskID)
		}
	}
	if taskToQueue.ID == "" {
		taskToQueue.ID = uuid.New().String()
	}
	var transaction *sql.Tx
	transaction, err = c.databaseConnection.BeginTx(ctx, nil)
	defer func(tx *sql.Tx) {
		if err != nil {
			if err := tx.Rollback(); err != nil {
				log.Error().Err(err).Msg("failed to rollback transaction for queuing migration tasks")
			}
			return
		}
		err = tx.Commit()
	}(transaction)
	if existingTaskID != "" {
		taskToQueue.ID = existingTaskID
		log.Info().Str("taskID", existingTaskID).Msg("found existing migration task")
	} else {
		query, args = queries.SetMigrationIdForPipeline(taskToQueue.SqlArgs()...)
		_, err = transaction.ExecContext(ctx, query, args...)
		if err != nil {
			return nil, fmt.Errorf("failed to set migration id for pipeline: %s: %w", taskToQueue.FromFQN(), err)
		}
	}
	query, args = queries.QueueMigration(taskToQueue.SqlArgs()...)
	row = transaction.QueryRowContext(ctx, query, args...)
	var totalDuration int64
	task = &migration.Task{}
	err = row.Scan(&task.ID, &task.Status, &task.LastStep, &task.Builds, &task.Releases, &totalDuration, &task.FromSource, &task.FromOwner, &task.FromName, &task.ToSource, &task.ToOwner, &task.ToName, &task.CallbackURL, &task.ErrorDetails, &task.QueuedAt, &task.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("failed to scan inserted/updated migration task %s: %w", task.ID, err)
	}
	task.TotalDuration = time.Duration(totalDuration)
	return
}

func (c *client) GetMigrationStatus(ctx context.Context, taskID string) (*migration.Task, error) {
	if taskID == "" {
		return nil, fmt.Errorf("taskID is required to get migration status")
	}
	query, args := queries.GetMigrationStatus(sql.NamedArg{Name: "id", Value: taskID})
	row := c.databaseConnection.QueryRowContext(ctx, query, args...)
	var task migration.Task
	var totalDuration int64
	err := row.Scan(&task.ID, &task.Status, &task.LastStep, &task.Builds, &task.Releases, &totalDuration, &task.FromSource, &task.FromOwner, &task.FromName, &task.ToSource, &task.ToOwner, &task.ToName, &task.CallbackURL, &task.ErrorDetails, &task.QueuedAt, &task.UpdatedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get migration status for taskID: %s : %w", taskID, err)
	}
	task.TotalDuration = time.Duration(totalDuration)
	return &task, nil
}

func (c *client) PickMigration(ctx context.Context, maxTasks int64) (tasks []*migration.Task, err error) {
	var transaction *sql.Tx
	transaction, err = c.databaseConnection.BeginTx(ctx, nil)
	defer func(tx *sql.Tx) {
		if err != nil {
			if err := tx.Rollback(); err != nil {
				log.Error().Err(err).Msg("failed to rollback transaction for picking up migration tasks")
			}
			return
		}
		err = tx.Commit()
	}(transaction)
	if err != nil {
		return nil, fmt.Errorf("failed to start transcation for picking up migration tasks: %w", err)
	}
	query, args := queries.PickMigration(sql.NamedArg{Name: "maxTasks", Value: maxTasks})
	var rows *sql.Rows
	rows, err = transaction.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to pick up migration tasks from database: %w", err)
	}
	var totalDuration int64
	pickedRepos := make([]string, 0)
	for rows.Next() {
		var task migration.Task
		err = rows.Scan(&task.ID, &task.Status, &task.LastStep, &task.Builds, &task.Releases, &totalDuration, &task.FromSource, &task.FromOwner, &task.FromName, &task.ToSource, &task.ToOwner, &task.ToName, &task.CallbackURL, &task.ErrorDetails, &task.QueuedAt, &task.UpdatedAt)
		if err != nil {
			return nil, fmt.Errorf("failed to scan migration task row from database: %w", err)
		}
		task.TotalDuration = time.Duration(totalDuration)
		tasks = append(tasks, &task)
		pickedRepos = append(pickedRepos, task.FromFQN())
	}
	if len(pickedRepos) > 0 {
		query, args = queries.MarkRepositoryArchived(sql.NamedArg{Name: "pickedRepos", Value: pq.Array(pickedRepos)})
		_, err = transaction.ExecContext(ctx, query, args...)
		if err != nil {
			return nil, fmt.Errorf("failed to mark repositories as archived: %w", err)
		}
	}
	return tasks, nil
}

func (c *client) UpdateMigration(ctx context.Context, task *migration.Task) error {
	// add pod details to error details
	if task.ErrorDetails != nil && *task.ErrorDetails != "" {
		HOSTNAME := "unknown"
		if os.Getenv("HOSTNAME") != "" {
			HOSTNAME = os.Getenv("HOSTNAME")
		}
		task.ErrorDetails = pointer.String(
			strings.TrimSpace(
				fmt.Sprintf("//%s start: %s\n%s\n//%s end", HOSTNAME, time.Now().Format(time.RFC3339), *task.ErrorDetails, HOSTNAME),
			),
		)
	}
	query, args := queries.UpdateMigration(task.SqlArgs()...)
	_, err := c.databaseConnection.ExecContext(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("failed to update migration task %s: %w", task.ID, err)
	}
	return nil
}

func (c *client) MigrateReleases(ctx context.Context, task *migration.Task) error {
	query, args := queries.MigrateReleases(task.SqlArgs()...)
	_, err := c.databaseConnection.ExecContext(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("failed to migrate releases for repository %s: %w", task.FromFQN(), err)
	}
	query, args = queries.GetMigratedReleasesCount(task.SqlArgs()...)
	row := c.databaseConnection.QueryRowContext(ctx, query, args...)
	err = row.Scan(&task.Releases)
	if err != nil {
		return fmt.Errorf("failed to get (migrated) releases count for repository %s: %w", task.ToFQN(), err)
	}
	return nil
}

func (c *client) MigrateReleaseLogs(ctx context.Context, task *migration.Task) error {
	query, args := queries.MigrateReleaseLogs(task.SqlArgs()...)
	_, err := c.databaseConnection.ExecContext(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("failed to migrate release_logs for repository %s: %w", task.ToFQN(), err)
	}
	return nil
}

func (c *client) MigrateBuilds(ctx context.Context, task *migration.Task) error {
	query, args := queries.MigrateBuilds(task.SqlArgs()...)
	_, err := c.databaseConnection.ExecContext(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("failed to migrate builds for repository %s: %w", task.FromFQN(), err)
	}
	query, args = queries.GetMigratedBuildsCount(task.SqlArgs()...)
	row := c.databaseConnection.QueryRowContext(ctx, query, args...)
	err = row.Scan(&task.Builds)
	if err != nil {
		return fmt.Errorf("failed to get (migrated) releases count for repository %s: %w", task.FromFQN(), err)
	}
	return nil
}

func (c *client) MigrateBuildLogs(ctx context.Context, task *migration.Task) error {
	query, args := queries.MigrateBuildLogs(task.SqlArgs()...)
	_, err := c.databaseConnection.ExecContext(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("failed to migrate build_logs for repository %s: %w", task.FromFQN(), err)
	}
	return nil
}

func (c *client) MigrateBuildVersions(ctx context.Context, task *migration.Task) error {
	query, args := queries.MigrateBuildVersions(task.SqlArgs()...)
	_, err := c.databaseConnection.ExecContext(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("failed to migrate build versions for repository %s: %w", task.FromFQN(), err)
	}
	return nil
}

func (c *client) MigrateComputedTables(ctx context.Context, task *migration.Task) error {
	ctx = context.WithValue(ctx, isMigration, true)
	err := c.UpdateComputedTables(ctx, task.ToSource, task.ToOwner, task.ToName)
	if err != nil {
		return fmt.Errorf("failed to migrate compueted tables for repository %s: %w", task.FromFQN(), err)
	}
	return nil
}

func (c *client) GetMigratedBuildLogs(ctx context.Context, task *migration.Task) ([]migration.Change, error) {
	query, args := queries.GetMigratedBuildLogs(task.SqlArgs()...)
	rows, err := c.databaseConnection.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed query to get migration build log changes for repository %s: %w", task.FromFQN(), err)
	}
	changes := make([]migration.Change, 0)
	for rows.Next() {
		var result migration.Change
		if err = rows.Scan(&result.FromID, &result.ToID); err != nil {
			return nil, fmt.Errorf("failed to get migration build log changes rows for repository %s: %w", task.FromFQN(), err)
		}
		changes = append(changes, result)
	}
	_CloseRows(rows)
	return changes, nil
}

func (c *client) GetMigratedReleaseLogs(ctx context.Context, task *migration.Task) ([]migration.Change, error) {
	query, args := queries.GetMigratedReleaseLogs(task.SqlArgs()...)
	rows, err := c.databaseConnection.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed query to get migration release log changes for repository %s: %w", task.FromFQN(), err)
	}
	changes := make([]migration.Change, 0)
	for rows.Next() {
		var result migration.Change
		if err = rows.Scan(&result.FromID, &result.ToID); err != nil {
			return nil, fmt.Errorf("failed to get migration release log changes rows for repository %s: %w", task.FromFQN(), err)
		}
		changes = append(changes, result)
	}
	_CloseRows(rows)
	return changes, nil
}

func (c *client) RollbackMigration(ctx context.Context, task *migration.Task) (cn *migration.Changes, err error) {
	changes := &migration.Changes{}
	tx, err := c.databaseConnection.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to start rollback migration transaction for repository %s: %w", task.FromFQN(), err)
	}
	defer func(tx *sql.Tx) {
		if err != nil {
			if err := tx.Rollback(); err != nil {
				log.Error().Err(err).Msg("failed to rollback transaction for queuing migration tasks")
			}
			return
		}
		cn = changes
		err = tx.Commit()
	}(tx)
	query, args := queries.RollbackReleaseLogs(task.SqlArgs()...)
	if err = tx.QueryRowContext(ctx, query, args...).Scan(&changes.ReleaseLogs); err != nil {
		return nil, fmt.Errorf("failed to get rolled back release logs for repository %s: %w", task.FromFQN(), err)
	}
	query, args = queries.RollbackReleases(task.SqlArgs()...)
	if err = tx.QueryRowContext(ctx, query, args...).Scan(&changes.Releases); err != nil {
		return nil, fmt.Errorf("failed to get rolled back releases for repository %s: %w", task.FromFQN(), err)
	}
	query, args = queries.RollbackBuildLogs(task.SqlArgs()...)
	if err = tx.QueryRowContext(ctx, query, args...).Scan(&changes.BuildLogs); err != nil {
		return nil, fmt.Errorf("failed to get rolled back build logs for repository %s: %w", task.FromFQN(), err)
	}
	query, args = queries.RollbackBuildVersions(task.SqlArgs()...)
	if err = tx.QueryRowContext(ctx, query, args...).Scan(&changes.BuildVersions); err != nil {
		return nil, fmt.Errorf("failed to get rolled back build versions for repository %s: %w", task.FromFQN(), err)
	}
	query, args = queries.RollbackBuilds(task.SqlArgs()...)
	if err = tx.QueryRowContext(ctx, query, args...).Scan(&changes.Builds); err != nil {
		return nil, fmt.Errorf("failed to get rolled back builds for repository %s: %w", task.FromFQN(), err)
	}
	query, args = queries.RollbackComputedPipelines(task.SqlArgs()...)
	if _, err = tx.ExecContext(ctx, query, args...); err != nil {
		return nil, fmt.Errorf("failed to get rolled back computed pipelines for repository %s: %w", task.FromFQN(), err)
	}
	query, args = queries.RollbackComputedReleases(task.SqlArgs()...)
	if _, err = tx.ExecContext(ctx, query, args...); err != nil {
		return nil, fmt.Errorf("failed to get rolled back computed releases for repository %s: %w", task.FromFQN(), err)
	}
	query, args = queries.RollbackMigrationTaskQueue(task.SqlArgs()...)
	if _, err = tx.ExecContext(ctx, query, args...); err != nil {
		return nil, fmt.Errorf("failed to get rolled back migration tasks for repository %s: %w", task.FromFQN(), err)
	}
	return
}

// logging

func (c *loggingClient) GetMigrationStatus(ctx context.Context, taskID string) (task *migration.Task, err error) {
	defer func() { api.HandleLogError(c.prefix, "Client", "GetMigrationStatus", err) }()
	return c.Client.GetMigrationStatus(ctx, taskID)
}
func (c *loggingClient) QueueMigration(ctx context.Context, task *migration.Task) (t *migration.Task, err error) {
	defer func() { api.HandleLogError(c.prefix, "Client", "QueueMigration", err) }()
	return c.Client.QueueMigration(ctx, task)
}
func (c *loggingClient) PickMigration(ctx context.Context, maxTasks int64) (tasks []*migration.Task, err error) {
	defer func() { api.HandleLogError(c.prefix, "Client", "PickMigration", err) }()
	return c.Client.PickMigration(ctx, maxTasks)
}
func (c *loggingClient) UpdateMigration(ctx context.Context, task *migration.Task) (err error) {
	defer func() { api.HandleLogError(c.prefix, "Client", "UpdateMigration", err) }()
	return c.Client.UpdateMigration(ctx, task)
}
func (c *loggingClient) MigrateReleases(ctx context.Context, task *migration.Task) (err error) {
	defer func() { api.HandleLogError(c.prefix, "Client", "MigrateReleases", err) }()
	return c.Client.MigrateReleases(ctx, task)
}
func (c *loggingClient) MigrateReleaseLogs(ctx context.Context, task *migration.Task) (err error) {
	defer func() { api.HandleLogError(c.prefix, "Client", "MigrateReleaseLogs", err) }()
	return c.Client.MigrateReleaseLogs(ctx, task)
}
func (c *loggingClient) MigrateBuilds(ctx context.Context, task *migration.Task) (err error) {
	defer func() { api.HandleLogError(c.prefix, "Client", "MigrateBuilds", err) }()
	return c.Client.MigrateBuilds(ctx, task)
}
func (c *loggingClient) MigrateBuildLogs(ctx context.Context, task *migration.Task) (err error) {
	defer func() { api.HandleLogError(c.prefix, "Client", "MigrateBuildLogs", err) }()
	return c.Client.MigrateBuildLogs(ctx, task)
}
func (c *loggingClient) MigrateBuildVersions(ctx context.Context, task *migration.Task) (err error) {
	defer func() { api.HandleLogError(c.prefix, "Client", "MigrateBuildVersions", err) }()
	return c.Client.MigrateBuildVersions(ctx, task)
}
func (c *loggingClient) MigrateComputedTables(ctx context.Context, task *migration.Task) (err error) {
	defer func() { api.HandleLogError(c.prefix, "Client", "MigrateComputedTables", err) }()
	return c.Client.MigrateComputedTables(ctx, task)
}
func (c *loggingClient) GetMigratedBuildLogs(ctx context.Context, task *migration.Task) (changes []migration.Change, err error) {
	defer func() { api.HandleLogError(c.prefix, "Client", "GetMigratedBuildLogs", err) }()
	return c.Client.GetMigratedBuildLogs(ctx, task)
}
func (c *loggingClient) GetMigratedReleaseLogs(ctx context.Context, task *migration.Task) (changes []migration.Change, err error) {
	defer func() { api.HandleLogError(c.prefix, "Client", "GetMigratedReleaseLogs", err) }()
	return c.Client.GetMigratedReleaseLogs(ctx, task)
}
func (c *loggingClient) RollbackMigration(ctx context.Context, task *migration.Task) (changes *migration.Changes, err error) {
	defer func() { api.HandleLogError(c.prefix, "Client", "RollbackMigration", err) }()
	return c.Client.RollbackMigration(ctx, task)
}

//metrics

func (c *metricsClient) GetMigrationStatus(ctx context.Context, taskID string) (task *migration.Task, err error) {
	api.UpdateMetrics(c.requestCount, c.requestLatency, "GetMigrationStatus", time.Now())
	return c.Client.GetMigrationStatus(ctx, taskID)
}
func (c *metricsClient) QueueMigration(ctx context.Context, task *migration.Task) (t *migration.Task, err error) {
	api.UpdateMetrics(c.requestCount, c.requestLatency, "QueueMigration", time.Now())
	return c.Client.QueueMigration(ctx, task)
}
func (c *metricsClient) PickMigration(ctx context.Context, maxTasks int64) (tasks []*migration.Task, err error) {
	api.UpdateMetrics(c.requestCount, c.requestLatency, "PickMigration", time.Now())
	return c.Client.PickMigration(ctx, maxTasks)
}
func (c *metricsClient) UpdateMigration(ctx context.Context, task *migration.Task) (err error) {
	api.UpdateMetrics(c.requestCount, c.requestLatency, "UpdateMigration", time.Now())
	return c.Client.UpdateMigration(ctx, task)
}
func (c *metricsClient) MigrateReleases(ctx context.Context, task *migration.Task) (err error) {
	api.UpdateMetrics(c.requestCount, c.requestLatency, "MigrateReleases", time.Now())
	return c.Client.MigrateReleases(ctx, task)
}
func (c *metricsClient) MigrateReleaseLogs(ctx context.Context, task *migration.Task) (err error) {
	api.UpdateMetrics(c.requestCount, c.requestLatency, "MigrateReleaseLogs", time.Now())
	return c.Client.MigrateReleaseLogs(ctx, task)
}
func (c *metricsClient) MigrateBuilds(ctx context.Context, task *migration.Task) (err error) {
	api.UpdateMetrics(c.requestCount, c.requestLatency, "MigrateBuilds", time.Now())
	return c.Client.MigrateBuilds(ctx, task)
}
func (c *metricsClient) MigrateBuildLogs(ctx context.Context, task *migration.Task) (err error) {
	api.UpdateMetrics(c.requestCount, c.requestLatency, "MigrateBuildLogs", time.Now())
	return c.Client.MigrateBuildLogs(ctx, task)
}
func (c *metricsClient) MigrateBuildVersions(ctx context.Context, task *migration.Task) (err error) {
	api.UpdateMetrics(c.requestCount, c.requestLatency, "MigrateBuildVersions", time.Now())
	return c.Client.MigrateBuildVersions(ctx, task)
}
func (c *metricsClient) MigrateComputedTables(ctx context.Context, task *migration.Task) (err error) {
	api.UpdateMetrics(c.requestCount, c.requestLatency, "MigrateComputedTables", time.Now())
	return c.Client.MigrateComputedTables(ctx, task)
}
func (c *metricsClient) GetMigratedBuildLogs(ctx context.Context, task *migration.Task) (changes []migration.Change, err error) {
	api.UpdateMetrics(c.requestCount, c.requestLatency, "GetMigratedBuildLogs", time.Now())
	return c.Client.GetMigratedBuildLogs(ctx, task)
}
func (c *metricsClient) GetMigratedReleaseLogs(ctx context.Context, task *migration.Task) (changes []migration.Change, err error) {
	api.UpdateMetrics(c.requestCount, c.requestLatency, "GetMigratedReleaseLogs", time.Now())
	return c.Client.GetMigratedReleaseLogs(ctx, task)
}
func (c *metricsClient) RollbackMigration(ctx context.Context, task *migration.Task) (changes *migration.Changes, err error) {
	api.UpdateMetrics(c.requestCount, c.requestLatency, "RollbackMigration", time.Now())
	return c.Client.RollbackMigration(ctx, task)
}

// logging

func (c *tracingClient) GetMigrationStatus(ctx context.Context, taskID string) (task *migration.Task, err error) {
	span := opentracing.StartSpan(api.GetSpanName(c.prefix, "GetMigrationStatus"))
	defer func() { api.FinishSpanWithError(span, err) }()
	return c.Client.GetMigrationStatus(ctx, taskID)
}
func (c *tracingClient) QueueMigration(ctx context.Context, task *migration.Task) (t *migration.Task, err error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, api.GetSpanName(c.prefix, "QueueMigration"))
	defer func() { api.FinishSpanWithError(span, err) }()
	return c.Client.QueueMigration(ctx, task)
}
func (c *tracingClient) PickMigration(ctx context.Context, maxTasks int64) (tasks []*migration.Task, err error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, api.GetSpanName(c.prefix, "PickMigration"))
	defer func() { api.FinishSpanWithError(span, err) }()
	return c.Client.PickMigration(ctx, maxTasks)
}
func (c *tracingClient) UpdateMigration(ctx context.Context, task *migration.Task) (err error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, api.GetSpanName(c.prefix, "UpdateMigration"))
	defer func() { api.FinishSpanWithError(span, err) }()
	return c.Client.UpdateMigration(ctx, task)
}
func (c *tracingClient) MigrateReleases(ctx context.Context, task *migration.Task) (err error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, api.GetSpanName(c.prefix, "MigrateReleases"))
	defer func() { api.FinishSpanWithError(span, err) }()
	return c.Client.MigrateReleases(ctx, task)
}
func (c *tracingClient) MigrateReleaseLogs(ctx context.Context, task *migration.Task) (err error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, api.GetSpanName(c.prefix, "MigrateReleaseLogs"))
	defer func() { api.FinishSpanWithError(span, err) }()
	return c.Client.MigrateReleaseLogs(ctx, task)
}
func (c *tracingClient) MigrateBuilds(ctx context.Context, task *migration.Task) (err error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, api.GetSpanName(c.prefix, "MigrateBuilds"))
	defer func() { api.FinishSpanWithError(span, err) }()
	return c.Client.MigrateBuilds(ctx, task)
}
func (c *tracingClient) MigrateBuildLogs(ctx context.Context, task *migration.Task) (err error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, api.GetSpanName(c.prefix, "MigrateBuildLogs"))
	defer func() { api.FinishSpanWithError(span, err) }()
	return c.Client.MigrateBuildLogs(ctx, task)
}
func (c *tracingClient) MigrateBuildVersions(ctx context.Context, task *migration.Task) (err error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, api.GetSpanName(c.prefix, "MigrateBuildVersions"))
	defer func() { api.FinishSpanWithError(span, err) }()
	return c.Client.MigrateBuildVersions(ctx, task)
}
func (c *tracingClient) MigrateComputedTables(ctx context.Context, task *migration.Task) (err error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, api.GetSpanName(c.prefix, "MigrateComputedTables"))
	defer func() { api.FinishSpanWithError(span, err) }()
	return c.Client.MigrateComputedTables(ctx, task)
}
func (c *tracingClient) GetMigratedBuildLogs(ctx context.Context, task *migration.Task) (changes []migration.Change, err error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, api.GetSpanName(c.prefix, "GetMigratedBuildLogs"))
	defer func() { api.FinishSpanWithError(span, err) }()
	return c.Client.GetMigratedBuildLogs(ctx, task)
}
func (c *tracingClient) GetMigratedReleaseLogs(ctx context.Context, task *migration.Task) (changes []migration.Change, err error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, api.GetSpanName(c.prefix, "GetMigratedReleaseLogs"))
	defer func() { api.FinishSpanWithError(span, err) }()
	return c.Client.GetMigratedReleaseLogs(ctx, task)
}
func (c *tracingClient) RollbackMigration(ctx context.Context, task *migration.Task) (changes *migration.Changes, err error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, api.GetSpanName(c.prefix, "RollbackMigration"))
	defer func() { api.FinishSpanWithError(span, err) }()
	return c.Client.RollbackMigration(ctx, task)
}
