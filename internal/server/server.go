package server

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"sync/atomic"
	"time"

	"github.com/google/uuid"
	"github.com/labstack/echo-contrib/echoprometheus"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"github.com/matthisholleville/mcp-gateway/internal/cfg"
	"github.com/matthisholleville/mcp-gateway/internal/metrics"
	"github.com/matthisholleville/mcp-gateway/internal/oauth"
	"github.com/matthisholleville/mcp-gateway/internal/proxy"
)

type Server struct {
	Router          *echo.Echo
	Logger          *slog.Logger
	Host            string
	Port            int
	ShutdownTimeout time.Duration
	Cfg             *cfg.Cfg
	Live            *int32
	Ready           *int32
}

func NewServer(
	logger *slog.Logger,
	host string,
	port int,
	shutdownTimeout time.Duration,
) (*Server, error) {
	router := echo.New()
	s := &Server{
		Logger:          logger,
		Host:            host,
		Port:            port,
		ShutdownTimeout: shutdownTimeout,
		Cfg:             cfg.LoadCfg(logger),
		Router:          router,
	}

	s.configureRouter()
	s.configureMetrics()
	s.registerHealthcheckRoutes()
	s.withCORSMiddleware()
	s.configureAuthMiddleware()
	s.withOAuthProtectedResources()
	s.configureMCP()
	return s, nil
}

// ListenAndServe starts the server
func (s *Server) ListenAndServe() error {
	s.Logger.Info("Starting server", slog.String("host", s.Host), slog.Int("port", s.Port))
	return s.Router.Start(s.Host + ":" + strconv.Itoa(s.Port))
}

func (s *Server) GetRouter() *echo.Echo {
	return s.Router
}

func (s *Server) GetHealthStatus() (*int32, *int32) {
	return s.Live, s.Ready
}

// configureServer configures the server
func (s *Server) configureRouter() {
	s.Router.HideBanner = true
	s.Router.HidePort = true
	s.Router.Host(s.Host)
}

// registerHealthcheckRoutes registers the healthcheck routes
func (s *Server) registerHealthcheckRoutes() {
	s.Live = new(int32)
	s.Ready = new(int32)
	*s.Live = 1
	*s.Ready = 1

	s.Router.GET("/live", echo.HandlerFunc(func(c echo.Context) error {
		if atomic.LoadInt32(s.Live) == 1 {
			return echo.NewHTTPError(http.StatusOK, "OK")
		}
		return echo.NewHTTPError(http.StatusServiceUnavailable, "KO")
	}))
	s.Router.GET("/ready", echo.HandlerFunc(func(c echo.Context) error {
		if atomic.LoadInt32(s.Ready) == 1 {
			return echo.NewHTTPError(http.StatusOK, "OK")
		}
		return echo.NewHTTPError(http.StatusServiceUnavailable, "KO")
	}))
}

// WithCORSMiddleware adds CORS middleware to the router
func (s *Server) withCORSMiddleware() {
	if !s.Cfg.Cors.Enabled {
		s.Logger.Warn("CORS is disabled")
		return
	}

	s.Router.Use(middleware.CORSWithConfig(middleware.CORSConfig{
		AllowOrigins: s.Cfg.Cors.AllowedOrigins,
		AllowMethods: s.Cfg.Cors.AllowedMethods,
		AllowHeaders: s.Cfg.Cors.AllowedHeaders,
	}))
}

// withOAuthProtectedResources adds OAuth protected resources to the router
func (s *Server) withOAuthProtectedResources() {
	if !s.Cfg.OAuth.Enabled {
		s.Logger.Warn("OAuth is disabled")
		return
	}

	meta := map[string]any{
		"resource":                 s.Cfg.Server.URL,
		"authorization_servers":    s.Cfg.OAuth.AuthorizationServers,
		"bearer_methods_supported": s.Cfg.OAuth.BearerMethodsSupported,
		"scopes_supported":         s.Cfg.OAuth.ScopesSupported,
	}
	wellKnown := func(c echo.Context) error {
		c.Response().Header().Set("Content-Type", "application/json")
		return c.JSON(http.StatusOK, meta)
	}

	s.Router.GET("/.well-known/oauth-protected-resource", wellKnown)
	s.Router.HEAD("/.well-known/oauth-protected-resource", wellKnown)
	s.Router.OPTIONS("/.well-known/oauth-protected-resource", func(c echo.Context) error {
		return c.NoContent(http.StatusNoContent)
	})
}

// configureMetrics configures the metrics endpoint
func (s *Server) configureMetrics() {
	customMetrics := metrics.NewMetrics()
	err := customMetrics.RegisterCustomMetrics()
	if err != nil {
		s.Logger.ErrorContext(context.Background(), "Failed to register metrics", "error", err)
	}
	s.Router.GET("/metrics", echoprometheus.NewHandler())

}

// configureMCP configures the MCP endpoint
func (s *Server) configureMCP() {
	mcpServer := server.NewMCPServer(
		"MCP Gateway",
		"1.0.0",
		server.WithToolCapabilities(true),
		server.WithHooks(s.mcpHooks()),
	)

	serverConfig := server.NewStreamableHTTPServer(
		mcpServer,
		server.WithEndpointPath("/mcp"),
		server.WithHTTPContextFunc(s.addGlobalMCPContext),
		server.WithStateLess(true),
	)

	s.addProxyTools(mcpServer)

	s.Router.GET("/mcp", echo.WrapHandler(serverConfig))
	s.Router.HEAD("/mcp", echo.WrapHandler(serverConfig))
	s.Router.OPTIONS("/mcp", func(c echo.Context) error {
		return c.NoContent(http.StatusNoContent)
	})
	s.Router.POST("/mcp", echo.WrapHandler(serverConfig))
}

func (s *Server) addProxyTools(mcpServer *server.MCPServer) {
	mcpProxy, err := proxy.NewProxy(&s.Cfg.Proxy, s.Logger)
	if err != nil {
		s.Logger.ErrorContext(context.Background(), "Failed to create MCP proxy", slog.Any("err", err))
		panic(err)
	}
	for _, proxy := range *mcpProxy {
		proxyTools, err := proxy.GetTools()
		if err != nil {
			s.Logger.ErrorContext(context.Background(), "Failed to get MCP proxy tools", slog.Any("err", err))
			continue
		}
		for i := range proxyTools {
			tool := proxyTools[i]
			toolName := proxy.GetName() + ":" + tool.Name
			tool.Name = toolName
			mcpServer.AddTool(tool, proxy.CallTool)
		}
	}
}

// mcpHooks configures the MCP hooks
func (s *Server) mcpHooks() *server.Hooks {
	hooks := &server.Hooks{}

	hooks.AddBeforeCallTool(func(ctx context.Context, id any, message *mcp.CallToolRequest) {
		logger, ok := ctx.Value("logger").(*slog.Logger)
		if !ok {
			s.Logger.ErrorContext(ctx, "Logger not found in context")
			return
		}
		logger.InfoContext(ctx, "Tool call started", "request_id", id)
		method := message.Method
		params := message.Params
		args := message.GetArguments()
		proxyName, toolName := s.parseToolName(params.Name)
		metrics.ToolsCalledGauge.WithLabelValues(toolName, proxyName).Inc()
		logger.DebugContext(ctx, "Tool call started", "request_method", method, "tool_name", params.Name, "request_arguments", args)
	})

	hooks.AddAfterCallTool(func(ctx context.Context, id any, message *mcp.CallToolRequest, result *mcp.CallToolResult) {
		logger, ok := ctx.Value("logger").(*slog.Logger)
		if !ok {
			s.Logger.ErrorContext(ctx, "Logger not found in context")
			return
		}
		response := "N/A"
		if len(result.Content) > 0 {
			textContent, ok := result.Content[0].(mcp.TextContent)
			if ok {
				response = textContent.Text
			}
		}
		idFloat, ok := id.(float64)
		if !ok {
			logger.ErrorContext(ctx, "Invalid request ID", slog.Any("request_id", id))
		}
		proxyName, toolName := s.parseToolName(message.Params.Name)
		if result.IsError {
			logger.ErrorContext(ctx, response, slog.String("toolName", message.Params.Name), slog.Float64("request_id", idFloat))
			metrics.ToolsCallErrorsGauge.WithLabelValues(toolName, proxyName).Inc()
		} else {
			logger.InfoContext(
				ctx,
				"Tool call completed with success",
				slog.String("toolName", message.Params.Name),
				slog.Float64("request_id", idFloat),
			)
			metrics.ToolsCallSuccessGauge.WithLabelValues(toolName, proxyName).Inc()
		}
	})

	hooks.AddBeforeListTools(func(ctx context.Context, id any, _ *mcp.ListToolsRequest) {
		logger, ok := ctx.Value("logger").(*slog.Logger)
		if !ok {
			s.Logger.ErrorContext(ctx, "Logger not found in context")
			return
		}
		logger.InfoContext(ctx, "Before List Tools Hook", "request_id", id)
		metrics.ListToolsGauge.WithLabelValues("").Inc()
	})

	return hooks
}

func (s *Server) parseToolName(toolName string) (string, string) {
	parts := strings.Split(toolName, ":")
	if len(parts) != 2 {
		return toolName, ""
	}
	return parts[0], parts[1]
}

// addGlobalMCPContext adds the global MCP context to the context
func (s *Server) addGlobalMCPContext(ctx context.Context, r *http.Request) context.Context {
	for key, values := range r.Header {
		if len(values) > 0 {
			//nolint:staticcheck,revive // We need to use the key as a string
			ctx = context.WithValue(ctx, key, values[0])
		}
	}
	correlationID := uuid.New().String()
	logger := s.Logger.With(slog.String("correlation_id", correlationID))
	//nolint:staticcheck,revive // We need to use the key as a string
	ctx = context.WithValue(ctx, "logger", logger)

	return ctx
}

func (s *Server) configureAuthMiddleware() {
	s.Router.Use(func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {

			isMCPPath := c.Path() == "/mcp" && c.Request().Method == "POST"
			if !isMCPPath {
				return next(c)
			}

			token := c.Request().Header.Get("Authorization")
			if token == "" {
				return s.unauth(c, "missing_token", "Missing token")
			}
			token = strings.TrimPrefix(token, "Bearer ")
			provider, err := oauth.NewProvider(s.Cfg.OAuth.Provider, s.Cfg, s.Logger)
			if err != nil {
				return s.unauth(c, "invalid_token", "Invalid token")
			}
			jwtToken, err := provider.VerifyToken(token)
			if err != nil {
				return s.unauth(c, "invalid_token", "Invalid token")
			}

			const maxBodySize = 1 << 20 // 1 MiB

			req := c.Request()
			body := req.Body
			req.Body = http.MaxBytesReader(c.Response(), body, maxBodySize)

			var copyBuf bytes.Buffer
			tee := io.TeeReader(req.Body, &copyBuf)

			dec := json.NewDecoder(tee)

			message := &mcp.CallToolRequest{}
			err = dec.Decode(message)
			if err != nil {
				s.Logger.ErrorContext(c.Request().Context(), "Failed to unmarshal request body", "error", err)
				return s.unauth(c, "invalid_request", "Invalid request")
			}

			req.Body = io.NopCloser(&copyBuf)

			if message.Method != "tools/call" {
				return next(c)
			}

			hasPermission := provider.VerifyPermissions(message.Params.Name, jwtToken.Claims)
			if !hasPermission {
				return s.unauth(c, "insufficient_scope", "Insufficient scope")
			}

			c.Set("claims", jwtToken.Claims)
			return next(c)
		}
	})

}

func (s *Server) unauth(c echo.Context, code, msg string) error {
	rsMetaURL := s.Cfg.OAuth.AuthorizationServers[0] + "/.well-known/oauth-protected-resource"
	c.Response().Header().Set("WWW-Authenticate",
		fmt.Sprintf(`Bearer resource_metadata=%q, error=%q`, rsMetaURL, code))
	return echo.NewHTTPError(http.StatusUnauthorized, msg)
}
