package handlers

import (
	"context"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"follow-email-backend/config"
	"follow-email-backend/internal/queue"
	"follow-email-backend/internal/services"
	"follow-email-backend/pkg/ai"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type WebhookHandler struct {
	config           *config.Config
	db               *gorm.DB
	emailSyncService *services.EmailSyncService
	aiService        *ai.AIService
	gmailSyncService *services.GmailSyncService
}

// NewWebhookHandler creates a new webhook handler
func NewWebhookHandler(cfg *config.Config, db *gorm.DB, emailSyncService *services.EmailSyncService, aiService *ai.AIService, gmailSyncService *services.GmailSyncService) *WebhookHandler {
	return &WebhookHandler{
		config:           cfg,
		db:               db,
		emailSyncService: emailSyncService,
		aiService:        aiService,
		gmailSyncService: gmailSyncService,
	}
}

// verifyQStashSignature verifies the QStash webhook signature
func (h *WebhookHandler) verifyQStashSignature(c *gin.Context, body []byte) error {
	signature := c.GetHeader("Upstash-Signature")
	if signature == "" {
		return fmt.Errorf("missing Upstash-Signature header")
	}

	// Parse signature header (format: "v1=<signature>")
	parts := strings.Split(signature, "=")
	if len(parts) != 2 || parts[0] != "v1" {
		return fmt.Errorf("invalid signature format")
	}

	expectedSignature, err := base64.StdEncoding.DecodeString(parts[1])
	if err != nil {
		return fmt.Errorf("invalid signature encoding: %w", err)
	}

	// Try current signing key first
	if h.verifySignatureWithKey(body, expectedSignature, h.config.QStashCurrentSigningKey) {
		return nil
	}

	// Try next signing key (for key rotation)
	if h.config.QStashNextSigningKey != "" && h.verifySignatureWithKey(body, expectedSignature, h.config.QStashNextSigningKey) {
		return nil
	}

	return fmt.Errorf("signature verification failed")
}

// verifySignatureWithKey verifies signature with a specific key
func (h *WebhookHandler) verifySignatureWithKey(body, expectedSignature []byte, key string) bool {
	mac := sha256.New()
	mac.Write([]byte(key))
	mac.Write(body)
	computedSignature := mac.Sum(nil)
	
	return subtle.ConstantTimeCompare(expectedSignature, computedSignature) == 1
}

// HandleEmailSync processes email sync webhook messages
func (h *WebhookHandler) HandleEmailSync(c *gin.Context) {
	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Failed to read request body"})
		return
	}

	// Skip signature verification for local development (QStash CLI doesn't send proper signatures)
	if h.config.Environment != "production" {
		fmt.Println("Skipping QStash signature verification for local development")
	} else {
		// Verify QStash signature in production
		if err := h.verifyQStashSignature(c, body); err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid signature"})
			return
		}
	}

	var msg queue.EmailSyncMessage
	if err := json.Unmarshal(body, &msg); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid message format"})
		return
	}

	// Parse user ID
	userID, err := uuid.Parse(msg.UserID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID format"})
		return
	}

	// Create sync request
	syncReq := &services.GmailSyncRequest{
		UserID: userID,
	}

	// Process email sync using the Gmail sync service
	fmt.Printf("Processing email sync for user: %s, message: %s\n", msg.UserID, msg.MessageID)
	
	result, err := h.gmailSyncService.SyncUserEmails(c.Request.Context(), syncReq)
	if err != nil {
		fmt.Printf("Email sync failed for user %s: %v\n", msg.UserID, err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"status": "error", 
			"message": "Email sync failed",
			"error": err.Error(),
		})
		return
	}

	fmt.Printf("Email sync completed for user %s: %d emails processed, %d new, %d updated\n", 
		msg.UserID, result.EmailsProcessed, result.NewEmails, result.UpdatedEmails)

	c.JSON(http.StatusOK, gin.H{
		"status": "success", 
		"message": "Email sync processed",
		"emails_processed": result.EmailsProcessed,
		"new_emails": result.NewEmails,
		"updated_emails": result.UpdatedEmails,
	})
}

// HandleEmailAnalysis processes email analysis webhook messages
func (h *WebhookHandler) HandleEmailAnalysis(c *gin.Context) {
	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Failed to read request body"})
		return
	}

	// Verify QStash signature
	if err := h.verifyQStashSignature(c, body); err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid signature"})
		return
	}

	var msg queue.EmailAnalysisMessage
	if err := json.Unmarshal(body, &msg); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid message format"})
		return
	}

	// TODO: Process email analysis using AI service
	// For now, just log the message
	fmt.Printf("Processing email analysis for user: %s, email: %s\n", msg.UserID, msg.EmailID)

	c.JSON(http.StatusOK, gin.H{"status": "success", "message": "Email analysis processed"})
}

// HandleFollowUp processes follow-up webhook messages
func (h *WebhookHandler) HandleFollowUp(c *gin.Context) {
	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Failed to read request body"})
		return
	}

	// Verify QStash signature
	if err := h.verifyQStashSignature(c, body); err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid signature"})
		return
	}

	var msg queue.FollowUpMessage
	if err := json.Unmarshal(body, &msg); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid message format"})
		return
	}

	// TODO: Process follow-up using AI service
	// For now, just log the message
	fmt.Printf("Processing follow-up for user: %s, email: %s\n", msg.UserID, msg.EmailID)

	c.JSON(http.StatusOK, gin.H{"status": "success", "message": "Follow-up processed"})
}

// HandleScheduledTask processes scheduled task webhook messages
func (h *WebhookHandler) HandleScheduledTask(c *gin.Context) {
	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Failed to read request body"})
		return
	}

	// Verify QStash signature
	if err := h.verifyQStashSignature(c, body); err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid signature"})
		return
	}

	var msg queue.ScheduledTaskMessage
	if err := json.Unmarshal(body, &msg); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid message format"})
		return
	}

	// Process scheduled task based on task type
	switch msg.TaskType {
	case "email_cleanup":
		// Handle email cleanup task
		if err := h.processEmailCleanupTask(c.Request.Context(), msg); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to process email cleanup task"})
			return
		}
	case "user_analytics":
		// Handle user analytics task
		if err := h.processUserAnalyticsTask(c.Request.Context(), msg); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to process user analytics task"})
			return
		}
	default:
		c.JSON(http.StatusBadRequest, gin.H{"error": "Unknown task type"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "success", "message": "Scheduled task processed"})
}

// processEmailCleanupTask handles email cleanup scheduled tasks
func (h *WebhookHandler) processEmailCleanupTask(ctx context.Context, msg queue.ScheduledTaskMessage) error {
	// Implement email cleanup logic
	// This could involve deleting old emails, archiving, etc.
	fmt.Printf("Processing email cleanup task for user: %s\n", msg.UserID)
	return nil
}

// processUserAnalyticsTask handles user analytics scheduled tasks
func (h *WebhookHandler) processUserAnalyticsTask(ctx context.Context, msg queue.ScheduledTaskMessage) error {
	// Implement user analytics logic
	// This could involve generating reports, updating metrics, etc.
	fmt.Printf("Processing user analytics task for user: %s\n", msg.UserID)
	return nil
}