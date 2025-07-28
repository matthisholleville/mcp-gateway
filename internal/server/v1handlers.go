package server

import (
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/matthisholleville/mcp-gateway/internal/storage"
)

func (s *Server) ConfigureRoutes(c *echo.Group) {
	admin := c.Group("/admin")
	admin.GET("/proxies", s.GetProxies)
	admin.GET("/proxies/:name", s.GetProxy)
	admin.PUT("/proxies/:name", s.UpsertProxy)
	admin.DELETE("/proxies/:name", s.DeleteProxy)

	admin.GET("/roles", s.GetRoles)
	admin.PUT("/roles", s.UpsertRole)
	admin.DELETE("/roles/:role", s.DeleteRole)

	admin.GET("/claim-to-roles", s.GetClaimToRoles)
	admin.PUT("/claim-to-roles", s.UpsertClaimToRole)
	admin.DELETE("/claim-to-roles/:claimKey/:claimValue", s.DeleteClaimToRole)
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
func (s *Server) GetProxies(c echo.Context) error {
	proxies, err := s.Storage.ListProxies(c.Request().Context())
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
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
func (s *Server) GetProxy(c echo.Context) error {
	name := c.Param("name")
	proxy, err := s.Storage.GetProxy(c.Request().Context(), storage.ProxyConfig{Name: name})
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
func (s *Server) UpsertProxy(c echo.Context) error {
	proxy := storage.ProxyConfig{}
	if err := c.Bind(&proxy); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
	}
	err := s.Storage.SetProxy(c.Request().Context(), proxy)
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
func (s *Server) DeleteProxy(c echo.Context) error {
	name := c.Param("name")
	err := s.Storage.DeleteProxy(c.Request().Context(), storage.ProxyConfig{Name: name})
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
func (s *Server) GetRoles(c echo.Context) error {
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
func (s *Server) UpsertRole(c echo.Context) error {
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
func (s *Server) DeleteRole(c echo.Context) error {
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

// @Summary		Get all claim to roles
// @Description	Get all claim to roles
// @Tags			claim to roles
// @Accept			json
// @Produce		json
// @Security		Authentication
// @Success		200	{array}	storage.ClaimToRolesConfig
// @Failure		500	{object}	map[string]string
// @Router			/v1/admin/claim-to-roles [get]
func (s *Server) GetClaimToRoles(c echo.Context) error {
	claimToRoles, err := s.Storage.ListClaimToRoles(c.Request().Context())
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}
	return c.JSON(http.StatusOK, claimToRoles)
}

// @Summary		Upsert a claim to role
// @Description	Upsert a claim to role
// @Tags			claim to roles
// @Accept			json
// @Produce		json
// @Param			claimToRole	body	storage.ClaimToRolesConfig	true	"Claim to role"
// @Success		200	{object}	storage.ClaimToRolesConfig
// @Failure		400	{object}	map[string]string
// @Failure		500	{object}	map[string]string
// @Security		Authentication
// @Router			/v1/admin/claim-to-roles [put]
func (s *Server) UpsertClaimToRole(c echo.Context) error {
	claimToRole := storage.ClaimToRolesConfig{}
	if err := c.Bind(&claimToRole); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
	}

	err := s.Storage.SetClaimToRoles(c.Request().Context(), claimToRole)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}
	return nil
}

// @Summary		Delete a claim to role
// @Description	Delete a claim to role
// @Tags			claim to roles
// @Accept			json
// @Produce		json
// @Param			claimKey	path	string	true	"Claim key"
// @Param			claimValue	path	string	true	"Claim value"
// @Success		200	{object}	map[string]string
// @Failure		400	{object}	map[string]string
// @Failure		500	{object}	map[string]string
// @Security		Authentication
// @Router			/v1/admin/claim-to-roles/{claimKey}/{claimValue} [delete]
func (s *Server) DeleteClaimToRole(c echo.Context) error {
	claimKey := c.Param("claimKey")
	claimValue := c.Param("claimValue")
	if claimKey == "" || claimValue == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "claim key and claim value are required"})
	}
	err := s.Storage.DeleteClaimToRoles(c.Request().Context(), claimKey, claimValue)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}
	return nil
}
