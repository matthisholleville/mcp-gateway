package migrate

import (
	"github.com/matthisholleville/mcp-gateway/cmd/util"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

// bindRunFlagsFunc binds the run flags to the command.
func bindRunFlagsFunc(flags *pflag.FlagSet) func(*cobra.Command, []string) {
	return func(_ *cobra.Command, _ []string) {
		util.MustBindPFlag(backendEngineFlag, flags.Lookup(backendEngineFlag))
		util.MustBindEnv(backendEngineFlag, "MCP_GATEWAY_BACKEND_ENGINE")

		util.MustBindPFlag(backendURIFlag, flags.Lookup(backendURIFlag))
		util.MustBindEnv("backend-uri", "MCP_GATEWAY_BACKEND_URI")

		util.MustBindPFlag(verboseMigrationFlag, flags.Lookup(verboseMigrationFlag))
		util.MustBindEnv(verboseMigrationFlag, "MCP_GATEWAY_VERBOSE")

		util.MustBindPFlag(logFormatFlag, flags.Lookup(logFormatFlag))
		util.MustBindEnv(logFormatFlag, "MCP_GATEWAY_LOG_FORMAT")

		util.MustBindPFlag(logLevelFlag, flags.Lookup(logLevelFlag))
		util.MustBindEnv(logLevelFlag, "MCP_GATEWAY_LOG_LEVEL")

		util.MustBindPFlag(logTimestampFlag, flags.Lookup(logTimestampFlag))
		util.MustBindEnv(logTimestampFlag, "MCP_GATEWAY_LOG_TIMESTAMP_FORMAT")

		util.MustBindPFlag(targetVersionFlag, flags.Lookup(targetVersionFlag))
		util.MustBindEnv(targetVersionFlag, "MCP_GATEWAY_TARGET_VERSION")

		util.MustBindPFlag(timeoutFlag, flags.Lookup(timeoutFlag))
		util.MustBindEnv(timeoutFlag, "MCP_GATEWAY_TIMEOUT")

		util.MustBindPFlag(dropFlag, flags.Lookup(dropFlag))
		util.MustBindEnv(dropFlag, "MCP_GATEWAY_DROP")

		util.MustBindPFlag(dirFlag, flags.Lookup(dirFlag))
		util.MustBindEnv(dirFlag, "MCP_GATEWAY_MIGRATION_DIR")
	}
}
