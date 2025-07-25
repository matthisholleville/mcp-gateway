package oauth

import (
	"context"
	"log/slog"
	"path/filepath"

	"github.com/matthisholleville/mcp-gateway/internal/cfg"
)

// BaseProvider is the base provider for the MCP Gateway
type BaseProvider struct {
	authCfg *cfg.Auth
	logger  *slog.Logger
}

// VerifyPermissions verifies the permissions of a user for a tool
func (b *BaseProvider) VerifyPermissions(toolName string, claims map[string]interface{}) bool {
	ctx := context.Background()
	flattenedClaims, err := flattenClaims(claims)
	if err != nil {
		return false
	}

	b.logger.InfoContext(ctx, "Flattened claims", slog.Any("claims", flattenedClaims))

	requiredScopes := b.getRequiredScopes(toolName)
	if len(requiredScopes) == 0 {
		return b.authCfg.Options.DefaultScope != nil
	}

	b.logger.InfoContext(ctx, "Required scopes", slog.Any("scopes", requiredScopes))

	userScopes := b.getUserScopes(flattenedClaims)
	hasRequiredPermission := b.hasRequiredPermission(userScopes, requiredScopes, b.authCfg.Options.ScopeMode)

	return hasRequiredPermission
}

// getRequiredScopes gets the required scopes for a tool
func (b *BaseProvider) getRequiredScopes(toolName string) []string {
	// Direct match first
	if scopes, exists := b.authCfg.Permissions[toolName]; exists {
		return scopes
	}

	// Pattern matching with wildcards
	for pattern, scopes := range b.authCfg.Permissions {
		match, _ := filepath.Match(pattern, toolName)
		if match {
			return scopes
		}
	}

	return []string{}
}

// getUserScopes gets the user scopes from the flattened claims
func (b *BaseProvider) getUserScopes(flattenedClaims []string) []string {
	scopeSet := make(map[string]bool)
	for _, group := range flattenedClaims {
		if scopes, exists := b.authCfg.Mappings[group]; exists {
			for _, scope := range scopes {
				scopeSet[scope] = true
			}
		}
	}

	// Convert map to slice
	userScopes := make([]string, 0, len(scopeSet))
	for scope := range scopeSet {
		userScopes = append(userScopes, scope)
	}
	return userScopes
}

// hasAllScopes checks if user has all required scopes (AND logic)
func (b *BaseProvider) hasAllScopes(userScopes, requiredScopes []string) bool {
	userScopeSet := make(map[string]bool)
	for _, scope := range userScopes {
		userScopeSet[scope] = true
	}

	for _, requiredScope := range requiredScopes {
		if !userScopeSet[requiredScope] {
			return false
		}
	}

	return true
}

// hasAnyScope checks if user has any of the required scopes (OR logic)
func (b *BaseProvider) hasAnyScope(userScopes, requiredScopes []string) bool {
	userScopeSet := make(map[string]bool)
	for _, scope := range userScopes {
		userScopeSet[scope] = true
	}

	for _, requiredScope := range requiredScopes {
		if userScopeSet[requiredScope] {
			return true
		}
	}

	return false
}

// hasRequiredPermission checks if the user has the required permission
func (b *BaseProvider) hasRequiredPermission(userScopes, requiredScopes []string, scopeMode string) bool {
	if len(requiredScopes) == 0 {
		return true
	}

	switch scopeMode {
	case "all":
		// AND logic - user must have ALL required scopes
		return b.hasAllScopes(userScopes, requiredScopes)
	default:
		// OR logic - user must have ANY of the required scopes
		return b.hasAnyScope(userScopes, requiredScopes)
	}
}
