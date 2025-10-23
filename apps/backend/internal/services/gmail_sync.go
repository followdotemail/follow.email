package services

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"strconv"
	"strings"
	"time"

	"follow-email-backend/internal/models"
	"follow-email-backend/pkg/email"
	"follow-email-backend/pkg/oauth"
	"follow-email-backend/pkg/storage"

	"github.com/google/uuid"
	"google.golang.org/api/gmail/v1"
	"gorm.io/gorm"
)

// GmailSyncService handles Gmail email synchronization
type GmailSyncService struct {
	db                *gorm.DB
	gmailOAuthService *oauth.GmailOAuthService
	tokenService      *GmailTokenService
	s3Service         *storage.S3Service
}

// SendEmailRequest represents a request to send an email
type SendEmailRequest struct {
	To         []string
	Cc         []string
	Bcc        []string
	Subject    string
	BodyText   string
	BodyHTML   string
	ThreadID   string
	InReplyTo  string
	References string
}

// GmailSyncRequest represents a Gmail sync request
type GmailSyncRequest struct {
	UserID     uuid.UUID  `json:"user_id"`
	FullSync   bool       `json:"full_sync"`
	MaxResults int64      `json:"max_results"`
	LabelIDs   []string   `json:"label_ids,omitempty"`
	Query      string     `json:"query,omitempty"`
	SyncSince  *time.Time `json:"sync_since,omitempty"`
}

// GmailSyncResult represents the result of a Gmail sync operation
type GmailSyncResult struct {
	UserID          uuid.UUID `json:"user_id"`
	EmailsProcessed int       `json:"emails_processed"`
	NewEmails       int       `json:"new_emails"`
	UpdatedEmails   int       `json:"updated_emails"`
	SkippedEmails   int       `json:"skipped_emails"`
	Errors          []string  `json:"errors"`
	StartedAt       time.Time `json:"started_at"`
	CompletedAt     time.Time `json:"completed_at"`
	LastSyncAt      time.Time `json:"last_sync_at"`
	HistoryID       string    `json:"history_id,omitempty"`
	SyncType        string    `json:"sync_type"` // "full" or "incremental"
}

// NewGmailSyncService creates a new Gmail sync service
func NewGmailSyncService(db *gorm.DB, gmailOAuthService *oauth.GmailOAuthService, tokenService *GmailTokenService, s3Service *storage.S3Service) *GmailSyncService {
	return &GmailSyncService{
		db:                db,
		gmailOAuthService: gmailOAuthService,
		tokenService:      tokenService,
		s3Service:         s3Service,
	}
}

// SyncUserEmails performs Gmail synchronization for a specific user
func (s *GmailSyncService) SyncUserEmails(ctx context.Context, req *GmailSyncRequest) (*GmailSyncResult, error) {
	result := &GmailSyncResult{
		UserID:    req.UserID,
		StartedAt: time.Now(),
		Errors:    []string{},
	}

	// Get valid token for user
	tokenInfo, err := s.tokenService.GetValidToken(ctx, req.UserID)
	if err != nil {
		return nil, fmt.Errorf("failed to get valid Gmail token: %w", err)
	}

	// Create Gmail service
	gmailService, err := s.gmailOAuthService.CreateGmailService(ctx, tokenInfo)
	if err != nil {
		return nil, fmt.Errorf("failed to create Gmail service: %w", err)
	}

	// Get user's Gmail profile
	profile, err := gmailService.Users.GetProfile("me").Context(ctx).Do()
	if err != nil {
		return nil, fmt.Errorf("failed to get Gmail profile: %w", err)
	}

	log.Printf("Starting Gmail sync for user %s, email: %s", req.UserID, profile.EmailAddress)

	// Get user from database to check last sync
	var user models.User
	if err := s.db.Where("id = ?", req.UserID).First(&user).Error; err != nil {
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	// Get Gmail consent record to check sync status
	var gmailConsent models.GmailConsent
	if err := s.db.Where("user_id = ?", req.UserID).First(&gmailConsent).Error; err != nil {
		return nil, fmt.Errorf("failed to get Gmail consent record: %w", err)
	}

	// Determine sync type
	if req.FullSync || gmailConsent.GmailHistoryId == "" || gmailConsent.LastGmailSyncAt == nil {
		result.SyncType = "full"
		return s.performFullSync(ctx, gmailService, req, result, &user, &gmailConsent)
	} else {
		result.SyncType = "incremental"
		return s.performIncrementalSync(ctx, gmailService, req, result, &user, &gmailConsent)
	}
}

// performFullSync performs a full email synchronization
func (s *GmailSyncService) performFullSync(ctx context.Context, service *gmail.Service, req *GmailSyncRequest, result *GmailSyncResult, user *models.User, gmailConsent *models.GmailConsent) (*GmailSyncResult, error) {
	log.Printf("Performing full Gmail sync for user %s", req.UserID)

	// Set default max results if not specified
	maxResults := req.MaxResults
	if maxResults == 0 {
		maxResults = 100 // Default batch size
	}

	// Build query
	query := req.Query
	if query == "" {
		query = "in:inbox OR in:sent" // Default to inbox and sent emails
	}

	// Add date filter if specified
	if req.SyncSince != nil {
		query += fmt.Sprintf(" after:%s", req.SyncSince.Format("2006/01/02"))
	}

	// List messages
	messagesCall := service.Users.Messages.List("me").Q(query).MaxResults(maxResults)
	if len(req.LabelIDs) > 0 {
		messagesCall = messagesCall.LabelIds(req.LabelIDs...)
	}

	var allMessageIDs []string
	var nextPageToken string

	for {
		if nextPageToken != "" {
			messagesCall = messagesCall.PageToken(nextPageToken)
		}

		messagesResponse, err := messagesCall.Context(ctx).Do()
		if err != nil {
			return nil, fmt.Errorf("failed to list Gmail messages: %w", err)
		}

		for _, message := range messagesResponse.Messages {
			allMessageIDs = append(allMessageIDs, message.Id)
		}

		nextPageToken = messagesResponse.NextPageToken
		if nextPageToken == "" || int64(len(allMessageIDs)) >= maxResults {
			break
		}
	}

	log.Printf("Found %d messages to sync for user %s", len(allMessageIDs), req.UserID)

	// Process messages in batches
	batchSize := 10
	for i := 0; i < len(allMessageIDs); i += batchSize {
		end := i + batchSize
		if end > len(allMessageIDs) {
			end = len(allMessageIDs)
		}

		batch := allMessageIDs[i:end]
		for _, messageID := range batch {
			if err := s.processGmailMessage(ctx, service, messageID, req.UserID, result); err != nil {
				log.Printf("Error processing message %s: %v", messageID, err)
				result.Errors = append(result.Errors, fmt.Sprintf("Message %s: %v", messageID, err))
				continue
			}
			result.EmailsProcessed++
		}

		// Add small delay between batches to avoid rate limiting
		time.Sleep(100 * time.Millisecond)
	}

	// Update user sync status
	now := time.Now()
	result.CompletedAt = now
	result.LastSyncAt = now

	// Get latest history ID for future incremental syncs
	profile, err := service.Users.GetProfile("me").Context(ctx).Do()
	if err == nil {
		result.HistoryID = strconv.FormatUint(profile.HistoryId, 10)
		if err := s.tokenService.UpdateUserSyncStatus(req.UserID, &now, result.HistoryID); err != nil {
			log.Printf("Failed to update user sync status: %v", err)
		}
	}

	return result, nil
}

// performIncrementalSync performs incremental email synchronization using Gmail history
func (s *GmailSyncService) performIncrementalSync(ctx context.Context, service *gmail.Service, req *GmailSyncRequest, result *GmailSyncResult, user *models.User, gmailConsent *models.GmailConsent) (*GmailSyncResult, error) {
	log.Printf("Performing incremental Gmail sync for user %s from history ID: %s", req.UserID, gmailConsent.GmailHistoryId)

	historyID, err := strconv.ParseUint(gmailConsent.GmailHistoryId, 10, 64)
	if err != nil {
		return nil, fmt.Errorf("invalid history ID: %w", err)
	}

	// Get history changes
	historyCall := service.Users.History.List("me").StartHistoryId(historyID)
	historyResponse, err := historyCall.Context(ctx).Do()
	if err != nil {
		// If history is too old, fall back to full sync
		if strings.Contains(err.Error(), "history") {
			log.Printf("History too old for user %s, falling back to full sync", req.UserID)
			req.FullSync = true
			return s.performFullSync(ctx, service, req, result, user, gmailConsent)
		}
		return nil, fmt.Errorf("failed to get Gmail history: %w", err)
	}

	if len(historyResponse.History) == 0 {
		log.Printf("No new changes for user %s", req.UserID)
		result.CompletedAt = time.Now()
		return result, nil
	}

	// Process history changes
	processedMessages := make(map[string]bool)
	for _, history := range historyResponse.History {
		// Process added messages
		for _, added := range history.MessagesAdded {
			if !processedMessages[added.Message.Id] {
				if err := s.processGmailMessage(ctx, service, added.Message.Id, req.UserID, result); err != nil {
					log.Printf("Error processing added message %s: %v", added.Message.Id, err)
					result.Errors = append(result.Errors, fmt.Sprintf("Added message %s: %v", added.Message.Id, err))
				} else {
					result.NewEmails++
				}
				processedMessages[added.Message.Id] = true
				result.EmailsProcessed++
			}
		}

		// Process deleted messages
		for _, deleted := range history.MessagesDeleted {
			if err := s.markEmailAsDeleted(deleted.Message.Id, req.UserID); err != nil {
				log.Printf("Error marking message as deleted %s: %v", deleted.Message.Id, err)
				result.Errors = append(result.Errors, fmt.Sprintf("Deleted message %s: %v", deleted.Message.Id, err))
			}
		}

		// Process label changes
		for _, labelChange := range history.LabelsAdded {
			if !processedMessages[labelChange.Message.Id] {
				if err := s.updateEmailLabels(ctx, service, labelChange.Message.Id, req.UserID); err != nil {
					log.Printf("Error updating labels for message %s: %v", labelChange.Message.Id, err)
					result.Errors = append(result.Errors, fmt.Sprintf("Label update %s: %v", labelChange.Message.Id, err))
				} else {
					result.UpdatedEmails++
				}
				processedMessages[labelChange.Message.Id] = true
				result.EmailsProcessed++
			}
		}

		for _, labelChange := range history.LabelsRemoved {
			if !processedMessages[labelChange.Message.Id] {
				if err := s.updateEmailLabels(ctx, service, labelChange.Message.Id, req.UserID); err != nil {
					log.Printf("Error updating labels for message %s: %v", labelChange.Message.Id, err)
					result.Errors = append(result.Errors, fmt.Sprintf("Label update %s: %v", labelChange.Message.Id, err))
				} else {
					result.UpdatedEmails++
				}
				processedMessages[labelChange.Message.Id] = true
				result.EmailsProcessed++
			}
		}
	}

	// Update user sync status
	now := time.Now()
	result.CompletedAt = now
	result.LastSyncAt = now
	result.HistoryID = strconv.FormatUint(historyResponse.HistoryId, 10)

	if err := s.tokenService.UpdateUserSyncStatus(req.UserID, &now, result.HistoryID); err != nil {
		log.Printf("Failed to update user sync status: %v", err)
	}

	return result, nil
}

// processGmailMessage processes a single Gmail message
func (s *GmailSyncService) processGmailMessage(ctx context.Context, service *gmail.Service, messageID string, userID uuid.UUID, result *GmailSyncResult) error {
	// Get full message details
	message, err := service.Users.Messages.Get("me", messageID).Format("full").Context(ctx).Do()
	if err != nil {
		return fmt.Errorf("failed to get message details: %w", err)
	}

	// Log raw Gmail API response for debugging
	// fmt.Println("=== RAW GMAIL API RESPONSE FOR MESSAGE %s ===", messageID)
	// fmt.Printf(message.Payload.Body.Data)
	// fmt.Printf("Message ID: %s", message.Id)
	// fmt.Printf("Thread ID: %s", message.ThreadId)
	// fmt.Printf("Label IDs: %v", message.LabelIds)
	// fmt.Println("Snippet: %s", message.Snippet)
	// fmt.Printf("History ID: %s", message.HistoryId)
	// fmt.Println("Internal Date: %d", message.NullFields)
	// fmt.Println("Size Estimate: %d", message.Raw)

	// if message.Payload != nil {
	// 	fmt.Println("Payload MIME Type: %s", message.Payload.MimeType)
	// 	fmt.Println("Payload Filename: %s", message.Payload.Filename)
	// 	fmt.Println("Payload Part Body Data: %s", message.Payload.Parts[0].Body.Data)
	// fmt.Printf("Payload Part ID: %s", message.Payload.PartId)
	// fmt.Printf("Number of Parts: %d", len(message.Payload.Parts))

	// if len(message.Payload.Headers) > 0 {
	// 	log.Printf("Headers:")
	// 	for _, header := range message.Payload.Headers {
	// 		fmt.Printf("  %s: %s", header.Name, header.Value)
	// 	}
	// }

	// 	if message.Payload.Body != nil {
	// 		fmt.Println("Body Size: %d", message.Payload.Body.Size)
	// 		fmt.Println("Body Attachment ID: %s", message.Payload.Body.AttachmentId)
	// 		// fmt.Printf("Body Attachment ID: %s", message.Raw)
	// 		if message.Payload.Body.Data != "" {
	// 			fmt.Println("Body Data Length: %d", len(message.Payload.Body.Data))
	// 		}
	// 	}
	// }
	// fmt.Println("=== END RAW GMAIL API RESPONSE ===")

	// Check if email already exists
	var existingEmail models.Email
	existsResult := s.db.Where("message_id = ? AND user_id = ?", messageID, userID).First(&existingEmail)

	// Extract email data
	emailData := s.extractEmailData(message, userID)

	// Extract and store email body
	emailBody := s.extractEmailBody(message.Payload)
	if emailBody != "" && s.s3Service != nil {
		bodyMetadata, err := s.s3Service.StoreEmailBody(ctx, userID.String(), messageID, emailBody)
		if err != nil {
			log.Printf("Failed to store email body for message %s: %v", messageID, err)
		} else {
			emailData.S3BodyKey = bodyMetadata.S3Key
			log.Printf("Stored email body for message %s", messageID)
		}
	}

	// Process attachments
	if len(message.Payload.Parts) > 1 {
		s.processAttachments(ctx, service, message, userID, messageID, emailData)
	}

	if existsResult.Error == gorm.ErrRecordNotFound {
		// Create new email record
		if err := s.db.Create(emailData).Error; err != nil {
			return fmt.Errorf("failed to create email record: %w", err)
		}
		result.NewEmails++
	} else if existsResult.Error == nil {
		// Update existing email record
		emailData.ID = existingEmail.ID
		emailData.CreatedAt = existingEmail.CreatedAt
		if err := s.db.Save(emailData).Error; err != nil {
			return fmt.Errorf("failed to update email record: %w", err)
		}
		result.UpdatedEmails++
	} else {
		return fmt.Errorf("failed to check existing email: %w", existsResult.Error)
	}

	return nil
}

// extractEmailData extracts email data from Gmail message
func (s *GmailSyncService) extractEmailData(message *gmail.Message, userID uuid.UUID) *models.Email {
	headers := make(map[string]string)
	for _, header := range message.Payload.Headers {
		headers[header.Name] = header.Value
	}

	// Parse date
	var receivedAt time.Time
	if dateStr, exists := headers["Date"]; exists {
		if parsed, err := time.Parse(time.RFC1123Z, dateStr); err == nil {
			receivedAt = parsed
		} else if parsed, err := time.Parse("Mon, 2 Jan 2006 15:04:05 -0700", dateStr); err == nil {
			receivedAt = parsed
		} else {
			receivedAt = time.Now()
		}
	} else {
		receivedAt = time.Now()
	}

	// Serialize labels to JSON
	var labelsJSON string
	if len(message.LabelIds) > 0 {
		if labelsBytes, err := json.Marshal(message.LabelIds); err == nil {
			labelsJSON = string(labelsBytes)
		}
	}

	// Determine if email is sent or received
	// Note: isOutbound logic available but not currently used in the model

	return &models.Email{
		UserID:         userID,
		MessageID:      message.Id,
		ThreadID:       message.ThreadId,
		Subject:        headers["Subject"],
		FromEmail:      headers["From"],
		ToEmails:       headers["To"],
		CcEmails:       headers["Cc"],
		BccEmails:      headers["Bcc"],
		ReceivedAt:     receivedAt,
		S3BodyKey:      "", // Will be set when storing body in S3
		HasAttachments: len(message.Payload.Parts) > 1,
		Labels:         labelsJSON,
		LastSyncAt:     time.Now(),
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}
}

// extractEmailBody extracts the email body from message payload using enhanced parsing logic
func (s *GmailSyncService) extractEmailBody(payload *gmail.MessagePart) string {
	if payload.Body != nil && payload.Body.Data != "" {
		// Use the new enhanced email body processing logic
		parsedBody := email.ProcessEmailBodyWithFallback(payload.Body.Data)
		fmt.Println("=== PARSED EMAIL BODY ===")
		fmt.Printf("Raw Data Length: %d\n", len(payload.Body.Data))
		fmt.Printf("Parsed Body Length: %d\n", len(parsedBody))
		fmt.Printf("Parsed Content:\n%s\n", parsedBody)
		fmt.Println("=== END PARSED EMAIL BODY ===")
		return parsedBody
	}

	// Check parts for body content
	for _, part := range payload.Parts {
		if part.MimeType == "text/plain" || part.MimeType == "text/html" {
			if part.Body != nil && part.Body.Data != "" {
				// Use the new enhanced email body processing logic
				parsedBody := email.ProcessEmailBodyWithFallback(part.Body.Data)
				fmt.Printf("=== PARSED EMAIL BODY (MIME: %s) ===\n", part.MimeType)
				fmt.Printf("Raw Data Length: %d\n", len(part.Body.Data))
				fmt.Printf("Parsed Body Length: %d\n", len(parsedBody))
				fmt.Printf("Parsed Content:\n%s\n", parsedBody)
				fmt.Println("=== END PARSED EMAIL BODY ===")
				return parsedBody
			}
		}
		// Recursively check nested parts
		if nestedBody := s.extractEmailBody(part); nestedBody != "" {
			return nestedBody
		}
	}

	return ""
}

// isMessageRead checks if a Gmail message is read
func (s *GmailSyncService) isMessageRead(message *gmail.Message) bool {
	for _, labelID := range message.LabelIds {
		if labelID == "UNREAD" {
			return false
		}
	}
	return true
}

// markEmailAsDeleted marks an email as deleted in the database
func (s *GmailSyncService) markEmailAsDeleted(messageID string, userID uuid.UUID) error {
	return s.db.Model(&models.Email{}).
		Where("message_id = ? AND user_id = ?", messageID, userID).
		Delete(&models.Email{}).Error
}

// updateEmailLabels updates email labels in the database
func (s *GmailSyncService) updateEmailLabels(ctx context.Context, service *gmail.Service, messageID string, userID uuid.UUID) error {
	// Fetch the latest message from Gmail to get updated labels
	message, err := service.Users.Messages.Get("me", messageID).Context(ctx).Do()
	if err != nil {
		return fmt.Errorf("failed to fetch message for label update: %w", err)
	}

	// Serialize labels to JSON
	var labelsJSON string
	if len(message.LabelIds) > 0 {
		if labelsBytes, err := json.Marshal(message.LabelIds); err == nil {
			labelsJSON = string(labelsBytes)
		}
	}

	return s.db.Model(&models.Email{}).
		Where("message_id = ? AND user_id = ?", messageID, userID).
		Updates(map[string]interface{}{
			"labels":       labelsJSON,
			"last_sync_at": time.Now(),
			"updated_at":   time.Now(),
		}).Error
}

// Helper function to decode base64url encoded strings
func decodeBase64URL(data string) (string, error) {
	// Gmail uses base64url encoding without padding
	// Add padding if necessary
	switch len(data) % 4 {
	case 2:
		data += "=="
	case 3:
		data += "="
	}

	// Replace URL-safe characters
	data = strings.ReplaceAll(data, "-", "+")
	data = strings.ReplaceAll(data, "_", "/")

	// Decode base64
	decoded, err := base64.StdEncoding.DecodeString(data)
	if err != nil {
		return "", err
	}

	return string(decoded), nil
}

// GetEmailContentFromGmail fetches email body directly from Gmail API
func (s *GmailSyncService) GetEmailContentFromGmail(ctx context.Context, userID uuid.UUID, messageID string) (string, error) {
	// Get Gmail service for user
	service, err := s.getGmailServiceForUser(ctx, userID)
	if err != nil {
		return "", fmt.Errorf("failed to get Gmail service: %w", err)
	}

	// Fetch message from Gmail
	message, err := service.Users.Messages.Get("me", messageID).Format("full").Context(ctx).Do()
	if err != nil {
		return "", fmt.Errorf("failed to fetch message from Gmail: %w", err)
	}

	// Extract email body
	body := s.extractEmailBody(message.Payload)
	if body == "" {
		return "", fmt.Errorf("email body not found")
	}

	return body, nil
}

// AttachmentInfo represents attachment metadata
type AttachmentInfo struct {
	Filename     string `json:"filename"`
	MimeType     string `json:"mime_type"`
	Size         int64  `json:"size"`
	AttachmentID string `json:"attachment_id"`
}

// GetEmailAttachmentsFromGmail fetches attachment metadata directly from Gmail API
func (s *GmailSyncService) GetEmailAttachmentsFromGmail(ctx context.Context, userID uuid.UUID, messageID string) ([]AttachmentInfo, error) {
	// Get Gmail service for user
	service, err := s.getGmailServiceForUser(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get Gmail service: %w", err)
	}

	// Fetch message from Gmail
	message, err := service.Users.Messages.Get("me", messageID).Format("full").Context(ctx).Do()
	if err != nil {
		return nil, fmt.Errorf("failed to fetch message from Gmail: %w", err)
	}

	// Extract attachments
	attachments := s.extractAttachmentMetadata(message.Payload)
	return attachments, nil
}

// DownloadAttachmentFromGmail downloads a specific attachment from Gmail API
func (s *GmailSyncService) DownloadAttachmentFromGmail(ctx context.Context, userID uuid.UUID, messageID, attachmentID string) ([]byte, string, error) {
	// Get Gmail service for user
	service, err := s.getGmailServiceForUser(ctx, userID)
	if err != nil {
		return nil, "", fmt.Errorf("failed to get Gmail service: %w", err)
	}

	// Fetch attachment data from Gmail
	attachment, err := service.Users.Messages.Attachments.Get("me", messageID, attachmentID).Context(ctx).Do()
	if err != nil {
		return nil, "", fmt.Errorf("failed to fetch attachment from Gmail: %w", err)
	}

	// Decode attachment data
	attachmentData, err := email.FromBinary(attachment.Data)
	if err != nil {
		return nil, "", fmt.Errorf("failed to decode attachment: %w", err)
	}

	// Get message to find the attachment filename and mime type
	message, err := service.Users.Messages.Get("me", messageID).Format("full").Context(ctx).Do()
	if err != nil {
		return nil, "", fmt.Errorf("failed to fetch message for attachment metadata: %w", err)
	}

	// Find the attachment part to get mime type
	mimeType := "application/octet-stream"
	filename := "attachment"
	s.findAttachmentMetadata(message.Payload, attachmentID, &filename, &mimeType)

	return []byte(attachmentData), mimeType, nil
}

// extractAttachmentMetadata extracts attachment metadata from message payload
func (s *GmailSyncService) extractAttachmentMetadata(payload *gmail.MessagePart) []AttachmentInfo {
	var attachments []AttachmentInfo

	if payload.Filename != "" && payload.Body != nil && payload.Body.AttachmentId != "" {
		attachments = append(attachments, AttachmentInfo{
			Filename:     payload.Filename,
			MimeType:     payload.MimeType,
			Size:         payload.Body.Size,
			AttachmentID: payload.Body.AttachmentId,
		})
	}

	// Recursively check parts
	for _, part := range payload.Parts {
		attachments = append(attachments, s.extractAttachmentMetadata(part)...)
	}

	return attachments
}

// findAttachmentMetadata finds attachment metadata by attachment ID
func (s *GmailSyncService) findAttachmentMetadata(payload *gmail.MessagePart, attachmentID string, filename, mimeType *string) bool {
	if payload.Body != nil && payload.Body.AttachmentId == attachmentID {
		if payload.Filename != "" {
			*filename = payload.Filename
		}
		if payload.MimeType != "" {
			*mimeType = payload.MimeType
		}
		return true
	}

	// Recursively search parts
	for _, part := range payload.Parts {
		if s.findAttachmentMetadata(part, attachmentID, filename, mimeType) {
			return true
		}
	}

	return false
}

// getGmailServiceForUser gets Gmail service for a specific user
func (s *GmailSyncService) getGmailServiceForUser(ctx context.Context, userID uuid.UUID) (*gmail.Service, error) {
	// Get user's Gmail token (returns valid token, refreshing if necessary)
	tokenInfo, err := s.tokenService.GetValidToken(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get Gmail token: %w", err)
	}

	// Create Gmail service with token info
	service, err := s.gmailOAuthService.CreateGmailService(ctx, tokenInfo)
	if err != nil {
		return nil, fmt.Errorf("failed to create Gmail service: %w", err)
	}

	return service, nil
}

// processAttachments processes and stores email attachments
func (s *GmailSyncService) processAttachments(ctx context.Context, service *gmail.Service, message *gmail.Message, userID uuid.UUID, messageID string, emailData *models.Email) {
	attachmentKeys := []string{}

	for _, part := range message.Payload.Parts {
		if part.Filename != "" && part.Body != nil && part.Body.AttachmentId != "" {
			// Get attachment data
			attachment, err := service.Users.Messages.Attachments.Get("me", messageID, part.Body.AttachmentId).Context(ctx).Do()
			if err != nil {
				log.Printf("Failed to get attachment %s for message %s: %v", part.Filename, messageID, err)
				continue
			}

			// Decode attachment data using enhanced parsing logic
			attachmentData, err := email.FromBinary(attachment.Data)
			if err != nil {
				log.Printf("Failed to decode attachment %s for message %s: %v", part.Filename, messageID, err)
				continue
			}

			// Store attachment in S3
			if s.s3Service != nil {
				contentType := part.MimeType
				if contentType == "" {
					contentType = "application/octet-stream"
				}

				attachmentMetadata, err := s.s3Service.StoreAttachment(ctx, userID.String(), messageID, part.Filename, bytes.NewReader([]byte(attachmentData)), contentType)
				if err != nil {
					log.Printf("Failed to store attachment %s for message %s: %v", part.Filename, messageID, err)
				} else {
					attachmentKeys = append(attachmentKeys, attachmentMetadata.S3Key)
					log.Printf("Stored attachment %s for message %s", part.Filename, messageID)
				}
			}
		}
	}

	// Store attachment keys in email record (as JSON string)
	if len(attachmentKeys) > 0 {
		// Convert to JSON string - for simplicity, we'll store as comma-separated values
		emailData.S3AttachmentsKey = strings.Join(attachmentKeys, ",")
		emailData.HasAttachments = true
	}
}

// SendEmail sends an email through Gmail API
func (s *GmailSyncService) SendEmail(ctx context.Context, userID uuid.UUID, req *SendEmailRequest) (string, string, error) {
	// Get Gmail service for user
	service, err := s.getGmailServiceForUser(ctx, userID)
	if err != nil {
		return "", "", fmt.Errorf("failed to get Gmail service: %w", err)
	}

	// Build the email message in RFC 2822 format
	var msgBuilder strings.Builder

	// Add To recipients
	if len(req.To) > 0 {
		msgBuilder.WriteString(fmt.Sprintf("To: %s\r\n", strings.Join(req.To, ", ")))
	}

	// Add CC recipients
	if len(req.Cc) > 0 {
		msgBuilder.WriteString(fmt.Sprintf("Cc: %s\r\n", strings.Join(req.Cc, ", ")))
	}

	// Add BCC recipients
	if len(req.Bcc) > 0 {
		msgBuilder.WriteString(fmt.Sprintf("Bcc: %s\r\n", strings.Join(req.Bcc, ", ")))
	}

	// Add Subject
	msgBuilder.WriteString(fmt.Sprintf("Subject: %s\r\n", req.Subject))

	// Add threading headers if provided
	if req.InReplyTo != "" {
		msgBuilder.WriteString(fmt.Sprintf("In-Reply-To: %s\r\n", req.InReplyTo))
	}
	if req.References != "" {
		msgBuilder.WriteString(fmt.Sprintf("References: %s\r\n", req.References))
	}

	// Add MIME headers for multipart content
	msgBuilder.WriteString("MIME-Version: 1.0\r\n")

	if req.BodyHTML != "" && req.BodyText != "" {
		// Both text and HTML versions
		boundary := fmt.Sprintf("boundary_%d", time.Now().UnixNano())
		msgBuilder.WriteString(fmt.Sprintf("Content-Type: multipart/alternative; boundary=\"%s\"\r\n\r\n", boundary))

		// Plain text version
		msgBuilder.WriteString(fmt.Sprintf("--%s\r\n", boundary))
		msgBuilder.WriteString("Content-Type: text/plain; charset=\"UTF-8\"\r\n")
		msgBuilder.WriteString("Content-Transfer-Encoding: 7bit\r\n\r\n")
		msgBuilder.WriteString(req.BodyText)
		msgBuilder.WriteString("\r\n\r\n")

		// HTML version
		msgBuilder.WriteString(fmt.Sprintf("--%s\r\n", boundary))
		msgBuilder.WriteString("Content-Type: text/html; charset=\"UTF-8\"\r\n")
		msgBuilder.WriteString("Content-Transfer-Encoding: 7bit\r\n\r\n")
		msgBuilder.WriteString(req.BodyHTML)
		msgBuilder.WriteString("\r\n\r\n")

		msgBuilder.WriteString(fmt.Sprintf("--%s--", boundary))
	} else if req.BodyHTML != "" {
		// HTML only
		msgBuilder.WriteString("Content-Type: text/html; charset=\"UTF-8\"\r\n\r\n")
		msgBuilder.WriteString(req.BodyHTML)
	} else {
		// Plain text only
		msgBuilder.WriteString("Content-Type: text/plain; charset=\"UTF-8\"\r\n\r\n")
		msgBuilder.WriteString(req.BodyText)
	}

	// Encode the message in base64url format
	rawMessage := msgBuilder.String()
	encodedMessage := base64.URLEncoding.EncodeToString([]byte(rawMessage))
	encodedMessage = strings.ReplaceAll(encodedMessage, "+", "-")
	encodedMessage = strings.ReplaceAll(encodedMessage, "/", "_")
	encodedMessage = strings.TrimRight(encodedMessage, "=")

	// Create Gmail message
	gmailMessage := &gmail.Message{
		Raw: encodedMessage,
	}

	// Add thread ID if replying
	if req.ThreadID != "" {
		gmailMessage.ThreadId = req.ThreadID
	}

	// Send the message
	sentMessage, err := service.Users.Messages.Send("me", gmailMessage).Context(ctx).Do()
	if err != nil {
		return "", "", fmt.Errorf("failed to send email via Gmail API: %w", err)
	}

	log.Printf("Email sent successfully. MessageID: %s, ThreadID: %s", sentMessage.Id, sentMessage.ThreadId)

	return sentMessage.Id, sentMessage.ThreadId, nil
}
