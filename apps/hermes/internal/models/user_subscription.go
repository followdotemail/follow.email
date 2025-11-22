package models

import (
	"time"
	"github.com/google/uuid"
)

// SubscriptionTier represents the available subscription tiers
type SubscriptionTier string

const (
	SubscriptionTierFree       SubscriptionTier = "free"
	SubscriptionTierPro        SubscriptionTier = "pro"
	SubscriptionTierEnterprise SubscriptionTier = "enterprise"
)

// SubscriptionStatus represents the current status of a subscription
type SubscriptionStatus string

const (
	SubscriptionStatusActive    SubscriptionStatus = "active"
	SubscriptionStatusCanceled  SubscriptionStatus = "canceled"
	SubscriptionStatusExpired   SubscriptionStatus = "expired"
	SubscriptionStatusPending   SubscriptionStatus = "pending"
	SubscriptionStatusSuspended SubscriptionStatus = "suspended"
)

// UserSubscription represents subscription and billing information for a user
type UserSubscription struct {
	ID                    uuid.UUID          `json:"id" gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	UserID                uuid.UUID          `json:"user_id" gorm:"type:uuid;unique;not null"`
	
	// Subscription Details
	Tier                  SubscriptionTier   `json:"tier" gorm:"default:'free'"`
	Status                SubscriptionStatus `json:"status" gorm:"default:'active'"`
	
	// Billing Cycle
	StartDate             time.Time          `json:"start_date"`
	EndDate               *time.Time         `json:"end_date"`
	NextBillingDate       *time.Time         `json:"next_billing_date"`
	BillingCycle          string             `json:"billing_cycle"` // "monthly", "yearly", "lifetime"
	
	// Pricing
	Amount                float64            `json:"amount" gorm:"type:decimal(10,2)"`
	Currency              string             `json:"currency" gorm:"default:'USD'"`
	
	// External Integration
	StripeCustomerID      *string            `json:"stripe_customer_id"`
	StripeSubscriptionID  *string            `json:"stripe_subscription_id"`
	StripePriceID         *string            `json:"stripe_price_id"`
	
	// Usage Tracking
	EmailsProcessed       int                `json:"emails_processed" gorm:"default:0"`
	EmailsLimit           int                `json:"emails_limit" gorm:"default:100"` // Monthly limit
	APICallsUsed          int                `json:"api_calls_used" gorm:"default:0"`
	APICallsLimit         int                `json:"api_calls_limit" gorm:"default:1000"` // Monthly limit
	
	// Trial Information
	IsTrialActive         bool               `json:"is_trial_active" gorm:"default:false"`
	TrialStartDate        *time.Time         `json:"trial_start_date"`
	TrialEndDate          *time.Time         `json:"trial_end_date"`
	
	// Cancellation
	CanceledAt            *time.Time         `json:"canceled_at"`
	CancellationReason    *string            `json:"cancellation_reason"`
	CanceledBy            *string            `json:"canceled_by"` // User ID or "system"
	
	// Auto-renewal
	AutoRenew             bool               `json:"auto_renew" gorm:"default:true"`
	
	CreatedAt             time.Time          `json:"created_at"`
	UpdatedAt             time.Time          `json:"updated_at"`
	
	// Relationships
	User                  User               `gorm:"foreignKey:UserID"`
}

func (UserSubscription) TableName() string {
	return "user_subscriptions"
}