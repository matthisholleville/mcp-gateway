// Package cmd provides the root command for the application.
package cmd

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// NewRootCommand creates a new root command.
func NewRootCommand() *cobra.Command {
	programName := "MCP Gateway"

	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.SetEnvPrefix("MCP_GATEWAY")
	viper.SetEnvKeyReplacer(strings.NewReplacer("-", "_"))
	viper.AutomaticEnv()

	configPaths := []string{"/etc/mcp-gateway", "$HOME/.mcp-gateway", "./config"}
	for _, path := range configPaths {
		viper.AddConfigPath(path)
	}

	err := viper.ReadInConfig()
	if err != nil {
		panic(fmt.Sprintf("unable to read config file: %s", err))
	}

	return &cobra.Command{
		Use:   programName,
		Short: "A proxy gateway for MCP servers",
		Long:  `MCP Gateway is a flexible and extensible proxy gateway for MCP servers, with built-in support for middleware, permissions, rate limiting, and observability.`,
	}
}
