package serve

import (
	"github.com/matthisholleville/mcp-gateway/cmd/util"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

func bindServeFlagsFunc(flags *pflag.FlagSet) func(*cobra.Command, []string) {
	return func(cmd *cobra.Command, args []string) {
		util.MustBindPFlag("http-addr", flags.Lookup("http-addr"))
		util.MustBindEnv("http-addr", "MCP_GATEWAY_HTTP_ADDR")

		util.MustBindPFlag("log.format", flags.Lookup("log-format"))
		util.MustBindEnv("log.format", "MCP_GATEWAY_LOG_FORMAT")

		util.MustBindPFlag("log.level", flags.Lookup("log-level"))
		util.MustBindEnv("log.level", "MCP_GATEWAY_LOG_LEVEL")

		util.MustBindPFlag("log.timestamp-format", flags.Lookup("log-timestamp-format"))
		util.MustBindEnv("log.timestamp-format", "MCP_GATEWAY_LOG_TIMESTAMP_FORMAT")

		util.MustBindPFlag("proxy.cache-ttl", flags.Lookup("proxy-cache-ttl"))
		util.MustBindEnv("proxy.cache-ttl", "MCP_GATEWAY_PROXY_CACHE_TTL")

		util.MustBindPFlag("proxy.heartbeat.interval", flags.Lookup("proxy-heartbeat-interval"))
		util.MustBindEnv("proxy.heartbeat.interval", "MCP_GATEWAY_PROXY_HEARTBEAT_INTERVAL")

		util.MustBindPFlag("oauth.enabled", flags.Lookup("oauth-enabled"))
		util.MustBindEnv("oauth.enabled", "MCP_GATEWAY_OAUTH_ENABLED")

		util.MustBindPFlag("oauth.authorizationServers", flags.Lookup("oauth-authorization-servers"))
		util.MustBindEnv("oauth.authorizationServers", "MCP_GATEWAY_OAUTH_AUTHORIZATION_SERVERS")

		util.MustBindPFlag("oauth.resource", flags.Lookup("oauth-resource"))
		util.MustBindEnv("oauth.resource", "MCP_GATEWAY_OAUTH_RESOURCE")

		util.MustBindPFlag("oauth.bearerMethodsSupported", flags.Lookup("oauth-bearer-methods-supported"))
		util.MustBindEnv("oauthConfig.bearerMethodsSupported", "MCP_GATEWAY_OAUTH_BEARER_METHODS_SUPPORTED")

		util.MustBindPFlag("oauth.scopesSupported", flags.Lookup("oauth-scopes-supported"))
		util.MustBindEnv("oauth.scopesSupported", "MCP_GATEWAY_OAUTH_SCOPES_SUPPORTED")

		util.MustBindPFlag("authProvider.enabled", flags.Lookup("auth-provider-enabled"))
		util.MustBindEnv("authProvider.enabled", "MCP_GATEWAY_AUTH_PROVIDER_ENABLED")

		cmd.MarkFlagsRequiredTogether("auth-provider-enabled", "auth-provider-name", "oauth-enabled", "oauth-authorization-servers", "oauth-bearer-methods-supported", "oauth-scopes-supported", "oauth-resource")

		util.MustBindPFlag("authProvider.name", flags.Lookup("auth-provider-name"))
		util.MustBindEnv("authProvider.name", "MCP_GATEWAY_AUTH_PROVIDER_NAME")

		util.MustBindPFlag("backendConfig.engine", flags.Lookup("backend-engine"))
		util.MustBindEnv("backendConfig.engine", "MCP_GATEWAY_BACKEND_ENGINE")

		util.MustBindPFlag("backendConfig.uri", flags.Lookup("backend-uri"))
		util.MustBindEnv("backendConfig.uri", "MCP_GATEWAY_BACKEND_URI")

		util.MustBindPFlag("backendConfig.maxOpenConns", flags.Lookup("backend-max-open-conns"))
		util.MustBindEnv("backendConfig.maxOpenConns", "MCP_GATEWAY_BACKEND_MAX_OPEN_CONNS")

		util.MustBindPFlag("backendConfig.maxIdleConns", flags.Lookup("backend-max-idle-conns"))
		util.MustBindEnv("backendConfig.maxIdleConns", "MCP_GATEWAY_BACKEND_MAX_IDLE_CONNS")

		util.MustBindPFlag("backendConfig.connMaxIdleTime", flags.Lookup("backend-conn-max-idle-time"))
		util.MustBindEnv("backendConfig.connMaxIdleTime", "MCP_GATEWAY_BACKEND_CONN_MAX_IDLE_TIME")

		util.MustBindPFlag("backendConfig.connMaxLifetime", flags.Lookup("backend-conn-max-lifetime"))
		util.MustBindEnv("backendConfig.connMaxLifetime", "MCP_GATEWAY_BACKEND_CONN_MAX_LIFETIME")

		util.MustBindPFlag("authProvider.firebase.projectId", flags.Lookup("firebase-project-id"))
		util.MustBindEnv("authProvider.firebase.projectId", "MCP_GATEWAY_FIREBASE_PROJECT_ID")

		util.MustBindPFlag("authProvider.okta.issuer", flags.Lookup("okta-issuer"))
		util.MustBindEnv("authProvider.okta.issuer", "MCP_GATEWAY_OKTA_ISSUER")

		util.MustBindPFlag("authProvider.okta.orgUrl", flags.Lookup("okta-org-url"))
		util.MustBindEnv("authProvider.okta.orgUrl", "MCP_GATEWAY_OKTA_ORG_URL")

		util.MustBindPFlag("authProvider.okta.clientId", flags.Lookup("okta-client-id"))
		util.MustBindEnv("authProvider.okta.clientId", "MCP_GATEWAY_OKTA_CLIENT_ID")

		util.MustBindPFlag("authProvider.okta.privateKey", flags.Lookup("okta-private-key"))
		util.MustBindEnv("authProvider.okta.privateKey", "MCP_GATEWAY_OKTA_PRIVATE_KEY")

		util.MustBindPFlag("authProvider.okta.privateKeyId", flags.Lookup("okta-private-key-id"))
		util.MustBindEnv("authProvider.okta.privateKeyId", "MCP_GATEWAY_OKTA_PRIVATE_KEY_ID")

		cmd.MarkFlagsRequiredTogether("okta-private-key", "okta-private-key-id", "okta-client-id", "okta-org-url", "okta-issuer")

		util.MustBindPFlag("http.adminApiKey", flags.Lookup("http-admin-api-key"))
		util.MustBindEnv("http.adminApiKey", "MCP_GATEWAY_HTTP_ADMIN_API_KEY")

	}
}
