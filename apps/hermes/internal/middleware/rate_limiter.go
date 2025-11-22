package middleware

import (
	"fmt"
	"strconv"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

// RateLimiter represents a token bucket rate limiter
type RateLimiter struct {
	tokens    int64
	maxTokens int64
	refillRate int64 // tokens per second
	lastRefill time.Time
	mu        sync.Mutex
}

// NewRateLimiter creates a new rate limiter
func NewRateLimiter(maxTokens, refillRate int64) *RateLimiter {
	return &RateLimiter{
		tokens:     maxTokens,
		maxTokens:  maxTokens,
		refillRate: refillRate,
		lastRefill: time.Now(),
	}
}

// Allow checks if a request is allowed and consumes a token
func (rl *RateLimiter) Allow() bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	elapsed := now.Sub(rl.lastRefill).Seconds()

	// Refill tokens based on elapsed time
	tokensToAdd := int64(elapsed * float64(rl.refillRate))
	rl.tokens = min(rl.maxTokens, rl.tokens+tokensToAdd)
	rl.lastRefill = now

	// Check if we have tokens available
	if rl.tokens > 0 {
		rl.tokens--
		return true
	}

	return false
}

// GetTokens returns the current number of tokens
func (rl *RateLimiter) GetTokens() int64 {
	rl.mu.Lock()
	defer rl.mu.Unlock()
	return rl.tokens
}

// RateLimiterStore manages rate limiters for different keys
type RateLimiterStore struct {
	limiters map[string]*RateLimiter
	mu       sync.RWMutex
	cleanup  time.Duration
}

// NewRateLimiterStore creates a new rate limiter store
func NewRateLimiterStore(cleanupInterval time.Duration) *RateLimiterStore {
	store := &RateLimiterStore{
		limiters: make(map[string]*RateLimiter),
		cleanup:  cleanupInterval,
	}

	// Start cleanup goroutine
	go store.cleanupRoutine()

	return store
}

// GetLimiter gets or creates a rate limiter for a key
func (s *RateLimiterStore) GetLimiter(key string, maxTokens, refillRate int64) *RateLimiter {
	s.mu.Lock()
	defer s.mu.Unlock()

	if limiter, exists := s.limiters[key]; exists {
		return limiter
	}

	limiter := NewRateLimiter(maxTokens, refillRate)
	s.limiters[key] = limiter
	return limiter
}

// cleanupRoutine removes inactive rate limiters
func (s *RateLimiterStore) cleanupRoutine() {
	ticker := time.NewTicker(s.cleanup)
	defer ticker.Stop()

	for range ticker.C {
		s.mu.Lock()
		for key, limiter := range s.limiters {
			// Remove limiters that haven't been used recently
			if time.Since(limiter.lastRefill) > s.cleanup*2 {
				delete(s.limiters, key)
			}
		}
		s.mu.Unlock()
	}
}

// RateLimitConfig represents rate limiting configuration
type RateLimitConfig struct {
	MaxRequests   int64         // Maximum requests
	Window        time.Duration // Time window
	KeyFunc       func(*gin.Context) string // Function to generate rate limit key
	SkipFunc      func(*gin.Context) bool   // Function to skip rate limiting
	ErrorHandler  func(*gin.Context)        // Custom error handler
}

// Global rate limiter store
var globalStore = NewRateLimiterStore(5 * time.Minute)

// RateLimit creates a rate limiting middleware
func RateLimit(config RateLimitConfig) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Skip rate limiting if skip function returns true
		if config.SkipFunc != nil && config.SkipFunc(c) {
			c.Next()
			return
		}

		// Generate rate limit key
		key := config.KeyFunc(c)
		if key == "" {
			c.Next()
			return
		}

		// Calculate refill rate (requests per second)
		refillRate := config.MaxRequests * int64(time.Second) / int64(config.Window)
		if refillRate == 0 {
			refillRate = 1
		}

		// Get or create rate limiter
		limiter := globalStore.GetLimiter(key, config.MaxRequests, refillRate)

		// Check if request is allowed
		if !limiter.Allow() {
			// Set rate limit headers
			c.Header("X-RateLimit-Limit", strconv.FormatInt(config.MaxRequests, 10))
			c.Header("X-RateLimit-Remaining", "0")
			c.Header("X-RateLimit-Reset", strconv.FormatInt(time.Now().Add(config.Window).Unix(), 10))

			// Use custom error handler or default
			if config.ErrorHandler != nil {
				config.ErrorHandler(c)
			} else {
				HandleError(c, ErrTooManyRequests)
			}
			c.Abort()
			return
		}

		// Set rate limit headers for successful requests
		c.Header("X-RateLimit-Limit", strconv.FormatInt(config.MaxRequests, 10))
		c.Header("X-RateLimit-Remaining", strconv.FormatInt(limiter.GetTokens(), 10))
		c.Header("X-RateLimit-Reset", strconv.FormatInt(time.Now().Add(config.Window).Unix(), 10))

		c.Next()
	}
}

// Common rate limit key functions

// ByIP creates a rate limit key based on client IP
func ByIP(c *gin.Context) string {
	return fmt.Sprintf("ip:%s", c.ClientIP())
}

// ByUserID creates a rate limit key based on user ID
func ByUserID(c *gin.Context) string {
	userID, exists := c.Get("user_id")
	if !exists {
		return ""
	}
	return fmt.Sprintf("user:%v", userID)
}

// ByIPAndEndpoint creates a rate limit key based on IP and endpoint
func ByIPAndEndpoint(c *gin.Context) string {
	return fmt.Sprintf("ip:%s:endpoint:%s", c.ClientIP(), c.FullPath())
}

// ByUserAndEndpoint creates a rate limit key based on user ID and endpoint
func ByUserAndEndpoint(c *gin.Context) string {
	userID, exists := c.Get("user_id")
	if !exists {
		return ByIPAndEndpoint(c)
	}
	return fmt.Sprintf("user:%v:endpoint:%s", userID, c.FullPath())
}

// Predefined rate limit configurations

// GlobalRateLimit applies to all requests (by IP)
func GlobalRateLimit() gin.HandlerFunc {
	return RateLimit(RateLimitConfig{
		MaxRequests: 1000,
		Window:      time.Hour,
		KeyFunc:     ByIP,
	})
}

// AuthRateLimit applies to authentication endpoints
func AuthRateLimit() gin.HandlerFunc {
	return RateLimit(RateLimitConfig{
		MaxRequests: 10,
		Window:      time.Minute,
		KeyFunc:     ByIP,
	})
}

// EmailSyncRateLimit applies to email sync endpoints
func EmailSyncRateLimit() gin.HandlerFunc {
	return RateLimit(RateLimitConfig{
		MaxRequests: 5,
		Window:      time.Minute,
		KeyFunc:     ByUserAndEndpoint,
	})
}

// AIAnalysisRateLimit applies to AI analysis endpoints
func AIAnalysisRateLimit() gin.HandlerFunc {
	return RateLimit(RateLimitConfig{
		MaxRequests: 20,
		Window:      time.Minute,
		KeyFunc:     ByUserAndEndpoint,
	})
}

// Helper function for min
func min(a, b int64) int64 {
	if a < b {
		return a
	}
	return b
}