package services

import (
	"context"
	"fmt"
	"log"
	"time"

	"follow-email-backend/internal/models"
	"follow-email-backend/pkg/storage"

	"gorm.io/gorm"
)

// PrivacyService handles privacy-related operations
type PrivacyService struct {
	db             *gorm.DB
	storageService *storage.S3Service
}

// NewPrivacyService creates a new privacy service
func NewPrivacyService(db *gorm.DB, storageService *storage.S3Service) *PrivacyService {
	return &PrivacyService{
		db:             db,
		storageService: storageService,
	}
}

// ConsentRequest represents a consent request
type ConsentRequest struct {
	UserID      int    `json:"user_id"`
	GDPRConsent bool   `json:"gdpr_consent"`
	CCPAConsent bool   `json:"ccpa_consent"`
	ConsentType string `json:"consent_type"` // "gdpr", "ccpa", "both"
	IPAddress   string `json:"ip_address"`
	UserAgent   string `json:"user_agent"`
}

// DataExportRequest represents a data export request
type DataExportRequest struct {
	UserID       int      `json:"user_id"`
	DataTypes    []string `json:"data_types"` // "profile", "emails", "analytics", "all"
	Format       string   `json:"format"`     // "json", "csv"
	IncludeFiles bool     `json:"include_files"`
}

// DataDeletionRequest represents a data deletion request
type DataDeletionRequest struct {
	UserID       int      `json:"user_id"`
	DataTypes    []string `json:"data_types"` // "profile", "emails", "analytics", "all"
	Reason       string   `json:"reason"`
	Confirmation bool     `json:"confirmation"`
}

// ConsentRecord represents a consent record
type ConsentRecord struct {
	ID          int       `json:"id" gorm:"primaryKey"`
	UserID      int       `json:"user_id"`
	ConsentType string    `json:"consent_type"`
	Consented   bool      `json:"consented"`
	IPAddress   string    `json:"ip_address"`
	UserAgent   string    `json:"user_agent"`
	CreatedAt   time.Time `json:"created_at"`
}

// DataRequest represents a data request
type DataRequest struct {
	ID          int        `json:"id" gorm:"primaryKey"`
	UserID      int        `json:"user_id"`
	RequestType string     `json:"request_type"` // "export", "deletion", "rectification"
	Status      string     `json:"status"`       // "pending", "processing", "completed", "failed"
	RequestData string     `json:"request_data"` // JSON data
	Response    string     `json:"response"`     // Response data or file path
	CreatedAt   time.Time  `json:"created_at"`
	CompletedAt *time.Time `json:"completed_at"`
}

// RecordConsent records user consent for GDPR/CCPA
func (s *PrivacyService) RecordConsent(ctx context.Context, req *ConsentRequest) error {
	// Update user consent using GORM
	result := s.db.WithContext(ctx).Model(&models.User{}).Where("id = ?", req.UserID).Updates(map[string]interface{}{
		"gdpr_consent": req.GDPRConsent,
		"ccpa_consent": req.CCPAConsent,
		"consent_date": time.Now(),
		"updated_at":   time.Now(),
	})
	if result.Error != nil {
		return fmt.Errorf("failed to update user consent: %w", result.Error)
	}

	// Record consent history
	if req.GDPRConsent {
		err := s.recordConsentHistory(ctx, req.UserID, "gdpr", true, req.IPAddress, req.UserAgent)
		if err != nil {
			log.Printf("Failed to record GDPR consent history: %v", err)
		}
	}

	if req.CCPAConsent {
		err := s.recordConsentHistory(ctx, req.UserID, "ccpa", true, req.IPAddress, req.UserAgent)
		if err != nil {
			log.Printf("Failed to record CCPA consent history: %v", err)
		}
	}

	return nil
}

// recordConsentHistory records consent in the consent_records table
func (s *PrivacyService) recordConsentHistory(ctx context.Context, userID int, consentType string, consented bool, ipAddress, userAgent string) error {
	consentRecord := ConsentRecord{
		UserID:      userID,
		ConsentType: consentType,
		Consented:   consented,
		IPAddress:   ipAddress,
		UserAgent:   userAgent,
		CreatedAt:   time.Now(),
	}
	result := s.db.WithContext(ctx).Create(&consentRecord)
	return result.Error
}

// GetConsentStatus retrieves user consent status
func (s *PrivacyService) GetConsentStatus(ctx context.Context, userID int) (*models.UserPrivacyMetadata, error) {
	var privacyMetadata models.UserPrivacyMetadata
	result := s.db.WithContext(ctx).Where("user_id = ?", userID).First(&privacyMetadata)
	if result.Error != nil {
		return nil, fmt.Errorf("failed to get user consent status: %w", result.Error)
	}
	return &privacyMetadata, nil
}

// RequestDataExport creates a data export request
func (s *PrivacyService) RequestDataExport(ctx context.Context, req *DataExportRequest) (*DataRequest, error) {
	dataRequest := DataRequest{
		UserID:      req.UserID,
		RequestType: "export",
		Status:      "pending",
		RequestData: fmt.Sprintf("{\"data_types\":%v,\"format\":\"%s\",\"include_files\":%t}", req.DataTypes, req.Format, req.IncludeFiles),
		CreatedAt:   time.Now(),
	}

	result := s.db.WithContext(ctx).Create(&dataRequest)
	if result.Error != nil {
		return nil, fmt.Errorf("failed to create data export request: %w", result.Error)
	}

	return &dataRequest, nil
}

// RequestDataDeletion creates a data deletion request
func (s *PrivacyService) RequestDataDeletion(ctx context.Context, req *DataDeletionRequest) (*DataRequest, error) {
	dataRequest := DataRequest{
		UserID:      req.UserID,
		RequestType: "deletion",
		Status:      "pending",
		RequestData: fmt.Sprintf("{\"data_types\":%v,\"reason\":\"%s\",\"confirmation\":%t}", req.DataTypes, req.Reason, req.Confirmation),
		CreatedAt:   time.Now(),
	}

	result := s.db.WithContext(ctx).Create(&dataRequest)
	if result.Error != nil {
		return nil, fmt.Errorf("failed to create data deletion request: %w", result.Error)
	}

	return &dataRequest, nil
}

// GetDataRequests retrieves data requests for a user
func (s *PrivacyService) GetDataRequests(ctx context.Context, userID int) ([]DataRequest, error) {
	var requests []DataRequest
	result := s.db.WithContext(ctx).Where("user_id = ?", userID).Order("created_at DESC").Find(&requests)
	if result.Error != nil {
		return nil, fmt.Errorf("failed to get data requests: %w", result.Error)
	}
	return requests, nil
}
