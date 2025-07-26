// Package oauth provides the providers for the MCP Gateway
package oauth

import (
	"fmt"
	"log/slog"

	"github.com/matthisholleville/mcp-gateway/internal/cfg"
)

// Provider is the interface for the providers
type Provider interface {
	Init() error
	VerifyToken(token string) (*Jwt, error)
	VerifyPermissions(toolName string, claims map[string]interface{}) bool
}

// Jwt is the struct for the JWT token
type Jwt struct {
	Claims map[string]interface{}
}

// NewProvider creates a new provider
func NewProvider(provider string, cfg *cfg.Cfg, logger *slog.Logger) (Provider, error) {
	switch provider {
	case "okta":
		return &OktaProvider{
			BaseProvider: BaseProvider{
				authCfg: &cfg.Auth,
				logger:  logger,
			},
			cfg:      &cfg.Okta,
			oauthCfg: &cfg.OAuth,
			authCfg:  &cfg.Auth,
			logger:   logger,
		}, nil
	default:
		return nil, fmt.Errorf("provider %s not found", provider)
	}
}

// flattenClaims flattens the claims into a list of strings
func flattenClaims(claims map[string]interface{}) ([]string, error) {
	flattenedClaims := []string{}
	for claim, value := range claims {
		switch v := value.(type) {
		case string:
			flattenedClaims = append(flattenedClaims, fmt.Sprintf("%s:%s", claim, v))
		case []string:
			for _, v := range v {
				flattenedClaims = append(flattenedClaims, fmt.Sprintf("%s:%s", claim, v))
			}
		case []interface{}:
			out := make([]string, len(v))
			for i, v := range v {
				out[i] = fmt.Sprintf("%s:%s", claim, v)
			}
			flattenedClaims = append(flattenedClaims, out...)
		default:
			return nil, fmt.Errorf("claim %s has unsupported type %T", claim, v)
		}
	}
	return flattenedClaims, nil
}
