package services

import (
	"fmt"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"follow-email-backend/internal/models"
)

// SubscriptionService handles subscription-related operations
type SubscriptionService struct {
	db *gorm.DB
}

// NewSubscriptionService creates a new subscription service
func NewSubscriptionService(db *gorm.DB) *SubscriptionService {
	return &SubscriptionService{
		db: db,
	}
}

// SubscriptionRequest represents a subscription creation/update request
type SubscriptionRequest struct {
	UserID           int     `json:"user_id"`
	Tier             string  `json:"tier"`
	Status           string  `json:"status"`
	BillingCycle     string  `json:"billing_cycle"`
	Amount           float64 `json:"amount"`
	Currency         string  `json:"currency"`
	StripeCustomerID string  `json:"stripe_customer_id,omitempty"`
	StripeSubID      string  `json:"stripe_subscription_id,omitempty"`
	StripePriceID    string  `json:"stripe_price_id,omitempty"`
}

// CreateSubscription creates a new subscription for a user
func (s *SubscriptionService) CreateSubscription(req *SubscriptionRequest) (*models.UserSubscription, error) {
	userUUID, err := uuid.Parse(fmt.Sprintf("%d", req.UserID))
	if err != nil {
		return nil, fmt.Errorf("invalid user ID: %v", err)
	}

	subscription := &models.UserSubscription{
		UserID:       userUUID,
		Tier:         models.SubscriptionTier(req.Tier),
		Status:       models.SubscriptionStatus(req.Status),
		BillingCycle: req.BillingCycle,
		Amount:       req.Amount,
		Currency:     req.Currency,
		StartDate:    time.Now(),
		AutoRenew:    true,
	}

	// Set limits based on tier
	switch models.SubscriptionTier(req.Tier) {
	case models.SubscriptionTierFree:
		subscription.EmailsLimit = 100
		subscription.APICallsLimit = 1000
	case models.SubscriptionTierPro:
		subscription.EmailsLimit = 5000
		subscription.APICallsLimit = 10000
	case models.SubscriptionTierEnterprise:
		subscription.EmailsLimit = -1 // Unlimited
		subscription.APICallsLimit = -1 // Unlimited
	}

	// Set Stripe information if provided
	if req.StripeCustomerID != "" {
		subscription.StripeCustomerID = &req.StripeCustomerID
	}
	if req.StripeSubID != "" {
		subscription.StripeSubscriptionID = &req.StripeSubID
	}
	if req.StripePriceID != "" {
		subscription.StripePriceID = &req.StripePriceID
	}

	if err := s.db.Create(subscription).Error; err != nil {
		return nil, fmt.Errorf("failed to create subscription: %v", err)
	}

	return subscription, nil
}

// GetSubscription retrieves a user's subscription
func (s *SubscriptionService) GetSubscription(userID int) (*models.UserSubscription, error) {
	userUUID, err := uuid.Parse(fmt.Sprintf("%d", userID))
	if err != nil {
		return nil, fmt.Errorf("invalid user ID: %v", err)
	}

	var subscription models.UserSubscription
	if err := s.db.Where("user_id = ?", userUUID).First(&subscription).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("subscription not found for user %d", userID)
		}
		return nil, fmt.Errorf("failed to get subscription: %v", err)
	}

	return &subscription, nil
}

// UpdateSubscription updates an existing subscription
func (s *SubscriptionService) UpdateSubscription(userID int, req *SubscriptionRequest) (*models.UserSubscription, error) {
	subscription, err := s.GetSubscription(userID)
	if err != nil {
		return nil, err
	}

	// Update fields
	subscription.Tier = models.SubscriptionTier(req.Tier)
	subscription.Status = models.SubscriptionStatus(req.Status)
	subscription.BillingCycle = req.BillingCycle
	subscription.Amount = req.Amount
	subscription.Currency = req.Currency

	// Update limits based on new tier
	switch models.SubscriptionTier(req.Tier) {
	case models.SubscriptionTierFree:
		subscription.EmailsLimit = 100
		subscription.APICallsLimit = 1000
	case models.SubscriptionTierPro:
		subscription.EmailsLimit = 5000
		subscription.APICallsLimit = 10000
	case models.SubscriptionTierEnterprise:
		subscription.EmailsLimit = -1 // Unlimited
		subscription.APICallsLimit = -1 // Unlimited
	}

	// Update Stripe information if provided
	if req.StripeCustomerID != "" {
		subscription.StripeCustomerID = &req.StripeCustomerID
	}
	if req.StripeSubID != "" {
		subscription.StripeSubscriptionID = &req.StripeSubID
	}
	if req.StripePriceID != "" {
		subscription.StripePriceID = &req.StripePriceID
	}

	if err := s.db.Save(subscription).Error; err != nil {
		return nil, fmt.Errorf("failed to update subscription: %v", err)
	}

	return subscription, nil
}

// CancelSubscription cancels a user's subscription
func (s *SubscriptionService) CancelSubscription(userID int, reason string) error {
	subscription, err := s.GetSubscription(userID)
	if err != nil {
		return err
	}

	now := time.Now()
	subscription.Status = models.SubscriptionStatusCanceled
	subscription.CanceledAt = &now
	subscription.CancellationReason = &reason
	subscription.AutoRenew = false

	if err := s.db.Save(subscription).Error; err != nil {
		return fmt.Errorf("failed to cancel subscription: %v", err)
	}

	return nil
}

// ReactivateSubscription reactivates a canceled subscription
func (s *SubscriptionService) ReactivateSubscription(userID int) error {
	subscription, err := s.GetSubscription(userID)
	if err != nil {
		return err
	}

	subscription.Status = models.SubscriptionStatusActive
	subscription.CanceledAt = nil
	subscription.CancellationReason = nil
	subscription.AutoRenew = true

	if err := s.db.Save(subscription).Error; err != nil {
		return fmt.Errorf("failed to reactivate subscription: %v", err)
	}

	return nil
}

// UpdateUsage updates the usage counters for a subscription
func (s *SubscriptionService) UpdateUsage(userID int, emailsProcessed int, apiCallsUsed int) error {
	subscription, err := s.GetSubscription(userID)
	if err != nil {
		return err
	}

	subscription.EmailsProcessed += emailsProcessed
	subscription.APICallsUsed += apiCallsUsed

	if err := s.db.Save(subscription).Error; err != nil {
		return fmt.Errorf("failed to update usage: %v", err)
	}

	return nil
}

// CheckUsageLimits checks if a user has exceeded their usage limits
func (s *SubscriptionService) CheckUsageLimits(userID int) (bool, error) {
	subscription, err := s.GetSubscription(userID)
	if err != nil {
		return false, err
	}

	// Check if limits are unlimited (-1)
	if subscription.EmailsLimit == -1 && subscription.APICallsLimit == -1 {
		return true, nil
	}

	// Check email limit
	if subscription.EmailsLimit > 0 && subscription.EmailsProcessed >= subscription.EmailsLimit {
		return false, nil
	}

	// Check API calls limit
	if subscription.APICallsLimit > 0 && subscription.APICallsUsed >= subscription.APICallsLimit {
		return false, nil
	}

	return true, nil
}