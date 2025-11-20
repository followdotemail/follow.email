package handlers

import (
	// "encoding/json"
	"fmt"
	"net/http"
	// "os"
	// "path/filepath"
	"strconv"
	// "strings"
	"time"

	"follow-email-backend/internal/models"
	"follow-email-backend/internal/queue"
	"follow-email-backend/internal/services"
	"follow-email-backend/pkg/ai"
	emailutils "follow-email-backend/pkg/email"
	"follow-email-backend/pkg/oauth"
	"follow-email-backend/pkg/storage"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// NEED TO CHECK THIS CODE MANUALLY
// EmailHandler handles email-related HTTP requests
type EmailHandler struct {
	db                *gorm.DB
	emailSyncService  *services.EmailSyncService
	aiService         *ai.AIService
	qstashService     *queue.QStashService
	storageService    *storage.S3Service
	gmailTokenService *services.GmailTokenService
	gmailSyncService  *services.GmailSyncService
}

// NEED TO CHECK THIS CODE MANUALLY
// NewEmailHandler creates a new email handler
func NewEmailHandler(
	db *gorm.DB,
	emailSyncService *services.EmailSyncService,
	aiService *ai.AIService,
	qstashService *queue.QStashService,
	storageService *storage.S3Service,
	gmailTokenService *services.GmailTokenService,
	gmailSyncService *services.GmailSyncService,
) *EmailHandler {
	return &EmailHandler{
		db:                db,
		emailSyncService:  emailSyncService,
		aiService:         aiService,
		qstashService:     qstashService,
		storageService:    storageService,
		gmailTokenService: gmailTokenService,
		gmailSyncService:  gmailSyncService,
	}
}

// NEED TO CHECK THIS CODE MANUALLY
// Helper function to get database user UUID from Clerk ID
func (h *EmailHandler) getUserUUIDFromClerkID(clerkID string) (uuid.UUID, error) {
	var user models.User
	if err := h.db.Select("id").Where("clerk_id = ?", clerkID).First(&user).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return uuid.Nil, fmt.Errorf("user not found for clerk_id: %s", clerkID)
		}
		return uuid.Nil, fmt.Errorf("database error: %w", err)
	}
	return user.ID, nil
}

// NEED TO CHECK THIS CODE MANUALLY
// SyncEmailsRequest represents the request body for email synchronization
type SyncEmailsRequest struct {
	// Email provider (google, microsoft)
	// example: google
	Provider string `json:"provider" binding:"required"`

	// Last sync timestamp for incremental sync
	// example: 2023-01-01T00:00:00Z
	LastSyncTime *time.Time `json:"last_sync_time,omitempty"`

	// Gmail history ID for incremental sync
	// example: 12345
	HistoryID string `json:"history_id,omitempty"`

	// Microsoft delta token for incremental sync
	// example: abc123
	DeltaToken string `json:"delta_token,omitempty"`
}

// NEED TO CHECK THIS CODE MANUALLY
// AnalyzeEmailRequest represents the request body for email analysis
type AnalyzeEmailRequest struct {
	// Email subject
	// example: Meeting Request
	Subject string `json:"subject" binding:"required"`

	// Sender email address
	// example: sender@example.com
	FromEmail string `json:"from_email" binding:"required"`

	// Email body content
	// example: I would like to schedule a meeting...
	Body string `json:"body" binding:"required"`
}

// NEED TO CHECK THIS CODE MANUALLY
// GenerateResponseRequest represents the request body for response generation
type GenerateResponseRequest struct {
	// Original email subject
	// example: Meeting Request
	OriginalSubject string `json:"original_subject" binding:"required"`

	// Original email body
	// example: I would like to schedule a meeting...
	OriginalBody string `json:"original_body" binding:"required"`

	// Sender email address
	// example: sender@example.com
	FromEmail string `json:"from_email" binding:"required"`

	// Additional context for response generation
	// example: I am available on weekdays after 2 PM
	UserContext string `json:"user_context,omitempty"`
}

// NEED TO CHECK THIS CODE MANUALLY
// ProcessEmailContentRequest represents a request to sanitize raw email HTML for rendering.
type ProcessEmailContentRequest struct {
	HTML             string `json:"html" binding:"required"`
	ShouldLoadImages bool   `json:"should_load_images"`
	Theme            string `json:"theme"`
}

// NEED TO CHECK THIS CODE MANUALLY
// ProcessEmailContentResponse represents the sanitized HTML response.
type ProcessEmailContentResponse struct {
	ProcessedHTML    string `json:"processed_html"`
	HasBlockedImages bool   `json:"has_blocked_images"`
}

// NEED TO CHECK THIS CODE MANUALLY
// SyncEmails handles email synchronization requests
func (h *EmailHandler) SyncEmails(c *gin.Context) {
	clerkID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	// Convert Clerk ID to database UUID
	userID, err := h.getUserUUIDFromClerkID(clerkID.(string))
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not found"})
		return
	}

	var req SyncEmailsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Validate provider
	if req.Provider != "google" && req.Provider != "microsoft" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid provider. Supported providers: google, microsoft"})
		return
	}

	// Validate that user has connected their account for the provider
	switch req.Provider {
		case "google":
			_, err := h.gmailTokenService.GetValidToken(c.Request.Context(), userID)
			if err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": "No valid Gmail token found. Please connect your Gmail account first."})
				return
			}
		case "microsoft":
			// TODO: Implement Microsoft token validation
			c.JSON(http.StatusNotImplemented, gin.H{"error": "Microsoft provider not yet implemented"})
			return
		default:
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid provider. Supported providers: google, microsoft"})
			return
	}

	// Queue the email sync job using QStash
	syncMessage := queue.EmailSyncMessage{
		UserID:    userID.String(),
		MessageID: "", // Optional message ID for incremental sync
	}

	err = h.qstashService.PublishEmailSync(c.Request.Context(), syncMessage)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to queue email sync: " + err.Error()})
		return
	}

	// Generate a job ID for tracking
	jobID := uuid.New().String()

	c.JSON(http.StatusAccepted, gin.H{
		"message":        "Email sync job queued successfully",
		"job_id":         jobID,
		"provider":       req.Provider,
		"user_id":        userID,
		"estimated_time": "5-10 minutes",
		"status":         "queued",
	})
}

// NEED TO CHECK THIS CODE MANUALLY
// GetSyncStatus handles sync status requests
func (h *EmailHandler) GetSyncStatus(c *gin.Context) {
	clerkID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	// Convert Clerk ID to database UUID
	userID, err := h.getUserUUIDFromClerkID(clerkID.(string))
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not found"})
		return
	}

	provider := c.Query("provider")
	if provider == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Provider parameter is required"})
		return
	}

	// Get sync status from service
	status, err := h.emailSyncService.GetSyncStatus(c.Request.Context(), userID, oauth.Provider(provider))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get sync status"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"user_id":          userID,
		"provider":         provider,
		"emails_processed": status.EmailsProcessed,
		"new_emails":       status.NewEmails,
		"updated_emails":   status.UpdatedEmails,
		"last_sync_at":     status.LastSyncAt,
		"errors":           status.Errors,
	})
}

// NEED TO CHECK THIS CODE MANUALLY
// AnalyzeEmail handles email analysis requests
func (h *EmailHandler) AnalyzeEmail(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	var req AnalyzeEmailRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Analyze email using AI service
	analysis, err := h.aiService.AnalyzeEmail(
		c.Request.Context(),
		req.Subject,
		req.Body,
		req.FromEmail,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to analyze email"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"analysis": analysis,
		"user_id":  userID,
	})
}

// NEED TO CHECK THIS CODE MANUALLY
// GenerateResponse handles response generation requests
func (h *EmailHandler) GenerateResponse(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	var req GenerateResponseRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Generate response using AI service
	response, err := h.aiService.GenerateResponse(
		c.Request.Context(),
		req.OriginalSubject,
		req.OriginalBody,
		req.FromEmail,
		req.UserContext,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate response"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"response": response,
		"user_id":  userID,
	})
}

// NEED TO CHECK THIS CODE MANUALLY
// ScheduleFollowUp handles follow-up scheduling requests
func (h *EmailHandler) ScheduleFollowUp(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	emailIDStr := c.Param("emailId")
	emailID, err := strconv.Atoi(emailIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid email ID"})
		return
	}

	// Parse request body for follow-up details
	var followUpReq struct {
		ScheduledFor time.Time `json:"scheduled_for" binding:"required"`
		TemplateID   int       `json:"template_id,omitempty"`
	}

	if err := c.ShouldBindJSON(&followUpReq); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Queue the follow-up job
	queueMsg := queue.FollowUpMessage{
		UserID:  userID.(string),
		EmailID: strconv.Itoa(emailID),
		Content: "Follow-up scheduled", // Default content since it's not in the request
	}

	if err := h.qstashService.PublishFollowUp(c.Request.Context(), queueMsg); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to schedule follow-up"})
		return
	}

	c.JSON(http.StatusAccepted, gin.H{
		"message":       "Follow-up scheduled successfully",
		"email_id":      emailID,
		"scheduled_for": followUpReq.ScheduledFor,
	})
}

// NEED TO CHECK THIS CODE MANUALLY
// ProcessEmailContent sanitizes Gmail HTML based on client preferences.
func (h *EmailHandler) ProcessEmailContent(c *gin.Context) {
	var req ProcessEmailContentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	result, err := emailutils.ProcessEmailHTML(req.HTML, emailutils.EmailRenderOptions{
		ShouldLoadImages: req.ShouldLoadImages,
		Theme:            req.Theme,
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to process email content"})
		return
	}

	c.JSON(http.StatusOK, ProcessEmailContentResponse{
		ProcessedHTML:    result.HTML,
		HasBlockedImages: result.HasBlockedImages,
	})
}

// EmailQueryRequest represents the request parameters for querying emails
type EmailQueryRequest struct {
	// Page number for pagination (starts from 1)
	// example: 1
	Page int `form:"page" json:"page"`

	// Number of emails per page (max 100)
	// example: 20
	Limit int `form:"limit" json:"limit"`

	// Filter by sender email
	// example: sender@example.com
	FromEmail string `form:"from_email" json:"from_email"`

	// Filter by subject (partial match)
	// example: meeting
	Subject string `form:"subject" json:"subject"`

	// Filter by date range - start date
	// example: 2023-01-01T00:00:00Z
	StartDate *time.Time `form:"start_date" json:"start_date"`

	// Filter by date range - end date
	// example: 2023-12-31T23:59:59Z
	EndDate *time.Time `form:"end_date" json:"end_date"`

	// Filter by read status
	// example: false
	IsRead *bool `form:"is_read" json:"is_read"`

	// Filter by importance
	// example: true
	IsImportant *bool `form:"is_important" json:"is_important"`

	// Filter by attachment presence
	// example: true
	HasAttachments *bool `form:"has_attachments" json:"has_attachments"`

	// Filter by AI sentiment
	// example: positive
	AISentiment string `form:"ai_sentiment" json:"ai_sentiment"`

	// Filter by AI priority
	// example: high
	AIPriority string `form:"ai_priority" json:"ai_priority"`

	// Filter by follow-up requirement
	// example: true
	RequiresFollowUp *bool `form:"requires_followup" json:"requires_followup"`

	// Filter by follow-up status
	// example: pending
	FollowUpStatus string `form:"followup_status" json:"followup_status"`

	// Sort field (sent_at, received_at, subject, from_email)
	// example: received_at
	SortBy string `form:"sort_by" json:"sort_by"`

	// Sort order (asc, desc)
	// example: desc
	SortOrder string `form:"sort_order" json:"sort_order"`
}

// FilteredEmail represents a filtered email response with only required fields
type FilteredEmail struct {
	ID             uuid.UUID  `json:"id"`
	ClerkID        string     `json:"clerk_id"`
	MessageID      string     `json:"message_id"`
	ThreadID       string     `json:"thread_id"`
	Subject        string     `json:"subject"`
	FromEmail      string     `json:"from_email"`
	FromName       string     `json:"from_name"`
	ToEmails       string     `json:"to_emails"`
	CCEmails       string     `json:"cc_emails"`
	BCCEmails      string     `json:"bcc_emails"`
	UpdatedAt      time.Time  `json:"updated_at"`
	IsRead         bool       `json:"is_read"`
	IsImportant    bool       `json:"is_important"`
	HasAttachments bool       `json:"has_attachments"`
	Labels         string     `json:"labels"`
	LastSyncAt     *time.Time `json:"last_sync_at"`
}

// EmailDetailMetadata represents detailed metadata for a single email
type EmailDetailMetadata struct {
	ID               uuid.UUID  `json:"id"`
	MessageID        string     `json:"message_id"`
	ThreadID         string     `json:"thread_id"`
	Subject          string     `json:"subject"`
	FromEmail        string     `json:"from_email"`
	FromName         string     `json:"from_name"`
	ToEmails         string     `json:"to_emails"`
	CCEmails         string     `json:"cc_emails"`
	BCCEmails        string     `json:"bcc_emails"`
	SentAt           time.Time  `json:"sent_at"`
	ReceivedAt       time.Time  `json:"received_at"`
	CreatedAt        time.Time  `json:"created_at"`
	UpdatedAt        time.Time  `json:"updated_at"`
	IsRead           bool       `json:"is_read"`
	IsImportant      bool       `json:"is_important"`
	HasAttachments   bool       `json:"has_attachments"`
	Labels           string     `json:"labels"`
	AISummary        string     `json:"ai_summary"`
	AISentiment      string     `json:"ai_sentiment"`
	AIPriority       string     `json:"ai_priority"`
	RequiresFollowUp bool       `json:"requires_followup"`
	FollowUpStatus   string     `json:"followup_status"`
	LastFollowUpAt   *time.Time `json:"last_followup_at"`
	FollowUpCount    int        `json:"followup_count"`
}

// EmailAttachmentResponse represents attachment metadata returned to clients
type EmailAttachmentResponse struct {
	Filename     string `json:"filename"`
	ContentType  string `json:"content_type"`
	Size         int64  `json:"size"`
	AttachmentID string `json:"attachment_id"`
	DownloadURL  string `json:"download_url"`
}

// EmailDetailResponse represents a combined email metadata + content payload
type EmailDetailResponse struct {
	Email       EmailDetailMetadata       `json:"email"`
	Body        string                    `json:"body"`      // Deprecated: use BodyHTML instead
	Attachments []EmailAttachmentResponse `json:"attachments"`
	Source      string                    `json:"source"`
}

// EmailQueryResponse represents the response for email queries
type EmailQueryResponse struct {
	// List of emails
	Emails []FilteredEmail `json:"emails"`

	// Pagination information
	Pagination PaginationInfo `json:"pagination"`
}

// PaginationInfo represents pagination metadata
type PaginationInfo struct {
	// Current page number
	Page int `json:"page"`

	// Number of items per page
	Limit int `json:"limit"`

	// Total number of items
	Total int64 `json:"total"`

	// Total number of pages
	TotalPages int `json:"total_pages"`

	// Whether there is a next page
	HasNext bool `json:"has_next"`

	// Whether there is a previous page
	HasPrev bool `json:"has_prev"`
}

// NEED TO CHECK THIS CODE MANUALLY
// GetEmails handles email query requests with filtering and pagination
func (h *EmailHandler) GetEmails(c *gin.Context) {
	// Get user ID from context (set by auth middleware)
	clerkID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	// Convert Clerk ID to database UUID
	userID, err := h.getUserUUIDFromClerkID(clerkID.(string))
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not found"})
		return
	}

	// Parse query parameters
	var req EmailQueryRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid query parameters: " + err.Error()})
		return
	}

	// Set default values
	if req.Page <= 0 {
		req.Page = 1
	}
	if req.Limit <= 0 {
		req.Limit = 20
	}
	if req.Limit > 100 {
		req.Limit = 100
	}
	if req.SortBy == "" {
		req.SortBy = "received_at"
	}
	if req.SortOrder == "" {
		req.SortOrder = "desc"
	}

	// Validate sort fields
	validSortFields := map[string]bool{
		"sent_at":     true,
		"received_at": true,
		"subject":     true,
		"from_email":  true,
		"created_at":  true,
	}
	if !validSortFields[req.SortBy] {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid sort field"})
		return
	}

	if req.SortOrder != "asc" && req.SortOrder != "desc" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid sort order. Use 'asc' or 'desc'"})
		return
	}

	// Build query
	query := h.db.Where("user_id = ?", userID)

	// Apply filters
	if req.FromEmail != "" {
		query = query.Where("from_email ILIKE ?", "%"+req.FromEmail+"%")
	}
	if req.Subject != "" {
		query = query.Where("subject ILIKE ?", "%"+req.Subject+"%")
	}
	if req.StartDate != nil {
		query = query.Where("received_at >= ?", *req.StartDate)
	}
	if req.EndDate != nil {
		query = query.Where("received_at <= ?", *req.EndDate)
	}
	if req.IsRead != nil {
		query = query.Where("is_read = ?", *req.IsRead)
	}
	if req.IsImportant != nil {
		query = query.Where("is_important = ?", *req.IsImportant)
	}
	if req.HasAttachments != nil {
		query = query.Where("has_attachments = ?", *req.HasAttachments)
	}
	if req.AISentiment != "" {
		query = query.Where("ai_sentiment = ?", req.AISentiment)
	}
	if req.AIPriority != "" {
		query = query.Where("ai_priority = ?", req.AIPriority)
	}
	if req.RequiresFollowUp != nil {
		query = query.Where("requires_followup = ?", *req.RequiresFollowUp)
	}
	if req.FollowUpStatus != "" {
		query = query.Where("followup_status = ?", req.FollowUpStatus)
	}

	// Get total count for pagination
	var total int64
	if err := query.Model(&models.Email{}).Count(&total).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to count emails"})
		return
	}

	// Apply sorting and pagination
	offset := (req.Page - 1) * req.Limit
	query = query.Order(req.SortBy + " " + req.SortOrder).Offset(offset).Limit(req.Limit)

	// Execute query
	var emails []models.Email
	if err := query.Find(&emails).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to query emails"})
		return
	}

	// Get user's clerk_id for the response
	var user models.User
	if err := h.db.Where("id = ?", userID).First(&user).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get user information"})
		return
	}

	// Convert to filtered email response
	filteredEmails := make([]FilteredEmail, len(emails))
	for i, email := range emails {
		clerkID := ""
		if user.ClerkID != nil {
			clerkID = *user.ClerkID
		}

		filteredEmails[i] = FilteredEmail{
			ID:             email.ID,
			ClerkID:        clerkID,
			MessageID:      email.MessageID,
			ThreadID:       email.ThreadID,
			Subject:        email.Subject,
			FromEmail:      email.FromEmail,
			FromName:       email.FromName,
			ToEmails:       email.ToEmails,
			CCEmails:       email.CcEmails,
			BCCEmails:      email.BccEmails,
			UpdatedAt:      email.UpdatedAt,
			IsRead:         email.IsRead,
			IsImportant:    email.IsImportant,
			HasAttachments: email.HasAttachments,
			Labels:         email.Labels,
			LastSyncAt:     &email.LastSyncAt,
		}
	}

	// Calculate pagination info
	totalPages := int((total + int64(req.Limit) - 1) / int64(req.Limit))
	hasNext := req.Page < totalPages
	hasPrev := req.Page > 1

	response := EmailQueryResponse{
		Emails: filteredEmails,
		Pagination: PaginationInfo{
			Page:       req.Page,
			Limit:      req.Limit,
			Total:      total,
			TotalPages: totalPages,
			HasNext:    hasNext,
			HasPrev:    hasPrev,
		},
	}

	c.JSON(http.StatusOK, response)
}

// MANUALLY CHECKED ✅
// GetEmailByID returns email metadata together with live content and attachments
func (h *EmailHandler) GetEmailByID(c *gin.Context) {
	// Get user ID from context (set by auth middleware)
	clerkID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	// Convert Clerk ID to database UUID
	userID, err := h.getUserUUIDFromClerkID(clerkID.(string))
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not found"})
		return
	}

	// Parse email ID from URL parameter
	emailIDStr := c.Param("id")
	emailID, err := uuid.Parse(emailIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid email ID format"})
		return
	}

	// Query email by ID and user ID - select only required columns
	var email models.Email
	if err := h.db.Select(
		"id, user_id, message_id, thread_id, subject, from_email, from_name, to_emails, cc_emails, bcc_emails, sent_at, received_at, created_at, updated_at, "+
			"is_read, is_important, has_attachments, labels, ai_summary, ai_sentiment, a_ipriority, requires_follow_up, follow_up_status, last_follow_up_at, follow_up_count").
		Where("id = ? AND user_id = ?", emailID, userID).
		First(&email).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "Email not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve email"})
		return
	}

	metadata := EmailDetailMetadata{
		ID:               email.ID,
		MessageID:        email.MessageID,
		ThreadID:         email.ThreadID,
		Subject:          email.Subject,
		FromEmail:        email.FromEmail,
		FromName:         email.FromName,
		ToEmails:         email.ToEmails,
		CCEmails:         email.CcEmails,
		BCCEmails:        email.BccEmails,
		SentAt:           email.SentAt,
		ReceivedAt:       email.ReceivedAt,
		CreatedAt:        email.CreatedAt,
		UpdatedAt:        email.UpdatedAt,
		IsRead:           email.IsRead,
		IsImportant:      email.IsImportant,
		HasAttachments:   email.HasAttachments,
		Labels:           email.Labels,
		AISummary:        email.AISummary,
		AISentiment:      email.AISentiment,
		AIPriority:       email.AIPriority,
		RequiresFollowUp: email.RequiresFollowUp,
		FollowUpStatus:   email.FollowUpStatus,
		LastFollowUpAt:   email.LastFollowUpAt,
		FollowUpCount:    email.FollowUpCount,
	}

	ctx := c.Request.Context()

	// Debug logging
	DebugTextPrint(fmt.Sprintf("GetEmailByID - Email ID: %s, MessageID: %s, UserID: %s", email.ID, email.MessageID, userID))
	DebugTextPrint(fmt.Sprintf("gmailSyncService is nil: %v", h.gmailSyncService == nil))

	// Check prerequisites
	if email.MessageID == "" {
		DebugErrorTextPrint(fmt.Sprintf("MessageID is empty for email %s", email.ID))
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Email MessageID not found. This email may not be synced from Gmail.",
		})
		return
	}

	if h.gmailSyncService == nil {
		DebugErrorTextPrint("Gmail sync service not initialized")
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"error": "Gmail service not available",
		})
		return
	}

	// Fetch full Gmail message from API
	DebugTextPrint(fmt.Sprintf("Fetching full Gmail message for MessageID: %s", email.MessageID))
	// fullMessage, err := h.gmailSyncService.GetFullGmailMessage(ctx, userID, email.MessageID)
	// if err != nil {
	// 	log.Printf("[ERROR] Failed to fetch full Gmail message: %v", err)
	// 	c.JSON(http.StatusInternalServerError, gin.H{
	// 		"error": fmt.Sprintf("Failed to fetch email from Gmail: %v", err),
	// 	})
	// 	return
	// }

	// Fetch email body from Gmail API only
	DebugTextPrint(fmt.Sprintf("Fetching email body from Gmail for MessageID: %s", email.MessageID))
	content, err := h.gmailSyncService.GetEmailContentFromGmail(ctx, userID, email.MessageID)
	if err != nil {
		DebugErrorTextPrint(fmt.Sprintf("Failed to fetch email body from Gmail: %v", err))
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": fmt.Sprintf("Failed to fetch email content from Gmail: %v", err),
		})
		return
	}
	bodyHTML := content.HTML

	// NEED TO REMOVE THIS PART OF CODE BEFORE DEPLOYING TO PRODUCTION
	// Save full Gmail response to JSON file 
	// saveDir := filepath.Join("../..", "gmail_response")
	// if err := os.MkdirAll(saveDir, 0755); err != nil {
	// 	log.Printf("[WARN] Failed to create gmail_response directory: %v", err)
	// } else {
	// 	jsonPath := filepath.Join(saveDir, fmt.Sprintf("%s_%s.json", emailID.String(), strings.ReplaceAll(email.Subject, " ", "_")))
	// 	htmlPath := filepath.Join(saveDir, fmt.Sprintf("%s_%s.html", emailID.String(), strings.ReplaceAll(email.Subject, " ", "_")))
		
	// 	if err := os.WriteFile(htmlPath, []byte(bodyHTML), 0644); err != nil {
	// 		log.Printf("[WARN] Failed to save Gmail response to %s: %v", htmlPath, err)
	// 	} else {
	// 		log.Printf("[INFO] Saved Gmail response to %s", htmlPath)
	// 	}
		
	// 	jsonData, err := json.MarshalIndent(fullMessage, "", "  ")
	// 	if err != nil {
	// 		log.Printf("[WARN] Failed to marshal Gmail response to JSON: %v", err)
	// 	} else {
	// 		if err := os.WriteFile(jsonPath, jsonData, 0644); err != nil {
	// 			log.Printf("[WARN] Failed to save Gmail response to %s: %v", jsonPath, err)
	// 		} else {
	// 			log.Printf("[INFO] Saved Gmail response to %s", jsonPath)
	// 		}
	// 	}
	// }

	// Fetch attachments from Gmail API only
	DebugTextPrint(fmt.Sprintf("Fetching attachments from Gmail for MessageID: %s", email.MessageID))
	attachments, err := h.gmailSyncService.GetEmailAttachmentsFromGmail(ctx, userID, email.MessageID)
	if err != nil {
		DebugErrorTextPrint(fmt.Sprintf("Failed to fetch attachments from Gmail: %v", err))
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": fmt.Sprintf("Failed to fetch attachments from Gmail: %v", err),
		})
		return
	}
	DebugSuccessTextPrint(fmt.Sprintf("Fetched %d attachments from Gmail", len(attachments)))

	responseAttachments := make([]EmailAttachmentResponse, 0, len(attachments))
	for _, att := range attachments {
		responseAttachments = append(responseAttachments, EmailAttachmentResponse{
			Filename:     att.Filename,
			ContentType:  att.MimeType,
			Size:         att.Size,
			AttachmentID: att.AttachmentID,
			DownloadURL:  fmt.Sprintf("/api/v1/emails/%s/attachments/%s", email.ID, att.AttachmentID),
		})
	}

	response := EmailDetailResponse{
		Email:       metadata,
		Body:        bodyHTML,
		Attachments: responseAttachments,
		Source:      "gmail",
	}

	c.JSON(http.StatusOK, response)
}

// DownloadAttachment handles attachment download requests
func (h *EmailHandler) DownloadAttachment(c *gin.Context) {
	// Get user ID from context (set by auth middleware)
	clerkID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	// Convert Clerk ID to database UUID
	userID, err := h.getUserUUIDFromClerkID(clerkID.(string))
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not found"})
		return
	}

	// Parse email ID and attachment ID from URL parameters
	emailIDStr := c.Param("id")
	emailID, err := uuid.Parse(emailIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid email ID format"})
		return
	}

	attachmentID := c.Param("attachment_id")
	if attachmentID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Attachment ID is required"})
		return
	}

	// Verify email belongs to user
	var email models.Email
	if err := h.db.Where("id = ? AND user_id = ?", emailID, userID).First(&email).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "Email not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve email"})
		return
	}

	// Download attachment from Gmail
	if email.MessageID != "" && h.gmailSyncService != nil {
		ctx := c.Request.Context()
		data, mimeType, err := h.gmailSyncService.DownloadAttachmentFromGmail(ctx, userID, email.MessageID, attachmentID)
		if err != nil {
			DebugErrorTextPrint(fmt.Sprintf("Failed to download attachment from Gmail: %v", err))
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to download attachment"})
			return
		}

		// Get attachment filename from metadata
		attachments, _ := h.gmailSyncService.GetEmailAttachmentsFromGmail(ctx, userID, email.MessageID)
		filename := "attachment"
		for _, att := range attachments {
			if att.AttachmentID == attachmentID {
				filename = att.Filename
				break
			}
		}

		// Set headers and return file
		c.Header("Content-Type", mimeType)
		c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", filename))
		c.Header("Content-Length", fmt.Sprintf("%d", len(data)))
		c.Data(http.StatusOK, mimeType, data)
		return
	}

	c.JSON(http.StatusNotImplemented, gin.H{"error": "Attachment download not available"})
}

// SendEmailRequest represents a request to send an email
type SendEmailRequest struct {
	// Recipient email addresses (required)
	// example: ["recipient@example.com"]
	To []string `json:"to" binding:"required,min=1,dive,email"`

	// CC email addresses (optional)
	// example: ["cc@example.com"]
	Cc []string `json:"cc" binding:"omitempty,dive,email"`

	// BCC email addresses (optional)
	// example: ["bcc@example.com"]
	Bcc []string `json:"bcc" binding:"omitempty,dive,email"`

	// Email subject (required)
	// example: Meeting Follow-up
	Subject string `json:"subject" binding:"required"`

	// Email body in plain text format
	// example: Hi, just following up on our meeting...
	BodyText string `json:"body_text"`

	// Email body in HTML format (optional)
	// example: <html><body>Hi, just following up on our meeting...</body></html>
	BodyHTML string `json:"body_html"`

	// Thread ID for replying to an existing thread (optional)
	// example: 18a1b2c3d4e5f6g7
	ThreadID string `json:"thread_id"`

	// In-Reply-To header for replying (optional)
	// example: <message-id@example.com>
	InReplyTo string `json:"in_reply_to"`

	// References header for threading (optional)
	// example: <message-id-1@example.com> <message-id-2@example.com>
	References string `json:"references"`

	// Reply-To header (optional)
	// example: noreply@example.com
	ReplyTo string `json:"reply_to" binding:"omitempty,email"`
}

// SendEmail handles sending email through Gmail API
func (h *EmailHandler) SendEmail(c *gin.Context) {
	// Get user ID from context (set by auth middleware)
	clerkID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	// Convert Clerk ID to database UUID
	userID, err := h.getUserUUIDFromClerkID(clerkID.(string))
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not found"})
		return
	}

	// Parse and validate request
	var req SendEmailRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Validate that at least one body format is provided
	if req.BodyText == "" && req.BodyHTML == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Either body_text or body_html must be provided"})
		return
	}

	// Check if Gmail sync service is available
	if h.gmailSyncService == nil {
		DebugErrorTextPrint("Gmail sync service not initialized")
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"error": "Gmail service not available",
		})
		return
	}

	ctx := c.Request.Context()

	// Validate recipient email addresses
	if len(req.To) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "At least one recipient email address is required"})
		return
	}

	// Validate subject length (Gmail has limits)
	if len(req.Subject) > 998 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Subject line is too long (maximum 998 characters)"})
		return
	}

	// Validate thread_id format if provided (Gmail thread IDs are alphanumeric, 10+ chars)
	if req.ThreadID != "" {
		if !isValidGmailThreadIDFormat(req.ThreadID) {
			DebugWarningTextPrint(fmt.Sprintf("Invalid thread_id format provided: '%s'. Will create new thread.", req.ThreadID))
			// Clear invalid thread_id - let Gmail create a new thread
			req.ThreadID = ""
		}
	}

	// Convert handler request to service request
	serviceReq := &services.SendEmailRequest{
		To:         req.To,
		Cc:         req.Cc,
		Bcc:        req.Bcc,
		Subject:    req.Subject,
		BodyText:   req.BodyText,
		BodyHTML:   req.BodyHTML,
		ThreadID:   req.ThreadID,
		InReplyTo:  req.InReplyTo,
		References: req.References,
		ReplyTo:    req.ReplyTo,
	}

	// Send email through Gmail API
	DebugTextPrint(fmt.Sprintf("Sending email for user %s to %v", userID, req.To))
	messageID, threadID, err := h.gmailSyncService.SendEmail(ctx, userID, serviceReq)
	if err != nil {
		DebugErrorTextPrint(fmt.Sprintf("Failed to send email: %v", err))
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": fmt.Sprintf("Failed to send email: %v", err),
		})
		return
	}

	DebugSuccessTextPrint(fmt.Sprintf("Email sent successfully. MessageID: %s, ThreadID: %s", messageID, threadID))

	// Return success response
	c.JSON(http.StatusOK, gin.H{
		"message":    "Email sent successfully",
		"message_id": messageID,
		"thread_id":  threadID,
		"to":         req.To,
		"subject":    req.Subject,
		"sent_at":    time.Now().Format(time.RFC3339),
	})
}

// isValidGmailThreadIDFormat validates if a string matches Gmail thread ID format
// Gmail thread IDs are alphanumeric strings, typically 16+ characters
func isValidGmailThreadIDFormat(threadID string) bool {
	if threadID == "" {
		return false
	}

	// Gmail thread IDs are alphanumeric (letters and numbers only)
	for _, char := range threadID {
		if !((char >= 'a' && char <= 'z') || (char >= 'A' && char <= 'Z') || (char >= '0' && char <= '9')) {
			return false
		}
	}

	// Gmail thread IDs are typically at least 16 characters, but we allow 10+ for flexibility
	if len(threadID) < 10 {
		return false
	}

	return true
}
