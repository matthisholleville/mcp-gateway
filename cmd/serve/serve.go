// Package serve provides the command to serve the api.
package serve

import (
	"context"
	"log/slog"
	"os"
	"time"

	"github.com/matthisholleville/mcp-gateway/internal/server"
	"github.com/matthisholleville/mcp-gateway/pkg/signals"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

const (
	defaultPort                  = 8082
	defaultConcurrency           = 10
	defaultServerShutdownTimeout = 30 * time.Second
)

var (
	port                  int
	serverShutdownTimeout time.Duration
	concurrency           int
)

// ServeCmd is the command to serve the api.
var ServeCmd = &cobra.Command{
	Use:   "serve",
	Short: "Serve the api",
	Long:  `Serve the api`,
	Run: func(_ *cobra.Command, _ []string) {
		logger, ok := viper.Get("logger").(*slog.Logger)
		if !ok {
			panic("Logger not found in viper")
		}
		viper.Set("concurrency", concurrency)

		logger.InfoContext(context.Background(), "Serve "+"sre-kit-app")

		server, err := server.NewServer(logger, "0.0.0.0", port, serverShutdownTimeout)
		if err != nil {
			logger.ErrorContext(context.Background(), "Failed to create server: "+err.Error())
			os.Exit(1)
		}

		err = server.ListenAndServe()
		if err != nil {
			logger.ErrorContext(context.Background(), "Failed to start server: "+err.Error())
			os.Exit(1)
		}
		live, ready := server.GetHealthStatus()

		// graceful shutdown
		stopCh := signals.SetupSignalHandler()
		sd, _ := signals.NewShutdown(serverShutdownTimeout, *logger)
		sd.Graceful(stopCh, server.GetRouter(), live, ready)

	},
}

//nolint:gochecknoinits // We need to initialize the flags
func init() {
	ServeCmd.Flags().DurationVar(&serverShutdownTimeout, "server-shutdown-timeout", defaultServerShutdownTimeout, "Server shutdown timeout")
	ServeCmd.Flags().IntVar(&port, "port", defaultPort, "Port to listen on")
	ServeCmd.Flags().IntVar(&concurrency, "concurrency", defaultConcurrency, "Concurrency for the server")
}
