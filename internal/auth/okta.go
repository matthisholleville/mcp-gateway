package auth

import (
	"fmt"

	"github.com/matthisholleville/mcp-gateway/internal/cfg"
	"github.com/matthisholleville/mcp-gateway/pkg/logger"
	jwtverifier "github.com/okta/okta-jwt-verifier-golang/v2"
	"github.com/okta/okta-sdk-golang/v5/okta"
	"go.uber.org/zap"
)

// OktaProvider is a provider for Okta
type OktaProvider struct {
	BaseProvider
	cfg      *cfg.OktaConfig
	oauthCfg *cfg.OAuthConfig
	client   *okta.APIClient
	logger   logger.Logger
}

// Init initializes the Okta provider
func (p *OktaProvider) Init() error {
	oktaConfig, err := okta.NewConfiguration(
		okta.WithOrgUrl(p.cfg.OrgURL),
		okta.WithClientId(p.cfg.ClientID),
		okta.WithAuthorizationMode("PrivateKey"),
		okta.WithScopes((p.oauthCfg.ScopesSupported)),
		okta.WithPrivateKey(p.cfg.PrivateKey),
		okta.WithPrivateKeyId(p.cfg.PrivateKeyID),
	)
	if err != nil {
		return err
	}

	p.client = okta.NewAPIClient(oktaConfig)
	return nil
}

// VerifyToken verifies a JWT token
func (p *OktaProvider) VerifyToken(token string) (*Jwt, error) {
	verifierSetup := jwtverifier.JwtVerifier{
		Issuer: p.cfg.Issuer,
	}

	verifier, err := verifierSetup.New()
	if err != nil {
		p.logger.Error("Error setting up JWT verifier", zap.Error(err))
		return nil, fmt.Errorf("error setting up JWT verifier: %w", err)
	}

	jwtToken, err := verifier.VerifyAccessToken(token)
	if err != nil {
		p.logger.Error("Error verifying JWT", zap.Error(err))
		return nil, fmt.Errorf("error verifying JWT: %w", err)
	}

	return &Jwt{Claims: jwtToken.Claims}, nil
}
