package migrate

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	_ "github.com/lib/pq"
	"github.com/matthisholleville/mcp-gateway/pkg/logger"
	"go.uber.org/zap"
)

type MigrationConfig struct {
	Engine       string
	URI          string
	Logger       logger.Logger
	Timeout      time.Duration
	Verbose      bool
	Version      int
	Drop         bool
	MigrationDir string
}

func RunMigrations(cfg MigrationConfig) error {

	var m *migrate.Migrate
	var err error
	fmt.Println("cfg.Engine", cfg.Engine)
	switch cfg.Engine {
	case "memory":
		cfg.Logger.Info("no migrations to run for memory engine")
		return nil
	case "postgres":
		db, err := sql.Open("postgres", cfg.URI)
		if err != nil {
			return fmt.Errorf("failed to create migrate instance: %w", err)
		}
		driver, err := postgres.WithInstance(db, &postgres.Config{
			MigrationsTable: "migrations",
			SchemaName:      "public",
		})
		if err != nil {
			return fmt.Errorf("failed to create migrate instance: %w", err)
		}
		if cfg.MigrationDir == "" {
			cfg.MigrationDir = "assets/migrations/postgres"
		}
		m, err = migrate.NewWithDatabaseInstance(
			fmt.Sprintf("file://%s", cfg.MigrationDir),
			"postgres", driver)
		if err != nil {
			return fmt.Errorf("failed to create migrate instance: %w", err)
		}
	}

	m.Log = cfg.Logger
	m.LockTimeout = cfg.Timeout
	m.PrefetchMigrations = 100

	if cfg.Drop {
		cfg.Logger.Info("down all migrations")

		err = m.Down()
		if err != nil {
			cfg.Logger.Error("failed to down migrations", zap.Error(err))
		}

		return m.Drop()
	}

	currentVersion, dirty, err := m.Version()
	if err != nil && err != migrate.ErrNilVersion {
		return fmt.Errorf("failed to get current version: %w", err)
	}

	if dirty && cfg.Version == 0 {
		return fmt.Errorf("database is dirty, please run migrations to a specific version")
	}

	if dirty {
		cfg.Logger.Info("database is dirty, forcing migration to", zap.Int("version", int(currentVersion)))
		err = m.Force(int(currentVersion))
		if err != nil {
			return fmt.Errorf("failed to force migration: %w", err)
		}
		cfg.Logger.Info("migration completed")
		return nil
	}

	if currentVersion == 0 {
		cfg.Logger.Info("running all migrations")
		err = m.Up()
		if err != nil {
			return fmt.Errorf("failed to run migrations: %w", err)
		}
		cfg.Logger.Info("migrations completed")
		return nil
	}

	cfg.Logger.Info("migration to", zap.Int("version", cfg.Version))

	switch {
	case cfg.Version > int(currentVersion):
		cfg.Logger.Info("running migrations to", zap.Int("version", cfg.Version))
		err = m.Migrate(uint(cfg.Version))
		if err != nil {
			return fmt.Errorf("failed to run migrations: %w", err)
		}
		cfg.Logger.Info("migrations completed")
	case cfg.Version < int(currentVersion):
		cfg.Logger.Info("running migrations down to", zap.Int("version", cfg.Version))
		err = m.Migrate(uint(cfg.Version))
		if err != nil {
			return fmt.Errorf("failed to run migrations: %w", err)
		}
		cfg.Logger.Info("migrations completed")
	default:
		cfg.Logger.Info("migrations already at target version", zap.Int("version", cfg.Version))
	}

	defer m.Close()

	cfg.Logger.Info("migrations done")
	return nil
}
