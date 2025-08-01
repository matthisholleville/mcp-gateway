package server

import (
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/matthisholleville/mcp-gateway/internal/storage"
)

func (s *Server) ConfigureRoutes(c *echo.Group) {
	admin := c.Group("/admin")
	admin.GET("/proxies", s.getProxies)
	admin.GET("/proxies/:name", s.getProxy)
	admin.PUT("/proxies/:name", s.upsertProxy)
	admin.DELETE("/proxies/:name", s.deleteProxy)

	admin.GET("/roles", s.getRoles)
	admin.PUT("/roles", s.upsertRole)
	admin.DELETE("/roles/:role", s.deleteRole)

	admin.GET("/attribute-to-roles", s.getAttributeToRoles)
	admin.PUT("/attribute-to-roles", s.upsertAttributeToRole)
	admin.DELETE("/attribute-to-roles/:attributeKey/:attributeValue", s.deleteAttributeToRole)
}

// @Summary		Get all proxies
// @Description	Get all proxies
// @Tags			proxies
// @Accept			json
// @Produce		json
// @Security		Authentication
// @Success		200	{array}	storage.ProxyConfig
// @Failure		500	{object}	map[string]string
// @Router			/v1/admin/proxies [get]
func (s *Server) getProxies(c echo.Context) error {
	proxies, err := s.Storage.ListProxies(c.Request().Context(), false)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}
	if len(proxies) == 0 {
		proxies = []storage.ProxyConfig{}
	}
	return c.JSON(http.StatusOK, proxies)
}

// @Summary		Get a proxy
// @Description	Get a proxy
// @Tags			proxies
// @Accept			json
// @Produce		json
// @Param			name	path	string	true	"Proxy name"
// @Success		200	{object}	storage.ProxyConfig
// @Failure		500	{object}	map[string]string
// @Security		Authentication
// @Router			/v1/admin/proxies/{name} [get]
func (s *Server) getProxy(c echo.Context) error {
	name := c.Param("name")
	proxy, err := s.Storage.GetProxy(c.Request().Context(), name, false)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}
	return c.JSON(http.StatusOK, proxy)
}

// @Summary		Upsert a proxy
// @Description	Upsert a proxy
// @Tags			proxies
// @Accept			json
// @Produce		json
// @Param			proxy	body	storage.ProxyConfig	true	"Proxy"
// @Success		200	{object}	storage.ProxyConfig
// @Failure		400	{object}	map[string]string
// @Failure		500	{object}	map[string]string
// @Security		Authentication
// @Router			/v1/admin/proxies/{name} [put]
func (s *Server) upsertProxy(c echo.Context) error {
	proxy := storage.ProxyConfig{}
	var err error
	if err := c.Bind(&proxy); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
	}

	err = s.Storage.SetProxy(c.Request().Context(), &proxy, true)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}
	return nil
}

// @Summary		Delete a proxy
// @Description	Delete a proxy
// @Tags			proxies
// @Accept			json
// @Produce		json
// @Param			name	path	string	true	"Proxy name"
// @Success		200	{object}	map[string]string
// @Failure		500	{object}	map[string]string
// @Security		Authentication
// @Router			/v1/admin/proxies/{name} [delete]
func (s *Server) deleteProxy(c echo.Context) error {
	name := c.Param("name")
	err := s.Storage.DeleteProxy(c.Request().Context(), name)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}
	return nil
}

// @Summary		Get all roles
// @Description	Get all roles
// @Tags			roles
// @Accept			json
// @Produce		json
// @Security		Authentication
// @Success		200	{array}	storage.RoleConfig
// @Failure		500	{object}	map[string]string
// @Router			/v1/admin/roles [get]
func (s *Server) getRoles(c echo.Context) error {
	roles, err := s.Storage.ListRoles(c.Request().Context())
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}
	return c.JSON(http.StatusOK, roles)
}

// @Summary		Upsert a role
// @Description	Upsert a role
// @Tags			roles
// @Accept			json
// @Produce		json
// @Param			role	body	storage.RoleConfig	true	"Role"
// @Success		200	{object}	storage.RoleConfig
// @Failure		400	{object}	map[string]string
// @Failure		500	{object}	map[string]string
// @Security		Authentication
// @Router			/v1/admin/roles [put]
func (s *Server) upsertRole(c echo.Context) error {
	role := storage.RoleConfig{}
	if err := c.Bind(&role); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
	}
	err := s.Storage.SetRole(c.Request().Context(), role)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}
	return nil
}

// @Summary		Delete a role
// @Description	Delete a role
// @Tags			roles
// @Accept			json
// @Produce		json
// @Param			role	path	string	true	"Role"
// @Success		200	{object}	map[string]string
// @Failure		400	{object}	map[string]string
// @Failure		500	{object}	map[string]string
// @Security		Authentication
// @Router			/v1/admin/roles/{role} [delete]
func (s *Server) deleteRole(c echo.Context) error {
	role := c.Param("role")
	if role == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "role is required"})
	}
	err := s.Storage.DeleteRole(c.Request().Context(), role)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}
	return nil
}

// @Summary		Get all attribute to roles
// @Description	Get all attribute to roles
// @Tags			attribute to roles
// @Accept			json
// @Produce		json
// @Security		Authentication
// @Success		200	{array}	storage.AttributeToRolesConfig
// @Failure		500	{object}	map[string]string
// @Router			/v1/admin/attribute-to-roles [get]
func (s *Server) getAttributeToRoles(c echo.Context) error {
	attributeToRoles, err := s.Storage.ListAttributeToRoles(c.Request().Context())
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}
	return c.JSON(http.StatusOK, attributeToRoles)
}

// @Summary		Upsert a attribute to role
// @Description	Upsert a attribute to role
// @Tags			attribute to roles
// @Accept			json
// @Produce		json
// @Param			attributeToRole	body	storage.AttributeToRolesConfig	true	"Attribute to role"
// @Success		200	{object}	storage.AttributeToRolesConfig
// @Failure		400	{object}	map[string]string
// @Failure		500	{object}	map[string]string
// @Security		Authentication
// @Router			/v1/admin/attribute-to-roles [put]
func (s *Server) upsertAttributeToRole(c echo.Context) error {
	attributeToRole := storage.AttributeToRolesConfig{}
	if err := c.Bind(&attributeToRole); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
	}

	err := s.Storage.SetAttributeToRoles(c.Request().Context(), attributeToRole)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}
	return nil
}

// @Summary		Delete a attribute to role
// @Description	Delete a attribute to role
// @Tags			attribute to roles
// @Accept			json
// @Produce		json
// @Param			attributeKey	path	string	true	"Attribute key"
// @Param			attributeValue	path	string	true	"Attribute value"
// @Success		200	{object}	map[string]string
// @Failure		400	{object}	map[string]string
// @Failure		500	{object}	map[string]string
// @Security		Authentication
// @Router			/v1/admin/attribute-to-roles/{attributeKey}/{attributeValue} [delete]
func (s *Server) deleteAttributeToRole(c echo.Context) error {
	attributeKey := c.Param("attributeKey")
	attributeValue := c.Param("attributeValue")
	if attributeKey == "" || attributeValue == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "attribute key and attribute value are required"})
	}
	err := s.Storage.DeleteAttributeToRoles(c.Request().Context(), attributeKey, attributeValue)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}
	return nil
}
