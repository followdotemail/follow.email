package oauth

import (
	"context"
	"time"

	"golang.org/x/oauth2"
)

// Minimal OAuth types for compilation compatibility during Clerk migration
// This file contains only type definitions to prevent compilation errors
// The actual OAuth functionality has been replaced by Clerk

type Provider string

const (
	ProviderGoogle    Provider = "gmail"
	ProviderMicrosoft Provider = "outlook"
)

type OAuthService struct {
	// Stub implementation - functionality moved to Clerk
}

type TokenInfo struct {
	AccessToken  string    `json:"access_token"`
	RefreshToken string    `json:"refresh_token"`
	TokenType    string    `json:"token_type"`
	ExpiresAt    time.Time `json:"expires_at"`
	Scope        string    `json:"scope"`
}

type UserInfo struct {
	ID       string `json:"id"`
	Email    string `json:"email"`
	Name     string `json:"name"`
	Provider string `json:"provider"`
}

// Stub functions to maintain compilation compatibility
// These should not be used - Clerk handles authentication now

func NewOAuthService(cfg interface{}) *OAuthService {
	return &OAuthService{}
}

func (s *OAuthService) GetAuthURL(provider Provider, state string) (string, error) {
	return "", nil
}

func (s *OAuthService) ExchangeCode(ctx context.Context, provider Provider, code string) (*TokenInfo, error) {
	return nil, nil
}

func (s *OAuthService) RefreshToken(ctx context.Context, provider Provider, refreshToken string) (*TokenInfo, error) {
	return nil, nil
}

func (s *OAuthService) GetUserInfo(ctx context.Context, provider Provider, accessToken string) (*UserInfo, error) {
	return nil, nil
}

func (s *OAuthService) ValidateToken(tokenInfo *TokenInfo) bool {
	return false
}

func (s *OAuthService) CreateTokenSource(ctx context.Context, provider Provider, tokenInfo *TokenInfo) oauth2.TokenSource {
	return nil
}