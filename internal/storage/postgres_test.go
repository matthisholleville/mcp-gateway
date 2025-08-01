package storage

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/matthisholleville/mcp-gateway/internal/cfg"
	testFixtures "github.com/matthisholleville/mcp-gateway/internal/storage/testsFixtures"
	"github.com/matthisholleville/mcp-gateway/pkg/aescipher"
	"github.com/matthisholleville/mcp-gateway/pkg/logger"
	"github.com/stretchr/testify/assert"
)

func testPostgresStorage(t *testing.T) (*PostgresStorage, error) {
	logger := logger.MustNewLogger("json", "debug", "")
	postgresOpts := &testFixtures.PostgresTestContainerOptions{
		MigrationsDir: "../../assets/migrations/postgres",
	}

	encryptor, err := aescipher.New("0123456789abcdeffedcba9876543210cafebabefacefeeddeadbeef00112233")
	if err != nil {
		return nil, err
	}

	db := testFixtures.NewPostgresTestContainer(postgresOpts).RunPostgresTestContainer(t)
	testConfig := &cfg.Config{
		BackendConfig: &cfg.BackendConfig{
			Engine: "postgres",
			URI:    db.GetConnectionURI(true),
		},
	}
	return NewPostgresStorage("test", logger, testConfig, encryptor)
}

func TestProxyStorage(t *testing.T) {
	storage, err := testPostgresStorage(t)
	assert.NoError(t, err)

	t.Run("insert proxy with unencrypted headers", func(t *testing.T) {
		fmt.Println("insert proxy with headers")
		proxy := ProxyConfig{
			Name:     "test",
			Type:     ProxyTypeStreamableHTTP,
			URL:      "https://example.com",
			Timeout:  time.Duration(10 * time.Second),
			AuthType: ProxyAuthTypeHeader,
			Headers: []ProxyHeader{
				{Key: "test", Value: "test"},
			},
		}
		err := storage.SetProxy(context.Background(), proxy, true)
		assert.NoError(t, err)
	})

	t.Run("ensure proxy & headers are inserted and decrypted", func(t *testing.T) {
		proxy, err := storage.GetProxy(context.Background(), "test", false)
		assert.NoError(t, err)
		assert.Equal(t, "test", proxy.Name)
		assert.Equal(t, ProxyTypeStreamableHTTP, proxy.Type)
		assert.Equal(t, "https://example.com", proxy.URL)
		assert.Equal(t, time.Duration(10*time.Second), proxy.Timeout)
		assert.Equal(t, ProxyAuthTypeHeader, proxy.AuthType)
		assert.NotEqual(t, "test", proxy.Headers[0].Value)
	})

	t.Run("ensure list proxies return 1 element", func(t *testing.T) {
		proxies, err := storage.ListProxies(context.Background(), false)
		assert.NoError(t, err)
		assert.Equal(t, 1, len(proxies))
		assert.Equal(t, "test", proxies[0].Name)
	})

	t.Run("update proxy with new header", func(t *testing.T) {
		proxy, err := storage.GetProxy(context.Background(), "test", false)
		assert.NoError(t, err)
		proxy.Headers = []ProxyHeader{
			{Key: "test", Value: "test2"},
			{Key: "test2", Value: "test3"},
		}
		err = storage.SetProxy(context.Background(), proxy, false)
		assert.NoError(t, err)
	})

	t.Run("ensure proxy headers are updated", func(t *testing.T) {
		proxy, err := storage.GetProxy(context.Background(), "test", true)
		assert.NoError(t, err)
		assert.Equal(t, "test2", proxy.Headers[0].Value)
		assert.Equal(t, "test3", proxy.Headers[1].Value)
	})

	t.Run("delete proxy", func(t *testing.T) {
		proxy, err := storage.GetProxy(context.Background(), "test", false)
		assert.NoError(t, err)
		err = storage.DeleteProxy(context.Background(), proxy)
		assert.NoError(t, err)
	})

	t.Run("ensure list proxies return 0 element", func(t *testing.T) {
		proxies, err := storage.ListProxies(context.Background(), false)
		assert.NoError(t, err)
		assert.Equal(t, 0, len(proxies))
	})
}

func TestRoleStorage(t *testing.T) {
	storage, err := testPostgresStorage(t)
	assert.NoError(t, err)

	t.Run("insert role", func(t *testing.T) {
		role := RoleConfig{
			Name: "test",
			Permissions: []PermissionConfig{
				{
					ObjectType: ObjectTypeTools,
					ObjectName: "*",
					Proxy:      "*",
				},
			},
		}
		err := storage.SetRole(context.Background(), role)
		assert.NoError(t, err)
	})

	t.Run("ensure role is inserted", func(t *testing.T) {
		role, err := storage.GetRole(context.Background(), "test")
		assert.NoError(t, err)
		assert.Equal(t, "test", role.Name)
	})

	t.Run("ensure list roles return 1 element", func(t *testing.T) {
		roles, err := storage.ListRoles(context.Background())
		assert.NoError(t, err)
		assert.Equal(t, 1, len(roles))
		assert.Equal(t, "test", roles[0].Name)
	})

	t.Run("delete role", func(t *testing.T) {
		err := storage.DeleteRole(context.Background(), "test")
		assert.NoError(t, err)
	})

	t.Run("ensure list roles return 0 element", func(t *testing.T) {
		roles, err := storage.ListRoles(context.Background())
		assert.NoError(t, err)
		assert.Equal(t, 0, len(roles))
	})

	t.Run("insert with invalid permission", func(t *testing.T) {
		role := RoleConfig{
			Name: "test",
			Permissions: []PermissionConfig{
				{
					ObjectType: "invalid",
					ObjectName: "*",
					Proxy:      "*",
				},
			},
		}
		err := storage.SetRole(context.Background(), role)
		assert.Error(t, err)
	})
	t.Run("ensure no role is inserted", func(t *testing.T) {
		roles, err := storage.ListRoles(context.Background())
		assert.NoError(t, err)
		assert.Equal(t, 0, len(roles))
	})
}

func TestAttributeToRolesStorage(t *testing.T) {
	storage, err := testPostgresStorage(t)
	assert.NoError(t, err)

	t.Run("ensure failure when reference to non existing role", func(t *testing.T) {
		attributeToRoles := AttributeToRolesConfig{
			AttributeKey:   "test",
			AttributeValue: "test",
			Roles:          []string{"test"},
		}
		err := storage.SetAttributeToRoles(context.Background(), attributeToRoles)
		assert.Error(t, err)
	})

	t.Run("insert role", func(t *testing.T) {
		role := RoleConfig{
			Name: "test",
			Permissions: []PermissionConfig{
				{
					ObjectType: ObjectTypeTools,
					ObjectName: "*",
					Proxy:      "*",
				},
			},
		}
		err := storage.SetRole(context.Background(), role)
		assert.NoError(t, err)
	})

	t.Run("insert attribute to roles", func(t *testing.T) {
		attributeToRoles := AttributeToRolesConfig{
			AttributeKey:   "test",
			AttributeValue: "test",
			Roles:          []string{"test"},
		}
		err := storage.SetAttributeToRoles(context.Background(), attributeToRoles)
		assert.NoError(t, err)
	})

	t.Run("ensure attribute to roles is inserted", func(t *testing.T) {
		attributeToRoles, err := storage.GetAttributeToRoles(context.Background(), "test", "test")
		assert.NoError(t, err)
		assert.Equal(t, "test", attributeToRoles.AttributeKey)
		assert.Equal(t, "test", attributeToRoles.AttributeValue)
	})

	t.Run("delete attribute to roles", func(t *testing.T) {
		err := storage.DeleteAttributeToRoles(context.Background(), "test", "test")
		assert.NoError(t, err)
	})

	t.Run("ensure attribute to roles is deleted", func(t *testing.T) {
		_, err := storage.GetAttributeToRoles(context.Background(), "test", "test")
		assert.Error(t, err)
	})
}
