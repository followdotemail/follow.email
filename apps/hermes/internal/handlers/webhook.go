package handlers

import (
	"context"
	"crypto/hmac"
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
		fmt.Println("DEBUG: Missing Upstash-Signature header")
		return fmt.Errorf("missing Upstash-Signature header")
	}

	fmt.Printf("DEBUG: Received signature: %s\n", signature)
	fmt.Printf("DEBUG: Body length: %d bytes\n", len(body))

	// QStash now sends JWT tokens directly (new format)
	// Check if it's a JWT (contains two dots)
	if strings.Count(signature, ".") == 2 {
		fmt.Println("DEBUG: Detected JWT signature format")
		// This is a JWT token - QStash's new signature format
		// For JWT verification, we just verify the signature using HMAC
		// The JWT payload contains a hash of the body

		// Try to verify with current key
		if h.verifyJWTSignature(signature, h.config.QStashCurrentSigningKey) {
			fmt.Println("DEBUG: JWT signature verified with current key")
			return nil
		}

		// Try next signing key
		if h.config.QStashNextSigningKey != "" && h.verifyJWTSignature(signature, h.config.QStashNextSigningKey) {
			fmt.Println("DEBUG: JWT signature verified with next key")
			return nil
		}

		fmt.Println("DEBUG: JWT signature verification failed with both keys")
		return fmt.Errorf("JWT signature verification failed")
	}

	// Old v1 format: "v1=<signature>"
	fmt.Println("DEBUG: Using v1 signature format")
	parts := strings.Split(signature, "=")
	if len(parts) != 2 || parts[0] != "v1" {
		fmt.Printf("DEBUG: Invalid v1 signature format. Got %d parts\n", len(parts))
		return fmt.Errorf("invalid signature format")
	}

	expectedSignature, err := base64.StdEncoding.DecodeString(parts[1])
	if err != nil {
		fmt.Printf("DEBUG: Failed to decode signature: %v\n", err)
		return fmt.Errorf("invalid signature encoding: %w", err)
	}

	// Try current signing key first
	if h.verifySignatureWithKey(body, expectedSignature, h.config.QStashCurrentSigningKey) {
		fmt.Println("DEBUG: v1 signature verified with current key")
		return nil
	}

	// Try next signing key (for key rotation)
	if h.config.QStashNextSigningKey != "" && h.verifySignatureWithKey(body, expectedSignature, h.config.QStashNextSigningKey) {
		fmt.Println("DEBUG: v1 signature verified with next key")
		return nil
	}

	fmt.Println("DEBUG: v1 signature verification failed with both keys")
	return fmt.Errorf("signature verification failed")
}

// verifyJWTSignature verifies a QStash JWT signature
func (h *WebhookHandler) verifyJWTSignature(token string, signingKey string) bool {
	// Split JWT into parts
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return false
	}

	// JWT format: header.payload.signature
	message := parts[0] + "." + parts[1]
	signature := parts[2]

	// Decode the signature from base64 URL encoding
	expectedSig, err := base64.RawURLEncoding.DecodeString(signature)
	if err != nil {
		fmt.Printf("DEBUG: Failed to decode JWT signature: %v\n", err)
		return false
	}

	// Compute HMAC using the signing key
	mac := hmac.New(sha256.New, []byte(signingKey))
	mac.Write([]byte(message))
	computedSig := mac.Sum(nil)

	// Compare signatures
	return subtle.ConstantTimeCompare(expectedSig, computedSig) == 1
}

// verifySignatureWithKey verifies signature with a specific key
func (h *WebhookHandler) verifySignatureWithKey(body, expectedSignature []byte, key string) bool {
	mac := hmac.New(sha256.New, []byte(key))
	// mac.Write([]byte(key))
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
		UserID:   userID,
		FullSync: msg.SyncType == "full",
	}

	// Add this debug log after line 182:
	fmt.Printf("DEBUG: Received SyncType='%s', FullSync=%v\n", msg.SyncType, syncReq.FullSync)

	// Process email sync using the Gmail sync service
	fmt.Printf("Processing email sync for user: %s, message: %s\n", msg.UserID, msg.MessageID)

	result, err := h.gmailSyncService.SyncUserEmails(c.Request.Context(), syncReq)
	if err != nil {
		fmt.Printf("Email sync failed for user %s: %v\n", msg.UserID, err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"status":  "error",
			"message": "Email sync failed",
			"error":   err.Error(),
		})
		return
	}

	fmt.Printf("Email sync completed for user %s: %d emails processed, %d new, %d updated\n",
		msg.UserID, result.EmailsProcessed, result.NewEmails, result.UpdatedEmails)

	c.JSON(http.StatusOK, gin.H{
		"status":           "success",
		"message":          "Email sync processed",
		"emails_processed": result.EmailsProcessed,
		"new_emails":       result.NewEmails,
		"updated_emails":   result.UpdatedEmails,
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
