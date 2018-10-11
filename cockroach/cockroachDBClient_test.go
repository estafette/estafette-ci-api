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
				From("builds a")

		query, _ = whereClauseGeneratorForAllFilters(query, map[string][]string{})

		// act
		sql, _, err := query.ToSql()

		assert.Nil(t, err)
		assert.Equal(t, "SELECT * FROM builds a", sql)
	})

	t.Run("GeneratesQueryWithStatusFilter", func(t *testing.T) {

		psql := sq.StatementBuilder.PlaceholderFormat(sq.Dollar)

		query :=
			psql.
				Select("*").
				From("builds a")

		query, _ = whereClauseGeneratorForAllFilters(query, map[string][]string{
			"status": []string{
				"succeeded",
			},
		})

		// act
		sql, _, err := query.ToSql()

		assert.Nil(t, err)
		assert.Equal(t, "SELECT * FROM builds a WHERE a.build_status IN ($1)", sql)
	})

	t.Run("GeneratesQueryWithSinceFilter", func(t *testing.T) {

		psql := sq.StatementBuilder.PlaceholderFormat(sq.Dollar)

		query :=
			psql.
				Select("*").
				From("builds a")

		query, _ = whereClauseGeneratorForAllFilters(query, map[string][]string{
			"since": []string{
				"1d",
			},
		})

		// act
		sql, _, err := query.ToSql()

		assert.Nil(t, err)
		assert.Equal(t, "SELECT * FROM builds a WHERE a.inserted_at >= $1", sql)
	})

	t.Run("GeneratesQueryWithLabelsFilter", func(t *testing.T) {

		psql := sq.StatementBuilder.PlaceholderFormat(sq.Dollar)

		query :=
			psql.
				Select("*").
				From("builds a")

		query, _ = whereClauseGeneratorForAllFilters(query, map[string][]string{
			"labels": []string{
				"key=value",
			},
		})

		// act
		sql, _, err := query.ToSql()

		assert.Nil(t, err)
		assert.Equal(t, "SELECT * FROM builds a WHERE a.labels @> $1", sql)
	})

	t.Run("GeneratesGetPipelinesQuery", func(t *testing.T) {

		psql := sq.StatementBuilder.PlaceholderFormat(sq.Dollar)

		query :=
			psql.
				Select("a.id,a.repo_source,a.repo_owner,a.repo_name,a.repo_branch,a.repo_revision,a.build_version,a.build_status,a.labels,a.releases,a.manifest,a.commits,a.inserted_at,a.updated_at").
				From("builds a").
				LeftJoin("builds b ON a.repo_source=b.repo_source AND a.repo_owner=b.repo_owner AND a.repo_name=b.repo_name AND a.inserted_at < b.inserted_at").
				Where("b.id IS NULL").
				OrderBy("a.repo_source,a.repo_owner,a.repo_name").
				Limit(uint64(2)).
				Offset(uint64((2 - 1) * 20))

		// act
		sql, _, err := query.ToSql()

		assert.Nil(t, err)
		assert.Equal(t, "SELECT a.id,a.repo_source,a.repo_owner,a.repo_name,a.repo_branch,a.repo_revision,a.build_version,a.build_status,a.labels,a.releases,a.manifest,a.commits,a.inserted_at,a.updated_at FROM builds a LEFT JOIN builds b ON a.repo_source=b.repo_source AND a.repo_owner=b.repo_owner AND a.repo_name=b.repo_name AND a.inserted_at < b.inserted_at WHERE b.id IS NULL ORDER BY a.repo_source,a.repo_owner,a.repo_name LIMIT 2 OFFSET 20", sql)
	})
}
