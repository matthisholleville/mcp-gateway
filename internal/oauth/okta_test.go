package oauth

import (
	"log/slog"
	"os"
	"testing"

	"github.com/matthisholleville/mcp-gateway/internal/cfg"
	"github.com/stretchr/testify/assert"
)

// TestOktaProvider_VerifyClaims tests the verifyClaims method of the OktaProvider
func TestOktaProvider_VerifyClaims(t *testing.T) {
	for _, test := range []struct {
		name      string
		cfgClaims []string
		claims    map[string]interface{}
		expected  map[string]interface{}
		expectErr bool
	}{
		{
			name:      "valid claims",
			cfgClaims: []string{"groups"},
			claims: map[string]interface{}{
				"groups": []string{"Base"},
			},
			expected:  map[string]interface{}{"groups": []string{"Base"}},
			expectErr: false,
		},
		{
			name:      "missing claims",
			cfgClaims: []string{"groups", "missing"},
			claims: map[string]interface{}{
				"groups": []string{"Base"},
			},
			expected:  nil,
			expectErr: true,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			provider := &OktaProvider{
				cfg: &cfg.Okta{
					Issuer: "https://dev-1234567890.okta.com",
					OrgURL: "https://dev-1234567890.okta.com",
				},
				authCfg: &cfg.Auth{
					Claims: test.cfgClaims,
				},
				logger: slog.New(slog.NewTextHandler(os.Stdout, nil)),
			}

			claims, err := provider.verifyClaims(&Jwt{Claims: test.claims})
			if test.expectErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, test.expected, claims)
			}
		})
	}
}

// TestOktaProvider_VerifyPermissions tests the VerifyPermissions method of the OktaProvider
func TestOktaProvider_VerifyPermissions(t *testing.T) {
	for _, test := range []struct {
		name      string
		cfgClaims []string
		authCfg   *cfg.Auth
		claims    map[string]interface{}
		toolName  string
		expected  bool
	}{
		{
			name:      "valid claims",
			cfgClaims: []string{"groups"},
			claims: map[string]interface{}{
				"groups": []string{"Base"},
			},
			authCfg: &cfg.Auth{
				Claims: []string{"groups"},
				Permissions: map[string][]string{
					"*": []string{"scope:1"},
				},
				Mappings: map[string][]string{
					"groups:Base": []string{"scope:1"},
				},
				Options: cfg.Options{
					ScopeMode: "any",
				},
			},
			toolName: "test",
			expected: true,
		},
		{
			name:      "invalid permissions",
			cfgClaims: []string{"groups"},
			claims: map[string]interface{}{
				"groups": []string{"Base"},
			},
			authCfg: &cfg.Auth{
				Claims: []string{"groups"},
				Permissions: map[string][]string{
					"*": []string{"scope:1"},
				},
				Mappings: map[string][]string{
					"groups:Engineering": []string{"scope:1"},
				},
				Options: cfg.Options{
					ScopeMode: "any",
				},
			},
			toolName: "test",
			expected: false,
		},
		{
			name:      "valid with multiple groups",
			cfgClaims: []string{"groups"},
			claims: map[string]interface{}{
				"groups": []string{"Base", "Engineering"},
			},
			authCfg: &cfg.Auth{
				Claims: []string{"groups"},
				Permissions: map[string][]string{
					"*": []string{"scope:1"},
				},
				Mappings: map[string][]string{
					"groups:Engineering": []string{"scope:1"},
				},
				Options: cfg.Options{
					ScopeMode: "any",
				},
			},
			toolName: "test",
			expected: true,
		},
		{
			name:      "invalid permissions for specific tool",
			cfgClaims: []string{"groups"},
			claims: map[string]interface{}{
				"groups": []string{"Base", "Engineering"},
			},
			authCfg: &cfg.Auth{
				Claims: []string{"groups"},
				Permissions: map[string][]string{
					"test": []string{"scope:1"},
				},
				Mappings: map[string][]string{
					"groups:Engineering": []string{"scope:1"},
				},
				Options: cfg.Options{
					ScopeMode: "any",
				},
			},
			toolName: "private-tool",
			expected: false,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			provider := &OktaProvider{
				BaseProvider: BaseProvider{
					authCfg: test.authCfg,
					logger:  slog.New(slog.NewTextHandler(os.Stdout, nil)),
				},
				cfg: &cfg.Okta{
					Issuer: "https://dev-1234567890.okta.com",
					OrgURL: "https://dev-1234567890.okta.com",
				},
				authCfg: test.authCfg,
				logger:  slog.New(slog.NewTextHandler(os.Stdout, nil)),
			}

			claims := provider.VerifyPermissions(test.toolName, test.claims)
			assert.Equal(t, test.expected, claims)
		})
	}
}
