package auth

import (
	"context"
	"fmt"
	"testing"

	"github.com/matthisholleville/mcp-gateway/internal/storage"
	"github.com/matthisholleville/mcp-gateway/pkg/logger"
	"github.com/stretchr/testify/assert"
)

func initData(t *testing.T, claimToRoles []storage.ClaimToRolesConfig, roles []storage.RoleConfig) storage.StorageInterface {
	engine := storage.NewMemoryStorage("")
	for _, role := range roles {
		err := engine.SetRole(context.Background(), role)
		if err != nil {
			t.Fatalf("Failed to set role: %v", err)
		}
	}
	for _, claimToRole := range claimToRoles {
		err := engine.SetClaimToRoles(context.Background(), claimToRole)
		if err != nil {
			t.Fatalf("Failed to set claim to roles: %v", err)
		}
	}

	return engine
}

func initLogger() logger.Logger {
	return logger.MustNewLogger("test", "debug", "test")
}

func TestBaseProvider_ClaimToRoles(t *testing.T) {
	fmt.Println("TestBaseProvider_ClaimToRoles")
	for _, test := range []struct {
		name         string
		claimToRoles []storage.ClaimToRolesConfig
		roles        []storage.RoleConfig
		endWithError bool
		expected     []string
	}{
		{
			name: "Admin",
			claimToRoles: []storage.ClaimToRolesConfig{
				{
					ClaimKey:   "Groups",
					ClaimValue: "group1",
					Roles:      []string{"Admin"},
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
			name:         "Empty data",
			claimToRoles: []storage.ClaimToRolesConfig{},
			roles:        []storage.RoleConfig{},
			endWithError: false,
			expected:     []string{},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			engine := initData(t, test.claimToRoles, test.roles)
			logger := initLogger()
			provider := BaseProvider{
				storage: engine,
				logger:  logger,
			}
			claimToRoles, err := provider.claimsToRoles(context.Background(), map[string]interface{}{
				"email":          "test@test.com",
				"auth_time":      1717000000,
				"email_verified": false,
				"identities":     map[string]string{"google.com": "test@test.com"},
				"Groups":         []string{"group1", "group2"},
			})
			if test.endWithError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, test.expected, claimToRoles)
			}
		})
	}
}

func TestBaseProvider_VerifyPermissions(t *testing.T) {
	fmt.Println("TestBaseProvider_VerifyPermissions")
	for _, test := range []struct {
		name         string
		claimToRoles []storage.ClaimToRolesConfig
		roles        []storage.RoleConfig
		objectType   string
		objectName   string
		proxy        string
		claims       map[string]interface{}
		expected     bool
	}{
		{
			name: "Authorized: Admin",
			claimToRoles: []storage.ClaimToRolesConfig{
				{
					ClaimKey:   "Groups",
					ClaimValue: "group1",
					Roles:      []string{"Admin"},
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
			claimToRoles: []storage.ClaimToRolesConfig{
				{
					ClaimKey:   "Groups",
					ClaimValue: "group1",
					Roles:      []string{"ToolsAdmin"},
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
			name: "Unauthorized: User with no role in claim to roles",
			claimToRoles: []storage.ClaimToRolesConfig{
				{
					ClaimKey:   "Groups",
					ClaimValue: "group1",
					Roles:      []string{"Admin"},
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
			engine := initData(t, test.claimToRoles, test.roles)
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
