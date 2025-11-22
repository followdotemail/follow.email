// Package main Follow Email Backend API
//
// This is the main API server for the Follow Email Backend application.
// It provides endpoints for email management, AI-powered analysis, and user authentication.
//
// Terms Of Service:
//
// there are no TOS at this moment, use at your own risk we take no responsibility
//
//	Schemes: http, https
//	Host: localhost:8081
//	BasePath: /api/v1
//	Version: 1.0.0
//	License: MIT http://opensource.org/licenses/MIT
//	Contact: Tanmoy Saha<tanmoy@example.com> http://example.com
//
//	Consumes:
//	- application/json
//
//	Produces:
//	- application/json
//
//	Security:
//	- bearer:
//
//	SecurityDefinitions:
//	bearer:
//	     type: apiKey
//	     name: Authorization
//	     in: header
//	     description: Bearer token for authentication
//
// swagger:meta
package main

import (
	"context"
	"log"
	"net/http"
	"os"

	"follow-email-backend/config"
	"follow-email-backend/internal/database"
	"follow-email-backend/internal/handlers"
	"follow-email-backend/internal/middleware"
	"follow-email-backend/internal/models"
	"follow-email-backend/internal/queue"
	"follow-email-backend/internal/routes"
	"follow-email-backend/internal/services"
	"follow-email-backend/pkg/ai"
	"follow-email-backend/pkg/encryption"
	"follow-email-backend/pkg/oauth"
	"follow-email-backend/pkg/storage"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
)

func main() {
	// Load environment variables from .env file if it exists
	env := os.Getenv("ENVIRONMENT")
	if env != "staging" && env != "production" {
		if err := godotenv.Load(); err != nil {
			log.Println("Warning: No .env file found. Using system environment variables.")
			log.Println("If you need to set environment variables, create a .env file in the project root.")
		} else {
			log.Println("Successfully loaded environment variables from .env file")
		}
	} else {
		log.Printf("Running in %s environment - using system environment variables", env)
	}

	// Load configuration
	cfg := config.Load()

	// Set Gin mode based on environment
	if cfg.Environment == "production" {
		gin.SetMode(gin.ReleaseMode)
	}

	// Initialize Gin router
	r := gin.Default()

	// Add middleware
	r.Use(middleware.ErrorHandler())
	r.Use(middleware.GlobalRateLimit())
	// CORS middleware - Allow all origins
	r.Use(func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS, PATCH")
		c.Header("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Request-ID, Accept, Origin, X-Requested-With")
		c.Header("Access-Control-Allow-Credentials", "true")
		c.Header("Access-Control-Max-Age", "86400")
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}
		c.Next()
	})

	// Initialize database
	db, err := database.Connect(cfg)
	if err != nil {
		log.Fatal("Failed to connect to database:", err)
	}
	defer database.Close(db)

	// Run database migrations
	if err := database.Migrate(db); err != nil {
		log.Fatal("Failed to run database migrations:", err)
	}
	log.Println("Database migrations completed successfully")

	// Initialize encryption service
	encryptionService, err := encryption.NewEncryptionService(cfg.EncryptionKey)
	if err != nil {
		log.Fatal("Failed to initialize encryption service:", err)
	}

	// Set global encryption service for models
	models.SetEncryptionService(encryptionService)
	log.Println("Encryption service initialized successfully")

	// Initialize AI service
	aiService, err := ai.NewAIService(cfg.GeminiAPIKey)
	if err != nil {
		log.Printf("Warning: Failed to initialize AI service: %v", err)
		// Continue without AI service for now
	}

	// Initialize QStash service
	qstashService := queue.NewQStashService(cfg.QStashToken, cfg.BaseURL+"/api/v1/webhooks")
	log.Println("QStash service initialized successfully")

	// Initialize storage service
	storageConfig := &storage.StorageConfig{
		BucketName: cfg.S3BucketName,
		Region:     cfg.AWSRegion,
		AccessKey:  cfg.AWSAccessKeyID,
		SecretKey:  cfg.AWSSecretAccessKey,
		Endpoint:   "", // Use default AWS endpoint
	}
	storageService, err := storage.NewS3Service(context.Background(), storageConfig)
	if err != nil {
		log.Printf("Warning: Failed to initialize storage service: %v", err)
		// Continue without storage service for now
	}

	// Initialize Gmail OAuth service
	gmailOAuthService := oauth.NewGmailOAuthService(cfg.GoogleClientID, cfg.GoogleClientSecret, cfg.BaseURL+"/api/v1/gmail/consent/callback")

	// Initialize Gmail services
	gmailTokenService := services.NewGmailTokenService(db, gmailOAuthService)
	gmailSyncService := services.NewGmailSyncService(db, gmailOAuthService, gmailTokenService, storageService)

	// Initialize email sync service
	oauthService := oauth.NewOAuthService(cfg)
	emailSyncService := services.NewEmailSyncService(oauthService, gmailOAuthService, db)

	// Initialize privacy service
	privacyService := services.NewPrivacyService(db, storageService)

	// Initialize handlers
	authHandler := handlers.NewAuthHandler(cfg, db)
	emailHandler := handlers.NewEmailHandler(db, emailSyncService, aiService, qstashService, storageService, gmailTokenService, gmailSyncService)
	privacyHandler := handlers.NewPrivacyHandler(privacyService)
	gmailConsentHandler := handlers.NewGmailConsentHandler(cfg, db, gmailOAuthService, qstashService)
	webhookHandler := handlers.NewWebhookHandler(cfg, db, emailSyncService, aiService, gmailSyncService)
	authMiddleware := middleware.NewAuthMiddleware(cfg)

	// Public routes
	public := r.Group("/api/v1")
	routes.SetupPublicRoutes(public, cfg, authHandler, emailHandler)

	// Webhook routes for QStash (no auth required - verified by signature)
	routes.SetupWebhookRoutes(r, webhookHandler)

	// Protected routes (require authentication)
	protected := r.Group("/api/v1")
	protected.Use(authMiddleware.RequireAuth())

	// Gmail routes - callback must be public, others require auth
	routes.SetupGmailRoutes(protected, public, gmailConsentHandler)

	routes.SetupProtectedRoutes(protected, privacyHandler, gmailConsentHandler, emailHandler)
	routes.SetupProtectedAuthRoutes(protected, authHandler)

	// Serve Swagger UI
	r.Static("/swagger-ui", "./swagger-ui")
	r.StaticFile("/swagger.yaml", "./swagger.yaml")

	// Swagger UI redirect
	r.GET("/docs", func(c *gin.Context) {
		c.Redirect(http.StatusMovedPermanently, "/swagger-ui/")
	})

	// Start server
	log.Printf("Starting Follow Email Backend server on 0.0.0.0:%s", cfg.Port)
	if err := r.Run("0.0.0.0:" + cfg.Port); err != nil {
		log.Fatal("Failed to start server:", err)
	}
}
