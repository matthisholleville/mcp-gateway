package storage

import (
	"context"
	"fmt"
)

type MemoryStorage struct {
	BaseStorage
	proxies          map[string]ProxyConfig
	roles            map[string]RoleConfig
	attributeToRoles map[string]AttributeToRolesConfig
}

func NewMemoryStorage(defaultScope string) *MemoryStorage {
	return &MemoryStorage{
		BaseStorage: BaseStorage{
			defaultScope: defaultScope,
		},
		proxies:          make(map[string]ProxyConfig),
		roles:            make(map[string]RoleConfig),
		attributeToRoles: make(map[string]AttributeToRolesConfig),
	}
}

func (s *MemoryStorage) GetProxy(ctx context.Context, proxy string, decrypt bool) (ProxyConfig, error) {
	proxyConfig, ok := s.proxies[proxy]
	if !ok {
		return ProxyConfig{}, fmt.Errorf("proxy not found")
	}
	return proxyConfig, nil
}

func (s *MemoryStorage) SetProxy(ctx context.Context, proxy ProxyConfig, encrypt bool) error {
	if !proxy.Type.IsValid() {
		return fmt.Errorf("invalid proxy type: %s", proxy.Type)
	}
	if !proxy.AuthType.IsValid() {
		return fmt.Errorf("invalid proxy auth type: %s", proxy.AuthType)
	}

	s.proxies[proxy.Name] = proxy
	return nil
}

func (s *MemoryStorage) DeleteProxy(ctx context.Context, proxy ProxyConfig) error {
	delete(s.proxies, proxy.Name)
	return nil
}

func (s *MemoryStorage) ListProxies(ctx context.Context, decrypt bool) ([]ProxyConfig, error) {
	proxies := make([]ProxyConfig, 0, len(s.proxies))
	for _, proxy := range s.proxies {
		proxies = append(proxies, proxy)
	}
	return proxies, nil
}

func (s *MemoryStorage) SetRole(ctx context.Context, role RoleConfig) error {
	for _, permission := range role.Permissions {
		if !permission.ObjectType.IsValid() {
			return fmt.Errorf("invalid object type: %s", permission.ObjectType)
		}
	}

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

func (s *MemoryStorage) SetAttributeToRoles(ctx context.Context, attributeToRoles AttributeToRolesConfig) error {
	_, ok := s.attributeToRoles[fmt.Sprintf("%s:%s", attributeToRoles.AttributeKey, attributeToRoles.AttributeValue)]
	if ok {
		return fmt.Errorf("attribute to roles already exists")
	}

	for _, role := range attributeToRoles.Roles {
		_, ok := s.roles[role]
		if !ok {
			return fmt.Errorf("role not found")
		}
	}
	s.attributeToRoles[fmt.Sprintf("%s:%s", attributeToRoles.AttributeKey, attributeToRoles.AttributeValue)] = attributeToRoles
	return nil
}

func (s *MemoryStorage) DeleteAttributeToRoles(ctx context.Context, attributeKey, attributeValue string) error {
	delete(s.attributeToRoles, fmt.Sprintf("%s:%s", attributeKey, attributeValue))
	return nil
}

func (s *MemoryStorage) ListAttributeToRoles(ctx context.Context) ([]AttributeToRolesConfig, error) {
	attributeToRoles := make([]AttributeToRolesConfig, 0, len(s.attributeToRoles))
	for _, attributeToRole := range s.attributeToRoles {
		attributeToRoles = append(attributeToRoles, attributeToRole)
	}
	return attributeToRoles, nil
}

func (s *MemoryStorage) GetAttributeToRoles(ctx context.Context, attributeKey, attributeValue string) (AttributeToRolesConfig, error) {
	fmt.Println("GetAttributeToRoles", attributeKey, attributeValue)
	attributeToRoles, ok := s.attributeToRoles[fmt.Sprintf("%s:%s", attributeKey, attributeValue)]
	if !ok {
		return AttributeToRolesConfig{}, fmt.Errorf("attribute to roles not found")
	}
	return attributeToRoles, nil
}
