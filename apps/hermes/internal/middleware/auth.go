package middleware

import (
	"context"
	"net/http"
	"strings"

	"follow-email-backend/config"
	"follow-email-backend/internal/models"
	"github.com/clerk/clerk-sdk-go/v2"
	"github.com/clerk/clerk-sdk-go/v2/jwt"
	"github.com/gin-gonic/gin"
)

type AuthMiddleware struct {
	config *config.Config
}

// NewAuthMiddleware creates a new authentication middleware
func NewAuthMiddleware(cfg *config.Config) *AuthMiddleware {
	// Initialize Clerk with the secret key
	clerk.SetKey(cfg.ClerkSecretKey)
	return &AuthMiddleware{
		config: cfg,
	}
}

// RequireAuth middleware that validates JWT tokens
func (m *AuthMiddleware) RequireAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Get token from Authorization header
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "Authorization header is required",
			})
			c.Abort()
			return
		}

		// Check if header starts with "Bearer "
		if !strings.HasPrefix(authHeader, "Bearer ") {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "Invalid authorization header format",
			})
			c.Abort()
			return
		}

		// Extract token
		tokenString := strings.TrimPrefix(authHeader, "Bearer ")
		if tokenString == "" {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "Token is required",
			})
			c.Abort()
			return
		}

		// Parse and validate token
		claims, err := m.validateToken(tokenString)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "Invalid or expired token",
			})
			c.Abort()
			return
		}

		// Set user information in context
		c.Set("user_id", claims.UserID)
		c.Set("user_email", claims.Email)
		c.Set("user_provider", claims.Provider)
		c.Set("claims", claims)

		c.Next()
	}
}

// OptionalAuth middleware that validates JWT tokens but doesn't require them
func (m *AuthMiddleware) OptionalAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Get token from Authorization header
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.Next()
			return
		}

		// Check if header starts with "Bearer "
		if !strings.HasPrefix(authHeader, "Bearer ") {
			c.Next()
			return
		}

		// Extract token
		tokenString := strings.TrimPrefix(authHeader, "Bearer ")
		if tokenString == "" {
			c.Next()
			return
		}

		// Parse and validate token
		claims, err := m.validateToken(tokenString)
		if err != nil {
			// Invalid token, but continue without authentication
			c.Next()
			return
		}

		// Set user information in context
		c.Set("user_id", claims.UserID)
		c.Set("user_email", claims.Email)
		c.Set("user_provider", claims.Provider)
		c.Set("claims", claims)

		c.Next()
	}
}

// validateToken parses and validates a Clerk JWT token
func (m *AuthMiddleware) validateToken(tokenString string) (*models.JWTClaims, error) {
	// Verify the Clerk JWT token
	claims, err := jwt.Verify(context.Background(), &jwt.VerifyParams{
		Token: tokenString,
	})
	if err != nil {
		return nil, err
	}

	// Convert Clerk claims to our custom claims structure
	return &models.JWTClaims{
		UserID:   claims.Subject, // Clerk uses Subject as user ID
		Email:    "",            // Will be populated from user data if needed
		Provider: "clerk",
	}, nil
}

// GetUserID extracts user ID from gin context
func GetUserID(c *gin.Context) (string, bool) {
	userID, exists := c.Get("user_id")
	if !exists {
		return "", false
	}
	id, ok := userID.(string)
	return id, ok
}

// GetUserEmail extracts user email from gin context
func GetUserEmail(c *gin.Context) (string, bool) {
	userEmail, exists := c.Get("user_email")
	if !exists {
		return "", false
	}
	email, ok := userEmail.(string)
	return email, ok
}

// GetUserProvider extracts user provider from gin context
func GetUserProvider(c *gin.Context) (string, bool) {
	userProvider, exists := c.Get("user_provider")
	if !exists {
		return "", false
	}
	provider, ok := userProvider.(string)
	return provider, ok
}

// GetClaims extracts JWT claims from gin context
func GetClaims(c *gin.Context) (*models.JWTClaims, bool) {
	claims, exists := c.Get("claims")
	if !exists {
		return nil, false
	}
	jwtClaims, ok := claims.(*models.JWTClaims)
	return jwtClaims, ok
}