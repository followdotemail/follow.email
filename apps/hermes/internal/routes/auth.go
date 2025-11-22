package routes

import (
	"follow-email-backend/internal/handlers"
	"follow-email-backend/internal/middleware"
	"github.com/gin-gonic/gin"
)

// SetupPublicAuthRoutes configures public authentication routes (no auth required)
func SetupPublicAuthRoutes(router *gin.RouterGroup, authHandler *handlers.AuthHandler) {
	// Public authentication routes with rate limiting
	auth := router.Group("/auth")
	auth.Use(middleware.AuthRateLimit())
	{
		// No public endpoints currently needed
	}
}

// SetupProtectedAuthRoutes configures protected authentication routes (auth required)
func SetupProtectedAuthRoutes(router *gin.RouterGroup, authHandler *handlers.AuthHandler) {
	// Protected authentication routes with rate limiting
	auth := router.Group("/auth")
	auth.Use(middleware.AuthRateLimit())
	{
		// Protected endpoints that require authentication
		auth.GET("/user", authHandler.GetUserInfo)
		auth.POST("/logout", authHandler.Logout)
		auth.GET("/status", authHandler.CheckUserStatus)
	}
}