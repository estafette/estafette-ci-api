package cockroach

import (
	"testing"

	"github.com/stretchr/testify/assert"

	sq "github.com/Masterminds/squirrel"
)

func TestQueryBuilder(t *testing.T) {
	t.Run("GeneratesQuery", func(t *testing.T) {

		psql := sq.StatementBuilder.PlaceholderFormat(sq.Dollar)

		query :=
			psql.
				Select("id,repo_source,repo_owner,repo_name,repo_branch,repo_revision,build_version,build_status,'' AS labels,manifest,inserted_at,updated_at").
				From("(SELECT *, RANK() OVER (PARTITION BY repo_source,repo_owner,repo_name ORDER BY inserted_at DESC) AS build_version_rank FROM builds)").
				Where(sq.Eq{"build_version_rank": 1}).
				OrderBy("repo_source,repo_owner,repo_name").
				Limit(uint64(2)).
				Offset(uint64((2 - 1) * 20))

		// act
		sql, _, err := query.ToSql()

		assert.Nil(t, err)
		assert.Equal(t, "SELECT id,repo_source,repo_owner,repo_name,repo_branch,repo_revision,build_version,build_status,'' AS labels,manifest,inserted_at,updated_at FROM (SELECT *, RANK() OVER (PARTITION BY repo_source,repo_owner,repo_name ORDER BY inserted_at DESC) AS build_version_rank FROM builds) WHERE build_version_rank = $1 ORDER BY repo_source,repo_owner,repo_name LIMIT 2 OFFSET 20", sql)
	})
}
