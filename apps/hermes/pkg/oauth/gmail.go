package oauth

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/gmail/v1"
	"google.golang.org/api/option"
)

// GmailOAuthService handles Gmail-specific OAuth operations
type GmailOAuthService struct {
	clientID     string
	clientSecret string
	redirectURL  string
	config       *oauth2.Config
}

// GmailTokenInfo represents Gmail OAuth token information
type GmailTokenInfo struct {
	AccessToken  string    `json:"access_token"`
	RefreshToken string    `json:"refresh_token"`
	TokenType    string    `json:"token_type"`
	ExpiresAt    time.Time `json:"expires_at"`
	Scope        string    `json:"scope"`
}

// GmailUserInfo represents Gmail user information
type GmailUserInfo struct {
	ID           string `json:"id"`
	Email        string `json:"email"`
	Name         string `json:"name"`
	Picture      string `json:"picture"`
	VerifiedEmail bool   `json:"verified_email"`
}

// NewGmailOAuthService creates a new Gmail OAuth service
func NewGmailOAuthService(clientID, clientSecret, redirectURL string) *GmailOAuthService {
	config := &oauth2.Config{
		ClientID:     clientID,
		ClientSecret: clientSecret,
		RedirectURL:  redirectURL,
		Scopes: []string{
			gmail.GmailReadonlyScope,
			gmail.GmailModifyScope,
			"https://www.googleapis.com/auth/userinfo.email",
			"https://www.googleapis.com/auth/userinfo.profile",
		},
		Endpoint: google.Endpoint,
	}

	return &GmailOAuthService{
		clientID:     clientID,
		clientSecret: clientSecret,
		redirectURL:  redirectURL,
		config:       config,
	}
}

// GetAuthURL generates the Gmail OAuth authorization URL
func (s *GmailOAuthService) GetAuthURL(state string) string {
	return s.config.AuthCodeURL(state, oauth2.AccessTypeOffline, oauth2.SetAuthURLParam("prompt", "consent"))
}

// ExchangeCode exchanges authorization code for access token
func (s *GmailOAuthService) ExchangeCode(ctx context.Context, code string) (*GmailTokenInfo, error) {
	token, err := s.config.Exchange(ctx, code)
	if err != nil {
		return nil, fmt.Errorf("failed to exchange code for token: %w", err)
	}

	return &GmailTokenInfo{
		AccessToken:  token.AccessToken,
		RefreshToken: token.RefreshToken,
		TokenType:    token.TokenType,
		ExpiresAt:    token.Expiry,
		Scope:        "", // Gmail scopes are predefined
	}, nil
}

// RefreshToken refreshes an expired access token
func (s *GmailOAuthService) RefreshToken(ctx context.Context, refreshToken string) (*GmailTokenInfo, error) {
	token := &oauth2.Token{
		RefreshToken: refreshToken,
	}

	tokenSource := s.config.TokenSource(ctx, token)
	newToken, err := tokenSource.Token()
	if err != nil {
		return nil, fmt.Errorf("failed to refresh token: %w", err)
	}

	return &GmailTokenInfo{
		AccessToken:  newToken.AccessToken,
		RefreshToken: newToken.RefreshToken,
		TokenType:    newToken.TokenType,
		ExpiresAt:    newToken.Expiry,
	}, nil
}

// GetUserInfo retrieves user information from Google
func (s *GmailOAuthService) GetUserInfo(ctx context.Context, accessToken string) (*GmailUserInfo, error) {
	client := &http.Client{}
	req, err := http.NewRequestWithContext(ctx, "GET", "https://www.googleapis.com/oauth2/v2/userinfo", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+accessToken)

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to get user info: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to get user info: status %d", resp.StatusCode)
	}

	var userInfo GmailUserInfo
	if err := json.NewDecoder(resp.Body).Decode(&userInfo); err != nil {
		return nil, fmt.Errorf("failed to decode user info: %w", err)
	}

	return &userInfo, nil
}

// ValidateToken checks if the token is valid and not expired
func (s *GmailOAuthService) ValidateToken(tokenInfo *GmailTokenInfo) bool {
	if tokenInfo == nil || tokenInfo.AccessToken == "" {
		return false
	}

	// Check if token is expired (with 5 minute buffer)
	return time.Now().Add(5 * time.Minute).Before(tokenInfo.ExpiresAt)
}

// CreateTokenSource creates an OAuth2 token source for Gmail API calls
func (s *GmailOAuthService) CreateTokenSource(ctx context.Context, tokenInfo *GmailTokenInfo) oauth2.TokenSource {
	token := &oauth2.Token{
		AccessToken:  tokenInfo.AccessToken,
		RefreshToken: tokenInfo.RefreshToken,
		TokenType:    tokenInfo.TokenType,
		Expiry:       tokenInfo.ExpiresAt,
	}

	return s.config.TokenSource(ctx, token)
}

// CreateGmailService creates a Gmail API service with the provided token
func (s *GmailOAuthService) CreateGmailService(ctx context.Context, tokenInfo *GmailTokenInfo) (*gmail.Service, error) {
	tokenSource := s.CreateTokenSource(ctx, tokenInfo)
	service, err := gmail.NewService(ctx, option.WithTokenSource(tokenSource))
	if err != nil {
		return nil, fmt.Errorf("failed to create Gmail service: %w", err)
	}

	return service, nil
}

// TestConnection tests the Gmail API connection with the provided token
func (s *GmailOAuthService) TestConnection(ctx context.Context, tokenInfo *GmailTokenInfo) error {
	service, err := s.CreateGmailService(ctx, tokenInfo)
	if err != nil {
		return err
	}

	// Test connection by getting user profile
	_, err = service.Users.GetProfile("me").Context(ctx).Do()
	if err != nil {
		return fmt.Errorf("failed to test Gmail connection: %w", err)
	}

	return nil
}