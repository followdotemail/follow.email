package routes

import (
	"follow-email-backend/internal/handlers"
	"github.com/gin-gonic/gin"
)

// SetupPrivacyRoutes configures privacy/compliance routes
func SetupPrivacyRoutes(router *gin.RouterGroup, privacyHandler *handlers.PrivacyHandler) {
	// Privacy/Compliance endpoints
	privacy := router.Group("/privacy")
	{
		privacy.POST("/consent", privacyHandler.RecordConsent)
		privacy.GET("/consent", privacyHandler.GetConsentStatus)
		privacy.POST("/export-request", privacyHandler.RequestDataExport)
		privacy.POST("/delete-request", privacyHandler.RequestDataDeletion)
		privacy.GET("/export/:requestId", privacyHandler.GetDataExport)
		privacy.GET("/requests", privacyHandler.GetUserRequests)
	}
}