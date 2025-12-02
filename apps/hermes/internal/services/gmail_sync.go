package services

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"math"
	"mime"
	"mime/multipart"
	"net/mail"
	"strconv"
	"strings"
	"time"

	"follow-email-backend/internal/models"
	"follow-email-backend/pkg/debug"
	"follow-email-backend/pkg/email"
	gmailpkg "follow-email-backend/pkg/gmail"
	"follow-email-backend/pkg/oauth"
	"follow-email-backend/pkg/storage"

	"github.com/google/uuid"
	"google.golang.org/api/gmail/v1"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
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
	ReplyTo    string // Reply-To header (optional)
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

// EmailBodyContent captures both HTML and plain-text versions of an email body.
type EmailBodyContent struct {
	HTML     string
	Plain    string
	MimeType string
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

// SyncUserLabels fetches all user-created labels from Gmail and upserts them to the database
func (s *GmailSyncService) SyncUserLabels(ctx context.Context, userID uuid.UUID, service *gmail.Service) error {
	log.Printf("Syncing user labels for user %s", userID)

	// Fetch all labels from Gmail API
	labelsResponse, err := service.Users.Labels.List("me").Context(ctx).Do()
	if err != nil {
		return fmt.Errorf("failed to list Gmail labels: %w", err)
	}

	// Collect user-created labels to upsert
	var labelsToUpsert []models.UserLabel
	for _, label := range labelsResponse.Labels {
		// Skip system labels - we only store user-created labels
		if label.Type == "system" {
			continue
		}

		// Extract color if present
		var color models.LabelColor
		if label.Color != nil {
			color = models.LabelColor{
				BackgroundColor: label.Color.BackgroundColor,
				TextColor:       label.Color.TextColor,
			}
		}

		// Get visibility settings with defaults
		messageListVisibility := label.MessageListVisibility
		if messageListVisibility == "" {
			messageListVisibility = "show"
		}
		labelListVisibility := label.LabelListVisibility
		if labelListVisibility == "" {
			labelListVisibility = "labelShow"
		}

		labelsToUpsert = append(labelsToUpsert, models.UserLabel{
			UserID:                userID,
			GmailLabelID:          label.Id,
			LabelName:             label.Name,
			Color:                 color,
			MessageListVisibility: messageListVisibility,
			LabelListVisibility:   labelListVisibility,
			CreatedAt:             time.Now(),
			UpdatedAt:             time.Now(),
		})
	}

	// Batch upsert with ON CONFLICT
	if len(labelsToUpsert) > 0 {
		err := s.db.Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "user_id"}, {Name: "gmail_label_id"}},
			DoUpdates: clause.AssignmentColumns([]string{"label_name", "color", "message_list_visibility", "label_list_visibility", "updated_at"}),
		}).Create(&labelsToUpsert).Error

		if err != nil {
			return fmt.Errorf("failed to upsert labels: %w", err)
		}
	}

	log.Printf("Successfully synced %d user labels for user %s", len(labelsToUpsert), userID)
	return nil
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

	// Sync user labels first (before syncing emails)
	if err := s.SyncUserLabels(ctx, req.UserID, service); err != nil {
		log.Printf("Warning: Failed to sync user labels for user %s: %v", req.UserID, err)
		// Don't fail the entire sync - just log the warning and continue
		result.Errors = append(result.Errors, fmt.Sprintf("Label sync warning: %v", err))
	}

	// Set default max results if not specified
	maxResults := req.MaxResults
	if maxResults == 0 {
		maxResults = 1000 // Default batch size
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

	// Also sync emails with user labels that might not be in inbox/sent
	processedMessageIDs := make(map[string]bool)
	for _, msgID := range allMessageIDs {
		processedMessageIDs[msgID] = true
	}

	// Get all user labels from database
	var userLabels []models.UserLabel
	if err := s.db.Where("user_id = ?", req.UserID).Find(&userLabels).Error; err == nil && len(userLabels) > 0 {
		log.Printf("Syncing emails with %d user labels for user %s", len(userLabels), req.UserID)

		for _, label := range userLabels {
			// Fetch emails with this label
			labelMessagesCall := service.Users.Messages.List("me").LabelIds(label.GmailLabelID).MaxResults(100)
			labelMessagesResp, err := labelMessagesCall.Context(ctx).Do()
			if err != nil {
				log.Printf("Failed to fetch emails for label %s: %v", label.LabelName, err)
				continue
			}

			for _, msg := range labelMessagesResp.Messages {
				// Skip if already processed
				if processedMessageIDs[msg.Id] {
					continue
				}

				processedMessageIDs[msg.Id] = true
				debug.DebugTextPrint(fmt.Sprintf("Syncing labeled email %s from label %s", msg.Id, label.LabelName))

				if err := s.processGmailMessage(ctx, service, msg.Id, req.UserID, result); err != nil {
					log.Printf("Error processing labeled message %s: %v", msg.Id, err)
					result.Errors = append(result.Errors, fmt.Sprintf("Labeled message %s: %v", msg.Id, err))
					continue
				}
				result.EmailsProcessed++
			}
		}
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

	// COMMENTING OUT NOW. USED FOR TESTING THE LABEL UPDATE
	// DEBUG: Save Gmail API response for specific subjects to investigate label issues
	// debugSubjects := []string{
	// 	"Amazon Web Services GST Invoice Available",
	// 	"Pay Your AWS Invoices Automatically with UPI AutoPay",
	// 	"Testing email label 2",
	// 	"Annual reminder about Google Play terms and policies",
	// 	"Security alert for tanmoytssaha@gmail.com",
	// }
	//
	// // Get subject from headers
	// var emailSubject string
	// for _, header := range message.Payload.Headers {
	// 	if strings.EqualFold(header.Name, "Subject") {
	// 		emailSubject = header.Value
	// 		break
	// 	}
	// }
	//
	// // Check if this email matches any debug subjects
	// for _, debugSubject := range debugSubjects {
	// 	if strings.Contains(emailSubject, debugSubject) || strings.Contains(debugSubject, emailSubject) {
	// 		s.saveGmailResponseForDebug(message, emailSubject)
	// 		break
	// 	}
	// }

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
	emailContent := s.extractEmailContent(message.Payload)
	if emailContent.HTML != "" {
		emailContent.HTML = s.inlineCIDImages(ctx, service, message, emailContent.HTML)
	}
	if emailContent.MimeType != "" {
		emailData.MimeType = emailContent.MimeType
	}

	bodyForStorage := emailContent.HTML
	if bodyForStorage == "" && emailContent.Plain != "" {
		bodyForStorage = email.ConvertNewlinesToBR(emailContent.Plain)
	}

	if bodyForStorage != "" && s.s3Service != nil {
		bodyMetadata, err := s.s3Service.StoreEmailBody(ctx, userID.String(), messageID, bodyForStorage)
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
		debug.DebugTextPrint(fmt.Sprintf("Updating email %s - Category: %s, SystemLabels: %s, UserLabelIDs: %s",
			messageID, emailData.Category, emailData.SystemLabels, emailData.UserLabelIDs))
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
		headers[strings.ToLower(header.Name)] = header.Value
	}

	// Parse "From" header into name and email components.
	var fromAddress string
	var fromName string
	if rawFrom, exists := headers["from"]; exists && rawFrom != "" {
		if parsed, err := mail.ParseAddress(rawFrom); err == nil {
			fromAddress = parsed.Address
			fromName = parsed.Name
		} else {
			fromAddress = rawFrom
		}
	}

	// Parse canonical date value.
	var sentAt time.Time
	if dateStr, exists := headers["date"]; exists && dateStr != "" {
		if parsed, err := mail.ParseDate(dateStr); err == nil {
			sentAt = parsed
		}
	}
	if sentAt.IsZero() && message.InternalDate > 0 {
		sentAt = time.UnixMilli(message.InternalDate)
	}
	if sentAt.IsZero() {
		sentAt = time.Now()
	}

	// Determine sync version (clamped to int range).
	syncVersion := 1
	if message.HistoryId > 0 {
		if message.HistoryId > math.MaxInt32 {
			syncVersion = math.MaxInt32
		} else {
			syncVersion = int(message.HistoryId)
		}
	}

	parsedLabels := gmailpkg.ParseGmailLabels(message.LabelIds)

	// DEBUG: Log parsed labels for this email
	debug.DebugTextPrint(fmt.Sprintf("Email %s - Raw LabelIds: %v", message.Id, message.LabelIds))
	debug.DebugTextPrint(fmt.Sprintf("Email %s - Category: %s, SystemLabels: %v, GmailUserLabelIDs: %v",
		message.Id, parsedLabels.Category, parsedLabels.SystemLabels, parsedLabels.GmailUserLabelIDs))

	var userLabelIDs []int
	if len(parsedLabels.GmailUserLabelIDs) > 0 {
		var labels []models.UserLabel
		s.db.Where("user_id = ? AND gmail_label_id IN ?", userID, parsedLabels.GmailUserLabelIDs).Select("id").Find(&labels)

		debug.DebugTextPrint(fmt.Sprintf("Email %s - Found %d matching labels in DB for GmailUserLabelIDs", message.Id, len(labels)))

		for _, label := range labels {
			userLabelIDs = append(userLabelIDs, label.ID)
		}
	}

	debug.DebugTextPrint(fmt.Sprintf("Email %s - Final userLabelIDs: %v (JSON: %s)", message.Id, userLabelIDs, gmailpkg.ToJson(userLabelIDs)))

	return &models.Email{
		UserID:         userID,
		MessageID:      message.Id,
		ThreadID:       message.ThreadId,
		Subject:        headers["subject"],
		FromEmail:      fromAddress,
		FromName:       fromName,
		ToEmails:       headers["to"],
		CcEmails:       headers["cc"],
		BccEmails:      headers["bcc"],
		SentAt:         sentAt,
		ReceivedAt:     sentAt,
		EmailSize:      int64(message.SizeEstimate),
		S3BodyKey:      "", // Will be set when storing body in S3
		HasAttachments: len(message.Payload.Parts) > 1,
		Category:       parsedLabels.Category,
		SystemLabels:   gmailpkg.ToJson(parsedLabels.SystemLabels),
		UserLabelIDs:   gmailpkg.ToJson(userLabelIDs),
		LastSyncAt:     time.Now(),
		SyncVersion:    syncVersion,
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}
}

// extractEmailContent walks a Gmail message payload and returns both HTML and plain-text bodies.
func (s *GmailSyncService) extractEmailContent(payload *gmail.MessagePart) EmailBodyContent {
	result := EmailBodyContent{}
	if payload == nil {
		return result
	}

	return EmailBodyContent{HTML: decodeBodyDataToHTML(payload.Body.Data, payload.MimeType), MimeType: payload.MimeType}
}

// decodeBodyData converts Gmail's base64url body data into HTML and plain-text variants.
func decodeBodyDataToHTML(bodyData, mimeType string) string {
	if bodyData == "" {
		return bodyData
	}

	switch mimeType {
	case "text/html":
		decodedData, err := base64.URLEncoding.DecodeString(bodyData)
		if err != nil {
			return ""
		}

		return email.DecodeHTMLEntities(string(decodedData))
	case "text/plain":
		return bodyData
	default:
		return bodyData
	}

}

func (s *GmailSyncService) inlineCIDImages(ctx context.Context, service *gmail.Service, message *gmail.Message, html string) string {
	if html == "" || message == nil || message.Payload == nil {
		return html
	}

	inlineParts := collectInlineParts(message.Payload)
	if len(inlineParts) == 0 {
		return html
	}

	result := html
	for _, part := range inlineParts {
		cid := sanitizeContentID(getHeader(part.Headers, "Content-ID"))
		if cid == "" {
			continue
		}

		dataURI, err := s.resolveInlinePartData(ctx, service, message.Id, part)
		if err != nil || dataURI == "" {
			continue
		}

		replacements := []string{
			"cid:" + cid,
			"CID:" + cid,
		}
		for _, target := range replacements {
			result = strings.ReplaceAll(result, target, dataURI)
		}
	}

	return result
}

func collectInlineParts(part *gmail.MessagePart) []*gmail.MessagePart {
	results := []*gmail.MessagePart{}
	var walk func(*gmail.MessagePart)
	walk = func(p *gmail.MessagePart) {
		if p == nil {
			return
		}

		if hasContentID(p.Headers) {
			disposition := strings.ToLower(getHeader(p.Headers, "Content-Disposition"))
			if disposition == "" || strings.Contains(disposition, "inline") {
				results = append(results, p)
			}
		}

		for _, child := range p.Parts {
			walk(child)
		}
	}
	walk(part)
	return results
}

func hasContentID(headers []*gmail.MessagePartHeader) bool {
	for _, header := range headers {
		if strings.EqualFold(header.Name, "Content-ID") && strings.TrimSpace(header.Value) != "" {
			return true
		}
	}
	return false
}

func getHeader(headers []*gmail.MessagePartHeader, name string) string {
	for _, header := range headers {
		if strings.EqualFold(header.Name, name) {
			return header.Value
		}
	}
	return ""
}

func sanitizeContentID(id string) string {
	id = strings.TrimSpace(strings.Trim(id, "<>"))
	id = strings.TrimPrefix(strings.TrimPrefix(id, "cid:"), "CID:")
	return strings.TrimSpace(id)
}

func (s *GmailSyncService) resolveInlinePartData(ctx context.Context, service *gmail.Service, messageID string, part *gmail.MessagePart) (string, error) {
	if part == nil || part.Body == nil {
		return "", nil
	}

	var encoded string
	var err error

	if part.Body.Data != "" {
		encoded, err = normalizeAttachmentData(part.Body.Data)
	} else if part.Body.AttachmentId != "" && service != nil {
		attachment, fetchErr := service.Users.Messages.Attachments.Get("me", messageID, part.Body.AttachmentId).Context(ctx).Do()
		if fetchErr != nil {
			return "", fetchErr
		}

		encoded, err = normalizeAttachmentData(attachment.Data)
	}

	if err != nil || encoded == "" {
		return "", err
	}

	mimeType := part.MimeType
	if mimeType == "" {
		mimeType = "application/octet-stream"
	}

	return buildDataURI(mimeType, encoded), nil
}

func normalizeAttachmentData(data string) (string, error) {
	if data == "" {
		return "", nil
	}

	decodings := []func(string) ([]byte, error){
		base64.RawURLEncoding.DecodeString,
		base64.URLEncoding.DecodeString,
		base64.StdEncoding.DecodeString,
	}

	var decoded []byte
	var err error
	for _, decode := range decodings {
		decoded, err = decode(data)
		if err == nil {
			break
		}
	}
	if err != nil {
		return "", err
	}

	return base64.StdEncoding.EncodeToString(decoded), nil
}

func buildDataURI(mimeType, encoded string) string {
	if encoded == "" {
		return ""
	}
	return fmt.Sprintf("data:%s;base64,%s", mimeType, encoded)
}

// extractContentFromRaw decodes a raw MIME message to retrieve HTML/plain representations.
func (s *GmailSyncService) extractContentFromRaw(raw string) EmailBodyContent {
	if raw == "" {
		return EmailBodyContent{}
	}

	decoded, err := decodeBase64URL(raw)
	if err != nil {
		return EmailBodyContent{}
	}

	msg, err := mail.ReadMessage(strings.NewReader(decoded))
	if err != nil {
		return EmailBodyContent{}
	}

	contentType := msg.Header.Get("Content-Type")
	if contentType == "" {
		bodyBytes, _ := io.ReadAll(msg.Body)
		return EmailBodyContent{Plain: string(bodyBytes)}
	}

	mediaType, params, err := mime.ParseMediaType(contentType)
	if err != nil {
		bodyBytes, _ := io.ReadAll(msg.Body)
		return EmailBodyContent{Plain: string(bodyBytes)}
	}

	return parseMIMEBody(mediaType, params, msg.Body)
}

func parseMIMEBody(mediaType string, params map[string]string, body io.Reader) EmailBodyContent {
	result := EmailBodyContent{}

	if strings.HasPrefix(mediaType, "multipart/") {
		boundary := params["boundary"]
		if boundary == "" {
			return result
		}

		mr := multipart.NewReader(body, boundary)
		for {
			part, err := mr.NextPart()
			if err == io.EOF {
				break
			}
			if err != nil {
				return result
			}

			pType := part.Header.Get("Content-Type")
			if pType == "" {
				pType = "text/plain"
			}
			pMedia, pParams, err := mime.ParseMediaType(pType)
			if err != nil {
				pMedia = pType
			}

			child := parseMIMEBody(pMedia, pParams, part)
			if result.HTML == "" && child.HTML != "" {
				result.HTML = child.HTML
			}
			if result.Plain == "" && child.Plain != "" {
				result.Plain = child.Plain
			}

			part.Close()
		}

		return result
	}

	bodyBytes, _ := io.ReadAll(body)
	text := string(bodyBytes)

	switch {
	case strings.HasPrefix(strings.ToLower(mediaType), "text/html"):
		result.HTML = text
		if result.Plain == "" {
			result.Plain = strings.TrimSpace(email.StripHTMLTags(text))
		}
	case strings.HasPrefix(strings.ToLower(mediaType), "text/plain"):
		result.Plain = text
	default:
		result.HTML = text
	}

	return result
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

// COMMENTING OUT NOW. USED FOR TESTING THE LABEL UPDATE
// saveGmailResponseForDebug saves the Gmail API response to a JSON file for debugging
// func (s *GmailSyncService) saveGmailResponseForDebug(message *gmail.Message, subject string) {
// 	// Create debug directory if it doesn't exist
// 	debugDir := "logs/gmail_debug"
// 	if err := os.MkdirAll(debugDir, 0755); err != nil {
// 		log.Printf("Failed to create debug directory: %v", err)
// 		return
// 	}
//
// 	// Create a safe filename from subject
// 	safeSubject := strings.ReplaceAll(subject, " ", "_")
// 	safeSubject = strings.ReplaceAll(safeSubject, "/", "_")
// 	safeSubject = strings.ReplaceAll(safeSubject, "\\", "_")
// 	safeSubject = strings.ReplaceAll(safeSubject, ":", "_")
// 	if len(safeSubject) > 50 {
// 		safeSubject = safeSubject[:50]
// 	}
//
// 	filename := fmt.Sprintf("%s/%s_%s.json", debugDir, message.Id, safeSubject)
//
// 	// Create debug response structure
// 	debugResponse := map[string]interface{}{
// 		"message_id":    message.Id,
// 		"thread_id":     message.ThreadId,
// 		"label_ids":     message.LabelIds,
// 		"snippet":       message.Snippet,
// 		"history_id":    message.HistoryId,
// 		"internal_date": message.InternalDate,
// 		"size_estimate": message.SizeEstimate,
// 		"subject":       subject,
// 	}
//
// 	// Add headers
// 	headers := make(map[string]string)
// 	if message.Payload != nil {
// 		for _, h := range message.Payload.Headers {
// 			headers[h.Name] = h.Value
// 		}
// 	}
// 	debugResponse["headers"] = headers
//
// 	// Marshal to JSON
// 	jsonData, err := json.MarshalIndent(debugResponse, "", "  ")
// 	if err != nil {
// 		log.Printf("Failed to marshal debug response: %v", err)
// 		return
// 	}
//
// 	// Write to file
// 	if err := os.WriteFile(filename, jsonData, 0644); err != nil {
// 		log.Printf("Failed to write debug file: %v", err)
// 		return
// 	}
//
// 	debug.DebugSuccessTextPrint(fmt.Sprintf("Saved Gmail API response for '%s' to %s", subject, filename))
// }

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

	// Serialize labels to JSON (for legacy labels column)
	var labelsJSON string
	if len(message.LabelIds) > 0 {
		if labelsBytes, err := json.Marshal(message.LabelIds); err == nil {
			labelsJSON = string(labelsBytes)
		}
	}

	// Parse labels into category, system labels, and user label IDs
	parsedLabels := gmailpkg.ParseGmailLabels(message.LabelIds)

	// Look up user label IDs from the database
	var userLabelIDs []int
	if len(parsedLabels.GmailUserLabelIDs) > 0 {
		var labels []models.UserLabel
		s.db.Where("user_id = ? AND gmail_label_id IN ?", userID, parsedLabels.GmailUserLabelIDs).Select("id").Find(&labels)
		for _, label := range labels {
			userLabelIDs = append(userLabelIDs, label.ID)
		}
	}

	debug.DebugTextPrint(fmt.Sprintf("updateEmailLabels for %s - Category: %s, SystemLabels: %v, UserLabelIDs: %v",
		messageID, parsedLabels.Category, parsedLabels.SystemLabels, userLabelIDs))

	return s.db.Model(&models.Email{}).
		Where("message_id = ? AND user_id = ?", messageID, userID).
		Updates(map[string]interface{}{
			"labels":         labelsJSON,
			"category":       parsedLabels.Category,
			"system_labels":  gmailpkg.ToJson(parsedLabels.SystemLabels),
			"user_label_ids": gmailpkg.ToJson(userLabelIDs),
			"last_sync_at":   time.Now(),
			"updated_at":     time.Now(),
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

// GetFullGmailMessage fetches the complete Gmail message from the API
func (s *GmailSyncService) GetFullGmailMessage(ctx context.Context, userID uuid.UUID, messageID string) (*gmail.Message, error) {
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

	return message, nil
}

// GetEmailContentFromGmail fetches email body directly from Gmail API
func (s *GmailSyncService) GetEmailContentFromGmail(ctx context.Context, userID uuid.UUID, messageID string) (EmailBodyContent, error) {
	// Get Gmail service for user
	service, err := s.getGmailServiceForUser(ctx, userID)
	if err != nil {
		return EmailBodyContent{}, fmt.Errorf("failed to get Gmail service: %w", err)
	}

	// Fetch message from Gmail
	message, err := service.Users.Messages.Get("me", messageID).Format("full").Context(ctx).Do()
	if err != nil {
		return EmailBodyContent{}, fmt.Errorf("failed to fetch message from Gmail: %w", err)
	}

	// Extract email body
	content := s.extractEmailContent(message.Payload)
	if content.HTML == "" {
		if rawMsg, rawErr := service.Users.Messages.Get("me", messageID).Format("raw").Context(ctx).Do(); rawErr == nil {
			if rawContent := s.extractContentFromRaw(rawMsg.Raw); rawContent.HTML != "" || rawContent.Plain != "" {
				content = rawContent
			}
		}
		if content.HTML == "" && content.Plain == "" {
			return EmailBodyContent{}, fmt.Errorf("email body not found")
		}
	}

	if content.HTML == "" && content.Plain != "" {
		content.HTML = email.ConvertNewlinesToBR(content.Plain)
	}
	if content.Plain == "" && content.HTML != "" {
		content.Plain = strings.TrimSpace(email.StripHTMLTags(content.HTML))
	}

	return content, nil
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

	// Add Reply-To header if provided
	if req.ReplyTo != "" {
		msgBuilder.WriteString(fmt.Sprintf("Reply-To: %s\r\n", req.ReplyTo))
	}

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

	// Add thread ID if replying and it's valid
	// Gmail thread IDs are alphanumeric strings (typically 16+ characters)
	if req.ThreadID != "" {
		if isValidGmailThreadID(req.ThreadID) {
			gmailMessage.ThreadId = req.ThreadID
		} else {
			log.Printf("[WARNING] Invalid thread_id format '%s', ignoring. Gmail will create a new thread.", req.ThreadID)
			// Don't set ThreadId - let Gmail create a new thread
		}
	}

	// Send the message
	sentMessage, err := service.Users.Messages.Send("me", gmailMessage).Context(ctx).Do()
	if err != nil {
		return "", "", fmt.Errorf("failed to send email via Gmail API: %w", err)
	}

	log.Printf("Email sent successfully. MessageID: %s, ThreadID: %s", sentMessage.Id, sentMessage.ThreadId)

	return sentMessage.Id, sentMessage.ThreadId, nil
}

// isValidGmailThreadID validates if a string is a valid Gmail thread ID format
// Gmail thread IDs are alphanumeric strings, typically 16+ characters
func isValidGmailThreadID(threadID string) bool {
	if threadID == "" {
		return false
	}

	// Gmail thread IDs are alphanumeric (letters and numbers only)
	// They're typically 16+ characters long
	// Check if it contains only alphanumeric characters
	for _, char := range threadID {
		if !((char >= 'a' && char <= 'z') || (char >= 'A' && char <= 'Z') || (char >= '0' && char <= '9')) {
			return false
		}
	}

	// Gmail thread IDs are typically at least 16 characters
	// But we'll be lenient and allow 10+ characters
	if len(threadID) < 10 {
		return false
	}

	return true
}
