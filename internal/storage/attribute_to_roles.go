package storage

import "context"

type AttributeToRolesConfig struct {
	AttributeKey   string   `json:"attribute_key"`
	AttributeValue string   `json:"attribute_value"`
	Roles          []string `json:"roles"`
}

type AttributeToRolesInterface interface {
	ListAttributeToRoles(ctx context.Context) ([]AttributeToRolesConfig, error)
	SetAttributeToRoles(ctx context.Context, attributeToRoles AttributeToRolesConfig) error
	GetAttributeToRoles(ctx context.Context, attributeKey, attributeValue string) (AttributeToRolesConfig, error)
	DeleteAttributeToRoles(ctx context.Context, attributeKey, attributeValue string) error
}
