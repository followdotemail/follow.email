package retry

import (
	"context"
	"fmt"
	"math"
	"math/rand"
	"time"
)

// RetryConfig defines retry behavior configuration
type RetryConfig struct {
	MaxAttempts   int           // Maximum number of retry attempts
	InitialDelay  time.Duration // Initial delay before first retry
	MaxDelay      time.Duration // Maximum delay between retries
	Multiplier    float64       // Backoff multiplier
	Jitter        bool          // Add random jitter to delays
	RetryableFunc func(error) bool // Function to determine if error is retryable
}

// DefaultConfig returns a default retry configuration
func DefaultConfig() *RetryConfig {
	return &RetryConfig{
		MaxAttempts:  3,
		InitialDelay: 100 * time.Millisecond,
		MaxDelay:     30 * time.Second,
		Multiplier:   2.0,
		Jitter:       true,
		RetryableFunc: DefaultRetryableFunc,
	}
}

// EmailSyncConfig returns retry configuration optimized for email sync operations
func EmailSyncConfig() *RetryConfig {
	return &RetryConfig{
		MaxAttempts:  5,
		InitialDelay: 1 * time.Second,
		MaxDelay:     5 * time.Minute,
		Multiplier:   2.0,
		Jitter:       true,
		RetryableFunc: EmailSyncRetryableFunc,
	}
}

// AIServiceConfig returns retry configuration for AI service calls
func AIServiceConfig() *RetryConfig {
	return &RetryConfig{
		MaxAttempts:  3,
		InitialDelay: 500 * time.Millisecond,
		MaxDelay:     10 * time.Second,
		Multiplier:   1.5,
		Jitter:       true,
		RetryableFunc: AIServiceRetryableFunc,
	}
}

// StorageConfig returns retry configuration for storage operations
func StorageConfig() *RetryConfig {
	return &RetryConfig{
		MaxAttempts:  4,
		InitialDelay: 200 * time.Millisecond,
		MaxDelay:     1 * time.Minute,
		Multiplier:   2.0,
		Jitter:       true,
		RetryableFunc: StorageRetryableFunc,
	}
}

// QueueConfig returns retry configuration for queue operations
func QueueConfig() *RetryConfig {
	return &RetryConfig{
		MaxAttempts:  5,
		InitialDelay: 100 * time.Millisecond,
		MaxDelay:     30 * time.Second,
		Multiplier:   2.0,
		Jitter:       true,
		RetryableFunc: QueueRetryableFunc,
	}
}

// RetryFunc represents a function that can be retried
type RetryFunc func() error

// RetryFuncWithResult represents a function that returns a result and can be retried
type RetryFuncWithResult[T any] func() (T, error)

// Do executes a function with retry logic
func Do(ctx context.Context, config *RetryConfig, fn RetryFunc) error {
	var lastErr error

	for attempt := 0; attempt < config.MaxAttempts; attempt++ {
		// Execute the function
		err := fn()
		if err == nil {
			return nil // Success
		}

		lastErr = err

		// Check if error is retryable
		if config.RetryableFunc != nil && !config.RetryableFunc(err) {
			return fmt.Errorf("non-retryable error after %d attempts: %w", attempt+1, err)
		}

		// Don't wait after the last attempt
		if attempt == config.MaxAttempts-1 {
			break
		}

		// Calculate delay with exponential backoff
		delay := calculateDelay(config, attempt)

		// Wait for the delay or context cancellation
		select {
		case <-ctx.Done():
			return fmt.Errorf("retry cancelled by context after %d attempts: %w", attempt+1, ctx.Err())
		case <-time.After(delay):
			// Continue to next attempt
		}
	}

	return fmt.Errorf("max retry attempts (%d) exceeded: %w", config.MaxAttempts, lastErr)
}

// DoWithResult executes a function with retry logic and returns a result
func DoWithResult[T any](ctx context.Context, config *RetryConfig, fn RetryFuncWithResult[T]) (T, error) {
	var lastErr error
	var zeroValue T

	for attempt := 0; attempt < config.MaxAttempts; attempt++ {
		// Execute the function
		result, err := fn()
		if err == nil {
			return result, nil // Success
		}

		lastErr = err

		// Check if error is retryable
		if config.RetryableFunc != nil && !config.RetryableFunc(err) {
			return zeroValue, fmt.Errorf("non-retryable error after %d attempts: %w", attempt+1, err)
		}

		// Don't wait after the last attempt
		if attempt == config.MaxAttempts-1 {
			break
		}

		// Calculate delay with exponential backoff
		delay := calculateDelay(config, attempt)

		// Wait for the delay or context cancellation
		select {
		case <-ctx.Done():
			return zeroValue, fmt.Errorf("retry cancelled by context after %d attempts: %w", attempt+1, ctx.Err())
		case <-time.After(delay):
			// Continue to next attempt
		}
	}

	return zeroValue, fmt.Errorf("max retry attempts (%d) exceeded: %w", config.MaxAttempts, lastErr)
}

// calculateDelay calculates the delay for the next retry attempt
func calculateDelay(config *RetryConfig, attempt int) time.Duration {
	// Calculate exponential backoff delay
	delay := float64(config.InitialDelay) * math.Pow(config.Multiplier, float64(attempt))

	// Apply maximum delay limit
	if time.Duration(delay) > config.MaxDelay {
		delay = float64(config.MaxDelay)
	}

	// Add jitter if enabled
	if config.Jitter {
		// Add random jitter up to 10% of the delay
		jitter := delay * 0.1 * rand.Float64()
		delay += jitter
	}

	return time.Duration(delay)
}

// Common retryable error checking functions

// DefaultRetryableFunc determines if an error is retryable by default
func DefaultRetryableFunc(err error) bool {
	if err == nil {
		return false
	}

	// Add common retryable error patterns
	errorStr := err.Error()
	retryablePatterns := []string{
		"connection refused",
		"connection reset",
		"timeout",
		"temporary failure",
		"service unavailable",
		"too many requests",
		"rate limit",
		"network error",
	}

	for _, pattern := range retryablePatterns {
		if contains(errorStr, pattern) {
			return true
		}
	}

	return false
}

// EmailSyncRetryableFunc determines if an email sync error is retryable
func EmailSyncRetryableFunc(err error) bool {
	if DefaultRetryableFunc(err) {
		return true
	}

	errorStr := err.Error()
	emailSyncRetryablePatterns := []string{
		"quota exceeded",
		"api limit",
		"backend error",
		"internal error",
		"oauth token",
		"authentication",
	}

	for _, pattern := range emailSyncRetryablePatterns {
		if contains(errorStr, pattern) {
			return true
		}
	}

	return false
}

// AIServiceRetryableFunc determines if an AI service error is retryable
func AIServiceRetryableFunc(err error) bool {
	if DefaultRetryableFunc(err) {
		return true
	}

	errorStr := err.Error()
	aiRetryablePatterns := []string{
		"model overloaded",
		"quota exceeded",
		"api key",
		"generation failed",
	}

	for _, pattern := range aiRetryablePatterns {
		if contains(errorStr, pattern) {
			return true
		}
	}

	return false
}

// StorageRetryableFunc determines if a storage error is retryable
func StorageRetryableFunc(err error) bool {
	if DefaultRetryableFunc(err) {
		return true
	}

	errorStr := err.Error()
	storageRetryablePatterns := []string{
		"s3 error",
		"aws error",
		"bucket error",
		"upload failed",
		"download failed",
	}

	for _, pattern := range storageRetryablePatterns {
		if contains(errorStr, pattern) {
			return true
		}
	}

	return false
}

// QueueRetryableFunc determines if a queue error is retryable
func QueueRetryableFunc(err error) bool {
	if DefaultRetryableFunc(err) {
		return true
	}

	errorStr := err.Error()
	queueRetryablePatterns := []string{
		"rabbitmq",
		"amqp",
		"queue full",
		"connection closed",
		"channel closed",
	}

	for _, pattern := range queueRetryablePatterns {
		if contains(errorStr, pattern) {
			return true
		}
	}

	return false
}

// Helper function to check if a string contains a substring (case-insensitive)
func contains(s, substr string) bool {
	return len(s) >= len(substr) && 
		   (s == substr || 
			len(s) > len(substr) && 
			(hasPrefix(s, substr) || hasSuffix(s, substr) || hasInfix(s, substr)))
}

func hasPrefix(s, prefix string) bool {
	return len(s) >= len(prefix) && s[0:len(prefix)] == prefix
}

func hasSuffix(s, suffix string) bool {
	return len(s) >= len(suffix) && s[len(s)-len(suffix):] == suffix
}

func hasInfix(s, infix string) bool {
	for i := 0; i <= len(s)-len(infix); i++ {
		if s[i:i+len(infix)] == infix {
			return true
		}
	}
	return false
}