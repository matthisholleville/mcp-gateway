package serve

import (
	"fmt"

	"github.com/matthisholleville/mcp-gateway/internal/cfg"
	"github.com/matthisholleville/mcp-gateway/internal/server"
	"github.com/matthisholleville/mcp-gateway/pkg/logger"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func NewRunCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "serve",
		Short: "Run the MCP Gateway server",
		Long:  "Run the MCP Gateway server.",
		Run:   run,
		Args:  cobra.NoArgs,
	}

	defaultConfig := cfg.DefaultConfig()
	flags := cmd.Flags()

	flags.String("http-addr", defaultConfig.HTTP.Addr, "The address to listen on for HTTP requests")

	flags.String("log-format", defaultConfig.Log.Format, "The format to use for logging")

	flags.String("log-level", defaultConfig.Log.Level, "The level to use for logging")

	flags.String("log-timestamp-format", defaultConfig.Log.TimestampFormat, "The format to use for logging timestamps")

	flags.Duration("proxy-cache-ttl", defaultConfig.Proxy.CacheTTL, "The TTL for the proxy cache")

	flags.Duration("proxy-heartbeat-interval", defaultConfig.Proxy.Heartbeat.IntervalSeconds, "The interval for the proxy heartbeat")

	flags.Bool("oauth-enabled", defaultConfig.OAuth.Enabled, "Whether to enable OAuth")

	flags.StringSlice("oauth-authorization-servers", defaultConfig.OAuth.AuthorizationServers, "The authorization servers for OAuth")

	flags.String("oauth-resource", defaultConfig.OAuth.Resource, "The resource for OAuth")

	flags.StringSlice("oauth-bearer-methods-supported", defaultConfig.OAuth.BearerMethodsSupported, "The bearer methods supported for OAuth")

	flags.StringSlice("oauth-scopes-supported", defaultConfig.OAuth.ScopesSupported, "The scopes supported for OAuth")

	flags.Bool("auth-provider-enabled", defaultConfig.AuthProvider.Enabled, "Whether to enable the auth provider")

	flags.String("auth-provider-name", defaultConfig.AuthProvider.Name, "The name of the auth provider")

	flags.String("backend-engine", defaultConfig.BackendConfig.Engine, "The engine to use for the auth backend")

	flags.String("backend-uri", defaultConfig.BackendConfig.URI, "The URI to use for the auth backend")

	flags.Int("backend-max-open-conns", defaultConfig.BackendConfig.MaxOpenConns, "The maximum number of open connections to the database")

	flags.Int("backend-max-idle-conns", defaultConfig.BackendConfig.MaxIdleConns, "The maximum number of connections to the datastore in the idle connection pool")

	flags.Duration("backend-conn-max-idle-time", defaultConfig.BackendConfig.ConnMaxIdleTime, "The maximum amount of time a connection to the datastore may be idle")

	flags.Duration("backend-conn-max-lifetime", defaultConfig.BackendConfig.ConnMaxLifetime, "The maximum amount of time a connection to the datastore may be reused")

	flags.String("okta-issuer", defaultConfig.AuthProvider.Okta.Issuer, "The issuer for the Okta auth provider")

	flags.String("okta-org-url", defaultConfig.AuthProvider.Okta.OrgURL, "The org URL for the Okta auth provider")

	flags.String("okta-client-id", defaultConfig.AuthProvider.Okta.ClientID, "The client ID for the Okta auth provider")

	flags.String("okta-private-key", defaultConfig.AuthProvider.Okta.PrivateKey, "The private key for the Okta auth provider")

	flags.String("okta-private-key-id", defaultConfig.AuthProvider.Okta.PrivateKeyID, "The private key ID for the Okta auth provider")

	flags.String("http-admin-api-key", defaultConfig.HTTP.AdminAPIKey, "The admin API key for the HTTP server. Using to configure the MCP Gateway API.")

	cmd.PreRun = bindServeFlagsFunc(flags)

	return cmd
}

func ReadConfig() (*cfg.Config, error) {
	config := cfg.DefaultConfig()

	viper.SetTypeByDefaultValue(true)
	err := viper.ReadInConfig()
	if err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil, fmt.Errorf("failed to load server config: %w", err)
		}
	}

	if err := viper.Unmarshal(config); err != nil {
		return nil, fmt.Errorf("failed to unmarshal server config: %w", err)
	}

	return config, nil
}

func run(_ *cobra.Command, _ []string) {
	config, err := ReadConfig()
	if err != nil {
		panic(err)
	}

	if err := config.Verify(); err != nil {
		panic(err)
	}
	logger := logger.MustNewLogger(config.Log.Format, config.Log.Level, config.Log.TimestampFormat)
	server, err := server.NewServer(logger, config)
	if err != nil {
		panic(err)
	}
	err = server.ListenAndServe()
	if err != nil {
		panic(err)
	}

	// serverCtx := &server.Server{Logger: logger}
	// if err := serverCtx.Run(context.Background(), config); err != nil {
	// 	panic(err)
	// }
}
