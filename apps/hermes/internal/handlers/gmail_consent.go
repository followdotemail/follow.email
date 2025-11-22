package handlers

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"follow-email-backend/config"
	"follow-email-backend/internal/models"
	"follow-email-backend/internal/queue"
	"follow-email-backend/pkg/oauth"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// GmailConsentHandler handles Gmail consent and OAuth operations
type GmailConsentHandler struct {
	config            *config.Config
	db                *gorm.DB
	gmailOAuthService *oauth.GmailOAuthService
	qstashService     *queue.QStashService
}

// GmailConsentRequest represents a request to initiate Gmail consent
type GmailConsentRequest struct {
	ReturnURL string `json:"return_url,omitempty"`
}

// GmailConsentResponse represents the response for Gmail consent initiation
type GmailConsentResponse struct {
	AuthURL string `json:"auth_url"`
	State   string `json:"state"`
	Message string `json:"message"`
}

// GmailCallbackRequest represents the OAuth callback parameters
type GmailCallbackRequest struct {
	Code  string `form:"code" binding:"required"`
	State string `form:"state"` // Optional - not present in direct OAuth URLs
	Error string `form:"error"`
}

// GmailStatusResponse represents the Gmail connection status
type GmailStatusResponse struct {
	Connected        bool       `json:"connected"`
	ConsentGiven     bool       `json:"consent_given"`
	ConsentDate      *time.Time `json:"consent_date"`
	SyncEnabled      bool       `json:"sync_enabled"`
	LastSyncAt       *time.Time `json:"last_sync_at"`
	EmailAddress     string     `json:"email_address,omitempty"`
	ConnectionStatus string     `json:"connection_status"`
}

// NewGmailConsentHandler creates a new Gmail consent handler
func NewGmailConsentHandler(cfg *config.Config, db *gorm.DB, gmailOAuthService *oauth.GmailOAuthService, qstashService *queue.QStashService) *GmailConsentHandler {
	return &GmailConsentHandler{
		config:            cfg,
		db:                db,
		gmailOAuthService: gmailOAuthService,
		qstashService:     qstashService,
	}
}

// InitiateConsent starts the Gmail OAuth consent flow
func (h *GmailConsentHandler) InitiateConsent(c *gin.Context) {
	// Get user ID from context (set by auth middleware)
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "User not authenticated",
		})
		return
	}

	var req GmailConsentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		// Optional request body, continue without binding
	}

	// Generate state parameter for CSRF protection
	// Include return_url in state if provided
	state := fmt.Sprintf("%s_%d", userID.(string), time.Now().Unix())
	if req.ReturnURL != "" {
		// Encode return_url in base64 and append to state
		encodedURL := base64.URLEncoding.EncodeToString([]byte(req.ReturnURL))
		state = fmt.Sprintf("%s_return_%s", state, encodedURL)
	}

	// Get authorization URL
	authURL := h.gmailOAuthService.GetAuthURL(state)

	// Fix Unicode encoding issue: replace \u0026 with & for proper URL format
	authURL = strings.ReplaceAll(authURL, "\\u0026", "&")

	// Create a custom JSON response with unescaped HTML
	response := GmailConsentResponse{
		AuthURL: authURL,
		State:   state,
		Message: "Please visit the auth_url to grant Gmail access permissions",
	}

	// Use custom JSON encoding to prevent HTML escaping
	c.Header("Content-Type", "application/json")
	encoder := json.NewEncoder(c.Writer)
	encoder.SetEscapeHTML(false)
	c.Status(http.StatusOK)
	encoder.Encode(response)
}

// HandleCallback processes the OAuth callback from Google
func (h *GmailConsentHandler) HandleCallback(c *gin.Context) {
	var req GmailCallbackRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		// If there's an error parsing parameters, redirect to frontend with error
		h.redirectWithError(c, req.State, "Invalid callback parameters", err.Error())
		return
	}

	// Extract return_url from state if present
	returnURL := h.extractReturnURL(req.State)

	// Check for OAuth error
	if req.Error != "" {
		h.redirectWithError(c, req.State, "OAuth authorization failed", req.Error)
		return
	}

	// Extract user ID from state (basic validation)
	// In production, you should store and validate state more securely
	ctx := context.Background()

	// Exchange code for token
	tokenInfo, err := h.gmailOAuthService.ExchangeCode(ctx, req.Code)
	if err != nil {
		h.redirectWithError(c, req.State, "Failed to exchange authorization code", err.Error())
		return
	}

	DebugTextPrint(fmt.Sprintf("Printing token info: %+v", tokenInfo))
	DebugTextPrint(fmt.Sprintf("\n\nPrinting refresh token: %v & scope: %v", tokenInfo.RefreshToken, tokenInfo.Scope))

	// Get user info from Google
	userInfo, err := h.gmailOAuthService.GetUserInfo(ctx, tokenInfo.AccessToken)
	if err != nil {
		h.redirectWithError(c, req.State, "Failed to get user information", err.Error())
		return
	}

	// Find user by email. If user does not exists, redirect to sign up page.
	var user models.User
	result := h.db.Select("id", "email").Where("email = ?", userInfo.Email).First(&user)
	if result.Error != nil {
		if result.Error == gorm.ErrRecordNotFound {
			h.redirectWithError(c, req.State, "User not found", "Please sign up or login with this email address first.")
			return
		} else {
			h.redirectWithError(c, req.State, "Database error", result.Error.Error())
			return
		}
	}

	// Test Gmail connection
	if err := h.gmailOAuthService.TestConnection(ctx, tokenInfo); err != nil {
		h.redirectWithError(c, req.State, "Failed to connect to Gmail", err.Error())
		return
	}

	// Store or update OAuth token
	now := time.Now()
	var oauthToken models.OAuthToken
	tokenResult := h.db.Select("id", "user_id", "provider", "access_token", "refresh_token", "token_type", "expires_at", "scope").Where("user_id = ? AND provider = ?", user.ID, "gmail").First(&oauthToken)
	DebugTextPrint(fmt.Sprintf("Printing token result: %+v", tokenResult))
	if tokenResult.Error == gorm.ErrRecordNotFound {
		DebugTextPrint("Token not found, creating new token record")
		// Create new token record
		oauthToken = models.OAuthToken{
			UserID:       user.ID,
			Provider:     "gmail",
			AccessToken:  tokenInfo.AccessToken,
			RefreshToken: tokenInfo.RefreshToken,
			TokenType:    tokenInfo.TokenType,
			ExpiresAt:    tokenInfo.ExpiresAt,
			Scope:        tokenInfo.Scope,
			CreatedAt:    now,
			UpdatedAt:    now,
		}
		if err := h.db.Create(&oauthToken).Error; err != nil {
			h.redirectWithError(c, req.State, "Failed to store OAuth token", err.Error())
			return
		}
	} else {
		// Update existing token
		updates := map[string]interface{}{
			"access_token": tokenInfo.AccessToken,
			"token_type":   tokenInfo.TokenType,
			"expires_at":   tokenInfo.ExpiresAt,
			"scope":        tokenInfo.Scope,
			"updated_at":   now,
		}

		// Only include refresh_token if it's not empty
		if tokenInfo.RefreshToken != "" {
			updates["refresh_token"] = tokenInfo.RefreshToken
		}

		// Use Updates to only modify specified fields
		if err := h.db.Model(&oauthToken).Updates(updates).Error; err != nil {
			h.redirectWithError(c, req.State, "Failed to update OAuth token", err.Error())
			return
		}
	}

	// Update user consent status
	var gmailConsent models.GmailConsent
	consentResult := h.db.Select("id", "user_id").Where("user_id = ?", user.ID).First(&gmailConsent)

	if consentResult.Error == gorm.ErrRecordNotFound {
		// Create new consent record
		gmailConsent = models.GmailConsent{
			UserID:           user.ID,
			GmailConsent:     true,
			GmailConsentDate: &now,
			GmailSyncEnabled: true,
			CreatedAt:        now,
			UpdatedAt:        now,
		}
		if err := h.db.Create(&gmailConsent).Error; err != nil {
			h.redirectWithError(c, req.State, "Failed to create consent record", err.Error())
			return
		}
	} else if consentResult.Error != nil {
		h.redirectWithError(c, req.State, "Failed to check consent record", consentResult.Error.Error())
		return
	} else {
		// Update existing consent record
		updates := map[string]interface{}{
			"gmail_consent":      true,
			"gmail_consent_date": &now,
			"gmail_sync_enabled": true,
			"updated_at":         now,
		}

		if err := h.db.Model(&gmailConsent).Updates(updates).Error; err != nil {
			h.redirectWithError(c, req.State, "Failed to update consent record", err.Error())
			return
		}
	}

	// Update user timestamp - NOT REQUIRED AS OF NOW (AI GENERATED CODE)
	// user.UpdatedAt = now
	// if err := h.db.Save(&user).Error; err != nil {
	// 	h.redirectWithError(c, req.State, "Failed to update user", err.Error())
	// 	return
	// }

	// Automatically trigger email sync for first-time consent
	if h.qstashService != nil {
		syncMessage := queue.EmailSyncMessage{
			UserID:    user.ID.String(),
			MessageID: "", // Full sync for first time
		}

		// Queue the email sync job asynchronously
		if err := h.qstashService.PublishEmailSync(context.Background(), syncMessage); err != nil {
			// Log the error but don't fail the consent process
			DebugWarningTextPrint(fmt.Sprintf("Warning: Failed to queue initial email sync for user %s: %v", user.ID, err))
		} else {
			DebugSuccessTextPrint(fmt.Sprintf("Successfully queued initial email sync for user %s", user.ID))
		}
	}

	// Redirect to frontend with success
	h.redirectWithSuccess(c, returnURL, userInfo.Email)
}

// GetStatus returns the current Gmail connection status
func (h *GmailConsentHandler) GetStatus(c *gin.Context) {
	// For development: use a default user ID when authentication is disabled
	userID, exists := c.Get("user_id")
	if !exists {
		// Development mode: return mock status
		c.JSON(http.StatusOK, GmailStatusResponse{
			Connected:        false,
			ConsentGiven:     false,
			ConsentDate:      nil,
			SyncEnabled:      false,
			LastSyncAt:       nil,
			EmailAddress:     "",
			ConnectionStatus: "Development mode - no authentication required",
		})
		return
	}

	// Get user from database
	var user models.User
	result := h.db.Where("clerk_id = ?", userID.(string)).First(&user)
	if result.Error != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}

	// Get OAuth token if exists
	var oauthToken models.OAuthToken
	tokenResult := h.db.Where("user_id = ? AND provider = ?", user.ID, "gmail").First(&oauthToken)

	// Get Gmail consent record
	var gmailConsent models.GmailConsent
	consentResult := h.db.Where("user_id = ?", user.ID).First(&gmailConsent)

	status := GmailStatusResponse{
		Connected:        false,
		ConsentGiven:     false,
		ConsentDate:      nil,
		SyncEnabled:      false,
		LastSyncAt:       nil,
		ConnectionStatus: "not_connected",
	}

	// Set consent status if record exists
	if consentResult.Error == nil {
		status.ConsentGiven = gmailConsent.GmailConsent
		status.ConsentDate = gmailConsent.GmailConsentDate
		status.SyncEnabled = gmailConsent.GmailSyncEnabled
		status.LastSyncAt = gmailConsent.LastGmailSyncAt
		status.Connected = gmailConsent.GmailConsent && tokenResult.Error == nil
	}

	if tokenResult.Error == nil {
		// Test connection if token exists
		tokenInfo := &oauth.GmailTokenInfo{
			AccessToken:  oauthToken.AccessToken,
			RefreshToken: oauthToken.RefreshToken,
			TokenType:    oauthToken.TokenType,
			ExpiresAt:    oauthToken.ExpiresAt,
			Scope:        oauthToken.Scope,
		}

		ctx := context.Background()
		if err := h.gmailOAuthService.TestConnection(ctx, tokenInfo); err != nil {
			status.ConnectionStatus = "error"
			status.Connected = false
		} else {
			status.ConnectionStatus = "active"
			// Get email address from Google
			if userInfo, err := h.gmailOAuthService.GetUserInfo(ctx, tokenInfo.AccessToken); err == nil {
				status.EmailAddress = userInfo.Email
			}
		}
	} else {
		status.ConnectionStatus = "not_connected"
	}

	c.JSON(http.StatusOK, status)
}

// RevokeConsent revokes Gmail access and deletes stored tokens
func (h *GmailConsentHandler) RevokeConsent(c *gin.Context) {
	// For development: use a default user ID when authentication is disabled
	userID, exists := c.Get("user_id")
	if !exists {
		// Development mode: return success message
		c.JSON(http.StatusOK, gin.H{
			"message": "Development mode - consent revocation simulated",
			"note":    "In production, this would require authentication and revoke actual tokens",
		})
		return
	}

	// Get user from database
	var user models.User
	result := h.db.Where("clerk_id = ?", userID.(string)).First(&user)
	if result.Error != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}

	// Delete OAuth token
	if err := h.db.Where("user_id = ? AND provider = ?", user.ID, "gmail").Delete(&models.OAuthToken{}).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to delete OAuth token",
			"details": err.Error(),
		})
		return
	}

	// Update Gmail consent status
	var gmailConsent models.GmailConsent
	consentResult := h.db.Where("user_id = ?", user.ID).First(&gmailConsent)

	if consentResult.Error == nil {
		// Update existing consent record
		now := time.Now()
		gmailConsent.GmailConsent = false
		gmailConsent.GmailConsentDate = nil
		gmailConsent.GmailSyncEnabled = false
		gmailConsent.LastGmailSyncAt = nil
		gmailConsent.GmailHistoryId = ""
		gmailConsent.UpdatedAt = now

		if err := h.db.Save(&gmailConsent).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error":   "Failed to update consent record",
				"details": err.Error(),
			})
			return
		}
	}

	// Update user timestamp
	now := time.Now()
	user.UpdatedAt = now
	if err := h.db.Save(&user).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to update user",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":    "Gmail access revoked successfully",
		"revoked_at": now,
	})
}

// extractReturnURL extracts the return URL from the state parameter
func (h *GmailConsentHandler) extractReturnURL(state string) string {
	// Check if state contains return URL
	if strings.Contains(state, "_return_") {
		parts := strings.Split(state, "_return_")
		if len(parts) == 2 {
			// Decode the base64 encoded URL
			if decoded, err := base64.URLEncoding.DecodeString(parts[1]); err == nil {
				return string(decoded)
			}
		}
	}
	return ""
}

// redirectWithSuccess redirects to frontend with success parameters
func (h *GmailConsentHandler) redirectWithSuccess(c *gin.Context, returnURL, userEmail string) {
	if returnURL == "" {
		// Fallback: return JSON if no return URL provided
		c.JSON(http.StatusOK, gin.H{
			"message":    "Gmail access granted successfully",
			"user_email": userEmail,
			"status":     "success",
		})
		return
	}

	// Parse the return URL to add query parameters
	parsedURL, err := url.Parse(returnURL)
	if err != nil {
		// If URL parsing fails, return JSON
		c.JSON(http.StatusOK, gin.H{
			"message":    "Gmail access granted successfully",
			"user_email": userEmail,
			"status":     "success",
			"note":       "Invalid return URL provided",
		})
		return
	}

	// Add success parameters to the URL
	query := parsedURL.Query()
	query.Set("status", "success")
	query.Set("message", "Gmail access granted successfully")
	query.Set("user_email", userEmail)
	parsedURL.RawQuery = query.Encode()

	// Redirect to the frontend
	c.Redirect(http.StatusFound, parsedURL.String())
}

// redirectWithError redirects to frontend with error parameters
func (h *GmailConsentHandler) redirectWithError(c *gin.Context, state, errorMsg, details string) {
	returnURL := h.extractReturnURL(state)

	if returnURL == "" {
		// Fallback: return JSON if no return URL provided
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   errorMsg,
			"details": details,
			"status":  "error",
		})
		return
	}

	// Parse the return URL to add query parameters
	parsedURL, err := url.Parse(returnURL)
	if err != nil {
		// If URL parsing fails, return JSON
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   errorMsg,
			"details": details,
			"status":  "error",
			"note":    "Invalid return URL provided",
		})
		return
	}

	// Add error parameters to the URL
	query := parsedURL.Query()
	query.Set("status", "error")
	query.Set("error", errorMsg)
	query.Set("details", details)
	parsedURL.RawQuery = query.Encode()

	// Redirect to the frontend
	c.Redirect(http.StatusFound, parsedURL.String())
}
