package routes

import (
	"net/http"

	"follow-email-backend/internal/handlers"
	"follow-email-backend/internal/middleware"

	"github.com/gin-gonic/gin"
)

// SetupProtectedRoutes configures all routes that require authentication
func SetupProtectedRoutes(
	router *gin.RouterGroup,
	privacyHandler *handlers.PrivacyHandler,
	gmailConsentHandler *handlers.GmailConsentHandler,
	emailHandler *handlers.EmailHandler,
	labelHandler *handlers.LabelHandler,
) {
	// User profile endpoint
	router.GET("/profile", func(c *gin.Context) {
		userID, _ := middleware.GetUserID(c)
		userEmail, _ := middleware.GetUserEmail(c)
		c.JSON(http.StatusOK, gin.H{
			"user_id": userID,
			"email":   userEmail,
		})
	})

	// Email management endpoints
	emailSync := router.Group("/emails")
	emailSync.Use(middleware.EmailSyncRateLimit())
	{
		emailSync.POST("/sync", emailHandler.SyncEmails)
		emailSync.GET("/sync/status", emailHandler.GetSyncStatus)

		// Email query endpoints
		emailSync.GET("", emailHandler.GetEmails)
		emailSync.GET("/:id", emailHandler.GetEmailByID)
		emailSync.GET("/:id/attachments/:attachment_id", emailHandler.DownloadAttachment)

		// Email sending endpoint
		emailSync.POST("/send", emailHandler.SendEmail)

		// Email status update endpoint
		emailSync.PATCH("/:id/status", emailHandler.UpdateEmailStatus)
	}

	// AI analysis endpoints with specific rate limits
	ai := router.Group("/emails")
	ai.Use(middleware.AIAnalysisRateLimit())
	{
		ai.POST("/analyze", emailHandler.AnalyzeEmail)
		ai.POST("/generate-response", emailHandler.GenerateResponse)
		ai.POST("/:emailId/follow-up", emailHandler.ScheduleFollowUp)
		ai.POST("/smart-search", emailHandler.SmartSearch) // AI-powered natural language search
	}

	// ADD: Label management endpoints
	labels := router.Group("/labels")
	{
		labels.GET("", labelHandler.GetLabels)
		labels.POST("/sync", labelHandler.SyncLabels)
	}

	// Setup privacy routes
	SetupPrivacyRoutes(router, privacyHandler)

	// Gmail consent routes moved to public for development
	// SetupGmailRoutes(router, gmailConsentHandler)
}
