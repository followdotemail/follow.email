package models

import (
	"time"
	"github.com/google/uuid"
)

type Email struct {
	ID              uuid.UUID `json:"id" gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	UserID          uuid.UUID `json:"user_id" gorm:"type:uuid;not null;index"`
	MessageID       string    `json:"message_id" gorm:"uniqueIndex;not null"` // Gmail/Outlook message ID
	ThreadID        string    `json:"thread_id" gorm:"index"`   // Gmail/Outlook thread ID
	Subject         string    `json:"subject"`
	FromEmail       string    `json:"from_email" gorm:"not null;index"`
	FromName        string    `json:"from_name"`
	ToEmails        string    `json:"to_emails"` // JSON array as string
	CcEmails        string    `json:"cc_emails"` // JSON array as string
	BccEmails       string    `json:"bcc_emails"` // JSON array as string
	SentAt          time.Time `json:"sent_at" gorm:"index"`
	ReceivedAt      time.Time `json:"received_at" gorm:"not null;index"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`

	// Email content storage references
	S3BodyKey        string `json:"s3_body_key"`
	S3AttachmentsKey string `json:"s3_attachments_key"`

	// Email metadata
	IsRead           bool   `json:"is_read" gorm:"default:false"`
	IsImportant      bool   `json:"is_important" gorm:"default:false"`
	HasAttachments   bool   `json:"has_attachments" gorm:"default:false"`
	EmailSize        int64  `json:"email_size"`
	MimeType         string `json:"mime_type"`
	Labels           string `json:"labels" gorm:"type:jsonb"` // JSON array of Gmail label IDs

	// Follow-up tracking
	RequiresFollowUp bool      `json:"requires_followup" gorm:"default:false"`
	FollowUpStatus   string    `json:"followup_status"` // "pending", "sent", "completed", "cancelled"
	LastFollowUpAt   *time.Time `json:"last_followup_at"`
	FollowUpCount    int       `json:"followup_count" gorm:"default:0"`

	// AI analysis
	AISummary        string    `json:"ai_summary" gorm:"type:text"`
	AISentiment      string    `json:"ai_sentiment"` // "positive", "neutral", "negative"
	AIPriority       string    `json:"ai_priority"`   // "high", "medium", "low"
	AIAnalyzedAt     *time.Time `json:"ai_analyzed_at"`

	// Sync metadata
	ProviderSyncID   string    `json:"provider_sync_id"`
	LastSyncAt       time.Time `json:"last_sync_at"`
	SyncVersion      int       `json:"sync_version" gorm:"default:0"`
}

type FollowUpTemplate struct {
	ID          uuid.UUID `json:"id" gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	UserID      uuid.UUID `json:"user_id" gorm:"type:uuid;not null;index"`
	Name        string    `json:"name" gorm:"not null"`
	Subject     string    `json:"subject"`
	BodyTemplate string   `json:"body_template" gorm:"type:text"`
	IsDefault   bool      `json:"is_default" gorm:"default:false"`
	IsActive    bool      `json:"is_active" gorm:"default:true"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type FollowUpSchedule struct {
	ID              uuid.UUID  `json:"id" gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	UserID          uuid.UUID  `json:"user_id" gorm:"type:uuid;not null;index"`
	EmailID         uuid.UUID  `json:"email_id" gorm:"type:uuid;not null;index"`
	TemplateID      *uuid.UUID `json:"template_id" gorm:"type:uuid"`
	ScheduledAt     time.Time  `json:"scheduled_at" gorm:"not null;index"`
	Status          string     `json:"status" gorm:"default:'pending'"` // "pending", "sent", "failed", "cancelled"
	AttemptCount    int        `json:"attempt_count" gorm:"default:0"`
	LastAttemptAt   *time.Time `json:"last_attempt_at"`
	ErrorMessage    string     `json:"error_message"`
	CreatedAt       time.Time  `json:"created_at"`
	UpdatedAt       time.Time  `json:"updated_at"`

	// Generated content
	GeneratedSubject string `json:"generated_subject"`
	GeneratedBody    string `json:"generated_body" gorm:"type:text"`
}

type EmailAnalytics struct {
	ID                uuid.UUID `json:"id" gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	UserID            uuid.UUID `json:"user_id" gorm:"type:uuid;not null;index"`
	EmailID           uuid.UUID `json:"email_id" gorm:"type:uuid;not null;index"`
	EventType         string    `json:"event_type" gorm:"not null;index"` // "sent", "opened", "clicked", "replied"
	EventTimestamp    time.Time `json:"event_timestamp" gorm:"not null;index"`
	Metadata          string    `json:"metadata" gorm:"type:text"` // JSON metadata
	CreatedAt         time.Time `json:"created_at"`
}

// TableName methods for database mapping
func (Email) TableName() string {
	return "emails"
}

func (FollowUpTemplate) TableName() string {
	return "followup_templates"
}

func (FollowUpSchedule) TableName() string {
	return "followup_schedules"
}

func (EmailAnalytics) TableName() string {
	return "email_analytics"
}