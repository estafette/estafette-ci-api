package cockroach

import (
	"testing"

	"github.com/stretchr/testify/assert"

	sq "github.com/Masterminds/squirrel"
)

func TestQueryBuilder(t *testing.T) {
	t.Run("GeneratesQueryWithoutFilters", func(t *testing.T) {

		psql := sq.StatementBuilder.PlaceholderFormat(sq.Dollar)

		query :=
			psql.
				Select("*").
				From("builds")

		query, _ = whereClauseGeneratorForAllFilters(query, map[string][]string{})

		// act
		sql, _, err := query.ToSql()

		assert.Nil(t, err)
		assert.Equal(t, "SELECT * FROM builds", sql)
	})

	t.Run("GeneratesQueryWithStatusFilter", func(t *testing.T) {

		psql := sq.StatementBuilder.PlaceholderFormat(sq.Dollar)

		query :=
			psql.
				Select("*").
				From("builds")

		query, _ = whereClauseGeneratorForAllFilters(query, map[string][]string{
			"status": []string{
				"succeeded",
			},
		})

		// act
		sql, _, err := query.ToSql()

		assert.Nil(t, err)
		assert.Equal(t, "SELECT * FROM builds WHERE build_status IN ($1)", sql)
	})

	t.Run("GeneratesQueryWithSinceFilter", func(t *testing.T) {

		psql := sq.StatementBuilder.PlaceholderFormat(sq.Dollar)

		query :=
			psql.
				Select("*").
				From("builds")

		query, _ = whereClauseGeneratorForAllFilters(query, map[string][]string{
			"since": []string{
				"1d",
			},
		})

		// act
		sql, _, err := query.ToSql()

		assert.Nil(t, err)
		assert.Equal(t, "SELECT * FROM builds WHERE inserted_at >= $1", sql)
	})

	t.Run("GeneratesQueryWithLabelsFilter", func(t *testing.T) {

		psql := sq.StatementBuilder.PlaceholderFormat(sq.Dollar)

		query :=
			psql.
				Select("*").
				From("builds")

		query, _ = whereClauseGeneratorForAllFilters(query, map[string][]string{
			"labels": []string{
				"key=value",
			},
		})

		// act
		sql, _, err := query.ToSql()

		assert.Nil(t, err)
		assert.Equal(t, "SELECT * FROM builds WHERE labels @> $1", sql)
	})

	t.Run("GeneratesGetPipelinesQuery", func(t *testing.T) {

		psql := sq.StatementBuilder.PlaceholderFormat(sq.Dollar)

		innerQuery :=
			psql.
				Select("*, RANK() OVER (PARTITION BY repo_source,repo_owner,repo_name ORDER BY inserted_at DESC) AS build_version_rank").
				From("builds")

		query :=
			psql.
				Select("id,repo_source,repo_owner,repo_name,repo_branch,repo_revision,build_version,build_status,labels,manifest,commits,inserted_at,updated_at").
				FromSelect(innerQuery, "ranked_builds").
				Where(sq.Eq{"build_version_rank": 1}).
				OrderBy("repo_source,repo_owner,repo_name").
				Limit(uint64(2)).
				Offset(uint64((2 - 1) * 20))

		// act
		sql, _, err := query.ToSql()

		assert.Nil(t, err)
		assert.Equal(t, "SELECT id,repo_source,repo_owner,repo_name,repo_branch,repo_revision,build_version,build_status,labels,manifest,commits,inserted_at,updated_at FROM (SELECT *, RANK() OVER (PARTITION BY repo_source,repo_owner,repo_name ORDER BY inserted_at DESC) AS build_version_rank FROM builds) AS ranked_builds WHERE build_version_rank = $1 ORDER BY repo_source,repo_owner,repo_name LIMIT 2 OFFSET 20", sql)
	})
}
