package auth

import (
	"context"

	firebase "firebase.google.com/go/v4"
	"firebase.google.com/go/v4/auth"
	"go.uber.org/zap"

	"github.com/matthisholleville/mcp-gateway/internal/cfg"
	"github.com/matthisholleville/mcp-gateway/pkg/logger"
)

type FirebaseProvider struct {
	BaseProvider
	cfg    *cfg.FirebaseConfig
	client *auth.Client
	logger logger.Logger
}

func (p *FirebaseProvider) Init() error {
	p.logger.Debug("Initializing Firebase provider", zap.String("project_id", p.cfg.ProjectID))
	fConfig := &firebase.Config{
		ProjectID: p.cfg.ProjectID,
	}
	ctx := context.Background()
	app, err := firebase.NewApp(ctx, fConfig)
	if err != nil {
		return err
	}

	client, err := app.Auth(ctx)
	if err != nil {
		return err
	}

	p.client = client

	return nil
}

func (p *FirebaseProvider) VerifyToken(token string) (*Jwt, error) {
	ctx := context.Background()
	fToken, err := p.client.VerifyIDToken(ctx, token)
	if err != nil {
		return nil, err
	}

	return &Jwt{
		Claims: fToken.Claims,
	}, nil
}
