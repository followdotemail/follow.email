package handlers

import (
	"net/http"
	"strconv"

	"follow-email-backend/internal/services"
	"github.com/gin-gonic/gin"
)

// PrivacyHandler handles privacy and compliance related requests
type PrivacyHandler struct {
	privacyService *services.PrivacyService
}

// NewPrivacyHandler creates a new privacy handler
func NewPrivacyHandler(privacyService *services.PrivacyService) *PrivacyHandler {
	return &PrivacyHandler{
		privacyService: privacyService,
	}
}

// ConsentRequest represents a consent recording request
type ConsentRequest struct {
	ConsentType string                 `json:"consent_type" binding:"required"`
	Granted     bool                   `json:"granted" binding:"required"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

// DataExportRequest represents a data export request
type DataExportRequest struct {
	Reason   string                 `json:"reason,omitempty"`
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}

// DataDeletionRequest represents a data deletion request
type DataDeletionRequest struct {
	Reason   string                 `json:"reason,omitempty"`
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}

// RecordConsent records user consent
func (h *PrivacyHandler) RecordConsent(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	var req ConsentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Convert userID string to int
	userIDStr := userID.(string)
	userIDInt, err := strconv.Atoi(userIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID format"})
		return
	}

	consentReq := &services.ConsentRequest{
		UserID:      userIDInt,
		ConsentType: req.ConsentType,
		IPAddress:   c.ClientIP(),
		UserAgent:   c.GetHeader("User-Agent"),
	}

	if req.ConsentType == "gdpr" || req.ConsentType == "both" {
		consentReq.GDPRConsent = req.Granted
	}
	if req.ConsentType == "ccpa" || req.ConsentType == "both" {
		consentReq.CCPAConsent = req.Granted
	}

	if err = h.privacyService.RecordConsent(c.Request.Context(), consentReq); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to record consent"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Consent recorded successfully"})
}

// RequestDataExport initiates a data export request
func (h *PrivacyHandler) RequestDataExport(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	var req DataExportRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Convert userID string to int
	userIDStr := userID.(string)
	userIDInt, err := strconv.Atoi(userIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID format"})
		return
	}

	exportReq := &services.DataExportRequest{
		UserID:      userIDInt,
		DataTypes:   []string{"all"}, // Default to all data types
		Format:      "json",          // Default format
		IncludeFiles: false,          // Default to not include files
	}

	dataRequest, err := h.privacyService.RequestDataExport(c.Request.Context(), exportReq)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create export request"})
		return
	}

	c.JSON(http.StatusAccepted, gin.H{
		"message": "Data export request submitted successfully",
		"request_id": dataRequest.ID,
	})
}

// RequestDataDeletion initiates a data deletion request
func (h *PrivacyHandler) RequestDataDeletion(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	var req DataDeletionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Convert userID string to int
	userIDStr := userID.(string)
	userIDInt, err := strconv.Atoi(userIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID format"})
		return
	}

	deletionReq := &services.DataDeletionRequest{
		UserID:      userIDInt,
		DataTypes:   []string{"all"}, // Default to all data types
		Reason:      req.Reason,
		Confirmation: true, // Assume confirmation is true if request is made
	}

	dataRequest, err := h.privacyService.RequestDataDeletion(c.Request.Context(), deletionReq)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create deletion request"})
		return
	}

	c.JSON(http.StatusAccepted, gin.H{
		"message": "Data deletion request submitted successfully",
		"request_id": dataRequest.ID,
	})
}

// GetDataExport retrieves a data export
func (h *PrivacyHandler) GetDataExport(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	exportID := c.Param("id")
	if exportID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Export ID is required"})
		return
	}

	// Convert userID string to int
	userIDStr := userID.(string)
	userIDInt, err := strconv.Atoi(userIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID format"})
		return
	}

	// Convert exportID string to int
	exportIDInt, err := strconv.Atoi(exportID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid export ID format"})
		return
	}

	requests, err := h.privacyService.GetDataRequests(c.Request.Context(), userIDInt)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve export"})
		return
	}

	// Find the specific export request
	for _, req := range requests {
		if req.ID == exportIDInt && req.RequestType == "export" {
			c.JSON(http.StatusOK, req)
			return
		}
	}

	c.JSON(http.StatusNotFound, gin.H{"error": "Export not found"})
}

// GetUserRequests retrieves user's privacy requests
func (h *PrivacyHandler) GetUserRequests(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	// Convert userID string to int
	userIDStr := userID.(string)
	userIDInt, err := strconv.Atoi(userIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID format"})
		return
	}

	requests, err := h.privacyService.GetDataRequests(c.Request.Context(), userIDInt)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve requests"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"requests": requests})
}

// GetConsentStatus retrieves user's consent status
func (h *PrivacyHandler) GetConsentStatus(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	// Convert userID string to int
	userIDStr := userID.(string)
	userIDInt, err := strconv.Atoi(userIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID format"})
		return
	}

	privacyMetadata, err := h.privacyService.GetConsentStatus(c.Request.Context(), userIDInt)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve consent status"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"gdpr_consent": privacyMetadata.GDPRConsent,
		"ccpa_consent": privacyMetadata.CCPAConsent,
		"gdpr_consent_date": privacyMetadata.GDPRConsentDate,
		"ccpa_consent_date": privacyMetadata.CCPAConsentDate,
		"data_retention_days": privacyMetadata.DataRetentionDays,
	})
}