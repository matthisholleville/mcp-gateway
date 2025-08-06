package auth

import (
	"context"
	"fmt"
	"sync"

	"github.com/matthisholleville/mcp-gateway/internal/storage"
	"github.com/matthisholleville/mcp-gateway/pkg/logger"
	"go.uber.org/zap"
	"golang.org/x/sync/errgroup"
)

// BaseProvider is the base provider for the MCP Gateway
type BaseProvider struct {
	logger  logger.Logger
	storage storage.Interface
}

// VerifyPermissions verifies the permissions of a user for a tool
func (b *BaseProvider) VerifyPermissions(
	ctx context.Context,
	objectType, proxy, objectName string,
	claims map[string]interface{},
) bool {
	b.logger.Debug("Verifying permissions",
		zap.String("objectType", objectType),
		zap.String("proxy", proxy),
		zap.String("objectName", objectName),
		zap.Any("claims", claims))
	roles := b.attributeToRoles(ctx, claims)

	if len(roles) == 0 {
		b.logger.Debug("No roles found for claims", zap.Any("claims", claims))
		return false
	}

	b.logger.Debug("Found roles for claims", zap.Strings("roles", roles))

	// Resolve all roles in parallel ‑ stored in a thread‑safe slice.
	type rolePerm struct {
		name        string
		permissions []storage.PermissionConfig
	}
	var (
		mu   sync.Mutex
		list []rolePerm
	)
	g, ctx := errgroup.WithContext(ctx)

	for _, roleName := range roles {
		g.Go(func() error {
			role, err := b.storage.GetRole(ctx, roleName)
			if err != nil {
				return fmt.Errorf("GetRole(%s): %w", roleName, err)
			}
			mu.Lock()
			list = append(list, rolePerm{roleName, role.Permissions})
			mu.Unlock()
			return nil
		})
	}

	if err := g.Wait(); err != nil {
		b.logger.Error("role fetch failed", zap.Error(err))
		return false
	}

	// Check if the user has the permission for the object type, object name and proxy
	for _, r := range list {
		for _, p := range r.permissions {
			if b.match(string(p.ObjectType), objectType) &&
				b.match(p.Proxy, proxy) &&
				b.match(p.ObjectName, objectName) {
				b.logger.Debug("permission OK", zap.String("role", r.name))
				return true
			}
		}
	}

	return false
}

// match handles the wildcard "*"
func (b *BaseProvider) match(pattern, value string) bool {
	return pattern == "*" || pattern == value
}

// attributeToRoles converts the claims into attribute to roles
func (b *BaseProvider) attributeToRoles(
	ctx context.Context,
	claims map[string]interface{},
) []string {
	out := make(map[string]struct{}) // set

	for claim, raw := range claims {
		switch v := raw.(type) {
		case string:
			b.appendRoles(out, b.lookup(ctx, claim, v))

		case bool: // true/false become "true"/"false"
			b.appendRoles(out, b.lookup(ctx, claim, fmt.Sprintf("%t", v)))

		case []string:
			for _, s := range v {
				b.appendRoles(out, b.lookup(ctx, claim, s))
			}

		case []interface{}:
			for _, any := range v {
				b.appendRoles(out, b.lookup(ctx, claim, fmt.Sprint(any)))
			}

		default:
			b.logger.Debug("unsupported claim type",
				zap.String("claim", claim),
				zap.Any("value", raw))
		}
	}

	roles := make([]string, 0, len(out))
	for r := range out {
		roles = append(roles, r)
	}
	return roles
}

// TODO: Actually we query the DB so multiple times (1 call perm), we could cache the results and search in memory
func (b *BaseProvider) lookup(
	ctx context.Context,
	claim, value string,
) []string {
	mapping, err := b.storage.GetAttributeToRoles(ctx, claim, value)
	b.logger.Debug("looking up attribute to roles",
		zap.String("claim", claim),
		zap.String("value", value),
		zap.Any("mapping", mapping),
		zap.Error(err))
	if err != nil || len(mapping.Roles) == 0 {
		b.logger.Debug("GetAttributeToRoles failed",
			zap.String("claim", claim),
			zap.String("value", value),
			zap.Error(err))
		return []string{}
	}
	return mapping.Roles
}

func (b *BaseProvider) appendRoles(dst map[string]struct{}, roles []string) {
	for _, r := range roles {
		dst[r] = struct{}{}
	}
}
