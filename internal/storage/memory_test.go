package storage

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMemoryProxyStorage(t *testing.T) {
	storage := NewMemoryStorage("")
	proxy := ProxyConfig{Name: "test", Type: ProxyTypeStreamableHTTP}
	err := storage.SetProxy(context.Background(), proxy)
	assert.NoError(t, err)
	proxy, err = storage.GetProxy(context.Background(), proxy)
	assert.NoError(t, err)
	assert.Equal(t, proxy.Name, "test")
	err = storage.DeleteProxy(context.Background(), proxy)
	assert.NoError(t, err)
	proxy, err = storage.GetProxy(context.Background(), proxy)
	assert.Error(t, err)
	assert.Equal(t, proxy.Name, "")
}

func TestMemoryStorageRoles(t *testing.T) {
	storage := NewMemoryStorage("")
	role := RoleConfig{Name: "admin", Permissions: []PermissionConfig{
		{
			ObjectType: "*",
			Proxy:      "*",
			ObjectName: "*",
		},
	}}
	err := storage.SetRole(context.Background(), role)
	assert.NoError(t, err)
	role, err = storage.GetRole(context.Background(), role.Name)
	assert.NoError(t, err)
	assert.Equal(t, role.Permissions, []PermissionConfig{
		{
			ObjectType: "*",
			Proxy:      "*",
			ObjectName: "*",
		},
	})
	err = storage.SetRole(context.Background(), RoleConfig{Name: "admin", Permissions: []PermissionConfig{
		{
			ObjectType: "*",
			Proxy:      "*",
			ObjectName: "*",
		},
	}})
	assert.Error(t, err, "role already exists")
	roles, err := storage.ListRoles(context.Background())
	assert.NoError(t, err)
	assert.Equal(t, roles, []RoleConfig{role})
	err = storage.DeleteRole(context.Background(), role.Name)
	assert.NoError(t, err)
	roles, err = storage.ListRoles(context.Background())
	assert.NoError(t, err)
	assert.Equal(t, roles, []RoleConfig{})
}

func TestMemoryStorageClaimToRoles(t *testing.T) {
	storage := NewMemoryStorage("")
	claimToRoles := ClaimToRolesConfig{ClaimKey: "email", ClaimValue: "test@test.com", Roles: []string{"test"}}
	err := storage.SetClaimToRoles(context.Background(), claimToRoles)
	assert.Error(t, err, "role not found")
	role := RoleConfig{Name: "test", Permissions: []PermissionConfig{
		{
			ObjectType: "*",
			Proxy:      "*",
			ObjectName: "*",
		},
	}}
	err = storage.SetRole(context.Background(), role)
	assert.NoError(t, err)
	err = storage.SetClaimToRoles(context.Background(), claimToRoles)
	assert.NoError(t, err)
	err = storage.SetClaimToRoles(context.Background(), claimToRoles)
	assert.Error(t, err, "claim to roles already exists")
	claimToRolesList, err := storage.ListClaimToRoles(context.Background())
	assert.NoError(t, err)
	assert.Equal(t, claimToRolesList, []ClaimToRolesConfig{claimToRoles})
	err = storage.DeleteClaimToRoles(context.Background(), claimToRoles.ClaimKey, claimToRoles.ClaimValue)
	assert.NoError(t, err)
}
