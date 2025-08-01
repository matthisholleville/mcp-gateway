// Package main is the main package for the MCP Gateway.
package main

import (
	"os"

	"github.com/matthisholleville/mcp-gateway/cmd"
	"github.com/matthisholleville/mcp-gateway/cmd/migrate"
	"github.com/matthisholleville/mcp-gateway/cmd/serve"
)

func main() {
	rootCmd := cmd.NewRootCommand()

	rootCmd.AddCommand(serve.NewRunCommand())
	rootCmd.AddCommand(migrate.NewMigrateCommand())

	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
