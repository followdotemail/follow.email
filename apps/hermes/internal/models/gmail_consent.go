package models

import (
	"time"
	"github.com/google/uuid"
)

// GmailConsent represents Gmail consent and sync preferences for a user
type GmailConsent struct {
	ID                uuid.UUID  `json:"id" gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	UserID            uuid.UUID  `json:"user_id" gorm:"type:uuid;unique;not null"`
	GmailConsent      bool       `json:"gmail_consent" gorm:"default:false"`
	GmailConsentDate  *time.Time `json:"gmail_consent_date"`
	GmailSyncEnabled  bool       `json:"gmail_sync_enabled" gorm:"default:false"`
	LastGmailSyncAt   *time.Time `json:"last_gmail_sync_at"`
	GmailHistoryId    string     `json:"gmail_history_id" gorm:"default:''"`
	ConsentIPAddress  string     `json:"consent_ip_address"`
	ConsentUserAgent  string     `json:"consent_user_agent"`
	CreatedAt         time.Time  `json:"created_at"`
	UpdatedAt         time.Time  `json:"updated_at"`
	User              User       `gorm:"foreignKey:UserID"`
}

// TableName returns the table name for GmailConsent
func (GmailConsent) TableName() string {
	return "gmail_consent"
}