package models

import (
	"time"
	"github.com/google/uuid"
)

// ProviderSyncMetadata stores sync metadata for different email providers
type ProviderSyncMetadata struct {
	ID        uuid.UUID `json:"id" gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	UserID    uuid.UUID `json:"user_id" gorm:"type:uuid;not null;index"`
	Provider  string    `json:"provider" gorm:"not null;index"` // "gmail", "outlook", etc.
	
	// Consent and permissions
	ConsentGranted   bool       `json:"consent_granted" gorm:"default:false"`
	ConsentDate      *time.Time `json:"consent_date"`
	SyncEnabled      bool       `json:"sync_enabled" gorm:"default:false"`
	
	// Sync state tracking
	LastSyncAt       *time.Time `json:"last_sync_at"`
	LastFullSyncAt   *time.Time `json:"last_full_sync_at"`
	SyncVersion      int        `json:"sync_version" gorm:"default:1"`
	
	// Provider-specific sync tokens/IDs
	HistoryID        string     `json:"history_id,omitempty"`        // Gmail history ID
	DeltaToken       string     `json:"delta_token,omitempty"`       // Microsoft Graph delta token
	SyncToken        string     `json:"sync_token,omitempty"`        // Generic sync token
	
	// Sync configuration
	SyncFrequencyHours int      `json:"sync_frequency_hours" gorm:"default:24"`
	AutoSyncEnabled    bool     `json:"auto_sync_enabled" gorm:"default:true"`
	
	// Sync statistics
	TotalEmailsSynced  int64     `json:"total_emails_synced" gorm:"default:0"`
	LastSyncDuration   int       `json:"last_sync_duration_ms" gorm:"default:0"` // in milliseconds
	ConsecutiveErrors  int       `json:"consecutive_errors" gorm:"default:0"`
	LastErrorMessage   string    `json:"last_error_message,omitempty"`
	LastErrorAt        *time.Time `json:"last_error_at"`
	
	// Timestamps
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	
	// Relationships
	User User `gorm:"foreignKey:UserID"`
}

// SyncJob represents individual sync operations for tracking and monitoring
type SyncJob struct {
	ID          uuid.UUID `json:"id" gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	UserID      uuid.UUID `json:"user_id" gorm:"type:uuid;not null;index"`
	Provider    string    `json:"provider" gorm:"not null;index"`
	JobID       string    `json:"job_id" gorm:"unique;not null"` // External job ID (e.g., QStash message ID)
	
	// Job details
	SyncType    string    `json:"sync_type" gorm:"not null"` // "full", "incremental"
	Status      string    `json:"status" gorm:"not null;default:'queued'"` // "queued", "running", "completed", "failed", "cancelled"
	
	// Progress tracking
	EmailsProcessed   int       `json:"emails_processed" gorm:"default:0"`
	NewEmails         int       `json:"new_emails" gorm:"default:0"`
	UpdatedEmails     int       `json:"updated_emails" gorm:"default:0"`
	SkippedEmails     int       `json:"skipped_emails" gorm:"default:0"`
	DeletedEmails     int       `json:"deleted_emails" gorm:"default:0"`
	
	// Timing
	QueuedAt    time.Time  `json:"queued_at"`
	StartedAt   *time.Time `json:"started_at"`
	CompletedAt *time.Time `json:"completed_at"`
	Duration    int        `json:"duration_ms" gorm:"default:0"` // in milliseconds
	
	// Error handling
	ErrorMessage string     `json:"error_message,omitempty"`
	ErrorCode    string     `json:"error_code,omitempty"`
	RetryCount   int        `json:"retry_count" gorm:"default:0"`
	MaxRetries   int        `json:"max_retries" gorm:"default:3"`
	NextRetryAt  *time.Time `json:"next_retry_at"`
	
	// Sync parameters (stored as JSON for flexibility)
	SyncParameters string `json:"sync_parameters,omitempty" gorm:"type:jsonb"` // JSON string of sync request parameters
	
	// Result data
	ResultData string `json:"result_data,omitempty" gorm:"type:jsonb"` // JSON string of sync result
	
	// Timestamps
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	
	// Relationships
	User User `gorm:"foreignKey:UserID"`
}

// SyncJobStatus represents the possible states of a sync job
type SyncJobStatus string

const (
	SyncJobStatusQueued    SyncJobStatus = "queued"
	SyncJobStatusRunning   SyncJobStatus = "running"
	SyncJobStatusCompleted SyncJobStatus = "completed"
	SyncJobStatusFailed    SyncJobStatus = "failed"
	SyncJobStatusCancelled SyncJobStatus = "cancelled"
)

// SyncType represents the type of synchronization
type SyncType string

const (
	SyncTypeFull        SyncType = "full"
	SyncTypeIncremental SyncType = "incremental"
)

// Table names
func (ProviderSyncMetadata) TableName() string {
	return "provider_sync_metadata"
}

func (SyncJob) TableName() string {
	return "sync_jobs"
}

// Helper methods for ProviderSyncMetadata
func (p *ProviderSyncMetadata) IsConsentValid() bool {
	return p.ConsentGranted && p.SyncEnabled
}

func (p *ProviderSyncMetadata) HasRecentError() bool {
	return p.ConsecutiveErrors > 0 && p.LastErrorAt != nil
}

func (p *ProviderSyncMetadata) ShouldAutoSync() bool {
	if !p.AutoSyncEnabled || !p.IsConsentValid() {
		return false
	}
	
	if p.LastSyncAt == nil {
		return true // First sync
	}
	
	nextSyncTime := p.LastSyncAt.Add(time.Duration(p.SyncFrequencyHours) * time.Hour)
	return time.Now().After(nextSyncTime)
}

// Helper methods for SyncJob
func (s *SyncJob) IsActive() bool {
	return s.Status == string(SyncJobStatusQueued) || s.Status == string(SyncJobStatusRunning)
}

func (s *SyncJob) IsCompleted() bool {
	return s.Status == string(SyncJobStatusCompleted)
}

func (s *SyncJob) IsFailed() bool {
	return s.Status == string(SyncJobStatusFailed)
}

func (s *SyncJob) CanRetry() bool {
	return s.IsFailed() && s.RetryCount < s.MaxRetries
}

func (s *SyncJob) CalculateDuration() {
	if s.StartedAt != nil && s.CompletedAt != nil {
		s.Duration = int(s.CompletedAt.Sub(*s.StartedAt).Milliseconds())
	}
}