package routes

import (
	"net/http"

	"follow-email-backend/config"
	"follow-email-backend/internal/handlers"

	"github.com/gin-gonic/gin"
)

// SetupPublicRoutes configures all public routes that don't require authentication
func SetupPublicRoutes(router *gin.RouterGroup, cfg *config.Config, authHandler *handlers.AuthHandler) {
	// Health check endpoint
	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status":      "healthy",
			"service":     "follow-email-backend",
			"environment": cfg.Environment,
		})
	})

	// Ping endpoint
	router.GET("/ping", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"message": "pong",
		})
	})

	// Setup public authentication routes
	SetupPublicAuthRoutes(router, authHandler)
}
