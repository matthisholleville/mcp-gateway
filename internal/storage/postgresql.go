package storage

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/lib/pq"
	"github.com/matthisholleville/mcp-gateway/internal/cfg"
	"github.com/matthisholleville/mcp-gateway/pkg/aescipher"
	"github.com/matthisholleville/mcp-gateway/pkg/logger"
	"go.uber.org/zap"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"
)

// PostgresStorage is a storage implementation for Postgres.
type PostgresStorage struct {
	BaseStorage
	db        *gorm.DB
	encryptor aescipher.Cryptor
	logger    logger.Logger
}

// NewPostgresStorage creates a new Postgres storage instance.
//
//nolint:gocritic // we need to keep logger as a parameter for the function
func NewPostgresStorage(defaultScope string, logger logger.Logger, cfg *cfg.Config, encryptor aescipher.Cryptor) (*PostgresStorage, error) {
	gormLogger := gormlogger.New(logger, gormlogger.Config{
		LogLevel: gormlogger.Warn,
	})
	db, err := gorm.Open(postgres.Open(cfg.BackendConfig.URI), &gorm.Config{
		Logger: gormLogger,
	})
	if err != nil {
		return nil, err
	}
	sqlDB, err := db.DB()
	if err != nil {
		return nil, err
	}
	sqlDB.SetMaxIdleConns(cfg.BackendConfig.MaxIdleConns)
	sqlDB.SetMaxOpenConns(cfg.BackendConfig.MaxOpenConns)
	sqlDB.SetConnMaxIdleTime(cfg.BackendConfig.ConnMaxIdleTime)
	sqlDB.SetConnMaxLifetime(cfg.BackendConfig.ConnMaxLifetime)

	if encryptor == nil {
		return nil, fmt.Errorf("encryptor is nil")
	}

	return &PostgresStorage{
		BaseStorage: BaseStorage{defaultScope: defaultScope},
		db:          db,
		encryptor:   encryptor,
		logger:      logger,
	}, nil
}

// GetDefaultScope gets the default scope from the Postgres storage.
func (s *PostgresStorage) GetDefaultScope(_ context.Context) string {
	return s.defaultScope
}

// GetProxy gets a proxy from the Postgres storage.
func (s *PostgresStorage) GetProxy(ctx context.Context, name string, decrypt bool) (ProxyConfig, error) {
	s.logger.Debug("GetProxy", zap.String("name", name), zap.Bool("decrypt", decrypt))
	const q = `
		SELECT
			p.name,
			p.type,
			p.url,
			p.timeout,
			p.authtype,
			COALESCE(ph.headers, '[]') AS headers_json,
			po.oauth                   AS oauth_json
		FROM mcp_gateway.proxy p
		LEFT JOIN LATERAL (
			SELECT json_agg(
				json_build_object('key', headerkey, 'value', headervalue)
				ORDER BY headerkey
			) AS headers
			FROM mcp_gateway.proxy_header
			WHERE proxyname = p.name
		) ph ON TRUE
		LEFT JOIN LATERAL (
			SELECT json_build_object(
				'clientId',      clientid,
				'clientSecret',  clientsecret,
				'tokenEndpoint', tokenendpoint,
				'scopes',        scopes
			) AS oauth
			FROM mcp_gateway.proxy_oauth
			WHERE proxyname = p.name
		) po ON TRUE
		WHERE p.name = $1;
	`

	var row struct {
		Name        string
		Type        string
		URL         string
		Timeout     int64
		AuthType    string `gorm:"column:authtype"`
		HeadersJSON []byte
		OAuthJSON   []byte
	}

	if err := s.db.WithContext(ctx).Raw(q, name).Scan(&row).Error; err != nil {
		return ProxyConfig{}, err
	}
	if row.Name == "" {
		return ProxyConfig{}, gorm.ErrRecordNotFound
	}

	var hdrs []ProxyHeader
	_ = json.Unmarshal(row.HeadersJSON, &hdrs)

	if decrypt {
		hdrs, err := s.decryptHeaders(hdrs)
		if err != nil {
			return ProxyConfig{}, err
		}
		row.HeadersJSON, _ = json.Marshal(hdrs)
	}

	var oauth *ProxyOAuth
	if len(row.OAuthJSON) > 0 && string(row.OAuthJSON) != "null" {
		oauth = new(ProxyOAuth)
		_ = json.Unmarshal(row.OAuthJSON, oauth)
	}

	return ProxyConfig{
		Name:     row.Name,
		Type:     ProxyType(row.Type),
		URL:      row.URL,
		Timeout:  time.Duration(row.Timeout) * time.Second,
		AuthType: ProxyAuthType(row.AuthType),
		Headers:  hdrs,
		OAuth:    oauth,
	}, nil
}

// ListProxies lists all proxies from the Postgres storage.
func (s *PostgresStorage) ListProxies(ctx context.Context, decrypt bool) ([]ProxyConfig, error) {
	s.logger.Debug("ListProxies", zap.Bool("decrypt", decrypt))
	const q = `
		SELECT
			p.name,
			p.type,
			p.url,
			p.timeout,
			p.authtype,
			COALESCE(ph.headers, '[]')   AS headers_json,
			po.oauth                     AS oauth_json
		FROM mcp_gateway.proxy p
		LEFT JOIN LATERAL (
			SELECT json_agg(
				json_build_object('key', headerkey, 'value', headervalue)
				ORDER BY headerkey
			) AS headers
			FROM mcp_gateway.proxy_header
			WHERE proxyname = p.name
		) ph ON TRUE
		LEFT JOIN LATERAL (
			SELECT json_build_object(
				'clientId',      clientid,
				'clientSecret',  clientsecret,
				'tokenEndpoint', tokenendpoint,
				'scopes',        scopes
			) AS oauth
			FROM mcp_gateway.proxy_oauth
			WHERE proxyname = p.name
		) po ON TRUE
		ORDER BY p.name;
	`

	type row struct {
		Name        string
		Type        string
		URL         string
		Timeout     int64
		AuthType    string
		HeadersJSON []byte
		OAuthJSON   []byte
	}

	var rows []row
	if err := s.db.WithContext(ctx).Raw(q).Scan(&rows).Error; err != nil {
		return nil, err
	}

	out := make([]ProxyConfig, 0, len(rows))
	for _, r := range rows {
		var hdrs []ProxyHeader
		_ = json.Unmarshal(r.HeadersJSON, &hdrs)

		var oauth *ProxyOAuth
		if len(r.OAuthJSON) > 0 && string(r.OAuthJSON) != "null" {
			oauth = new(ProxyOAuth)
			_ = json.Unmarshal(r.OAuthJSON, oauth)
		}

		out = append(out, ProxyConfig{
			Name:     r.Name,
			Type:     ProxyType(r.Type),
			URL:      r.URL,
			Timeout:  time.Duration(r.Timeout) * time.Second,
			AuthType: ProxyAuthType(r.AuthType),
			Headers:  hdrs,
			OAuth:    oauth,
		})
	}

	if decrypt {
		for i, p := range out {
			hdrs, err := s.decryptHeaders(p.Headers)
			if err != nil {
				return nil, err
			}
			out[i].Headers = hdrs
		}
	}
	return out, nil
}

// decryptHeaders decrypts the headers of a proxy.
func (s *PostgresStorage) decryptHeaders(headers []ProxyHeader) ([]ProxyHeader, error) {
	for i, h := range headers {
		value, err := s.decryptIfNeeded(h.Value)
		if err != nil {
			return nil, err
		}
		headers[i].Key = h.Key
		headers[i].Value = value
	}
	return headers, nil
}

// SetProxy sets a proxy in the Postgres storage.
func (s *PostgresStorage) SetProxy(ctx context.Context, p *ProxyConfig, encrypt bool) error {
	s.logger.Debug("SetProxy", zap.Any("proxy", p.Name), zap.Bool("encrypt", encrypt))
	if err := s.validateSetProxy(p); err != nil {
		return err
	}

	if encrypt {
		for i, h := range p.Headers {
			value, err := s.encryptIfNeeded(h.Value)
			if err != nil {
				return err
			}
			p.Headers[i].Value = value
		}
	}

	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Exec(`
			INSERT INTO mcp_gateway.proxy (name, type, url, timeout, authtype)
			VALUES ($1,$2,$3,$4,$5)
			ON CONFLICT (name) DO UPDATE SET
			    type     = EXCLUDED.type,
			    url      = EXCLUDED.url,
			    timeout  = EXCLUDED.timeout,
			    authtype = EXCLUDED.authtype
		`, p.Name, string(p.Type), p.URL, int64(p.Timeout/time.Second), string(p.AuthType)).Error; err != nil {
			return err
		}

		keys := make([]string, len(p.Headers))
		values := make([]string, len(p.Headers))
		for i, h := range p.Headers {
			keys[i], values[i] = h.Key, h.Value
		}

		if err := tx.Exec(`
			WITH data AS (
				SELECT
					$1::text AS proxyname,
					unnest(COALESCE($2::text[], ARRAY[]::text[])) AS headerkey,
					unnest(COALESCE($3::text[], ARRAY[]::text[])) AS headervalue
			), up AS (
				INSERT INTO mcp_gateway.proxy_header (proxyname, headerkey, headervalue)
				SELECT proxyname, headerkey, headervalue FROM data
				ON CONFLICT (proxyname, headerkey)
				     DO UPDATE SET headervalue = EXCLUDED.headervalue
				RETURNING headerkey
			)
			DELETE FROM mcp_gateway.proxy_header
			WHERE proxyname = $1
			  AND headerkey NOT IN (SELECT headerkey FROM up)
		`, p.Name, pq.Array(keys), pq.Array(values)).Error; err != nil {
			return err
		}

		if p.OAuth != nil {
			return tx.Exec(`
				INSERT INTO mcp_gateway.proxy_oauth (proxyname, clientid, clientsecret,
				                                     tokenendpoint, scopes)
				VALUES ($1,$2,$3,$4,$5)
				ON CONFLICT (proxyname) DO UPDATE SET
				      clientid      = EXCLUDED.clientid,
				      clientsecret  = EXCLUDED.clientsecret,
				      tokenendpoint = EXCLUDED.tokenendpoint,
				      scopes        = EXCLUDED.scopes
			`, p.Name, p.OAuth.ClientID, p.OAuth.ClientSecret,
				p.OAuth.TokenEndpoint, p.OAuth.Scopes).Error
		}
		return tx.Exec(`DELETE FROM mcp_gateway.proxy_oauth WHERE proxyname = $1`, p.Name).Error
	})
}

func (s *PostgresStorage) validateSetProxy(p *ProxyConfig) error {
	if !p.Type.IsValid() {
		return fmt.Errorf("invalid proxy type: %s", p.Type)
	}
	if !p.AuthType.IsValid() {
		return fmt.Errorf("invalid proxy auth type: %s", p.AuthType)
	}
	return nil
}

// DeleteProxy deletes a proxy from the Postgres storage.
func (s *PostgresStorage) DeleteProxy(ctx context.Context, proxy string) error {
	s.logger.Debug("DeleteProxy", zap.Any("proxy", proxy))
	tx := s.db.WithContext(ctx).Begin()
	if tx.Error != nil {
		return tx.Error
	}
	defer tx.Rollback()

	tx = tx.Exec(`
        DELETE FROM mcp_gateway.proxy WHERE name = $1
    `, proxy)
	if tx.Error != nil {
		return tx.Error
	}

	return tx.Commit().Error
}

// GetRole gets a role from the Postgres storage.
func (s *PostgresStorage) GetRole(ctx context.Context, role string) (RoleConfig, error) {
	s.logger.Debug("GetRole", zap.String("role", role))
	query := `
		SELECT 
			r.name,
			rp.objecttype,
			rp.proxyname,
			rp.objectname
		FROM mcp_gateway.role r
		LEFT JOIN mcp_gateway.role_permission rp ON r.name = rp.rolename
		WHERE r.name = $1
		ORDER BY rp.objecttype ASC, rp.proxyname ASC, rp.objectname ASC
	`

	rows, err := s.db.WithContext(ctx).Raw(query, role).Rows()
	if err != nil {
		return RoleConfig{}, err
	}
	defer rows.Close() //nolint:errcheck // no need to check the error here

	var result RoleConfig
	var permissions []PermissionConfig
	var firstRow = true

	for rows.Next() {
		var (
			name                          string
			objectType, proxy, objectName sql.NullString
		)

		if err := rows.Scan(&name, &objectType, &proxy, &objectName); err != nil {
			return RoleConfig{}, err
		}

		// Fill the main data (once)
		if firstRow {
			result = RoleConfig{Name: name}
			firstRow = false
		}

		// Add permission if present
		if objectType.Valid && proxy.Valid && objectName.Valid {
			permissions = append(permissions, PermissionConfig{
				ObjectType: ObjectType(objectType.String),
				Proxy:      proxy.String,
				ObjectName: objectName.String,
			})
		}
	}

	if err := rows.Err(); err != nil {
		return RoleConfig{}, err
	}

	if result.Name == "" {
		return RoleConfig{}, gorm.ErrRecordNotFound
	}

	result.Permissions = permissions
	return result, nil
}

// SetRole sets a role in the Postgres storage.
func (s *PostgresStorage) SetRole(ctx context.Context, role RoleConfig) error {
	s.logger.Debug("SetRole", zap.Any("role", role.Name))
	for _, p := range role.Permissions {
		if !p.ObjectType.IsValid() {
			return fmt.Errorf("invalid object type: %s", p.ObjectType)
		}
	}

	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Exec(`
			INSERT INTO mcp_gateway.role (name)
			VALUES ($1)
			ON CONFLICT (name) DO NOTHING
		`, role.Name).Error; err != nil {
			return err
		}

		if len(role.Permissions) == 0 {
			return tx.Exec(`
				DELETE FROM mcp_gateway.role_permission
				WHERE rolename = $1
			`, role.Name).Error
		}

		objTypes := make([]string, len(role.Permissions))
		proxies := make([]string, len(role.Permissions))
		objNames := make([]string, len(role.Permissions))
		for i, p := range role.Permissions {
			objTypes[i] = string(p.ObjectType)
			proxies[i] = p.Proxy
			objNames[i] = p.ObjectName
		}

		return tx.Exec(`
			WITH data AS (
				SELECT
					$1::varchar AS rolename,
					unnest(COALESCE($2::varchar[], ARRAY[]::varchar[])) AS objecttype,
					unnest(COALESCE($3::varchar[], ARRAY[]::varchar[])) AS proxyname,
					unnest(COALESCE($4::text[],    ARRAY[]::text[]))    AS objectname
			), up AS (
				INSERT INTO mcp_gateway.role_permission
				(rolename, objecttype, proxyname, objectname)
				SELECT rolename, objecttype, proxyname, objectname FROM data
				ON CONFLICT (rolename, objecttype, objectname, proxyname) DO NOTHING
				RETURNING objecttype, proxyname, objectname
			)
			DELETE FROM mcp_gateway.role_permission
			WHERE rolename = $1
			  AND (objecttype, proxyname, objectname)
			      NOT IN (SELECT objecttype, proxyname, objectname FROM up)
		`, role.Name,
			pq.Array(objTypes), pq.Array(proxies), pq.Array(objNames)).Error
	})
}

// DeleteRole deletes a role from the Postgres storage.
func (s *PostgresStorage) DeleteRole(ctx context.Context, role string) error {
	s.logger.Debug("DeleteRole", zap.String("role", role))
	tx := s.db.WithContext(ctx).Begin()
	if tx.Error != nil {
		return tx.Error
	}
	defer tx.Rollback()

	tx = tx.Exec(`DELETE FROM mcp_gateway.role WHERE name = $1`, role)
	if tx.Error != nil {
		return tx.Error
	}

	return tx.Commit().Error
}

func (s *PostgresStorage) ListRoles(ctx context.Context) ([]RoleConfig, error) {
	s.logger.Debug("ListRoles")
	const q = `
		SELECT
			r.name,
			COALESCE(json_agg(
				json_build_object(
					'objectType', rp.objecttype,
					'proxy',      rp.proxyname,
					'objectName', rp.objectname
				)
				ORDER BY rp.objecttype, rp.proxyname, rp.objectname
			) FILTER (WHERE rp.objecttype IS NOT NULL), '[]') AS perms_json
		FROM mcp_gateway.role r
		LEFT JOIN mcp_gateway.role_permission rp ON rp.rolename = r.name
		GROUP BY r.name
		ORDER BY r.name;
	`

	var rows []struct {
		Name      string
		PermsJSON []byte
	}
	if err := s.db.WithContext(ctx).Raw(q).Scan(&rows).Error; err != nil {
		return nil, err
	}

	out := make([]RoleConfig, 0, len(rows))
	for _, r := range rows {
		var perms []PermissionConfig
		_ = json.Unmarshal(r.PermsJSON, &perms)
		out = append(out, RoleConfig{
			Name:        r.Name,
			Permissions: perms,
		})
	}
	return out, nil
}

// SetAttributeToRoles sets an attribute to roles in the Postgres storage.
func (s *PostgresStorage) SetAttributeToRoles(ctx context.Context, at AttributeToRolesConfig) error {
	s.logger.Debug("SetAttributeToRoles", zap.Any("attributeToRoles", at))
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		return tx.Exec(`
			WITH data AS (
				SELECT
					$1::text  AS attributekey,
					$2::text  AS attributevalue,
					unnest(COALESCE($3::varchar[], ARRAY[]::varchar[])) AS rolename
			), up AS (
				INSERT INTO mcp_gateway.attribute_to_roles
				(attributekey, attributevalue, rolename)
				SELECT attributekey, attributevalue, rolename FROM data
				ON CONFLICT (attributekey, attributevalue, rolename) DO NOTHING
				RETURNING rolename
			)
			DELETE FROM mcp_gateway.attribute_to_roles
			WHERE attributekey  = $1
			  AND attributevalue = $2
			  AND rolename NOT IN (SELECT rolename FROM up)
		`, at.AttributeKey, at.AttributeValue, pq.Array(at.Roles)).Error
	})
}

// GetAttributeToRoles gets an attribute to roles from the Postgres storage.
func (s *PostgresStorage) GetAttributeToRoles(ctx context.Context, attributeKey, attributeValue string) (AttributeToRolesConfig, error) {
	s.logger.Debug("GetAttributeToRoles", zap.String("attributeKey", attributeKey), zap.String("attributeValue", attributeValue))
	query := `
		SELECT rolename 
		FROM mcp_gateway.attribute_to_roles 
		WHERE attributekey = $1 AND attributevalue = $2
		ORDER BY rolename ASC
	`

	rows, err := s.db.WithContext(ctx).Raw(query, attributeKey, attributeValue).Rows()
	if err != nil {
		return AttributeToRolesConfig{}, err
	}
	defer rows.Close() //nolint:errcheck // no need to check the error here

	var roles []string
	for rows.Next() {
		var roleName string
		if err := rows.Scan(&roleName); err != nil {
			return AttributeToRolesConfig{}, err
		}
		roles = append(roles, roleName)
	}

	if err := rows.Err(); err != nil {
		return AttributeToRolesConfig{}, err
	}

	if len(roles) == 0 {
		return AttributeToRolesConfig{}, gorm.ErrRecordNotFound
	}

	return AttributeToRolesConfig{
		AttributeKey:   attributeKey,
		AttributeValue: attributeValue,
		Roles:          roles,
	}, nil
}

// ListAttributeToRoles lists all attribute to roles from the Postgres storage.
func (s *PostgresStorage) ListAttributeToRoles(ctx context.Context) ([]AttributeToRolesConfig, error) {
	s.logger.Debug("ListAttributeToRoles")
	query := `
		SELECT attributekey, attributevalue, rolename 
		FROM mcp_gateway.attribute_to_roles 
		ORDER BY attributekey ASC, attributevalue ASC, rolename ASC
	`

	rows, err := s.db.WithContext(ctx).Raw(query).Rows()
	if err != nil {
		return nil, err
	}
	defer rows.Close() //nolint:errcheck // no need to check the error here

	var attributeToRoles []AttributeToRolesConfig
	var current *AttributeToRolesConfig

	for rows.Next() {
		var attributeKey, attributeValue, roleName string
		if err := rows.Scan(&attributeKey, &attributeValue, &roleName); err != nil {
			return nil, err
		}

		// New mapping or same mapping ?
		key := attributeKey + ":" + attributeValue
		currentKey := ""
		if current != nil {
			currentKey = current.AttributeKey + ":" + current.AttributeValue
		}

		if current == nil || currentKey != key {
			// Save the previous mapping
			if current != nil {
				attributeToRoles = append(attributeToRoles, *current)
			}

			// Create new mapping
			current = &AttributeToRolesConfig{
				AttributeKey:   attributeKey,
				AttributeValue: attributeValue,
				Roles:          []string{roleName},
			}
		} else {
			// Add role to the existing mapping
			current.Roles = append(current.Roles, roleName)
		}
	}

	// Add the last mapping
	if current != nil {
		attributeToRoles = append(attributeToRoles, *current)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return attributeToRoles, nil
}

// DeleteAttributeToRoles deletes an attribute to roles from the Postgres storage.
func (s *PostgresStorage) DeleteAttributeToRoles(ctx context.Context, attributeKey, attributeValue string) error {
	s.logger.Debug("DeleteAttributeToRoles", zap.String("attributeKey", attributeKey), zap.String("attributeValue", attributeValue))
	tx := s.db.WithContext(ctx).Begin()
	if tx.Error != nil {
		return tx.Error
	}
	defer tx.Rollback()

	tx = tx.Exec(`
		DELETE FROM mcp_gateway.attribute_to_roles 
		WHERE attributekey = $1 AND attributevalue = $2
	`, attributeKey, attributeValue)

	if tx.Error != nil {
		return tx.Error
	}

	return tx.Commit().Error
}

// encryptIfNeeded encrypts a value if needed.
func (s *PostgresStorage) encryptIfNeeded(value string) (string, error) {
	fmt.Println("encryptIfNeeded", value, s.encryptor.IsEncryptedString(value))
	if s.encryptor.IsEncryptedString(value) {
		return value, nil
	}

	return s.encryptor.EncryptString(value)
}

// decryptIfNeeded decrypts a value if needed.
func (s *PostgresStorage) decryptIfNeeded(value string) (string, error) {
	if s.encryptor.IsEncryptedString(value) {
		return s.encryptor.DecryptString(value)
	}

	return value, nil
}
