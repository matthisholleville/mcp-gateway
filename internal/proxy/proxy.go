// Package proxy provides a proxy for the MCP server.
package proxy

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/mark3labs/mcp-go/client"
	"github.com/mark3labs/mcp-go/client/transport"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/matthisholleville/mcp-gateway/internal/storage"
	"github.com/matthisholleville/mcp-gateway/pkg/logger"
	"go.uber.org/zap"
)

var (
	defaultTimeout      = 30 * time.Hour
	initialBackoff      = 500 * time.Millisecond
	maxBackoff          = 5 * time.Second
	maxRetriesOnConnect = 5
)

type proxy struct {
	name   string
	cfg    *storage.ProxyConfig
	logger logger.Logger
	client *client.Client
	mu     sync.Mutex
}

type proxyInterface interface {
	GetTools() ([]mcp.Tool, error)
	CallTool(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error)
	GetName() string
}

var _ proxyInterface = &proxy{}

func NewProxy(proxyCfg *[]storage.ProxyConfig, logger logger.Logger) (*[]proxyInterface, error) {
	proxies := &[]proxyInterface{}

	for _, srv := range *proxyCfg {
		cfgCopy := srv
		p := &proxy{
			name:   cfgCopy.Name,
			cfg:    &cfgCopy,
			logger: logger.With(zap.String("mcp_proxy", cfgCopy.Name)),
		}

		if err := p.ensureConnected(context.Background()); err != nil {
			logger.Error("unable to connect to MCP server", zap.String("proxy", cfgCopy.Name), zap.Error(err))
			continue
		}

		*proxies = append(*proxies, p)
	}

	return proxies, nil
}

func (p *proxy) dial(ctx context.Context) error {
	tr, err := openStreamableHTTPProxy(p.cfg, p.logger)
	if err != nil {
		return err
	}

	cli := client.NewClient(tr) // transport wrapper

	// handshake MCP/initialize
	_, err = cli.Initialize(ctx, mcp.InitializeRequest{
		Params: mcp.InitializeParams{
			ProtocolVersion: mcp.LATEST_PROTOCOL_VERSION,
			ClientInfo: mcp.Implementation{
				Name:    "MCP Gateway Proxy",
				Version: "1.1.0",
			},
		},
	})
	if err != nil {
		_ = tr.Close()
		return err
	}

	p.client = cli
	p.logger.Info("connected")
	return nil
}

func (p *proxy) ensureConnected(ctx context.Context) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.client != nil {
		return nil
	}

	b := initialBackoff
	for i := 0; i < maxRetriesOnConnect; i++ {
		err := p.dial(ctx)
		if err == nil {
			return nil
		}
		p.logger.Warn("dial failed, retrying...",
			zap.Int("attempt", i+1),
			zap.Error(err))
		time.Sleep(b)
		b *= 2
		if b > maxBackoff {
			b = maxBackoff
		}
	}
	return fmt.Errorf("unable to connect after %d attempts", maxRetriesOnConnect)
}

func (p *proxy) CallTool(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	req.Params.Name = strings.TrimPrefix(req.Params.Name, p.name+":")

	if err := p.ensureConnected(ctx); err != nil {
		return nil, err
	}

	res, err := p.client.CallTool(ctx, req)
	if err == nil || !isTransient(err) {
		return res, err
	}

	p.logger.Warn("transient error, forcing reconnect", zap.Error(err))
	p.resetClient()

	if err := p.ensureConnected(ctx); err != nil {
		return nil, err
	}
	return p.client.CallTool(ctx, req)
}

func isTransient(err error) bool {
	if err == nil {
		return false
	}
	msg := err.Error()
	return strings.Contains(msg, "context canceled") ||
		strings.Contains(msg, "transport error") ||
		strings.Contains(msg, "session terminated") ||
		strings.Contains(msg, "connection reset")
}

func (p *proxy) resetClient() {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.client != nil {
		_ = p.client.Close()
		p.client = nil
	}
}

func (p *proxy) GetTools() ([]mcp.Tool, error) {
	ctx := context.Background()

	if err := p.ensureConnected(ctx); err != nil {
		return nil, err
	}

	toolsResult, err := p.client.ListTools(ctx, mcp.ListToolsRequest{})
	if err != nil {
		return nil, err
	}
	return toolsResult.Tools, nil
}

func (p *proxy) startHeartbeat(interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for range ticker.C {
		p.logger.Debug("heartbeat...", zap.String("interval", interval.String()), zap.String("proxy", p.name))
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		if err := p.ensureConnected(ctx); err != nil {
			p.logger.Warn("heartbeat failed", zap.Error(err))
		}
		cancel()
	}
}

func (p *proxy) GetName() string {
	return p.name
}

func openStreamableHTTPProxy(proxyConfig *storage.ProxyConfig, logger logger.Logger) (*transport.StreamableHTTP, error) {
	ctx := context.Background()
	endpoint := proxyConfig.Connection.URL

	headers := map[string]string{}
	if proxyConfig.Auth.Type == "header" {
		headers[proxyConfig.Auth.Header] = proxyConfig.Auth.Value
	}

	timeout := defaultTimeout
	// if proxyConfig.Connection.Timeout != 0 {
	// 	if t, err := time.ParseDuration(proxyConfig.Connection.Timeout.String()); err == nil {
	// 		timeout = t
	// 	} else {
	// 		logger.Error("Failed to parse timeout", zap.Error(err))
	// 	}
	// }

	httpTransport, err := transport.NewStreamableHTTP(
		endpoint,
		transport.WithHTTPTimeout(timeout),
		transport.WithHTTPHeaders(headers),
	)
	if err != nil {
		return nil, err
	}

	if err := httpTransport.Start(ctx); err != nil {
		return nil, err
	}

	return httpTransport, nil
}
