package handlers

import (
	"context"
	"net/http"
	"time"

	"follow-email-backend/config"
	"follow-email-backend/internal/models"
	"follow-email-backend/internal/services"
	"github.com/clerk/clerk-sdk-go/v2"
	"github.com/clerk/clerk-sdk-go/v2/user"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type AuthHandler struct {
	config *config.Config
	db     *gorm.DB
}

type AuthResponse struct {
	User      *models.User `json:"user"`
	Message   string       `json:"message"`
}

type UserInfoResponse struct {
	User *models.User `json:"user"`
}

type UserStatusResponse struct {
	UserExists    bool `json:"user_exists"`
	GmailConsent  bool `json:"gmail_consent"`
	UserID        string `json:"user_id,omitempty"`
	ClerkID       string `json:"clerk_id"`
}

// NewAuthHandler creates a new authentication handler
func NewAuthHandler(cfg *config.Config, db *gorm.DB) *AuthHandler {
	// Initialize Clerk with secret key
	clerk.SetKey(cfg.ClerkSecretKey)
	return &AuthHandler{
		config: cfg,
		db:     db,
	}
}

// GetUserInfo retrieves user information from Clerk session
func (h *AuthHandler) GetUserInfo(c *gin.Context) {
	// Get user ID from the authenticated session (set by middleware)
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "User not authenticated",
		})
		return
	}

	// Get user from Clerk
	clerkUser, err := user.Get(context.Background(), userID.(string))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to get user information",
		})
		return
	}

	// Try to get user from database first
	var dbUser models.User
	result := h.db.Where("clerk_id = ?", clerkUser.ID).First(&dbUser)
	if result.Error == nil {
		// User exists in database, update with latest Clerk data
		updatedUser := convertClerkUserToAppUser(clerkUser)
		updatedUser.ID = dbUser.ID
		updatedUser.CreatedAt = dbUser.CreatedAt
		
		// Update user in database
		h.db.Save(updatedUser)
		
		c.JSON(http.StatusOK, UserInfoResponse{
			User: updatedUser,
		})
		return
	}

	// User not in database, convert from Clerk data
	appUser := convertClerkUserToAppUser(clerkUser)

	c.JSON(http.StatusOK, UserInfoResponse{
		User: appUser,
	})
}

// Logout handles user logout
func (h *AuthHandler) Logout(c *gin.Context) {
	// Clerk handles logout on the client side
	c.JSON(http.StatusOK, gin.H{
		"message": "Logged out successfully",
	})
}

// convertClerkUserToAppUser converts a Clerk user to our application user model
func convertClerkUserToAppUser(clerkUser *clerk.User) *models.User {
	var email string
	if len(clerkUser.EmailAddresses) > 0 {
		email = clerkUser.EmailAddresses[0].EmailAddress
	}

	var name string
	var firstName, lastName string
	if clerkUser.FirstName != nil {
		firstName = *clerkUser.FirstName
	}
	if clerkUser.LastName != nil {
		lastName = *clerkUser.LastName
	}

	if firstName != "" && lastName != "" {
		name = firstName + " " + lastName
	} else if firstName != "" {
		name = firstName
	} else if lastName != "" {
		name = lastName
	}

	// Determine provider from external accounts
	var provider string
	if len(clerkUser.ExternalAccounts) > 0 {
		provider = clerkUser.ExternalAccounts[0].Provider
	} else {
		provider = "clerk"
	}

	// Safely convert Clerk timestamps (they are in milliseconds)
	createdAt := time.Now()
	updatedAt := time.Now()
	
	// Clerk timestamps are in milliseconds, convert to seconds
	if clerkUser.CreatedAt > 0 && clerkUser.CreatedAt < 4102444800000 { // Valid range check (before year 2100)
		createdAt = time.Unix(clerkUser.CreatedAt/1000, 0)
	}
	if clerkUser.UpdatedAt > 0 && clerkUser.UpdatedAt < 4102444800000 { // Valid range check (before year 2100)
		updatedAt = time.Unix(clerkUser.UpdatedAt/1000, 0)
	}

	return &models.User{
		// ID will be auto-generated as UUID
		ClerkID:   &clerkUser.ID,
		Email:     email,
		Name:      name,
		FirstName: firstName,
		LastName:  lastName,
		Username:  clerkUser.Username,
		ImageURL:  clerkUser.ImageURL,
		Provider:  provider,
		IsActive:  true,
		CreatedAt: createdAt,
		UpdatedAt: updatedAt,
	}
}

// CheckUserStatus checks if user exists in database and Gmail consent status
func (h *AuthHandler) CheckUserStatus(c *gin.Context) {
	// Create Clerk auth service
	clerkAuthService := services.NewClerkAuthService(h.config, h.db)
	
	// Verify JWT and get/create user
	authResult, err := clerkAuthService.VerifyJWTAndGetUser(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": err.Error(),
		})
		return
	}

	// Check Gmail consent status
	var gmailConsent models.GmailConsent
	consentResult := h.db.Where("user_id = ?", authResult.User.ID).First(&gmailConsent)
	
	hasGmailConsent := false
	if consentResult.Error == nil {
		hasGmailConsent = gmailConsent.GmailConsent
	}

	c.JSON(http.StatusOK, UserStatusResponse{
		UserExists:   true, // Always true since we create user if not exists
		GmailConsent: hasGmailConsent,
		UserID:       authResult.User.ID.String(),
		ClerkID:      authResult.ClerkID,
	})
}