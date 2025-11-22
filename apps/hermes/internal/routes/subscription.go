package routes

import (
	"github.com/gin-gonic/gin"

	"follow-email-backend/internal/handlers"
)

// SetupSubscriptionRoutes sets up subscription-related routes
func SetupSubscriptionRoutes(router *gin.Engine, subscriptionHandler *handlers.SubscriptionHandler) {
	subscriptionGroup := router.Group("/api/v1/subscriptions")
	{
		// Create subscription
		subscriptionGroup.POST("/", subscriptionHandler.CreateSubscription)
		
		// Get user subscription
		subscriptionGroup.GET("/:userID", subscriptionHandler.GetSubscription)
		
		// Update subscription
		subscriptionGroup.PUT("/:userID", subscriptionHandler.UpdateSubscription)
		
		// Cancel subscription
		subscriptionGroup.DELETE("/:userID", subscriptionHandler.CancelSubscription)
		
		// Reactivate subscription
		subscriptionGroup.POST("/:userID/reactivate", subscriptionHandler.ReactivateSubscription)
		
		// Update usage
		subscriptionGroup.POST("/:userID/usage", subscriptionHandler.UpdateUsage)
		
		// Check usage limits
		subscriptionGroup.GET("/:userID/limits", subscriptionHandler.CheckUsageLimits)
	}
}