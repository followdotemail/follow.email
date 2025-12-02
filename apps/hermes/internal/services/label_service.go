package services

import (
	"context"
	"fmt"

	"follow-email-backend/internal/models"
	"follow-email-backend/pkg/oauth"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type LabelService struct {
	db                *gorm.DB
	gmailOAuthService *oauth.GmailOAuthService
	tokenService      *GmailTokenService
}

func NewLabelService(db *gorm.DB, gmailOAuthService *oauth.GmailOAuthService, tokenService *GmailTokenService) *LabelService {
	return &LabelService{
		db:                db,
		gmailOAuthService: gmailOAuthService,
		tokenService:      tokenService,
	}
}

// SyncUserLabels fetches all user labels from Gmail and syncs to database
func (s *LabelService) SyncUserLabels(ctx context.Context, userID uuid.UUID) error {
	// Get Gmail service
	tokenInfo, err := s.tokenService.GetValidToken(ctx, userID)
	if err != nil {
		return fmt.Errorf("failed to get Gmail token: %w", err)
	}

	service, err := s.gmailOAuthService.CreateGmailService(ctx, tokenInfo)
	if err != nil {
		return fmt.Errorf("failed to create Gmail service: %w", err)
	}

	// Fetch labels from Gmail
	labelsResponse, err := service.Users.Labels.List("me").Context(ctx).Do()
	if err != nil {
		return fmt.Errorf("failed to list Gmail labels: %w", err)
	}

	// Sync each user label
	for _, label := range labelsResponse.Labels {
		// Skip system labels
		if label.Type == "system" {
			continue
		}

		// Prepare color JSON
		var labelColor models.LabelColor
		if label.Color != nil {
			labelColor = models.LabelColor{
				BackgroundColor: label.Color.BackgroundColor,
				TextColor:       label.Color.TextColor,
			}
		}

		// Upsert label
		userLabel := models.UserLabel{
			UserID:                userID,
			GmailLabelID:          label.Id,
			LabelName:             label.Name,
			MessageListVisibility: label.MessageListVisibility,
			LabelListVisibility:   label.LabelListVisibility,
			Color:                 labelColor,
		}

		err := s.db.Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "user_id"}, {Name: "gmail_label_id"}},
			DoUpdates: clause.AssignmentColumns([]string{"name", "color", "updated_at"}),
		}).Create(&userLabel).Error

		if err != nil {
			return fmt.Errorf("failed to upsert label %s: %w", label.Name, err)
		}
	}

	return nil
}

// GetUserLabels returns all labels for a user
func (s *LabelService) GetUserLabels(ctx context.Context, userID uuid.UUID) ([]models.UserLabel, error) {
	var labels []models.UserLabel
	err := s.db.Where("user_id = ?", userID).Find(&labels).Error
	return labels, err
}

// GetLabelsByGmailIDs returns labels by their Gmail IDs for a user
func (s *LabelService) GetLabelsByGmailIDs(ctx context.Context, userID uuid.UUID, gmailLabelIDs []string) ([]models.UserLabel, error) {
	var labels []models.UserLabel
	err := s.db.Where("user_id = ? AND gmail_label_id IN ?", userID, gmailLabelIDs).Find(&labels).Error
	return labels, err
}
