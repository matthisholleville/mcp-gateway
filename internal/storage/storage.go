package storage

import (
	"context"
	"fmt"
	"net/url"

	"github.com/matthisholleville/mcp-gateway/internal/cfg"
	"github.com/matthisholleville/mcp-gateway/pkg/aescipher"
	"github.com/matthisholleville/mcp-gateway/pkg/logger"
)

type BaseInterface interface {
	GetDefaultScope(ctx context.Context) string
}

type BaseStorage struct {
	defaultScope string
}

// GetDefaultScope gets the default scope from the base storage.
func (b *BaseStorage) GetDefaultScope(_ context.Context) string {
	return b.defaultScope
}

// Interface is an interface that provides a storage interface for the MCP Gateway.
type Interface interface {
	BaseInterface
	ProxyInterface
	RoleInterface
	AttributeToRolesInterface
}

// NewStorage creates a new storage instance.
//
//nolint:gocritic // we need to keep logger as a parameter for the function
func NewStorage(_ context.Context, storageType, defaultScope string, logger logger.Logger, cfg *cfg.Config, encryptor aescipher.Cryptor) (Interface, error) {
	switch storageType {
	case "memory":
		return NewMemoryStorage(defaultScope), nil
	case "postgres":
		return NewPostgresStorage(defaultScope, logger, cfg, encryptor)
	}
	return nil, fmt.Errorf("invalid storage type: %s", storageType)
}

// getURI gets the URI for the storage backend.
func getURI(config *cfg.Config) (string, error) {
	if config.BackendConfig.Username != "" || config.BackendConfig.Password != "" {
		parsed, err := url.Parse(config.BackendConfig.URI)
		if err != nil {
			return "", fmt.Errorf("parse postgres connection uri: %w", err)
		}
		username := ""
		switch {
		case config.BackendConfig.Username != "":
			username = config.BackendConfig.Username
		case parsed.User != nil:
			username = parsed.User.Username()
		default:
			username = ""
		}
		switch {
		case config.BackendConfig.Password != "":
			parsed.User = url.UserPassword(username, config.BackendConfig.Password)
		case parsed.User != nil:
			if password, ok := parsed.User.Password(); ok {
				parsed.User = url.UserPassword(username, password)
			} else {
				parsed.User = url.User(username)
			}
		default:
			parsed.User = url.User(username)
		}

		return parsed.String(), nil
	}
	return config.BackendConfig.URI, nil
}
