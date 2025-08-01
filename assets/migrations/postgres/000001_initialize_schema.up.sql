-- Create the mcp_gateway schema if it doesn't exist
CREATE SCHEMA IF NOT EXISTS mcp_gateway;

-- Set the search path to use the agicapd schema
SET search_path TO mcp_gateway, public;

-- Create the proxy table
CREATE TABLE proxy (
    Name TEXT PRIMARY KEY,
    Type VARCHAR(255) NOT NULL,
    URL TEXT NOT NULL,
    AuthType VARCHAR(255) NOT NULL,
    Timeout INT NOT NULL
);

-- Create the proxy_header table
CREATE TABLE proxy_header (
    ProxyName TEXT NOT NULL,
    HeaderKey TEXT NOT NULL,
    HeaderValue TEXT NOT NULL,
    PRIMARY KEY (ProxyName, HeaderKey),
    FOREIGN KEY (ProxyName) REFERENCES proxy(Name) ON DELETE CASCADE
);

-- Create the proxy_oauth table
CREATE TABLE proxy_oauth (
    ProxyName TEXT,
    ClientId TEXT NOT NULL,
    ClientSecret TEXT NOT NULL,
    TokenEndpoint TEXT NOT NULL,
    Scopes TEXT,
    PRIMARY KEY (ProxyName, ClientId, ClientSecret),
    FOREIGN KEY (ProxyName) REFERENCES proxy(Name) ON DELETE CASCADE
);

-- Create the role table
CREATE TABLE role (
    Name VARCHAR(255) PRIMARY KEY
);

-- Create the role_permission table
CREATE TABLE role_permission (
    RoleName VARCHAR(255) NOT NULL,
    ObjectType VARCHAR(255) NOT NULL,
    ObjectName TEXT NOT NULL,
    ProxyName TEXT NOT NULL,
    PRIMARY KEY (RoleName, ObjectType, ObjectName, ProxyName),
    FOREIGN KEY (RoleName) REFERENCES role(Name) ON DELETE CASCADE
);

-- Create the attribute_to_roles table
CREATE TABLE attribute_to_roles (
    AttributeKey TEXT NOT NULL,
    AttributeValue TEXT NOT NULL,
    RoleName VARCHAR(255) NOT NULL,
    PRIMARY KEY (AttributeKey, AttributeValue, RoleName),
    FOREIGN KEY (RoleName) REFERENCES role(Name)
);

-- accelerate GetAttributeToRoles
CREATE INDEX IF NOT EXISTS idx_attr_roles_key_value
    ON mcp_gateway.attribute_to_roles (attributekey, attributevalue);

-- protect frequent role_permission ↔ proxy joins
CREATE INDEX IF NOT EXISTS idx_role_permission_proxyname
    ON mcp_gateway.role_permission (proxyname);

-- allow fast search by header
CREATE INDEX IF NOT EXISTS idx_proxy_header_key
    ON mcp_gateway.proxy_header (proxyname, headerkey);

-- allow fast search by role name
CREATE INDEX IF NOT EXISTS idx_role_permission_rolename
    ON mcp_gateway.role_permission (rolename);

-- allow fast search by object type and proxy name
CREATE INDEX IF NOT EXISTS idx_role_permission_object
    ON mcp_gateway.role_permission (objecttype, proxyname);