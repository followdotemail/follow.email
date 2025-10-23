package services

import (
	"context"
	"fmt"
	"log"
	"strconv"
	"time"

	"follow-email-backend/internal/models"
	"follow-email-backend/pkg/oauth" // Stub implementation for compilation

	"github.com/google/uuid"
	"google.golang.org/api/gmail/v1"
	"gorm.io/gorm"
)

type EmailSyncService struct {
	oauthService      *oauth.OAuthService
	gmailOAuthService *oauth.GmailOAuthService
	db                *gorm.DB
}

type SyncResult struct {
	EmailsProcessed int       `json:"emails_processed"`
	NewEmails       int       `json:"new_emails"`
	UpdatedEmails   int       `json:"updated_emails"`
	Errors          []string  `json:"errors"`
	LastSyncAt      time.Time `json:"last_sync_at"`
}

type EmailSyncRequest struct {
	UserID       uuid.UUID        `json:"user_id"`
	Provider     oauth.Provider   `json:"provider"`
	TokenInfo    *oauth.TokenInfo `json:"token_info"`
	LastSyncTime *time.Time       `json:"last_sync_time"`
	HistoryID    string           `json:"history_id,omitempty"`  // For Gmail incremental sync
	DeltaToken   string           `json:"delta_token,omitempty"` // For Microsoft Graph delta sync
}

// NewEmailSyncService creates a new email synchronization service
func NewEmailSyncService(oauthService *oauth.OAuthService, gmailOAuthService *oauth.GmailOAuthService, db *gorm.DB) *EmailSyncService {
	return &EmailSyncService{
		oauthService:      oauthService,
		gmailOAuthService: gmailOAuthService,
		db:                db,
	}
}

// SyncEmails performs email synchronization for a user
func (s *EmailSyncService) SyncEmails(ctx context.Context, req *EmailSyncRequest) (*SyncResult, error) {
	switch req.Provider {
	case oauth.ProviderGoogle:
		return s.syncGmailEmails(ctx, req)
	case oauth.ProviderMicrosoft:
		return s.syncMicrosoftEmails(ctx, req)
	default:
		return nil, fmt.Errorf("unsupported provider: %s", req.Provider)
	}
}

// syncGmailEmails handles Gmail email synchronization using history.list API
func (s *EmailSyncService) syncGmailEmails(ctx context.Context, req *EmailSyncRequest) (*SyncResult, error) {
	result := &SyncResult{
		LastSyncAt: time.Now(),
		Errors:     []string{},
	}

	// Convert TokenInfo to GmailTokenInfo
	gmailTokenInfo := &oauth.GmailTokenInfo{
		AccessToken:  req.TokenInfo.AccessToken,
		RefreshToken: req.TokenInfo.RefreshToken,
		TokenType:    req.TokenInfo.TokenType,
		ExpiresAt:    req.TokenInfo.ExpiresAt,
		Scope:        req.TokenInfo.Scope,
	}

	// Create Gmail service using the proper Gmail OAuth service
	gmailService, err := s.gmailOAuthService.CreateGmailService(ctx, gmailTokenInfo)
	if err != nil {
		return nil, fmt.Errorf("failed to create Gmail service: %w", err)
	}

	// Get user's Gmail profile
	profile, err := gmailService.Users.GetProfile("me").Do()
	if err != nil {
		return nil, fmt.Errorf("failed to get Gmail profile: %w", err)
	}

	log.Printf("Syncing emails for user %d, Gmail address: %s", req.UserID, profile.EmailAddress)

	// Perform incremental sync if we have a history ID
	if req.HistoryID != "" {
		return s.performGmailIncrementalSync(ctx, gmailService, req, result)
	}

	// Perform full sync for first-time sync
	return s.performGmailFullSync(ctx, gmailService, req, result)
}

// performGmailIncrementalSync uses Gmail's history.list API for efficient incremental sync
func (s *EmailSyncService) performGmailIncrementalSync(ctx context.Context, service *gmail.Service, req *EmailSyncRequest, result *SyncResult) (*SyncResult, error) {
	log.Printf("Performing Gmail incremental sync from history ID: %s", req.HistoryID)

	// Get history changes since last sync
	historyID, err := strconv.ParseUint(req.HistoryID, 10, 64)
	if err != nil {
		return nil, fmt.Errorf("invalid history ID: %w", err)
	}
	historyCall := service.Users.History.List("me").StartHistoryId(historyID)
	historyResponse, err := historyCall.Do()
	if err != nil {
		return nil, fmt.Errorf("failed to get Gmail history: %w", err)
	}

	// Process history changes
	for _, history := range historyResponse.History {
		// Process messages added
		for _, messageAdded := range history.MessagesAdded {
			if err := s.processGmailMessage(ctx, service, messageAdded.Message.Id, req.UserID); err != nil {
				result.Errors = append(result.Errors, fmt.Sprintf("Error processing added message %s: %v", messageAdded.Message.Id, err))
				continue
			}
			result.NewEmails++
		}

		// Process messages deleted
		for _, messageDeleted := range history.MessagesDeleted {
			// TODO: Mark message as deleted in database
			log.Printf("Message deleted: %s", messageDeleted.Message.Id)
		}

		// Process labels added/removed
		for _, labelAdded := range history.LabelsAdded {
			// TODO: Update message labels in database
			log.Printf("Labels added to message %s: %v", labelAdded.Message.Id, labelAdded.LabelIds)
		}
	}

	result.EmailsProcessed = len(historyResponse.History)
	return result, nil
}

// performGmailFullSync performs a full email sync (used for initial sync)
func (s *EmailSyncService) performGmailFullSync(ctx context.Context, service *gmail.Service, req *EmailSyncRequest, result *SyncResult) (*SyncResult, error) {
	log.Printf("Performing Gmail full sync for user %d", req.UserID)

	// Get list of messages (limit to recent messages for initial sync)
	messagesCall := service.Users.Messages.List("me").MaxResults(100) // Limit for demo
	if req.LastSyncTime != nil {
		// Only get messages after last sync time
		query := fmt.Sprintf("after:%d", req.LastSyncTime.Unix())
		messagesCall = messagesCall.Q(query)
	}

	messagesResponse, err := messagesCall.Do()
	if err != nil {
		return nil, fmt.Errorf("failed to list Gmail messages: %w", err)
	}

	// Process each message
	for _, message := range messagesResponse.Messages {
		if err := s.processGmailMessage(ctx, service, message.Id, req.UserID); err != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("Error processing message %s: %v", message.Id, err))
			continue
		}
		result.NewEmails++
	}

	result.EmailsProcessed = len(messagesResponse.Messages)
	return result, nil
}

// processGmailMessage processes a single Gmail message
func (s *EmailSyncService) processGmailMessage(ctx context.Context, service *gmail.Service, messageID string, userID uuid.UUID) error {
	// Get full message details
	message, err := service.Users.Messages.Get("me", messageID).Format("full").Do()
	if err != nil {
		return fmt.Errorf("failed to get message details: %w", err)
	}

	// Log raw Gmail API response for debugging
	log.Printf("=== RAW GMAIL API RESPONSE FOR MESSAGE %s (EmailSyncService) ===", messageID)
	log.Printf("Message ID: %s", message.Id)
	log.Printf("Thread ID: %s", message.ThreadId)
	log.Printf("Label IDs: %v", message.LabelIds)
	log.Printf("Snippet: %s", message.Snippet)
	log.Printf("History ID: %s", message.HistoryId)
	log.Printf("Internal Date: %d", message.InternalDate)
	log.Printf("Size Estimate: %d", message.SizeEstimate)

	if message.Payload != nil {
		log.Printf("Payload MIME Type: %s", message.Payload.MimeType)
		log.Printf("Payload Filename: %s", message.Payload.Filename)
		log.Printf("Payload Part ID: %s", message.Payload.PartId)
		log.Printf("Number of Parts: %d", len(message.Payload.Parts))

		if len(message.Payload.Headers) > 0 {
			log.Printf("Headers:")
			for _, header := range message.Payload.Headers {
				log.Printf("  %s: %s", header.Name, header.Value)
			}
		}

		if message.Payload.Body != nil {
			log.Printf("Body Size: %d", message.Payload.Body.Size)
			log.Printf("Body Attachment ID: %s", message.Payload.Body.AttachmentId)
			if message.Payload.Body.Data != "" {
				log.Printf("Body Data Length: %d", len(message.Payload.Body.Data))
			}
		}
	}
	log.Printf("=== END RAW GMAIL API RESPONSE (EmailSyncService) ===")

	// Extract email metadata
	email := &models.Email{
		UserID:    userID,
		MessageID: message.Id,
		ThreadID:  message.ThreadId,
	}

	// Parse headers
	for _, header := range message.Payload.Headers {
		switch header.Name {
		case "Subject":
			email.Subject = header.Value
		case "From":
			email.FromEmail = header.Value // TODO: Parse name and email separately
		case "To":
			email.ToEmails = header.Value // TODO: Convert to JSON array
		case "Date":
			// TODO: Parse date properly
			email.SentAt = time.Now()
		}
	}

	email.ReceivedAt = time.Unix(message.InternalDate/1000, 0)
	email.EmailSize = message.SizeEstimate
	email.HasAttachments = len(message.Payload.Parts) > 1

	// Save email to database
	if err := s.db.Create(email).Error; err != nil {
		return fmt.Errorf("failed to save email to database: %w", err)
	}

	// TODO: Store email body in S3
	// TODO: Queue for AI analysis

	log.Printf("Processed Gmail message: %s - %s", message.Id, email.Subject)
	return nil
}

// syncMicrosoftEmails handles Microsoft Graph email synchronization using delta query
func (s *EmailSyncService) syncMicrosoftEmails(ctx context.Context, req *EmailSyncRequest) (*SyncResult, error) {
	result := &SyncResult{
		LastSyncAt: time.Now(),
		Errors:     []string{},
	}

	// TODO: Implement Microsoft Graph email sync using delta query
	// This would use the Microsoft Graph SDK to:
	// 1. Create Graph service client with token
	// 2. Use delta query for incremental sync
	// 3. Process messages similar to Gmail

	log.Printf("Microsoft Graph email sync not yet implemented for user %d", req.UserID)
	result.Errors = append(result.Errors, "Microsoft Graph sync not implemented")

	return result, nil
}

// GetSyncStatus returns the current sync status for a user
func (s *EmailSyncService) GetSyncStatus(ctx context.Context, userID uuid.UUID, provider oauth.Provider) (*SyncResult, error) {
	var totalEmails int64
	var newEmails int64
	var updatedEmails int64
	var lastSyncAt time.Time

	// Count total emails for this user
	if err := s.db.Model(&models.Email{}).Where("user_id = ?", userID).Count(&totalEmails).Error; err != nil {
		return nil, fmt.Errorf("failed to count total emails: %w", err)
	}

	// Count emails created in the last 24 hours (new emails)
	yesterday := time.Now().Add(-24 * time.Hour)
	if err := s.db.Model(&models.Email{}).Where("user_id = ? AND created_at > ?", userID, yesterday).Count(&newEmails).Error; err != nil {
		return nil, fmt.Errorf("failed to count new emails: %w", err)
	}

	// Count emails updated in the last 24 hours (excluding newly created ones)
	if err := s.db.Model(&models.Email{}).Where("user_id = ? AND updated_at > ? AND created_at <= ?", userID, yesterday, yesterday).Count(&updatedEmails).Error; err != nil {
		return nil, fmt.Errorf("failed to count updated emails: %w", err)
	}

	// Get the most recent sync time from the emails table
	var mostRecentEmail models.Email
	if err := s.db.Where("user_id = ?", userID).Order("last_sync_at DESC").First(&mostRecentEmail).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			// No emails found, use current time
			lastSyncAt = time.Now()
		} else {
			return nil, fmt.Errorf("failed to get last sync time: %w", err)
		}
	} else {
		lastSyncAt = mostRecentEmail.LastSyncAt
	}

	return &SyncResult{
		EmailsProcessed: int(totalEmails),
		NewEmails:       int(newEmails),
		UpdatedEmails:   int(updatedEmails),
		Errors:          []string{},
		LastSyncAt:      lastSyncAt,
	}, nil
}

// ScheduleSync schedules an email sync job for a user
func (s *EmailSyncService) ScheduleSync(ctx context.Context, userID uuid.UUID, provider oauth.Provider) error {
	// TODO: Implement sync scheduling using QStash
	log.Printf("Scheduling email sync for user %d, provider %s", userID, provider)
	return nil
}
