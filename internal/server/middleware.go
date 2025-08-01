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

func (s *Server) authMiddleware(next echo.HandlerFunc) echo.HandlerFunc {
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

		jwtToken, err := s.Provider.VerifyToken(token)
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

		hasPermission := s.Provider.VerifyPermissions(c.Request().Context(), objectType, objectName, proxyName, jwtToken.Claims)
		if !hasPermission {
			return s.unauth(c, "insufficient_scope", "Insufficient scope")
		}

		c.Set("claims", jwtToken.Claims)
		return next(c)
	}
}
