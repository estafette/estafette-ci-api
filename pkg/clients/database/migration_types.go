package database

import (
	"context"
	"database/sql"
	"fmt"
	"regexp"
)

var (
	isMigration = struct{}{}
	tld         = regexp.MustCompile(`\.(com|org)`)
)

type MigrationApi interface {
	PrepareMigrations(ctx context.Context) error
	MigrateBuilds(ctx context.Context, migrate Migrate) (*Migrate, error)
	MigrateBuildLogs(ctx context.Context, migrate Migrate) (*Migrate, error)
	MigrateBuildVersions(ctx context.Context, migrate Migrate) (*Migrate, error)
	MigrateReleases(ctx context.Context, migrate Migrate) (*Migrate, error)
	MigrateReleaseLogs(ctx context.Context, migrate Migrate) (*Migrate, error)
}

type Repository struct {
	Source string `json:"source,omitempty"`
	Owner  string `json:"owner,omitempty"`
	Name   string `json:"name,omitempty"`
}

type MigrationHistory struct {
	FromID int64 `json:"fromId"`
	ToID   int64 `json:"toId"`
}

type Migrate struct {
	From    Repository         `json:"from"`
	To      Repository         `json:"to"`
	Results []MigrationHistory `json:"results,omitempty"`
}

func (m Migrate) FromFQN() string {
	return fmt.Sprintf("%s/%s/%s", m.From.Source, m.From.Owner, m.From.Name)
}

func (m Migrate) ToFQN() string {
	return fmt.Sprintf("%s/%s/%s", m.To.Source, m.To.Owner, m.To.Name)
}

func (m Migrate) args() []any {
	return []any{
		// FromFQN
		sql.NamedArg{Name: "fromSource", Value: m.From.Source},
		sql.NamedArg{Name: "fromSourceName", Value: tld.ReplaceAllString(m.From.Source, "")},
		sql.NamedArg{Name: "fromOwner", Value: m.From.Owner},
		sql.NamedArg{Name: "fromName", Value: m.From.Name},
		sql.NamedArg{Name: "fromFullName", Value: m.From.Owner + "/" + m.From.Name},
		// ToFQN
		sql.NamedArg{Name: "toSource", Value: m.To.Source},
		sql.NamedArg{Name: "toSourceName", Value: tld.ReplaceAllString(m.To.Source, "")},
		sql.NamedArg{Name: "toOwner", Value: m.To.Owner},
		sql.NamedArg{Name: "toName", Value: m.To.Name},
		sql.NamedArg{Name: "toFullName", Value: m.To.Owner + "/" + m.To.Name},
	}
}
