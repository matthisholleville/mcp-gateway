// Package cmd provides the root command for the application.
package cmd

import (
	"errors"
	"fmt"
	"log/slog"
	"os"
	"regexp"
	"strings"

	"github.com/adrg/xdg"
	"github.com/matthisholleville/mcp-gateway/cmd/serve"
	"github.com/matthisholleville/mcp-gateway/internal/cfg"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

const (
	programName = "MCP Gateway"
	cfgDir      = "mcp-gateway"
)

var (
	// LogFormat is the format of the log.
	LogFormat string
	// LogLevel is the level of the log.
	LogLevel string
	// cfgFile is the path to the configuration file.
	cfgFile string
	// cfgDirPath is the path to the configuration directory.
	cfgDirPath = fmt.Sprintf("%s/%s", xdg.ConfigHome, cfgDir)
)

var rootCmd = &cobra.Command{
	Use:   programName,
	Short: "A proxy gateway for MCP servers",
	Long:  `MCP Gateway is a flexible and extensible proxy gateway for MCP servers, with built-in support for middleware, permissions, rate limiting, and observability.`,
}

// Execute executes the root command.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
	}
}

//nolint:gochecknoinits // We need to initialize the config
func init() {
	cobra.OnInitialize(initConfig)
	rootCmd.AddCommand(serve.ServeCmd)
	rootCmd.PersistentFlags().StringVar(&LogFormat, "log-format", "raw", "Log format (raw|json)")
	rootCmd.PersistentFlags().StringVar(&LogLevel, "log-level", "info", "Log level (debug|info|warn|error|fatal|panic)")
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", fmt.Sprintf("config file (default is %s)", cfgDirPath))
}

func initLogger() *slog.Logger {

	var level slog.Level
	switch LogLevel {
	case "debug":
		level = slog.LevelDebug
	case "info":
		level = slog.LevelInfo
	case "warn":
		level = slog.LevelWarn
	case "error":
		level = slog.LevelError
	default:
		level = slog.LevelInfo
	}

	return slog.New(
		slog.NewTextHandler(
			os.Stdout,
			&slog.HandlerOptions{Level: level},
		),
	)
}

func initConfig() {
	logger := initLogger()

	if cfgFile != "" {
		// Use config file from the flag.
		viper.SetConfigFile(cfgFile)
	} else {
		viper.AddConfigPath(cfgDirPath)
		viper.SetConfigType("yaml")
		viper.SetConfigName(programName)

		// Set default values

		//nolint:gosec,mnd // We need to create the configuration directory with the default permissions
		err := os.MkdirAll(cfgDirPath, 0755)
		if err != nil {
			panic(fmt.Sprintf("Unable to create configuration directory: %v", err))
		}

		cfg.WriteInitConfiguration(logger)

		if err := viper.SafeWriteConfig(); err != nil {
			var configFileAlreadyExistsErr viper.ConfigFileAlreadyExistsError
			if !errors.As(err, &configFileAlreadyExistsErr) {
				panic(fmt.Sprintf("Unable to write configuration file: %v", err))
			}
		}
	}

	configContent := readAndSubstituteEnvVars()

	viper.SetConfigType("yaml")
	if err := viper.ReadConfig(strings.NewReader(configContent)); err != nil {
		panic(fmt.Sprintf("unable to read configuration: %s", err))
	}

	viper.Set("logger", logger)
}

func readAndSubstituteEnvVars() string {
	var configPath string
	if cfgFile != "" {
		configPath = cfgFile
	} else {
		configPath = fmt.Sprintf("%s/%s.yaml", cfgDirPath, programName)
	}

	content, err := os.ReadFile(configPath)
	if err != nil {
		panic(fmt.Sprintf("unable to read config file: %s", err))
	}

	return substituteEnvVarsInString(string(content))
}

// substituteEnvVarsInString remplace toutes les occurrences de ${VAR_NAME} par la valeur de la variable d'environnement
func substituteEnvVarsInString(content string) string {
	// Regex pour matcher ${VAR_NAME}
	re := regexp.MustCompile(`\$\{([^}]+)\}`)

	return re.ReplaceAllStringFunc(content, func(match string) string {
		// Extraire le nom de la variable (sans ${ et })
		varName := match[2 : len(match)-1]
		return GetEnvOrPanic(varName)
	})
}

// GetEnvOrPanic gets the environment variable or panics if it is not found.
func GetEnvOrPanic(env string) string {
	res := os.Getenv(env)
	if res == "" {
		panic("Mandatory env variable not found:" + env)
	}
	return res
}
