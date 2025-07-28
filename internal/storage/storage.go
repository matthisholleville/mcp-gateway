package storage

import (
	"context"
	"fmt"
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
	ClaimToRolesInterface
}

func NewStorage(ctx context.Context, storageType, defaultScope string) (StorageInterface, error) {
	switch storageType {
	case "memory":
		return NewMemoryStorage(defaultScope), nil
	}
	return nil, fmt.Errorf("invalid storage type: %s", storageType)
}
