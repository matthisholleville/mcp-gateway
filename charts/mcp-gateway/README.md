# mcp-gateway

Simple Helm chart to deploy MCP Gateway on Kubernetes.

![Type: application](https://img.shields.io/badge/Type-application-informational?style=flat-square)

## Quick Installation

```bash
# Create namespace
kubectl create namespace mcp-gateway

# Create secrets for authentication
kubectl create secret generic okta-secret \
  --from-literal=issuer="https://your-okta-domain.okta.com/oauth2/default" \
  --from-literal=org_url="https://your-okta-domain.okta.com" \
  --from-literal=client_id="your-client-id" \
  --from-literal=private_key="your-private-key" \
  --from-literal=private_key_id="your-key-id" \
  -n mcp-gateway

# Create secret for MCP server (example: n8n)
kubectl create secret generic n8n-secret \
  --from-literal=proxy_key="your-n8n-api-key" \
  -n mcp-gateway

# Install chart
helm install mcp-gateway . --namespace mcp-gateway

# Verify deployment
kubectl get pods -n mcp-gateway
```

## Configuration

Check `values.yaml` for all available configuration options. Customize environment variables and secrets according to your setup.

## Uninstall

```bash
helm uninstall mcp-gateway -n mcp-gateway
kubectl delete namespace mcp-gateway
```

## Values

| Key | Type | Default | Description |
|-----|------|---------|-------------|
| affinity | object | `{}` |  |
| autoscaling.enabled | bool | `false` |  |
| autoscaling.maxReplicas | int | `100` |  |
| autoscaling.minReplicas | int | `1` |  |
| autoscaling.targetCPUUtilizationPercentage | int | `80` |  |
| configuration | string | `"# MCP Gateway configuration\n\n# CORS\ncors:\n  enabled: true\n  allowed_origins:\n    - \"*\"\n  allowed_methods:\n    - \"GET\"\n    - \"POST\"\n    - \"PUT\"\n    - \"DELETE\"\n  allowed_headers:\n    - \"Content-Type\"\n    - \"Authorization\"\n  allow_credentials: true\n\n# Server\nserver:\n  url: \"http://localhost:8082\"\n\n# OAuth\noauth:\n  enabled: true\n  provider: \"okta\"\n  authorization_servers:\n    - \"${OKTA_ISSUER}\"\n  bearer_methods_supported:\n    - \"header\"\n  scopes_supported:\n    - \"openid\"\n\n# Okta\nokta:\n  issuer: \"${OKTA_ISSUER}\"\n  org_url: \"${OKTA_ORG_URL}\"\n  client_id: \"${OKTA_CLIENT_ID}\"\n  private_key: \"${OKTA_PRIVATE_KEY}\"\n  private_key_id: \"${OKTA_PRIVATE_KEY_ID}\"\n\n# Auth\nauth:\n  claims: [\"groups\"]\n  mappings:\n    # claim name:group name -> scope name\n    \"groups:Base\": [\"scope:1\"]\n  permissions:\n    \"*\": [\"scope:1\"]\n\n  options:\n    scope_mode: \"any\" # OR logic. Any is the user has at least one of the scopes. All is the user has all the scopes.\n    default_scope: null # deny by default\n    enabled: true\n\n# Proxy\nproxy:\n  servers:\n    - name: \"n8n\"\n      type: \"streamable-http\"\n      connection:\n        url: \"${N8N_URL}\"\n        timeout: 30s\n      auth:\n        type: \"header\"\n        header: \"x-n8n-key\"\n        value: \"${N8N_PROXY_KEY}\"\n  proxy_config:\n    # Cache for tools and schemas to avoid repeated calls\n    cache_ttl: 300s # 5 minutes\n    heartbeat:\n      enabled: true\n      interval_seconds: 10\n"` |  |
| containerPort | int | `8082` |  |
| extraEnv[0].name | string | `"OKTA_ISSUER"` |  |
| extraEnv[0].valueFrom.secretKeyRef.key | string | `"issuer"` |  |
| extraEnv[0].valueFrom.secretKeyRef.name | string | `"okta-secret"` |  |
| extraEnv[1].name | string | `"OKTA_ORG_URL"` |  |
| extraEnv[1].valueFrom.secretKeyRef.key | string | `"org_url"` |  |
| extraEnv[1].valueFrom.secretKeyRef.name | string | `"okta-secret"` |  |
| extraEnv[2].name | string | `"OKTA_CLIENT_ID"` |  |
| extraEnv[2].valueFrom.secretKeyRef.key | string | `"client_id"` |  |
| extraEnv[2].valueFrom.secretKeyRef.name | string | `"okta-secret"` |  |
| extraEnv[3].name | string | `"OKTA_PRIVATE_KEY"` |  |
| extraEnv[3].valueFrom.secretKeyRef.key | string | `"private_key"` |  |
| extraEnv[3].valueFrom.secretKeyRef.name | string | `"okta-secret"` |  |
| extraEnv[4].name | string | `"OKTA_PRIVATE_KEY_ID"` |  |
| extraEnv[4].valueFrom.secretKeyRef.key | string | `"private_key_id"` |  |
| extraEnv[4].valueFrom.secretKeyRef.name | string | `"okta-secret"` |  |
| extraEnv[5].name | string | `"N8N_URL"` |  |
| extraEnv[5].value | string | `"https://n8n.example.com"` |  |
| extraEnv[6].name | string | `"N8N_PROXY_KEY"` |  |
| extraEnv[6].valueFrom.secretKeyRef.key | string | `"proxy_key"` |  |
| extraEnv[6].valueFrom.secretKeyRef.name | string | `"n8n-secret"` |  |
| fullnameOverride | string | `""` |  |
| image.pullPolicy | string | `"IfNotPresent"` |  |
| image.repository | string | `"ghcr.io/matthisholleville/mcp-gateway-dev"` |  |
| image.tag | string | `"dev-41d8a422698fe223d91b8813a2185e1e13854b12"` |  |
| imagePullSecrets | list | `[]` |  |
| ingress.annotations | object | `{}` |  |
| ingress.className | string | `""` |  |
| ingress.enabled | bool | `false` |  |
| ingress.hosts[0].host | string | `"chart-example.local"` |  |
| ingress.hosts[0].paths[0].path | string | `"/"` |  |
| ingress.hosts[0].paths[0].pathType | string | `"ImplementationSpecific"` |  |
| ingress.tls | list | `[]` |  |
| nameOverride | string | `""` |  |
| nodeSelector | object | `{}` |  |
| podAnnotations."prometheus.io/port" | string | `"8082"` |  |
| podAnnotations."prometheus.io/scrape" | string | `"true"` |  |
| podSecurityContext | object | `{}` |  |
| replicaCount | int | `1` |  |
| resources.limits.memory | string | `"1Gi"` |  |
| resources.requests.cpu | string | `"100m"` |  |
| resources.requests.memory | string | `"512Mi"` |  |
| securityContext | object | `{}` |  |
| service.port | int | `80` |  |
| service.type | string | `"ClusterIP"` |  |
| serviceAccount.annotations | object | `{}` |  |
| serviceAccount.create | bool | `true` |  |
| serviceAccount.name | string | `""` |  |
| tolerations | list | `[]` |  |

----------------------------------------------
Autogenerated from chart metadata using [helm-docs v1.13.1](https://github.com/norwoodj/helm-docs/releases/v1.13.1)