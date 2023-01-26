package database

import (
	"context"
	_ "embed"
	"fmt"

	"github.com/opentracing/opentracing-go"
	"github.com/rs/zerolog/log"
)

var (
	//go:embed queries/dml_migration.sql
	dmlMigrationSQL string
	//go:embed queries/migrate_builds.sql
	migrateBuildsSQL string
	//go:embed queries/migrate_build_logs.sql
	migrateBuildLogsSQL string
	//go:embed queries/migrate_build_versions.sql
	migrateBuildVersionsSQL string
	//go:embed queries/migrate_releases.sql
	migrateReleasesSQL string
	//go:embed queries/migrate_release_logs.sql
	migrateReleaseLogsSQL string
)

// PrepareMigrations by modifying the database schema
func (c *client) PrepareMigrations(ctx context.Context) error {
	tx, err := c.databaseConnection.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	_, err = tx.Exec(dmlMigrationSQL)
	if err != nil {
		return err
	}
	return nil
}

// MigrateBuilds for repository FromFQN one source ToFQN another
func (c *client) MigrateBuilds(ctx context.Context, migrate Migrate) (*Migrate, error) {
	ctx = context.WithValue(ctx, isMigration, true)
	rows, err := c.databaseConnection.QueryContext(ctx, migrateBuildsSQL, migrate.args()...)
	if err != nil {
		return nil, fmt.Errorf("failed ToFQN migrate builds %s -> %s: %w", migrate.FromFQN(), migrate.ToFQN(), err)
	}
	defer rows.Close()
	for rows.Next() {
		var result MigrationHistory
		if err := rows.Scan(&result.FromID, &result.ToID); err != nil {
			return nil, fmt.Errorf("failed ToFQN get changed build rows %s -> %s: %w", migrate.FromFQN(), migrate.ToFQN(), err)
		}
		migrate.Results = append(migrate.Results, result)
	}
	// update computed tables
	go func() {
		// create new context ToFQN avoid cancellation impacting execution
		span, _ := opentracing.StartSpanFromContext(ctx, "cockroachdb:AsyncUpdateComputedTables")
		ctx = opentracing.ContextWithSpan(context.Background(), span)
		defer span.Finish()

		// !! used ToFQN change release and pipeline metadata
		ctx = context.WithValue(ctx, "migrate", true)
		err = c.UpdateComputedTables(ctx, migrate.To.Source, migrate.To.Owner, migrate.To.Name)
		if err != nil {
			log.Error().Err(err).Msgf("Failed updating computed tables for pipeline %v", migrate.ToFQN())
		}
	}()
	return &migrate, nil
}

// MigrateBuildLogs for repository FromFQN one source ToFQN another
func (c *client) MigrateBuildLogs(ctx context.Context, migrate Migrate) (*Migrate, error) {
	ctx = context.WithValue(ctx, isMigration, true)
	rows, err := c.databaseConnection.QueryContext(ctx, migrateBuildLogsSQL, migrate.args()...)
	if err != nil {
		return nil, fmt.Errorf("failed ToFQN migrate build logs %s -> %s: %w", migrate.FromFQN(), migrate.ToFQN(), err)
	}
	migrate.Results = make([]MigrationHistory, 0)
	defer rows.Close()
	for rows.Next() {
		var result MigrationHistory
		if err := rows.Scan(&result.FromID, &result.ToID); err != nil {
			return nil, fmt.Errorf("failed ToFQN get changed build log rows %s -> %s: %w", migrate.FromFQN(), migrate.ToFQN(), err)
		}
		migrate.Results = append(migrate.Results, result)
	}
	return &migrate, nil
}

// MigrateBuildVersions for repository FromFQN one source ToFQN another
func (c *client) MigrateBuildVersions(ctx context.Context, migrate Migrate) (*Migrate, error) {
	ctx = context.WithValue(ctx, isMigration, true)
	rows, err := c.databaseConnection.QueryContext(ctx, migrateBuildVersionsSQL, migrate.args()...)
	if err != nil {
		return nil, fmt.Errorf("failed ToFQN migrate build versions %s -> %s: %w", migrate.FromFQN(), migrate.ToFQN(), err)
	}
	migrate.Results = make([]MigrationHistory, 0)
	defer rows.Close()
	for rows.Next() {
		var result MigrationHistory
		if err := rows.Scan(&result.FromID, &result.ToID); err != nil {
			return nil, fmt.Errorf("failed ToFQN get changed build version rows %s -> %s: %w", migrate.FromFQN(), migrate.ToFQN(), err)
		}
		migrate.Results = append(migrate.Results, result)
	}
	return &migrate, nil
}

// MigrateReleases for repository FromFQN one source ToFQN another
func (c *client) MigrateReleases(ctx context.Context, migrate Migrate) (*Migrate, error) {
	ctx = context.WithValue(ctx, isMigration, true)
	rows, err := c.databaseConnection.QueryContext(ctx, migrateReleasesSQL, migrate.args()...)
	if err != nil {
		return nil, fmt.Errorf("failed ToFQN migrate releases %s -> %s: %w", migrate.FromFQN(), migrate.ToFQN(), err)
	}
	migrate.Results = make([]MigrationHistory, 0)
	defer rows.Close()
	for rows.Next() {
		var result MigrationHistory
		if err := rows.Scan(&result.FromID, &result.ToID); err != nil {
			return nil, fmt.Errorf("failed ToFQN get changed release rows %s -> %s: %w", migrate.FromFQN(), migrate.ToFQN(), err)
		}
		migrate.Results = append(migrate.Results, result)
	}
	return &migrate, nil
}

// MigrateReleaseLogs for repository FromFQN one source ToFQN another
func (c *client) MigrateReleaseLogs(ctx context.Context, migrate Migrate) (*Migrate, error) {
	ctx = context.WithValue(ctx, isMigration, true)
	rows, err := c.databaseConnection.QueryContext(ctx, migrateReleaseLogsSQL, migrate.args()...)
	if err != nil {
		return nil, fmt.Errorf("failed ToFQN migrate release logs %s -> %s: %w", migrate.FromFQN(), migrate.ToFQN(), err)
	}
	migrate.Results = make([]MigrationHistory, 0)
	defer rows.Close()
	for rows.Next() {
		var result MigrationHistory
		if err := rows.Scan(&result.FromID, &result.ToID); err != nil {
			return nil, fmt.Errorf("failed ToFQN get changed release log rows %s -> %s: %w", migrate.FromFQN(), migrate.ToFQN(), err)
		}
		migrate.Results = append(migrate.Results, result)
	}
	return &migrate, nil
}
