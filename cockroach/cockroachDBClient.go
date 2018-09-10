package cockroach

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/estafette/estafette-ci-manifest"
	"gopkg.in/yaml.v2"

	sq "github.com/Masterminds/squirrel"
	"github.com/estafette/estafette-ci-api/config"
	"github.com/estafette/estafette-ci-contracts"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/rs/zerolog/log"

	// use postgres client library to connect to cockroachdb
	_ "github.com/lib/pq"
)

// DBClient is the interface for communicating with CockroachDB
type DBClient interface {
	Connect() error
	ConnectWithDriverAndSource(string, string) error
	GetAutoIncrement(string, string) (int, error)
	InsertBuild(contracts.Build) (contracts.Build, error)
	UpdateBuildStatus(string, string, string, string, string, string) error
	InsertRelease(contracts.Release) (contracts.Release, error)
	UpdateReleaseStatus(string, string, string, int, string) error
	GetPipelines(int, int, map[string][]string) ([]*contracts.Pipeline, error)
	GetPipelinesByRepoName(string) ([]*contracts.Pipeline, error)
	GetPipelinesCount(map[string][]string) (int, error)
	GetPipeline(string, string, string) (*contracts.Pipeline, error)
	GetPipelineBuilds(string, string, string, int, int) ([]*contracts.Build, error)
	GetPipelineBuildsCount(string, string, string) (int, error)
	GetPipelineBuild(string, string, string, string) (*contracts.Build, error)
	GetPipelineBuildByID(string, string, string, int) (*contracts.Build, error)
	GetPipelineBuildByVersion(string, string, string, string) (*contracts.Build, error)
	GetPipelineBuildLogs(string, string, string, string, string) (*contracts.BuildLog, error)
	GetPipelineReleases(string, string, string, int, int) ([]*contracts.Release, error)
	GetPipelineReleasesCount(string, string, string) (int, error)
	GetPipelineRelease(string, string, string, int) (*contracts.Release, error)
	GetPipelineLastReleaseByName(string, string, string, string) (*contracts.Release, error)
	GetPipelineReleaseLogs(string, string, string, int) (*contracts.ReleaseLog, error)
	GetBuildsCount(map[string][]string) (int, error)
	GetReleasesCount(map[string][]string) (int, error)
	GetBuildsDuration(map[string][]string) (time.Duration, error)
	InsertBuildLog(contracts.BuildLog) error
	InsertReleaseLog(contracts.ReleaseLog) error
}

type cockroachDBClientImpl struct {
	databaseDriver                  string
	config                          config.DatabaseConfig
	PrometheusOutboundAPICallTotals *prometheus.CounterVec
	databaseConnection              *sql.DB
}

// NewCockroachDBClient returns a new cockroach.DBClient
func NewCockroachDBClient(config config.DatabaseConfig, prometheusOutboundAPICallTotals *prometheus.CounterVec) (cockroachDBClient DBClient) {

	cockroachDBClient = &cockroachDBClientImpl{
		databaseDriver: "postgres",
		config:         config,
		PrometheusOutboundAPICallTotals: prometheusOutboundAPICallTotals,
	}

	return
}

// Connect sets up a connection with CockroachDB
func (dbc *cockroachDBClientImpl) Connect() (err error) {

	log.Debug().Msgf("Connecting to database %v on host %v...", dbc.config.DatabaseName, dbc.config.Host)

	sslMode := ""
	if dbc.config.Insecure {
		sslMode = "?sslmode=disable"
	}

	dataSourceName := fmt.Sprintf("postgresql://%v:%v@%v:%v/%v%v", dbc.config.User, dbc.config.Password, dbc.config.Host, dbc.config.Port, dbc.config.DatabaseName, sslMode)

	return dbc.ConnectWithDriverAndSource(dbc.databaseDriver, dataSourceName)
}

// ConnectWithDriverAndSource set up a connection with any database
func (dbc *cockroachDBClientImpl) ConnectWithDriverAndSource(driverName string, dataSourceName string) (err error) {

	dbc.databaseConnection, err = sql.Open(driverName, dataSourceName)
	if err != nil {
		return
	}

	return
}

// GetAutoIncrement returns the autoincrement number for a pipeline
func (dbc *cockroachDBClientImpl) GetAutoIncrement(gitSource, gitFullname string) (autoincrement int, err error) {

	dbc.PrometheusOutboundAPICallTotals.With(prometheus.Labels{"target": "cockroachdb"}).Inc()

	// insert or increment if record for repo_source and repo_full_name combination already exists
	_, err = dbc.databaseConnection.Exec(
		`
		INSERT INTO
			build_versions
		(
			repo_source,
			repo_full_name
		)
		VALUES
		(
			$1,
			$2
		)
		ON CONFLICT
		(
			repo_source,
			repo_full_name
		)
		DO UPDATE SET
			auto_increment = build_versions.auto_increment + 1,
			updated_at = now()
		`,
		gitSource,
		gitFullname,
	)
	if err != nil {
		return
	}

	dbc.PrometheusOutboundAPICallTotals.With(prometheus.Labels{"target": "cockroachdb"}).Inc()

	// fetching auto_increment value, because RETURNING is not supported with UPSERT / INSERT ON CONFLICT (see issue https://github.com/cockroachdb/cockroach/issues/6637)
	rows, err := dbc.databaseConnection.Query(
		`
		SELECT
			auto_increment
		FROM
			build_versions
		WHERE
			repo_source=$1 AND
			repo_full_name=$2
		`,
		gitSource,
		gitFullname,
	)
	if err != nil {
		return
	}

	defer rows.Close()
	for rows.Next() {
		if err = rows.Scan(&autoincrement); err != nil {
			return
		}
	}

	return
}

func (dbc *cockroachDBClientImpl) InsertBuild(build contracts.Build) (insertedBuild contracts.Build, err error) {
	dbc.PrometheusOutboundAPICallTotals.With(prometheus.Labels{"target": "cockroachdb"}).Inc()

	labelsBytes, err := json.Marshal(build.Labels)
	if err != nil {
		return
	}
	releasesBytes, err := json.Marshal(build.Releases)
	if err != nil {
		return
	}
	commitsBytes, err := json.Marshal(build.Commits)
	if err != nil {
		return
	}

	// insert logs
	row := dbc.databaseConnection.QueryRow(
		`
		INSERT INTO
			builds
		(
			repo_source,
			repo_owner,
			repo_name,
			repo_branch,
			repo_revision,
			build_version,
			build_status,
			labels,
			releases,
			manifest,
			commits
		)
		VALUES
		(
			$1,
			$2,
			$3,
			$4,
			$5,
			$6,
			$7,
			$8,
			$9,
			$10,
			$11
		)
		RETURNING
			id
		`,
		build.RepoSource,
		build.RepoOwner,
		build.RepoName,
		build.RepoBranch,
		build.RepoRevision,
		build.BuildVersion,
		build.BuildStatus,
		labelsBytes,
		releasesBytes,
		build.Manifest,
		commitsBytes,
	)

	insertedBuild = build

	if err := row.Scan(&insertedBuild.ID); err != nil {
		return insertedBuild, err
	}

	return
}

func (dbc *cockroachDBClientImpl) UpdateBuildStatus(repoSource, repoOwner, repoName, repoBranch, repoRevision, buildStatus string) (err error) {

	dbc.PrometheusOutboundAPICallTotals.With(prometheus.Labels{"target": "cockroachdb"}).Inc()

	if repoBranch != "" {
		// update build status
		_, err = dbc.databaseConnection.Exec(
			`
		UPDATE
			builds
		SET
			build_status=$1,
			updated_at=now()
		WHERE
			repo_source=$2 AND
			repo_owner=$3 AND
			repo_name=$4 AND
			repo_branch=$5 AND
			repo_revision=$6
		`,
			buildStatus,
			repoSource,
			repoOwner,
			repoName,
			repoBranch,
			repoRevision,
		)

		if err != nil {
			return
		}

	} else {
		// update build status
		_, err = dbc.databaseConnection.Exec(
			`
		UPDATE
			builds
		SET
			build_status=$1,
			updated_at=now()
		WHERE
			repo_source=$2 AND
			repo_owner=$3 AND
			repo_name=$4 AND
			repo_revision=$5
		`,
			buildStatus,
			repoSource,
			repoOwner,
			repoName,
			repoRevision,
		)

		if err != nil {
			return
		}
	}

	return
}

func (dbc *cockroachDBClientImpl) InsertRelease(release contracts.Release) (insertedRelease contracts.Release, err error) {
	dbc.PrometheusOutboundAPICallTotals.With(prometheus.Labels{"target": "cockroachdb"}).Inc()

	// insert logs
	rows, err := dbc.databaseConnection.Query(
		`
		INSERT INTO
			releases
		(
			repo_source,
			repo_owner,
			repo_name,
			release,
			release_version,
			release_status,
			triggered_by
		)
		VALUES
		(
			$1,
			$2,
			$3,
			$4,
			$5,
			$6,
			$7
		)
		RETURNING 
			id
		`,
		release.RepoSource,
		release.RepoOwner,
		release.RepoName,
		release.Name,
		release.ReleaseVersion,
		release.ReleaseStatus,
		release.TriggeredBy,
	)

	if err != nil {
		return insertedRelease, err
	}

	defer rows.Close()
	recordExists := rows.Next()

	if !recordExists {
		return
	}

	insertedRelease = release

	if err := rows.Scan(&insertedRelease.ID); err != nil {
		return insertedRelease, err
	}

	return
}

func (dbc *cockroachDBClientImpl) UpdateReleaseStatus(repoSource, repoOwner, repoName string, id int, releaseStatus string) (err error) {

	dbc.PrometheusOutboundAPICallTotals.With(prometheus.Labels{"target": "cockroachdb"}).Inc()

	// insert logs
	_, err = dbc.databaseConnection.Exec(
		`
		UPDATE
			releases
		SET
			release_status=$1,
			updated_at=now()
		WHERE
			id=$2 AND
			repo_source=$3 AND
			repo_owner=$4 AND
			repo_name=$5
		`,
		releaseStatus,
		id,
		repoSource,
		repoOwner,
		repoName,
	)

	if err != nil {
		return
	}

	return
}

func (dbc *cockroachDBClientImpl) GetPipelines(pageNumber, pageSize int, filters map[string][]string) (pipelines []*contracts.Pipeline, err error) {

	dbc.PrometheusOutboundAPICallTotals.With(prometheus.Labels{"target": "cockroachdb"}).Inc()

	psql := sq.StatementBuilder.PlaceholderFormat(sq.Dollar)

	innerQuery :=
		psql.
			Select("*, RANK() OVER (PARTITION BY repo_source,repo_owner,repo_name ORDER BY inserted_at DESC) AS build_version_rank").
			From("builds")

	query :=
		psql.
			Select("id,repo_source,repo_owner,repo_name,repo_branch,repo_revision,build_version,build_status,labels,releases,manifest,commits,inserted_at,updated_at").
			FromSelect(innerQuery, "ranked_builds").
			Where(sq.Eq{"build_version_rank": 1})

	query, err = whereClauseGeneratorForAllFilters(query, filters)
	if err != nil {
		return
	}

	query = query.
		OrderBy("repo_source,repo_owner,repo_name").
		Limit(uint64(pageSize)).
		Offset(uint64((pageNumber - 1) * pageSize))

	pipelines = make([]*contracts.Pipeline, 0)

	rows, err := query.RunWith(dbc.databaseConnection).Query()

	if err != nil {
		return
	}

	defer rows.Close()
	for rows.Next() {

		var labelsData, releasesData, commitsData []uint8

		pipeline := contracts.Pipeline{}

		if err := rows.Scan(
			&pipeline.ID,
			&pipeline.RepoSource,
			&pipeline.RepoOwner,
			&pipeline.RepoName,
			&pipeline.RepoBranch,
			&pipeline.RepoRevision,
			&pipeline.BuildVersion,
			&pipeline.BuildStatus,
			&labelsData,
			&releasesData,
			&pipeline.Manifest,
			&commitsData,
			&pipeline.InsertedAt,
			&pipeline.UpdatedAt); err != nil {
			return nil, err
		}

		if len(labelsData) > 0 {
			if err = json.Unmarshal(labelsData, &pipeline.Labels); err != nil {
				return
			}
		}
		if len(releasesData) > 0 {
			if err = json.Unmarshal(releasesData, &pipeline.Releases); err != nil {
				return
			}
		}
		if len(commitsData) > 0 {
			if err = json.Unmarshal(commitsData, &pipeline.Commits); err != nil {
				return
			}
		}

		// set released versions
		updatedReleases := make([]contracts.Release, 0)
		for _, r := range pipeline.Releases {
			latestRelease, err := dbc.GetPipelineLastReleaseByName(pipeline.RepoSource, pipeline.RepoOwner, pipeline.RepoName, r.Name)
			if err != nil {
				log.Error().Err(err).Msgf("Failed retrieving latest release for %v/%v/%v %v", pipeline.RepoSource, pipeline.RepoOwner, pipeline.RepoName, r.Name)
			} else if latestRelease != nil {
				updatedReleases = append(updatedReleases, *latestRelease)
			} else {
				updatedReleases = append(updatedReleases, r)
			}
		}
		pipeline.Releases = updatedReleases

		// unmarshal then marshal manifest to include defaults
		var manifest manifest.EstafetteManifest
		err = yaml.Unmarshal([]byte(pipeline.Manifest), &manifest)
		if err == nil {
			manifestWithDefaultBytes, err := yaml.Marshal(manifest)
			if err == nil {
				pipeline.ManifestWithDefaults = string(manifestWithDefaultBytes)
			} else {
				log.Warn().Err(err).Interface("manifest", manifest).Msgf("Marshalling manifest for %v/%v/%v revision %v failed", pipeline.RepoSource, pipeline.RepoOwner, pipeline.RepoName, pipeline.RepoRevision)
			}
		} else {
			log.Warn().Err(err).Str("manifest", pipeline.Manifest).Msgf("Unmarshalling manifest for %v/%v/%v revision %v failed", pipeline.RepoSource, pipeline.RepoOwner, pipeline.RepoName, pipeline.RepoRevision)
		}

		pipelines = append(pipelines, &pipeline)
	}

	return
}

func (dbc *cockroachDBClientImpl) GetPipelinesByRepoName(repoName string) (pipelines []*contracts.Pipeline, err error) {

	dbc.PrometheusOutboundAPICallTotals.With(prometheus.Labels{"target": "cockroachdb"}).Inc()

	psql := sq.StatementBuilder.PlaceholderFormat(sq.Dollar)

	innerQuery :=
		psql.
			Select("*, RANK() OVER (PARTITION BY repo_source,repo_owner,repo_name ORDER BY inserted_at DESC) AS build_version_rank").
			From("builds")

	query :=
		psql.
			Select("id,repo_source,repo_owner,repo_name,repo_branch,repo_revision,build_version,build_status,labels,releases,manifest,commits,inserted_at,updated_at").
			FromSelect(innerQuery, "ranked_builds").
			Where(sq.Eq{"build_version_rank": 1}).
			Where(sq.Eq{"repo_name": repoName})

	pipelines = make([]*contracts.Pipeline, 0)

	rows, err := query.RunWith(dbc.databaseConnection).Query()

	if err != nil {
		return
	}

	defer rows.Close()
	for rows.Next() {

		var labelsData, releasesData, commitsData []uint8

		pipeline := contracts.Pipeline{}

		if err := rows.Scan(
			&pipeline.ID,
			&pipeline.RepoSource,
			&pipeline.RepoOwner,
			&pipeline.RepoName,
			&pipeline.RepoBranch,
			&pipeline.RepoRevision,
			&pipeline.BuildVersion,
			&pipeline.BuildStatus,
			&labelsData,
			&releasesData,
			&pipeline.Manifest,
			&commitsData,
			&pipeline.InsertedAt,
			&pipeline.UpdatedAt); err != nil {
			return nil, err
		}

		if len(labelsData) > 0 {
			if err = json.Unmarshal(labelsData, &pipeline.Labels); err != nil {
				return
			}
		}
		if len(releasesData) > 0 {
			if err = json.Unmarshal(releasesData, &pipeline.Releases); err != nil {
				return
			}
		}
		if len(commitsData) > 0 {
			if err = json.Unmarshal(commitsData, &pipeline.Commits); err != nil {
				return
			}
		}

		// unmarshal then marshal manifest to include defaults
		var manifest manifest.EstafetteManifest
		err = yaml.Unmarshal([]byte(pipeline.Manifest), &manifest)
		if err == nil {
			manifestWithDefaultBytes, err := yaml.Marshal(manifest)
			if err == nil {
				pipeline.ManifestWithDefaults = string(manifestWithDefaultBytes)
			} else {
				log.Warn().Err(err).Interface("manifest", manifest).Msgf("Marshalling manifest for %v/%v/%v revision %v failed", pipeline.RepoSource, pipeline.RepoOwner, pipeline.RepoName, pipeline.RepoRevision)
			}
		} else {
			log.Warn().Err(err).Str("manifest", pipeline.Manifest).Msgf("Unmarshalling manifest for %v/%v/%v revision %v failed", pipeline.RepoSource, pipeline.RepoOwner, pipeline.RepoName, pipeline.RepoRevision)
		}

		pipelines = append(pipelines, &pipeline)
	}

	return
}

func (dbc *cockroachDBClientImpl) GetPipelinesCount(filters map[string][]string) (totalCount int, err error) {

	dbc.PrometheusOutboundAPICallTotals.With(prometheus.Labels{"target": "cockroachdb"}).Inc()

	psql := sq.StatementBuilder.PlaceholderFormat(sq.Dollar)

	innerQuery :=
		psql.
			Select("*, RANK() OVER (PARTITION BY repo_source,repo_owner,repo_name ORDER BY inserted_at DESC) AS build_version_rank").
			From("builds")

	query :=
		psql.
			Select("COUNT(*)").
			FromSelect(innerQuery, "ranked_builds").
			Where(sq.Eq{"build_version_rank": 1})

	query, err = whereClauseGeneratorForAllFilters(query, filters)

	rows, err := query.RunWith(dbc.databaseConnection).Query()
	if err != nil {
		return
	}

	defer rows.Close()
	recordExists := rows.Next()

	if !recordExists {
		return
	}

	if err := rows.Scan(&totalCount); err != nil {
		return 0, err
	}

	return
}

func (dbc *cockroachDBClientImpl) GetPipeline(repoSource, repoOwner, repoName string) (pipeline *contracts.Pipeline, err error) {

	dbc.PrometheusOutboundAPICallTotals.With(prometheus.Labels{"target": "cockroachdb"}).Inc()

	rows, err := dbc.databaseConnection.Query(
		`
		SELECT
			id,
			repo_source,
			repo_owner,
			repo_name,
			repo_branch,
			repo_revision,
			build_version,
			build_status,
			labels,
			releases,
			manifest,
			commits,
			inserted_at,
			updated_at
		FROM
			builds
		WHERE
			repo_source=$1 AND
			repo_owner=$2 AND
			repo_name=$3
		ORDER BY
			inserted_at DESC
		LIMIT 1
		OFFSET 0
		`,
		repoSource,
		repoOwner,
		repoName,
	)
	if err != nil {
		return
	}

	recordExists := false

	defer rows.Close()
	recordExists = rows.Next()

	if !recordExists {
		return
	}

	var labelsData, releasesData, commitsData []uint8

	pipeline = &contracts.Pipeline{}

	if err := rows.Scan(
		&pipeline.ID,
		&pipeline.RepoSource,
		&pipeline.RepoOwner,
		&pipeline.RepoName,
		&pipeline.RepoBranch,
		&pipeline.RepoRevision,
		&pipeline.BuildVersion,
		&pipeline.BuildStatus,
		&labelsData,
		&releasesData,
		&pipeline.Manifest,
		&commitsData,
		&pipeline.InsertedAt,
		&pipeline.UpdatedAt); err != nil {
		return nil, err
	}

	if len(labelsData) > 0 {
		if err = json.Unmarshal(labelsData, &pipeline.Labels); err != nil {
			return
		}
	}
	if len(releasesData) > 0 {
		if err = json.Unmarshal(releasesData, &pipeline.Releases); err != nil {
			return
		}
	}
	if len(commitsData) > 0 {
		if err = json.Unmarshal(commitsData, &pipeline.Commits); err != nil {
			return
		}
	}

	// set released versions
	updatedReleases := make([]contracts.Release, 0)
	for _, r := range pipeline.Releases {
		latestRelease, err := dbc.GetPipelineLastReleaseByName(pipeline.RepoSource, pipeline.RepoOwner, pipeline.RepoName, r.Name)
		if err != nil {
			log.Error().Err(err).Msgf("Failed retrieving latest release for %v/%v/%v %v", pipeline.RepoSource, pipeline.RepoOwner, pipeline.RepoName, r.Name)
		} else if latestRelease != nil {
			updatedReleases = append(updatedReleases, *latestRelease)
		} else {
			updatedReleases = append(updatedReleases, r)
		}
	}
	pipeline.Releases = updatedReleases

	// unmarshal then marshal manifest to include defaults
	var manifest manifest.EstafetteManifest
	err = yaml.Unmarshal([]byte(pipeline.Manifest), &manifest)
	if err == nil {
		manifestWithDefaultBytes, err := yaml.Marshal(manifest)
		if err == nil {
			pipeline.ManifestWithDefaults = string(manifestWithDefaultBytes)
		} else {
			log.Warn().Err(err).Interface("manifest", manifest).Msgf("Marshalling manifest for %v/%v/%v revision %v failed", pipeline.RepoSource, pipeline.RepoOwner, pipeline.RepoName, pipeline.RepoRevision)
		}
	} else {
		log.Warn().Err(err).Str("manifest", pipeline.Manifest).Msgf("Unmarshalling manifest for %v/%v/%v revision %v failed", pipeline.RepoSource, pipeline.RepoOwner, pipeline.RepoName, pipeline.RepoRevision)
	}

	return
}

func (dbc *cockroachDBClientImpl) GetPipelineBuilds(repoSource, repoOwner, repoName string, pageNumber, pageSize int) (builds []*contracts.Build, err error) {

	dbc.PrometheusOutboundAPICallTotals.With(prometheus.Labels{"target": "cockroachdb"}).Inc()

	builds = make([]*contracts.Build, 0)

	rows, err := dbc.databaseConnection.Query(
		`
		SELECT
			id,
			repo_source,
			repo_owner,
			repo_name,
			repo_branch,
			repo_revision,
			build_version,
			build_status,
			labels,
			releases,
			manifest,
			commits,
			inserted_at,
			updated_at
		FROM
			builds
		WHERE
			repo_source=$1 AND
			repo_owner=$2 AND
			repo_name=$3
		ORDER BY
			inserted_at DESC
		LIMIT $4
		OFFSET $5
		`,
		repoSource,
		repoOwner,
		repoName,
		pageSize,
		(pageNumber-1)*pageSize,
	)
	if err != nil {
		return
	}

	defer rows.Close()
	for rows.Next() {

		var labelsData, releasesData, commitsData []uint8

		build := contracts.Build{}

		if err := rows.Scan(
			&build.ID,
			&build.RepoSource,
			&build.RepoOwner,
			&build.RepoName,
			&build.RepoBranch,
			&build.RepoRevision,
			&build.BuildVersion,
			&build.BuildStatus,
			&labelsData,
			&releasesData,
			&build.Manifest,
			&commitsData,
			&build.InsertedAt,
			&build.UpdatedAt); err != nil {
			return nil, err
		}

		if len(labelsData) > 0 {
			if err = json.Unmarshal(labelsData, &build.Labels); err != nil {
				return
			}
		}
		if len(releasesData) > 0 {
			if err = json.Unmarshal(releasesData, &build.Releases); err != nil {
				return
			}
		}
		if len(commitsData) > 0 {
			if err = json.Unmarshal(commitsData, &build.Commits); err != nil {
				return
			}
		}

		// unmarshal then marshal manifest to include defaults
		var manifest manifest.EstafetteManifest
		err = yaml.Unmarshal([]byte(build.Manifest), &manifest)
		if err == nil {
			manifestWithDefaultBytes, err := yaml.Marshal(manifest)
			if err == nil {
				build.ManifestWithDefaults = string(manifestWithDefaultBytes)
			} else {
				log.Warn().Err(err).Interface("manifest", manifest).Msgf("Marshalling manifest for %v/%v/%v revision %v failed", build.RepoSource, build.RepoOwner, build.RepoName, build.RepoRevision)
			}
		} else {
			log.Warn().Err(err).Str("manifest", build.Manifest).Msgf("Unmarshalling manifest for %v/%v/%v revision %v failed", build.RepoSource, build.RepoOwner, build.RepoName, build.RepoRevision)
		}

		builds = append(builds, &build)
	}

	return
}

func (dbc *cockroachDBClientImpl) GetPipelineBuildsCount(repoSource, repoOwner, repoName string) (totalCount int, err error) {

	dbc.PrometheusOutboundAPICallTotals.With(prometheus.Labels{"target": "cockroachdb"}).Inc()

	rows, err := dbc.databaseConnection.Query(
		`
		SELECT
			COUNT(*)
		FROM
			builds
		WHERE
			repo_source=$1 AND
			repo_owner=$2 AND
			repo_name=$3
		`,
		repoSource,
		repoOwner,
		repoName,
	)
	if err != nil {
		return
	}

	defer rows.Close()
	recordExists := rows.Next()

	if !recordExists {
		return
	}

	if err := rows.Scan(&totalCount); err != nil {
		return 0, err
	}

	return
}

func (dbc *cockroachDBClientImpl) GetPipelineBuild(repoSource, repoOwner, repoName, repoRevision string) (build *contracts.Build, err error) {

	dbc.PrometheusOutboundAPICallTotals.With(prometheus.Labels{"target": "cockroachdb"}).Inc()

	rows, err := dbc.databaseConnection.Query(
		`
		SELECT
			id,
			repo_source,
			repo_owner,
			repo_name,
			repo_branch,
			repo_revision,
			build_version,
			build_status,
			labels,
			releases,
			manifest,
			commits,
			inserted_at,
			updated_at
		FROM
			builds
		WHERE
			repo_source=$1 AND
			repo_owner=$2 AND
			repo_name=$3 AND
			repo_revision=$4
		ORDER BY
			inserted_at DESC
		LIMIT 1
		OFFSET 0
		`,
		repoSource,
		repoOwner,
		repoName,
		repoRevision,
	)
	if err != nil {
		return
	}

	recordExists := false

	defer rows.Close()
	recordExists = rows.Next()

	if !recordExists {
		return
	}

	var labelsData, releasesData, commitsData []uint8

	build = &contracts.Build{}

	if err := rows.Scan(
		&build.ID,
		&build.RepoSource,
		&build.RepoOwner,
		&build.RepoName,
		&build.RepoBranch,
		&build.RepoRevision,
		&build.BuildVersion,
		&build.BuildStatus,
		&labelsData,
		&releasesData,
		&build.Manifest,
		&commitsData,
		&build.InsertedAt,
		&build.UpdatedAt); err != nil {
		return nil, err
	}

	if len(labelsData) > 0 {
		if err = json.Unmarshal(labelsData, &build.Labels); err != nil {
			return nil, err
		}
	}
	if len(releasesData) > 0 {
		if err = json.Unmarshal(releasesData, &build.Releases); err != nil {
			return
		}
	}
	if len(commitsData) > 0 {
		if err = json.Unmarshal(commitsData, &build.Commits); err != nil {
			return nil, err
		}
	}

	// unmarshal then marshal manifest to include defaults
	var manifest manifest.EstafetteManifest
	err = yaml.Unmarshal([]byte(build.Manifest), &manifest)
	if err == nil {
		manifestWithDefaultBytes, err := yaml.Marshal(manifest)
		if err == nil {
			build.ManifestWithDefaults = string(manifestWithDefaultBytes)
		} else {
			log.Warn().Err(err).Interface("manifest", manifest).Msgf("Marshalling manifest for %v/%v/%v revision %v failed", build.RepoSource, build.RepoOwner, build.RepoName, build.RepoRevision)
		}
	} else {
		log.Warn().Err(err).Str("manifest", build.Manifest).Msgf("Unmarshalling manifest for %v/%v/%v revision %v failed", build.RepoSource, build.RepoOwner, build.RepoName, build.RepoRevision)
	}

	return
}

func (dbc *cockroachDBClientImpl) GetPipelineBuildByID(repoSource, repoOwner, repoName string, id int) (build *contracts.Build, err error) {

	dbc.PrometheusOutboundAPICallTotals.With(prometheus.Labels{"target": "cockroachdb"}).Inc()

	rows, err := dbc.databaseConnection.Query(
		`
		SELECT
			id,
			repo_source,
			repo_owner,
			repo_name,
			repo_branch,
			repo_revision,
			build_version,
			build_status,
			labels,
			releases,
			manifest,
			commits,
			inserted_at,
			updated_at
		FROM
			builds
		WHERE
			repo_source=$1 AND
			repo_owner=$2 AND
			repo_name=$3 AND
			id=$4
		ORDER BY
			inserted_at DESC
		LIMIT 1
		OFFSET 0
		`,
		repoSource,
		repoOwner,
		repoName,
		id,
	)
	if err != nil {
		return
	}

	recordExists := false

	defer rows.Close()
	recordExists = rows.Next()

	if !recordExists {
		return
	}

	var labelsData, releasesData, commitsData []uint8

	build = &contracts.Build{}

	if err := rows.Scan(
		&build.ID,
		&build.RepoSource,
		&build.RepoOwner,
		&build.RepoName,
		&build.RepoBranch,
		&build.RepoRevision,
		&build.BuildVersion,
		&build.BuildStatus,
		&labelsData,
		&releasesData,
		&build.Manifest,
		&commitsData,
		&build.InsertedAt,
		&build.UpdatedAt); err != nil {
		return nil, err
	}

	if len(labelsData) > 0 {
		if err = json.Unmarshal(labelsData, &build.Labels); err != nil {
			return nil, err
		}
	}
	if len(releasesData) > 0 {
		if err = json.Unmarshal(releasesData, &build.Releases); err != nil {
			return
		}
	}
	if len(commitsData) > 0 {
		if err = json.Unmarshal(commitsData, &build.Commits); err != nil {
			return nil, err
		}
	}

	// unmarshal then marshal manifest to include defaults
	var manifest manifest.EstafetteManifest
	err = yaml.Unmarshal([]byte(build.Manifest), &manifest)
	if err == nil {
		manifestWithDefaultBytes, err := yaml.Marshal(manifest)
		if err == nil {
			build.ManifestWithDefaults = string(manifestWithDefaultBytes)
		} else {
			log.Warn().Err(err).Interface("manifest", manifest).Msgf("Marshalling manifest for %v/%v/%v revision %v failed", build.RepoSource, build.RepoOwner, build.RepoName, build.RepoRevision)
		}
	} else {
		log.Warn().Err(err).Str("manifest", build.Manifest).Msgf("Unmarshalling manifest for %v/%v/%v revision %v failed", build.RepoSource, build.RepoOwner, build.RepoName, build.RepoRevision)
	}

	return
}

func (dbc *cockroachDBClientImpl) GetPipelineBuildByVersion(repoSource, repoOwner, repoName, buildVersion string) (build *contracts.Build, err error) {

	dbc.PrometheusOutboundAPICallTotals.With(prometheus.Labels{"target": "cockroachdb"}).Inc()

	rows, err := dbc.databaseConnection.Query(
		`
		SELECT
			id,
			repo_source,
			repo_owner,
			repo_name,
			repo_branch,
			repo_revision,
			build_version,
			build_status,
			labels,
			releases,
			manifest,
			commits,
			inserted_at,
			updated_at
		FROM
			builds
		WHERE
			repo_source=$1 AND
			repo_owner=$2 AND
			repo_name=$3 AND
			build_version=$4
		ORDER BY
			inserted_at DESC
		LIMIT 1
		OFFSET 0
		`,
		repoSource,
		repoOwner,
		repoName,
		buildVersion,
	)
	if err != nil {
		return
	}

	recordExists := false

	defer rows.Close()
	recordExists = rows.Next()

	if !recordExists {
		return
	}

	var labelsData, releasesData, commitsData []uint8

	build = &contracts.Build{}

	if err := rows.Scan(
		&build.ID,
		&build.RepoSource,
		&build.RepoOwner,
		&build.RepoName,
		&build.RepoBranch,
		&build.RepoRevision,
		&build.BuildVersion,
		&build.BuildStatus,
		&labelsData,
		&releasesData,
		&build.Manifest,
		&commitsData,
		&build.InsertedAt,
		&build.UpdatedAt); err != nil {
		return nil, err
	}

	if len(labelsData) > 0 {
		if err = json.Unmarshal(labelsData, &build.Labels); err != nil {
			return nil, err
		}
	}
	if len(releasesData) > 0 {
		if err = json.Unmarshal(releasesData, &build.Releases); err != nil {
			return
		}
	}
	if len(commitsData) > 0 {
		if err = json.Unmarshal(commitsData, &build.Commits); err != nil {
			return nil, err
		}
	}

	// unmarshal then marshal manifest to include defaults
	var manifest manifest.EstafetteManifest
	err = yaml.Unmarshal([]byte(build.Manifest), &manifest)
	if err == nil {
		manifestWithDefaultBytes, err := yaml.Marshal(manifest)
		if err == nil {
			build.ManifestWithDefaults = string(manifestWithDefaultBytes)
		} else {
			log.Warn().Err(err).Interface("manifest", manifest).Msgf("Marshalling manifest for %v/%v/%v revision %v failed", build.RepoSource, build.RepoOwner, build.RepoName, build.RepoRevision)
		}
	} else {
		log.Warn().Err(err).Str("manifest", build.Manifest).Msgf("Unmarshalling manifest for %v/%v/%v revision %v failed", build.RepoSource, build.RepoOwner, build.RepoName, build.RepoRevision)
	}

	return
}

func (dbc *cockroachDBClientImpl) GetPipelineBuildLogs(repoSource, repoOwner, repoName, repoBranch, repoRevision string) (buildLog *contracts.BuildLog, err error) {

	dbc.PrometheusOutboundAPICallTotals.With(prometheus.Labels{"target": "cockroachdb"}).Inc()

	rows, err := dbc.databaseConnection.Query(
		`
		SELECT
			id,
			repo_source,
			repo_owner,
			repo_name,
			repo_branch,
			repo_revision,
			steps,
			inserted_at
		FROM
			build_logs_v2
		WHERE
			repo_source=$1 AND
			repo_owner=$2 AND
			repo_name=$3 AND
			repo_branch=$4 AND
			repo_revision=$5
		LIMIT 1
		`,
		repoSource,
		repoOwner,
		repoName,
		repoBranch,
		repoRevision,
	)
	if err != nil {
		return
	}

	recordExists := false

	defer rows.Close()
	recordExists = rows.Next()

	if !recordExists {
		return
	}

	buildLog = &contracts.BuildLog{}

	var stepsData []uint8

	if err = rows.Scan(&buildLog.ID, &buildLog.RepoSource, &buildLog.RepoOwner, &buildLog.RepoName, &buildLog.RepoBranch, &buildLog.RepoRevision, &stepsData, &buildLog.InsertedAt); err != nil {
		return
	}

	if err = json.Unmarshal(stepsData, &buildLog.Steps); err != nil {
		return
	}

	return
}

func (dbc *cockroachDBClientImpl) GetPipelineReleases(repoSource, repoOwner, repoName string, pageNumber, pageSize int) (releases []*contracts.Release, err error) {

	dbc.PrometheusOutboundAPICallTotals.With(prometheus.Labels{"target": "cockroachdb"}).Inc()

	releases = make([]*contracts.Release, 0)

	rows, err := dbc.databaseConnection.Query(
		`
		SELECT
			id,
			repo_source,
			repo_owner,
			repo_name,
			release,
			release_version,
			release_status,
			triggered_by,
			inserted_at,
			updated_at
		FROM
			releases
		WHERE
			repo_source=$1 AND
			repo_owner=$2 AND
			repo_name=$3
		ORDER BY
			inserted_at DESC
		LIMIT $4
		OFFSET $5
		`,
		repoSource,
		repoOwner,
		repoName,
		pageSize,
		(pageNumber-1)*pageSize,
	)
	if err != nil {
		return
	}

	defer rows.Close()
	for rows.Next() {

		release := contracts.Release{}
		var id int

		if err := rows.Scan(
			&id,
			&release.RepoSource,
			&release.RepoOwner,
			&release.RepoName,
			&release.Name,
			&release.ReleaseVersion,
			&release.ReleaseStatus,
			&release.TriggeredBy,
			&release.InsertedAt,
			&release.UpdatedAt); err != nil {
			return nil, err
		}

		release.ID = strconv.Itoa(id)

		releases = append(releases, &release)
	}

	return
}

func (dbc *cockroachDBClientImpl) GetPipelineReleasesCount(repoSource, repoOwner, repoName string) (totalCount int, err error) {

	dbc.PrometheusOutboundAPICallTotals.With(prometheus.Labels{"target": "cockroachdb"}).Inc()

	row := dbc.databaseConnection.QueryRow(
		`
		SELECT
			COUNT(*)
		FROM
			releases
		WHERE
			repo_source=$1 AND
			repo_owner=$2 AND
			repo_name=$3
		`,
		repoSource,
		repoOwner,
		repoName,
	)

	if err := row.Scan(&totalCount); err != nil {
		return 0, err
	}

	return
}

func (dbc *cockroachDBClientImpl) GetPipelineRelease(repoSource, repoOwner, repoName string, id int) (release *contracts.Release, err error) {

	dbc.PrometheusOutboundAPICallTotals.With(prometheus.Labels{"target": "cockroachdb"}).Inc()

	row := dbc.databaseConnection.QueryRow(
		`
		SELECT
			id,
			repo_source,
			repo_owner,
			repo_name,
			release,
			release_version,
			release_status,
			triggered_by,
			inserted_at,
			updated_at
		FROM
			releases
		WHERE
			id=$1 AND
			repo_source=$2 AND
			repo_owner=$3 AND
			repo_name=$4
		LIMIT 1
		OFFSET 0
		`,
		id,
		repoSource,
		repoOwner,
		repoName,
	)

	release = &contracts.Release{}

	if err := row.Scan(
		&id,
		&release.RepoSource,
		&release.RepoOwner,
		&release.RepoName,
		&release.Name,
		&release.ReleaseVersion,
		&release.ReleaseStatus,
		&release.TriggeredBy,
		&release.InsertedAt,
		&release.UpdatedAt); err != nil {
		return nil, err
	}
	release.ID = strconv.Itoa(id)

	return
}

func (dbc *cockroachDBClientImpl) GetPipelineLastReleaseByName(repoSource, repoOwner, repoName, releaseName string) (release *contracts.Release, err error) {

	dbc.PrometheusOutboundAPICallTotals.With(prometheus.Labels{"target": "cockroachdb"}).Inc()

	rows, err := dbc.databaseConnection.Query(
		`
		SELECT
			id,
			repo_source,
			repo_owner,
			repo_name,
			release,
			release_version,
			release_status,
			triggered_by,
			inserted_at,
			updated_at
		FROM
			releases
		WHERE
			release=$1 AND
			repo_source=$2 AND
			repo_owner=$3 AND
			repo_name=$4
		ORDER BY
			inserted_at DESC
		LIMIT 1
		OFFSET 0
		`,
		releaseName,
		repoSource,
		repoOwner,
		repoName,
	)
	if err != nil {
		return
	}

	recordExists := false

	defer rows.Close()
	recordExists = rows.Next()

	if !recordExists {
		return
	}

	release = &contracts.Release{}
	var id int

	if err := rows.Scan(
		&id,
		&release.RepoSource,
		&release.RepoOwner,
		&release.RepoName,
		&release.Name,
		&release.ReleaseVersion,
		&release.ReleaseStatus,
		&release.TriggeredBy,
		&release.InsertedAt,
		&release.UpdatedAt); err != nil {
		return nil, err
	}
	release.ID = strconv.Itoa(id)

	return
}

func (dbc *cockroachDBClientImpl) GetPipelineReleaseLogs(repoSource, repoOwner, repoName string, id int) (releaseLog *contracts.ReleaseLog, err error) {

	dbc.PrometheusOutboundAPICallTotals.With(prometheus.Labels{"target": "cockroachdb"}).Inc()

	rows, err := dbc.databaseConnection.Query(
		`
		SELECT
			id,
			repo_source,
			repo_owner,
			repo_name,
			release_id,
			steps,
			inserted_at
		FROM
			release_logs
		WHERE
			repo_source=$1 AND
			repo_owner=$2 AND
			repo_name=$3 AND
			release_id=$4
		LIMIT 1
		`,
		repoSource,
		repoOwner,
		repoName,
		id,
	)
	if err != nil {
		return
	}

	recordExists := false

	defer rows.Close()
	recordExists = rows.Next()

	if !recordExists {
		return
	}

	releaseLog = &contracts.ReleaseLog{}

	var stepsData []uint8
	var releaseID int

	if err = rows.Scan(
		&releaseLog.ID,
		&releaseLog.RepoSource,
		&releaseLog.RepoOwner,
		&releaseLog.RepoName,
		&releaseID,
		&stepsData,
		&releaseLog.InsertedAt); err != nil {
		return
	}

	releaseLog.ReleaseID = strconv.Itoa(releaseID)

	if err = json.Unmarshal(stepsData, &releaseLog.Steps); err != nil {
		return
	}

	return
}

func (dbc *cockroachDBClientImpl) InsertBuildLog(buildLog contracts.BuildLog) (err error) {

	dbc.PrometheusOutboundAPICallTotals.With(prometheus.Labels{"target": "cockroachdb"}).Inc()

	bytes, err := json.Marshal(buildLog.Steps)
	if err != nil {
		return
	}

	// insert logs
	_, err = dbc.databaseConnection.Exec(
		`
		INSERT INTO
			build_logs_v2
		(
			repo_source,
			repo_owner,
			repo_name,
			repo_branch,
			repo_revision,
			steps
		)
		VALUES
		(
			$1,
			$2,
			$3,
			$4,
			$5,
			$6
		)
		`,
		buildLog.RepoSource,
		buildLog.RepoOwner,
		buildLog.RepoName,
		buildLog.RepoBranch,
		buildLog.RepoRevision,
		bytes,
	)

	if err != nil {
		return
	}

	return
}

func (dbc *cockroachDBClientImpl) InsertReleaseLog(releaseLog contracts.ReleaseLog) (err error) {

	dbc.PrometheusOutboundAPICallTotals.With(prometheus.Labels{"target": "cockroachdb"}).Inc()

	bytes, err := json.Marshal(releaseLog.Steps)
	if err != nil {
		return
	}

	releaseID, err := strconv.Atoi(releaseLog.ReleaseID)
	if err != nil {
		return err
	}

	// insert logs
	_, err = dbc.databaseConnection.Exec(
		`
		INSERT INTO
			release_logs
		(
			repo_source,
			repo_owner,
			repo_name,
			release_id,
			steps
		)
		VALUES
		(
			$1,
			$2,
			$3,
			$4,
			$5
		)
		`,
		releaseLog.RepoSource,
		releaseLog.RepoOwner,
		releaseLog.RepoName,
		releaseID,
		bytes,
	)

	if err != nil {
		return
	}

	return
}

func (dbc *cockroachDBClientImpl) GetBuildsCount(filters map[string][]string) (totalCount int, err error) {

	dbc.PrometheusOutboundAPICallTotals.With(prometheus.Labels{"target": "cockroachdb"}).Inc()

	psql := sq.StatementBuilder.PlaceholderFormat(sq.Dollar)

	query :=
		psql.
			Select("COUNT(*)").
			From("builds")

	query, err = whereClauseGeneratorForAllFilters(query, filters)
	if err != nil {
		return
	}

	row := query.RunWith(dbc.databaseConnection).QueryRow()

	if err := row.Scan(&totalCount); err != nil {
		return 0, err
	}

	return
}

func (dbc *cockroachDBClientImpl) GetReleasesCount(filters map[string][]string) (totalCount int, err error) {

	dbc.PrometheusOutboundAPICallTotals.With(prometheus.Labels{"target": "cockroachdb"}).Inc()

	psql := sq.StatementBuilder.PlaceholderFormat(sq.Dollar)

	query :=
		psql.
			Select("COUNT(*)").
			From("releases")

	query, err = whereClauseGeneratorForAllReleaseFilters(query, filters)
	if err != nil {
		return
	}

	row := query.RunWith(dbc.databaseConnection).QueryRow()

	if err := row.Scan(&totalCount); err != nil {
		return 0, err
	}

	return
}

func (dbc *cockroachDBClientImpl) GetBuildsDuration(filters map[string][]string) (totalDuration time.Duration, err error) {

	dbc.PrometheusOutboundAPICallTotals.With(prometheus.Labels{"target": "cockroachdb"}).Inc()

	psql := sq.StatementBuilder.PlaceholderFormat(sq.Dollar)

	query :=
		psql.
			Select("SUM(AGE(updated_at,inserted_at))::string").
			From("builds")

	query, err = whereClauseGeneratorForAllFilters(query, filters)
	if err != nil {
		return
	}

	row := query.RunWith(dbc.databaseConnection).QueryRow()

	var totalDurationAsString string

	if err := row.Scan(&totalDurationAsString); err != nil {
		return 0, err
	}

	totalDuration, err = time.ParseDuration(totalDurationAsString)
	if err != nil {
		return
	}

	return
}

func whereClauseGeneratorForAllFilters(query sq.SelectBuilder, filters map[string][]string) (sq.SelectBuilder, error) {

	query, err := whereClauseGeneratorForStatusFilter(query, filters)
	if err != nil {
		return query, err
	}
	query, err = whereClauseGeneratorForSinceFilter(query, filters)
	if err != nil {
		return query, err
	}
	query, err = whereClauseGeneratorForLabelsFilter(query, filters)
	if err != nil {
		return query, err
	}

	return query, nil
}

func whereClauseGeneratorForAllReleaseFilters(query sq.SelectBuilder, filters map[string][]string) (sq.SelectBuilder, error) {

	query, err := whereClauseGeneratorForReleaseStatusFilter(query, filters)
	if err != nil {
		return query, err
	}
	query, err = whereClauseGeneratorForSinceFilter(query, filters)
	if err != nil {
		return query, err
	}

	return query, nil
}

func whereClauseGeneratorForStatusFilter(query sq.SelectBuilder, filters map[string][]string) (sq.SelectBuilder, error) {

	if statuses, ok := filters["status"]; ok && len(statuses) > 0 && statuses[0] != "all" {
		query = query.Where(sq.Eq{"build_status": statuses})
	}

	return query, nil
}

func whereClauseGeneratorForReleaseStatusFilter(query sq.SelectBuilder, filters map[string][]string) (sq.SelectBuilder, error) {

	if statuses, ok := filters["status"]; ok && len(statuses) > 0 && statuses[0] != "all" {
		query = query.Where(sq.Eq{"release_status": statuses})
	}

	return query, nil
}

func whereClauseGeneratorForSinceFilter(query sq.SelectBuilder, filters map[string][]string) (sq.SelectBuilder, error) {

	if since, ok := filters["since"]; ok && len(since) > 0 && since[0] != "eternity" {
		sinceValue := since[0]
		switch sinceValue {
		case "1d":
			query = query.Where(sq.GtOrEq{"inserted_at": time.Now().AddDate(0, 0, -1)})
		case "1w":
			query = query.Where(sq.GtOrEq{"inserted_at": time.Now().AddDate(0, 0, -7)})
		case "1m":
			query = query.Where(sq.GtOrEq{"inserted_at": time.Now().AddDate(0, -1, 0)})
		case "1y":
			query = query.Where(sq.GtOrEq{"inserted_at": time.Now().AddDate(-1, 0, 0)})
		}
	}

	return query, nil
}

func whereClauseGeneratorForLabelsFilter(query sq.SelectBuilder, filters map[string][]string) (sq.SelectBuilder, error) {

	if labels, ok := filters["labels"]; ok && len(labels) > 0 {

		labelsParam := []contracts.Label{}

		for _, label := range labels {
			keyValuePair := strings.Split(label, "=")

			if len(keyValuePair) == 2 {
				labelsParam = append(labelsParam, contracts.Label{
					Key:   keyValuePair[0],
					Value: keyValuePair[1],
				})
			}
		}

		if len(labelsParam) > 0 {
			bytes, err := json.Marshal(labelsParam)
			if err != nil {
				return query, err
			}

			query = query.Where("labels @> ?", string(bytes))
		}
	}

	return query, nil
}
