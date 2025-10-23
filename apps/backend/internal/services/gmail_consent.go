package services

import (
	"fmt"
	"time"

	"follow-email-backend/internal/models"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// GmailConsentService manages Gmail consent and user preferences
type GmailConsentService struct {
	db *gorm.DB
}

// GmailConsentRequest represents a consent request
type GmailConsentRequest struct {
	UserID      uuid.UUID `json:"user_id"`
	ConsentType string    `json:"consent_type"` // "gmail_access", "data_sync", "ai_analysis"
	Granted     bool      `json:"granted"`
	Purpose     string    `json:"purpose,omitempty"`
	RetentionDays int     `json:"retention_days,omitempty"`
}

// GmailConsentRecord represents a stored consent record
type GmailConsentRecord struct {
	UserID        uuid.UUID  `json:"user_id"`
	ConsentType   string     `json:"consent_type"`
	Granted       bool       `json:"granted"`
	GrantedAt     time.Time  `json:"granted_at"`
	RevokedAt     *time.Time `json:"revoked_at,omitempty"`
	Purpose       string     `json:"purpose"`
	RetentionDays int        `json:"retention_days"`
	ExpiresAt     *time.Time `json:"expires_at,omitempty"`
}

// GmailUserPreferences represents Gmail-specific user preferences
type GmailUserPreferences struct {
	UserID              uuid.UUID `json:"user_id"`
	AutoSync            bool      `json:"auto_sync"`
	SyncFrequencyHours  int       `json:"sync_frequency_hours"`
	EnableAIAnalysis    bool      `json:"enable_ai_analysis"`
	EnableNotifications bool      `json:"enable_notifications"`
	DataRetentionDays   int       `json:"data_retention_days"`
	SyncLabels          []string  `json:"sync_labels"`
	ExcludeLabels       []string  `json:"exclude_labels"`
}

// NewGmailConsentService creates a new Gmail consent service
func NewGmailConsentService(db *gorm.DB) *GmailConsentService {
	return &GmailConsentService{
		db: db,
	}
}

// RecordConsent records user consent for Gmail access
func (s *GmailConsentService) RecordConsent(req *GmailConsentRequest) error {
	now := time.Now()
	
	// Check if consent record exists
	var gmailConsent models.GmailConsent
	result := s.db.Where("user_id = ?", req.UserID).First(&gmailConsent)
	
	if result.Error == gorm.ErrRecordNotFound {
		// Create new consent record
		gmailConsent = models.GmailConsent{
			UserID:    req.UserID,
			CreatedAt: now,
			UpdatedAt: now,
		}
		
		switch req.ConsentType {
		case "gmail_access":
			gmailConsent.GmailConsent = req.Granted
			if req.Granted {
				gmailConsent.GmailConsentDate = &now
				gmailConsent.GmailSyncEnabled = true
			}
		case "data_sync":
			gmailConsent.GmailSyncEnabled = req.Granted
		default:
			return fmt.Errorf("unknown consent type: %s", req.ConsentType)
		}
		
		return s.db.Create(&gmailConsent).Error
	} else if result.Error != nil {
		return fmt.Errorf("failed to check existing consent: %w", result.Error)
	}
	
	// Update existing consent record
	updates := map[string]interface{}{
		"updated_at": now,
	}
	
	switch req.ConsentType {
	case "gmail_access":
		updates["gmail_consent"] = req.Granted
		if req.Granted {
			updates["gmail_consent_date"] = now
			updates["gmail_sync_enabled"] = true
		} else {
			updates["gmail_consent_date"] = nil
			updates["gmail_sync_enabled"] = false
		}
	case "data_sync":
		updates["gmail_sync_enabled"] = req.Granted
	default:
		return fmt.Errorf("unknown consent type: %s", req.ConsentType)
	}
	
	return s.db.Model(&gmailConsent).Updates(updates).Error
}

// GetUserConsent retrieves user consent status
func (s *GmailConsentService) GetUserConsent(userID uuid.UUID) (*GmailConsentRecord, error) {
	var gmailConsent models.GmailConsent
	result := s.db.Where("user_id = ?", userID).First(&gmailConsent)
	if result.Error != nil {
		if result.Error == gorm.ErrRecordNotFound {
			// Return default consent record if none exists
			return &GmailConsentRecord{
				UserID:      userID,
				ConsentType: "gmail_access",
				Granted:     false,
				Purpose:     "Gmail email synchronization and analysis",
			}, nil
		}
		return nil, fmt.Errorf("failed to get gmail consent: %w", result.Error)
	}

	record := &GmailConsentRecord{
		UserID:      userID,
		ConsentType: "gmail_access",
		Granted:     gmailConsent.GmailConsent,
		Purpose:     "Gmail email synchronization and analysis",
	}

	if gmailConsent.GmailConsentDate != nil {
		record.GrantedAt = *gmailConsent.GmailConsentDate
	}

	return record, nil
}

// RevokeConsent revokes user consent for Gmail access
func (s *GmailConsentService) RevokeConsent(userID uuid.UUID, consentType string) error {
	updates := map[string]interface{}{
		"updated_at": time.Now(),
	}

	switch consentType {
	case "gmail_access":
		updates["gmail_consent"] = false
		updates["gmail_consent_date"] = nil
		updates["gmail_sync_enabled"] = false
	case "data_sync":
		updates["gmail_sync_enabled"] = false
	default:
		return fmt.Errorf("unknown consent type: %s", consentType)
	}

	return s.db.Model(&models.User{}).Where("id = ?", userID).Updates(updates).Error
}

// GetUserPreferences retrieves Gmail preferences for a user
func (s *GmailConsentService) GetUserPreferences(userID uuid.UUID) (*GmailUserPreferences, error) {
	var userPrefs models.UserPreferences
	result := s.db.Where("user_id = ?", userID).First(&userPrefs)
	if result.Error != nil {
		if result.Error == gorm.ErrRecordNotFound {
			// Return default preferences
			return &GmailUserPreferences{
				UserID:              userID,
				AutoSync:            true,
				SyncFrequencyHours:  1,
				EnableAIAnalysis:    true,
				EnableNotifications: true,
				DataRetentionDays:   365,
				SyncLabels:          []string{"INBOX", "SENT"},
				ExcludeLabels:       []string{"SPAM", "TRASH"},
			}, nil
		}
		return nil, fmt.Errorf("failed to get user preferences: %w", result.Error)
	}

	// Convert model to service struct
	prefs := &GmailUserPreferences{
		UserID:              userID,
		AutoSync:            userPrefs.AutoFollowUpEnabled, // Use available field
		SyncFrequencyHours:  1, // Default value
		EnableAIAnalysis:    userPrefs.AIResponseEnabled, // Use available field
		EnableNotifications: userPrefs.EmailNotifications, // Use available field
		DataRetentionDays:   365, // Default value
		SyncLabels:          []string{"INBOX", "SENT"},
		ExcludeLabels:       []string{"SPAM", "TRASH"},
	}

	return prefs, nil
}

// UpdateUserPreferences updates Gmail preferences for a user
func (s *GmailConsentService) UpdateUserPreferences(userID uuid.UUID, prefs *GmailUserPreferences) error {
	// Check if preferences exist
	var userPrefs models.UserPreferences
	result := s.db.Where("user_id = ?", userID).First(&userPrefs)
	
	updates := map[string]interface{}{
		"auto_followup_enabled": prefs.AutoSync,
		"ai_response_enabled":   prefs.EnableAIAnalysis,
		"email_notifications":   prefs.EnableNotifications,
	}

	if result.Error == gorm.ErrRecordNotFound {
		// Create new preferences
		newPrefs := models.UserPreferences{
			UserID:               userID,
			AutoFollowUpEnabled:  prefs.AutoSync,
			AIResponseEnabled:    prefs.EnableAIAnalysis,
			EmailNotifications:   prefs.EnableNotifications,
		}
		return s.db.Create(&newPrefs).Error
	} else if result.Error != nil {
		return fmt.Errorf("failed to get user preferences: %w", result.Error)
	}

	// Update existing preferences
	return s.db.Model(&userPrefs).Updates(updates).Error
}

// IsConsentValid checks if user consent is still valid
func (s *GmailConsentService) IsConsentValid(userID uuid.UUID) (bool, error) {
	var gmailConsent models.GmailConsent
	result := s.db.Where("user_id = ?", userID).First(&gmailConsent)
	if result.Error != nil {
		if result.Error == gorm.ErrRecordNotFound {
			return false, nil
		}
		return false, fmt.Errorf("failed to get gmail consent: %w", result.Error)
	}

	// Check if consent is granted and sync is enabled
	return gmailConsent.GmailConsent && gmailConsent.GmailSyncEnabled, nil
}

// GetConsentHistory retrieves consent history for audit purposes
func (s *GmailConsentService) GetConsentHistory(userID uuid.UUID) ([]GmailConsentRecord, error) {
	// For now, return current consent status
	// In a full implementation, you'd have a separate consent_history table
	current, err := s.GetUserConsent(userID)
	if err != nil {
		return nil, err
	}

	return []GmailConsentRecord{*current}, nil
}

// CleanupExpiredConsents removes expired consent records (for GDPR compliance)
func (s *GmailConsentService) CleanupExpiredConsents() error {
	// Update users where consent has expired
	// This is a placeholder - in production you'd have more sophisticated logic
	return s.db.Model(&models.User{}).
		Where("gmail_consent_date < ?", time.Now().AddDate(-2, 0, 0)). // 2 years ago
		Updates(map[string]interface{}{
			"gmail_consent":      false,
			"gmail_sync_enabled": false,
			"updated_at":        time.Now(),
		}).Error
}