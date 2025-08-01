package signals

import (
	"context"
	"log/slog"
	"sync/atomic"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/spf13/viper"
)

const (
	preStopSleep = 3 * time.Second
)

// Shutdown is a struct that contains the logger and the server shutdown timeout.
type Shutdown struct {
	logger                slog.Logger
	serverShutdownTimeout time.Duration
}

// NewShutdown creates a new Shutdown instance.
func NewShutdown(serverShutdownTimeout time.Duration, logger slog.Logger) (*Shutdown, error) {
	srv := &Shutdown{
		logger:                logger,
		serverShutdownTimeout: serverShutdownTimeout,
	}

	return srv, nil
}

// Graceful shuts down the MCP Gateway gracefully.
func (s *Shutdown) Graceful(stopCh <-chan struct{}, httpServer *echo.Echo, healthy, ready *int32) {
	ctx := context.Background()

	// wait for SIGTERM or SIGINT
	<-stopCh
	ctx, cancel := context.WithTimeout(ctx, s.serverShutdownTimeout)
	defer cancel()

	// all calls to /healthz and /readyz will fail from now on
	atomic.StoreInt32(healthy, 0)
	atomic.StoreInt32(ready, 0)

	//nolint:noctx // no need to pass a context here
	s.logger.Info("Shutting down HTTP server", slog.Duration("timeout", s.serverShutdownTimeout))

	// There could be a period where a terminating pod may still receive requests. Implementing a brief wait can mitigate this.
	// See: https://kubernetes.io/docs/concepts/workloads/pods/pod-lifecycle/#pod-termination
	// the readiness check interval must be lower than the timeout
	if viper.GetString("level") != "debug" {
		time.Sleep(preStopSleep)
	}

	// determine if the http server was started
	if httpServer != nil {
		if err := httpServer.Shutdown(ctx); err != nil {
			//nolint:noctx // no need to pass a context here
			s.logger.Warn("HTTP server graceful shutdown failed", slog.Any("error", err))
		}
	}
}
