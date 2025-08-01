package auth

import (
	"context"
	"testing"

	"github.com/matthisholleville/mcp-gateway/internal/storage"
	"github.com/matthisholleville/mcp-gateway/pkg/logger"
	"github.com/stretchr/testify/assert"
)

func initData(t *testing.T, attributeToRoles []storage.AttributeToRolesConfig, roles []storage.RoleConfig) storage.Interface {
	engine := storage.NewMemoryStorage("")
	for _, role := range roles {
		err := engine.SetRole(context.Background(), role)
		if err != nil {
			t.Fatalf("Failed to set role: %v", err)
		}
	}
	for _, attributeToRole := range attributeToRoles {
		err := engine.SetAttributeToRoles(context.Background(), attributeToRole)
		if err != nil {
			t.Fatalf("Failed to set attribute to roles: %v", err)
		}
	}

	return engine
}

func initLogger() logger.Logger {
	return logger.MustNewLogger("json", "debug", "test")
}

func TestBaseProvider_ClaimToRoles(t *testing.T) {
	for _, test := range []struct {
		name             string
		attributeToRoles []storage.AttributeToRolesConfig
		roles            []storage.RoleConfig
		endWithError     bool
		expected         []string
	}{
		{
			name: "Admin",
			attributeToRoles: []storage.AttributeToRolesConfig{
				{
					AttributeKey:   "Groups",
					AttributeValue: "group1",
					Roles:          []string{"Admin"},
				},
			},
			roles: []storage.RoleConfig{
				{
					Name: "Admin",
					Permissions: []storage.PermissionConfig{
						{
							ObjectType: "*",
							Proxy:      "*",
							ObjectName: "*",
						},
					},
				},
			},
			endWithError: false,
			expected:     []string{"Admin"},
		},
		{
			name:             "Empty data",
			attributeToRoles: []storage.AttributeToRolesConfig{},
			roles:            []storage.RoleConfig{},
			endWithError:     false,
			expected:         []string{},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			engine := initData(t, test.attributeToRoles, test.roles)
			logger := initLogger()
			provider := BaseProvider{
				storage: engine,
				logger:  logger,
			}
			attributeToRoles := provider.attributeToRoles(context.Background(), map[string]interface{}{
				"email":          "test@test.com",
				"auth_time":      1717000000,
				"email_verified": false,
				"identities":     map[string]string{"google.com": "test@test.com"},
				"Groups":         []string{"group1", "group2"},
			})
			assert.Equal(t, test.expected, attributeToRoles)
		})
	}
}

func TestBaseProvider_VerifyPermissions(t *testing.T) {
	for _, test := range []struct {
		name             string
		attributeToRoles []storage.AttributeToRolesConfig
		roles            []storage.RoleConfig
		objectType       string
		objectName       string
		proxy            string
		claims           map[string]interface{}
		expected         bool
	}{
		{
			name: "Authorized: Admin",
			attributeToRoles: []storage.AttributeToRolesConfig{
				{
					AttributeKey:   "Groups",
					AttributeValue: "group1",
					Roles:          []string{"Admin"},
				},
			},
			roles: []storage.RoleConfig{
				{
					Name: "Admin",
					Permissions: []storage.PermissionConfig{
						{
							ObjectType: "*",
							Proxy:      "*",
							ObjectName: "*",
						},
					},
				},
			},
			expected:   true,
			objectType: "tools",
			objectName: "all",
			proxy:      "tools",
			claims: map[string]interface{}{
				"Groups": []string{"group1"},
			},
		},
		{
			name: "Unauthorized: Tools Admin requested prompts",
			attributeToRoles: []storage.AttributeToRolesConfig{
				{
					AttributeKey:   "Groups",
					AttributeValue: "group1",
					Roles:          []string{"ToolsAdmin"},
				},
			},
			roles: []storage.RoleConfig{
				{
					Name: "ToolsAdmin",
					Permissions: []storage.PermissionConfig{
						{
							ObjectType: "tools",
							Proxy:      "*",
							ObjectName: "*",
						},
					},
				},
			},
			expected:   false,
			objectType: "prompts",
			objectName: "all",
			proxy:      "tools",
			claims: map[string]interface{}{
				"Groups": []string{"group1"},
			},
		},
		{
			name: "Unauthorized: User with no role in attribute to roles",
			attributeToRoles: []storage.AttributeToRolesConfig{
				{
					AttributeKey:   "Groups",
					AttributeValue: "group1",
					Roles:          []string{"Admin"},
				},
			},
			roles: []storage.RoleConfig{
				{
					Name: "Admin",
					Permissions: []storage.PermissionConfig{
						{
							ObjectType: "tools",
							Proxy:      "*",
							ObjectName: "*",
						},
					},
				},
			},
			expected:   false,
			objectType: "tools",
			objectName: "all",
			proxy:      "tools",
			claims: map[string]interface{}{
				"Groups": []string{"group2"},
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			engine := initData(t, test.attributeToRoles, test.roles)
			logger := initLogger()
			provider := BaseProvider{
				storage: engine,
				logger:  logger,
			}
			permissions := provider.VerifyPermissions(context.Background(), test.objectType, test.objectName, test.proxy, test.claims)
			assert.Equal(t, test.expected, permissions)
		})
	}
}
