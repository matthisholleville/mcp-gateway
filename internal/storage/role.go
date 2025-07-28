package storage

import "context"

type RoleConfig struct {
	Name        string             `json:"name"`
	Permissions []PermissionConfig `json:"permissions"`
}

type ObjectType string

const (
	ObjectTypeTools ObjectType = "tools"
	ObjectTypeAll   ObjectType = "*"
)

type PermissionConfig struct {
	ObjectType ObjectType `json:"object_type"`
	Proxy      string     `json:"proxy"`
	ObjectName string     `json:"object_name"`
}

type RoleInterface interface {
	ListRoles(ctx context.Context) ([]RoleConfig, error)
	SetRole(ctx context.Context, role RoleConfig) error
	GetRole(ctx context.Context, role string) (RoleConfig, error)
	DeleteRole(ctx context.Context, role string) error
}
