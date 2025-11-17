package models

import (
	"log"
	"strings"
	"time"

	"follow-email-backend/pkg/encryption"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type User struct {
	ID        uuid.UUID `json:"id" gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	ClerkID   *string   `json:"clerk_id" gorm:"uniqueIndex"`
	Email     string    `json:"email" gorm:"uniqueIndex;not null"`
	Name      string    `json:"name"`
	FirstName string    `json:"first_name"`
	LastName  string    `json:"last_name"`
	Username  *string   `json:"username" gorm:"uniqueIndex"`
	ImageURL  *string   `json:"image_url"`
	Provider  string    `json:"provider"` // "google", "github", "linkedin", "clerk"
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	IsActive  bool      `json:"is_active" gorm:"default:true"`
}

// ExternalAccount represents social login connections managed by Clerk
type ExternalAccount struct {
	ID                 uuid.UUID `json:"id" gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	UserID             uuid.UUID `json:"user_id" gorm:"type:uuid;not null;index"`
	ClerkExternalID    string    `json:"clerk_external_id" gorm:"uniqueIndex;not null"`
	Provider           string    `json:"provider" gorm:"not null"` // "oauth_google", "oauth_github", "oauth_facebook"
	ProviderUserID     string    `json:"provider_user_id"`
	EmailAddress       string    `json:"email_address"`
	FirstName          *string   `json:"first_name"`
	LastName           *string   `json:"last_name"`
	ImageURL           *string   `json:"image_url"`
	Username           *string   `json:"username"`
	PublicMetadata     *string   `json:"public_metadata" gorm:"type:jsonb"`
	Label              *string   `json:"label"`
	VerificationStatus string    `json:"verification_status" gorm:"default:'verified'"`
	CreatedAt          time.Time `json:"created_at"`
	UpdatedAt          time.Time `json:"updated_at"`
	User               User      `gorm:"foreignKey:UserID"`
}

// OAuthToken - Legacy model, kept for backward compatibility
type OAuthToken struct {
	ID           uuid.UUID `json:"id" gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	UserID       uuid.UUID `json:"user_id" gorm:"type:uuid;not null;index"`
	Provider     string    `json:"provider" gorm:"not null"`
	AccessToken  string    `json:"-" gorm:"not null"` // Hidden from JSON
	RefreshToken string    `json:"-"`                 // Hidden from JSON
	TokenType    string    `json:"token_type"`
	ExpiresAt    time.Time `json:"expires_at"`
	Scope        string    `json:"scope"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
	User         User      `gorm:"foreignKey:UserID"`
}

// Global encryption service instance
var encryptionService *encryption.EncryptionService

// SetEncryptionService sets the global encryption service
func SetEncryptionService(service *encryption.EncryptionService) {
	encryptionService = service
}

// BeforeCreate encrypts sensitive fields before creating the record
func (o *OAuthToken) BeforeCreate(tx *gorm.DB) error {
	if encryptionService != nil {
		var err error
		if o.AccessToken != "" {
			o.AccessToken, err = encryptionService.Encrypt(o.AccessToken)
			if err != nil {
				return err
			}
		}
		if o.RefreshToken != "" {
			o.RefreshToken, err = encryptionService.Encrypt(o.RefreshToken)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

// BeforeUpdate encrypts sensitive fields before updating the record
func (o *OAuthToken) BeforeUpdate(tx *gorm.DB) error {
	if encryptionService != nil {
		var err error
		if o.AccessToken != "" {
			o.AccessToken, err = encryptionService.Encrypt(o.AccessToken)
			if err != nil {
				return err
			}
		}
		if o.RefreshToken != "" {
			o.RefreshToken, err = encryptionService.Encrypt(o.RefreshToken)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

// AfterFind decrypts sensitive fields after finding the record
func (o *OAuthToken) AfterFind(tx *gorm.DB) error {
	if encryptionService != nil {
		if o.AccessToken != "" {
			decrypted, err := encryptionService.Decrypt(o.AccessToken)
			if err != nil {
				if strings.Contains(err.Error(), "failed to decode base64") || strings.Contains(err.Error(), "ciphertext too short") {
					log.Printf("[WARN] OAuthToken access token appears to be stored in plaintext; skipping decryption for token %s", o.ID)
				} else {
					return err
				}
			} else {
				o.AccessToken = decrypted
			}
		}
		if o.RefreshToken != "" {
			decrypted, err := encryptionService.Decrypt(o.RefreshToken)
			if err != nil {
				if strings.Contains(err.Error(), "failed to decode base64") || strings.Contains(err.Error(), "ciphertext too short") {
					log.Printf("[WARN] OAuthToken refresh token appears to be stored in plaintext; skipping decryption for token %s", o.ID)
				} else {
					return err
				}
			} else {
				o.RefreshToken = decrypted
			}
		}
	}
	return nil
}

type UserPreferences struct {
	ID                   uuid.UUID `json:"id" gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	UserID               uuid.UUID `json:"user_id" gorm:"type:uuid;uniqueIndex;not null"`
	AutoFollowUpEnabled  bool      `json:"auto_followup_enabled" gorm:"default:false"`
	FollowUpDelayHours   int       `json:"followup_delay_hours" gorm:"default:24"`
	MaxFollowUpAttempts  int       `json:"max_followup_attempts" gorm:"default:3"`
	AIResponseEnabled    bool      `json:"ai_response_enabled" gorm:"default:true"`
	EmailNotifications   bool      `json:"email_notifications" gorm:"default:true"`
	WebhookNotifications bool      `json:"webhook_notifications" gorm:"default:false"`
	User                 User      `gorm:"foreignKey:UserID"`
	WebhookURL           string    `json:"webhook_url" db:"webhook_url"`
}

// JWT Claims for authentication
type JWTClaims struct {
	UserID   string `json:"user_id"` // Changed to string for Clerk compatibility
	Email    string `json:"email"`
	Provider string `json:"provider"`
}

// TableName methods for database mapping
func (User) TableName() string {
	return "users"
}

func (ExternalAccount) TableName() string {
	return "external_accounts"
}

func (OAuthToken) TableName() string {
	return "oauth_tokens"
}

func (UserPreferences) TableName() string {
	return "user_preferences"
}
