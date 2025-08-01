package testfixtures

import (
	"context"
	"database/sql"
	"fmt"
	"io"
	"strings"
	"testing"
	"time"

	"github.com/cenkalti/backoff/v4"
	"github.com/containerd/errdefs"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/image"
	"github.com/docker/docker/client"
	"github.com/docker/go-connections/nat"
	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file" // need to import the file source
	_ "github.com/jackc/pgx/v5/stdlib"                   // need to import the PostgreSQL driver.
	_ "github.com/lib/pq"                                // need to import the PostgreSQL driver.
	"github.com/oklog/ulid/v2"
	"github.com/stretchr/testify/require"
)

const (
	postgresImage  = "postgres:17"
	maxElapsedTime = 30 * time.Second
)

// PostgresTestContainerOptions are the options for the PostgresTestContainer.
type PostgresTestContainerOptions struct {
	MigrationsDir string
}

// PostgresTestContainer is a test container for Postgres.
type PostgresTestContainer struct {
	addr     string
	version  int64
	username string
	password string
	opts     *PostgresTestContainerOptions
}

// NewPostgresTestContainer returns an implementation of the DatastoreTestContainer interface
// for Postgres.
func NewPostgresTestContainer(opts *PostgresTestContainerOptions) *PostgresTestContainer {
	return &PostgresTestContainer{
		opts: opts,
	}
}

func (p *PostgresTestContainer) GetDatabaseSchemaVersion() int64 {
	return p.version
}

// RunPostgresTestContainer runs a Postgres container, connects to it, and returns a
// bootstrapped implementation of the DatastoreTestContainer interface wired up for the
// Postgres datastore engine.
func (p *PostgresTestContainer) RunPostgresTestContainer(t testing.TB) *PostgresTestContainer {
	dockerClient, err := client.NewClientWithOpts(
		client.FromEnv,
		client.WithAPIVersionNegotiation(),
	)
	require.NoError(t, err)
	t.Cleanup(func() {
		dockerClient.Close() //nolint:errcheck // no need to check the error here
	})

	allImages, err := dockerClient.ImageList(context.Background(), image.ListOptions{
		All: true,
	})
	require.NoError(t, err)

	foundPostgresImage := false

AllImages:
	for i := range allImages {
		dockerImage := allImages[i]
		for _, tag := range dockerImage.RepoTags {
			if strings.Contains(tag, postgresImage) {
				foundPostgresImage = true
				break AllImages
			}
		}
	}

	if !foundPostgresImage {
		reader, err := dockerClient.ImagePull(context.Background(), postgresImage, image.PullOptions{})
		require.NoError(t, err)

		_, err = io.Copy(io.Discard, reader) // consume the image pull output to make sure it's done
		require.NoError(t, err)
	}

	containerCfg := container.Config{
		Env: []string{
			"POSTGRES_DB=defaultdb",
			"POSTGRES_PASSWORD=secret",
		},
		ExposedPorts: nat.PortSet{
			nat.Port("5432/tcp"): {},
		},
		Image: postgresImage,
	}

	hostCfg := container.HostConfig{
		AutoRemove:      true,
		PublishAllPorts: true,
		Tmpfs:           map[string]string{"/var/lib/postgresql/data": ""},
	}

	name := "postgres-" + ulid.Make().String()

	cont, err := dockerClient.ContainerCreate(context.Background(), &containerCfg, &hostCfg, nil, nil, name)
	require.NoError(t, err, "failed to create postgres docker container")

	t.Cleanup(func() {
		timeoutSec := 5

		err := dockerClient.ContainerStop(context.Background(), cont.ID, container.StopOptions{Timeout: &timeoutSec})
		if err != nil && !errdefs.IsNotFound(err) {
			t.Logf("failed to stop postgres container: %v", err)
		}
	})

	err = dockerClient.ContainerStart(context.Background(), cont.ID, container.StartOptions{})
	require.NoError(t, err, "failed to start postgres container")

	containerJSON, err := dockerClient.ContainerInspect(context.Background(), cont.ID)
	require.NoError(t, err)

	m, ok := containerJSON.NetworkSettings.Ports["5432/tcp"]
	if !ok || len(m) == 0 {
		require.Fail(t, "failed to get host port mapping from postgres container")
	}

	pgTestContainer := &PostgresTestContainer{
		addr:     "localhost:" + m[0].HostPort,
		username: "postgres",
		password: "secret",
	}

	uri := fmt.Sprintf(
		"postgres://%s:%s@%s/defaultdb?sslmode=disable",
		pgTestContainer.username,
		pgTestContainer.password,
		pgTestContainer.addr,
	)

	db, err := sql.Open("postgres", uri)
	require.NoError(t, err)

	// Wait for the database to be ready before creating the driver
	backoffPolicy := backoff.NewExponentialBackOff()
	backoffPolicy.MaxElapsedTime = maxElapsedTime
	err = backoff.Retry(
		func() error {
			return db.PingContext(context.Background())
		},
		backoffPolicy,
	)
	require.NoError(t, err, "failed to connect to postgres container")

	p.executeMigrationsIfNeeded(t, db)

	return pgTestContainer
}

// GetConnectionURI returns the postgres connection uri for the running postgres test container.
func (p *PostgresTestContainer) GetConnectionURI(includeCredentials bool) string {
	creds := ""
	if includeCredentials {
		creds = fmt.Sprintf("%s:%s@", p.username, p.password)
	}

	return fmt.Sprintf(
		"postgres://%s%s/%s?sslmode=disable",
		creds,
		p.addr,
		"defaultdb",
	)
}

func (p *PostgresTestContainer) GetUsername() string {
	return p.username
}

func (p *PostgresTestContainer) GetPassword() string {
	return p.password
}

func (p *PostgresTestContainer) executeMigrationsIfNeeded(t testing.TB, db *sql.DB) {
	if p.opts.MigrationsDir != "" {
		t.Logf("executing migrations from %s", p.opts.MigrationsDir)
		driver, err := postgres.WithInstance(db, &postgres.Config{})
		require.NoError(t, err)
		migrateInstance, err := migrate.NewWithDatabaseInstance(
			fmt.Sprintf("file://%s", p.opts.MigrationsDir),
			"postgres", driver)
		require.NoError(t, err)
		err = migrateInstance.Up()
		require.NoError(t, err)

		version, _, err := migrateInstance.Version()
		require.NoError(t, err)
		p.version = int64(version) //nolint:gosec // G115: migration versions are always small integers
		t.Logf("migrations executed successfully, version: %d", p.version)
	}
}
