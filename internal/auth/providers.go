// Package oauth provides the providers for the MCP Gateway
package auth

import (
	"context"
	"fmt"

	"github.com/matthisholleville/mcp-gateway/internal/cfg"
	"github.com/matthisholleville/mcp-gateway/internal/storage"
	"github.com/matthisholleville/mcp-gateway/pkg/logger"
)

// Provider is the interface for the providers
type Provider interface {
	Init() error
	VerifyToken(token string) (*Jwt, error)
	VerifyPermissions(ctx context.Context, objectType, objectName, proxy string, claims map[string]interface{}) bool
}

// Jwt is the struct for the JWT token
type Jwt struct {
	Claims map[string]interface{}
}

// NewProvider creates a new provider
func NewProvider(provider string, cfg *cfg.Config, logger logger.Logger, storage storage.StorageInterface) (Provider, error) {
	switch provider {
	case "okta":
		return &OktaProvider{
			BaseProvider: BaseProvider{
				logger:  logger,
				storage: storage,
			},
			cfg:      cfg.AuthProvider.Okta,
			oauthCfg: cfg.OAuth,
			logger:   logger,
		}, nil
	default:
		return nil, fmt.Errorf("provider %s not found", provider)
	}
}
