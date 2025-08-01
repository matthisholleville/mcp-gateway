package migrate

import (
	"database/sql"
	"fmt"
	"testing"
	"time"

	testFixtures "github.com/matthisholleville/mcp-gateway/internal/storage/testsFixtures"
	"github.com/matthisholleville/mcp-gateway/pkg/logger"
	"github.com/stretchr/testify/assert"
)

func setupFixtures(t *testing.T, engine string) (string, logger.Logger, error) {
	logger := logger.MustNewLogger("json", "debug", "")
	switch engine {
	case "postgres":
		postgresOpts := &testFixtures.PostgresTestContainerOptions{}
		db := testFixtures.NewPostgresTestContainer(postgresOpts).RunPostgresTestContainer(t)
		return db.GetConnectionURI(true), logger, nil
	case "memory":
		return "", logger, nil
	default:
		return "", logger, fmt.Errorf("invalid engine: %s", engine)
	}
}

func TestMigrateUpDownDrop(t *testing.T) {
	type EngineConfig struct {
		Engine string
	}

	engines := []EngineConfig{
		{
			Engine: "postgres",
		},
		{
			Engine: "memory",
		},
	}

	for _, engine := range engines {
		uri, logger, err := setupFixtures(t, engine.Engine)
		assert.NoError(t, err)

		cfg := &MigrationConfig{
			Engine:       engine.Engine,
			URI:          uri,
			Logger:       logger,
			Timeout:      10 * time.Second,
			Verbose:      true,
			MigrationDir: "../../../assets/migrations/postgres",
		}

		err = RunMigrations(*cfg)
		assert.NoError(t, err)

		if engine.Engine == "memory" {
			continue
		}

		// verify the database is not empty
		db, err := sql.Open(engine.Engine, uri)
		assert.NoError(t, err)
		defer db.Close()
		var count int
		err = db.QueryRow("SELECT COUNT(*) FROM mcp_gateway.proxy").Scan(&count)
		assert.NoError(t, err)

		// drop
		cfg.Drop = true
		err = RunMigrations(*cfg)
		assert.NoError(t, err)

		db, err = sql.Open(engine.Engine, uri)
		assert.NoError(t, err)
		defer db.Close()

		// check if the database is empty
		err = db.QueryRow("SELECT COUNT(*) FROM mcp_gateway.proxy").Scan(&count)
		assert.Error(t, err)
	}

}
