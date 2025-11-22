package routes

import (
	"follow-email-backend/internal/handlers"
	"github.com/gin-gonic/gin"
)

// SetupWebhookRoutes sets up webhook routes for QStash
func SetupWebhookRoutes(r *gin.Engine, webhookHandler *handlers.WebhookHandler) {
	// Create a separate webhook group under /api/v1 that bypasses auth middleware
	// QStash webhooks are verified by signature, not auth tokens
	webhooks := r.Group("/api/v1/webhooks")
	
	webhooks.POST("/email-sync", webhookHandler.HandleEmailSync)
	webhooks.POST("/email-analysis", webhookHandler.HandleEmailAnalysis)
	webhooks.POST("/follow-up", webhookHandler.HandleFollowUp)
	webhooks.POST("/scheduled-task", webhookHandler.HandleScheduledTask)
}