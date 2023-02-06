package database

import (
	"context"
	"database/sql"
	"fmt"
	"regexp"
	"time"

	"github.com/rs/zerolog/log"

	"github.com/estafette/estafette-ci-api/pkg/api"
	"github.com/estafette/estafette-ci-api/pkg/clients/database/queries"
	"github.com/estafette/estafette-ci-api/pkg/migration"
	"github.com/opentracing/opentracing-go"
)

var (
	isMigration = struct{}{}
	tld         = regexp.MustCompile(`\.(com|org)`)
)

type MigrationDatabaseApi interface {
	CreateMigrationSchema() error
	QueueMigration(ctx context.Context, task *migration.Task) error
	PickMigration(ctx context.Context) (*migration.Task, error)
	UpdateMigration(ctx context.Context, task *migration.Task) error
	MigrateBuilds(ctx context.Context, task *migration.Task) ([]migration.Change, error)
	MigrateBuildLogs(ctx context.Context, task *migration.Task) ([]migration.Change, error)
	MigrateBuildVersions(ctx context.Context, task *migration.Task) ([]migration.Change, error)
	MigrateReleases(ctx context.Context, task *migration.Task) ([]migration.Change, error)
	MigrateReleaseLogs(ctx context.Context, task *migration.Task) ([]migration.Change, error)
}

func args(m *migration.Task) []any {
	return []any{
		// id, status, lastStep
		sql.NamedArg{Name: "id", Value: m.ID},
		sql.NamedArg{Name: "status", Value: m.Status.String()},
		sql.NamedArg{Name: "lastStep", Value: m.LastStep.String()},
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

func (c *client) QueueMigration(ctx context.Context, task *migration.Task) error {
	if task.ID == "" {
		return fmt.Errorf("failed to queue migration requestID: %s %s -> %s: missing task id, should have been genereted if client did't send existing", task.ID, task.FromFQN(), task.ToFQN())
	}
	_, err := c.databaseConnection.ExecContext(ctx, queries.QueueMigration, args(task)...)
	if err != nil {
		return fmt.Errorf("failed to queue migration xrequestID: %s %s -> %s: %w", task.ID, task.FromFQN(), task.ToFQN(), err)
	}
	return nil
}

func (c *client) PickMigration(ctx context.Context) (*migration.Task, error) {
	rows, err := c.databaseConnection.QueryContext(ctx, queries.PickMigration)
	if err != nil {
		return nil, fmt.Errorf("failed to pick migration from database: %w", err)
	}
	defer closeRows(rows)
	var e migration.Task
	for rows.Next() {
		var status, lastStep string
		err = rows.Scan(&e.ID, &status, &lastStep, &e.FromSource, &e.FromOwner, &e.FromName, &e.ToSource, &e.ToOwner, &e.ToName, &e.CallbackURL, &e.QueuedAt, &e.UpdatedAt)
		if err != nil {
			return nil, fmt.Errorf("failed to scan picked up migration from database %s -> %s: %w", e.FromFQN(), e.ToFQN(), err)
		}
		e.Status = migration.StatusFrom(status)
		e.LastStep = migration.StepFrom(lastStep)
		if rows.Next() {
			return nil, fmt.Errorf("failed to pick migration from database: more than one migration was picked up")
		}
	}
	return &e, nil
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

// logging

func (c *loggingClient) CreateMigrationSchema() (err error) {
	defer func() { api.HandleLogError(c.prefix, "Client", "CreateMigrationSchema", err) }()
	return c.Client.CreateMigrationSchema()
}
func (c *loggingClient) QueueMigration(ctx context.Context, task *migration.Task) (err error) {
	defer func() { api.HandleLogError(c.prefix, "Client", "QueueMigration", err) }()
	return c.Client.QueueMigration(ctx, task)
}
func (c *loggingClient) PickMigration(ctx context.Context) (task *migration.Task, err error) {
	defer func() { api.HandleLogError(c.prefix, "Client", "PickMigration", err) }()
	return c.Client.PickMigration(ctx)
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

// metrics

func (c *metricsClient) CreateMigrationSchema() (err error) {
	api.UpdateMetrics(c.requestCount, c.requestLatency, "CreateMigrationSchema", time.Now())
	return c.Client.CreateMigrationSchema()
}
func (c *metricsClient) QueueMigration(ctx context.Context, task *migration.Task) (err error) {
	api.UpdateMetrics(c.requestCount, c.requestLatency, "QueueMigration", time.Now())
	return c.Client.QueueMigration(ctx, task)
}
func (c *metricsClient) PickMigration(ctx context.Context) (task *migration.Task, err error) {
	api.UpdateMetrics(c.requestCount, c.requestLatency, "PickMigration", time.Now())
	return c.Client.PickMigration(ctx)
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

// logging

func (c *tracingClient) CreateMigrationSchema() (err error) {
	span := opentracing.StartSpan(api.GetSpanName(c.prefix, "CreateMigrationSchema"))
	defer func() { api.FinishSpanWithError(span, err) }()
	return c.Client.CreateMigrationSchema()
}
func (c *tracingClient) QueueMigration(ctx context.Context, task *migration.Task) (err error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, api.GetSpanName(c.prefix, "QueueMigration"))
	defer func() { api.FinishSpanWithError(span, err) }()
	return c.Client.QueueMigration(ctx, task)
}
func (c *tracingClient) PickMigration(ctx context.Context) (task *migration.Task, err error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, api.GetSpanName(c.prefix, "PickMigration"))
	defer func() { api.FinishSpanWithError(span, err) }()
	return c.Client.PickMigration(ctx)
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
