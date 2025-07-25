// Package cfg provides the configuration for the application.
package cfg

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"reflect"

	"github.com/spf13/viper"
)

type Cors struct {
	Enabled          bool     `mapstructure:"enabled"`
	AllowedOrigins   []string `mapstructure:"allowed_origins"`
	AllowedMethods   []string `mapstructure:"allowed_methods"`
	AllowedHeaders   []string `mapstructure:"allowed_headers"`
	AllowCredentials bool     `mapstructure:"allow_credentials"`
}

type Cfg struct {
	Cors   Cors   `mapstructure:"cors"`
	OAuth  OAuth  `mapstructure:"oauth"`
	Server Server `mapstructure:"server"`
}

type Server struct {
	URL string `mapstructure:"url"`
}

type OAuth struct {
	Enabled                bool     `mapstructure:"enabled"`
	AuthorizationServers   []string `mapstructure:"authorization_servers"`
	BearerMethodsSupported []string `mapstructure:"bearer_methods_supported"`
	ScopesSupported        []string `mapstructure:"scopes_supported"`
}

func WriteInitConfiguration(logger *slog.Logger) {
	// Write the default configuration to a file
	if err := viper.SafeWriteConfig(); err != nil {
		var configFileAlreadyExistsErr viper.ConfigFileAlreadyExistsError
		if errors.As(err, &configFileAlreadyExistsErr) {
			logger.DebugContext(context.Background(), "Configuration file already exists. No changes made.")
		} else {
			logger.ErrorContext(context.Background(), fmt.Sprintf("Unable to write configuration file: %v", err))
			os.Exit(1)
		}
	} else {
		logger.InfoContext(context.Background(), "Default configuration written successfully")
	}
}

func LoadCfg(logger *slog.Logger) *Cfg {
	var config Cfg
	err := viper.Unmarshal(&config)
	if err != nil {
		logger.ErrorContext(context.Background(), "error on fetching configuration file")
		os.Exit(1)
	}

	if reflect.DeepEqual(config, Cfg{}) {
		logger.ErrorContext(context.Background(), "Configuration is empty")
		os.Exit(1)
	}

	return &config
}
