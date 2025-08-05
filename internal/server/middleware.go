package server

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"strings"

	"github.com/labstack/echo/v4"
	"github.com/mark3labs/mcp-go/mcp"
	"go.uber.org/zap"
)

// authMiddleware is the middleware that checks if the request is valid and if the user has the necessary permissions
func (s *Server) authMiddleware(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		isMCPPath := c.Path() == "/mcp" && c.Request().Method == "POST"
		if !isMCPPath {
			return next(c)
		}

		message, err := s.parseRequestBody(c)
		if err != nil {
			return s.unauth(c, "invalid_request", "Invalid request")
		}

		isOAuthEnabled := s.Config.OAuth.Enabled
		isToolCall := message.Method == "tools/call"
		if !isOAuthEnabled && !isToolCall {
			return next(c)
		}

		token := c.Request().Header.Get("Authorization")
		if token == "" {
			return s.unauth(c, "missing_token", "Missing token")
		}
		token = strings.TrimPrefix(token, "Bearer ")

		jwtToken, err := s.Provider.VerifyToken(token)
		if err != nil {
			return s.unauth(c, "invalid_token", "Invalid token")
		}

		// tools/call:tools
		s.Logger.Debug("Verifying permissions for tool call",
			zap.String("method", message.Method),
			zap.String("params", message.Params.Name),
			zap.Any("claims", jwtToken.Claims))
		objectType := strings.Split(message.Method, "/")[0]
		paramsSplit := strings.Split(message.Params.Name, ":")
		objectName := paramsSplit[1]
		proxyName := paramsSplit[0]

		hasPermission := s.Provider.VerifyPermissions(c.Request().Context(), objectType, proxyName, objectName, jwtToken.Claims)
		if !hasPermission {
			return s.unauth(c, "insufficient_scope", "Insufficient scope")
		}

		c.Set("claims", jwtToken.Claims)
		return next(c)
	}
}

// parseRequestBody parses the request body and returns a MCP request
func (s *Server) parseRequestBody(c echo.Context) (*mcp.CallToolRequest, error) {
	const maxBodySize = 1 << 20 // 1 MiB

	req := c.Request()
	body := req.Body
	req.Body = http.MaxBytesReader(c.Response(), body, maxBodySize)

	var copyBuf bytes.Buffer
	tee := io.TeeReader(req.Body, &copyBuf)

	dec := json.NewDecoder(tee)

	message := &mcp.CallToolRequest{}
	err := dec.Decode(message)
	if err != nil {
		s.Logger.Error("Failed to unmarshal request body", zap.Error(err))
		return nil, err
	}

	req.Body = io.NopCloser(&copyBuf)

	return message, nil
}
