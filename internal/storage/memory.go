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

// GetProxy gets a proxy from the memory storage.
func (s *MemoryStorage) GetProxy(_ context.Context, proxy string, _ bool) (ProxyConfig, error) {
	proxyConfig, ok := s.proxies[proxy]
	if !ok {
		return ProxyConfig{}, fmt.Errorf("proxy not found")
	}
	return proxyConfig, nil
}

// SetProxy sets a proxy in the memory storage.
func (s *MemoryStorage) SetProxy(_ context.Context, proxy *ProxyConfig, _ bool) error {
	if !proxy.Type.IsValid() {
		return fmt.Errorf("invalid proxy type: %s", proxy.Type)
	}
	if !proxy.AuthType.IsValid() {
		return fmt.Errorf("invalid proxy auth type: %s", proxy.AuthType)
	}

	s.proxies[proxy.Name] = *proxy
	return nil
}

// DeleteProxy deletes a proxy from the memory storage.
func (s *MemoryStorage) DeleteProxy(_ context.Context, proxy string) error {
	delete(s.proxies, proxy)
	return nil
}

// ListProxies lists all proxies from the memory storage.
func (s *MemoryStorage) ListProxies(_ context.Context, _ bool) ([]ProxyConfig, error) {
	proxies := make([]ProxyConfig, 0, len(s.proxies))
	for _, proxy := range s.proxies {
		proxies = append(proxies, proxy)
	}
	return proxies, nil
}

// SetRole sets a role in the memory storage.
func (s *MemoryStorage) SetRole(_ context.Context, role RoleConfig) error {
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

// GetRole gets a role from the memory storage.
func (s *MemoryStorage) GetRole(_ context.Context, role string) (RoleConfig, error) {
	roleConfig, ok := s.roles[role]
	if !ok {
		return RoleConfig{}, fmt.Errorf("role not found")
	}
	return roleConfig, nil
}

// DeleteRole deletes a role from the memory storage.
func (s *MemoryStorage) DeleteRole(_ context.Context, role string) error {
	delete(s.roles, role)
	return nil
}

// ListRoles lists all roles from the memory storage.
func (s *MemoryStorage) ListRoles(_ context.Context) ([]RoleConfig, error) {
	roles := make([]RoleConfig, 0, len(s.roles))
	for _, role := range s.roles {
		roles = append(roles, role)
	}
	return roles, nil
}

// SetAttributeToRoles sets an attribute to roles in the memory storage.
func (s *MemoryStorage) SetAttributeToRoles(_ context.Context, attributeToRoles AttributeToRolesConfig) error {
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

// DeleteAttributeToRoles deletes an attribute to roles from the memory storage.
func (s *MemoryStorage) DeleteAttributeToRoles(_ context.Context, attributeKey, attributeValue string) error {
	delete(s.attributeToRoles, fmt.Sprintf("%s:%s", attributeKey, attributeValue))
	return nil
}

// ListAttributeToRoles lists all attribute to roles from the memory storage.
func (s *MemoryStorage) ListAttributeToRoles(_ context.Context) ([]AttributeToRolesConfig, error) {
	attributeToRoles := make([]AttributeToRolesConfig, 0, len(s.attributeToRoles))
	for _, attributeToRole := range s.attributeToRoles {
		attributeToRoles = append(attributeToRoles, attributeToRole)
	}
	return attributeToRoles, nil
}

// GetAttributeToRoles gets an attribute to roles from the memory storage.
func (s *MemoryStorage) GetAttributeToRoles(_ context.Context, attributeKey, attributeValue string) (AttributeToRolesConfig, error) {
	fmt.Println("GetAttributeToRoles", attributeKey, attributeValue)
	attributeToRoles, ok := s.attributeToRoles[fmt.Sprintf("%s:%s", attributeKey, attributeValue)]
	if !ok {
		return AttributeToRolesConfig{}, fmt.Errorf("attribute to roles not found")
	}
	return attributeToRoles, nil
}
