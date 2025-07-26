# MCP Gateway

![Go](https://img.shields.io/badge/go-%2300ADD8.svg?style=flat&logo=go&logoColor=white)
![Docker](https://img.shields.io/badge/docker-%230db7ed.svg?style=flat&logo=docker&logoColor=white)
![License](https://img.shields.io/badge/license-Apache%202.0-blue.svg)

A **flexible and extensible proxy gateway** for [MCP (Model Context Protocol)](https://modelcontextprotocol.io) servers, providing enterprise-grade middleware capabilities including authentication, authorization, rate limiting (coming soon), and observability.

## 🚀 Why MCP Gateway?

MCP Gateway acts as a centralized proxy that sits between your AI applications and MCP servers, providing:

- **🔐 Security First**: OAuth2/JWT authentication with fine-grained permissions
- **📊 Enterprise Observability**: Built-in Prometheus metrics and structured logging
- **🔄 High Availability**: Automatic reconnection, heartbeat monitoring, and resilient proxy
- **⚙️ Flexible Configuration**: Support for multiple MCP servers with individual authentication
- **📈 Production Ready**: Docker support, graceful shutdown, and comprehensive error handling

## 🏗️ Architecture

```
┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐
│   AI Client     │───▶│  MCP Gateway    │───▶│   MCP Server    │
│ (Claude, etc.)  │    │                 │    │  (n8n, etc.)    │
└─────────────────┘    │  ┌───────────┐  │    └─────────────────┘
                       │  │   Auth    │  │
                       │  │   CORS    │  │    ┌─────────────────┐
                       │  │ Metrics   │  │───▶│ Another Server  │
                       │  │  Cache    │  │    └─────────────────┘
                       │  └───────────┘  │
                       └─────────────────┘
```

## ✨ Key Features

### 🔒 Authentication & Authorization
- **OAuth2/JWT Integration**: Seamless integration with identity providers
- **Fine-grained Permissions**: Tool-level access control based on user scopes and groups
- **Flexible Claim Mapping**: Map JWT claims to internal permission scopes

**Supported Auth Providers:**
- ✅ **Okta** - Full OAuth2/JWT support with claim mapping
- 🔄 More providers coming soon (contributions welcome!)

## 🔧 Usage Examples

MCP Gateway serves as a **unified entry point** for multiple MCP servers, allowing you to:
- **Centralize access** to all your MCP tools through a single endpoint
- **Unify authentication** across different backend servers 
- **Namespace tools** to avoid conflicts between servers (`server:tool_name`)
- **Apply consistent security policies** regardless of the backend implementation

### 🛠️ MCP Proxy Capabilities
- **Multi-Server Support**: Proxy requests to multiple MCP servers simultaneously
- **Tool Namespacing**: Automatic prefixing to avoid naming conflicts (`server:tool_name`)
- **Connection Management**: Automatic reconnection and connection pooling
- **Heartbeat Monitoring**: Configurable health checks for backend servers

### 📊 Observability & Monitoring
- **Prometheus Metrics**: Built-in metrics for tools called, errors, and performance
- **Structured Logging**: JSON and text logging with configurable levels
- **Health Endpoints**: `/live` and `/ready` endpoints for container orchestration
- **Request Tracing**: Correlation IDs for request tracking

### ⚡ Performance & Reliability
- **Periodic Tool Discovery**: Regular re-interrogation of tools exposed by proxied servers
- **Error Handling**: Retry logic with exponential backoff
- **Graceful Shutdown**: Clean termination of connections and requests

## 🚀 Quick Start

### Using Docker (Recommended)

```bash
# Use dev image if you want early feature
# docker pull ghcr.io/matthisholleville/mcp-gateway-dev:latest

# Pull the latest image
docker pull ghcr.io/matthisholleville/mcp-gateway:latest

# Run with environment variables
docker run -p 8082:8082 \
  -e OKTA_ISSUER="https://your-okta-domain.okta.com/oauth2/default" \
  -e OKTA_ORG_URL="https://your-okta-domain.okta.com" \
  -e OKTA_CLIENT_ID="your-client-id" \
  -e OKTA_PRIVATE_KEY="your-private-key" \
  -e OKTA_PRIVATE_KEY_ID="your-key-id" \
  -e N8N_URL="http://your-n8n-instance:5678" \
  -e N8N_PROXY_KEY="your-n8n-api-key" \
  ghcr.io/matthisholleville/mcp-gateway:latest serve
```

### Using Go

```bash
# Install
go install github.com/matthisholleville/mcp-gateway@latest

# Run
export OKTA_ISSUER="https://your-okta-domain.okta.com/oauth2/default"
export N8N_URL="http://localhost:5678"
# ... other environment variables
mcp-gateway serve
```

### Using Make (Development)

```bash
# Clone the repository
git clone https://github.com/matthisholleville/mcp-gateway.git
cd mcp-gateway

# Install dependencies
make deps

# Run in development mode
make dev
```

### Using Helm (Kubernetes)

For Kubernetes deployment with Helm, see: **[charts/mcp-gateway/README.md](charts/mcp-gateway/README.md)**

## ⚙️ Configuration

MCP Gateway uses a YAML configuration file with environment variable substitution:

```yaml
# Example: Server configuration
server:
  url: "http://localhost:8082"

# OAuth configuration
oauth:
  enabled: true
  provider: "okta"
  authorization_servers:
    - "${OKTA_ISSUER}"

# Authentication & authorization
auth:
  claims: ["groups"]
  mappings:
    "groups:Admin": ["scope:admin"]
    "groups:User": ["scope:user"]
  permissions:
    "n8n:*": ["scope:user"]  # All n8n tools require user scope
    "admin:*": ["scope:admin"]  # Admin tools require admin scope

# Proxy servers
proxy:
  servers:
    - name: "n8n"
      type: "streamable-http"
      connection:
        url: "${N8N_URL}"
        timeout: 30s
      auth:
        type: "header"
        header: "x-n8n-key"
        value: "${N8N_PROXY_KEY}"
```

### Environment Variables

The configuration above uses environment variables for sensitive values. The exact variables depend on your OAuth provider and MCP servers configuration.

**💡 Tip**: Use [mcp-inspector](https://github.com/modelcontextprotocol/inspector) to discover and test your MCP servers before configuring the gateway:

```bash
# Install mcp-inspector
npm install -g @modelcontextprotocol/inspector

# Inspect your MCP server
mcp-inspector http://your-server:port

# Use the discovered configuration in your gateway setup
```

## OAuth2 Integration

MCP Gateway provides OAuth2 resource server capabilities:

```bash
# Discover OAuth2 metadata
curl http://localhost:8082/.well-known/oauth-protected-resource
```

## Monitoring & Metrics

Built-in Prometheus metrics include:

- `mcp_gateway_tools_called` - Number of tool calls by tool and proxy
- `mcp_gateway_tools_call_errors` - Number of failed tool calls
- `mcp_gateway_tools_call_success` - Number of successful tool calls
- `mcp_gateway_list_tools` - Number of list tools requests

## Permission System

Fine-grained access control based on JWT claims:

```yaml
auth:
  claims: ["groups", "roles"]
  mappings:
    "groups:Developers": ["scope:dev"]
    "roles:Admin": ["scope:admin"]
  permissions:
    "n8n:*": ["scope:dev"]           # Developers can use n8n tools
    "admin:user_management": ["scope:admin"]  # Only admins can manage users
  options:
    scope_mode: "any"  # User needs at least one matching scope
    default_scope: null  # Deny by default
```

## 📊 API Reference

### Endpoints

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/mcp` | POST | MCP protocol endpoint |
| `/live` | GET | Liveness probe |
| `/ready` | GET | Readiness probe |
| `/metrics` | GET | Prometheus metrics |
| `/.well-known/oauth-protected-resource` | GET | OAuth2 metadata |

## 🛠️ Development

### Prerequisites

- Go 1.24.3 or later
- Docker (optional)
- Make (optional)

### Building

```bash
# Build binary
make build

# Run tests
make test

# Run linting
make lint

# Generate test coverage
make test-cover

# Build Docker image
make docker-build
```

## 🤝 Contributing

We welcome contributions! Please see our [Contributing Guidelines](CONTRIBUTING.md) for details.

## 📝 License

This project is licensed under the Apache License 2.0 - see the [LICENSE](LICENSE) file for details.

## 🆘 Support

- **Issues**: [GitHub Issues](https://github.com/matthisholleville/mcp-gateway/issues)

## 🙏 Acknowledgments

- [Model Context Protocol](https://modelcontextprotocol.io) - The protocol this gateway implements
- [mark3labs/mcp-go](https://github.com/mark3labs/mcp-go) - Go implementation of MCP
- [Echo Framework](https://echo.labstack.com/) - Web framework

---

**Made with ❤️ by [Matthis Holleville](https://github.com/matthisholleville)**
