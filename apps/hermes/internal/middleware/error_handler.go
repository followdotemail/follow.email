package middleware

import (
	"fmt"
	"log"
	"net/http"
	"runtime/debug"
	"time"

	"github.com/gin-gonic/gin"
)

// ErrorResponse represents a standardized error response
type ErrorResponse struct {
	Error     string                 `json:"error"`
	Message   string                 `json:"message"`
	Code      string                 `json:"code,omitempty"`
	Details   map[string]interface{} `json:"details,omitempty"`
	Timestamp time.Time              `json:"timestamp"`
	RequestID string                 `json:"request_id,omitempty"`
}

// ErrorHandler middleware for centralized error handling
func ErrorHandler() gin.HandlerFunc {
	return gin.CustomRecovery(func(c *gin.Context, recovered interface{}) {
		if err, ok := recovered.(string); ok {
			log.Printf("Panic recovered: %s\n%s", err, debug.Stack())
			c.JSON(http.StatusInternalServerError, ErrorResponse{
				Error:     "Internal Server Error",
				Message:   "An unexpected error occurred",
				Code:      "INTERNAL_ERROR",
				Timestamp: time.Now(),
				RequestID: getRequestID(c),
			})
		}
		if err, ok := recovered.(error); ok {
			log.Printf("Panic recovered: %v\n%s", err, debug.Stack())
			c.JSON(http.StatusInternalServerError, ErrorResponse{
				Error:     "Internal Server Error",
				Message:   err.Error(),
				Code:      "INTERNAL_ERROR",
				Timestamp: time.Now(),
				RequestID: getRequestID(c),
			})
		}
		c.AbortWithStatus(http.StatusInternalServerError)
	})
}

// APIError represents a structured API error
type APIError struct {
	StatusCode int
	Code       string
	Message    string
	Details    map[string]interface{}
}

func (e *APIError) Error() string {
	return fmt.Sprintf("[%d] %s: %s", e.StatusCode, e.Code, e.Message)
}

// NewAPIError creates a new API error
func NewAPIError(statusCode int, code, message string) *APIError {
	return &APIError{
		StatusCode: statusCode,
		Code:       code,
		Message:    message,
		Details:    make(map[string]interface{}),
	}
}

// WithDetails adds details to the API error
func (e *APIError) WithDetails(key string, value interface{}) *APIError {
	e.Details[key] = value
	return e
}

// Common API errors
var (
	ErrBadRequest          = NewAPIError(http.StatusBadRequest, "BAD_REQUEST", "Invalid request")
	ErrUnauthorized        = NewAPIError(http.StatusUnauthorized, "UNAUTHORIZED", "Authentication required")
	ErrForbidden           = NewAPIError(http.StatusForbidden, "FORBIDDEN", "Access denied")
	ErrNotFound            = NewAPIError(http.StatusNotFound, "NOT_FOUND", "Resource not found")
	ErrConflict            = NewAPIError(http.StatusConflict, "CONFLICT", "Resource conflict")
	ErrTooManyRequests     = NewAPIError(http.StatusTooManyRequests, "RATE_LIMITED", "Too many requests")
	ErrInternalServer      = NewAPIError(http.StatusInternalServerError, "INTERNAL_ERROR", "Internal server error")
	ErrServiceUnavailable  = NewAPIError(http.StatusServiceUnavailable, "SERVICE_UNAVAILABLE", "Service temporarily unavailable")
	ErrGatewayTimeout      = NewAPIError(http.StatusGatewayTimeout, "GATEWAY_TIMEOUT", "Gateway timeout")
)

// OAuth specific errors
var (
	ErrInvalidToken        = NewAPIError(http.StatusUnauthorized, "INVALID_TOKEN", "Invalid or expired token")
	ErrTokenExpired        = NewAPIError(http.StatusUnauthorized, "TOKEN_EXPIRED", "Token has expired")
	ErrInvalidScope        = NewAPIError(http.StatusForbidden, "INVALID_SCOPE", "Insufficient token scope")
	ErrProviderError       = NewAPIError(http.StatusBadGateway, "PROVIDER_ERROR", "OAuth provider error")
)

// Email sync specific errors
var (
	ErrSyncInProgress      = NewAPIError(http.StatusConflict, "SYNC_IN_PROGRESS", "Email sync already in progress")
	ErrSyncFailed          = NewAPIError(http.StatusInternalServerError, "SYNC_FAILED", "Email synchronization failed")
	ErrInvalidProvider     = NewAPIError(http.StatusBadRequest, "INVALID_PROVIDER", "Unsupported email provider")
	ErrQuotaExceeded       = NewAPIError(http.StatusTooManyRequests, "QUOTA_EXCEEDED", "API quota exceeded")
)

// AI service specific errors
var (
	ErrAIServiceUnavailable = NewAPIError(http.StatusServiceUnavailable, "AI_SERVICE_UNAVAILABLE", "AI service is currently unavailable")
	ErrAIAnalysisFailed     = NewAPIError(http.StatusInternalServerError, "AI_ANALYSIS_FAILED", "AI analysis failed")
	ErrInvalidPrompt        = NewAPIError(http.StatusBadRequest, "INVALID_PROMPT", "Invalid AI prompt")
)

// Storage specific errors
var (
	ErrStorageUnavailable   = NewAPIError(http.StatusServiceUnavailable, "STORAGE_UNAVAILABLE", "Storage service unavailable")
	ErrStorageQuotaExceeded = NewAPIError(http.StatusInsufficientStorage, "STORAGE_QUOTA_EXCEEDED", "Storage quota exceeded")
	ErrFileNotFound         = NewAPIError(http.StatusNotFound, "FILE_NOT_FOUND", "File not found in storage")
)

// Queue specific errors
var (
	ErrQueueUnavailable     = NewAPIError(http.StatusServiceUnavailable, "QUEUE_UNAVAILABLE", "Message queue unavailable")
	ErrQueueFull            = NewAPIError(http.StatusServiceUnavailable, "QUEUE_FULL", "Message queue is full")
	ErrInvalidMessage       = NewAPIError(http.StatusBadRequest, "INVALID_MESSAGE", "Invalid queue message")
)

// HandleError processes and responds with appropriate error
func HandleError(c *gin.Context, err error) {
	if apiErr, ok := err.(*APIError); ok {
		c.JSON(apiErr.StatusCode, ErrorResponse{
			Error:     apiErr.Code,
			Message:   apiErr.Message,
			Code:      apiErr.Code,
			Details:   apiErr.Details,
			Timestamp: time.Now(),
			RequestID: getRequestID(c),
		})
		return
	}

	// Log unexpected errors
	log.Printf("Unexpected error: %v", err)
	c.JSON(http.StatusInternalServerError, ErrorResponse{
		Error:     "INTERNAL_ERROR",
		Message:   "An unexpected error occurred",
		Code:      "INTERNAL_ERROR",
		Timestamp: time.Now(),
		RequestID: getRequestID(c),
	})
}

// getRequestID extracts or generates a request ID
func getRequestID(c *gin.Context) string {
	if requestID := c.GetHeader("X-Request-ID"); requestID != "" {
		return requestID
	}
	if requestID, exists := c.Get("request_id"); exists {
		if id, ok := requestID.(string); ok {
			return id
		}
	}
	return ""
}