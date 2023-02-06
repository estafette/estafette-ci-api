package queries

import (
	_ "embed"
)

var (
	//go:embed migrate_build_logs.sql
	MigrateBuildLogs string
	//go:embed migrate_build_versions.sql
	MigrateBuildVersions string
	//go:embed migrate_builds.sql
	MigrateBuilds string
	//go:embed migrate_release_logs.sql
	MigrateReleaseLogs string
	//go:embed migrate_releases.sql
	MigrateReleases string
	//go:embed migration_ddl.sql
	MigrationDDL string
	//go:embed pick_migration.sql
	PickMigration string
	//go:embed queue_migration.sql
	QueueMigration string
	//go:embed update_migration.sql
	UpdateMigration string
)
