package storage

import "context"

type ClaimToRolesConfig struct {
	ClaimKey   string   `json:"claim_key"`
	ClaimValue string   `json:"claim_value"`
	Roles      []string `json:"roles"`
}

type ClaimToRolesInterface interface {
	ListClaimToRoles(ctx context.Context) ([]ClaimToRolesConfig, error)
	SetClaimToRoles(ctx context.Context, claimToRoles ClaimToRolesConfig) error
	GetClaimToRoles(ctx context.Context, claimKey, claimValue string) (ClaimToRolesConfig, error)
	DeleteClaimToRoles(ctx context.Context, claimKey, claimValue string) error
}
