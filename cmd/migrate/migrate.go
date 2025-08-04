// Package migrate provides a command to run the MCP Gateway migrations.
package migrate

import (
	"time"

	"github.com/matthisholleville/mcp-gateway/internal/cfg"
	"github.com/matthisholleville/mcp-gateway/internal/storage/migrate"
	"github.com/matthisholleville/mcp-gateway/pkg/logger"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

const (
	backendEngineFlag    = "backend-engine"
	backendURIFlag       = "backend-uri"
	backendUsernameFlag  = "backend-username"
	backendPasswordFlag  = "backend-password"
	logFormatFlag        = "log-format"
	logLevelFlag         = "log-level"
	logTimestampFlag     = "log-timestamp-format"
	targetVersionFlag    = "target-version"
	verboseMigrationFlag = "verbose"
	timeoutFlag          = "timeout"
	dropFlag             = "drop"
	dirFlag              = "dir"

	defaultTimeout = 30 * time.Second
	defaultVersion = 0
)

// NewMigrateCommand creates a new migrate command.
func NewMigrateCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "migrate",
		Short: "Run the MCP Gateway migrations",
		Long:  "Run the MCP Gateway migrations.",
		RunE:  runMigration,
		Args:  cobra.NoArgs,
	}
	defaultConfig := cfg.DefaultConfig()
	flags := cmd.Flags()

	flags.String(backendEngineFlag, defaultConfig.BackendConfig.Engine, "(required) The engine to use for the auth backend")

	flags.String(backendURIFlag, defaultConfig.BackendConfig.URI, "(required) The URI to use for the auth backend")

	flags.String(backendUsernameFlag, defaultConfig.BackendConfig.Username, "The username to use for the auth backend")

	flags.String(backendPasswordFlag, defaultConfig.BackendConfig.Password, "The password to use for the auth backend")

	flags.Bool(verboseMigrationFlag, false, "enable verbose migration logs (default false)")

	flags.String(logFormatFlag, defaultConfig.Log.Format, "The format to use for logging")

	flags.String(logLevelFlag, defaultConfig.Log.Level, "The level to use for logging")

	flags.String(logTimestampFlag, defaultConfig.Log.TimestampFormat, "The format to use for logging timestamps")

	flags.Int(targetVersionFlag, defaultVersion, "The target version to migrate to (default 0)")

	flags.Duration(timeoutFlag, defaultTimeout, "The timeout to use for the migration")

	flags.Bool(dropFlag, false, "Drop all migrations")

	flags.String(dirFlag, "", "The directory to use for the migrations")

	cmd.PreRun = bindRunFlagsFunc(flags)

	return cmd
}

func runMigration(_ *cobra.Command, _ []string) error {
	engine := viper.GetString(backendEngineFlag)
	uri := viper.GetString(backendURIFlag)
	username := viper.GetString(backendUsernameFlag)
	password := viper.GetString(backendPasswordFlag)
	verbose := viper.GetBool(verboseMigrationFlag)
	logFormat := viper.GetString(logFormatFlag)
	logLevel := viper.GetString(logLevelFlag)
	logTimestamp := viper.GetString(logTimestampFlag)
	targetVersion := viper.GetInt(targetVersionFlag)
	timeout := viper.GetDuration(timeoutFlag)
	drop := viper.GetBool(dropFlag)

	log := logger.MustNewLogger(logFormat, logLevel, logTimestamp)

	config := migrate.MigrationConfig{
		Engine:   engine,
		URI:      uri,
		Username: username,
		Password: password,
		Version:  targetVersion,
		Timeout:  timeout,
		Logger:   log,
		Verbose:  verbose,
		Drop:     drop,
	}

	return migrate.RunMigrations(&config)
}
