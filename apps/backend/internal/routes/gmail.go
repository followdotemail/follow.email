package routes

import (
	"follow-email-backend/internal/handlers"
	"follow-email-backend/internal/middleware"
	"github.com/gin-gonic/gin"
)

// SetupGmailRoutes configures Gmail consent and sync routes
// protectedRouter requires authentication, publicRouter does not
func SetupGmailRoutes(protectedRouter *gin.RouterGroup, publicRouter *gin.RouterGroup, gmailConsentHandler *handlers.GmailConsentHandler) {
	// Protected Gmail routes (require authentication)
	protectedGmail := protectedRouter.Group("/gmail")
	protectedGmail.Use(middleware.AuthRateLimit()) // Apply rate limiting
	{
		// Gmail consent flow - requires auth to know which user is initiating
		protectedGmail.POST("/consent/initiate", gmailConsentHandler.InitiateConsent)
		protectedGmail.GET("/consent/status", gmailConsentHandler.GetStatus)
		protectedGmail.DELETE("/consent/revoke", gmailConsentHandler.RevokeConsent)
	}

	// Public Gmail routes (no auth required - OAuth callback from Google)
	publicGmail := publicRouter.Group("/gmail")
	{
		// OAuth callback - no auth required as Google redirects here
		publicGmail.GET("/consent/callback", gmailConsentHandler.HandleCallback)
	}
}