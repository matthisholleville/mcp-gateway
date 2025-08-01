# MCP Gateway

![Go](https://img.shields.io/badge/go-%2300ADD8.svg?style=flat&logo=go&logoColor=white)
![Docker](https://img.shields.io/badge/docker-%230db7ed.svg?style=flat&logo=docker&logoColor=white)
![License](https://img.shields.io/badge/license-Apache%202.0-blue.svg)

A **flexible and extensible proxy gateway** for [MCP (Model Context Protocol)](https://modelcontextprotocol.io) servers, providing enterprise-grade middleware capabilities including authentication, authorization, rate limiting, and observability.

## 🚀 Features

### 🔐 Authentication & Authorization
- **Multiple Auth Providers**: Okta OAuth2/JWT
- **Role-Based Permissions**: Fine-grained tool access control
- **attribute-to-Role Mapping**: Flexible user permission assignment
- **JWT Token Verification**: Secure token validation

### 📊 Enterprise Ready
- **Multiple Storage Backends**: Memory (dev), PostgreSQL (coming soon)
- **RESTful Admin API**: Dynamic configuration management
- **Prometheus Metrics**: Built-in observability
- **Structured Logging**: JSON and text output formats
- **Health Endpoints**: Container orchestration support

### ⚙️ Flexible Configuration
- **YAML Configuration**: Environment variable substitution
- **CLI Flags**: Override any configuration option
- **Hot Configuration**: Runtime proxy/role management via API

## 🏗️ Architecture

```
┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐
│   AI Client     │───▶│  MCP Gateway    │───▶│   MCP Server    │
│ (Claude, etc.)  │    │                 │    │  (n8n, etc.)    │
└─────────────────┘    │  ┌───────────┐  │    └─────────────────┘
                       │  │           │  │
                       │  │Okta Auth  │  │    ┌─────────────────┐
                       │  │  Roles    │  │───▶│ Another Server  │
                       │  │ Metrics   │  │    └─────────────────┘
                       │  └───────────┘  │
                       └─────────────────┘
```

## 🚀 Quick Start

### Using Go (Development)

```bash
# Clone and run without authentication
git clone https://github.com/matthisholleville/mcp-gateway.git
cd mcp-gateway

# Run without authentication
go run main.go serve \
  --log-format=text \
  --log-level=debug 
```

### Using Docker

```bash
# Pull the latest image
docker pull ghcr.io/matthisholleville/mcp-gateway:latest

# Run with environment variables
docker run -p 8082:8082 \
  ghcr.io/matthisholleville/mcp-gateway:latest serve
```

### Using Helm

```bash
helm repo add mcp-gateway https://matthisholleville.github.io/mcp-gateway
helm install mcp-gateway mcp-gateway/mcp-gateway
```

## ⚙️ Configuration

### Configuration Sources (Priority Order)
1. **CLI Flags** (highest priority)
2. **Environment Variables** (`MCP_GATEWAY_*`)
3. **YAML Configuration File** (`config/config.yaml`)
4. **Default Values** (lowest priority)

### YAML Configuration Example

```yaml
# config/config.yaml
server:
  url: "http://localhost:8082"

# Authentication
authProvider:
  enabled: true
  name: "okta"
  okta:
    issuer: "https://custom-xxx.okta.com/oauth2/default"
    orgUrl: "https://custom-xxx.okta.com"
    clientId: "xxx"
    privateKey: "-----BEGIN PRIVATE KEY-----xxx-----END PRIVATE KEY-----"
    privateKeyId: "xxx"

oauth:
  enabled: true
  provider: "okta"
  authorizationServers:
    - "https://custom-xxx.okta.com/oauth2/default"
  bearerMethodsSupported: ["Bearer"]
  scopesSupported: ["openid", "email", "profile"]

# Storage backend
backendConfig:
  engine: "memory"  # "postgres" coming soon
  # uri: "postgres://user:pass@localhost/mcp_gateway"

# Proxy configuration
proxy:
  cacheTTL: 300s
  heartbeat:
    enabled: true
    intervalSeconds: 10s
```

### Environment Variables

All configuration options can be set via environment variables with `MCP_GATEWAY_` prefix:

```bash
export MCP_GATEWAY_AUTH_PROVIDER_ENABLED=true
export MCP_GATEWAY_OAUTH_ENABLED=true
```

## 🔐 Authentication Providers

### Okta OAuth2

```bash
go run main.go serve \
  --auth-provider-name=okta \
  --okta-issuer=https://your-domain.okta.com/oauth2/default \
  --okta-org-url=https://your-domain.okta.com \
  --okta-client-id=your-client-id
  --okta-private-key="-----BEGIN RSA PRIVATE KEY-----\n..."
  --okta-private-key-id="akXpH7Ha5VKCe2kNT3eCPn_YRaJ0..."
```

## 📦 Storage Backends

### Memory Backend (Development)
- **Usage**: Development and testing
- **Persistence**: None (data lost on restart)
- **Configuration**: `--backend-engine=memory`

### PostgreSQL Backend (Coming Soon)
- **Usage**: Production environments
- **Persistence**: Full durability
- **Configuration**: `--backend-engine=postgres --backend-uri=postgres://...`

## 🛠️ Admin API

**You can update the admin API Key with `--http-admin-api-key` flag**

The gateway provides RESTful APIs for runtime configuration management:

### Proxy Management

**Swagger is available at http://localhost:8082/swagger/index.html**

```bash
# List all proxies
curl -H "X-API-Key: your-api-key" http://localhost:8082/v1/admin/proxies

# Add/Update proxy
curl -X PUT -H "X-API-Key: your-api-key" \
  -H "Content-Type: application/json" \
  -d '{"name":"n8n","type":"streamable-http","connection":{"url":"http://n8n:5678"}}' \
  http://localhost:8082/v1/admin/proxies/n8n
```

### Role Management

- `objectType` can be `*` or `tools`
- `objectName` is the tool name if `objectType` is `tools`. Can be `*` or your object name
- `proxy` is the proxy name. Can be `*` or your proxy name

```bash
# Create role
curl -X PUT -H "X-API-Key: your-api-key" \
  -H "Content-Type: application/json" \
  -d '{"name":"admin","permissions":[{"objectType":"*","proxy":"*","objectName":"*"}]}' \
  http://localhost:8082/v1/admin/roles
```

### Attribute-to-Role Mapping

- `attributeKey` is the key in your JWT `attributes`
- `attributeValue` is the attribute value
- `roles` is the list of roles. You must create the roles before creating the attribute-to-role mapping

```bash
# Map user attributes to roles
curl -X PUT -H "X-API-Key: your-api-key" \
  -H "Content-Type: application/json" \
  -d '{"attributeKey":"groups","attributeValue":"admins","roles":["admin"]}' \
  http://localhost:8082/v1/admin/attribute-to-roles
```

## 📊 API Endpoints

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/mcp` | POST | MCP protocol endpoint |
| `/live` | GET | Liveness probe |
| `/ready` | GET | Readiness probe |
| `/metrics` | GET | Prometheus metrics |
| `/swagger/*` | GET | API Documentation |
| `/v1/admin/proxies` | GET, PUT, DELETE | Proxy management |
| `/v1/admin/roles` | GET, PUT, DELETE | Role management |
| `/v1/admin/attribute-to-roles` | GET, PUT, DELETE | attribute mapping |

## 🛠️ Development

### Prerequisites
- Go 1.24.3+
- Docker (optional)
- Make

### Commands
```bash
# Install dependencies
make deps

# Run in development
make dev

# Build binary
make build

# Run tests
make test

# Generate coverage
make test-cover

# Build Docker image
make docker-build
```

### Configuration Paths
The gateway searches for `config.yaml` in:
- `/etc/mcp-gateway/`
- `$HOME/.mcp-gateway/`
- `./config/`

## 📝 CLI Reference

### Common Flags
```bash
--log-format              # text, json
--log-level               # debug, info, warn, error
--log-timestamp-format    # Format for logging timestamps
--auth-provider-enabled   # Enable authentication
--auth-provider-name      # okta
--oauth-enabled           # Enable OAuth2
--backend-engine          # memory, postgres (coming soon)
--http-addr               # Server address (default: :8082)
--http-admin-api-key      # Admin API key for MCP Gateway configuration
```

### Proxy Flags
```bash
--proxy-cache-ttl         # TTL for the proxy cache
--proxy-heartbeat-interval # Interval for the proxy heartbeat
```

### Backend Flags
```bash
--backend-uri                    # URI for the auth backend
--backend-max-open-conns         # Maximum number of open database connections
--backend-max-idle-conns         # Maximum number of idle connections in pool
--backend-conn-max-idle-time     # Maximum time a connection may be idle
--backend-conn-max-lifetime      # Maximum time a connection may be reused
```

### OAuth Flags
```bash
--oauth-authorization-servers           # OAuth authorization servers
--oauth-resource                        # OAuth resource (e.g. http://localhost:8082)
--oauth-bearer-methods-supported        # Bearer methods supported for OAuth
--oauth-scopes-supported                # OAuth scopes supported (e.g. openid,email,profile)
```

### Okta Flags
```bash
--okta-issuer           # Okta authorization server
--okta-org-url          # Okta organization URL
--okta-client-id        # Okta client ID
--okta-private-key      # Private key for client auth
--okta-private-key-id   # Private key ID
```

## 🤝 Contributing

We welcome contributions! Please see [CONTRIBUTING.md](CONTRIBUTING.md) for guidelines.

## 📄 License

Licensed under the Apache License 2.0 - see [LICENSE](LICENSE) for details.

---

**Made with ❤️ by [Matthis Holleville](https://github.com/matthisholleville)**
