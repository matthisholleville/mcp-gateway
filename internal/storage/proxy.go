// Package storage provides a storage interface for the MCP Gateway.
package storage

import (
	"context"
	"time"
)

type ProxyType string
type ProxyAuthType string

const (
	ProxyTypeStreamableHTTP ProxyType     = "streamable-http"
	ProxyAuthTypeHeader     ProxyAuthType = "header"
	ProxyAuthTypeOAuth      ProxyAuthType = "oauth"
)

func (p ProxyType) IsValid() bool {
	return p == ProxyTypeStreamableHTTP
}

func (p ProxyAuthType) IsValid() bool {
	return p == ProxyAuthTypeHeader || p == ProxyAuthTypeOAuth
}

type ProxyConfig struct {
	Name     string        `json:"name"`
	Type     ProxyType     `json:"type"`
	URL      string        `json:"url"`
	Timeout  time.Duration `json:"timeout"`
	AuthType ProxyAuthType `json:"authType"`
	Headers  []ProxyHeader `json:"headers"`
	OAuth    *ProxyOAuth   `json:"oauth"`
}

type ProxyHeader struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

type ProxyOAuth struct {
	ClientID      string `json:"clientId"`
	ClientSecret  string `json:"clientSecret"`
	TokenEndpoint string `json:"tokenEndpoint"`
	Scopes        string `json:"scopes"`
}

type ProxyInterface interface {
	GetProxy(ctx context.Context, proxy string, decrypt bool) (ProxyConfig, error)
	ListProxies(ctx context.Context, decrypt bool) ([]ProxyConfig, error)
	SetProxy(ctx context.Context, proxy *ProxyConfig, encrypt bool) error
	DeleteProxy(ctx context.Context, proxy string) error
}
