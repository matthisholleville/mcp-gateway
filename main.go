package main

import (
	"os"

	"github.com/matthisholleville/mcp-gateway/cmd"
	"github.com/matthisholleville/mcp-gateway/cmd/serve"
)

func main() {
	rootCmd := cmd.NewRootCommand()

	rootCmd.AddCommand(serve.NewRunCommand())

	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
