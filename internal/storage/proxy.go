package storage

import (
	"context"
	"time"
)

type ProxyType string

const (
	ProxyTypeStreamableHTTP ProxyType = "streamable-http"
)

type ProxyConfig struct {
	Name       string            `json:"name"`
	Type       ProxyType         `json:"type"`
	Connection *ConnectionConfig `json:"connection"`
	Auth       *ProxyAuthConfig  `json:"auth"`
}

type ConnectionConfig struct {
	URL     string        `json:"url"`
	Timeout time.Duration `json:"timeout"`
}

type ProxyAuthConfig struct {
	Type   string `json:"type"`
	Header string `json:"header"`
	Value  string `json:"value"`
}

type ProxyInterface interface {
	GetProxy(ctx context.Context, proxy ProxyConfig) (ProxyConfig, error)
	ListProxies(ctx context.Context) ([]ProxyConfig, error)
	SetProxy(ctx context.Context, proxy ProxyConfig) error
	DeleteProxy(ctx context.Context, proxy ProxyConfig) error
}
