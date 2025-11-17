package services

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"follow-email-backend/config"
	"follow-email-backend/internal/models"
	"github.com/clerk/clerk-sdk-go/v2"
	"github.com/clerk/clerk-sdk-go/v2/user"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// ClerkAuthService handles Clerk JWT verification and user management
type ClerkAuthService struct {
	config *config.Config
	db     *gorm.DB
}

// UserAuthResult contains the result of user authentication and creation
type UserAuthResult struct {
	User         *models.User `json:"user"`
	ClerkUser    *clerk.User  `json:"clerk_user"`
	IsNewUser    bool         `json:"is_new_user"`
	ClerkID      string       `json:"clerk_id"`
}

// NewClerkAuthService creates a new Clerk authentication service
func NewClerkAuthService(cfg *config.Config, db *gorm.DB) *ClerkAuthService {
	// Initialize Clerk with secret key
	clerk.SetKey(cfg.ClerkSecretKey)
	return &ClerkAuthService{
		config: cfg,
		db:     db,
	}
}

// VerifyJWTAndGetUser verifies JWT token and returns user, creating if necessary
func (s *ClerkAuthService) VerifyJWTAndGetUser(c *gin.Context) (*UserAuthResult, error) {
	// Extract JWT token from Authorization header
	authHeader := c.GetHeader("Authorization")
	if authHeader == "" {
		return nil, errors.New("authorization header is required")
	}

	// Check if header starts with "Bearer "
	if !strings.HasPrefix(authHeader, "Bearer ") {
		return nil, errors.New("authorization header must start with 'Bearer '")
	}

	// Extract token
	token := strings.TrimPrefix(authHeader, "Bearer ")
	if token == "" {
		return nil, errors.New("JWT token is required")
	}

	// Verify token with Clerk and get user ID
	// Note: In a real implementation, you would verify the JWT token
	// For now, we'll use the middleware approach but enhance it
	userID, exists := c.Get("user_id")
	if !exists {
		return nil, errors.New("user not authenticated - invalid or expired token")
	}

	clerkUserID := userID.(string)

	// Get user details from Clerk
	clerkUser, err := user.Get(context.Background(), clerkUserID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user from Clerk: %w", err)
	}

	// Check if user exists in our database
	var dbUser models.User
	result := s.db.Where("clerk_id = ?", clerkUserID).First(&dbUser)
	
	if result.Error != nil {
		if result.Error == gorm.ErrRecordNotFound {
			// User doesn't exist, create new user
			newUser, err := s.createUserFromClerk(clerkUser)
			if err != nil {
				return nil, fmt.Errorf("failed to create user: %w", err)
			}
			
			return &UserAuthResult{
				User:      newUser,
				ClerkUser: clerkUser,
				IsNewUser: true,
				ClerkID:   clerkUserID,
			}, nil
		}
		
		return nil, fmt.Errorf("database error: %w", result.Error)
	}

	// User exists, update with latest Clerk data if needed
	updatedUser := s.updateUserFromClerk(&dbUser, clerkUser)
	
	return &UserAuthResult{
		User:      updatedUser,
		ClerkUser: clerkUser,
		IsNewUser: false,
		ClerkID:   clerkUserID,
	}, nil
}

// createUserFromClerk creates a new user in the database from Clerk user data
func (s *ClerkAuthService) createUserFromClerk(clerkUser *clerk.User) (*models.User, error) {
	// Convert Clerk user to our user model
	newUser := convertClerkUserToAppUser(clerkUser)
	
	// Save to database
	if err := s.db.Create(newUser).Error; err != nil {
		return nil, fmt.Errorf("failed to save user to database: %w", err)
	}
	
	return newUser, nil
}

// updateUserFromClerk updates existing user with latest Clerk data
func (s *ClerkAuthService) updateUserFromClerk(dbUser *models.User, clerkUser *clerk.User) *models.User {
	// Update fields that might have changed in Clerk
	if clerkUser.FirstName != nil {
		dbUser.FirstName = *clerkUser.FirstName
	}
	if clerkUser.LastName != nil {
		dbUser.LastName = *clerkUser.LastName
	}
	if clerkUser.ImageURL != nil {
		dbUser.ImageURL = clerkUser.ImageURL
	}
	
	// Update email if it has changed
	if len(clerkUser.EmailAddresses) > 0 {
		primaryEmail := clerkUser.EmailAddresses[0].EmailAddress
		if dbUser.Email != primaryEmail {
			dbUser.Email = primaryEmail
		}
	}
	
	// Update name
	var name string
	firstName := ""
	lastName := ""
	if clerkUser.FirstName != nil {
		firstName = *clerkUser.FirstName
	}
	if clerkUser.LastName != nil {
		lastName = *clerkUser.LastName
	}
	
	if firstName != "" && lastName != "" {
		name = firstName + " " + lastName
	} else if firstName != "" {
		name = firstName
	} else if lastName != "" {
		name = lastName
	}
	dbUser.Name = name
	
	// Save updates
	s.db.Save(dbUser)
	
	return dbUser
}

// convertClerkUserToAppUser converts a Clerk user to our application user model
func convertClerkUserToAppUser(clerkUser *clerk.User) *models.User {
	var email string
	if len(clerkUser.EmailAddresses) > 0 {
		email = clerkUser.EmailAddresses[0].EmailAddress
	}

	var name string
	var firstName, lastName string
	if clerkUser.FirstName != nil {
		firstName = *clerkUser.FirstName
	}
	if clerkUser.LastName != nil {
		lastName = *clerkUser.LastName
	}

	if firstName != "" && lastName != "" {
		name = firstName + " " + lastName
	} else if firstName != "" {
		name = firstName
	} else if lastName != "" {
		name = lastName
	}

	// Determine provider from external accounts
	var provider string
	if len(clerkUser.ExternalAccounts) > 0 {
		provider = clerkUser.ExternalAccounts[0].Provider
	} else {
		provider = "clerk"
	}

	return &models.User{
		ClerkID:   &clerkUser.ID,
		Email:     email,
		Name:      name,
		FirstName: firstName,
		LastName:  lastName,
		Username:  clerkUser.Username,
		ImageURL:  clerkUser.ImageURL,
		Provider:  provider,
		IsActive:  true,
	}
}