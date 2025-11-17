package models

import (
	"time"
	"github.com/google/uuid"
)

// UserPrivacyMetadata represents privacy and compliance settings for a user
type UserPrivacyMetadata struct {
	ID                uuid.UUID  `json:"id" gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	UserID            uuid.UUID  `json:"user_id" gorm:"type:uuid;unique;not null"`
	
	// GDPR Compliance
	GDPRConsent       bool       `json:"gdpr_consent" gorm:"default:false"`
	GDPRConsentDate   *time.Time `json:"gdpr_consent_date"`
	GDPRWithdrawn     bool       `json:"gdpr_withdrawn" gorm:"default:false"`
	GDPRWithdrawnDate *time.Time `json:"gdpr_withdrawn_date"`
	
	// CCPA Compliance
	CCPAConsent       bool       `json:"ccpa_consent" gorm:"default:false"`
	CCPAConsentDate   *time.Time `json:"ccpa_consent_date"`
	CCPAOptOut        bool       `json:"ccpa_opt_out" gorm:"default:false"`
	CCPAOptOutDate    *time.Time `json:"ccpa_opt_out_date"`
	
	// Data Retention
	DataRetentionDays int        `json:"data_retention_days" gorm:"default:2555"` // 7 years default
	DataRetentionSet  bool       `json:"data_retention_set" gorm:"default:false"` // Whether user explicitly set this
	
	// Audit Trail
	ConsentIPAddress  *string    `json:"consent_ip_address"`
	ConsentUserAgent  *string    `json:"consent_user_agent"`
	LastUpdatedBy     *string    `json:"last_updated_by"` // System or user ID who made the change
	
	CreatedAt         time.Time  `json:"created_at"`
	UpdatedAt         time.Time  `json:"updated_at"`
	
	// Relationships
	User              User       `gorm:"foreignKey:UserID"`
}

func (UserPrivacyMetadata) TableName() string {
	return "user_privacy_metadata"
}