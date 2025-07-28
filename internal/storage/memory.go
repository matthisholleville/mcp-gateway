package storage

import (
	"context"
	"fmt"
)

type MemoryStorage struct {
	BaseStorage
	proxies      map[string]ProxyConfig
	roles        map[string]RoleConfig
	claimToRoles map[string]ClaimToRolesConfig
}

func NewMemoryStorage(defaultScope string) *MemoryStorage {
	return &MemoryStorage{
		BaseStorage: BaseStorage{
			defaultScope: defaultScope,
		},
		proxies:      make(map[string]ProxyConfig),
		roles:        make(map[string]RoleConfig),
		claimToRoles: make(map[string]ClaimToRolesConfig),
	}
}

func (s *MemoryStorage) GetProxy(ctx context.Context, proxy ProxyConfig) (ProxyConfig, error) {
	proxy, ok := s.proxies[proxy.Name]
	if !ok {
		return ProxyConfig{}, fmt.Errorf("proxy not found")
	}
	return proxy, nil
}

func (s *MemoryStorage) SetProxy(ctx context.Context, proxy ProxyConfig) error {
	s.proxies[proxy.Name] = proxy
	return nil
}

func (s *MemoryStorage) DeleteProxy(ctx context.Context, proxy ProxyConfig) error {
	delete(s.proxies, proxy.Name)
	return nil
}

func (s *MemoryStorage) ListProxies(ctx context.Context) ([]ProxyConfig, error) {
	proxies := make([]ProxyConfig, 0, len(s.proxies))
	for _, proxy := range s.proxies {
		proxies = append(proxies, proxy)
	}
	return proxies, nil
}

func (s *MemoryStorage) SetRole(ctx context.Context, role RoleConfig) error {
	_, ok := s.roles[role.Name]
	if ok {
		return fmt.Errorf("role already exists")
	}

	for _, permission := range role.Permissions {
		if permission.Proxy == "*" {
			continue
		}
		_, ok := s.proxies[permission.Proxy]
		if !ok {
			return fmt.Errorf("proxy %s not found", permission.Proxy)
		}
		if permission.ObjectType != ObjectTypeAll && permission.ObjectType != ObjectTypeTools {
			return fmt.Errorf("invalid object type")
		}
	}

	s.roles[role.Name] = role
	return nil
}

func (s *MemoryStorage) GetRole(ctx context.Context, role string) (RoleConfig, error) {
	roleConfig, ok := s.roles[role]
	if !ok {
		return RoleConfig{}, fmt.Errorf("role not found")
	}
	return roleConfig, nil
}

func (s *MemoryStorage) DeleteRole(ctx context.Context, role string) error {
	delete(s.roles, role)
	return nil
}

func (s *MemoryStorage) ListRoles(ctx context.Context) ([]RoleConfig, error) {
	roles := make([]RoleConfig, 0, len(s.roles))
	for _, role := range s.roles {
		roles = append(roles, role)
	}
	return roles, nil
}

func (s *MemoryStorage) SetClaimToRoles(ctx context.Context, claimToRoles ClaimToRolesConfig) error {
	_, ok := s.claimToRoles[fmt.Sprintf("%s:%s", claimToRoles.ClaimKey, claimToRoles.ClaimValue)]
	if ok {
		return fmt.Errorf("claim to roles already exists")
	}

	for _, role := range claimToRoles.Roles {
		_, ok := s.roles[role]
		if !ok {
			return fmt.Errorf("role not found")
		}
	}
	s.claimToRoles[fmt.Sprintf("%s:%s", claimToRoles.ClaimKey, claimToRoles.ClaimValue)] = claimToRoles
	return nil
}

func (s *MemoryStorage) DeleteClaimToRoles(ctx context.Context, claimKey, claimValue string) error {
	delete(s.claimToRoles, fmt.Sprintf("%s:%s", claimKey, claimValue))
	return nil
}

func (s *MemoryStorage) ListClaimToRoles(ctx context.Context) ([]ClaimToRolesConfig, error) {
	claimToRoles := make([]ClaimToRolesConfig, 0, len(s.claimToRoles))
	for _, claimToRole := range s.claimToRoles {
		claimToRoles = append(claimToRoles, claimToRole)
	}
	return claimToRoles, nil
}

func (s *MemoryStorage) GetClaimToRoles(ctx context.Context, claimKey, claimValue string) (ClaimToRolesConfig, error) {
	fmt.Println("GetClaimToRoles", claimKey, claimValue)
	claimToRoles, ok := s.claimToRoles[fmt.Sprintf("%s:%s", claimKey, claimValue)]
	if !ok {
		return ClaimToRolesConfig{}, fmt.Errorf("claim to roles not found")
	}
	return claimToRoles, nil
}
