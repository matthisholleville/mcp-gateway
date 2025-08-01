package server

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
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
	Storage   storage.StorageInterface
	Encryptor aescipher.Cryptor
}

func NewServer(
	logger logger.Logger,
	config *cfg.Config,

) (*Server, error) {
	router := echo.New()
	s := &Server{
		Logger: logger,
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

func (s *Server) GetHealthStatus() (*int32, *int32) {
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

	s.addProxyTools(mcpServer)

	s.Router.GET("/mcp", echo.WrapHandler(serverConfig))
	s.Router.HEAD("/mcp", echo.WrapHandler(serverConfig))
	s.Router.OPTIONS("/mcp", func(c echo.Context) error {
		return c.NoContent(http.StatusNoContent)
	})
	s.Router.POST("/mcp", echo.WrapHandler(serverConfig))
}

func (s *Server) addProxyTools(mcpServer *server.MCPServer) {
	go func() {
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
	}()
}

// mcpHooks configures the MCP hooks
func (s *Server) mcpHooks() *server.Hooks {
	hooks := &server.Hooks{}

	hooks.AddBeforeCallTool(func(ctx context.Context, id any, message *mcp.CallToolRequest) {
		logger, ok := ctx.Value("logger").(logger.Logger)
		if !ok {
			s.Logger.Error("Logger not found in context")
			return
		}
		logger.Info("Tool call started", zap.Any("request_id", id))
		method := message.Method
		params := message.Params
		args := message.GetArguments()
		proxyName, toolName := s.parseToolName(params.Name)
		metrics.ToolsCalledGauge.WithLabelValues(toolName, proxyName).Inc()
		logger.Debug("Tool call started", zap.String("request_method", method), zap.String("tool_name", params.Name), zap.Any("request_arguments", args))
	})

	hooks.AddAfterCallTool(func(ctx context.Context, id any, message *mcp.CallToolRequest, result *mcp.CallToolResult) {
		logger, ok := ctx.Value("logger").(logger.Logger)
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
			logger.Error("Invalid request ID", zap.Any("request_id", id))
		}
		proxyName, toolName := s.parseToolName(message.Params.Name)
		if result.IsError {
			logger.Error(response, zap.String("toolName", message.Params.Name), zap.Float64("request_id", idFloat))
			metrics.ToolsCallErrorsGauge.WithLabelValues(toolName, proxyName).Inc()
		} else {
			logger.Info(
				"Tool call completed with success",
				zap.String("toolName", message.Params.Name),
				zap.Float64("request_id", idFloat),
			)
			metrics.ToolsCallSuccessGauge.WithLabelValues(toolName, proxyName).Inc()
		}
	})

	hooks.AddBeforeListTools(func(ctx context.Context, id any, _ *mcp.ListToolsRequest) {
		logger, ok := ctx.Value("logger").(logger.Logger)
		if !ok {
			s.Logger.Error("Logger not found in context")
			return
		}
		logger.Info("Before List Tools Hook", zap.Any("request_id", id))
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
	logger := s.Logger.With(zap.String("correlation_id", correlationID))
	//nolint:staticcheck,revive // We need to use the key as a string
	ctx = context.WithValue(ctx, "logger", logger)

	return ctx
}

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
				s.Logger.Error("Failed to unmarshal request body", zap.Error(err))
				return s.unauth(c, "invalid_request", "Invalid request")
			}

			req.Body = io.NopCloser(&copyBuf)

			if message.Method != "tools/call" {
				return next(c)
			}

			// tools/call:tools
			objectType := strings.Split(message.Method, ":")[0]
			toolSplit := strings.Split(message.Params.Name, ":")
			objectName := toolSplit[1]
			proxyName := toolSplit[0]

			hasPermission := provider.VerifyPermissions(c.Request().Context(), objectType, objectName, proxyName, jwtToken.Claims)
			if !hasPermission {
				return s.unauth(c, "insufficient_scope", "Insufficient scope")
			}

			c.Set("claims", jwtToken.Claims)
			return next(c)
		}
	})

}

func (s *Server) unauth(c echo.Context, code, msg string) error {
	rsMetaURL := s.Config.OAuth.AuthorizationServers[0] + "/.well-known/oauth-protected-resource"
	c.Response().Header().Set("WWW-Authenticate",
		fmt.Sprintf(`Bearer resource_metadata=%q, error=%q`, rsMetaURL, code))
	return echo.NewHTTPError(http.StatusUnauthorized, msg)
}

func (s *Server) configureStorage() {
	if s.Config.BackendConfig.Engine == "memory" {
		s.Logger.Warn("Using memory storage. This is not recommended for production.")
	}
	storage, err := storage.NewStorage(context.Background(), s.Config.BackendConfig.Engine, "", s.Logger, s.Config, s.Encryptor)
	if err != nil {
		s.Logger.Error("Failed to create storage", zap.Error(err))
		panic(err)
	}
	s.Storage = storage
}

func (s *Server) configureSwaggerRoutes() {
	s.Logger.Info(fmt.Sprintf("Configuring Swagger routes. Swagger UI is available at http://%s/swagger/index.html", s.Config.HTTP.Addr))
	s.Router.GET("/swagger/*", echoSwagger.WrapHandler)
}

func (s *Server) configureEncryption() {
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
