package cfg

import (
	"fmt"
	"time"
)

type Config struct {
	HTTP          *HTTPConfig
	Log           *LogConfig
	OAuth         *OAuthConfig
	Proxy         *ProxyConfig
	AuthProvider  *AuthProviderConfig
	BackendConfig *BackendConfig
}

type HTTPConfig struct {
	Addr        string
	CORS        *CORSConfig
	AdminAPIKey string
}

type LogConfig struct {
	// Format is the log format to use in the log output (e.g. 'text' or 'json')
	Format string

	// Level is the log level to use in the log output (e.g. 'none', 'debug', or 'info')
	Level string

	// Format of the timestamp in the log output (e.g. 'Unix'(default) or 'ISO8601')
	TimestampFormat string
}

type ProxyConfig struct {
	CacheTTL  time.Duration
	Heartbeat *HeartbeatConfig
}

type HeartbeatConfig struct {
	Enabled         bool
	IntervalSeconds time.Duration
}
type CORSConfig struct {
	Enabled          bool
	AllowedOrigins   []string
	AllowedMethods   []string
	AllowedHeaders   []string
	AllowCredentials bool
}

type OAuthConfig struct {
	Enabled                bool
	AuthorizationServers   []string
	BearerMethodsSupported []string
	ScopesSupported        []string
}

type AuthProviderConfig struct {
	Enabled  bool
	Name     string
	Firebase *FirebaseConfig
	Okta     *OktaConfig
}

type FirebaseConfig struct {
	ProjectID string
}

type OktaConfig struct {
	Issuer       string
	OrgURL       string
	ClientID     string
	PrivateKey   string `json:"-"` // private field, won't be logged
	PrivateKeyID string `json:"-"` // private field, won't be logged
}

type BackendConfig struct {
	// Engine is the auth backend engine to use (e.g. 'memory', 'postgres')
	Engine string
	URI    string `json:"-"` // private field, won't be logged

	// MaxOpenConns is the maximum number of open connections to the database.
	MaxOpenConns int

	// MaxIdleConns is the maximum number of connections to the datastore in the idle connection
	// pool.
	MaxIdleConns int

	// ConnMaxIdleTime is the maximum amount of time a connection to the datastore may be idle.
	ConnMaxIdleTime time.Duration

	// ConnMaxLifetime is the maximum amount of time a connection to the datastore may be reused.
	ConnMaxLifetime time.Duration
}

func DefaultConfig() *Config {
	return &Config{
		HTTP: &HTTPConfig{
			Addr: ":8082",
			CORS: &CORSConfig{
				Enabled:          true,
				AllowedOrigins:   []string{"*"},
				AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE"},
				AllowedHeaders:   []string{"Content-Type", "Authorization"},
				AllowCredentials: true,
			},
			AdminAPIKey: "change-me",
		},
		Log: &LogConfig{
			Format: "text",
			Level:  "info",
		},
		Proxy: &ProxyConfig{
			CacheTTL: 10 * time.Second,
			Heartbeat: &HeartbeatConfig{
				Enabled:         true,
				IntervalSeconds: 10 * time.Second,
			},
		},
		OAuth: &OAuthConfig{
			Enabled: false,
		},
		AuthProvider: &AuthProviderConfig{
			Enabled: false,
			Name:    "",
			Firebase: &FirebaseConfig{
				ProjectID: "change-me",
			},
			Okta: &OktaConfig{
				Issuer: "",
				OrgURL: "",
			},
		},
		BackendConfig: &BackendConfig{
			Engine: "memory",
		},
	}
}

func (cfg *Config) Verify() error {

	if cfg.Proxy.CacheTTL <= 5*time.Second {
		return fmt.Errorf("proxy cache TTL must be greater than 5 seconds")
	}

	if cfg.Proxy.Heartbeat.IntervalSeconds <= 5*time.Second {
		return fmt.Errorf("proxy heartbeat interval must be greater than 5 seconds")
	}

	return nil
}
