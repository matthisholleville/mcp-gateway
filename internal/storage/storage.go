package storage

import (
	"context"
	"fmt"

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

func (b *BaseStorage) GetDefaultScope(ctx context.Context) string {
	return b.defaultScope
}

type StorageInterface interface {
	BaseInterface
	ProxyInterface
	RoleInterface
	AttributeToRolesInterface
}

func NewStorage(ctx context.Context, storageType, defaultScope string, logger logger.Logger, cfg *cfg.Config, encryptor aescipher.Cryptor) (StorageInterface, error) {
	switch storageType {
	case "memory":
		return NewMemoryStorage(defaultScope), nil
	case "postgres":
		return NewPostgresStorage(defaultScope, logger, cfg, encryptor)
	}
	return nil, fmt.Errorf("invalid storage type: %s", storageType)
}
