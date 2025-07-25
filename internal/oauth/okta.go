package oauth

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/matthisholleville/mcp-gateway/internal/cfg"
	jwtverifier "github.com/okta/okta-jwt-verifier-golang/v2"
	"github.com/okta/okta-sdk-golang/v5/okta"
)

// OktaProvider is a provider for Okta
type OktaProvider struct {
	BaseProvider
	cfg      *cfg.Okta
	oauthCfg *cfg.OAuth
	authCfg  *cfg.Auth
	client   *okta.APIClient
	logger   *slog.Logger
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
	ctx := context.Background()
	verifierSetup := jwtverifier.JwtVerifier{
		Issuer: p.cfg.Issuer,
	}

	verifier, err := verifierSetup.New()
	if err != nil {
		p.logger.ErrorContext(ctx, "Error setting up JWT verifier", slog.Any("err", err))
		return nil, fmt.Errorf("error setting up JWT verifier: %w", err)
	}

	jwtToken, err := verifier.VerifyAccessToken(token)
	if err != nil {
		p.logger.ErrorContext(ctx, "Error verifying JWT", slog.Any("err", err))
		return nil, fmt.Errorf("error verifying JWT: %w", err)
	}

	claims, err := p.verifyClaims(&Jwt{Claims: jwtToken.Claims})
	if err != nil {
		p.logger.ErrorContext(ctx, "Error verifying claims", slog.Any("err", err))
		return nil, fmt.Errorf("error verifying claims: %w", err)
	}

	return &Jwt{Claims: claims}, nil
}

// verifyClaims verifies the claims of a JWT token
func (p *OktaProvider) verifyClaims(jwtToken *Jwt) (map[string]interface{}, error) {
	claims := make(map[string]interface{})
	for _, claim := range p.authCfg.Claims {
		if _, ok := jwtToken.Claims[claim]; ok {
			claims[claim] = jwtToken.Claims[claim]
		}
	}
	if len(claims) != len(p.authCfg.Claims) {
		return nil, fmt.Errorf("missing claims in JWT: %v", claims)
	}
	return claims, nil
}
