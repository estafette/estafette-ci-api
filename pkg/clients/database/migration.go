package database

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"k8s.io/utils/ptr"
	"os"
	"time"

	"github.com/estafette/estafette-ci-api/pkg/api"
	"github.com/estafette/estafette-ci-api/pkg/clients/database/queries"
	contracts "github.com/estafette/estafette-ci-contracts"
	"github.com/estafette/migration"
	"github.com/google/uuid"
	"github.com/lib/pq"
	"github.com/opentracing/opentracing-go"
	"github.com/rs/zerolog/log"
)

var (
	isMigration          = struct{}{}
	ErrPipelineNotFound  = errors.New("pipeline not found")
	ErrMigrationNotFound = errors.New("migration not found")
	ErrMigrationExists   = errors.New("migration already exists")
)

type MigrationDatabaseApi interface {
	GetAllMigrations(ctx context.Context) ([]*migration.Task, error)
	GetMigratedBuild(ctx context.Context, buildID string) (*contracts.Build, error)
	GetMigratedRelease(ctx context.Context, buildID string) (*contracts.Release, error)
	GetMigratedBuildLogs(ctx context.Context, task *migration.Task) ([]migration.Change, error)
	GetMigratedReleaseLogs(ctx context.Context, task *migration.Task) ([]migration.Change, error)
	GetMigrationByFromRepo(ctx context.Context, fromSource, fromOwner, fromName string) (*migration.Task, error)
	GetMigrationByToRepo(ctx context.Context, toSource, toOwner, toName string) (*migration.Task, error)
	GetMigrationByID(ctx context.Context, taskID string) (*migration.Task, error)
	MigrateBuildLogs(ctx context.Context, task *migration.Task) error
	MigrateBuildVersions(ctx context.Context, task *migration.Task) error
	MigrateBuilds(ctx context.Context, task *migration.Task) error
	MigrateComputedTables(ctx context.Context, task *migration.Task) error
	MigrateReleaseLogs(ctx context.Context, task *migration.Task) error
	MigrateReleases(ctx context.Context, task *migration.Task) error
	PickMigration(ctx context.Context, maxTasks int64) ([]*migration.Task, error)
	QueueMigration(ctx context.Context, task *migration.Task) (*migration.Task, error)
	RollbackMigration(ctx context.Context, task *migration.Task) (*migration.Changes, error)
	UpdateMigration(ctx context.Context, task *migration.Task) error
	SetPipelineArchival(ctx context.Context, source, owner, name string, archived bool) error
}

func _CloseRows(rows *sql.Rows) {
	err := rows.Close()
	if err != nil {
		log.Error().Err(err).Msg("failed to close rows")
	}
}

func (c *client) GetMigratedBuild(ctx context.Context, buildID string) (*contracts.Build, error) {
	query, args := queries.GetMigratedBuild(sql.NamedArg{Name: "id", Value: buildID})
	row := c.databaseConnection.QueryRowContext(ctx, query, args...)
	build, err := c.scanBuild(ctx, row, true, false)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get migrated build %v: %w", buildID, err)
	}
	return build, nil
}

func (c *client) GetMigratedRelease(ctx context.Context, buildID string) (*contracts.Release, error) {
	query, args := queries.GetMigratedRelease(sql.NamedArg{Name: "id", Value: buildID})
	row := c.databaseConnection.QueryRowContext(ctx, query, args...)
	release, err := c.scanRelease(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get migrated release %v: %w", buildID, err)
	}
	return release, nil
}

func (c *client) GetAllMigrations(ctx context.Context) ([]*migration.Task, error) {
	query, args := queries.GetAllMigrations()
	rows, err := c.databaseConnection.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to get all migrations: %w", err)
	}
	tasks := make([]*migration.Task, 0)
	defer _CloseRows(rows)
	for rows.Next() {
		var task migration.Task
		var totalDuration int64
		err = rows.Scan(&task.ID, &task.Status, &task.LastStep, &task.FromSource, &task.FromOwner, &task.FromName, &task.ToSource, &task.ToOwner, &task.ToName, &task.ErrorDetails, &task.QueuedAt, &task.UpdatedAt)
		if err != nil {
			return nil, fmt.Errorf("failed to scan migration: %w", err)
		}
		task.TotalDuration = time.Duration(totalDuration)
		tasks = append(tasks, &task)
	}
	return tasks, nil
}

func (c *client) GetMigratedBuildLogs(ctx context.Context, task *migration.Task) ([]migration.Change, error) {
	query, args := queries.GetMigratedBuildLogs(task.SqlArgs()...)
	rows, err := c.databaseConnection.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed query to get migration build log changes for repository %s: %w", task.FromFQN(), err)
	}
	defer _CloseRows(rows)
	changes := make([]migration.Change, 0)
	for rows.Next() {
		var result migration.Change
		if err = rows.Scan(&result.FromID, &result.ToID); err != nil {
			return nil, fmt.Errorf("failed to get migration build log changes rows for repository %s: %w", task.FromFQN(), err)
		}
		changes = append(changes, result)
	}
	return changes, nil
}

func (c *client) GetMigratedReleaseLogs(ctx context.Context, task *migration.Task) ([]migration.Change, error) {
	query, args := queries.GetMigratedReleaseLogs(task.SqlArgs()...)
	rows, err := c.databaseConnection.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed query to get migration release log changes for repository %s: %w", task.FromFQN(), err)
	}
	defer _CloseRows(rows)
	changes := make([]migration.Change, 0)
	for rows.Next() {
		var result migration.Change
		if err = rows.Scan(&result.FromID, &result.ToID); err != nil {
			return nil, fmt.Errorf("failed to get migration release log changes rows for repository %s: %w", task.FromFQN(), err)
		}
		changes = append(changes, result)
	}
	return changes, nil
}

func (c *client) GetMigrationByFromRepo(ctx context.Context, fromSource, fromOwner, fromName string) (*migration.Task, error) {
	query, args := queries.GetMigrationByFromRepo(sql.NamedArg{Name: "fromSource", Value: fromSource}, sql.NamedArg{Name: "fromOwner", Value: fromOwner}, sql.NamedArg{Name: "fromName", Value: fromName})
	row := c.databaseConnection.QueryRowContext(ctx, query, args...)
	var task migration.Task
	var totalDuration int64
	err := row.Scan(&task.ID, &task.Status, &task.LastStep, &task.Builds, &task.Releases, &totalDuration, &task.FromSource, &task.FromOwner, &task.FromName, &task.ToSource, &task.ToOwner, &task.ToName, &task.CallbackURL, &task.ErrorDetails, &task.QueuedAt, &task.UpdatedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get migration status for repository %s/%s/%s: %w", fromSource, fromOwner, fromName, err)
	}
	task.TotalDuration = time.Duration(totalDuration)
	return &task, nil
}

func (c *client) GetMigrationByToRepo(ctx context.Context, toSource, toOwner, toName string) (*migration.Task, error) {
	query, args := queries.GetMigrationByToRepo(sql.NamedArg{Name: "toSource", Value: toSource}, sql.NamedArg{Name: "toOwner", Value: toOwner}, sql.NamedArg{Name: "toName", Value: toName})
	row := c.databaseConnection.QueryRowContext(ctx, query, args...)
	var task migration.Task
	var totalDuration int64
	err := row.Scan(&task.ID, &task.Status, &task.LastStep, &task.Builds, &task.Releases, &totalDuration, &task.FromSource, &task.FromOwner, &task.FromName, &task.ToSource, &task.ToOwner, &task.ToName, &task.CallbackURL, &task.ErrorDetails, &task.QueuedAt, &task.UpdatedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get migration status for repository %s/%s/%s: %w", toSource, toOwner, toName, err)
	}
	task.TotalDuration = time.Duration(totalDuration)
	return &task, nil
}

func (c *client) GetMigrationByID(ctx context.Context, taskID string) (*migration.Task, error) {
	if taskID == "" {
		return nil, fmt.Errorf("taskID is required to get migration status")
	}
	query, args := queries.GetMigrationByID(sql.NamedArg{Name: "id", Value: taskID})
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

func (c *client) MigrateBuilds(ctx context.Context, task *migration.Task) error {
	// Get Min and Max build date from the source repository.
	var minBuildDate, maxBuildDate time.Time
	minQuery, minMaxArgs := queries.GetBuildsToMigrateMinMaxDateCreated(task.SqlArgs()...)
	row := c.databaseConnection.QueryRowContext(ctx, minQuery, minMaxArgs...)

	err := row.Scan(&minBuildDate, &maxBuildDate)
	if err != nil {
		return fmt.Errorf("failed to get min and max build date for repository %s: %w", task.FromFQN(), err)
	}

	defaultTimestamp := time.Date(1970, time.January, 1, 0, 0, 0, 0, time.UTC)
	if minBuildDate.Equal(defaultTimestamp) || maxBuildDate.Equal(defaultTimestamp) {
		return nil
	}

	// Calculate the initial start and end dates for the first month.
	startMonth := minBuildDate
	endMonth := minBuildDate.AddDate(0, 1, 0)

	for {
		// Check if the end date of the month exceeds the max build date.
		if endMonth.After(maxBuildDate) {
			endMonth = maxBuildDate
		}

		// Construct the base SQL query for the migration with date range conditions.
		baseMigrateQuery, baseMigrateArgs := queries.MigrateBuilds(task.SqlArgs()...)
		baseMigrateQuery += " AND inserted_at >= $7 AND inserted_at < $8"

		args := []interface{}{startMonth, endMonth}
		baseMigrateArgs = append(baseMigrateArgs, args...)

		// Execute the batched migration query.
		_, err = c.databaseConnection.ExecContext(ctx, baseMigrateQuery, baseMigrateArgs...)
		if err != nil {
			return fmt.Errorf("failed to migrate builds for repository %s (month %s): %w", task.FromFQN(), startMonth.Format("2006-01"), err)
		}

		// Move to the next month.
		startMonth = startMonth.AddDate(0, 1, 0)
		endMonth = startMonth.AddDate(0, 1, 0)

		// Check if we have exceeded the max build date.
		if startMonth.After(maxBuildDate) {
			break
		}
	}

	// Execute the count query to get the total count of migrated builds.
	migratedBuildQuery, migratedBuildArgs := queries.GetMigratedBuildsCount(task.SqlArgs()...)
	row = c.databaseConnection.QueryRowContext(ctx, migratedBuildQuery, migratedBuildArgs...)
	err = row.Scan(&task.Builds)
	if err != nil {
		return fmt.Errorf("failed to get (migrated) builds count for repository %s: %w", task.FromFQN(), err)
	}

	return nil
}

func (c *client) MigrateComputedTables(ctx context.Context, task *migration.Task) error {
	query, args := queries.MigrateComputedPipeline(task.SqlArgs()...)
	_, err := c.databaseConnection.ExecContext(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("failed to migrate computed pipeline for repository %s: %w", task.FromFQN(), err)
	}
	err = c.UpdateComputedTables(context.WithValue(ctx, isMigration, true), task.ToSource, task.ToOwner, task.ToName)
	if err != nil {
		return fmt.Errorf("failed to migrate compueted tables for repository %s: %w", task.FromFQN(), err)
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

func (c *client) PickMigration(ctx context.Context, maxTasks int64) (tasks []*migration.Task, err error) {
	var transaction *sql.Tx
	transaction, err = c.databaseConnection.BeginTx(ctx, nil)
	defer func(tx *sql.Tx) {
		if err != nil {
			if err := tx.Rollback(); err != nil {
				log.Error().Err(err).Msg("failed to rollback transaction for picking up migration tasks")
			}
		}
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
	err = transaction.Commit()
	// TODO: handle transaction commit error
	return tasks, nil
}

func (c *client) QueueMigration(ctx context.Context, taskToQueue *migration.Task) (task *migration.Task, err error) {
	query, args := queries.CheckExistingMigration(taskToQueue.SqlArgs()...)
	row := c.databaseConnection.QueryRowContext(ctx, query, args...)
	var pipelineExists bool
	var existingTaskID string
	if err = row.Scan(&pipelineExists, &existingTaskID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("failed to check existing migration for: %s: %w", taskToQueue.FromFQN(), ErrPipelineNotFound)
		}
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
	query, args = queries.UnmarkRepositoryArchived(sql.NamedArg{Name: "pickedRepos", Value: pq.Array([]string{task.FromFQN()})})
	if _, err = tx.ExecContext(ctx, query, args...); err != nil {
		return nil, fmt.Errorf("failed to unmark repository %s as archived: %w", task.FromFQN(), err)
	}
	return
}

func (c *client) UpdateMigration(ctx context.Context, task *migration.Task) error {
	// add pod details to error details
	if task.ErrorDetails != nil && *task.ErrorDetails != "" {
		errorSpan := os.Getenv("HOSTNAME")
		if errorSpan == "" {
			errorSpan = "unknown"
		}
		errorSpan = fmt.Sprintf("----\n# pod: %s, time: %s", errorSpan, time.Now().Format(time.RFC3339))
		task.ErrorDetails = ptr.To(fmt.Sprintf("%s\n%s\n----\n", errorSpan, *task.ErrorDetails))
	}
	query, args := queries.UpdateMigration(task.SqlArgs()...)
	_, err := c.databaseConnection.ExecContext(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("failed to update migration task %s: %w", task.ID, err)
	}
	return nil
}

func (c *client) SetPipelineArchival(ctx context.Context, source, owner, name string, archived bool) error {
	query, args := queries.SetPipelineArchival(
		sql.NamedArg{Name: "archived", Value: archived},
		sql.NamedArg{Name: "name", Value: name},
		sql.NamedArg{Name: "owner", Value: owner},
		sql.NamedArg{Name: "source", Value: source},
	)
	_, err := c.databaseConnection.ExecContext(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("failed to set archived status for repository %s/%s: %w", owner, name, err)
	}
	return nil
}

// logging

func (c *loggingClient) GetAllMigrations(ctx context.Context) (tasks []*migration.Task, err error) {
	defer func() { api.HandleLogError(c.prefix, "Client", "GetAllMigrations", err) }()
	return c.Client.GetAllMigrations(ctx)
}
func (c *loggingClient) GetMigratedBuildLogs(ctx context.Context, task *migration.Task) (changes []migration.Change, err error) {
	defer func() { api.HandleLogError(c.prefix, "Client", "GetMigratedBuildLogs", err) }()
	return c.Client.GetMigratedBuildLogs(ctx, task)
}
func (c *loggingClient) GetMigratedReleaseLogs(ctx context.Context, task *migration.Task) (changes []migration.Change, err error) {
	defer func() { api.HandleLogError(c.prefix, "Client", "GetMigratedReleaseLogs", err) }()
	return c.Client.GetMigratedReleaseLogs(ctx, task)
}
func (c *loggingClient) GetMigrationByFromRepo(ctx context.Context, fromSource, fromOwner, fromName string) (task *migration.Task, err error) {
	defer func() { api.HandleLogError(c.prefix, "Client", "GetMigrationByFromRepo", err) }()
	return c.Client.GetMigrationByFromRepo(ctx, fromSource, fromOwner, fromName)
}
func (c *loggingClient) GetMigrationByToRepo(ctx context.Context, toSource, toOwner, toName string) (task *migration.Task, err error) {
	defer func() { api.HandleLogError(c.prefix, "Client", "GetMigrationByToRepo", err) }()
	return c.Client.GetMigrationByToRepo(ctx, toSource, toOwner, toName)
}
func (c *loggingClient) GetMigrationByID(ctx context.Context, taskID string) (task *migration.Task, err error) {
	defer func() { api.HandleLogError(c.prefix, "Client", "GetMigrationByID", err) }()
	return c.Client.GetMigrationByID(ctx, taskID)
}
func (c *loggingClient) MigrateBuildLogs(ctx context.Context, task *migration.Task) (err error) {
	defer func() { api.HandleLogError(c.prefix, "Client", "MigrateBuildLogs", err) }()
	return c.Client.MigrateBuildLogs(ctx, task)
}
func (c *loggingClient) MigrateBuildVersions(ctx context.Context, task *migration.Task) (err error) {
	defer func() { api.HandleLogError(c.prefix, "Client", "MigrateBuildVersions", err) }()
	return c.Client.MigrateBuildVersions(ctx, task)
}
func (c *loggingClient) MigrateBuilds(ctx context.Context, task *migration.Task) (err error) {
	defer func() { api.HandleLogError(c.prefix, "Client", "MigrateBuilds", err) }()
	return c.Client.MigrateBuilds(ctx, task)
}
func (c *loggingClient) MigrateComputedTables(ctx context.Context, task *migration.Task) (err error) {
	defer func() { api.HandleLogError(c.prefix, "Client", "MigrateComputedTables", err) }()
	return c.Client.MigrateComputedTables(ctx, task)
}
func (c *loggingClient) MigrateReleaseLogs(ctx context.Context, task *migration.Task) (err error) {
	defer func() { api.HandleLogError(c.prefix, "Client", "MigrateReleaseLogs", err) }()
	return c.Client.MigrateReleaseLogs(ctx, task)
}
func (c *loggingClient) MigrateReleases(ctx context.Context, task *migration.Task) (err error) {
	defer func() { api.HandleLogError(c.prefix, "Client", "MigrateReleases", err) }()
	return c.Client.MigrateReleases(ctx, task)
}
func (c *loggingClient) PickMigration(ctx context.Context, maxTasks int64) (tasks []*migration.Task, err error) {
	defer func() { api.HandleLogError(c.prefix, "Client", "PickMigration", err) }()
	return c.Client.PickMigration(ctx, maxTasks)
}
func (c *loggingClient) QueueMigration(ctx context.Context, task *migration.Task) (t *migration.Task, err error) {
	defer func() { api.HandleLogError(c.prefix, "Client", "QueueMigration", err) }()
	return c.Client.QueueMigration(ctx, task)
}
func (c *loggingClient) RollbackMigration(ctx context.Context, task *migration.Task) (changes *migration.Changes, err error) {
	defer func() { api.HandleLogError(c.prefix, "Client", "RollbackMigration", err) }()
	return c.Client.RollbackMigration(ctx, task)
}
func (c *loggingClient) UpdateMigration(ctx context.Context, task *migration.Task) (err error) {
	defer func() { api.HandleLogError(c.prefix, "Client", "UpdateMigration", err) }()
	return c.Client.UpdateMigration(ctx, task)
}
func (c *loggingClient) GetMigratedBuild(ctx context.Context, buildID string) (build *contracts.Build, err error) {
	defer func() { api.HandleLogError(c.prefix, "Client", "GetMigratedBuild", err) }()
	return c.Client.GetMigratedBuild(ctx, buildID)
}
func (c *loggingClient) GetMigratedRelease(ctx context.Context, releaseID string) (release *contracts.Release, err error) {
	defer func() { api.HandleLogError(c.prefix, "Client", "GetMigratedRelease", err) }()
	return c.Client.GetMigratedRelease(ctx, releaseID)
}
func (c *loggingClient) SetPipelineArchival(ctx context.Context, source, owner, name string, archived bool) (err error) {
	defer func() { api.HandleLogError(c.prefix, "Client", "SetPipelineArchival", err) }()
	return c.Client.SetPipelineArchival(ctx, source, owner, name, archived)
}

//metrics

func (c *metricsClient) GetAllMigrations(ctx context.Context) (tasks []*migration.Task, err error) {
	api.UpdateMetrics(c.requestCount, c.requestLatency, "GetAllMigrations", time.Now())
	return c.Client.GetAllMigrations(ctx)
}
func (c *metricsClient) GetMigratedBuildLogs(ctx context.Context, task *migration.Task) (changes []migration.Change, err error) {
	api.UpdateMetrics(c.requestCount, c.requestLatency, "GetMigratedBuildLogs", time.Now())
	return c.Client.GetMigratedBuildLogs(ctx, task)
}
func (c *metricsClient) GetMigratedReleaseLogs(ctx context.Context, task *migration.Task) (changes []migration.Change, err error) {
	api.UpdateMetrics(c.requestCount, c.requestLatency, "GetMigratedReleaseLogs", time.Now())
	return c.Client.GetMigratedReleaseLogs(ctx, task)
}
func (c *metricsClient) GetMigrationByFromRepo(ctx context.Context, fromSource, fromOwner, fromName string) (task *migration.Task, err error) {
	api.UpdateMetrics(c.requestCount, c.requestLatency, "GetMigrationByFromRepo", time.Now())
	return c.Client.GetMigrationByFromRepo(ctx, fromSource, fromOwner, fromName)
}

func (c *metricsClient) GetMigrationByToRepo(ctx context.Context, toSource, toOwner, toName string) (task *migration.Task, err error) {
	api.UpdateMetrics(c.requestCount, c.requestLatency, "GetMigrationByToRepo", time.Now())
	return c.Client.GetMigrationByToRepo(ctx, toSource, toOwner, toName)
}
func (c *metricsClient) GetMigrationByID(ctx context.Context, taskID string) (task *migration.Task, err error) {
	api.UpdateMetrics(c.requestCount, c.requestLatency, "GetMigrationByID", time.Now())
	return c.Client.GetMigrationByID(ctx, taskID)
}
func (c *metricsClient) MigrateBuildLogs(ctx context.Context, task *migration.Task) (err error) {
	api.UpdateMetrics(c.requestCount, c.requestLatency, "MigrateBuildLogs", time.Now())
	return c.Client.MigrateBuildLogs(ctx, task)
}
func (c *metricsClient) MigrateBuildVersions(ctx context.Context, task *migration.Task) (err error) {
	api.UpdateMetrics(c.requestCount, c.requestLatency, "MigrateBuildVersions", time.Now())
	return c.Client.MigrateBuildVersions(ctx, task)
}
func (c *metricsClient) MigrateBuilds(ctx context.Context, task *migration.Task) (err error) {
	api.UpdateMetrics(c.requestCount, c.requestLatency, "MigrateBuilds", time.Now())
	return c.Client.MigrateBuilds(ctx, task)
}
func (c *metricsClient) MigrateComputedTables(ctx context.Context, task *migration.Task) (err error) {
	api.UpdateMetrics(c.requestCount, c.requestLatency, "MigrateComputedTables", time.Now())
	return c.Client.MigrateComputedTables(ctx, task)
}
func (c *metricsClient) MigrateReleases(ctx context.Context, task *migration.Task) (err error) {
	api.UpdateMetrics(c.requestCount, c.requestLatency, "MigrateReleases", time.Now())
	return c.Client.MigrateReleases(ctx, task)
}
func (c *metricsClient) MigrateReleaseLogs(ctx context.Context, task *migration.Task) (err error) {
	api.UpdateMetrics(c.requestCount, c.requestLatency, "MigrateReleaseLogs", time.Now())
	return c.Client.MigrateReleaseLogs(ctx, task)
}
func (c *metricsClient) PickMigration(ctx context.Context, maxTasks int64) (tasks []*migration.Task, err error) {
	api.UpdateMetrics(c.requestCount, c.requestLatency, "PickMigration", time.Now())
	return c.Client.PickMigration(ctx, maxTasks)
}
func (c *metricsClient) QueueMigration(ctx context.Context, task *migration.Task) (t *migration.Task, err error) {
	api.UpdateMetrics(c.requestCount, c.requestLatency, "QueueMigration", time.Now())
	return c.Client.QueueMigration(ctx, task)
}
func (c *metricsClient) RollbackMigration(ctx context.Context, task *migration.Task) (changes *migration.Changes, err error) {
	api.UpdateMetrics(c.requestCount, c.requestLatency, "RollbackMigration", time.Now())
	return c.Client.RollbackMigration(ctx, task)
}
func (c *metricsClient) UpdateMigration(ctx context.Context, task *migration.Task) (err error) {
	api.UpdateMetrics(c.requestCount, c.requestLatency, "UpdateMigration", time.Now())
	return c.Client.UpdateMigration(ctx, task)
}
func (c *metricsClient) GetMigratedBuild(ctx context.Context, buildID string) (build *contracts.Build, err error) {
	api.UpdateMetrics(c.requestCount, c.requestLatency, "GetMigratedBuild", time.Now())
	return c.Client.GetMigratedBuild(ctx, buildID)
}
func (c *metricsClient) GetMigratedRelease(ctx context.Context, releaseID string) (release *contracts.Release, err error) {
	api.UpdateMetrics(c.requestCount, c.requestLatency, "GetMigratedRelease", time.Now())
	return c.Client.GetMigratedRelease(ctx, releaseID)
}
func (c *metricsClient) SetPipelineArchival(ctx context.Context, source, owner, name string, archived bool) (err error) {
	api.UpdateMetrics(c.requestCount, c.requestLatency, "SetPipelineArchival", time.Now())
	return c.Client.SetPipelineArchival(ctx, source, owner, name, archived)
}

// logging

func (c *tracingClient) GetAllMigrations(ctx context.Context) (tasks []*migration.Task, err error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, api.GetSpanName(c.prefix, "GetAllMigrations"))
	defer func() { api.FinishSpanWithError(span, err) }()
	return c.Client.GetAllMigrations(ctx)
}
func (c *tracingClient) GetMigratedReleaseLogs(ctx context.Context, task *migration.Task) (changes []migration.Change, err error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, api.GetSpanName(c.prefix, "GetMigratedReleaseLogs"))
	defer func() { api.FinishSpanWithError(span, err) }()
	return c.Client.GetMigratedReleaseLogs(ctx, task)
}
func (c *tracingClient) GetMigrationByFromRepo(ctx context.Context, fromSource, fromOwner, fromName string) (task *migration.Task, err error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, api.GetSpanName(c.prefix, "GetMigrationByFromRepo"))
	defer func() { api.FinishSpanWithError(span, err) }()
	return c.Client.GetMigrationByFromRepo(ctx, fromSource, fromOwner, fromName)
}
func (c *tracingClient) GetMigrationByToRepo(ctx context.Context, toSource, toOwner, toName string) (task *migration.Task, err error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, api.GetSpanName(c.prefix, "GetMigrationByToRepo"))
	defer func() { api.FinishSpanWithError(span, err) }()
	return c.Client.GetMigrationByToRepo(ctx, toSource, toOwner, toName)
}
func (c *tracingClient) GetMigrationByID(ctx context.Context, taskID string) (task *migration.Task, err error) {
	span := opentracing.StartSpan(api.GetSpanName(c.prefix, "GetMigrationByID"))
	defer func() { api.FinishSpanWithError(span, err) }()
	return c.Client.GetMigrationByID(ctx, taskID)
}
func (c *tracingClient) GetMigratedBuildLogs(ctx context.Context, task *migration.Task) (changes []migration.Change, err error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, api.GetSpanName(c.prefix, "GetMigratedBuildLogs"))
	defer func() { api.FinishSpanWithError(span, err) }()
	return c.Client.GetMigratedBuildLogs(ctx, task)
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
func (c *tracingClient) MigrateBuilds(ctx context.Context, task *migration.Task) (err error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, api.GetSpanName(c.prefix, "MigrateBuilds"))
	defer func() { api.FinishSpanWithError(span, err) }()
	return c.Client.MigrateBuilds(ctx, task)
}
func (c *tracingClient) MigrateComputedTables(ctx context.Context, task *migration.Task) (err error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, api.GetSpanName(c.prefix, "MigrateComputedTables"))
	defer func() { api.FinishSpanWithError(span, err) }()
	return c.Client.MigrateComputedTables(ctx, task)
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
func (c *tracingClient) RollbackMigration(ctx context.Context, task *migration.Task) (changes *migration.Changes, err error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, api.GetSpanName(c.prefix, "RollbackMigration"))
	defer func() { api.FinishSpanWithError(span, err) }()
	return c.Client.RollbackMigration(ctx, task)
}
func (c *tracingClient) UpdateMigration(ctx context.Context, task *migration.Task) (err error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, api.GetSpanName(c.prefix, "UpdateMigration"))
	defer func() { api.FinishSpanWithError(span, err) }()
	return c.Client.UpdateMigration(ctx, task)
}
func (c *tracingClient) GetMigratedBuild(ctx context.Context, buildID string) (build *contracts.Build, err error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, api.GetSpanName(c.prefix, "GetMigratedBuild"))
	defer func() { api.FinishSpanWithError(span, err) }()
	return c.Client.GetMigratedBuild(ctx, buildID)
}
func (c *tracingClient) GetMigratedRelease(ctx context.Context, releaseID string) (release *contracts.Release, err error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, api.GetSpanName(c.prefix, "GetMigratedRelease"))
	defer func() { api.FinishSpanWithError(span, err) }()
	return c.Client.GetMigratedRelease(ctx, releaseID)
}
func (c *tracingClient) SetPipelineArchival(ctx context.Context, source, owner, name string, archived bool) (err error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, api.GetSpanName(c.prefix, "SetPipelineArchival"))
	defer func() { api.FinishSpanWithError(span, err) }()
	return c.Client.SetPipelineArchival(ctx, source, owner, name, archived)
}
