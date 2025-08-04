// Package migrate provides a migration engine for the MCP Gateway.
package migrate

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file" // import file source
	_ "github.com/lib/pq"                                // import postgres driver
	"github.com/matthisholleville/mcp-gateway/internal/storage/utils"
	"github.com/matthisholleville/mcp-gateway/pkg/logger"
	"go.uber.org/zap"
)

// MigrationConfig bundles every parameter needed to run a migration session.
type MigrationConfig struct {
	Engine       string        // "memory", "postgres", ...
	URI          string        // connection string for the target database
	Username     string        // username for the target database
	Password     string        // password for the target database
	Logger       logger.Logger // structured logger implementation
	Timeout      time.Duration // advisory lock timeout
	Verbose      bool          // enable verbose output on migrate CLI
	Version      int           // target version (0 means "latest")
	Drop         bool          // drop all objects before migrating
	MigrationDir string        // filesystem path that contains *.sql files
}

// RunMigrations orchestrates the migration workflow according to cfg.
func RunMigrations(cfg *MigrationConfig) error {
	m, err := newMigrator(cfg)
	if err != nil {
		return err
	}
	// m == nil means the selected engine does not require migrations (e.g. memory).
	if m == nil {
		return nil
	}
	defer m.Close() //nolint:errcheck // nothing interesting to do with the error

	switch {
	case cfg.Drop:
		return applyDrop(m, cfg.Logger)

	case cfg.Version == 0:
		// No explicit version: migrate to the most recent.
		return applyUp(m, cfg.Logger)

	default:
		// A specific target version was requested.
		return applyVersion(m, cfg.Version, cfg.Logger)
	}
}

// newMigrator returns a ready‑to‑use migrate.Migrate instance or nil for
// engines that do not require migrations. All instance‑level settings such
// as the logger, lock timeout and prefetch size are configured here.
func newMigrator(cfg *MigrationConfig) (*migrate.Migrate, error) {
	switch cfg.Engine {
	case "memory":
		cfg.Logger.Debug("no migrations to run for memory engine")
		return nil, nil

	case "postgres":
		if cfg.MigrationDir == "" {
			cfg.MigrationDir = "assets/migrations/postgres"
		}

		uri, err := utils.GetURI(cfg.Username, cfg.Password, cfg.URI)
		if err != nil {
			return nil, fmt.Errorf("get uri: %w", err)
		}

		db, err := sql.Open("postgres", uri)
		if err != nil {
			return nil, fmt.Errorf("open database: %w", err)
		}

		driver, err := postgres.WithInstance(db, &postgres.Config{
			MigrationsTable: "migrations",
			SchemaName:      "public",
		})
		if err != nil {
			return nil, fmt.Errorf("create driver: %w", err)
		}

		m, err := migrate.NewWithDatabaseInstance(
			"file://"+cfg.MigrationDir,
			"postgres",
			driver,
		)
		if err != nil {
			return nil, fmt.Errorf("create migrator: %w", err)
		}

		m.Log = cfg.Logger
		m.LockTimeout = cfg.Timeout
		m.PrefetchMigrations = 100
		return m, nil

	default:
		return nil, fmt.Errorf("unsupported engine %q", cfg.Engine)
	}
}

// applyDrop drops every migration then drops the schema itself.
// It is destructive and should only be used in development / CI.
func applyDrop(m *migrate.Migrate, log logger.Logger) error {
	log.Info("dropping all migrations")

	if err := m.Down(); err != nil && err != migrate.ErrNoChange {
		return fmt.Errorf("down: %w", err)
	}
	return m.Drop()
}

// applyUp migrates the database to the latest available version.
func applyUp(m *migrate.Migrate, log logger.Logger) error {
	log.Info("running all migrations (up)")

	if err := m.Up(); err != nil && err != migrate.ErrNoChange {
		return fmt.Errorf("up: %w", err)
	}
	log.Info("migrations completed")
	return nil
}

// applyVersion migrates up or down until the requested target version is reached.
func applyVersion(m *migrate.Migrate, target int, log logger.Logger) error {
	current, dirty, err := m.Version()
	if err != nil && err != migrate.ErrNilVersion {
		return fmt.Errorf("current version: %w", err)
	}

	if dirty {
		// For safety, we refuse to move a dirty database automatically.
		return fmt.Errorf("database is dirty at version %d, manual intervention required", current)
	}

	log.Info("migrating to version", zap.Int("target", target))

	targetUint := uint(target) //nolint:gosec // G115: migration versions are always small integers
	if err := m.Migrate(targetUint); err != nil && err != migrate.ErrNoChange {
		return fmt.Errorf("migrate to %d: %w", target, err)
	}

	log.Info("migrations completed")
	return nil
}
