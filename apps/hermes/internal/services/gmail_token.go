package services

import (
	"context"
	"fmt"
	"time"

	"follow-email-backend/internal/models"
	"follow-email-backend/pkg/oauth"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// GmailTokenService manages Gmail OAuth tokens
type GmailTokenService struct {
	db                *gorm.DB
	gmailOAuthService *oauth.GmailOAuthService
}

// NewGmailTokenService creates a new Gmail token service
func NewGmailTokenService(db *gorm.DB, gmailOAuthService *oauth.GmailOAuthService) *GmailTokenService {
	return &GmailTokenService{
		db:                db,
		gmailOAuthService: gmailOAuthService,
	}
}

// GetValidToken retrieves a valid Gmail token for the user, refreshing if necessary
func (s *GmailTokenService) GetValidToken(ctx context.Context, userID uuid.UUID) (*oauth.GmailTokenInfo, error) {
	// Get token from database
	var oauthToken models.OAuthToken
	result := s.db.Select("id, user_id, provider, access_token, refresh_token, token_type, expires_at, scope, updated_at").
		Where("user_id = ? AND provider = ?", userID, "gmail").First(&oauthToken)
	if result.Error != nil {
		if result.Error == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("no Gmail token found for user %s", userID)
		}
		return nil, fmt.Errorf("failed to get Gmail token: %w", result.Error)
	}

	tokenInfo := &oauth.GmailTokenInfo{
		AccessToken:  oauthToken.AccessToken,
		RefreshToken: oauthToken.RefreshToken,
		TokenType:    oauthToken.TokenType,
		ExpiresAt:    oauthToken.ExpiresAt,
		Scope:        oauthToken.Scope,
	}

	// Check if token is valid
	if s.gmailOAuthService.ValidateToken(tokenInfo) {
		return tokenInfo, nil
	}

	// Token is expired, try to refresh
	if oauthToken.RefreshToken == "" {
		return nil, fmt.Errorf("no refresh token available for user %s", userID)
	}

	newTokenInfo, err := s.gmailOAuthService.RefreshToken(ctx, oauthToken.RefreshToken)
	if err != nil {
		return nil, fmt.Errorf("failed to refresh Gmail token: %w", err)
	}

	// Update token in database
	now := time.Now()
	oauthToken.AccessToken = newTokenInfo.AccessToken
	if newTokenInfo.RefreshToken != "" {
		oauthToken.RefreshToken = newTokenInfo.RefreshToken
	}
	oauthToken.TokenType = newTokenInfo.TokenType
	oauthToken.ExpiresAt = newTokenInfo.ExpiresAt
	oauthToken.UpdatedAt = now

	if err := s.db.Save(&oauthToken).Error; err != nil {
		return nil, fmt.Errorf("failed to update refreshed token: %w", err)
	}

	return newTokenInfo, nil
}

// StoreToken stores or updates a Gmail OAuth token for a user
func (s *GmailTokenService) StoreToken(userID uuid.UUID, tokenInfo *oauth.GmailTokenInfo) error {
	now := time.Now()

	// Check if token already exists
	var oauthToken models.OAuthToken
	result := s.db.Where("user_id = ? AND provider = ?", userID, "gmail").First(&oauthToken)

	if result.Error == gorm.ErrRecordNotFound {
		// Create new token record
		oauthToken = models.OAuthToken{
			UserID:       userID,
			Provider:     "gmail",
			AccessToken:  tokenInfo.AccessToken,
			RefreshToken: tokenInfo.RefreshToken,
			TokenType:    tokenInfo.TokenType,
			ExpiresAt:    tokenInfo.ExpiresAt,
			Scope:        tokenInfo.Scope,
			CreatedAt:    now,
			UpdatedAt:    now,
		}
		return s.db.Create(&oauthToken).Error
	} else if result.Error != nil {
		return fmt.Errorf("failed to check existing token: %w", result.Error)
	}

	// Update existing token
	oauthToken.AccessToken = tokenInfo.AccessToken
	if tokenInfo.RefreshToken != "" {
		oauthToken.RefreshToken = tokenInfo.RefreshToken
	}
	oauthToken.TokenType = tokenInfo.TokenType
	oauthToken.ExpiresAt = tokenInfo.ExpiresAt
	oauthToken.Scope = tokenInfo.Scope
	oauthToken.UpdatedAt = now

	return s.db.Save(&oauthToken).Error
}

// DeleteToken removes a Gmail OAuth token for a user
func (s *GmailTokenService) DeleteToken(userID uuid.UUID) error {
	return s.db.Where("user_id = ? AND provider = ?", userID, "gmail").Delete(&models.OAuthToken{}).Error
}

// ValidateTokenExists checks if a valid Gmail token exists for the user
func (s *GmailTokenService) ValidateTokenExists(ctx context.Context, userID uuid.UUID) (bool, error) {
	var oauthToken models.OAuthToken
	result := s.db.Where("user_id = ? AND provider = ?", userID, "gmail").First(&oauthToken)
	if result.Error != nil {
		if result.Error == gorm.ErrRecordNotFound {
			return false, nil
		}
		return false, result.Error
	}

	tokenInfo := &oauth.GmailTokenInfo{
		AccessToken:  oauthToken.AccessToken,
		RefreshToken: oauthToken.RefreshToken,
		TokenType:    oauthToken.TokenType,
		ExpiresAt:    oauthToken.ExpiresAt,
		Scope:        oauthToken.Scope,
	}

	// Test connection to Gmail
	if err := s.gmailOAuthService.TestConnection(ctx, tokenInfo); err != nil {
		// Try to refresh token if connection fails
		if oauthToken.RefreshToken != "" {
			newTokenInfo, refreshErr := s.gmailOAuthService.RefreshToken(ctx, oauthToken.RefreshToken)
			if refreshErr == nil {
				// Update token and test again
				if storeErr := s.StoreToken(userID, newTokenInfo); storeErr == nil {
					return s.gmailOAuthService.TestConnection(ctx, newTokenInfo) == nil, nil
				}
			}
		}
		return false, nil
	}

	return true, nil
}

// GetTokenInfo retrieves token information without validation
func (s *GmailTokenService) GetTokenInfo(userID uuid.UUID) (*oauth.GmailTokenInfo, error) {
	var oauthToken models.OAuthToken
	result := s.db.Where("user_id = ? AND provider = ?", userID, "gmail").First(&oauthToken)
	if result.Error != nil {
		if result.Error == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("no Gmail token found for user %s", userID)
		}
		return nil, fmt.Errorf("failed to get Gmail token: %w", result.Error)
	}

	return &oauth.GmailTokenInfo{
		AccessToken:  oauthToken.AccessToken,
		RefreshToken: oauthToken.RefreshToken,
		TokenType:    oauthToken.TokenType,
		ExpiresAt:    oauthToken.ExpiresAt,
		Scope:        oauthToken.Scope,
	}, nil
}

// UpdateUserSyncStatus updates the user's Gmail sync status
func (s *GmailTokenService) UpdateUserSyncStatus(userID uuid.UUID, lastSyncAt *time.Time, historyID string) error {
	updates := map[string]interface{}{
		"updated_at": time.Now(),
	}

	if lastSyncAt != nil {
		updates["last_gmail_sync_at"] = *lastSyncAt
	}

	if historyID != "" {
		updates["gmail_history_id"] = historyID
	}

	return s.db.Model(&models.GmailConsent{}).Where("user_id = ?", userID).Updates(updates).Error
}

// GetUsersWithGmailAccess returns all users who have granted Gmail access
func (s *GmailTokenService) GetUsersWithGmailAccess() ([]models.User, error) {
	var users []models.User
	err := s.db.Joins("JOIN gmail_consent ON users.id = gmail_consent.user_id").
		Where("gmail_consent.gmail_consent = ? AND gmail_consent.gmail_sync_enabled = ?", true, true).
		Find(&users).Error
	return users, err
}
