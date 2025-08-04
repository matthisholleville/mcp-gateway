package server

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/labstack/echo/v4"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/matthisholleville/mcp-gateway/internal/auth"
	"github.com/matthisholleville/mcp-gateway/internal/cfg"
	"github.com/matthisholleville/mcp-gateway/pkg/logger"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// MockProvider est un mock simple du Provider pour les tests
type MockProvider struct {
	shouldVerifyToken       bool
	shouldVerifyPermissions bool
	verifyTokenError        error
}

func (m *MockProvider) Init() error {
	return nil
}

func (m *MockProvider) VerifyToken(token string) (*auth.Jwt, error) {
	if m.verifyTokenError != nil {
		return nil, m.verifyTokenError
	}
	if !m.shouldVerifyToken {
		return nil, assert.AnError
	}
	return &auth.Jwt{
		Claims: map[string]interface{}{
			"sub": "test-user",
		},
	}, nil
}

func (m *MockProvider) VerifyPermissions(ctx context.Context, objectType, objectName, proxy string, claims map[string]interface{}) bool {
	return m.shouldVerifyPermissions
}

// createTestServer creates a test server with the given OAuth enabled and provider
func createTestServer(oauthEnabled bool, provider auth.Provider) *Server {
	log := logger.MustNewLogger("json", "debug", "test")
	return &Server{
		Config: &cfg.Config{
			OAuth: &cfg.OAuthConfig{
				Enabled:              oauthEnabled,
				AuthorizationServers: []string{"https://test.example.com"},
			},
		},
		Router:   echo.New(),
		Logger:   log,
		Provider: provider,
	}
}

// createMCPRequest creates a MCP request with the given method and tool name
func createMCPRequest(method, toolName string) *http.Request {
	toolRequest := mcp.CallToolRequest{
		Request: mcp.Request{
			Method: method,
		},
		Params: mcp.CallToolParams{
			Name: toolName,
		},
	}

	body, _ := json.Marshal(toolRequest)
	req := httptest.NewRequest(http.MethodPost, "/mcp", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	return req
}

// createTestContext creates a test context with the given server, request, response recorder and path
func createTestContext(server *Server, req *http.Request, rec *httptest.ResponseRecorder, path string) echo.Context {
	c := server.Router.NewContext(req, rec)
	c.SetPath(path)
	c.SetRequest(req)
	return c
}

// TestAuthMiddleware_NonMCPPath tests the auth middleware with a non-MCP path
func TestAuthMiddleware_NonMCPPath(t *testing.T) {
	provider := &MockProvider{}
	server := createTestServer(true, provider)

	// Handler simple qui retourne OK
	nextHandler := func(c echo.Context) error {
		return c.String(http.StatusOK, "ok")
	}

	middleware := server.authMiddleware(nextHandler)

	// Test avec un path non-MCP
	req := httptest.NewRequest(http.MethodGet, "/other", nil)
	rec := httptest.NewRecorder()
	c := createTestContext(server, req, rec, "/other")

	err := middleware(c)

	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "ok", rec.Body.String())
}

// TestAuthMiddleware_OAuthDisabledAndNotToolCall tests the auth middleware with a MCP request and OAuth disabled
func TestAuthMiddleware_OAuthDisabledAndNotToolCall(t *testing.T) {
	provider := &MockProvider{}
	server := createTestServer(false, provider) // OAuth disabled

	nextHandler := func(c echo.Context) error {
		return c.String(http.StatusOK, "ok")
	}

	middleware := server.authMiddleware(nextHandler)

	// MCP request but not a tool call
	req := createMCPRequest("tools/list", "")
	rec := httptest.NewRecorder()
	c := createTestContext(server, req, rec, "/mcp")

	err := middleware(c)

	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)
}

// TestAuthMiddleware_MissingToken tests the auth middleware with a MCP request and missing token
func TestAuthMiddleware_MissingToken(t *testing.T) {
	provider := &MockProvider{}
	server := createTestServer(true, provider)

	nextHandler := func(c echo.Context) error {
		return c.String(http.StatusOK, "ok")
	}

	middleware := server.authMiddleware(nextHandler)

	// MCP request but no token
	req := createMCPRequest("tools/call", "proxy1:tool1")
	rec := httptest.NewRecorder()
	c := createTestContext(server, req, rec, "/mcp")

	err := middleware(c)

	// Should return a HTTP 401 error
	fmt.Println(err)
	httpErr, ok := err.(*echo.HTTPError)
	require.True(t, ok)
	assert.Equal(t, http.StatusUnauthorized, httpErr.Code)
	assert.Equal(t, "Missing token", httpErr.Message)
}

// TestAuthMiddleware_InvalidToken tests the auth middleware with a MCP request and invalid token
func TestAuthMiddleware_InvalidToken(t *testing.T) {
	provider := &MockProvider{
		shouldVerifyToken: false, // Invalid token
	}
	server := createTestServer(true, provider)

	nextHandler := func(c echo.Context) error {
		return c.String(http.StatusOK, "ok")
	}

	middleware := server.authMiddleware(nextHandler)

	// Request with invalid token
	req := createMCPRequest("tools/call", "proxy1:tool1")
	req.Header.Set("Authorization", "Bearer invalid-token")
	rec := httptest.NewRecorder()
	c := createTestContext(server, req, rec, "/mcp")

	err := middleware(c)

	httpErr, ok := err.(*echo.HTTPError)
	require.True(t, ok)
	assert.Equal(t, http.StatusUnauthorized, httpErr.Code)
	assert.Equal(t, "Invalid token", httpErr.Message)
}

// TestAuthMiddleware_InsufficientPermissions tests the auth middleware with a MCP request and insufficient permissions
func TestAuthMiddleware_InsufficientPermissions(t *testing.T) {
	provider := &MockProvider{
		shouldVerifyToken:       true,  // Valid token
		shouldVerifyPermissions: false, // Insufficient permissions
	}
	server := createTestServer(true, provider)

	nextHandler := func(c echo.Context) error {
		return c.String(http.StatusOK, "ok")
	}

	middleware := server.authMiddleware(nextHandler)

	req := createMCPRequest("tools/call", "proxy1:tool1")
	req.Header.Set("Authorization", "Bearer valid-token")
	rec := httptest.NewRecorder()
	c := createTestContext(server, req, rec, "/mcp")

	err := middleware(c)

	httpErr, ok := err.(*echo.HTTPError)
	require.True(t, ok)
	assert.Equal(t, http.StatusUnauthorized, httpErr.Code)
	assert.Equal(t, "Insufficient scope", httpErr.Message)
}

// TestAuthMiddleware_Success tests the auth middleware with a MCP request and valid token and permissions
func TestAuthMiddleware_Success(t *testing.T) {
	provider := &MockProvider{
		shouldVerifyToken:       true, // Valid token
		shouldVerifyPermissions: true, // Permissions OK
	}
	server := createTestServer(true, provider)

	nextHandler := func(c echo.Context) error {
		// Check that the claims are added to the context
		claims := c.Get("claims")
		assert.NotNil(t, claims)
		return c.String(http.StatusOK, "ok")
	}

	middleware := server.authMiddleware(nextHandler)

	req := createMCPRequest("tools/call", "proxy1:tool1")
	req.Header.Set("Authorization", "Bearer valid-token")
	rec := httptest.NewRecorder()
	c := createTestContext(server, req, rec, "/mcp")

	err := middleware(c)

	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "ok", rec.Body.String())
}

// TestAuthMiddleware_TokenWithBearerPrefix tests the auth middleware with a MCP request and token with bearer prefix
func TestAuthMiddleware_TokenWithBearerPrefix(t *testing.T) {
	provider := &MockProvider{
		shouldVerifyToken:       true,
		shouldVerifyPermissions: true,
	}
	server := createTestServer(true, provider)

	nextHandler := func(c echo.Context) error {
		return c.String(http.StatusOK, "ok")
	}

	middleware := server.authMiddleware(nextHandler)

	req := createMCPRequest("tools/call", "proxy1:tool1")
	req.Header.Set("Authorization", "Bearer my-token") // With bearer prefix
	rec := httptest.NewRecorder()
	c := createTestContext(server, req, rec, "/mcp")

	err := middleware(c)

	assert.NoError(t, err)
}

// TestAuthMiddleware_InvalidRequestBody tests the auth middleware with a MCP request and invalid request body
func TestAuthMiddleware_InvalidRequestBody(t *testing.T) {
	provider := &MockProvider{}
	server := createTestServer(true, provider)

	nextHandler := func(c echo.Context) error {
		return c.String(http.StatusOK, "ok")
	}

	middleware := server.authMiddleware(nextHandler)

	// Request with invalid JSON body
	req := httptest.NewRequest(http.MethodPost, "/mcp", strings.NewReader("invalid json"))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	c := createTestContext(server, req, rec, "/mcp")

	err := middleware(c)

	httpErr, ok := err.(*echo.HTTPError)
	require.True(t, ok)
	assert.Equal(t, http.StatusUnauthorized, httpErr.Code)
	assert.Equal(t, "Invalid request", httpErr.Message)
}

// TestAuthMiddleware_OAuthDisabledButToolCall tests the auth middleware with a MCP request and OAuth disabled but tool call
func TestAuthMiddleware_OAuthDisabledButToolCall(t *testing.T) {
	provider := &MockProvider{
		shouldVerifyToken: true,
	}
	server := createTestServer(false, provider) // OAuth disabled

	nextHandler := func(c echo.Context) error {
		return c.String(http.StatusOK, "ok")
	}

	middleware := server.authMiddleware(nextHandler)

	// Tool call with OAuth disabled but token present
	req := createMCPRequest("tools/call", "proxy1:tool1")
	req.Header.Set("Authorization", "Bearer valid-token")
	rec := httptest.NewRecorder()
	c := createTestContext(server, req, rec, "/mcp")

	err := middleware(c)

	// Shouldn't pass because insufficient permissions
	httpErr, ok := err.(*echo.HTTPError)
	require.True(t, ok)
	assert.Equal(t, http.StatusUnauthorized, httpErr.Code)
	assert.Equal(t, "Insufficient scope", httpErr.Message)

}
