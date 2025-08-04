// Package server provides a server for the MCP Gateway.
package server

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"sync/atomic"
	"time"

	"github.com/google/uuid"
	"github.com/labstack/echo-contrib/echoprometheus"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"github.com/matthisholleville/mcp-gateway/internal/auth"
	"github.com/matthisholleville/mcp-gateway/internal/cfg"
	"github.com/matthisholleville/mcp-gateway/internal/metrics"
	"github.com/matthisholleville/mcp-gateway/internal/proxy"
	"github.com/matthisholleville/mcp-gateway/internal/storage"
	"github.com/matthisholleville/mcp-gateway/pkg/aescipher"
	"github.com/matthisholleville/mcp-gateway/pkg/logger"
	_ "github.com/matthisholleville/mcp-gateway/swagger" // We need to import the swagger documentation
	echoSwagger "github.com/swaggo/echo-swagger"
	"go.uber.org/zap"
)

//	@title			MCP Gateway API
//	@version		1.0
//	@description	This is the MCP Gateway API documentation.

//	@contact.name	Source Code
//	@contact.url	https://github.com/matthisholleville/mcp-gateway

//	@securitydefinitions.apikey Authentication
//	@in header
//	@name X-API-Key

//	@BasePath	/
//	@schemes	http https

type Server struct {
	Router    *echo.Echo
	Logger    logger.Logger
	Config    *cfg.Config
	Live      *int32
	Ready     *int32
	Storage   storage.Interface
	Encryptor aescipher.Cryptor
	Provider  auth.Provider
}

func NewServer(
	log logger.Logger,
	config *cfg.Config,

) (*Server, error) {
	router := echo.New()
	s := &Server{
		Logger: log,
		Config: config,
		Router: router,
	}

	s.configureRouter()
	s.configureEncryption()
	s.configureStorage()
	s.configureMetrics()
	s.registerHealthcheckRoutes()
	s.withCORSMiddleware()
	s.configureSwaggerRoutes()
	s.configureV1Routes()
	s.configureAuthMiddleware()
	s.withOAuthProtectedResources()
	s.configureMCP()
	return s, nil
}

// ListenAndServe starts the server
func (s *Server) ListenAndServe() error {
	s.Logger.Info("Starting server", zap.String("host", s.Config.HTTP.Addr))
	return s.Router.Start(s.Config.HTTP.Addr)
}

func (s *Server) GetRouter() *echo.Echo {
	return s.Router
}

// GetHealthStatus gets the health status of the server.
func (s *Server) GetHealthStatus() (live, ready *int32) {
	return s.Live, s.Ready
}

// configureServer configures the server
func (s *Server) configureRouter() {
	s.Router.HideBanner = true
	s.Router.HidePort = true
	s.Router.Host(s.Config.HTTP.Addr)
}

// registerHealthcheckRoutes registers the healthcheck routes
func (s *Server) registerHealthcheckRoutes() {
	s.Live = new(int32)
	s.Ready = new(int32)
	*s.Live = 1
	*s.Ready = 1

	s.Router.GET("/live", echo.HandlerFunc(func(_ echo.Context) error {
		if atomic.LoadInt32(s.Live) == 1 {
			return echo.NewHTTPError(http.StatusOK, "OK")
		}
		return echo.NewHTTPError(http.StatusServiceUnavailable, "KO")
	}))
	s.Router.GET("/ready", echo.HandlerFunc(func(_ echo.Context) error {
		if atomic.LoadInt32(s.Ready) == 1 {
			return echo.NewHTTPError(http.StatusOK, "OK")
		}
		return echo.NewHTTPError(http.StatusServiceUnavailable, "KO")
	}))
}

// WithCORSMiddleware adds CORS middleware to the router
func (s *Server) withCORSMiddleware() {
	if !s.Config.HTTP.CORS.Enabled {
		s.Logger.Warn("CORS is disabled")
		return
	}

	s.Router.Use(middleware.CORSWithConfig(middleware.CORSConfig{
		AllowOrigins: s.Config.HTTP.CORS.AllowedOrigins,
		AllowMethods: s.Config.HTTP.CORS.AllowedMethods,
		AllowHeaders: s.Config.HTTP.CORS.AllowedHeaders,
	}))
}

// withOAuthProtectedResources adds OAuth protected resources to the router
func (s *Server) withOAuthProtectedResources() {
	if !s.Config.OAuth.Enabled {
		s.Logger.Warn("OAuth is disabled. Skipping OAuth protected resources.")
		return
	}

	meta := map[string]any{
		"resource":                 s.Config.OAuth.Resource,
		"authorization_servers":    s.Config.OAuth.AuthorizationServers,
		"bearer_methods_supported": s.Config.OAuth.BearerMethodsSupported,
		"scopes_supported":         s.Config.OAuth.ScopesSupported,
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
		s.Logger.Error("Failed to register metrics", zap.Error(err))
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

	go s.addProxyTools(mcpServer)

	s.Router.GET("/mcp", echo.WrapHandler(serverConfig))
	s.Router.HEAD("/mcp", echo.WrapHandler(serverConfig))
	s.Router.OPTIONS("/mcp", func(c echo.Context) error {
		return c.NoContent(http.StatusNoContent)
	})
	s.Router.POST("/mcp", echo.WrapHandler(serverConfig))
}

// addProxyTools adds the proxy tools to the MCP server.
func (s *Server) addProxyTools(mcpServer *server.MCPServer) {
	for {
		time.Sleep(s.Config.Proxy.CacheTTL)
		s.Logger.Info("Refreshing MCP proxies")
		proxies, err := s.Storage.ListProxies(context.Background(), true)
		if err != nil {
			s.Logger.Error("Failed to get MCP proxies", zap.Error(err))
			continue
		}
		if len(proxies) == 0 {
			s.Logger.Info("No MCP proxies found. Deleting all tools.")
			mcpServer.DeleteTools()
			continue
		}
		mcpProxy, err := proxy.NewProxy(&proxies, s.Logger)
		if err != nil {
			s.Logger.Error("Failed to create MCP proxy", zap.Error(err))
			continue
		}
		for _, proxy := range *mcpProxy {
			proxyTools, err := proxy.GetTools()
			if err != nil {
				s.Logger.Error("Failed to get MCP proxy tools", zap.Error(err))
				continue
			}
			for i := range proxyTools {
				tool := proxyTools[i]
				toolName := proxy.GetName() + ":" + tool.Name
				tool.Name = toolName
				s.Logger.Debug("Adding tool", zap.String("tool", toolName))
				mcpServer.AddTool(tool, proxy.CallTool)
			}
		}
	}
}

// mcpHooks configures the MCP hooks
func (s *Server) mcpHooks() *server.Hooks {
	hooks := &server.Hooks{}

	hooks.AddBeforeCallTool(func(ctx context.Context, id any, message *mcp.CallToolRequest) {
		ctxLogger, ok := ctx.Value("logger").(logger.Logger)
		if !ok {
			s.Logger.Error("Logger not found in context")
			return
		}
		ctxLogger.Info("Tool call started", zap.Any("request_id", id))
		method := message.Method
		params := message.Params
		args := message.GetArguments()
		proxyName, toolName := s.parseToolName(params.Name)
		metrics.ToolsCalledGauge.WithLabelValues(toolName, proxyName).Inc()
		ctxLogger.Debug(
			"Tool call started",
			zap.String("request_method", method),
			zap.String("tool_name", params.Name),
			zap.Any("request_arguments", args),
		)
	})

	hooks.AddAfterCallTool(func(ctx context.Context, id any, message *mcp.CallToolRequest, result *mcp.CallToolResult) {
		ctxLogger, ok := ctx.Value("logger").(logger.Logger)
		if !ok {
			s.Logger.Error("Logger not found in context")
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
			ctxLogger.Error("Invalid request ID", zap.Any("request_id", id))
		}
		proxyName, toolName := s.parseToolName(message.Params.Name)
		if result.IsError {
			ctxLogger.Error(response, zap.String("toolName", message.Params.Name), zap.Float64("request_id", idFloat))
			metrics.ToolsCallErrorsGauge.WithLabelValues(toolName, proxyName).Inc()
		} else {
			ctxLogger.Info(
				"Tool call completed with success",
				zap.String("toolName", message.Params.Name),
				zap.Float64("request_id", idFloat),
			)
			metrics.ToolsCallSuccessGauge.WithLabelValues(toolName, proxyName).Inc()
		}
	})

	hooks.AddBeforeListTools(func(ctx context.Context, id any, _ *mcp.ListToolsRequest) {
		ctxLogger, ok := ctx.Value("logger").(logger.Logger)
		if !ok {
			s.Logger.Error("Logger not found in context")
			return
		}
		ctxLogger.Info("Before List Tools Hook", zap.Any("request_id", id))
		metrics.ListToolsGauge.WithLabelValues("").Inc()
	})

	return hooks
}

func (s *Server) parseToolName(toolName string) (proxyName, toolNameParsed string) {
	parts := strings.Split(toolName, ":")
	if len(parts) != 2 { //nolint:mnd // always return 2 parts
		return "", ""
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
	ctxLogger := s.Logger.With(zap.String("correlation_id", correlationID))
	//nolint:staticcheck,revive // We need to use the key as a string
	ctx = context.WithValue(ctx, "logger", ctxLogger)

	return ctx
}

// configureAuthMiddleware configures the auth middleware
func (s *Server) configureAuthMiddleware() {
	if !s.Config.AuthProvider.Enabled {
		s.Logger.Warn("Auth is disabled. Skipping auth middleware.")
		return
	}

	provider, err := auth.NewProvider(s.Config.AuthProvider.Name, s.Config, s.Logger, s.Storage)
	if err != nil {
		s.Logger.Error("Failed to create provider", zap.Error(err))
		panic(err)
	}
	s.Logger.Info("Configuring auth middleware", zap.String("provider", s.Config.AuthProvider.Name))
	err = provider.Init()
	if err != nil {
		s.Logger.Error("Failed to initialize provider", zap.Error(err))
		panic(err)
	}

	s.Provider = provider

	s.Router.Use(s.authMiddleware)
}

func (s *Server) unauth(c echo.Context, code, msg string) error {
	if s.Config.OAuth.Enabled {
		if len(s.Config.OAuth.AuthorizationServers) == 0 {
			s.Logger.Error("OAuth is enabled but no authorization servers are configured")
			return echo.NewHTTPError(http.StatusInternalServerError, "OAuth configuration error")
		}
		// Set the WWW-Authenticate header for OAuth protected resources
		// This is used by the client to redirect to the authorization server
		// to obtain a token
		rsMetaURL := s.Config.OAuth.AuthorizationServers[0] + "/.well-known/oauth-protected-resource"
		c.Response().Header().Set("WWW-Authenticate",
			fmt.Sprintf(`Bearer resource_metadata=%q, error=%q`, rsMetaURL, code))
	}
	return echo.NewHTTPError(http.StatusUnauthorized, msg)
}

func (s *Server) configureStorage() {
	if s.Config.BackendConfig.Engine == "memory" {
		s.Logger.Warn("Using memory storage. This is not recommended for production.")
	}
	storageClient, err := storage.NewStorage(context.Background(), s.Config.BackendConfig.Engine, "", s.Logger, s.Config, s.Encryptor)
	if err != nil {
		s.Logger.Error("Failed to create storage", zap.Error(err))
		panic(err)
	}
	s.Storage = storageClient
}

func (s *Server) configureSwaggerRoutes() {
	s.Logger.Info(fmt.Sprintf("Configuring Swagger routes. Swagger UI is available at http://%s/swagger/index.html", s.Config.HTTP.Addr))
	s.Router.GET("/swagger/*", echoSwagger.WrapHandler)
}

func (s *Server) configureEncryption() {
	if s.Config.BackendConfig.Engine == "memory" {
		s.Logger.Warn("Using memory storage. Skipping encryption.")
		return
	}
	encryptor, err := aescipher.New(s.Config.BackendConfig.EncryptionKey)
	if err != nil {
		s.Logger.Error("Failed to create encryptor mandatory for backend data encryption", zap.Error(err))
		panic(err)
	}
	s.Encryptor = encryptor
}

func (s *Server) configureV1Routes() {
	v1 := s.Router.Group("/v1")
	v1.Use(func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			apiKey := c.Request().Header.Get("X-API-Key")
			if apiKey != s.Config.HTTP.AdminAPIKey {
				return echo.NewHTTPError(http.StatusUnauthorized, "Invalid API key")
			}
			return next(c)
		}
	})
	s.ConfigureRoutes(v1)
}
