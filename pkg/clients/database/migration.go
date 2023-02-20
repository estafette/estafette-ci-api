package database

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/lib/pq"
	"github.com/opentracing/opentracing-go"
	"github.com/rs/zerolog/log"

	"github.com/estafette/estafette-ci-api/pkg/api"
	"github.com/estafette/estafette-ci-api/pkg/clients/database/queries"
	"github.com/estafette/migration"
)

var (
	isMigration = struct{}{}
	tld         = regexp.MustCompile(`\.(com|org)`)
)

type MigrationDatabaseApi interface {
	CreateMigrationSchema() error
	QueueMigration(ctx context.Context, task *migration.Task) (*migration.Task, error)
	GetMigrationStatus(ctx context.Context, taskID string) (*migration.Task, error)
	PickMigration(ctx context.Context, maxTasks int64) ([]*migration.Task, error)
	UpdateMigration(ctx context.Context, task *migration.Task) error
	MigrateBuilds(ctx context.Context, task *migration.Task) ([]migration.Change, error)
	MigrateBuildLogs(ctx context.Context, task *migration.Task) ([]migration.Change, error)
	MigrateBuildVersions(ctx context.Context, task *migration.Task) ([]migration.Change, error)
	MigrateReleases(ctx context.Context, task *migration.Task) ([]migration.Change, error)
	MigrateReleaseLogs(ctx context.Context, task *migration.Task) ([]migration.Change, error)
	GetMigrationBuildLogsChanges(ctx context.Context, task *migration.Task) ([]migration.Change, error)
	GetMigrationReleaseLogsChanges(ctx context.Context, task *migration.Task) ([]migration.Change, error)
}

func args(m *migration.Task) []any {
	return []any{
		// id, status, lastStep
		sql.NamedArg{Name: "id", Value: m.ID},
		sql.NamedArg{Name: "status", Value: m.Status.String()},
		sql.NamedArg{Name: "lastStep", Value: m.LastStep.String()},
		sql.NamedArg{Name: "builds", Value: m.Builds},
		sql.NamedArg{Name: "releases", Value: m.Releases},
		sql.NamedArg{Name: "totalDuration", Value: m.TotalDuration},
		// From
		sql.NamedArg{Name: "fromSource", Value: m.FromSource},
		sql.NamedArg{Name: "fromSourceName", Value: tld.ReplaceAllString(m.FromSource, "")},
		sql.NamedArg{Name: "fromOwner", Value: m.FromOwner},
		sql.NamedArg{Name: "fromName", Value: m.FromName},
		sql.NamedArg{Name: "fromFullName", Value: m.FromOwner + "/" + m.FromName},
		// To
		sql.NamedArg{Name: "toSource", Value: m.ToSource},
		sql.NamedArg{Name: "toSourceName", Value: tld.ReplaceAllString(m.ToSource, "")},
		sql.NamedArg{Name: "toOwner", Value: m.ToOwner},
		sql.NamedArg{Name: "toName", Value: m.ToName},
		sql.NamedArg{Name: "toFullName", Value: m.ToOwner + "/" + m.ToName},
		// Other
		sql.NamedArg{Name: "callbackURL", Value: m.CallbackURL},
		sql.NamedArg{Name: "errorDetails", Value: m.ErrorDetails},
		sql.NamedArg{Name: "queuedAt", Value: m.QueuedAt},
		sql.NamedArg{Name: "updatedAt", Value: m.UpdatedAt},
	}
}
func closeRows(rows *sql.Rows) {
	err := rows.Close()
	if err != nil {
		log.Error().Err(err).Msg("failed to close rows")
	}
}

// CreateMigrationSchema by modifying the database schema
func (c *client) CreateMigrationSchema() error {
	tx, err := c.databaseConnection.Begin()
	if err != nil {
		return err
	}
	_, err = tx.Exec(queries.MigrationDDL)
	if err != nil {
		return err
	}
	return tx.Commit()
}

func (c *client) QueueMigration(ctx context.Context, taskToQueue *migration.Task) (*migration.Task, error) {
	if taskToQueue.ID == "" {
		return nil, fmt.Errorf("failed to queue migration requestID: %s, %s -> %s: missing task id, should have been genereted if client did't send existing", taskToQueue.ID, taskToQueue.FromFQN(), taskToQueue.ToFQN())
	}
	row := c.databaseConnection.QueryRowContext(ctx, queries.QueueMigration, args(taskToQueue)...)
	var status, lastStep string
	var task migration.Task
	var totalDuration int64
	err := row.Scan(&task.ID, &status, &lastStep, &task.Builds, &task.Releases, &totalDuration, &task.FromSource, &task.FromOwner, &task.FromName, &task.ToSource, &task.ToOwner, &task.ToName, &task.CallbackURL, &task.ErrorDetails, &task.QueuedAt, &task.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("failed to queue migration requestID: %s %s -> %s: %w", task.ID, task.FromFQN(), task.ToFQN(), err)
	}
	task.Status = migration.StatusFrom(status)
	task.LastStep = migration.StepFrom(lastStep)
	task.TotalDuration = time.Duration(totalDuration)
	return &task, nil
}

func (c *client) GetMigrationStatus(ctx context.Context, taskID string) (*migration.Task, error) {
	if taskID == "" {
		return nil, fmt.Errorf("taskID is required to get migration status")
	}
	row := c.databaseConnection.QueryRowContext(ctx, queries.GetMigrationStatus, sql.NamedArg{Name: "id", Value: taskID})
	var status, lastStep string
	var task migration.Task
	var totalDuration int64
	err := row.Scan(&task.ID, &status, &lastStep, &task.Builds, &task.Releases, &totalDuration, &task.FromSource, &task.FromOwner, &task.FromName, &task.ToSource, &task.ToOwner, &task.ToName, &task.CallbackURL, &task.QueuedAt, &task.UpdatedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get migration status for taskID: %s : %w", taskID, err)
	}
	task.Status = migration.StatusFrom(status)
	task.LastStep = migration.StepFrom(lastStep)
	task.TotalDuration = time.Duration(totalDuration)
	return &task, nil
}

func (c *client) PickMigration(ctx context.Context, maxTasks int64) ([]*migration.Task, error) {
	tx, err := c.databaseConnection.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to start transcation for picking up migration tasks: %w", err)
	}
	// query selects task with repositories that don't have running build/ release in descending order of queued_at
	// and puts those tasks in_progress atomically
	rows, err := tx.QueryContext(ctx, strings.ReplaceAll(queries.PickMigration, "@maxTasks", strconv.FormatInt(maxTasks, 10)))
	// for some reason @maxTasks is not working with sql named arguments
	if err != nil {
		return nil, fmt.Errorf("failed to pick up migration tasks from database: %w", err)
	}
	var status, lastStep string
	//var totalDuration int64
	tasks := make([]*migration.Task, 0)
	pickedRepos := make([]string, 0)
	for rows.Next() {
		var task migration.Task
		err = rows.Scan(&task.ID, &status, &lastStep, &task.Builds, &task.Releases, &task.TotalDuration, &task.FromSource, &task.FromOwner, &task.FromName, &task.ToSource, &task.ToOwner, &task.ToName, &task.CallbackURL, &task.QueuedAt, &task.UpdatedAt)
		if err != nil {
			return nil, fmt.Errorf("failed to scan migration task row from database: %w", err)
		}
		task.Status = migration.StatusFrom(status)
		task.LastStep = migration.StepFrom(lastStep)
		//task.TotalDuration = time.Duration(totalDuration)
		tasks = append(tasks, &task)
		pickedRepos = append(pickedRepos, task.FromFQN())
	}
	if len(pickedRepos) > 0 {
		_, err = tx.ExecContext(ctx, queries.MarkRepositoryArchived, sql.NamedArg{Name: "pickedRepos", Value: pq.Array(pickedRepos)})
		if err != nil {
			return nil, fmt.Errorf("failed to mark repositories as archived: %w", err)
		}
	}
	return tasks, tx.Commit()
}

func (c *client) UpdateMigration(ctx context.Context, task *migration.Task) error {
	_, err := c.databaseConnection.ExecContext(ctx, queries.UpdateMigration, args(task)...)
	if err != nil {
		return fmt.Errorf("failed to update migration %s -> %s: %w", task.FromFQN(), task.ToFQN(), err)
	}
	return nil
}

// MigrateReleases for repository FromFQN one source ToFQN another
func (c *client) MigrateReleases(ctx context.Context, task *migration.Task) ([]migration.Change, error) {
	rows, err := c.databaseConnection.QueryContext(ctx, queries.MigrateReleases, args(task)...)
	if err != nil {
		return nil, fmt.Errorf("failed to migrate releases %s -> %s: %w", task.FromFQN(), task.ToFQN(), err)
	}
	changes := make([]migration.Change, 0)
	defer closeRows(rows)
	for rows.Next() {
		var result migration.Change
		if err := rows.Scan(&result.FromID, &result.ToID); err != nil {
			return nil, fmt.Errorf("failed to get changed release rows %s -> %s: %w", task.FromFQN(), task.ToFQN(), err)
		}
		changes = append(changes, result)
	}
	return changes, nil
}

// MigrateReleaseLogs for repository FromFQN one source ToFQN another
func (c *client) MigrateReleaseLogs(ctx context.Context, task *migration.Task) ([]migration.Change, error) {
	rows, err := c.databaseConnection.QueryContext(ctx, queries.MigrateReleaseLogs, args(task)...)
	if err != nil {
		return nil, fmt.Errorf("failed to migrate release logs %s -> %s: %w", task.FromFQN(), task.ToFQN(), err)
	}
	changes := make([]migration.Change, 0)
	defer closeRows(rows)
	for rows.Next() {
		var result migration.Change
		if err := rows.Scan(&result.FromID, &result.ToID); err != nil {
			return nil, fmt.Errorf("failed to get changed release log rows %s -> %s: %w", task.FromFQN(), task.ToFQN(), err)
		}
		changes = append(changes, result)
	}
	return changes, nil
}

// MigrateBuilds for repository FromFQN one source ToFQN another
func (c *client) MigrateBuilds(ctx context.Context, task *migration.Task) ([]migration.Change, error) {
	ctx = context.WithValue(ctx, isMigration, true)
	rows, err := c.databaseConnection.QueryContext(ctx, queries.MigrateBuilds, args(task)...)
	if err != nil {
		return nil, fmt.Errorf("failed to migrate builds %s -> %s: %w", task.FromFQN(), task.ToFQN(), err)
	}
	defer closeRows(rows)
	changes := make([]migration.Change, 0)
	for rows.Next() {
		var change migration.Change
		if err := rows.Scan(&change.FromID, &change.ToID); err != nil {
			return nil, fmt.Errorf("failed to get changed build rows %s -> %s: %w", task.FromFQN(), task.ToFQN(), err)
		}
		changes = append(changes, change)
	}
	// update computed tables
	go func() {
		// create new context to avoid cancellation impacting execution
		span, _ := opentracing.StartSpanFromContext(ctx, "cockroachdb:AsyncUpdateComputedTables")
		ctx = opentracing.ContextWithSpan(context.Background(), span)
		defer span.Finish()

		// !! used to change release and pipeline metadata
		ctx = context.WithValue(ctx, isMigration, true)
		err = c.UpdateComputedTables(ctx, task.ToSource, task.ToOwner, task.ToName)
		if err != nil {
			log.Error().Err(err).Msgf("Failed updating computed tables for pipeline %v", task.ToFQN())
		}
	}()
	return changes, nil
}

// MigrateBuildLogs for repository FromFQN one source ToFQN another
func (c *client) MigrateBuildLogs(ctx context.Context, task *migration.Task) ([]migration.Change, error) {
	rows, err := c.databaseConnection.QueryContext(ctx, queries.MigrateBuildLogs, args(task)...)
	if err != nil {
		return nil, fmt.Errorf("failed to migrate build logs %s -> %s: %w", task.FromFQN(), task.ToFQN(), err)
	}
	changes := make([]migration.Change, 0)
	defer closeRows(rows)
	for rows.Next() {
		var result migration.Change
		if err := rows.Scan(&result.FromID, &result.ToID); err != nil {
			return nil, fmt.Errorf("failed to get changed build log rows %s -> %s: %w", task.FromFQN(), task.ToFQN(), err)
		}
		changes = append(changes, result)
	}
	return changes, nil
}

// MigrateBuildVersions for repository FromFQN one source to another
func (c *client) MigrateBuildVersions(ctx context.Context, task *migration.Task) ([]migration.Change, error) {
	rows, err := c.databaseConnection.QueryContext(ctx, queries.MigrateBuildVersions, args(task)...)
	if err != nil {
		return nil, fmt.Errorf("failed to migrate build versions %s -> %s: %w", task.FromFQN(), task.ToFQN(), err)
	}
	changes := make([]migration.Change, 0)
	defer closeRows(rows)
	for rows.Next() {
		var result migration.Change
		if err := rows.Scan(&result.FromID, &result.ToID); err != nil {
			return nil, fmt.Errorf("failed to get changed build version rows %s -> %s: %w", task.FromFQN(), task.ToFQN(), err)
		}
		changes = append(changes, result)
	}
	return changes, nil
}

func (c *client) GetMigrationBuildLogsChanges(ctx context.Context, task *migration.Task) ([]migration.Change, error) {
	rows, err := c.databaseConnection.QueryContext(ctx, queries.GetMigrationBuildLogsChanges, args(task)...)
	if err != nil {
		return nil, fmt.Errorf("failed query to get migration build log changes %s -> %s: %w", task.FromFQN(), task.ToFQN(), err)
	}
	changes := make([]migration.Change, 0)
	defer closeRows(rows)
	for rows.Next() {
		var result migration.Change
		if err := rows.Scan(&result.FromID, &result.ToID); err != nil {
			return nil, fmt.Errorf("failed to get migration build log changes rows %s -> %s: %w", task.FromFQN(), task.ToFQN(), err)
		}
		changes = append(changes, result)
	}
	return changes, nil
}
func (c *client) GetMigrationReleaseLogsChanges(ctx context.Context, task *migration.Task) ([]migration.Change, error) {
	rows, err := c.databaseConnection.QueryContext(ctx, queries.GetMigrationReleaseLogsChanges, args(task)...)
	if err != nil {
		return nil, fmt.Errorf("failed query to get migration release log changes %s -> %s: %w", task.FromFQN(), task.ToFQN(), err)
	}
	changes := make([]migration.Change, 0)
	defer closeRows(rows)
	for rows.Next() {
		var result migration.Change
		if err := rows.Scan(&result.FromID, &result.ToID); err != nil {
			return nil, fmt.Errorf("failed to get migration release log changes rows %s -> %s: %w", task.FromFQN(), task.ToFQN(), err)
		}
		changes = append(changes, result)
	}
	return changes, nil
}

// logging

func (c *loggingClient) CreateMigrationSchema() (err error) {
	defer func() { api.HandleLogError(c.prefix, "Client", "CreateMigrationSchema", err) }()
	return c.Client.CreateMigrationSchema()
}
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
func (c *loggingClient) MigrateReleases(ctx context.Context, task *migration.Task) (changes []migration.Change, err error) {
	defer func() { api.HandleLogError(c.prefix, "Client", "MigrateReleases", err) }()
	return c.Client.MigrateReleases(ctx, task)
}
func (c *loggingClient) MigrateReleaseLogs(ctx context.Context, task *migration.Task) (changes []migration.Change, err error) {
	defer func() { api.HandleLogError(c.prefix, "Client", "MigrateReleaseLogs", err) }()
	return c.Client.MigrateReleaseLogs(ctx, task)
}
func (c *loggingClient) MigrateBuilds(ctx context.Context, task *migration.Task) (changes []migration.Change, err error) {
	defer func() { api.HandleLogError(c.prefix, "Client", "MigrateBuilds", err) }()
	return c.Client.MigrateBuilds(ctx, task)
}
func (c *loggingClient) MigrateBuildLogs(ctx context.Context, task *migration.Task) (changes []migration.Change, err error) {
	defer func() { api.HandleLogError(c.prefix, "Client", "MigrateBuildLogs", err) }()
	return c.Client.MigrateBuildLogs(ctx, task)
}
func (c *loggingClient) MigrateBuildVersions(ctx context.Context, task *migration.Task) (changes []migration.Change, err error) {
	defer func() { api.HandleLogError(c.prefix, "Client", "MigrateBuildVersions", err) }()
	return c.Client.MigrateBuildVersions(ctx, task)
}
func (c *loggingClient) GetMigrationBuildLogsChanges(ctx context.Context, task *migration.Task) (changes []migration.Change, err error) {
	defer func() { api.HandleLogError(c.prefix, "Client", "GetMigrationBuildLogsChanges", err) }()
	return c.Client.GetMigrationBuildLogsChanges(ctx, task)
}
func (c *loggingClient) GetMigrationReleaseLogsChanges(ctx context.Context, task *migration.Task) (changes []migration.Change, err error) {
	defer func() { api.HandleLogError(c.prefix, "Client", "GetMigrationReleaseLogsChanges", err) }()
	return c.Client.GetMigrationReleaseLogsChanges(ctx, task)
}

// metrics

func (c *metricsClient) CreateMigrationSchema() (err error) {
	api.UpdateMetrics(c.requestCount, c.requestLatency, "CreateMigrationSchema", time.Now())
	return c.Client.CreateMigrationSchema()
}
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

func (c *metricsClient) MigrateReleases(ctx context.Context, task *migration.Task) (changes []migration.Change, err error) {
	api.UpdateMetrics(c.requestCount, c.requestLatency, "MigrateReleases", time.Now())
	return c.Client.MigrateReleases(ctx, task)
}
func (c *metricsClient) MigrateReleaseLogs(ctx context.Context, task *migration.Task) (changes []migration.Change, err error) {
	api.UpdateMetrics(c.requestCount, c.requestLatency, "MigrateReleaseLogs", time.Now())
	return c.Client.MigrateReleaseLogs(ctx, task)
}
func (c *metricsClient) MigrateBuilds(ctx context.Context, task *migration.Task) (changes []migration.Change, err error) {
	api.UpdateMetrics(c.requestCount, c.requestLatency, "MigrateBuilds", time.Now())
	return c.Client.MigrateBuilds(ctx, task)
}
func (c *metricsClient) MigrateBuildLogs(ctx context.Context, task *migration.Task) (changes []migration.Change, err error) {
	api.UpdateMetrics(c.requestCount, c.requestLatency, "MigrateBuildLogs", time.Now())
	return c.Client.MigrateBuildLogs(ctx, task)
}
func (c *metricsClient) MigrateBuildVersions(ctx context.Context, task *migration.Task) (changes []migration.Change, err error) {
	api.UpdateMetrics(c.requestCount, c.requestLatency, "MigrateBuildVersions", time.Now())
	return c.Client.MigrateBuildVersions(ctx, task)
}
func (c *metricsClient) GetMigrationBuildLogsChanges(ctx context.Context, task *migration.Task) (changes []migration.Change, err error) {
	api.UpdateMetrics(c.requestCount, c.requestLatency, "GetMigrationBuildLogsChanges", time.Now())
	return c.Client.GetMigrationBuildLogsChanges(ctx, task)
}
func (c *metricsClient) GetMigrationReleaseLogsChanges(ctx context.Context, task *migration.Task) (changes []migration.Change, err error) {
	api.UpdateMetrics(c.requestCount, c.requestLatency, "GetMigrationReleaseLogsChanges", time.Now())
	return c.Client.GetMigrationReleaseLogsChanges(ctx, task)
}

// logging

func (c *tracingClient) CreateMigrationSchema() (err error) {
	span := opentracing.StartSpan(api.GetSpanName(c.prefix, "CreateMigrationSchema"))
	defer func() { api.FinishSpanWithError(span, err) }()
	return c.Client.CreateMigrationSchema()
}
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

func (c *tracingClient) MigrateReleases(ctx context.Context, task *migration.Task) (changes []migration.Change, err error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, api.GetSpanName(c.prefix, "MigrateReleases"))
	defer func() { api.FinishSpanWithError(span, err) }()
	return c.Client.MigrateReleases(ctx, task)
}
func (c *tracingClient) MigrateReleaseLogs(ctx context.Context, task *migration.Task) (changes []migration.Change, err error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, api.GetSpanName(c.prefix, "MigrateReleaseLogs"))
	defer func() { api.FinishSpanWithError(span, err) }()
	return c.Client.MigrateReleaseLogs(ctx, task)
}
func (c *tracingClient) MigrateBuilds(ctx context.Context, task *migration.Task) (changes []migration.Change, err error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, api.GetSpanName(c.prefix, "MigrateBuilds"))
	defer func() { api.FinishSpanWithError(span, err) }()
	return c.Client.MigrateBuilds(ctx, task)
}
func (c *tracingClient) MigrateBuildLogs(ctx context.Context, task *migration.Task) (changes []migration.Change, err error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, api.GetSpanName(c.prefix, "MigrateBuildLogs"))
	defer func() { api.FinishSpanWithError(span, err) }()
	return c.Client.MigrateBuildLogs(ctx, task)
}
func (c *tracingClient) MigrateBuildVersions(ctx context.Context, task *migration.Task) (changes []migration.Change, err error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, api.GetSpanName(c.prefix, "MigrateBuildVersions"))
	defer func() { api.FinishSpanWithError(span, err) }()
	return c.Client.MigrateBuildVersions(ctx, task)
}
func (c *tracingClient) GetMigrationBuildLogsChanges(ctx context.Context, task *migration.Task) (changes []migration.Change, err error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, api.GetSpanName(c.prefix, "GetMigrationBuildLogsChanges"))
	defer func() { api.FinishSpanWithError(span, err) }()
	return c.Client.MigrateBuildVersions(ctx, task)
}
func (c *tracingClient) GetMigrationReleaseLogsChanges(ctx context.Context, task *migration.Task) (changes []migration.Change, err error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, api.GetSpanName(c.prefix, "GetMigrationReleaseLogsChanges"))
	defer func() { api.FinishSpanWithError(span, err) }()
	return c.Client.MigrateBuildVersions(ctx, task)
}
