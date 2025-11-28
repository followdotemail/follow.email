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

func (OAuthToken) TableName() string {
	return "oauth_tokens"
}
